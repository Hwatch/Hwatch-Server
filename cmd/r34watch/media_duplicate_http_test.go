package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type nodeClaimResponse struct {
	Job struct {
		ID                string `json:"id"`
		Type              string `json:"type"`
		Subtype           string `json:"subtype"`
		VideoID           string `json:"video_id"`
		AlgorithmVersion  string `json:"algorithm_version"`
		MediaRelativePath string `json:"media_relative_path"`
		InputJSON         string `json:"input_json"`
	} `json:"job"`
	LeaseSeconds int `json:"lease_seconds"`
}

func nodeRequest(t *testing.T, client *http.Client, method, url, token string, body any) *http.Response {
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

func decodeNodeClaim(t *testing.T, res *http.Response) nodeClaimResponse {
	t.Helper()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("claim status=%d", res.StatusCode)
	}
	var parsed nodeClaimResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestMediaDuplicateNodeFlowOverHTTP(t *testing.T) {
	mediaRoot := t.TempDir()
	a, s := newMediaDuplicateTestApp(t, mediaRoot)
	defer s.Close()
	target := seedMediaVideo(t, s, mediaRoot, "test:target", 100, "sha-target", "target.mp4")
	seedMediaVideo(t, s, mediaRoot, "test:near", 101, "sha-near", "near.mp4")
	cfg, _ := s.Settings()

	token := "media-duplicate-node-token"
	if err := s.CreateComputeNode("node_md", "Mac", tokenHash(token), `{"dedupe":true,"media_duplicate":true}`); err != nil {
		t.Fatal(err)
	}
	job, created, err := a.createMediaDuplicateJob(target, cfg, false)
	if err != nil || !created {
		t.Fatalf("create job created=%v err=%v", created, err)
	}

	server := httptest.NewServer(a.router(mediaRoot))
	defer server.Close()
	client := server.Client()

	res := nodeRequest(t, client, http.MethodPost, server.URL+"/api/node/v1/heartbeat", token, map[string]any{
		"capabilities": map[string]any{"dedupe": true, "media_duplicate": true},
		"version":      "test",
		"platform":     "darwin",
	})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("heartbeat status=%d", res.StatusCode)
	}

	claim := decodeNodeClaim(t, nodeRequest(t, client, http.MethodPost, server.URL+"/api/node/v1/jobs/claim", token, map[string]any{"types": []string{"dedupe"}}))
	if claim.Job.ID != job.ID || claim.Job.Subtype != mediaDuplicateSubtype {
		t.Fatalf("claimed job=%+v", claim.Job)
	}
	var options mediaDuplicateOptions
	if err := json.Unmarshal([]byte(claim.Job.InputJSON), &options); err != nil {
		t.Fatal(err)
	}
	if options.Mode != mediaDuplicateMode || len(options.Candidates) != 1 || options.Candidates[0].VideoID != "test:near" || options.Candidates[0].MediaRelativePath != "near.mp4" {
		t.Fatalf("claim payload candidates=%+v", options)
	}

	res = nodeRequest(t, client, http.MethodPost, server.URL+"/api/node/v1/jobs/"+claim.Job.ID+"/progress", token, map[string]any{"status": "transcoding", "progress": 0.5})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("progress status=%d (lease renew)", res.StatusCode)
	}

	res = nodeRequest(t, client, http.MethodPost, server.URL+"/api/node/v1/jobs/"+claim.Job.ID+"/complete", token, map[string]any{
		"result": map[string]any{"matches": []map[string]any{{
			"video_id":          "test:near",
			"classification":    "whole-video",
			"backend":           "media-fingerprint-v1",
			"algorithm_version": "media-fingerprint-v1",
			"confidence":        0.98,
			"duration_delta":    1,
			"metrics":           map[string]any{"sample_count": 32, "matched_samples": 32},
		}}},
	})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("complete status=%d", res.StatusCode)
	}

	var count int
	if err := s.DB.QueryRow("SELECT count(*) FROM duplicate_candidates WHERE match_type='media_duplicate'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("duplicate candidate count=%d want 1", count)
	}
	var classification, version string
	if err := s.DB.QueryRow("SELECT classification,algorithm_version FROM duplicate_candidates WHERE match_type='media_duplicate'").Scan(&classification, &version); err != nil {
		t.Fatal(err)
	}
	if classification != "whole-video" || version != "media-fingerprint-v1" {
		t.Fatalf("classification=%q version=%q", classification, version)
	}

	// A "different" result must not create a candidate.
	other := seedMediaVideo(t, s, mediaRoot, "test:other", 100, "sha-other", "other.mp4")
	job2, created2, err := a.createMediaDuplicateJob(other, cfg, false)
	if err != nil || !created2 {
		t.Fatalf("create job2 created=%v err=%v", created2, err)
	}
	claim2 := decodeNodeClaim(t, nodeRequest(t, client, http.MethodPost, server.URL+"/api/node/v1/jobs/claim", token, map[string]any{"types": []string{"dedupe"}}))
	if claim2.Job.ID != job2.ID {
		t.Fatalf("claim2 job=%s want %s", claim2.Job.ID, job2.ID)
	}
	candidateID := func() string {
		var options mediaDuplicateOptions
		if err := json.Unmarshal([]byte(claim2.Job.InputJSON), &options); err != nil || len(options.Candidates) == 0 {
			t.Fatalf("claim2 candidates missing: %v", err)
		}
		return options.Candidates[0].VideoID
	}()
	res = nodeRequest(t, client, http.MethodPost, server.URL+"/api/node/v1/jobs/"+claim2.Job.ID+"/complete", token, map[string]any{
		"result": map[string]any{"matches": []map[string]any{{"video_id": candidateID, "classification": "different", "confidence": 0.1}}},
	})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("complete different status=%d", res.StatusCode)
	}
	if err := s.DB.QueryRow("SELECT count(*) FROM duplicate_candidates WHERE match_type='media_duplicate'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("different must not create a candidate, count=%d", count)
	}
}
