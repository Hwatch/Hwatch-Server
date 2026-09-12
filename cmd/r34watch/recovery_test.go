package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"r34watch/internal/store"
)

func TestRecoveryRemapsMediaPaths(t *testing.T) {
	rpc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": map[string]any{"version": "1.37"}})
	}))
	defer rpc.Close()
	dir := t.TempDir()
	targetRoot := filepath.Join(dir, "new-media")
	sourceRoot := "/old/media"
	wantedPath := filepath.Join(targetRoot, "Akt", "video.mp4")
	if err := os.MkdirAll(filepath.Dir(wantedPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wantedPath, []byte("video"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := store.Open(filepath.Join(dir, "hwatch.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	cfg, _ := s.Settings()
	cfg.DownloadDir = targetRoot
	cfg.AriaRPCURL = rpc.URL
	if err = s.SaveSettings(cfg); err != nil {
		t.Fatal(err)
	}
	sub := store.Subscription{Slug: "rule34video:akt", Platform: "rule34video", Type: "creator", SourceID: "akt", AdapterConfig: "{}", Cursor: "{}", Name: "Akt", URL: "https://example.invalid/akt", Enabled: true, BaselineComplete: true, CreatedAt: store.Now(), ExternalLinks: "[]"}
	if err = s.AddSubscription(sub); err != nil {
		t.Fatal(err)
	}
	id := "rule34video:1"
	if _, err = s.UpsertDiscovered(store.Video{ID: id, Platform: "rule34video", SourceID: "1", SubscriptionSlug: sub.Slug, Title: "Video", PageURL: "https://example.invalid/video/1"}); err != nil {
		t.Fatal(err)
	}
	oldPath := filepath.Join(sourceRoot, "Akt", "video.mp4")
	if err = s.UpdateTask(id, "completed", "", oldPath, oldPath, "", 5, 5); err != nil {
		t.Fatal(err)
	}
	_ = s.SetSetting("restore_pending", "true")
	_ = s.SetSetting("restore_source_root", sourceRoot)
	_ = s.SetSetting("restore_target_root", targetRoot)

	app := New(s)
	app.reconcileRecovery()
	video, _ := s.Video(id)
	if video.PlexPath != wantedPath {
		t.Fatalf("plex path = %q, want %q", video.PlexPath, wantedPath)
	}
	pending, _ := s.Setting("restore_pending")
	if pending != "false" {
		t.Fatalf("restore_pending = %q", pending)
	}
}

func TestRecoveryKeepsGIDsWhenAriaUnavailable(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "hwatch.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	cfg, _ := s.Settings()
	cfg.DownloadDir = dir
	cfg.AriaRPCURL = "http://127.0.0.1:1/jsonrpc"
	_ = s.SaveSettings(cfg)
	sub := store.Subscription{Slug: "rule34video:akt", Platform: "rule34video", Type: "creator", SourceID: "akt", AdapterConfig: "{}", Cursor: "{}", Name: "Akt", URL: "https://example.invalid/akt", Enabled: true, BaselineComplete: true, CreatedAt: store.Now(), ExternalLinks: "[]"}
	_ = s.AddSubscription(sub)
	id := "rule34video:2"
	_, _ = s.UpsertDiscovered(store.Video{ID: id, Platform: "rule34video", SourceID: "2", SubscriptionSlug: sub.Slug, Title: "Video", PageURL: "https://example.invalid/video/2"})
	_ = s.UpdateDownloaderTask(id, "active", "aria2", "old-gid", "", "", "", 0, 100)
	_ = s.SetSetting("restore_pending", "true")

	New(s).reconcileRecovery()
	video, _ := s.Video(id)
	if video.Status != "active" || video.AriaGID != "old-gid" {
		t.Fatalf("task was changed while aria2 was unavailable: status=%q gid=%q", video.Status, video.AriaGID)
	}
	pending, _ := s.Setting("restore_pending")
	if pending != "true" {
		t.Fatalf("restore_pending = %q, want true", pending)
	}
}

func TestRecoveryRequeuesOnlyMissingAriaGID(t *testing.T) {
	rpc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     int64  `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		if request.Method == "aria2.getVersion" {
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": map[string]any{"version": "1.37"}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "error": map[string]any{"code": 1, "message": "GID old-gid is not found"}})
	}))
	defer rpc.Close()
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "hwatch.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	cfg, _ := s.Settings()
	cfg.DownloadDir = dir
	cfg.AriaRPCURL = rpc.URL
	_ = s.SaveSettings(cfg)
	sub := store.Subscription{Slug: "rule34video:akt", Platform: "rule34video", Type: "creator", SourceID: "akt", AdapterConfig: "{}", Cursor: "{}", Name: "Akt", URL: "https://example.invalid/akt", Enabled: true, BaselineComplete: true, CreatedAt: store.Now(), ExternalLinks: "[]"}
	_ = s.AddSubscription(sub)
	id := "rule34video:3"
	_, _ = s.UpsertDiscovered(store.Video{ID: id, Platform: "rule34video", SourceID: "3", SubscriptionSlug: sub.Slug, Title: "Video", PageURL: "https://example.invalid/video/3"})
	_ = s.UpdateDownloaderTask(id, "active", "aria2", "old-gid", "", "", "", 0, 100)
	_ = s.SetSetting("restore_pending", "true")

	New(s).reconcileRecovery()
	video, _ := s.Video(id)
	if video.Status != "queued" || video.AriaGID != "" || video.ExternalID != "" {
		t.Fatalf("missing task was not safely requeued: status=%q gid=%q external=%q", video.Status, video.AriaGID, video.ExternalID)
	}
}
