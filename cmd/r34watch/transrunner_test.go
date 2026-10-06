package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"r34watch/internal/store"
)

func newTransRunnerTestApp(t *testing.T, mediaRoot string) (*App, *store.Store) {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "tr.db"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := s.Settings()
	cfg.DownloadDir = mediaRoot
	if err := s.SaveSettings(cfg); err != nil {
		t.Fatal(err)
	}
	return New(s), s
}

func seedTRVideo(t *testing.T, s *store.Store, mediaRoot, id string, duration int64, sha, rel string) store.Video {
	t.Helper()
	if _, err := s.DB.Exec("INSERT OR IGNORE INTO subscriptions(slug,name,url,platform,subscription_type,source_id,created_at,enabled,baseline_complete) VALUES('test:sub','sub','https://example.invalid/sub','test','creator','',?,1,1)", store.Now()); err != nil {
		t.Fatal(err)
	}
	v := store.Video{ID: id, SubscriptionSlug: "test:sub", Title: id, PageURL: "https://example.invalid/v/" + id}
	if _, err := s.UpsertDiscovered(v); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(mediaRoot, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("payload-"+id), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec("UPDATE videos SET status='completed',duration_seconds=?,file_path=?,plex_path=?,downloaded_at=?,updated_at=? WHERE id=?", duration, path, path, store.Now(), store.Now(), id); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveFileHashes(id, 10, "q-"+id, sha); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Video(id)
	return got
}

func createTRToken(t *testing.T, s *store.Store, scopes []string) string {
	t.Helper()
	token, err := randomToken()
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(scopes)
	if _, err := s.CreateIntegrationToken("transrunner", tokenHash(token), string(raw)); err != nil {
		t.Fatal(err)
	}
	return token
}

func trRequest(t *testing.T, client *http.Client, method, url, token string, body any) *http.Response {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func requireStatus(t *testing.T, res *http.Response, want int) {
	t.Helper()
	defer res.Body.Close()
	if res.StatusCode != want {
		buf := make([]byte, 512)
		n, _ := res.Body.Read(buf)
		t.Fatalf("status=%d want %d body=%s", res.StatusCode, want, string(buf[:n]))
	}
}

func TestTransRunnerServiceAPISecurity(t *testing.T) {
	mediaRoot := t.TempDir()
	a, s := newTransRunnerTestApp(t, mediaRoot)
	defer s.Close()
	seedTRVideo(t, s, mediaRoot, "t:v", 100, "sha-v", "A/v.mp4")
	full := createTRToken(t, s, []string{"read_media"})
	server := httptest.NewServer(a.router(mediaRoot))
	defer server.Close()
	client := server.Client()
	url := server.URL + "/api/transrunner/v1/media/t:v"

	requireStatus(t, trRequest(t, client, http.MethodGet, url, "", nil), http.StatusUnauthorized)
	requireStatus(t, trRequest(t, client, http.MethodGet, url, "wrong-token", nil), http.StatusUnauthorized)
	requireStatus(t, trRequest(t, client, http.MethodGet, url, full, nil), http.StatusOK)

	narrow := createTRToken(t, s, []string{"read_dedupe_candidates"})
	requireStatus(t, trRequest(t, client, http.MethodGet, url, narrow, nil), http.StatusForbidden)

	// revoke full token -> 401
	if _, err := s.RevokeIntegrationTokens("transrunner"); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, trRequest(t, client, http.MethodGet, url, full, nil), http.StatusUnauthorized)
}

func TestTransRunnerMediaAndCandidates(t *testing.T) {
	mediaRoot := t.TempDir()
	a, s := newTransRunnerTestApp(t, mediaRoot)
	defer s.Close()
	target := seedTRVideo(t, s, mediaRoot, "t:target", 100, "sha-target", "A/target.mp4")
	seedTRVideo(t, s, mediaRoot, "t:near", 101, "sha-near", "A/near.mp4")
	seedTRVideo(t, s, mediaRoot, "t:far", 400, "sha-far", "A/far.mp4")
	seedTRVideo(t, s, mediaRoot, "t:exact", 100, "sha-target", "A/exact.mp4")
	token := createTRToken(t, s, []string{"read_media", "read_dedupe_candidates"})
	server := httptest.NewServer(a.router(mediaRoot))
	defer server.Close()
	client := server.Client()

	res := trRequest(t, client, http.MethodGet, server.URL+"/api/transrunner/v1/media/"+target.ID, token, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("media status=%d", res.StatusCode)
	}
	var media map[string]any
	_ = json.NewDecoder(res.Body).Decode(&media)
	res.Body.Close()
	if media["media_relative_path"] != "A/target.mp4" {
		t.Fatalf("relative path=%v", media["media_relative_path"])
	}
	if body, _ := json.Marshal(media); bytes.Contains(body, []byte(mediaRoot)) {
		t.Fatalf("response leaked absolute path: %s", body)
	}

	res = trRequest(t, client, http.MethodGet, server.URL+"/api/transrunner/v1/media/"+target.ID+"/dedupe-candidates", token, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("candidates status=%d", res.StatusCode)
	}
	var cands struct {
		AlgorithmVersion string `json:"algorithm_version"`
		Candidates       []struct {
			VideoID string `json:"video_id"`
		} `json:"candidates"`
	}
	_ = json.NewDecoder(res.Body).Decode(&cands)
	res.Body.Close()
	if cands.AlgorithmVersion != trAlgorithmVersion || len(cands.Candidates) != 1 || cands.Candidates[0].VideoID != "t:near" {
		t.Fatalf("candidates=%+v", cands)
	}
}

func TestTransRunnerDuplicateResults(t *testing.T) {
	mediaRoot := t.TempDir()
	a, s := newTransRunnerTestApp(t, mediaRoot)
	defer s.Close()
	seedTRVideo(t, s, mediaRoot, "t:a", 100, "sha-a", "A/a.mp4")
	seedTRVideo(t, s, mediaRoot, "t:b", 101, "sha-b", "A/b.mp4")
	seedTRVideo(t, s, mediaRoot, "t:c", 102, "sha-c", "A/c.mp4")
	seedTRVideo(t, s, mediaRoot, "t:d", 103, "sha-d", "A/d.mp4")
	token := createTRToken(t, s, []string{"write_duplicate_results"})
	server := httptest.NewServer(a.router(mediaRoot))
	defer server.Close()
	client := server.Client()
	url := server.URL + "/api/transrunner/v1/duplicate-results"
	post := func(id, match, classification string) *http.Response {
		return trRequest(t, client, http.MethodPost, url, token, map[string]any{
			"video_id": id,
			"matches":  []map[string]any{{"video_id": match, "classification": classification, "backend": "phash", "confidence": 0.9, "metrics": map[string]any{"sample_count": 32}}},
		})
	}

	requireStatus(t, post("t:a", "t:b", "whole-video"), http.StatusOK)
	requireStatus(t, post("t:a", "t:c", "trimmed-video"), http.StatusOK)
	requireStatus(t, post("t:b", "t:c", "possible-match"), http.StatusOK)
	requireStatus(t, post("t:a", "t:d", "different"), http.StatusOK) // no candidate

	var count int
	if err := s.DB.QueryRow("SELECT count(*) FROM duplicate_candidates WHERE match_type='media_duplicate'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("candidate count=%d want 3", count)
	}

	// A-B / B-A idempotent
	post("t:a", "t:b", "whole-video").Body.Close()
	post("t:b", "t:a", "whole-video").Body.Close()
	if err := s.DB.QueryRow("SELECT count(*) FROM duplicate_candidates WHERE match_type='media_duplicate'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("candidate count after both directions=%d want 3", count)
	}

	requireStatus(t, post("t:a", "t:b", "bogus"), http.StatusUnprocessableEntity)
	requireStatus(t, post("t:a", "t:missing", "whole-video"), http.StatusUnprocessableEntity)
	requireStatus(t, trRequest(t, client, http.MethodPost, url, token, map[string]any{"video_id": "t:missing", "matches": []map[string]any{}}), http.StatusNotFound)
}

func TestTransRunnerTranscodeResults(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not available")
	}
	mediaRoot := t.TempDir()
	a, s := newTransRunnerTestApp(t, mediaRoot)
	defer s.Close()
	source := seedTRVideo(t, s, mediaRoot, "t:src", 5, "sha-source", "A/source.mp4")
	staging := filepath.Join(mediaRoot, transRunnerStagingDir, "job_1")
	if err := os.MkdirAll(staging, 0755); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(staging, "output.mp4")
	if out, err := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "testsrc=size=64x64:rate=5:duration=1", "-c:v", "libx264", "-pix_fmt", "yuv420p", output).CombinedOutput(); err != nil {
		t.Skipf("ffmpeg failed: %v %s", err, out)
	}
	token := createTRToken(t, s, []string{"write_transcode_results"})
	server := httptest.NewServer(a.router(mediaRoot))
	defer server.Close()
	client := server.Client()
	url := server.URL + "/api/transrunner/v1/transcode-results"

	valid := map[string]any{"video_id": source.ID, "runner_job_id": "job_1", "source_sha256": "sha-source", "output_relative_path": ".transrunner/job_1/output.mp4", "profile": "hevc-1080p"}
	requireStatus(t, trRequest(t, client, http.MethodPost, url, token, valid), http.StatusAccepted)

	// source changed -> 409
	changed := map[string]any{"video_id": source.ID, "source_sha256": "different", "output_relative_path": ".transrunner/job_1/output.mp4"}
	requireStatus(t, trRequest(t, client, http.MethodPost, url, token, changed), http.StatusConflict)

	// path traversal / absolute / outside staging -> 422
	for _, p := range []string{"../escape.mp4", "/etc/passwd", "Other/output.mp4", ".transrunner/../../etc/passwd", "output.mp4"} {
		requireStatus(t, trRequest(t, client, http.MethodPost, url, token, map[string]any{"video_id": source.ID, "output_relative_path": p}), http.StatusUnprocessableEntity)
	}
	// missing output -> 422
	requireStatus(t, trRequest(t, client, http.MethodPost, url, token, map[string]any{"video_id": source.ID, "output_relative_path": ".transrunner/job_none/output.mp4"}), http.StatusUnprocessableEntity)
	// probe failure -> 422
	if err := os.WriteFile(filepath.Join(staging, "notvideo.mp4"), []byte("not a video"), 0644); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, trRequest(t, client, http.MethodPost, url, token, map[string]any{"video_id": source.ID, "output_relative_path": ".transrunner/job_1/notvideo.mp4"}), http.StatusUnprocessableEntity)

	// symlink escape -> 422
	outside := filepath.Join(t.TempDir(), "outside.mp4")
	if err := os.WriteFile(outside, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(staging, "link.mp4")
	if err := os.Symlink(outside, link); err == nil {
		requireStatus(t, trRequest(t, client, http.MethodPost, url, token, map[string]any{"video_id": source.ID, "output_relative_path": ".transrunner/job_1/link.mp4"}), http.StatusUnprocessableEntity)
	}

	// source must NOT be replaced
	got, err := os.ReadFile(filepath.Join(mediaRoot, "A/source.mp4"))
	if err != nil || string(got) != "payload-"+source.ID {
		t.Fatalf("source file was modified: %q err=%v", got, err)
	}
	var results int
	if err := s.DB.QueryRow("SELECT count(*) FROM transcode_results WHERE video_id=? AND status='pending'", source.ID).Scan(&results); err != nil {
		t.Fatal(err)
	}
	if results != 1 {
		t.Fatalf("pending transcode results=%d want 1", results)
	}
}

func TestIntegrationTokenAdminFlow(t *testing.T) {
	mediaRoot := t.TempDir()
	a, s := newTransRunnerTestApp(t, mediaRoot)
	defer s.Close()
	uid, err := s.CreateFirstAdmin("admin", "x")
	if err != nil {
		t.Fatal(err)
	}
	cookie := "test-cookie-token"
	csrf := "test-csrf"
	if err := s.CreateSession(tokenHash(cookie), uid, csrf, time.Now().UTC().Add(time.Hour).Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(a.router(mediaRoot))
	defer server.Close()
	client := server.Client()
	adminReq := func(method string, body any) *http.Response {
		var payload []byte
		if body != nil {
			payload, _ = json.Marshal(body)
		}
		req, _ := http.NewRequest(method, server.URL+"/api/integrations/transrunner/token", bytes.NewReader(payload))
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

	res := adminReq(http.MethodGet, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var status map[string]any
	_ = json.NewDecoder(res.Body).Decode(&status)
	res.Body.Close()
	if status["configured"] != false {
		t.Fatalf("initially configured=%v", status["configured"])
	}

	res = adminReq(http.MethodPost, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d", res.StatusCode)
	}
	var created map[string]any
	_ = json.NewDecoder(res.Body).Decode(&created)
	res.Body.Close()
	token, _ := created["token"].(string)
	if token == "" {
		t.Fatal("no token returned")
	}
	// only hash stored
	var stored string
	if err := s.DB.QueryRow("SELECT token_hash FROM integration_tokens WHERE name='transrunner' AND revoked_at=''").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == token || stored != tokenHash(token) {
		t.Fatalf("token not hashed correctly: %q", stored)
	}
	// GET must never return plaintext
	res = adminReq(http.MethodGet, nil)
	raw, _ := readAll(res)
	if bytes.Contains(raw, []byte(token)) {
		t.Fatalf("status leaked plaintext token: %s", raw)
	}
	// second create without rotate -> 409
	requireStatus(t, adminReq(http.MethodPost, nil), http.StatusConflict)
	// rotate -> 201 new token
	res = adminReq(http.MethodPost, map[string]any{"rotate": true})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("rotate status=%d", res.StatusCode)
	}
	res.Body.Close()
	// old token now invalid over service API
	requireStatus(t, trRequest(t, client, http.MethodGet, server.URL+"/api/transrunner/v1/media/t:v", token, nil), http.StatusUnauthorized)
	// delete/revoke
	requireStatus(t, adminReq(http.MethodDelete, nil), http.StatusOK)
	res = adminReq(http.MethodGet, nil)
	raw, _ = readAll(res)
	if !bytes.Contains(raw, []byte("\"configured\":false")) {
		t.Fatalf("expected unconfigured after revoke: %s", raw)
	}
}

func readAll(res *http.Response) ([]byte, error) {
	defer res.Body.Close()
	var buf bytes.Buffer
	_, err := buf.ReadFrom(res.Body)
	return buf.Bytes(), err
}
