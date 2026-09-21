package store

import (
	"path/filepath"
	"sync"
	"testing"
)

func TestSQLiteLifecycle(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	sub := Subscription{Slug: "rule34video:artist", Platform: "rule34video", Type: "creator", SourceID: "artist", AdapterConfig: "{}", Cursor: "{}", Name: "Artist", URL: "https://rule34video.com/models/artist/", Enabled: true, BaselineComplete: true, CreatedAt: Now(), ExternalLinks: "[]"}
	if err = s.AddSubscription(sub); err != nil {
		t.Fatal(err)
	}
	created, err := s.UpsertDiscovered(Video{ID: "rule34video:123", Platform: "rule34video", SourceID: "123", SubscriptionSlug: sub.Slug, Title: "Episode", PageURL: "https://rule34video.com/video/123/x/"})
	if err != nil || !created {
		t.Fatalf("created=%v err=%v", created, err)
	}
	created, err = s.UpsertDiscovered(Video{ID: "rule34video:123", Platform: "rule34video", SourceID: "123", SubscriptionSlug: sub.Slug, Title: "Episode"})
	if err != nil || created {
		t.Fatalf("duplicate created=%v err=%v", created, err)
	}
	episode, err := s.NextEpisode(sub.Slug)
	if err != nil || episode != 1 {
		t.Fatalf("episode=%d err=%v", episode, err)
	}
	if err = s.SaveTranslation(Translation{EntityID: "rule34video:123", Language: "zh-CN", Field: "title", SourceHash: "sha256:test", Text: "标题"}); err != nil {
		t.Fatal(err)
	}
	translations, err := s.Translations("zh-CN")
	if err != nil || translations["rule34video:123\x00title"].Text != "标题" {
		t.Fatalf("translations=%+v err=%v", translations, err)
	}
	if err = s.UpdateTask("rule34video:123", "completed", "", "/tmp/video.mp4", "/tmp/video.mp4", "", 10, 10); err != nil {
		t.Fatal(err)
	}
	if err = s.RecordPlayback("rule34video:123"); err != nil {
		t.Fatal(err)
	}
	recent, err := s.RecentlyPlayed(10)
	if err != nil || len(recent) != 1 || recent[0].ID != "rule34video:123" {
		t.Fatalf("recent=%+v err=%v", recent, err)
	}
}

func TestSubmissionClaimIsConcurrentSafeAndAuditable(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	sub := Subscription{Slug: "rule34video:claim", Name: "claim", URL: "https://example.invalid/claim", Enabled: true, CreatedAt: Now()}
	if err = s.AddSubscription(sub); err != nil {
		t.Fatal(err)
	}
	v := Video{ID: "rule34video:claim", SubscriptionSlug: sub.Slug, Title: "claim", PageURL: "https://example.invalid/v"}
	if _, err = s.UpsertDiscovered(v); err != nil {
		t.Fatal(err)
	}
	if err = s.QueueVideo(v.ID); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); results <- s.BeginSubmission(v.ID, "aria2", "same-gid", "/media/x.mp4") }()
	}
	wg.Wait()
	close(results)
	success := 0
	for result := range results {
		if result == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("successful claims=%d want 1", success)
	}
	events, err := s.TaskEvents(v.ID, 10)
	if err != nil || len(events) != 1 || events[0].ToStatus != "submitting" {
		t.Fatalf("events=%+v err=%v", events, err)
	}
	if err = s.AddAudit("admin", "POST /api/test", "http", "", "success", map[string]any{"status": 200}); err != nil {
		t.Fatal(err)
	}
	logs, err := s.AuditLogs(10, 0)
	if err != nil || len(logs) != 1 || logs[0].Actor != "admin" {
		t.Fatalf("logs=%+v err=%v", logs, err)
	}
}

func TestRequeueCapacityFailure(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	sub := Subscription{Slug: "rule34video:test", Name: "test", URL: "https://example.invalid", Enabled: true, CreatedAt: Now()}
	if err = s.AddSubscription(sub); err != nil {
		t.Fatal(err)
	}
	v := Video{ID: "rule34video:capacity", SubscriptionSlug: sub.Slug, Title: "test", PageURL: "https://example.invalid/capacity"}
	if _, err = s.UpsertDiscovered(v); err != nil {
		t.Fatal(err)
	}
	if err = s.UpdateTask(v.ID, "failed", "", "", "", "wget capacity reached", 0, 0); err != nil {
		t.Fatal(err)
	}
	if count, requeueErr := s.RequeueTransientFailures(); requeueErr != nil || count != 1 {
		t.Fatalf("requeued=%d err=%v", count, requeueErr)
	}
	got, _ := s.Video(v.ID)
	if got.Status != "queued" || got.Error != "" {
		t.Fatalf("unexpected task after requeue: status=%q error=%q", got.Status, got.Error)
	}
}
