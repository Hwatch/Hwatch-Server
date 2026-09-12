package downloader

import (
	"context"
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
