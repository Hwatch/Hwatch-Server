package store

import (
	"path/filepath"
	"testing"
)

func TestDuplicateReviewRequeuesWithoutDeletingMetadata(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "dedupe.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	sub := Subscription{Slug: "test:creator", Platform: "test", Type: "creator", SourceID: "creator", Name: "creator", URL: "https://example.invalid/creator", Enabled: true, CreatedAt: Now()}
	if err = s.AddSubscription(sub); err != nil {
		t.Fatal(err)
	}
	for _, v := range []Video{{ID: "test:a", SubscriptionSlug: sub.Slug, Platform: "test", SourceID: "a", Title: "A", PageURL: "https://example.invalid/a", DiscoveredAt: Now()}, {ID: "test:b", SubscriptionSlug: sub.Slug, Platform: "test", SourceID: "b", Title: "B", PageURL: "https://example.invalid/b", DiscoveredAt: Now()}} {
		if _, err = s.UpsertDiscovered(v); err != nil {
			t.Fatal(err)
		}
	}
	if err = s.AddDuplicate("test:b", "test:a", "cover_duration", .97, 3, true); err != nil {
		t.Fatal(err)
	}
	v, _ := s.Video("test:b")
	if v.Status != "excluded" {
		t.Fatalf("status=%s", v.Status)
	}
	items, err := s.DuplicateCandidates()
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%v err=%v", items, err)
	}
	if _, err = s.ReviewDuplicate(items[0].ID, "dismiss"); err != nil {
		t.Fatal(err)
	}
	v, _ = s.Video("test:b")
	if v.Status != "queued" || v.Title != "B" {
		t.Fatalf("review lost metadata: %+v", v)
	}
}
