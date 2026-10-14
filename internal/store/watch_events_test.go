package store

import "testing"

func TestWatchEventsDedupeAndList(t *testing.T) {
	s := newLibraryStore(t)
	seedLibraryVideo(t, s, "rule34video:1", "rule34video", "completed", "https://img/1.jpg")

	if err := s.RecordWatch("rule34video:1"); err != nil {
		t.Fatal(err)
	}
	// A second event within the 10-minute window must be ignored.
	if err := s.RecordWatch("rule34video:1"); err != nil {
		t.Fatal(err)
	}
	events, err := s.WatchEvents(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].VideoID != "rule34video:1" {
		t.Fatalf("dedupe/roundtrip failed: %#v", events)
	}
	if n, _ := s.CountWatchEvents(); n != 1 {
		t.Fatalf("count = %d, want 1", n)
	}
}
