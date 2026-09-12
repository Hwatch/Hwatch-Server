package main

import (
	"os"
	"path/filepath"
	"testing"

	"r34watch/internal/store"
)

func TestCreateBackupAndRetention(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "hwatch.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	cfg, _ := s.Settings()
	cfg.DownloadDir = filepath.Join(dir, "media")
	cfg.BackupRetention = 7
	if err = s.SaveSettings(cfg); err != nil {
		t.Fatal(err)
	}
	app := New(s)
	item, err := app.createBackup(false)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfg.DownloadDir, ".hwatch-backups", item.Name)
	if err = validateSQLite(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("backup permissions = %o", info.Mode().Perm())
	}
	app.pruneBackups(filepath.Dir(path), 7) // must not panic with fewer files than retention
}

func TestBackupNameRejectsTraversal(t *testing.T) {
	for _, name := range []string{"../hwatch-20260912T120000Z.db", "other.db", "hwatch-20260912T120000Z.db/extra"} {
		if backupNamePattern.MatchString(name) {
			t.Fatalf("unsafe name accepted: %q", name)
		}
	}
}
