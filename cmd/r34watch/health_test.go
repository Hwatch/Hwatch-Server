package main

import (
	"context"
	"math"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"r34watch/internal/store"
)

func TestHealthDegradesWhenSelectedDownloaderIsOffline(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "health.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	app := New(s)
	cfg, _ := s.Settings()
	cfg.DownloadDir = t.TempDir()
	cfg.DownloaderMode = "aria2"
	cfg.AriaRPCURL = "http://127.0.0.1:1/jsonrpc"
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	health := app.healthSnapshot(ctx, cfg)
	if health["status"] != "degraded" {
		t.Fatalf("health = %#v", health)
	}
}

func TestHealthDetectsMinimumFreeSpace(t *testing.T) {
	if _, err := exec.LookPath("wget"); err != nil {
		t.Skip("wget unavailable")
	}
	s, err := store.Open(filepath.Join(t.TempDir(), "health.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	app := New(s)
	cfg, _ := s.Settings()
	cfg.DownloadDir = t.TempDir()
	cfg.DownloaderMode = "wget"
	cfg.MinFreeBytes = math.MaxInt64
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	health := app.healthSnapshot(ctx, cfg)
	if health["status"] != "degraded" {
		t.Fatalf("health = %#v", health)
	}
}
