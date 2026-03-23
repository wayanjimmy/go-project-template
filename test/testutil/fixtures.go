package testutil

import (
	"database/sql"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-testfixtures/testfixtures/v3"
)

func repoRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller path for fixtures")
	}

	// this file is at <repo>/test/testutil/fixtures.go
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

// LoadFixtures resets tables and loads YAML fixtures from testdata/fixtures/<scenario>.
func LoadFixtures(t *testing.T, db *sql.DB, scenario string) {
	t.Helper()

	fixturesDir := filepath.Join(repoRoot(t), "testdata", "fixtures", scenario)
	fixtures, err := testfixtures.New(
		testfixtures.Database(db),
		testfixtures.Dialect("postgres"),
		testfixtures.UseAlterConstraint(),
		testfixtures.Directory(fixturesDir),
	)
	if err != nil {
		t.Fatalf("init fixtures (%s): %v", scenario, err)
	}

	if err := fixtures.EnsureTestDatabase(); err != nil {
		t.Fatalf("ensure test database before loading fixtures (%s): %v", scenario, err)
	}

	if err := fixtures.Load(); err != nil {
		t.Fatalf("load fixtures (%s): %v", scenario, err)
	}
}
