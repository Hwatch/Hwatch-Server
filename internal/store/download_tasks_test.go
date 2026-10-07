package store

import (
	"path/filepath"
	"testing"
)

func TestTaskVideosOnlyReturnsDownloadTasks(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.AddSubscription(Subscription{Slug: "t:sub", Name: "s", URL: "https://example.invalid/sub", Enabled: true, CreatedAt: Now()}); err != nil {
		t.Fatal(err)
	}
	add := func(id, status string) {
		t.Helper()
		if _, err := s.DB.Exec("INSERT INTO videos(id,subscription_slug,title,page_url,discovered_at,updated_at,status) VALUES(?,?,?,?,?,?,?)", id, "t:sub", id, "https://example.invalid/v/"+id, Now(), Now(), status); err != nil {
			t.Fatal(err)
		}
	}
	add("t:media", "removed") // media only, no download task
	add("t:disc", "discovered")
	add("t:excl", "excluded")
	add("t:empty", "")
	add("t:queued", "queued")
	add("t:done", "completed")

	items, err := s.TaskVideos(100, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, v := range items {
		got[v.ID] = true
	}
	if !got["t:queued"] || !got["t:done"] || len(got) != 2 {
		t.Fatalf("TaskVideos returned %v, want only queued + completed", got)
	}
	for _, id := range []string{"t:media", "t:disc", "t:excl", "t:empty"} {
		if got[id] {
			t.Fatalf("non-download state %s leaked into download center", id)
		}
	}
	if total, _ := s.CountTasks(""); total != 2 {
		t.Fatalf("CountTasks=%d want 2", total)
	}
	if q, _ := s.TaskVideos(100, 0, "queued"); len(q) != 1 || q[0].ID != "t:queued" {
		t.Fatalf("queued filter=%v", q)
	}
	if n, _ := s.TaskVideos(100, 0, "removed"); len(n) != 0 {
		t.Fatalf("removed filter must be empty, got %v", n)
	}
}
