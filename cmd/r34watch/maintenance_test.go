package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"r34watch/internal/adapters"
	"r34watch/internal/store"
)

type fakeThumbAdapter struct {
	thumb string
	err   error
	calls atomic.Int64
}

func (f *fakeThumbAdapter) ID() string   { return "rule34video" }
func (f *fakeThumbAdapter) Name() string { return "fake" }
func (f *fakeThumbAdapter) MatchURL(string) bool {
	return false
}
func (f *fakeThumbAdapter) SearchCreators(context.Context, string, string) ([]adapters.CreatorResult, error) {
	return nil, nil
}
func (f *fakeThumbAdapter) ParseSubscription(context.Context, string, string) (store.Subscription, error) {
	return store.Subscription{}, nil
}
func (f *fakeThumbAdapter) ListVideos(context.Context, store.Subscription, string) ([]store.Video, error) {
	return nil, nil
}
func (f *fakeThumbAdapter) FetchDetails(_ context.Context, v store.Video, _ string) (store.Video, error) {
	f.calls.Add(1)
	if f.err != nil {
		return store.Video{}, f.err
	}
	return store.Video{ID: v.ID, ThumbnailURL: f.thumb}, nil
}
func (f *fakeThumbAdapter) ResolveDownload(context.Context, store.Video, string) (adapters.DownloadSource, error) {
	return adapters.DownloadSource{}, nil
}
func (f *fakeThumbAdapter) ParseImported(string) (store.Video, error) { return store.Video{}, nil }

func newThumbRefreshApp(t *testing.T, fake *fakeThumbAdapter) (*App, *store.Store) {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "thumb.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.AddSubscription(store.Subscription{Slug: "t:sub", Name: "s", URL: "https://example.invalid/sub", Enabled: true, CreatedAt: store.Now()}); err != nil {
		t.Fatal(err)
	}
	a := New(s)
	a.adapters.Register(fake)
	return a, s
}

func seedRefreshMedia(t *testing.T, s *store.Store, id, status, thumb string) store.Video {
	t.Helper()
	if _, err := s.DB.Exec("INSERT INTO videos(id,subscription_slug,platform,title,page_url,discovered_at,updated_at,status,thumbnail_url,file_path,plex_path,video_codec,width,height) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		id, "t:sub", "rule34video", id, "https://example.invalid/v/"+id, store.Now(), store.Now(), status, thumb, "/media/"+id+".mp4", "/plex/"+id+".mp4", "h264", 1080, 1920); err != nil {
		t.Fatal(err)
	}
	v, err := s.Video(id)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func testSettings(t *testing.T, s *store.Store) store.Settings {
	t.Helper()
	cfg, _ := s.Settings()
	cfg.DownloadDir = t.TempDir()
	return cfg
}

func TestChooseThumbnailRules(t *testing.T) {
	if _, ok := chooseThumbnail("missing", "", "https://img/x.jpg"); !ok {
		t.Fatal("missing cover must be updated")
	}
	if _, ok := chooseThumbnail("force", "https://img/a_1280x720_1.jpg", "https://img/a_320x180_1.jpg"); ok {
		t.Fatal("force must not downgrade a larger cover")
	}
	if _, ok := chooseThumbnail("force", "", "data:image/gif;base64,AAA"); ok {
		t.Fatal("must not store a placeholder")
	}
	if area, ok := thumbnailArea("https://rule34video.com/contents/videos_screenshots/1/2/320x180/7.jpg"); !ok || area != 320*180 {
		t.Fatalf("thumbnailArea=%d ok=%v", area, ok)
	}
}

func TestRefreshThumbnailMissingPlaceholderUpdated(t *testing.T) {
	fake := &fakeThumbAdapter{thumb: "https://rule34video.com/contents/videos_screenshots/1/2/1280x720/1.jpg"}
	a, s := newThumbRefreshApp(t, fake)
	v := seedRefreshMedia(t, s, "rule34video:1", "completed", "")
	res := a.refreshThumbnail(context.Background(), v, "missing", testSettings(t, s), thumbnailHTTPClient(""))
	if res.Status != thumbUpdated {
		t.Fatalf("status=%s", res.Status)
	}
	after, _ := s.Video(v.ID)
	if after.ThumbnailURL != fake.thumb {
		t.Fatalf("thumbnail=%q", after.ThumbnailURL)
	}
	if after.FilePath != v.FilePath || after.PlexPath != v.PlexPath || after.Status != v.Status || after.VideoCodec != v.VideoCodec || after.Width != v.Width || after.Height != v.Height {
		t.Fatalf("refresh changed non-cover fields: %+v", after)
	}
}

func TestRefreshThumbnailDataPlaceholderUpdated(t *testing.T) {
	fake := &fakeThumbAdapter{thumb: "https://rule34video.com/contents/videos_screenshots/3/4/1280x720/2.jpg"}
	a, s := newThumbRefreshApp(t, fake)
	v := seedRefreshMedia(t, s, "rule34video:2", "completed", "data:image/gif;base64,AAA")
	res := a.refreshThumbnail(context.Background(), v, "missing", testSettings(t, s), thumbnailHTTPClient(""))
	if res.Status != thumbUpdated {
		t.Fatalf("status=%s", res.Status)
	}
}

func TestRefreshThumbnailForceRefetchesValidCover(t *testing.T) {
	fake := &fakeThumbAdapter{thumb: "https://rule34video.com/contents/videos_screenshots/5/6/1280x720/1.jpg"}
	a, s := newThumbRefreshApp(t, fake)
	old := "https://rule34video.com/contents/videos_screenshots/5/6/320x180/1.jpg"
	v := seedRefreshMedia(t, s, "rule34video:3", "completed", old)
	res := a.refreshThumbnail(context.Background(), v, "force", testSettings(t, s), thumbnailHTTPClient(""))
	if res.Status != thumbUpdated || fake.calls.Load() != 1 {
		t.Fatalf("status=%s calls=%d", res.Status, fake.calls.Load())
	}
}

func TestRefreshThumbnailFetchFailureKeepsValue(t *testing.T) {
	fake := &fakeThumbAdapter{err: errors.New("boom")}
	a, s := newThumbRefreshApp(t, fake)
	v := seedRefreshMedia(t, s, "rule34video:4", "completed", "data:image/gif;base64,AAA")
	res := a.refreshThumbnail(context.Background(), v, "missing", testSettings(t, s), thumbnailHTTPClient(""))
	if res.Status != thumbFailed {
		t.Fatalf("status=%s", res.Status)
	}
	after, _ := s.Video(v.ID)
	if after.ThumbnailURL != v.ThumbnailURL || after.FilePath != v.FilePath || after.Status != v.Status {
		t.Fatalf("failed refresh must not modify row: %+v", after)
	}
}

func TestRefreshThumbnailMissingModeSkipsLiveCover(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ok.jpg" {
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	fake := &fakeThumbAdapter{thumb: "https://rule34video.com/contents/videos_screenshots/7/8/1280x720/1.jpg"}
	a, s := newThumbRefreshApp(t, fake)
	good := seedRefreshMedia(t, s, "rule34video:5", "completed", srv.URL+"/ok.jpg")
	if res := a.refreshThumbnail(context.Background(), good, "missing", testSettings(t, s), thumbnailHTTPClient("")); res.Status != thumbUnchanged || fake.calls.Load() != 0 {
		t.Fatalf("valid cover must be skipped, status=%s calls=%d", res.Status, fake.calls.Load())
	}
	bad := seedRefreshMedia(t, s, "rule34video:6", "completed", srv.URL+"/page.html")
	if res := a.refreshThumbnail(context.Background(), bad, "missing", testSettings(t, s), thumbnailHTTPClient("")); res.Status != thumbUpdated {
		t.Fatalf("non-image cover must be refreshed, status=%s", res.Status)
	}
}

func TestRunThumbnailRefreshIsIdempotent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	fake := &fakeThumbAdapter{thumb: "https://rule34video.com/contents/videos_screenshots/9/9/1280x720/1.jpg"}
	a, s := newThumbRefreshApp(t, fake)
	seedRefreshMedia(t, s, "rule34video:7", "completed", "")
	seedRefreshMedia(t, s, "rule34video:8", "completed", srv.URL+"/ok.jpg")
	cfg, _ := s.Settings()
	cfg.DownloadDir = t.TempDir()
	_ = s.SaveSettings(cfg)
	if err := a.runThumbnailRefresh(context.Background(), thumbRefreshOptions{Source: "rule34video", Mode: "missing", Limit: 50, Concurrency: 1, DelayMS: 0}); err != nil {
		t.Fatal(err)
	}
	if fake.calls.Load() != 1 {
		t.Fatalf("only the placeholder should be fetched, calls=%d", fake.calls.Load())
	}
}

func TestRefreshThumbnailAPIRegressionAuthAndUpdate(t *testing.T) {
	fake := &fakeThumbAdapter{thumb: "https://rule34video.com/contents/videos_screenshots/1/1/1280x720/1.jpg"}
	a, s := newThumbRefreshApp(t, fake)
	v := seedRefreshMedia(t, s, "rule34video:9", "completed", "")
	cookie, csrf := newAdminSession(t, s)
	server := httptest.NewServer(a.router(t.TempDir()))
	defer server.Close()

	res := adminRequest(t, server.Client(), http.MethodPost, server.URL+"/api/videos/"+v.ID+"/refresh-thumbnail", cookie, csrf, nil)
	if res.StatusCode != http.StatusOK {
		buf := make([]byte, 512)
		n, _ := res.Body.Read(buf)
		t.Fatalf("status=%d body=%s", res.StatusCode, string(buf[:n]))
	}
	var body struct {
		Status string `json:"status"`
		Before string `json:"before"`
		After  string `json:"after"`
	}
	_ = json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()
	if body.Status != thumbUpdated || body.After != fake.thumb {
		t.Fatalf("body=%+v", body)
	}

	// Service token must not be able to call this management endpoint.
	srv2 := httptest.NewServer(a.router(t.TempDir()))
	defer srv2.Close()
	token := createTRToken(t, s, []string{"read_media"})
	requireStatus(t, trRequest(t, srv2.Client(), http.MethodPost, srv2.URL+"/api/videos/"+v.ID+"/refresh-thumbnail", token, nil), http.StatusUnauthorized)
	requireStatus(t, trRequest(t, srv2.Client(), http.MethodPost, srv2.URL+"/api/videos/"+v.ID+"/refresh-thumbnail", "", nil), http.StatusUnauthorized)
}
