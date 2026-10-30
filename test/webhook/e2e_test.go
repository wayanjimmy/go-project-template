//go:build e2e

package webhook_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Run against the Tilt host stack and its controlled receiver. No containers are started.
func TestWebhookDelivery(t *testing.T) {
	base := os.Getenv("BASE_URL")
	dsn := os.Getenv("DATABASE_URL")
	receiver := os.Getenv("WEBHOOK_RECEIVER_URL")
	secret := os.Getenv("WEBHOOK_SECRET")
	if base == "" || dsn == "" || receiver == "" || secret == "" {
		t.Fatal("BASE_URL, DATABASE_URL, WEBHOOK_RECEIVER_URL and WEBHOOK_SECRET are required")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	client := &http.Client{Timeout: 5 * time.Second}
	call := func(method, url string, body []byte) []byte {
		t.Helper()
		req, err := http.NewRequestWithContext(t.Context(), method, url, bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Request-ID", "webhook-e2e")
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		data, _ := io.ReadAll(res.Body)
		if res.StatusCode >= 300 {
			t.Fatalf("%s %s: %d %s", method, url, res.StatusCode, data)
		}
		return data
	}
	for _, tc := range []struct {
		name     string
		statuses []int
		outcome  string
		attempts int
	}{
		{"success", []int{204}, "succeeded", 1},
		{"retry", []int{500, 500, 204}, "succeeded", 3},
		{"permanent", []int{400}, "failed", 1},
		{"redirect", []int{302}, "failed", 1},
		{"exhausted", []int{503}, "failed", 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config, _ := json.Marshal(map[string]any{"statuses": tc.statuses})
			call("PUT", receiver+"/control", config)
			token := fmt.Sprintf("webhook-%s-%d", tc.name, time.Now().UnixNano())
			data := call("POST", base+"/v1/users", []byte(fmt.Sprintf(`{"name":%q,"email":%q,"address":"must-not-leak"}`, token, token+"@example.com")))
			var user struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(data, &user); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { call("DELETE", base+"/v1/users/"+user.ID, nil) })
			var delivery struct {
				ID       string `db:"id"`
				Status   string `db:"status"`
				Attempts int    `db:"attempts"`
				EventID  string `db:"event_id"`
			}
			deadline := time.Now().Add(120 * time.Second)
			for {
				err := db.GetContext(t.Context(), &delivery, `SELECT id,status,attempts,event_id FROM webhook_deliveries WHERE convert_from(body,'UTF8')::jsonb->'data'->>'id'=$1`, user.ID)
				if err == nil && delivery.Status == tc.outcome {
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("delivery did not finish: %+v error=%v", delivery, err)
				}
				time.Sleep(250 * time.Millisecond)
			}
			if delivery.Attempts != tc.attempts {
				t.Fatalf("attempts: got %d want %d", delivery.Attempts, tc.attempts)
			}
			var requests []struct {
				Body       string `json:"body"`
				DeliveryID string `json:"delivery_id"`
				Timestamp  string `json:"timestamp"`
				Signature  string `json:"signature"`
			}
			if err := json.Unmarshal(call("GET", receiver+"/requests", nil), &requests); err != nil {
				t.Fatal(err)
			}
			if len(requests) != tc.attempts {
				t.Fatalf("receiver requests: got %d want %d", len(requests), tc.attempts)
			}
			for _, req := range requests {
				mac := hmac.New(sha256.New, []byte(secret))
				mac.Write([]byte(req.Timestamp + "." + req.Body))
				if req.Signature != "sha256="+hex.EncodeToString(mac.Sum(nil)) {
					t.Fatal("invalid HMAC signature")
				}
				if req.DeliveryID != delivery.ID || strings.Contains(req.Body, "must-not-leak") || strings.Contains(req.Body, "address") {
					t.Fatalf("unsafe or unstable payload: %+v", req)
				}
				if req.Body != requests[0].Body {
					t.Fatal("body changed on retry")
				}
			}
			// Search is an independent subscriber, not work stolen by the webhook group.
			deadline = time.Now().Add(15 * time.Second)
			for {
				data := call("GET", base+"/v1/search/users?q="+token, nil)
				if bytes.Contains(data, []byte(user.ID)) {
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("search missing user: %s", data)
				}
				time.Sleep(250 * time.Millisecond)
			}
			// Republishing an already acknowledged event must not create a second delivery.
			_, err := db.ExecContext(t.Context(), `UPDATE event_outbox SET published_at=NULL WHERE id=$1`, delivery.EventID)
			if err != nil {
				t.Fatal(err)
			}
			time.Sleep(3 * time.Second)
			var count int
			if err := db.GetContext(t.Context(), &count, `SELECT count(*) FROM webhook_deliveries WHERE event_id=$1`, delivery.EventID); err != nil || count != 1 {
				t.Fatalf("duplicate delivery count=%d err=%v", count, err)
			}
			if tc.name == "success" {
				// An old creation event must not roll back a completed update.
				newName, newEmail := token+" revised", token+"-revised@example.com"
				call("PUT", base+"/v1/users/"+user.ID, []byte(fmt.Sprintf(`{"name":%q,"email":%q,"address":"must-not-leak"}`, newName, newEmail)))
				deadline = time.Now().Add(15 * time.Second)
				for !bytes.Contains(call("GET", base+"/v1/search/users?q="+token, nil), []byte(newEmail)) {
					if time.Now().After(deadline) {
						t.Fatal("updated user was not indexed")
					}
					time.Sleep(250 * time.Millisecond)
				}
				var indexedAt time.Time
				if err := db.GetContext(t.Context(), &indexedAt, `SELECT updated_at FROM user_search_index WHERE user_id=$1`, user.ID); err != nil {
					t.Fatal(err)
				}
				if _, err := db.ExecContext(t.Context(), `UPDATE event_outbox SET published_at=NULL WHERE id=$1`, delivery.EventID); err != nil {
					t.Fatal(err)
				}
				deadline = time.Now().Add(15 * time.Second)
				for {
					var replayed bool
					if err := db.GetContext(t.Context(), &replayed, `SELECT updated_at>$2 FROM user_search_index WHERE user_id=$1`, user.ID, indexedAt); err != nil {
						t.Fatal(err)
					}
					if replayed {
						break
					}
					if time.Now().After(deadline) {
						t.Fatal("creation replay was not indexed")
					}
					time.Sleep(250 * time.Millisecond)
				}
				data := call("GET", base+"/v1/search/users?q=revised", nil)
				if !bytes.Contains(data, []byte(user.ID)) || !bytes.Contains(data, []byte(newName)) || !bytes.Contains(data, []byte(newEmail)) {
					t.Fatalf("creation replay rolled back updated search fields: %s", data)
				}
				t.Logf("user=%s creation_event=%s replay retained updated name, email and search document", user.ID, delivery.EventID)
			}
			t.Logf("delivery=%s event=%s status=%s attempts=%d signature=verified search=present duplicate_event=deduplicated", delivery.ID, delivery.EventID, delivery.Status, delivery.Attempts)
		})
	}
}
