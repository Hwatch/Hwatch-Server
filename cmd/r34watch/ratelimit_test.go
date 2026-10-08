package main

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"r34watch/internal/adapters"
	"r34watch/internal/downloader"
	"r34watch/internal/store"
)

type fakeEngine struct {
	mu   sync.Mutex
	urls []string
}

func (f *fakeEngine) ID() string { return "aria2" }
func (f *fakeEngine) Add(_ context.Context, req downloader.Request) (string, error) {
	f.mu.Lock()
	f.urls = append(f.urls, req.URL)
	f.mu.Unlock()
	return req.ExternalID, nil
}
func (f *fakeEngine) Tell(context.Context, string) (downloader.Status, error) {
	return downloader.Status{}, errors.New("not found")
}
func (f *fakeEngine) Pause(context.Context, string) error         { return nil }
func (f *fakeEngine) Resume(context.Context, string) error        { return nil }
func (f *fakeEngine) Remove(context.Context, string) error        { return nil }
func (f *fakeEngine) PauseAll(context.Context) error              { return nil }
func (f *fakeEngine) ResumeAll(context.Context) error             { return nil }
func (f *fakeEngine) SetGlobal(context.Context, int, int64) error { return nil }
func (f *fakeEngine) Health(context.Context) downloader.Health {
	return downloader.Health{ID: "aria2", Online: true, SupportsMulti: true}
}

type fakeResolveAdapter struct {
	fakeThumbAdapter
	mu   sync.Mutex
	idx  int
	urls []string
}

func (f *fakeResolveAdapter) ResolveDownload(context.Context, store.Video, string) (adapters.DownloadSource, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	url := f.urls[f.idx%len(f.urls)]
	f.idx++
	return adapters.DownloadSource{URL: url, Quality: 1080}, nil
}

func TestRetryResolvesFreshDownloadURL(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "q.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.AddSubscription(store.Subscription{Slug: "rule34video:creator1", Name: "Creator", URL: "https://rule34video.com/models/creator1/", Platform: "rule34video", Type: "creator", SourceID: "creator1", Enabled: true, CreatedAt: store.Now()}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec("INSERT INTO videos(id,subscription_slug,platform,title,page_url,discovered_at,updated_at,status,duration_seconds,creator_url) VALUES(?,?,?,?,?,?,?,?,?,?)",
		"rule34video:1", "rule34video:creator1", "rule34video", "t", "https://rule34video.com/video/1/", store.Now(), store.Now(), "queued", 100, "https://rule34video.com/models/creator1/"); err != nil {
		t.Fatal(err)
	}
	cfg, _ := s.Settings()
	cfg.DownloadDir = t.TempDir()
	cfg.MinFreeBytes = 0
	_ = s.SaveSettings(cfg)

	a := New(s)
	engine := &fakeEngine{}
	a.engineOverride = engine
	adapter := &fakeResolveAdapter{urls: []string{"https://cdn.example/a_1080p.mp4?v-acctoken=AAA", "https://cdn.example/a_1080p.mp4?v-acctoken=BBB"}}
	a.adapters.Register(adapter)

	if err := a.queue("rule34video:1"); err != nil {
		t.Fatalf("first queue: %v", err)
	}
	// Simulate an aria2 429 retry: HWatch clears the downloader binding and
	// puts the task back in the queue.
	if _, err := s.DB.Exec("UPDATE videos SET status='queued',downloader_id='',external_task_id='',aria_gid='' WHERE id=?", "rule34video:1"); err != nil {
		t.Fatal(err)
	}
	if err := a.queue("rule34video:1"); err != nil {
		t.Fatalf("retry queue: %v", err)
	}

	engine.mu.Lock()
	got := append([]string(nil), engine.urls...)
	engine.mu.Unlock()
	if len(got) != 2 {
		t.Fatalf("engine submissions=%v, want 2 (one per attempt)", got)
	}
	if got[0] != adapter.urls[0] || got[1] != adapter.urls[1] {
		t.Fatalf("engine URLs=%v, want fresh signed URL on retry %v", got, adapter.urls)
	}
}

func TestTaskRateLimitRetryCounterResets(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "cap.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.AddSubscription(store.Subscription{Slug: "rule34video:c", Name: "c", URL: "https://rule34video.com/models/c/", Platform: "rule34video", Enabled: true, CreatedAt: store.Now()}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec("INSERT INTO videos(id,subscription_slug,platform,title,page_url,discovered_at,updated_at,status) VALUES(?,?,?,?,?,?,?,?)", "rule34video:9", "rule34video:c", "rule34video", "t", "https://rule34video.com/video/9/", store.Now(), store.Now(), "queued"); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= maxRateLimitRetries+1; i++ {
		n, err := s.AddTaskRateLimitRetry("rule34video:9")
		if err != nil {
			t.Fatal(err)
		}
		if n != i {
			t.Fatalf("retry %d: counter=%d", i, n)
		}
	}
	if err := s.ResetTaskRateLimitRetry("rule34video:9"); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.TaskRateLimitRetries("rule34video:9"); n != 0 {
		t.Fatalf("counter not reset: %d", n)
	}
	// Manual retry resets the counter too.
	_, _ = s.AddTaskRateLimitRetry("rule34video:9")
	_, _ = s.DB.Exec("UPDATE videos SET status='failed' WHERE id='rule34video:9'")
	if err := s.QueueVideo("rule34video:9"); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.TaskRateLimitRetries("rule34video:9"); n != 0 {
		t.Fatalf("manual retry must reset counter, got %d", n)
	}
}
