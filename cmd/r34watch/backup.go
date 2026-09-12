package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	"r34watch/internal/store"
)

var backupNamePattern = regexp.MustCompile(`^hwatch-[0-9]{8}T[0-9]{6}Z(?:-rollback)?\.db$`)

type backupInfo struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
	SHA256    string `json:"sha256"`
	Valid     bool   `json:"valid"`
}

func backupDirectory(cfg store.Settings) string {
	if strings.TrimSpace(cfg.BackupDir) != "" {
		return filepath.Clean(cfg.BackupDir)
	}
	return filepath.Join(cfg.DownloadDir, ".hwatch-backups")
}

func checksumFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func validateSQLite(path string) error {
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		return err
	}
	defer db.Close()
	var integrity string
	if err = db.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
		if err == nil {
			err = fmt.Errorf("integrity_check: %s", integrity)
		}
		return err
	}
	for _, table := range []string{"settings", "subscriptions", "videos", "users"} {
		var count int
		if err = db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count); err != nil || count != 1 {
			return fmt.Errorf("备份缺少必要数据表 %s", table)
		}
	}
	return nil
}

func (a *App) createBackup(rollback bool) (backupInfo, error) {
	a.backupMu.Lock()
	defer a.backupMu.Unlock()
	cfg, err := a.store.Settings()
	if err != nil {
		return backupInfo{}, err
	}
	dir := backupDirectory(cfg)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return backupInfo{}, err
	}
	suffix := ""
	if rollback {
		suffix = "-rollback"
	}
	name := "hwatch-" + time.Now().UTC().Format("20060102T150405Z") + suffix + ".db"
	path := filepath.Join(dir, name)
	if _, err = a.store.DB.Exec("VACUUM INTO ?", path); err != nil {
		_ = a.store.SetBackupResult(false, err.Error())
		return backupInfo{}, err
	}
	if err = os.Chmod(path, 0600); err != nil {
		return backupInfo{}, err
	}
	if err = validateSQLite(path); err != nil {
		_ = os.Remove(path)
		_ = a.store.SetBackupResult(false, err.Error())
		return backupInfo{}, err
	}
	sum, err := checksumFile(path)
	if err != nil {
		return backupInfo{}, err
	}
	if err = os.WriteFile(path+".sha256", []byte(sum+"  "+name+"\n"), 0600); err != nil {
		return backupInfo{}, err
	}
	info, _ := os.Stat(path)
	_ = a.store.SetBackupResult(true, "")
	if !rollback {
		a.pruneBackups(dir, cfg.BackupRetention)
	}
	return backupInfo{Name: name, Size: info.Size(), CreatedAt: info.ModTime().UTC().Format(time.RFC3339), SHA256: sum, Valid: true}, nil
}

func (a *App) pruneBackups(dir string, retention int) {
	if retention < 1 {
		retention = 7
	}
	entries, _ := os.ReadDir(dir)
	var names []string
	for _, entry := range entries {
		if backupNamePattern.MatchString(entry.Name()) && !strings.Contains(entry.Name(), "-rollback") {
			names = append(names, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	if len(names) <= retention {
		return
	}
	for _, name := range names[retention:] {
		_ = os.Remove(filepath.Join(dir, name))
		_ = os.Remove(filepath.Join(dir, name) + ".sha256")
	}
}

func (a *App) backups(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		item, err := a.createBackup(false)
		if err != nil {
			fail(w, http.StatusInternalServerError, err)
			return
		}
		respond(w, http.StatusCreated, item)
		return
	}
	cfg, err := a.store.Settings()
	if err != nil {
		fail(w, 500, err)
		return
	}
	dir := backupDirectory(cfg)
	entries, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fail(w, 500, err)
		return
	}
	items := []backupInfo{}
	for _, entry := range entries {
		if entry.IsDir() || !backupNamePattern.MatchString(entry.Name()) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		stat, statErr := entry.Info()
		sum, sumErr := checksumFile(path)
		if statErr != nil || sumErr != nil {
			continue
		}
		items = append(items, backupInfo{Name: entry.Name(), Size: stat.Size(), CreatedAt: stat.ModTime().UTC().Format(time.RFC3339), SHA256: sum, Valid: validateSQLite(path) == nil})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name > items[j].Name })
	respond(w, 200, map[string]any{"items": items, "directory": dir, "last_success": cfg.BackupLastSuccess, "last_error": cfg.BackupLastError, "sensitive": cfg.HTTPProxy != "" || cfg.AriaSecret != ""})
}

func copyPrivateFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	syncErr := out.Sync()
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if syncErr != nil {
		return syncErr
	}
	return closeErr
}

func (a *App) restoreBackup(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !backupNamePattern.MatchString(name) {
		fail(w, 400, errors.New("备份名称无效"))
		return
	}
	var input struct {
		Confirm bool `json:"confirm"`
	}
	if decode(r, &input) != nil || !input.Confirm {
		fail(w, 400, errors.New("恢复数据库需要明确确认"))
		return
	}
	cfg, _ := a.store.Settings()
	source := filepath.Join(backupDirectory(cfg), name)
	if err := validateSQLite(source); err != nil {
		fail(w, 422, err)
		return
	}
	if _, err := a.createBackup(true); err != nil {
		fail(w, 500, fmt.Errorf("创建恢复前回滚快照失败: %w", err))
		return
	}
	_ = a.store.SetDownloadPaused(true)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	_ = a.aria(cfg).PauseAll(ctx)
	cancel()
	dbPath, err := filepath.Abs(a.store.Path)
	if err != nil {
		fail(w, 500, err)
		return
	}
	tmp := dbPath + ".restore.tmp"
	old := dbPath + ".restore.previous"
	_ = os.Remove(tmp)
	_ = os.Remove(old)
	if err = copyPrivateFile(source, tmp); err != nil {
		fail(w, 500, err)
		return
	}
	if err = validateSQLite(tmp); err != nil {
		_ = os.Remove(tmp)
		fail(w, 422, err)
		return
	}
	restored, openErr := sql.Open("sqlite", tmp)
	if openErr == nil {
		_, openErr = restored.Exec("INSERT INTO settings(key,value) VALUES('download_paused','true') ON CONFLICT(key) DO UPDATE SET value='true'")
		_ = restored.Close()
	}
	if openErr != nil {
		_ = os.Remove(tmp)
		fail(w, 500, fmt.Errorf("无法在恢复后保持下载暂停状态: %w", openErr))
		return
	}
	if err = a.store.Close(); err != nil {
		_ = os.Remove(tmp)
		fail(w, 500, err)
		return
	}
	if err = os.Rename(dbPath, old); err == nil {
		_ = os.Chmod(old, 0600)
		err = os.Rename(tmp, dbPath)
	}
	if err != nil {
		_ = os.Rename(old, dbPath)
		fail(w, 500, fmt.Errorf("原子替换数据库失败: %w", err))
		return
	}
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")
	respond(w, http.StatusAccepted, map[string]any{"restoring": true, "backup": name, "restart_required": true})
	go func() {
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()
}
