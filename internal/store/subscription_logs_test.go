package store

import "testing"

func TestSubscriptionLogsRoundTripAndFilters(t *testing.T) {
	s := newLibraryStore(t)
	if err := s.AddSubscriptionLog("rule34video:a", "rule34video", "info", "检查完成：发现 3，新增 1，入队 1", 3, 1, 1, 842, []SubscriptionLogEvent{{At: Now(), Level: "info", Message: "新增：rule34video:1"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.AddSubscriptionLog("rule34video:b", "rule34video", "error", "boom", 0, 0, 0, 120, nil); err != nil {
		t.Fatal(err)
	}

	all, err := s.SubscriptionLogs(50, 0, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d logs, want 2", len(all))
	}
	if all[0].Message != "boom" {
		t.Fatalf("logs must be newest first, got %q", all[0].Message)
	}
	if all[1].DurationMS != 842 || all[1].FoundCount != 3 || all[1].CreatedCount != 1 || all[1].QueuedCount != 1 {
		t.Fatalf("structured fields wrong: %#v", all[1])
	}
	if len(all[1].Events) != 1 || all[1].Events[0].Message != "新增：rule34video:1" {
		t.Fatalf("events not persisted: %#v", all[1].Events)
	}

	errs, err := s.SubscriptionLogs(50, 0, "", "error")
	if err != nil {
		t.Fatal(err)
	}
	if len(errs) != 1 || errs[0].SubscriptionSlug != "rule34video:b" {
		t.Fatalf("level filter wrong: %#v", errs)
	}

	one, err := s.SubscriptionLogs(50, 0, "rule34video:a", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(one) != 1 || one[0].Level != "info" || one[0].Platform != "rule34video" {
		t.Fatalf("slug filter wrong: %#v", one)
	}

	if n, _ := s.CountSubscriptionLogs("", "error"); n != 1 {
		t.Fatalf("count error logs = %d, want 1", n)
	}
	if n, _ := s.CountSubscriptionLogs("", ""); n != 2 {
		t.Fatalf("count all logs = %d, want 2", n)
	}
}
