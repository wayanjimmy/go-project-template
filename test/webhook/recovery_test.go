//go:build e2e

package webhook_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

// This deliberately disrupts Redis permissions and kills this checkout's worker.
// It must only run on a disposable, isolated Tilt host stack.
func TestWebhookRecovery(t *testing.T) {
	if os.Getenv("WEBHOOK_TEST_DISPOSABLE") != "1" {
		t.Skip("set WEBHOOK_TEST_DISPOSABLE=1 only on isolated infrastructure")
	}
	base, receiver := os.Getenv("BASE_URL"), os.Getenv("WEBHOOK_RECEIVER_URL")
	db, err := sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	rdb := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_ADDR")})
	t.Cleanup(func() { rdb.Close() })
	call := func(method, url string, body []byte) []byte {
		t.Helper()
		req, err := http.NewRequestWithContext(context.WithoutCancel(t.Context()), method, url, bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Request-ID", "webhook-recovery")
		res, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		data, _ := io.ReadAll(res.Body)
		if res.StatusCode >= 300 {
			t.Fatalf("HTTP %d: %s", res.StatusCode, data)
		}
		return data
	}
	wait := func(name string, limit time.Duration, check func() bool) {
		t.Helper()
		deadline := time.Now().Add(limit)
		for !check() {
			if time.Now().After(deadline) {
				t.Fatalf("timed out: %s", name)
			}
			time.Sleep(250 * time.Millisecond)
		}
	}
	create := func(label string) string {
		t.Helper()
		token := fmt.Sprintf("recovery-%s-%d", label, time.Now().UnixNano())
		var user struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(call("POST", base+"/v1/users", []byte(fmt.Sprintf(`{"name":%q,"email":%q,"address":"recovery-private"}`, token, token+"@example.com"))), &user); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { call("DELETE", base+"/v1/users/"+user.ID, nil) })
		return user.ID
	}
	type delivery struct {
		ID       string `db:"id"`
		Status   string `db:"status"`
		Attempts int    `db:"attempts"`
	}
	load := func(userID string) *delivery {
		var d delivery
		if err := db.GetContext(t.Context(), &d, `SELECT id,status,attempts FROM webhook_deliveries WHERE convert_from(body,'UTF8')::jsonb->'data'->>'id'=$1`, userID); err != nil {
			return nil
		}
		return &d
	}
	completed := func(userID string) {
		wait("delivery success", 180*time.Second, func() bool { d := load(userID); return d != nil && d.Status == "succeeded" })
	}
	call("PUT", receiver+"/control", []byte(`{"statuses":[204]}`))

	t.Run("redis_outage_after_database_commit", func(t *testing.T) {
		if err := rdb.Do(t.Context(), "CLIENT", "PAUSE", 6000, "ALL").Err(); err != nil {
			t.Fatal(err)
		}
		id := create("outage")
		var pending int
		if err := db.GetContext(t.Context(), &pending, `SELECT count(*) FROM event_outbox WHERE published_at IS NULL AND convert_from(payload,'UTF8')::jsonb->>'user_id'=$1`, id); err != nil || pending != 1 {
			t.Fatalf("write must commit with pending event: count=%d err=%v", pending, err)
		}
		completed(id)
		t.Logf("user=%s committed during Redis pause; pending event recovered; delivery=%s", id, load(id).ID)
	})
	t.Run("enqueue_failure", func(t *testing.T) {
		if err := rdb.Do(t.Context(), "ACL", "SETUSER", "default", "-eval", "-evalsha").Err(); err != nil {
			t.Fatal(err)
		}
		restore := func() {
			if err := rdb.Do(t.Context(), "ACL", "SETUSER", "default", "+eval", "+evalsha").Err(); err != nil {
				t.Error(err)
			}
		}
		defer restore()
		id := create("enqueue")
		wait("durable handoff", 15*time.Second, func() bool { return load(id) != nil })
		time.Sleep(2 * time.Second)
		if d := load(id); d.Status != "pending" || d.Attempts != 0 {
			t.Fatalf("enqueue failure lost pending intent: %+v", d)
		}
		restore()
		completed(id)
		t.Logf("delivery=%s remained pending during Redis EVAL denial and succeeded after restoration", load(id).ID)
	})
	t.Run("worker_crash_after_receiver_acceptance", func(t *testing.T) {
		call("PUT", receiver+"/control", []byte(`{"statuses":[204],"response_delay":8}`))
		id := create("crash")
		var requests []struct {
			DeliveryID string `json:"delivery_id"`
			Body       string `json:"body"`
		}
		wait("receiver acceptance", 20*time.Second, func() bool {
			_ = json.Unmarshal(call("GET", receiver+"/requests", nil), &requests)
			return len(requests) > 0
		})
		tilt := os.Getenv("TILT_BIN")
		if tilt == "" {
			tilt = "tilt"
		}
		data, err := exec.CommandContext(t.Context(), tilt, "get", "cmds", "-o", "json").Output()
		if err != nil {
			t.Fatal(err)
		}
		var cmds struct {
			Items []struct {
				Spec struct {
					Args []string `json:"args"`
				} `json:"spec"`
				Status struct {
					Running *struct {
						PID int `json:"pid"`
					} `json:"running"`
				} `json:"status"`
			} `json:"items"`
		}
		if err := json.Unmarshal(data, &cmds); err != nil {
			t.Fatal(err)
		}
		killed := false
		root, _ := filepath.Abs("../..")
		for _, cmd := range cmds.Items {
			if cmd.Status.Running == nil || len(cmd.Spec.Args) == 0 || cmd.Spec.Args[len(cmd.Spec.Args)-1] != "./bin/worker" {
				continue
			}
			pid := cmd.Status.Running.PID
			children, err := os.ReadFile(fmt.Sprintf("/proc/%d/task/%d/children", pid, pid))
			if err != nil {
				t.Fatal(err)
			}
			for _, child := range strings.Fields(string(children)) {
				childPID, _ := strconv.Atoi(child)
				exe, _ := os.Readlink(fmt.Sprintf("/proc/%d/exe", childPID))
				if exe != filepath.Join(root, "bin/worker") {
					continue
				}
				process, err := os.FindProcess(childPID)
				if err != nil {
					t.Fatal(err)
				}
				if err := process.Kill(); err != nil {
					t.Fatal(err)
				}
				killed = true
			}
		}
		if !killed {
			t.Fatal("refused to kill anything: no worker from this checkout found")
		}
		time.Sleep(time.Second)
		call("PUT", receiver+"/control", []byte(`{"statuses":[204]}`))
		if data, err := exec.CommandContext(t.Context(), tilt, "trigger", "worker").CombinedOutput(); err != nil {
			t.Fatalf("restart: %s %v", data, err)
		}
		completed(id)
		var after []struct {
			DeliveryID string `json:"delivery_id"`
			Body       string `json:"body"`
		}
		if err := json.Unmarshal(call("GET", receiver+"/requests", nil), &after); err != nil {
			t.Fatal(err)
		}
		if len(after) != 1 || after[0].DeliveryID != requests[0].DeliveryID || after[0].Body != requests[0].Body {
			t.Fatalf("recovery changed delivery identity/body: before=%+v after=%+v", requests, after)
		}
		t.Logf("delivery=%s receiver accepted before SIGKILL; restart replayed identical ID/body; database attempts=%d", load(id).ID, load(id).Attempts)
	})
}
