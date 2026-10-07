package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newThumbServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/ok.jpg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("fake-jpeg-bytes"))
	})
	mux.HandleFunc("/missing.jpg", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	})
	return httptest.NewServer(mux)
}

func TestThumbnailEndpointCoverBehavior(t *testing.T) {
	mediaRoot := t.TempDir()
	a, s := newTransRunnerTestApp(t, mediaRoot)
	defer s.Close()
	cookie, csrf := newAdminSession(t, s)
	server := httptest.NewServer(a.router(mediaRoot))
	defer server.Close()
	client := server.Client()
	img := newThumbServer(t)
	defer img.Close()

	seedTRVideo(t, s, mediaRoot, "t:ok", 10, "sha-ok", "A/ok.mp4")
	seedTRVideo(t, s, mediaRoot, "t:missing", 10, "sha-missing", "A/missing.mp4")
	seedTRVideo(t, s, mediaRoot, "t:empty", 10, "sha-empty", "A/empty.mp4")
	seedTRVideo(t, s, mediaRoot, "t:data", 10, "sha-data", "A/data.mp4")
	if _, err := s.DB.Exec("UPDATE videos SET thumbnail_url=? WHERE id='t:ok'", img.URL+"/ok.jpg"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec("UPDATE videos SET thumbnail_url=? WHERE id='t:missing'", img.URL+"/missing.jpg"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec("UPDATE videos SET thumbnail_url='' WHERE id='t:empty'"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec("UPDATE videos SET thumbnail_url='data:image/gif;base64,AAA' WHERE id='t:data'"); err != nil {
		t.Fatal(err)
	}

	get := func(id string) (int, string) {
		res := adminRequest(t, client, http.MethodGet, server.URL+"/thumb/"+id, cookie, csrf, nil)
		defer res.Body.Close()
		return res.StatusCode, res.Header.Get("Content-Type")
	}

	if code, kind := get("t:ok"); code != http.StatusOK || kind != "image/jpeg" {
		t.Fatalf("ok cover: code=%d type=%q", code, kind)
	}
	// Missing data and failed remote load both fall back to a placeholder, never 404 / broken image.
	if code, kind := get("t:empty"); code != http.StatusOK || kind != "image/svg+xml; charset=utf-8" {
		t.Fatalf("empty cover: code=%d type=%q", code, kind)
	}
	if code, kind := get("t:missing"); code != http.StatusOK || kind != "image/svg+xml; charset=utf-8" {
		t.Fatalf("failed remote cover: code=%d type=%q", code, kind)
	}
	if code, kind := get("t:data"); code != http.StatusOK || kind != "image/svg+xml; charset=utf-8" {
		t.Fatalf("data placeholder cover: code=%d type=%q", code, kind)
	}
	// unknown video stays 404
	if code, _ := get("t:nope"); code != http.StatusNotFound {
		t.Fatalf("unknown video code=%d", code)
	}
}
