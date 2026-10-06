package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"r34watch/internal/store"
)

func TestSettingsMapHidesTransRunnerToken(t *testing.T) {
	cfg := store.Settings{TransRunnerURL: "http://runner:8787", TransRunnerAPIToken: "super-secret", TransRunnerDefaultProfile: "hevc-1080p", EnableTransRunner: true}
	got := settingsMap(cfg)
	if _, ok := got["transrunner_api_token"]; ok {
		t.Fatal("settings map must not expose the TransRunner API token")
	}
	if got["transrunner_token_configured"] != true {
		t.Fatalf("token_configured=%v", got["transrunner_token_configured"])
	}
	if body, _ := json.Marshal(got); bytes.Contains(body, []byte("super-secret")) {
		t.Fatalf("settings leaked token: %s", body)
	}
}

func TestTransRunnerSubmitTranscode(t *testing.T) {
	var captured struct {
		Input struct {
			Type              string `json:"type"`
			VideoID           string `json:"video_id"`
			MediaRelativePath string `json:"media_relative_path"`
		} `json:"input"`
		Profile string `json:"profile"`
	}
	var lastAuth string
	runner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/jobs" {
			lastAuth = r.Header.Get("Authorization")
			_ = json.NewDecoder(r.Body).Decode(&captured)
			_, _ = w.Write([]byte(`{"id":"job_42","state":"queued","profile":"hevc-1080p"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer runner.Close()

	mediaRoot := t.TempDir()
	a, s := newTransRunnerTestApp(t, mediaRoot)
	defer s.Close()
	seedTRVideo(t, s, mediaRoot, "t:src", 10, "sha-src", "A/src.mp4")
	cfg, _ := s.Settings()
	cfg.EnableTransRunner = true
	cfg.TransRunnerURL = runner.URL
	cfg.TransRunnerAPIToken = "runner-token"
	cfg.TransRunnerDefaultProfile = "hevc-1080p"
	if err := s.SaveSettings(cfg); err != nil {
		t.Fatal(err)
	}
	uid, _ := s.CreateFirstAdmin("admin", "x")
	cookie, csrf := "c", "s"
	if err := s.CreateSession(tokenHash(cookie), uid, csrf, time.Now().UTC().Add(time.Hour).Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(a.router(mediaRoot))
	defer server.Close()
	client := server.Client()

	submit := func(body map[string]any) *http.Response {
		payload, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/integrations/transrunner/submit/transcode", bytes.NewReader(payload))
		req.Header.Set("Cookie", "hwatch_session="+cookie)
		req.Header.Set("X-CSRF-Token", csrf)
		req.Header.Set("Content-Type", "application/json")
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	res := submit(map[string]any{"video_id": "t:src"})
	if res.StatusCode != http.StatusAccepted {
		buf := make([]byte, 256)
		n, _ := res.Body.Read(buf)
		t.Fatalf("status=%d body=%s", res.StatusCode, buf[:n])
	}
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	res.Body.Close()
	if out["runner_job_id"] != "job_42" {
		t.Fatalf("runner_job_id=%v", out["runner_job_id"])
	}
	if lastAuth != "Bearer runner-token" {
		t.Fatalf("auth=%q", lastAuth)
	}
	if captured.Input.Type != "hwatch" || captured.Input.VideoID != "t:src" || captured.Input.MediaRelativePath != "A/src.mp4" {
		t.Fatalf("payload=%+v", captured)
	}
	if captured.Input.MediaRelativePath == "" || captured.Input.MediaRelativePath[0] == '/' {
		t.Fatalf("absolute path leaked: %q", captured.Input.MediaRelativePath)
	}
	if captured.Profile != "hevc-1080p" {
		t.Fatalf("profile=%q", captured.Profile)
	}
	// submission persisted
	var runnerJobID string
	if err := s.DB.QueryRow("SELECT runner_job_id FROM transrunner_submissions WHERE video_id='t:src' AND operation='transcode'").Scan(&runnerJobID); err != nil {
		t.Fatal(err)
	}
	if runnerJobID != "job_42" {
		t.Fatalf("persisted=%q", runnerJobID)
	}
	// duplicate click protected unless forced
	requireStatus(t, submit(map[string]any{"video_id": "t:src"}), http.StatusConflict)
	requireStatus(t, submit(map[string]any{"video_id": "t:src", "force": true}), http.StatusAccepted)
}

func TestTransRunnerTestConnectionEndpoint(t *testing.T) {
	runner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			_, _ = w.Write([]byte(`{"status":"ok","version":"0.2.1"}`))
		case "/api/v1/capabilities":
			if r.Header.Get("Authorization") != "Bearer tok" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"videotoolbox":true,"profiles":["hevc-1080p"]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer runner.Close()
	mediaRoot := t.TempDir()
	a, s := newTransRunnerTestApp(t, mediaRoot)
	defer s.Close()
	cfg, _ := s.Settings()
	cfg.TransRunnerURL = runner.URL
	cfg.TransRunnerAPIToken = "tok"
	_ = s.SaveSettings(cfg)
	uid, _ := s.CreateFirstAdmin("admin", "x")
	cookie, csrf := "c2", "s2"
	if err := s.CreateSession(tokenHash(cookie), uid, csrf, time.Now().UTC().Add(time.Hour).Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(a.router(mediaRoot))
	defer server.Close()
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/integrations/transrunner/test", nil)
	req.Header.Set("Cookie", "hwatch_session="+cookie)
	req.Header.Set("X-CSRF-Token", csrf)
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	res.Body.Close()
	if out["ok"] != true || out["version"] != "0.2.1" {
		t.Fatalf("out=%v", out)
	}
}
