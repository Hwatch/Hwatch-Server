package store

import (
	"path/filepath"
	"testing"
)

func TestIntegrationTablesAndOrphanComputeTables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "integration.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a database that already has the old compute tables.
	if _, err = s.DB.Exec("CREATE TABLE IF NOT EXISTS compute_nodes (id TEXT PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("CREATE TABLE IF NOT EXISTS compute_jobs (id TEXT PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}

	// Reopening must not fail and must create the integration tables while
	// leaving the orphan compute tables untouched.
	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()
	for _, table := range []string{"integration_tokens", "transcode_results", "compute_nodes", "compute_jobs"} {
		var name string
		if err := reopened.DB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name); err != nil {
			t.Fatalf("table %s missing: %v", table, err)
		}
	}
}

func TestIntegrationTokenLifecycle(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "token.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.CreateIntegrationToken("transrunner", "hash-1", `["read_media"]`); err != nil {
		t.Fatal(err)
	}
	active, err := s.ActiveIntegrationToken("transrunner")
	if err != nil || active.ID == 0 {
		t.Fatalf("active=%+v err=%v", active, err)
	}
	if _, err := s.IntegrationTokenByHash("hash-1"); err != nil {
		t.Fatalf("lookup by hash: %v", err)
	}
	if err := s.TouchIntegrationToken(active.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RevokeIntegrationTokens("transrunner"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.IntegrationTokenByHash("hash-1"); err == nil {
		t.Fatal("revoked token still resolves")
	}
}
