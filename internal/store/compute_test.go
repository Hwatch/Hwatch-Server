package store

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func seedComputeTest(t *testing.T, s *Store, suffix string) Video {
	t.Helper()
	sub := Subscription{Slug: "test:" + suffix, Name: suffix, URL: "https://example.invalid/" + suffix, Enabled: true, CreatedAt: Now()}
	if err := s.AddSubscription(sub); err != nil {
		t.Fatal(err)
	}
	v := Video{ID: "test:" + suffix, SubscriptionSlug: sub.Slug, Title: suffix, PageURL: "https://example.invalid/video/" + suffix}
	if _, err := s.UpsertDiscovered(v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestComputeJobLeaseAndIdempotency(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "compute.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	v := seedComputeTest(t, s, "video")
	if err = s.CreateComputeNode("node_a", "A", "hash-a", `{"dedupe":true}`); err != nil {
		t.Fatal(err)
	}
	if err = s.CreateComputeNode("node_b", "B", "hash-b", `{"dedupe":true}`); err != nil {
		t.Fatal(err)
	}
	want := ComputeJob{ID: "job_1", Type: "dedupe", VideoID: v.ID, IdempotencyKey: "same-input", AlgorithmVersion: "v1", MediaRelativePath: "video.mp4", InputJSON: "{}"}
	createdJob, created, err := s.CreateComputeJob(want)
	if err != nil || !created {
		t.Fatalf("create=%v job=%+v err=%v", created, createdJob, err)
	}
	duplicate, created, err := s.CreateComputeJob(ComputeJob{ID: "job_2", Type: "dedupe", VideoID: v.ID, IdempotencyKey: "same-input", AlgorithmVersion: "v1", MediaRelativePath: "video.mp4", InputJSON: "{}"})
	if err != nil || created || duplicate.ID != "job_1" {
		t.Fatalf("idempotent create=%v job=%+v err=%v", created, duplicate, err)
	}
	claimed, err := s.ClaimComputeJob("node_a", []string{"dedupe"}, nil, time.Minute)
	if err != nil || claimed.NodeID != "node_a" || claimed.Attempts != 1 {
		t.Fatalf("claim=%+v err=%v", claimed, err)
	}
	if err = s.RenewComputeLease(claimed.ID, "node_b", "analyzing", .2, time.Minute); !errors.Is(err, ErrComputeLease) {
		t.Fatalf("wrong node lease: %v", err)
	}
	if err = s.RenewComputeLease(claimed.ID, "node_a", "analyzing", .5, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err = s.CompleteComputeJob(claimed.ID, "node_a", "completed", `{"matches":[]}`); err != nil {
		t.Fatal(err)
	}
	done, err := s.ComputeJob(claimed.ID)
	if err != nil || done.Status != "completed" || done.Progress != 1 {
		t.Fatalf("done=%+v err=%v", done, err)
	}
}

func TestExpiredComputeLeaseIsReclaimed(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "compute.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	v := seedComputeTest(t, s, "lease")
	_ = s.CreateComputeNode("node_a", "A", "hash-a", `{"dedupe":true}`)
	_ = s.CreateComputeNode("node_b", "B", "hash-b", `{"dedupe":true}`)
	_, _, err = s.CreateComputeJob(ComputeJob{ID: "job_lease", Type: "dedupe", VideoID: v.ID, IdempotencyKey: "lease", AlgorithmVersion: "v1", MediaRelativePath: "v.mp4", InputJSON: "{}"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ClaimComputeJob("node_a", []string{"dedupe"}, nil, time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("UPDATE compute_jobs SET lease_until=? WHERE id=?", time.Now().UTC().Add(-time.Minute).Format(time.RFC3339), "job_lease"); err != nil {
		t.Fatal(err)
	}
	reclaimed, err := s.ClaimComputeJob("node_b", []string{"dedupe"}, nil, time.Minute)
	if err != nil || reclaimed.NodeID != "node_b" || reclaimed.Attempts != 2 {
		t.Fatalf("reclaimed=%+v err=%v", reclaimed, err)
	}
}
