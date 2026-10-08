package store

import (
	"fmt"
	"testing"
)

func randomIDs(items []Video) map[string]bool {
	out := map[string]bool{}
	for _, v := range items {
		out[v.ID] = true
	}
	return out
}

// TestRandomVideosOnlyReturnsFormalLibraryMedia verifies the random discovery
// query never surfaces removed, excluded, unfinished, test-fixture or
// path-less rows, and that it draws from the whole library rather than one page.
func TestRandomVideosOnlyReturnsFormalLibraryMedia(t *testing.T) {
	s := newLibraryStore(t)
	seedLibraryVideo(t, s, "rule34video:1", "rule34video", "completed", "https://img/1.jpg")
	seedLibraryVideo(t, s, "rule34video:2", "rule34video", "completed", "https://img/2.jpg")
	seedLibraryVideo(t, s, "hanime1:3", "hanime1", "completed", "https://img/3.jpg")
	seedLibraryVideo(t, s, "rule34video:4", "rule34video", "discovered", "")
	seedLibraryVideo(t, s, "rule34video:5", "rule34video", "queued", "")
	seedLibraryVideo(t, s, "rule34video:6", "rule34video", "removed", "")
	seedLibraryVideo(t, s, "rule34video:7", "rule34video", "excluded", "")
	seedLibraryVideo(t, s, "rule34video:8", "rule34video", "failed", "")
	seedLibraryVideo(t, s, "rule34video:9", "rule34video", "completed", "")
	seedLibraryVideo(t, s, "test:md:source", "test", "completed", "https://img/t.jpg")
	// A completed row whose media path is empty must not be recommended.
	if _, err := s.DB.Exec("UPDATE videos SET file_path='' WHERE id='rule34video:9'"); err != nil {
		t.Fatal(err)
	}

	items, err := s.RandomVideos(50, VideoSearch{Language: "zh-CN"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("RandomVideos returned %d rows, want 3 formal media", len(items))
	}
	want := map[string]bool{"rule34video:1": true, "rule34video:2": true, "hanime1:3": true}
	seen := map[string]bool{}
	for _, v := range items {
		if !want[v.ID] {
			t.Fatalf("unexpected random row %q status=%q path=%q", v.ID, v.Status, v.FilePath)
		}
		if v.Status != "completed" || v.FilePath == "" {
			t.Fatalf("random row not formal media: %+v", v)
		}
		if seen[v.ID] {
			t.Fatalf("duplicate row %q within one batch", v.ID)
		}
		seen[v.ID] = true
	}
}

func TestRandomVideosRespectsExcludeAndLimit(t *testing.T) {
	s := newLibraryStore(t)
	for i, id := range []string{"rule34video:1", "rule34video:2", "rule34video:3"} {
		seedLibraryVideo(t, s, id, "rule34video", "completed", "https://img/x.jpg")
		if i == 0 {
			if _, err := s.DB.Exec("UPDATE videos SET creator_name='Alice', height=720 WHERE id=?", id); err != nil {
				t.Fatal(err)
			}
		} else {
			if _, err := s.DB.Exec("UPDATE videos SET creator_name='Bob', height=1080 WHERE id=?", id); err != nil {
				t.Fatal(err)
			}
		}
	}

	// exclude removes a candidate without shrinking the pool to a fixed page.
	items, err := s.RandomVideos(6, VideoSearch{Language: "zh-CN"}, []string{"rule34video:1"})
	if err != nil {
		t.Fatal(err)
	}
	if ids := randomIDs(items); len(ids) != 2 || ids["rule34video:1"] {
		t.Fatalf("exclude not honored: %v", ids)
	}

	// Filtering narrows the random pool with library semantics.
	items, err = s.RandomVideos(6, VideoSearch{Language: "zh-CN", MinHeight: 1080}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ids := randomIDs(items); len(ids) != 2 || ids["rule34video:1"] {
		t.Fatalf("min_height filter not honored: %v", ids)
	}

	items, err = s.RandomVideos(6, VideoSearch{Language: "zh-CN", Creator: "Alice"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ids := randomIDs(items); len(ids) != 1 || !ids["rule34video:1"] {
		t.Fatalf("creator filter not honored: %v", ids)
	}

	// limit bounds the batch.
	items, err = s.RandomVideos(2, VideoSearch{Language: "zh-CN"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("limit not honored: got %d want 2", len(items))
	}
}

func TestRandomVideosEmptyLibrary(t *testing.T) {
	s := newLibraryStore(t)
	items, err := s.RandomVideos(6, VideoSearch{Language: "zh-CN"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("empty library returned %d rows", len(items))
	}
}

// TestRandomVideosLimitAndExcludeBounds covers the boundary inputs that a
// client could send: non-positive limits and an oversized exclusion list.
func TestRandomVideosLimitAndExcludeBounds(t *testing.T) {
	s := newLibraryStore(t)
	seedLibraryVideo(t, s, "rule34video:1", "rule34video", "completed", "https://img/1.jpg")

	for _, limit := range []int{0, -1} {
		items, err := s.RandomVideos(limit, VideoSearch{Language: "zh-CN"}, nil)
		if err != nil {
			t.Fatalf("limit %d: %v", limit, err)
		}
		if len(items) != 0 {
			t.Fatalf("limit %d returned %d rows", limit, len(items))
		}
	}

	// A huge exclusion list must be safely capped (no SQLite variable error)
	// and must not exclude the real candidate when it falls beyond the cap.
	huge := make([]string, 0, MaxRandomExcludeIDs+50)
	for i := 0; i < MaxRandomExcludeIDs+50; i++ {
		huge = append(huge, fmt.Sprintf("rule34video:other-%d", i))
	}
	items, err := s.RandomVideos(6, VideoSearch{Language: "zh-CN"}, huge)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "rule34video:1" {
		t.Fatalf("oversized exclude list mishandled: %v", randomIDs(items))
	}
}
