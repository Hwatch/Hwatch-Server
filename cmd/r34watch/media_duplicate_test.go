package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"r34watch/internal/store"
)

func newMediaDuplicateTestApp(t *testing.T, mediaRoot string) (*App, *store.Store) {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := s.Settings()
	cfg.DownloadDir = mediaRoot
	cfg.MediaDuplicateMaxCandidates = 20
	cfg.MediaDuplicateDurationPercent = 1
	if err := s.SaveSettings(cfg); err != nil {
		t.Fatal(err)
	}
	return New(s), s
}

func seedMediaVideo(t *testing.T, s *store.Store, mediaRoot, id string, duration int64, sha, rel string) store.Video {
	t.Helper()
	sub := store.Subscription{Slug: "test:sub", Name: "sub", URL: "https://example.invalid/sub", Enabled: true, CreatedAt: store.Now()}
	_ = s.AddSubscription(sub)
	v := store.Video{ID: id, SubscriptionSlug: sub.Slug, Title: id, PageURL: "https://example.invalid/v/" + id}
	if _, err := s.UpsertDiscovered(v); err != nil {
		t.Fatalf("upsert %s: %v", id, err)
	}
	path := filepath.Join(mediaRoot, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("data-"+id), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec("UPDATE videos SET status='completed',duration_seconds=?,file_path=?,plex_path=?,downloaded_at=?,updated_at=? WHERE id=?", duration, path, path, store.Now(), store.Now(), id); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveFileHashes(id, 100, "quick-"+id, sha); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Video(id)
	return got
}

func TestBuildMediaDuplicateOptionsPayload(t *testing.T) {
	mediaRoot := t.TempDir()
	a, s := newMediaDuplicateTestApp(t, mediaRoot)
	defer s.Close()
	target := seedMediaVideo(t, s, mediaRoot, "test:target", 100, "sha-target", "target.mp4")
	seedMediaVideo(t, s, mediaRoot, "test:near", 101, "sha-near", "near.mp4")
	seedMediaVideo(t, s, mediaRoot, "test:far", 400, "sha-far", "far.mp4")
	cfg, _ := s.Settings()
	options, sha, err := a.buildMediaDuplicateOptions(target, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if sha != "sha-target" || options.Mode != mediaDuplicateMode {
		t.Fatalf("sha=%q mode=%q", sha, options.Mode)
	}
	if len(options.Candidates) != 1 {
		t.Fatalf("candidates=%+v want 1", options.Candidates)
	}
	c := options.Candidates[0]
	if c.VideoID != "test:near" || c.MediaRelativePath != "near.mp4" {
		t.Fatalf("candidate=%+v", c)
	}
}

func TestApplyMediaDuplicateResult(t *testing.T) {
	cases := []struct {
		classification string
		want           int
	}{
		{"whole-video", 1},
		{"trimmed-video", 1},
		{"possible-match", 1},
		{"different", 0},
	}
	for _, tc := range cases {
		t.Run(tc.classification, func(t *testing.T) {
			mediaRoot := t.TempDir()
			a, s := newMediaDuplicateTestApp(t, mediaRoot)
			defer s.Close()
			target := seedMediaVideo(t, s, mediaRoot, "test:target", 100, "sha-target", "target.mp4")
			seedMediaVideo(t, s, mediaRoot, "test:near", 100, "sha-near", "near.mp4")
			cfg, _ := s.Settings()
			job, created, err := a.createMediaDuplicateJob(target, cfg, false)
			if err != nil || !created {
				t.Fatalf("job created=%v err=%v", created, err)
			}
			result := dedupeNodeResult{Matches: []dedupeNodeMatch{{
				VideoID:          "test:near",
				Classification:   tc.classification,
				Backend:          "media-fingerprint-v1",
				AlgorithmVersion: "media-fingerprint-v1",
				Confidence:       0.9,
				Metrics:          json.RawMessage(`{"sample_count":32,"matched_samples":32}`),
			}}}
			if err := a.applyMediaDuplicateResult(job, result); err != nil {
				t.Fatal(err)
			}
			var count int
			if err := s.DB.QueryRow("SELECT count(*) FROM duplicate_candidates WHERE match_type='media_duplicate'").Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != tc.want {
				t.Fatalf("classification=%s candidates=%d want %d", tc.classification, count, tc.want)
			}
		})
	}
}

func TestApplyMediaDuplicateRejectsUnknownCandidate(t *testing.T) {
	mediaRoot := t.TempDir()
	a, s := newMediaDuplicateTestApp(t, mediaRoot)
	defer s.Close()
	target := seedMediaVideo(t, s, mediaRoot, "test:target", 100, "sha-target", "target.mp4")
	seedMediaVideo(t, s, mediaRoot, "test:near", 100, "sha-near", "near.mp4")
	cfg, _ := s.Settings()
	job, _, err := a.createMediaDuplicateJob(target, cfg, false)
	if err != nil {
		t.Fatal(err)
	}
	result := dedupeNodeResult{Matches: []dedupeNodeMatch{{VideoID: "test:unknown", Classification: "whole-video"}}}
	if err := a.applyMediaDuplicateResult(job, result); err == nil {
		t.Fatal("expected rejection of candidate outside job payload")
	}
	result = dedupeNodeResult{Matches: []dedupeNodeMatch{{VideoID: "test:near", Classification: "bogus"}}}
	if err := a.applyMediaDuplicateResult(job, result); err == nil {
		t.Fatal("expected rejection of unknown classification")
	}
}
