package downloader

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestWgetDownloadsFile(t *testing.T) {
	if _, err := os.Stat("/usr/bin/wget"); err != nil {
		t.Skip("wget unavailable")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("hwatch wget test")) }))
	defer server.Close()
	w := NewWget()
	id, err := w.Add(context.Background(), Request{URL: server.URL, Directory: t.TempDir(), FileName: "video.bin"})
	if err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		status, err := w.Tell(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if status.State == "complete" {
			if status.Completed == 0 {
				t.Fatal("empty download")
			}
			return
		}
		if status.State == "error" {
			t.Fatal(status.ErrorMessage)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("wget did not finish")
}

func TestWgetCapacityIsRetryable(t *testing.T) {
	if _, err := os.Stat("/usr/bin/wget"); err != nil {
		t.Skip("wget unavailable")
	}
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-release
		_, _ = w.Write([]byte("done"))
	}))
	defer server.Close()
	defer close(release)
	w := NewWget()
	_ = w.SetGlobal(context.Background(), 1, 0)
	if _, err := w.Add(context.Background(), Request{URL: server.URL, Directory: t.TempDir(), FileName: "first.bin"}); err != nil {
		t.Fatal(err)
	}
	_, err := w.Add(context.Background(), Request{URL: server.URL, Directory: t.TempDir(), FileName: "second.bin"})
	if !errors.Is(err, ErrCapacityReached) {
		t.Fatalf("capacity error = %v", err)
	}
}

func TestWgetCapacityWaitWakesWhenTaskEnds(t *testing.T) {
	if _, err := os.Stat("/usr/bin/wget"); err != nil {
		t.Skip("wget unavailable")
	}
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { <-release; _, _ = w.Write([]byte("done")) }))
	defer server.Close()
	w := NewWget()
	_ = w.SetGlobal(context.Background(), 1, 0)
	if _, err := w.Add(context.Background(), Request{URL: server.URL, Directory: t.TempDir(), FileName: "first.bin"}); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() { done <- w.WaitForCapacity(ctx) }()
	select {
	case err := <-done:
		t.Fatalf("wait returned before capacity changed: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
