package useronboarding_test

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"go-project-template/config"
	"go-project-template/database/sqldb"
	"go-project-template/logger"
	"go-project-template/publisher"
	"go-project-template/repository"
	"go-project-template/test/testutil"
	"go-project-template/workflows"
	"go-project-template/workflows/useronboarding"

	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	workflowpostgres "github.com/cschleiden/go-workflows/backend/postgres"
	workflowclient "github.com/cschleiden/go-workflows/client"
	workflowworker "github.com/cschleiden/go-workflows/worker"
	"github.com/stretchr/testify/require"
)

func TestUserOnboardingWorkflow_Integration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := logger.Noop()

	dbURL := testutil.StartPostgresContainer(t)
	require.NoError(t, sqldb.RunMigrations(dbURL))

	db, err := sqldb.Open(&config.Config{DatabaseURL: dbURL}, log)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close(ctx) })

	workflowDB, err := parseWorkflowPostgresConfig(dbURL)
	require.NoError(t, err)

	wb := workflowpostgres.NewPostgresBackend(
		workflowDB.host,
		workflowDB.port,
		workflowDB.user,
		workflowDB.password,
		workflowDB.database,
		workflowpostgres.WithApplyMigrations(false),
	)
	require.NotNil(t, wb)
	t.Cleanup(func() { _ = wb.Close() })

	encryptor := &testEncryptor{}
	repos := repository.NewPostgresRepositories(db, encryptor, log)
	eventPub := publisher.NewWatermillPublisher(gochannel.NewGoChannel(gochannel.Config{}, logger.NewWatermillAdapter(log)))

	w := workflowworker.New(wb, nil)
	require.NoError(t, workflows.Register(w, workflows.Dependencies{UserRepo: repos.UserRepo, Publisher: eventPub, Log: log}))
	require.NoError(t, w.Start(ctx))
	t.Cleanup(func() {
		cancel()
		_ = w.WaitForCompletion()
	})

	c := workflowclient.New(wb)
	in := useronboarding.Input{UserID: "it-u-1", Name: "It User", Email: "it@example.com", Address: "IT Street"}
	instance, err := c.CreateWorkflowInstance(ctx, workflowclient.WorkflowInstanceOptions{InstanceID: useronboarding.InstanceID(in.UserID)}, useronboarding.WorkflowName, in)
	require.NoError(t, err)

	time.Sleep(300 * time.Millisecond)
	require.NoError(t, c.SignalWorkflow(ctx, instance.InstanceID, useronboarding.SignalEmailVerified, useronboarding.EmailVerifiedSignal{Verified: true}))

	res, err := workflowclient.GetWorkflowResult[useronboarding.Result](ctx, c, instance, 30*time.Second)
	require.NoError(t, err)
	require.Equal(t, useronboarding.StatusActive, res.Status)

	u, err := repos.UserRepo.FindByID(ctx, in.UserID)
	require.NoError(t, err)
	require.Equal(t, useronboarding.UserStatusActive, u.Status)
}

type workflowPostgresConfig struct {
	host     string
	port     int
	user     string
	password string
	database string
}

func parseWorkflowPostgresConfig(databaseURL string) (*workflowPostgresConfig, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return nil, err
	}

	host := u.Hostname()
	port := 5432
	if p := u.Port(); p != "" {
		parsedPort, err := strconv.Atoi(p)
		if err != nil {
			return nil, err
		}
		port = parsedPort
	}

	user := ""
	password := ""
	if u.User != nil {
		user = u.User.Username()
		password, _ = u.User.Password()
	}

	database := strings.TrimPrefix(u.Path, "/")

	return &workflowPostgresConfig{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		database: database,
	}, nil
}

type testEncryptor struct{}

func (m *testEncryptor) Encrypt(ctx context.Context, plaintext string, aad []byte) ([]byte, error) {
	return []byte(plaintext), nil
}

func (m *testEncryptor) Decrypt(ctx context.Context, ciphertext []byte, aad []byte) (string, error) {
	return string(ciphertext), nil
}
