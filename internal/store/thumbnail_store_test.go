package store

import (
	"path/filepath"
	"testing"
)

func seedLibraryVideo(t *testing.T, s *Store, id, platform, status, thumb string) {
	t.Helper()
	if _, err := s.DB.Exec("INSERT INTO videos(id,subscription_slug,platform,title,page_url,discovered_at,updated_at,status,thumbnail_url,file_path,plex_path,video_codec,width,height) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		id, "t:sub", platform, id, "https://example.invalid/v/"+id, Now(), Now(), status, thumb, "/media/"+id+".mp4", "/plex/"+id+".mp4", "h264", 1920, 1080); err != nil {
		t.Fatal(err)
	}
}

func newLibraryStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.AddSubscription(Subscription{Slug: "t:sub", Name: "s", URL: "https://example.invalid/sub", Enabled: true, CreatedAt: Now()}); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCountDownloadedVideosFormalMedia(t *testing.T) {
	s := newLibraryStore(t)
	seedLibraryVideo(t, s, "rule34video:1", "rule34video", "completed", "https://img/1.jpg")
	seedLibraryVideo(t, s, "rule34video:2", "rule34video", "completed", "https://img/2.jpg")
	seedLibraryVideo(t, s, "hanime1:3", "hanime1", "completed", "https://img/3.jpg") // TransRunner HEVC stays completed
	seedLibraryVideo(t, s, "rule34video:4", "rule34video", "discovered", "")
	seedLibraryVideo(t, s, "rule34video:5", "rule34video", "queued", "")
	seedLibraryVideo(t, s, "rule34video:6", "rule34video", "removed", "")
	seedLibraryVideo(t, s, "rule34video:7", "rule34video", "excluded", "")
	seedLibraryVideo(t, s, "rule34video:8", "rule34video", "failed", "")
	seedLibraryVideo(t, s, "test:md:source", "test", "completed", "https://img/t.jpg")

	n, err := s.CountDownloadedVideos()
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("CountDownloadedVideos=%d want 3 (two r34 + hanime, no test/discovered/queued/removed/excluded/failed)", n)
	}

	items, err := s.DownloadedVideos("rule34video", 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("DownloadedVideos(rule34video)=%d want 2", len(items))
	}
	for _, v := range items {
		if v.ID == "test:md:source" || v.Status != "completed" {
			t.Fatalf("unexpected library row %+v", v)
		}
	}
}

func TestUpdateThumbnailOnlyTouchesCover(t *testing.T) {
	s := newLibraryStore(t)
	seedLibraryVideo(t, s, "rule34video:9", "rule34video", "completed", "https://img/old.jpg")
	before, err := s.Video("rule34video:9")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateThumbnail(before.ID, "https://img/new.jpg"); err != nil {
		t.Fatal(err)
	}
	after, err := s.Video(before.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.ThumbnailURL != "https://img/new.jpg" {
		t.Fatalf("thumbnail=%q", after.ThumbnailURL)
	}
	if after.FilePath != before.FilePath || after.PlexPath != before.PlexPath || after.VideoCodec != before.VideoCodec || after.Width != before.Width || after.Height != before.Height || after.Status != before.Status {
		t.Fatalf("UpdateThumbnail changed non-cover fields: before=%+v after=%+v", before, after)
	}
}
