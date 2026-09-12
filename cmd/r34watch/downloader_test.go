package main

import (
	"path/filepath"
	"testing"

	"r34watch/internal/store"
)

func TestFixedDownloaderSelection(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	app := New(s)
	cfg, _ := s.Settings()
	cfg.DownloaderMode = "wget"
	if got := app.selectedEngine(cfg).ID(); got != "wget" {
		t.Fatalf("selected %s", got)
	}
	cfg.DownloaderMode = "aria2"
	if got := app.selectedEngine(cfg).ID(); got != "aria2" {
		t.Fatalf("selected %s", got)
	}
}

func TestDetachedWgetTasksReturnToQueue(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	sub := store.Subscription{Slug: "rule34video:test", Name: "test", URL: "https://example.invalid", Enabled: true, CreatedAt: store.Now()}
	if err = s.AddSubscription(sub); err != nil {
		t.Fatal(err)
	}
	v := store.Video{ID: "rule34video:1", SubscriptionSlug: sub.Slug, Title: "test", PageURL: "https://example.invalid/1", DiscoveredAt: store.Now(), UpdatedAt: store.Now()}
	if _, err = s.UpsertDiscovered(v); err != nil {
		t.Fatal(err)
	}
	if err = s.BindDownloader(v.ID, "wget", "old-process", "active", "/tmp/test.mp4"); err != nil {
		t.Fatal(err)
	}
	if n, err := s.RequeueDetachedWget(); err != nil || n != 1 {
		t.Fatalf("requeued=%d err=%v", n, err)
	}
	got, _ := s.Video(v.ID)
	if got.Status != "queued" || got.ExternalID != "" {
		t.Fatalf("unexpected task: %+v", got)
	}
}
