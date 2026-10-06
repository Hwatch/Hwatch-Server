package store

import (
	"path/filepath"
	"testing"
)

// TestOpenMigratesLegacyComputeSchema guards the upgrade path: an existing
// database whose compute_jobs/duplicate_candidates predate the media-duplicate
// columns must open successfully and receive the additive columns. A schema
// index over a not-yet-migrated column would otherwise abort Open() before the
// migrations run.
func TestOpenMigratesLegacyComputeSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("DROP TABLE compute_jobs"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("DROP TABLE duplicate_candidates"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("CREATE TABLE compute_jobs (id TEXT PRIMARY KEY,job_type TEXT NOT NULL,video_id TEXT NOT NULL,status TEXT NOT NULL,node_id TEXT NOT NULL DEFAULT '',idempotency_key TEXT NOT NULL UNIQUE,algorithm_version TEXT NOT NULL,media_relative_path TEXT NOT NULL,input_json TEXT NOT NULL DEFAULT '{}',result_json TEXT NOT NULL DEFAULT '{}',progress REAL NOT NULL DEFAULT 0,error_code TEXT NOT NULL DEFAULT '',error_message TEXT NOT NULL DEFAULT '',lease_until TEXT NOT NULL DEFAULT '',attempts INTEGER NOT NULL DEFAULT 0,created_at TEXT NOT NULL,updated_at TEXT NOT NULL,completed_at TEXT NOT NULL DEFAULT '',FOREIGN KEY(video_id) REFERENCES videos(id) ON DELETE CASCADE)"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("CREATE TABLE duplicate_candidates (id INTEGER PRIMARY KEY AUTOINCREMENT,video_id TEXT NOT NULL,match_video_id TEXT NOT NULL,match_type TEXT NOT NULL,confidence REAL NOT NULL DEFAULT 1,duration_delta INTEGER NOT NULL DEFAULT 0,status TEXT NOT NULL DEFAULT 'pending',created_at TEXT NOT NULL,reviewed_at TEXT NOT NULL DEFAULT '',UNIQUE(video_id,match_video_id,match_type))"); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}

	migrated, err := Open(path)
	if err != nil {
		t.Fatalf("opening legacy database must migrate, got %v", err)
	}
	defer migrated.Close()
	for _, col := range []string{"subtype"} {
		assertHasColumn(t, migrated, "compute_jobs", col)
	}
	for _, col := range []string{"classification", "backend", "algorithm_version", "metrics_json"} {
		assertHasColumn(t, migrated, "duplicate_candidates", col)
	}
}

func assertHasColumn(t *testing.T, s *Store, table, column string) {
	t.Helper()
	rows, err := s.DB.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt any
		if err = rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatal(err)
		}
		if name == column {
			return
		}
	}
	t.Fatalf("%s.%s missing after migration", table, column)
}
