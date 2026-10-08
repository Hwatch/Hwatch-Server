package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"r34watch/internal/dedupe"
	"r34watch/internal/mediainfo"
	"r34watch/internal/store"
)

type applyFixture struct {
	app      *App
	store    *store.Store
	server   *httptest.Server
	videoID  string
	srcAbs   string
	staging  string
	writeTok string
	readTok  string
}

func newApplyFixture(t *testing.T) *applyFixture {
	t.Helper()
	a, h, srcAbs, staging := transcodeApplyFixture(t)
	server := httptest.NewServer(a.router(h.mediaRoot))
	t.Cleanup(func() { server.Close(); h.s.Close() })
	return &applyFixture{
		app: a, store: h.s, server: server,
		videoID: h.videoID, srcAbs: srcAbs, staging: staging,
		writeTok: createTRToken(t, h.s, []string{"write_transcode_results"}),
		readTok:  createTRToken(t, h.s, []string{"read_media"}),
	}
}

func (f *applyFixture) makeOutput(t *testing.T, name string) string {
	t.Helper()
	out := filepath.Join(f.staging, name)
	makeVideo(t, []string{"-i", f.srcAbs}, []string{"-c:v", "libx265", "-tag:v", "hvc1", "-an"}, out)
	return out
}

func (f *applyFixture) post(t *testing.T, token, key, job string, body map[string]any) *http.Response {
	t.Helper()
	if key != "" {
		body["idempotency_key"] = key
	}
	if job != "" {
		body["runner_job_id"] = job
	}
	return trRequest(t, f.server.Client(), http.MethodPost, f.server.URL+"/api/transrunner/v1/transcode-results", token, body)
}

func (f *applyFixture) get(t *testing.T, job string) map[string]any {
	t.Helper()
	res := trRequest(t, f.server.Client(), http.MethodGet, f.server.URL+"/api/transrunner/v1/transcode-results/"+job, f.readTok, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET status=%d", res.StatusCode)
	}
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	res.Body.Close()
	return out
}

func (f *applyFixture) applyBody(rel string) map[string]any {
	return map[string]any{"video_id": f.videoID, "source_sha256": "srchash", "output_relative_path": rel, "profile": "hevc-preserve", "apply_mode": "replace_source"}
}

func TestTranscodeApplyIdempotentReplay(t *testing.T) {
	f := newApplyFixture(t)
	f.makeOutput(t, "output.mp4")
	rel := transRunnerStagingDir + "/job_1/output.mp4"

	res := f.post(t, f.writeTok, "key-A", "job-A", f.applyBody(rel))
	if res.StatusCode != http.StatusOK {
		buf := make([]byte, 300)
		n, _ := res.Body.Read(buf)
		t.Fatalf("first apply status=%d body=%s", res.StatusCode, buf[:n])
	}
	var first map[string]any
	_ = json.NewDecoder(res.Body).Decode(&first)
	res.Body.Close()
	if first["status"] != "applied" || first["idempotent"] != false {
		t.Fatalf("first apply=%v", first)
	}
	srcInfo, _ := os.Stat(f.srcAbs)
	srcSHA1, _, _, _ := dedupe.FileHashes(f.srcAbs)

	// Replay the exact same request; the staging file is already consumed.
	res2 := f.post(t, f.writeTok, "key-A", "job-A", f.applyBody(rel))
	if res2.StatusCode != http.StatusOK {
		buf := make([]byte, 300)
		n, _ := res2.Body.Read(buf)
		t.Fatalf("replay status=%d body=%s", res2.StatusCode, buf[:n])
	}
	var second map[string]any
	_ = json.NewDecoder(res2.Body).Decode(&second)
	res2.Body.Close()
	if second["status"] != "applied" || second["idempotent"] != true {
		t.Fatalf("replay=%v", second)
	}
	srcInfo2, _ := os.Stat(f.srcAbs)
	srcSHA2, _, _, _ := dedupe.FileHashes(f.srcAbs)
	if !srcInfo.ModTime().Equal(srcInfo2.ModTime()) || srcSHA1 != srcSHA2 {
		t.Fatal("replay must not replace the file a second time")
	}

	// GET reflects the applied state with hashes.
	got := f.get(t, "job-A")
	if got["status"] != "applied" || got["applied"] != true || got["source_sha256"] != "srchash" || got["output_sha256"] == "" {
		t.Fatalf("GET=%v", got)
	}
}

func TestTranscodeApplyKeyConflict(t *testing.T) {
	f := newApplyFixture(t)
	f.makeOutput(t, "output.mp4")
	rel := transRunnerStagingDir + "/job_1/output.mp4"
	if res := f.post(t, f.writeTok, "key-A", "job-A", f.applyBody(rel)); res.StatusCode != http.StatusOK {
		res.Body.Close()
		t.Fatalf("first apply status=%d", res.StatusCode)
	}
	// Same idempotency key, different source sha => conflict.
	body := f.applyBody(rel)
	body["source_sha256"] = "different-sha"
	res := f.post(t, f.writeTok, "key-A", "job-A", body)
	requireStatus(t, res, http.StatusConflict)
}

func TestTranscodeApplyConcurrentSingleExecution(t *testing.T) {
	f := newApplyFixture(t)
	f.makeOutput(t, "output.mp4")
	rel := transRunnerStagingDir + "/job_1/output.mp4"
	body, _ := json.Marshal(map[string]any{"video_id": f.videoID, "source_sha256": "srchash", "output_relative_path": rel, "profile": "hevc-preserve", "apply_mode": "replace_source", "idempotency_key": "key-C", "runner_job_id": "job-C"})

	var wg sync.WaitGroup
	results := make([]map[string]any, 2)
	codes := make([]int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodPost, f.server.URL+"/api/transrunner/v1/transcode-results", bytes.NewReader(body))
			req.Header.Set("Authorization", "Bearer "+f.writeTok)
			req.Header.Set("Content-Type", "application/json")
			res, err := f.server.Client().Do(req)
			if err != nil {
				return
			}
			codes[i] = res.StatusCode
			_ = json.NewDecoder(res.Body).Decode(&results[i])
			res.Body.Close()
		}(i)
	}
	wg.Wait()
	nonIdempotentApplied := 0
	for _, r := range results {
		if r != nil && r["status"] == "applied" && r["idempotent"] == false {
			nonIdempotentApplied++
		}
	}
	if nonIdempotentApplied != 1 {
		t.Fatalf("exactly one executor expected, got %d (codes=%v results=%v)", nonIdempotentApplied, codes, results)
	}
	if got := f.get(t, "job-C"); got["status"] != "applied" {
		t.Fatalf("final state=%v", got)
	}
}

func TestTranscodeApplyCrashRecovery(t *testing.T) {
	f := newApplyFixture(t)
	out := f.makeOutput(t, "output.mp4")
	_, _, outSHA, err := dedupe.FileHashes(out)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate: file already replaced, but the ledger row is stuck at applying.
	if err := moveFile(out, f.srcAbs); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.store.BeginTranscodeApply(store.TranscodeApplyOp{OpKey: "job:crash1", RunnerJobID: "crash1", VideoID: f.videoID, SourceSHA256: "srchash", OutputSHA256: outSHA, OutputRelativePath: transRunnerStagingDir + "/job_1/output.mp4", Status: "applying"}); err != nil {
		t.Fatal(err)
	}
	got := f.get(t, "crash1")
	if got["status"] != "applied" || got["output_sha256"] != outSHA {
		t.Fatalf("crash recovery GET=%v", got)
	}
	// A repeat POST after recovery is an idempotent no-op, not a second replace.
	res := f.post(t, f.writeTok, "", "crash1", map[string]any{"video_id": f.videoID, "source_sha256": "srchash", "output_relative_path": transRunnerStagingDir + "/job_1/output.mp4", "profile": "hevc-preserve", "apply_mode": "replace_source"})
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		t.Fatalf("post after recovery status=%d", res.StatusCode)
	}
	var body map[string]any
	_ = json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()
	if body["idempotent"] != true {
		t.Fatalf("post after recovery=%v", body)
	}
}

func TestTranscodeApplyUncertainState(t *testing.T) {
	f := newApplyFixture(t)
	// Source is neither the recorded source nor the output hash.
	if err := os.WriteFile(f.srcAbs, []byte("corrupted-in-the-middle"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.store.BeginTranscodeApply(store.TranscodeApplyOp{OpKey: "job:crash2", RunnerJobID: "crash2", VideoID: f.videoID, SourceSHA256: "srchash", OutputSHA256: "deadbeef", OutputRelativePath: transRunnerStagingDir + "/job_1/output.mp4", Status: "applying"}); err != nil {
		t.Fatal(err)
	}
	if got := f.get(t, "crash2"); got["status"] != "uncertain" {
		t.Fatalf("expected uncertain, got %v", got)
	}
}

func TestTranscodeResultGetLegacyAndNotFound(t *testing.T) {
	f := newApplyFixture(t)
	// Legacy row written before the ledger existed.
	if _, err := f.store.AddTranscodeResult(store.TranscodeResult{VideoID: f.videoID, RunnerJobID: "legacy-1", SourceSHA256: "srchash", OutputRelativePath: "x", Status: "applied"}); err != nil {
		t.Fatal(err)
	}
	legacy := f.get(t, "legacy-1")
	if legacy["status"] != "applied" || legacy["legacy"] != true || legacy["output_sha256"] != "" {
		t.Fatalf("legacy=%v", legacy)
	}
	missing := f.get(t, "never-seen")
	if missing["status"] != "not_found" || missing["applied"] != false || missing["note"] == "" {
		t.Fatalf("not_found=%v", missing)
	}
}

func TestColorValidationUnchanged(t *testing.T) {
	src := mediainfo.Info{VideoCodec: "H.264", Width: 1920, Height: 1080, DurationSeconds: 10, FrameRate: 30, ColorTransfer: "smpte170m"}
	out := mediainfo.Info{VideoCodec: "H.265/HEVC", Width: 1920, Height: 1080, DurationSeconds: 10, FrameRate: 30, ColorTransfer: "bt709"}
	if err := validateTranscodeOutput(src, out); err == nil {
		t.Fatal("smpte170m -> bt709 must still be rejected")
	}
	out.ColorTransfer = "smpte170m"
	if err := validateTranscodeOutput(src, out); err != nil {
		t.Fatalf("smpte170m -> smpte170m must pass: %v", err)
	}
}

func TestServiceTokenCannotDeleteMedia(t *testing.T) {
	f := newApplyFixture(t)
	res := trRequest(t, f.server.Client(), http.MethodPost, f.server.URL+"/api/videos/"+f.videoID+"/delete", f.readTok, map[string]any{"mode": "delete_only"})
	requireStatus(t, res, http.StatusUnauthorized)
}
