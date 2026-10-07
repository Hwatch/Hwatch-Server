package store

import (
	"path/filepath"
	"testing"
)

func seedExclusionTestMedia(t *testing.T, s *Store, id string) {
	t.Helper()
	sub := Subscription{Slug: "test:sub", Name: "sub", URL: "https://example.invalid/sub", Enabled: true, CreatedAt: Now()}
	_ = s.AddSubscription(sub)
	if _, err := s.UpsertDiscovered(Video{ID: id, SubscriptionSlug: sub.Slug, Title: id, PageURL: "https://example.invalid/v/" + id}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec("UPDATE videos SET status='completed',file_path='/media/x.mp4',plex_path='/media/x.mp4',duration_seconds=10,updated_at=? WHERE id=?", Now(), id); err != nil {
		t.Fatal(err)
	}
}

func TestMediaExclusionPersistsAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "excl.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	seedExclusionTestMedia(t, s, "test:x")
	if err := s.AddMediaExclusion(MediaExclusion{VideoID: "test:x", Source: "test", Reason: "user_delete", TitleSnapshot: "T"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if !reopened.IsMediaExcluded("test:x") {
		t.Fatal("exclusion not persisted")
	}
	items, total, err := reopened.MediaExclusions(10, 0, "", "")
	if err != nil || total != 1 || len(items) != 1 || items[0].VideoID != "test:x" {
		t.Fatalf("items=%v total=%d err=%v", items, total, err)
	}
	if err := reopened.RemoveMediaExclusion("test:x"); err != nil {
		t.Fatal(err)
	}
	if reopened.IsMediaExcluded("test:x") {
		t.Fatal("exclusion not removed")
	}
}
