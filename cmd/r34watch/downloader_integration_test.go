package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"r34watch/internal/adapters"
	"r34watch/internal/store"
)

type refreshTestAdapter struct{ calls *atomic.Int32 }

func (a refreshTestAdapter) ID() string           { return "test" }
func (a refreshTestAdapter) Name() string         { return "test" }
func (a refreshTestAdapter) MatchURL(string) bool { return false }
func (a refreshTestAdapter) SearchCreators(context.Context, string, string) ([]adapters.CreatorResult, error) {
	return nil, nil
}
func (a refreshTestAdapter) ParseSubscription(context.Context, string, string) (store.Subscription, error) {
	return store.Subscription{}, nil
}
func (a refreshTestAdapter) ListVideos(context.Context, store.Subscription, string) ([]store.Video, error) {
	return nil, nil
}
func (a refreshTestAdapter) FetchDetails(_ context.Context, v store.Video, _ string) (store.Video, error) {
	a.calls.Add(1)
	v.Title = "refreshed"
	return v, nil
}
func (a refreshTestAdapter) ResolveDownload(context.Context, store.Video, string) (adapters.DownloadSource, error) {
	return adapters.DownloadSource{}, errors.New("not used")
}
func (a refreshTestAdapter) ParseImported(string) (store.Video, error) { return store.Video{}, nil }

func TestExpiredAriaSourceRefreshesAndRequeues(t *testing.T) {
	rpc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		result := any("ok")
		if request.Method == "aria2.tellStatus" {
			result = map[string]any{"gid": "old-gid", "status": "error", "errorCode": "22", "completedLength": "128", "totalLength": "1024", "files": []any{map[string]any{"path": "/tmp/partial.mp4"}}}
		} else if request.Method == "aria2.getVersion" {
			result = map[string]any{"version": "test"}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
	}))
	defer rpc.Close()
	s, err := store.Open(filepath.Join(t.TempDir(), "integration.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	cfg, _ := s.Settings()
	cfg.AriaRPCURL = rpc.URL
	cfg.DownloaderMode = "aria2"
	cfg.DownloadPaused = true
	if err = s.SaveSettings(cfg); err != nil {
		t.Fatal(err)
	}
	if err = s.SetDownloadPaused(true); err != nil {
		t.Fatal(err)
	}
	sub := store.Subscription{Slug: "test:creator", Platform: "test", Type: "creator", SourceID: "creator", Name: "creator", URL: "https://example.invalid", Enabled: true, CreatedAt: store.Now()}
	if err = s.AddSubscription(sub); err != nil {
		t.Fatal(err)
	}
	v := store.Video{ID: "test:video", Platform: "test", SourceID: "video", SubscriptionSlug: sub.Slug, Title: "old", PageURL: "https://example.invalid/video", DiscoveredAt: store.Now(), UpdatedAt: store.Now()}
	if _, err = s.UpsertDiscovered(v); err != nil {
		t.Fatal(err)
	}
	if err = s.BindDownloader(v.ID, "aria2", "old-gid", "active", "/tmp/partial.mp4"); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	app := New(s)
	app.adapters = adapters.NewRegistry(refreshTestAdapter{calls: &calls})
	app.pollTasks(context.Background())
	got, err := s.Video(v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "queued" || got.ExternalID != "" || got.PlexPath != "/tmp/partial.mp4" || got.Title != "refreshed" || calls.Load() != 1 {
		t.Fatalf("source refresh reconciliation failed: %+v calls=%d", got, calls.Load())
	}
}
