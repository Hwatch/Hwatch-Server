package store

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func seedCompletedMedia(t *testing.T, s *Store, id string, duration int64, sha, rel string) {
	t.Helper()
	sub := Subscription{Slug: "test:sub", Name: "sub", URL: "https://example.invalid/sub", Enabled: true, CreatedAt: Now()}
	_ = s.AddSubscription(sub)
	v := Video{ID: id, SubscriptionSlug: sub.Slug, Title: id, PageURL: "https://example.invalid/v/" + id}
	if _, err := s.UpsertDiscovered(v); err != nil {
		t.Fatalf("upsert %s: %v", id, err)
	}
	path := "/media/" + rel
	if _, err := s.DB.Exec("UPDATE videos SET status='completed',duration_seconds=?,file_path=?,plex_path=?,downloaded_at=?,updated_at=? WHERE id=?", duration, path, path, Now(), Now(), id); err != nil {
		t.Fatalf("complete %s: %v", id, err)
	}
	if err := s.SaveFileHashes(id, 100, "quick-"+id, sha); err != nil {
		t.Fatalf("hashes %s: %v", id, err)
	}
}

func TestMediaDuplicateCandidatesDurationAndExclusions(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "cand.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	seedCompletedMedia(t, s, "test:target", 100, "sha-target", "target.mp4")
	seedCompletedMedia(t, s, "test:near", 101, "sha-near", "near.mp4")
	seedCompletedMedia(t, s, "test:far", 130, "sha-far", "far.mp4")
	seedCompletedMedia(t, s, "test:exact", 100, "sha-target", "exact.mp4")

	rows, err := s.MediaDuplicateCandidates("test:target", "sha-target", 100, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != "test:near" {
		t.Fatalf("candidates=%+v want only test:near", ids(rows))
	}
}

func ids(videos []Video) []string {
	out := make([]string, 0, len(videos))
	for _, v := range videos {
		out = append(out, v.ID)
	}
	return out
}

func TestMediaDuplicatePairCanonicalAndVersion(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "pair.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	seedCompletedMedia(t, s, "test:a", 100, "sha-a", "a.mp4")
	seedCompletedMedia(t, s, "test:b", 100, "sha-b", "b.mp4")
	created, err := s.AddMediaDuplicate("test:b", "test:a", "whole-video", "media-fingerprint-v1", "media-fingerprint-v1", `{"sample_count":32}`, .97, 0)
	if err != nil || !created {
		t.Fatalf("first insert created=%v err=%v", created, err)
	}
	again, err := s.AddMediaDuplicate("test:a", "test:b", "whole-video", "media-fingerprint-v1", "media-fingerprint-v1", `{"sample_count":32}`, .99, 1)
	if err != nil || again {
		t.Fatalf("reverse insert created=%v err=%v", again, err)
	}
	var count int
	if err = s.DB.QueryRow("SELECT count(*) FROM duplicate_candidates WHERE match_type='media_duplicate'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("rows=%d want 1", count)
	}
	var videoID, matchID, classification, version string
	if err = s.DB.QueryRow("SELECT video_id,match_video_id,classification,algorithm_version FROM duplicate_candidates WHERE match_type='media_duplicate'").Scan(&videoID, &matchID, &classification, &version); err != nil {
		t.Fatal(err)
	}
	if videoID != "test:a" || matchID != "test:b" {
		t.Fatalf("not canonical: %s/%s", videoID, matchID)
	}
	if classification != "whole-video" || version != "media-fingerprint-v1" {
		t.Fatalf("classification=%q version=%q", classification, version)
	}
}

func TestMediaDuplicateTargetsSkipExistingJob(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "targets.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	seedCompletedMedia(t, s, "test:t", 100, "sha-t", "t.mp4")
	targets, err := s.MediaDuplicateTargets("media-fingerprint-v1", 10)
	if err != nil || len(targets) != 1 {
		t.Fatalf("targets=%v err=%v", ids(targets), err)
	}
	job := ComputeJob{ID: "job_x", Type: "dedupe", Subtype: "media_duplicate", VideoID: "test:t", IdempotencyKey: "key-x", AlgorithmVersion: "media-fingerprint-v1", MediaRelativePath: "t.mp4", InputJSON: "{}"}
	if _, _, err = s.CreateComputeJob(job); err != nil {
		t.Fatal(err)
	}
	targets, err = s.MediaDuplicateTargets("media-fingerprint-v1", 10)
	if err != nil || len(targets) != 0 {
		t.Fatalf("targets after job=%v err=%v", ids(targets), err)
	}
}

func TestClaimMediaDuplicateRequiresSubtype(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "claim.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	seedCompletedMedia(t, s, "test:t", 100, "sha-t", "t.mp4")
	if _, _, err = s.CreateComputeJob(ComputeJob{ID: "job_md", Type: "dedupe", Subtype: "media_duplicate", VideoID: "test:t", IdempotencyKey: "md-key", AlgorithmVersion: "media-fingerprint-v1", MediaRelativePath: "t.mp4", InputJSON: "{}"}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ClaimComputeJob("node_plain", []string{"dedupe"}, nil, time.Minute); !errors.Is(err, ErrNoComputeJob) {
		t.Fatalf("plain node claim err=%v want ErrNoComputeJob", err)
	}
	claimed, err := s.ClaimComputeJob("node_dup", []string{"dedupe"}, []string{"media_duplicate"}, time.Minute)
	if err != nil {
		t.Fatalf("media-duplicate node claim err=%v", err)
	}
	if claimed.Subtype != "media_duplicate" || claimed.NodeID != "node_dup" {
		t.Fatalf("claimed=%+v", claimed)
	}
}
