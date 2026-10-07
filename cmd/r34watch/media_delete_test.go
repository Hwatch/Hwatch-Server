package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"r34watch/internal/store"
)

func newAdminSession(t *testing.T, s *store.Store) (string, string) {
	t.Helper()
	uid, err := s.CreateFirstAdmin("admin", "x")
	if err != nil {
		t.Fatal(err)
	}
	cookie, csrf := "admin-cookie", "admin-csrf"
	if err := s.CreateSession(tokenHash(cookie), uid, csrf, time.Now().UTC().Add(time.Hour).Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	return cookie, csrf
}

func adminRequest(t *testing.T, client *http.Client, method, url, cookie, csrf string, body any) *http.Response {
	t.Helper()
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Cookie", "hwatch_session="+cookie)
	req.Header.Set("X-CSRF-Token", csrf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestDeleteAndExcludeFlow(t *testing.T) {
	mediaRoot := t.TempDir()
	a, s := newTransRunnerTestApp(t, mediaRoot)
	defer s.Close()
	seedTRVideo(t, s, mediaRoot, "t:d", 10, "sha-d", "A/d.mp4")
	cookie, csrf := newAdminSession(t, s)
	server := httptest.NewServer(a.router(mediaRoot))
	defer server.Close()
	client := server.Client()

	requireStatus(t, adminRequest(t, client, http.MethodPost, server.URL+"/api/videos/t:d/delete", cookie, csrf, map[string]any{"mode": "delete_and_exclude"}), http.StatusOK)
	if _, err := os.Stat(filepath.Join(mediaRoot, "A/d.mp4")); !os.IsNotExist(err) {
		t.Fatal("media file still present")
	}
	if v, _ := s.Video("t:d"); v.Status != "removed" {
		t.Fatalf("status=%q want removed", v.Status)
	}
	if !s.IsMediaExcluded("t:d") {
		t.Fatal("exclusion tombstone missing")
	}
	// second delete is idempotent and does not corrupt state
	requireStatus(t, adminRequest(t, client, http.MethodPost, server.URL+"/api/videos/t:d/delete", cookie, csrf, map[string]any{"mode": "delete_and_exclude"}), http.StatusOK)

	res := adminRequest(t, client, http.MethodGet, server.URL+"/api/media-exclusions", cookie, csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list status=%d", res.StatusCode)
	}
	var page struct {
		Items []store.MediaExclusion `json:"items"`
		Total int                    `json:"total"`
	}
	_ = json.NewDecoder(res.Body).Decode(&page)
	res.Body.Close()
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].VideoID != "t:d" {
		t.Fatalf("exclusions=%+v", page)
	}
	// restore removes tombstone only (no auto download)
	requireStatus(t, adminRequest(t, client, http.MethodDelete, server.URL+"/api/media-exclusions/t:d", cookie, csrf, nil), http.StatusOK)
	if s.IsMediaExcluded("t:d") {
		t.Fatal("tombstone not removed")
	}
	if v, _ := s.Video("t:d"); v.Status == "queued" || v.Status == "active" {
		t.Fatalf("restore must not auto-download, status=%q", v.Status)
	}
}

func TestDeleteOnlyNoExclusion(t *testing.T) {
	mediaRoot := t.TempDir()
	a, s := newTransRunnerTestApp(t, mediaRoot)
	defer s.Close()
	seedTRVideo(t, s, mediaRoot, "t:o", 10, "sha-o", "A/o.mp4")
	cookie, csrf := newAdminSession(t, s)
	server := httptest.NewServer(a.router(mediaRoot))
	defer server.Close()
	requireStatus(t, adminRequest(t, server.Client(), http.MethodPost, server.URL+"/api/videos/t:o/delete", cookie, csrf, map[string]any{"mode": "delete_only"}), http.StatusOK)
	if _, err := os.Stat(filepath.Join(mediaRoot, "A/o.mp4")); !os.IsNotExist(err) {
		t.Fatal("media file still present")
	}
	if s.IsMediaExcluded("t:o") {
		t.Fatal("delete_only must not create an exclusion")
	}
	if v, _ := s.Video("t:o"); v.Status != "removed" {
		t.Fatalf("status=%q want removed", v.Status)
	}
}

func TestDeleteRejectsEscape(t *testing.T) {
	mediaRoot := t.TempDir()
	a, s := newTransRunnerTestApp(t, mediaRoot)
	defer s.Close()
	outside := filepath.Join(t.TempDir(), "outside.mp4")
	if err := os.WriteFile(outside, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	seedTRVideo(t, s, mediaRoot, "t:e", 10, "sha-e", "A/e.mp4")
	if _, err := s.DB.Exec("UPDATE videos SET file_path=?,plex_path=? WHERE id='t:e'", outside, outside); err != nil {
		t.Fatal(err)
	}
	cookie, csrf := newAdminSession(t, s)
	server := httptest.NewServer(a.router(mediaRoot))
	defer server.Close()
	requireStatus(t, adminRequest(t, server.Client(), http.MethodPost, server.URL+"/api/videos/t:e/delete", cookie, csrf, map[string]any{"mode": "delete_only"}), http.StatusConflict)
	if _, err := os.Stat(outside); err != nil {
		t.Fatal("outside file must not be deleted")
	}
}

func TestServiceTokenCannotDelete(t *testing.T) {
	mediaRoot := t.TempDir()
	a, s := newTransRunnerTestApp(t, mediaRoot)
	defer s.Close()
	seedTRVideo(t, s, mediaRoot, "t:s", 10, "sha-s", "A/s.mp4")
	token := createTRToken(t, s, []string{"read_media", "write_transcode_results"})
	server := httptest.NewServer(a.router(mediaRoot))
	defer server.Close()
	requireStatus(t, trRequest(t, server.Client(), http.MethodPost, server.URL+"/api/videos/t:s/delete", token, map[string]any{"mode": "delete_and_exclude"}), http.StatusUnauthorized)
	requireStatus(t, trRequest(t, server.Client(), http.MethodGet, server.URL+"/api/media-exclusions", token, nil), http.StatusUnauthorized)
	if _, err := os.Stat(filepath.Join(mediaRoot, "A/s.mp4")); err != nil {
		t.Fatal("service token must not delete media")
	}
}

func TestExclusionBlocksQueue(t *testing.T) {
	mediaRoot := t.TempDir()
	a, s := newTransRunnerTestApp(t, mediaRoot)
	defer s.Close()
	seedTRVideo(t, s, mediaRoot, "t:q", 10, "sha-q", "A/q.mp4")
	if err := s.AddMediaExclusion(store.MediaExclusion{VideoID: "t:q", Source: "test", Reason: "user_delete"}); err != nil {
		t.Fatal(err)
	}
	if err := a.queue("t:q"); err == nil {
		t.Fatal("queue must reject excluded video")
	}
}
