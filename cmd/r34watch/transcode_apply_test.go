package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"r34watch/internal/mediainfo"
	"r34watch/internal/store"
)

func TestIntegrationReadMediaListAndRedaction(t *testing.T) {
	mediaRoot := t.TempDir()
	a, s := newTransRunnerTestApp(t, mediaRoot)
	defer s.Close()
	seedTRVideo(t, s, mediaRoot, "t:v", 10, "sha-v", "A/v.mp4")
	readToken := createTRToken(t, s, []string{"read_media"})
	writeOnly := createTRToken(t, s, []string{"write_duplicate_results"})
	server := httptest.NewServer(a.router(mediaRoot))
	defer server.Close()
	client := server.Client()

	// read_media token can list media, with absolute paths redacted
	res := trRequest(t, client, http.MethodGet, server.URL+"/api/videos", readToken, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list status=%d", res.StatusCode)
	}
	var page struct {
		Items []map[string]any `json:"items"`
	}
	_ = json.NewDecoder(res.Body).Decode(&page)
	res.Body.Close()
	if len(page.Items) != 1 || page.Items[0]["video_id"] != "t:v" {
		t.Fatalf("items=%v", page.Items)
	}
	if page.Items[0]["file_path"] != "" || page.Items[0]["plex_path"] != "" {
		t.Fatalf("absolute paths leaked: %v", page.Items[0])
	}

	// no token / wrong scope cannot list
	requireStatus(t, trRequest(t, client, http.MethodGet, server.URL+"/api/videos", "", nil), http.StatusUnauthorized)
	requireStatus(t, trRequest(t, client, http.MethodGet, server.URL+"/api/videos", writeOnly, nil), http.StatusUnauthorized)
	// read_media token cannot reach admin mutation or settings
	requireStatus(t, trRequest(t, client, http.MethodPost, server.URL+"/api/videos/t:v/download", readToken, nil), http.StatusUnauthorized)
	requireStatus(t, trRequest(t, client, http.MethodGet, server.URL+"/api/settings", readToken, nil), http.StatusUnauthorized)
}

func makeVideo(t *testing.T, src, extraArgs []string, dst string) {
	t.Helper()
	args := []string{"-hide_banner", "-loglevel", "error", "-y"}
	args = append(args, src...)
	args = append(args, extraArgs...)
	args = append(args, dst)
	if out, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		t.Skipf("ffmpeg failed: %v %s", err, out)
	}
}

func transcodeApplyFixture(t *testing.T) (*App, *storeHandle, string, string) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not available")
	}
	mediaRoot := t.TempDir()
	a, s := newTransRunnerTestApp(t, mediaRoot)
	v := seedTRVideo(t, s, mediaRoot, "t:src", 1, "srchash", "A/src.mp4")
	srcAbs := filepath.Join(mediaRoot, "A/src.mp4")
	makeVideo(t, []string{"-f", "lavfi", "-i", "testsrc=size=64x64:rate=5:duration=1"}, []string{"-c:v", "libx264", "-pix_fmt", "yuv420p"}, srcAbs)
	staging := filepath.Join(mediaRoot, transRunnerStagingDir, "job_1")
	if err := os.MkdirAll(staging, 0755); err != nil {
		t.Fatal(err)
	}
	return a, &storeHandle{s, v.ID, mediaRoot}, srcAbs, staging
}

type storeHandle struct {
	s         *store.Store
	videoID   string
	mediaRoot string
}

func TestTranscodeApplyReplaceSource(t *testing.T) {
	a, h, srcAbs, staging := transcodeApplyFixture(t)
	defer h.s.Close()
	out := filepath.Join(staging, "output.mp4")
	makeVideo(t, []string{"-i", srcAbs}, []string{"-c:v", "libx265", "-tag:v", "hvc1", "-an"}, out)
	token := createTRToken(t, h.s, []string{"write_transcode_results"})
	server := httptest.NewServer(a.router(h.mediaRoot))
	defer server.Close()
	res := trRequest(t, server.Client(), http.MethodPost, server.URL+"/api/transrunner/v1/transcode-results", token, map[string]any{
		"video_id": h.videoID, "source_sha256": "srchash", "output_relative_path": ".transrunner/job_1/output.mp4", "profile": "hevc-preserve", "apply_mode": "replace_source",
	})
	if res.StatusCode != http.StatusOK {
		buf := make([]byte, 300)
		n, _ := res.Body.Read(buf)
		t.Fatalf("status=%d body=%s", res.StatusCode, buf[:n])
	}
	var out2 map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out2)
	res.Body.Close()
	if out2["status"] != "applied" {
		t.Fatalf("out=%v", out2)
	}
	// source replaced with HEVC (probe truth)
	probe, err := mediainfo.Probe(t.Context(), srcAbs)
	if err != nil || !isHEVCCodec(probe.VideoCodec) {
		t.Fatalf("source not HEVC: %+v err=%v", probe, err)
	}
	// DB metadata updated from HWatch ffprobe (not the callback)
	v, _ := h.s.Video(h.videoID)
	if !isHEVCCodec(v.VideoCodec) {
		t.Fatalf("db codec not updated: %q", v.VideoCodec)
	}
	if _, sha, _ := h.s.MediaFingerprint(h.videoID); sha == "" || sha == "srchash" {
		t.Fatalf("fingerprint not refreshed: %q", sha)
	}
	if _, err := os.Stat(srcAbs + ".hwatch-pre-transcode"); !os.IsNotExist(err) {
		t.Fatalf("backup left behind")
	}
}

func TestTranscodeApplyRejectsInvalidAndPreservesSource(t *testing.T) {
	a, h, srcAbs, staging := transcodeApplyFixture(t)
	defer h.s.Close()
	mediaRoot := h.mediaRoot
	server := httptest.NewServer(a.router(mediaRoot))
	defer server.Close()
	token := createTRToken(t, h.s, []string{"write_transcode_results"})
	before, _ := os.ReadFile(srcAbs)

	// wrong codec (h264 output) -> 422, source unchanged
	h264 := filepath.Join(staging, "h264.mp4")
	makeVideo(t, []string{"-i", srcAbs}, []string{"-c:v", "libx264"}, h264)
	requireStatus(t, trRequest(t, server.Client(), http.MethodPost, server.URL+"/api/transrunner/v1/transcode-results", token, map[string]any{
		"video_id": h.videoID, "source_sha256": "srchash", "output_relative_path": ".transrunner/job_1/h264.mp4", "apply_mode": "replace_source",
	}), http.StatusUnprocessableEntity)

	// resolution changed -> 422, source unchanged
	scaled := filepath.Join(staging, "scaled.mp4")
	makeVideo(t, []string{"-i", srcAbs}, []string{"-vf", "scale=128:128", "-c:v", "libx265", "-an"}, scaled)
	requireStatus(t, trRequest(t, server.Client(), http.MethodPost, server.URL+"/api/transrunner/v1/transcode-results", token, map[string]any{
		"video_id": h.videoID, "source_sha256": "srchash", "output_relative_path": ".transrunner/job_1/scaled.mp4", "apply_mode": "replace_source",
	}), http.StatusUnprocessableEntity)

	// source SHA mismatch -> 409
	good := filepath.Join(staging, "good.mp4")
	makeVideo(t, []string{"-i", srcAbs}, []string{"-c:v", "libx265", "-an"}, good)
	requireStatus(t, trRequest(t, server.Client(), http.MethodPost, server.URL+"/api/transrunner/v1/transcode-results", token, map[string]any{
		"video_id": h.videoID, "source_sha256": "different", "output_relative_path": ".transrunner/job_1/good.mp4", "apply_mode": "replace_source",
	}), http.StatusConflict)

	after, _ := os.ReadFile(srcAbs)
	if string(before) != string(after) {
		t.Fatal("source changed after rejected apply")
	}
}
