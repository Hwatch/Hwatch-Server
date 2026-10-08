package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"r34watch/internal/dedupe"
	"r34watch/internal/mediainfo"
	"r34watch/internal/store"
)

// TransRunner service API.
//
// HWatch is the media-library authority: TransRunner reads media metadata and
// server-filtered duplicate candidates, then writes back duplicate and transcode
// results via a scoped bearer token. HWatch does not track runner jobs, runner
// state, progress, leases or capabilities.
const (
	transRunnerName       = "transrunner"
	trAlgorithmVersion    = "media-fingerprint-v1"
	transRunnerStagingDir = ".transrunner"
	trCandidateDefault    = 20
	trCandidateMax        = 100
	trToleranceMinSeconds = 2
	trTolerancePercent    = 1
)

var transRunnerScopes = map[string]bool{
	"read_media":              true,
	"read_dedupe_candidates":  true,
	"write_duplicate_results": true,
	"write_transcode_results": true,
}

var transRunnerDefaultScopes = []string{
	"read_media",
	"read_dedupe_candidates",
	"write_duplicate_results",
	"write_transcode_results",
}

var transRunnerClassifications = map[string]bool{
	"whole-video":    true,
	"trimmed-video":  true,
	"possible-match": true,
	"different":      true,
}

func integrationError(w http.ResponseWriter, status int, code, message string) {
	respond(w, status, map[string]string{"code": code, "error": message})
}

func parseScopes(raw string) []string {
	var scopes []string
	if json.Unmarshal([]byte(raw), &scopes) != nil {
		return nil
	}
	return scopes
}

func hasScope(scopes []string, want string) bool {
	for _, scope := range scopes {
		if scope == want {
			return true
		}
	}
	return false
}

// pathInside reports whether target is inside root (after both are cleaned and
// resolved by the caller).
func pathInside(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// mediaRelativePath resolves target under root and rejects any escape, including
// symlinks. Only the relative path is ever exposed to an integration.
func mediaRelativePath(root, target string) (string, error) {
	if strings.TrimSpace(target) == "" {
		return "", errors.New("媒体路径为空")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	realRoot, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", err
	}
	realTarget, err := filepath.EvalSymlinks(targetAbs)
	if err != nil {
		return "", err
	}
	if !pathInside(realRoot, realTarget) {
		return "", errors.New("媒体路径越界")
	}
	rel, err := filepath.Rel(realRoot, realTarget)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

func mediaPathOf(v store.Video) string {
	if v.PlexPath != "" {
		return v.PlexPath
	}
	return v.FilePath
}

func transRunnerTolerance(duration int64) int64 {
	if duration <= 0 {
		return trToleranceMinSeconds
	}
	tolerance := duration * trTolerancePercent / 100
	if tolerance < trToleranceMinSeconds {
		tolerance = trToleranceMinSeconds
	}
	return tolerance
}

// integrationTokenKey carries the authenticated Service Token through the
// request context so standard read handlers can tailor their response.
type integrationTokenKey struct{}

func integrationTokenFromContext(r *http.Request) (store.IntegrationToken, bool) {
	token, ok := r.Context().Value(integrationTokenKey{}).(store.IntegrationToken)
	return token, ok
}

// integrationReadablePaths are standard read-only endpoints a Service Token with
// read_media may call in addition to /api/transrunner/v1.
var integrationReadablePaths = map[string]bool{
	"/api/videos": true,
}

// integrationReadableRequest authenticates a read_media Service Token for a
// whitelisted read-only path. It is used by requireAuth as a second, scoped auth
// path so TransRunner can enumerate the library without an admin session.
func (a *App) integrationReadableRequest(r *http.Request) (store.IntegrationToken, bool) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return store.IntegrationToken{}, false
	}
	if !integrationReadablePaths[r.URL.Path] {
		return store.IntegrationToken{}, false
	}
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return store.IntegrationToken{}, false
	}
	raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if raw == "" {
		return store.IntegrationToken{}, false
	}
	token, err := a.store.IntegrationTokenByHash(tokenHash(raw))
	if err != nil {
		return store.IntegrationToken{}, false
	}
	if !hasScope(parseScopes(token.ScopesJSON), "read_media") {
		return store.IntegrationToken{}, false
	}
	a.touchIntegrationToken(token)
	return token, true
}

// requireIntegrationScope authenticates a scoped bearer token and enforces the
// required scope. Missing/invalid/revoked tokens are 401; a valid token without
// the scope is 403.
func (a *App) requireIntegrationScope(scope string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := a.integrationTokenFromRequest(w, r)
		if !ok {
			return
		}
		if !hasScope(parseScopes(token.ScopesJSON), scope) {
			integrationError(w, http.StatusForbidden, "insufficient_scope", "集成令牌缺少权限: "+scope)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), integrationTokenKey{}, token)))
	}
}

func (a *App) integrationTokenFromRequest(w http.ResponseWriter, r *http.Request) (store.IntegrationToken, bool) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		integrationError(w, http.StatusUnauthorized, "unauthorized", "缺少集成令牌")
		return store.IntegrationToken{}, false
	}
	raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if raw == "" {
		integrationError(w, http.StatusUnauthorized, "unauthorized", "缺少集成令牌")
		return store.IntegrationToken{}, false
	}
	token, err := a.store.IntegrationTokenByHash(tokenHash(raw))
	if err != nil {
		integrationError(w, http.StatusUnauthorized, "unauthorized", "无效或已撤销的集成令牌")
		return store.IntegrationToken{}, false
	}
	a.touchIntegrationToken(token)
	return token, true
}

// touchIntegrationToken updates last_used_at at most once per minute to avoid
// write amplification on every request.
func (a *App) touchIntegrationToken(token store.IntegrationToken) {
	if token.LastUsedAt != "" {
		if seen, err := time.Parse(time.RFC3339, token.LastUsedAt); err == nil && time.Since(seen) < time.Minute {
			return
		}
	}
	_ = a.store.TouchIntegrationToken(token.ID)
}

// --- Service API handlers -------------------------------------------------

func (a *App) trGetMedia(w http.ResponseWriter, r *http.Request) {
	v, err := a.store.Video(r.PathValue("video_id"))
	if err == sql.ErrNoRows {
		integrationError(w, http.StatusNotFound, "not_found", "视频不存在")
		return
	}
	if err != nil {
		integrationError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	cfg, _ := a.store.Settings()
	if !storagePathAvailable(cfg.DownloadDir, cfg.StorageMountRequired) {
		integrationError(w, http.StatusServiceUnavailable, "media_unavailable", "媒体存储当前不可用")
		return
	}
	rel, err := mediaRelativePath(cfg.DownloadDir, mediaPathOf(v))
	if err != nil {
		integrationError(w, http.StatusConflict, "media_unavailable", "媒体路径不在媒体根目录内")
		return
	}
	if info, statErr := os.Stat(filepath.Join(cfg.DownloadDir, filepath.FromSlash(rel))); statErr != nil || !info.Mode().IsRegular() {
		integrationError(w, http.StatusConflict, "media_file_missing", "媒体文件不可访问")
		return
	}
	size, sha, _ := a.store.MediaFingerprint(v.ID)
	respond(w, http.StatusOK, map[string]any{
		"video_id":            v.ID,
		"media_relative_path": rel,
		"file_size":           size,
		"duration":            v.DurationSeconds,
		"container":           v.Container,
		"video_codec":         v.VideoCodec,
		"audio_codec":         v.AudioCodec,
		"width":               v.Width,
		"height":              v.Height,
		"fps":                 v.FrameRate,
		"bit_depth":           v.BitDepth,
		"bitrate":             v.BitRate,
		"sha256":              sha,
	})
}

func (a *App) trDedupeCandidates(w http.ResponseWriter, r *http.Request) {
	v, err := a.store.Video(r.PathValue("video_id"))
	if err == sql.ErrNoRows {
		integrationError(w, http.StatusNotFound, "not_found", "视频不存在")
		return
	}
	if err != nil {
		integrationError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	cfg, _ := a.store.Settings()
	if !storagePathAvailable(cfg.DownloadDir, cfg.StorageMountRequired) {
		integrationError(w, http.StatusServiceUnavailable, "media_unavailable", "媒体存储当前不可用")
		return
	}
	sha, _ := a.store.MediaDuplicateHash(v.ID)
	limit := intParam(r, "limit", trCandidateDefault, 1, trCandidateMax)
	rows, err := a.store.MediaDuplicateCandidates(v.ID, sha, v.DurationSeconds, transRunnerTolerance(v.DurationSeconds), limit)
	if err != nil {
		integrationError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	candidates := []map[string]any{}
	for _, c := range rows {
		if len(candidates) >= limit {
			break
		}
		rel, err := mediaRelativePath(cfg.DownloadDir, mediaPathOf(c))
		if err != nil {
			continue
		}
		if info, statErr := os.Stat(filepath.Join(cfg.DownloadDir, filepath.FromSlash(rel))); statErr != nil || !info.Mode().IsRegular() {
			continue
		}
		size, csha, _ := a.store.MediaFingerprint(c.ID)
		candidates = append(candidates, map[string]any{
			"video_id":            c.ID,
			"media_relative_path": rel,
			"duration":            c.DurationSeconds,
			"sha256":              csha,
			"file_size":           size,
		})
	}
	respond(w, http.StatusOK, map[string]any{
		"video_id":          v.ID,
		"algorithm_version": trAlgorithmVersion,
		"candidates":        candidates,
	})
}

func (a *App) trDuplicateResults(w http.ResponseWriter, r *http.Request) {
	var input struct {
		VideoID          string `json:"video_id"`
		AlgorithmVersion string `json:"algorithm_version"`
		Matches          []struct {
			VideoID        string          `json:"video_id"`
			Classification string          `json:"classification"`
			Backend        string          `json:"backend"`
			Confidence     float64         `json:"confidence"`
			Metrics        json.RawMessage `json:"metrics"`
		} `json:"matches"`
	}
	if err := decode(r, &input); err != nil {
		integrationError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	source, err := a.store.Video(input.VideoID)
	if err == sql.ErrNoRows {
		integrationError(w, http.StatusNotFound, "not_found", "视频不存在")
		return
	}
	if err != nil {
		integrationError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	version := input.AlgorithmVersion
	if version == "" {
		version = trAlgorithmVersion
	}
	accepted := 0
	for _, match := range input.Matches {
		if !transRunnerClassifications[match.Classification] {
			integrationError(w, http.StatusUnprocessableEntity, "invalid_classification", "未知的重复分类: "+match.Classification)
			return
		}
		if match.VideoID == "" || match.VideoID == source.ID {
			continue
		}
		target, err := a.store.Video(match.VideoID)
		if err != nil {
			integrationError(w, http.StatusUnprocessableEntity, "unknown_match", "匹配视频不存在: "+match.VideoID)
			return
		}
		if match.Classification == "different" {
			continue
		}
		backend := match.Backend
		if backend == "" {
			backend = trAlgorithmVersion
		}
		metrics := "{}"
		if len(match.Metrics) > 0 && json.Valid(match.Metrics) {
			metrics = string(match.Metrics)
		}
		delta := source.DurationSeconds - target.DurationSeconds
		if delta < 0 {
			delta = -delta
		}
		created, err := a.store.AddMediaDuplicate(source.ID, target.ID, match.Classification, backend, version, metrics, match.Confidence, delta)
		if err != nil {
			integrationError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		if created {
			accepted++
		}
	}
	_ = a.store.AddAudit("integration:"+transRunnerName, "duplicate_result_received", "video", source.ID, "success", map[string]any{"matches": len(input.Matches), "created": accepted})
	respond(w, http.StatusOK, map[string]any{"status": "accepted", "candidates_created": accepted})
}

func (a *App) trTranscodeResults(w http.ResponseWriter, r *http.Request) {
	var input struct {
		VideoID            string `json:"video_id"`
		RunnerJobID        string `json:"runner_job_id"`
		SourceSHA256       string `json:"source_sha256"`
		OutputRelativePath string `json:"output_relative_path"`
		Profile            string `json:"profile"`
		ApplyMode          string `json:"apply_mode"`
		IdempotencyKey     string `json:"idempotency_key"`
	}
	if err := decode(r, &input); err != nil {
		integrationError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	idemKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idemKey == "" {
		idemKey = strings.TrimSpace(input.IdempotencyKey)
	}
	applyMode := strings.TrimSpace(input.ApplyMode)
	if applyMode != "" && applyMode != transcodeApplyReplace {
		integrationError(w, http.StatusUnprocessableEntity, "invalid_apply_mode", "未知的 apply_mode")
		return
	}
	v, err := a.store.Video(input.VideoID)
	if err == sql.ErrNoRows {
		integrationError(w, http.StatusNotFound, "not_found", "视频不存在")
		return
	}
	if err != nil {
		integrationError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	cfg, _ := a.store.Settings()
	if !storagePathAvailable(cfg.DownloadDir, cfg.StorageMountRequired) {
		integrationError(w, http.StatusServiceUnavailable, "media_unavailable", "媒体存储当前不可用")
		return
	}
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(input.OutputRelativePath)))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		integrationError(w, http.StatusUnprocessableEntity, "invalid_path", "输出路径无效")
		return
	}
	if clean != transRunnerStagingDir && !strings.HasPrefix(clean, transRunnerStagingDir+string(filepath.Separator)) {
		integrationError(w, http.StatusUnprocessableEntity, "invalid_path", "输出必须位于暂存目录 "+transRunnerStagingDir+"/ 内")
		return
	}
	// Idempotent replay: if this job/key was already applied, answer from the
	// durable ledger before touching the (possibly already consumed) staging
	// file or the source.
	if applyMode == transcodeApplyReplace && (idemKey != "" || strings.TrimSpace(input.RunnerJobID) != "") {
		var existing store.TranscodeApplyOp
		found := false
		if idemKey != "" {
			if op, e := a.store.TranscodeApplyByKey("idem:" + idemKey); e == nil {
				existing, found = op, true
			}
		}
		if !found && strings.TrimSpace(input.RunnerJobID) != "" {
			if op, e := a.store.LatestTranscodeApplyByJob(input.RunnerJobID); e == nil {
				existing, found = op, true
			}
		}
		if found {
			conflict := (existing.SourceSHA256 != "" && input.SourceSHA256 != "" && existing.SourceSHA256 != input.SourceSHA256) ||
				(existing.OutputRelativePath != "" && existing.OutputRelativePath != clean) ||
				(existing.Profile != "" && input.Profile != "" && existing.Profile != input.Profile)
			if conflict {
				integrationError(w, http.StatusConflict, "idempotency_conflict", "相同操作标识但请求内容不同，拒绝执行")
				return
			}
			switch existing.Status {
			case "applied":
				respond(w, http.StatusOK, map[string]any{"status": "applied", "id": existing.ResultID, "idempotent": true})
				return
			case "pending", "applying":
				reconciled, _ := a.reconcileTranscodeApply(existing)
				if reconciled.Status == "applied" {
					respond(w, http.StatusOK, map[string]any{"status": "applied", "id": reconciled.ResultID, "idempotent": true, "reconciled": true})
					return
				}
				integrationError(w, http.StatusConflict, "apply_in_progress", "该任务已有正在进行的应用操作，请稍后查询")
				return
			case "uncertain", "needs_recovery":
				integrationError(w, http.StatusConflict, "apply_state_uncertain", "该任务先前的应用结果无法确认，需人工核查")
				return
			}
			// failed: fall through to retry via the durable claim below.
		}
	}
	realRoot, err := filepath.EvalSymlinks(cfg.DownloadDir)
	if err != nil {
		integrationError(w, http.StatusServiceUnavailable, "media_unavailable", "媒体根目录不可访问")
		return
	}
	realOutput, err := filepath.EvalSymlinks(filepath.Join(cfg.DownloadDir, clean))
	if err != nil {
		integrationError(w, http.StatusUnprocessableEntity, "output_missing", "暂存输出不存在或不可访问")
		return
	}
	if !pathInside(realRoot, realOutput) || !pathInside(filepath.Join(realRoot, transRunnerStagingDir), realOutput) {
		integrationError(w, http.StatusUnprocessableEntity, "invalid_path", "暂存输出越界")
		return
	}
	if info, statErr := os.Stat(realOutput); statErr != nil || !info.Mode().IsRegular() {
		integrationError(w, http.StatusUnprocessableEntity, "output_missing", "暂存输出不是普通文件")
		return
	}
	if input.SourceSHA256 != "" {
		if _, currentSHA, _ := a.store.MediaFingerprint(v.ID); currentSHA != "" && currentSHA != input.SourceSHA256 {
			integrationError(w, http.StatusConflict, "source_changed", "源媒体已变化，拒绝转码结果")
			return
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	probe, probeErr := mediainfo.Probe(ctx, realOutput)
	cancel()
	if probeErr != nil {
		integrationError(w, http.StatusUnprocessableEntity, "probe_failed", "转码产物无法通过 ffprobe 校验")
		return
	}
	probeJSON, _ := json.Marshal(map[string]any{
		"video_codec":      probe.VideoCodec,
		"audio_codec":      probe.AudioCodec,
		"container":        probe.Container,
		"width":            probe.Width,
		"height":           probe.Height,
		"frame_rate":       probe.FrameRate,
		"bit_rate":         probe.BitRate,
		"bit_depth":        probe.BitDepth,
		"duration_seconds": probe.DurationSeconds,
	})
	if applyMode == transcodeApplyReplace {
		_, _, outputSHA, hashErr := dedupe.FileHashes(realOutput)
		if hashErr != nil || outputSHA == "" {
			integrationError(w, http.StatusUnprocessableEntity, "output_unreadable", "无法计算转码产物哈希")
			return
		}
		opKey := transcodeOpKey(input.RunnerJobID, idemKey, v.ID, input.SourceSHA256, outputSHA)
		contentHash := transcodeContentHash(v.ID, input.SourceSHA256, outputSHA, clean, input.Profile)
		op := store.TranscodeApplyOp{OpKey: opKey, RunnerJobID: input.RunnerJobID, VideoID: v.ID, SourceSHA256: input.SourceSHA256, OutputSHA256: outputSHA, OutputRelativePath: clean, Profile: input.Profile, ContentHash: contentHash, Status: "applying"}
		existing, claimed, beginErr := a.store.BeginTranscodeApply(op)
		if beginErr != nil {
			integrationError(w, http.StatusInternalServerError, "internal_error", beginErr.Error())
			return
		}
		if !claimed {
			if existing.ContentHash != "" && existing.ContentHash != contentHash {
				integrationError(w, http.StatusConflict, "idempotency_conflict", "相同操作标识但请求内容不同，拒绝执行")
				return
			}
			switch existing.Status {
			case "applied":
				respond(w, http.StatusOK, map[string]any{"status": "applied", "id": existing.ResultID, "idempotent": true})
				return
			case "failed":
				if err := a.store.ReclaimTranscodeApply(opKey); err != nil {
					integrationError(w, http.StatusInternalServerError, "internal_error", err.Error())
					return
				}
			case "pending", "applying":
				reconciled, _ := a.reconcileTranscodeApply(existing)
				if reconciled.Status == "applied" {
					respond(w, http.StatusOK, map[string]any{"status": "applied", "id": reconciled.ResultID, "idempotent": true, "reconciled": true})
					return
				}
				integrationError(w, http.StatusConflict, "apply_in_progress", "该任务已有正在进行的应用操作，请稍后查询")
				return
			default:
				integrationError(w, http.StatusConflict, "apply_state_uncertain", "该任务先前的应用结果无法确认，需人工核查")
				return
			}
		}
		if applyErr := a.applyTranscodeResult(v, realOutput, input.SourceSHA256); applyErr != nil {
			_, _ = a.store.AddTranscodeResult(store.TranscodeResult{VideoID: v.ID, RunnerJobID: input.RunnerJobID, SourceSHA256: input.SourceSHA256, OutputRelativePath: clean, Profile: input.Profile, ProbeJSON: string(probeJSON), Status: "failed"})
			_ = a.store.UpdateTranscodeApply(opKey, "failed", 0, applyErr.Error(), "")
			_ = a.store.AddAudit("integration:"+transRunnerName, "transcode_result_received", "video", v.ID, "failed", map[string]any{"profile": input.Profile, "apply_mode": applyMode})
			if errors.Is(applyErr, errSourceChanged) {
				integrationError(w, http.StatusConflict, "source_changed", "源媒体已变化，拒绝替换")
				return
			}
			integrationError(w, http.StatusUnprocessableEntity, "apply_failed", applyErr.Error())
			return
		}
		id, addErr := a.store.AddTranscodeResult(store.TranscodeResult{VideoID: v.ID, RunnerJobID: input.RunnerJobID, SourceSHA256: input.SourceSHA256, OutputRelativePath: clean, Profile: input.Profile, ProbeJSON: string(probeJSON), Status: "applied"})
		if addErr != nil {
			integrationError(w, http.StatusInternalServerError, "internal_error", addErr.Error())
			return
		}
		_ = a.store.UpdateTranscodeApply(opKey, "applied", id, "", "")
		_ = a.store.AddAudit("integration:"+transRunnerName, "transcode_result_received", "video", v.ID, "success", map[string]any{"result_id": id, "profile": input.Profile, "apply_mode": applyMode})
		respond(w, http.StatusOK, map[string]any{"status": "applied", "id": id, "idempotent": false})
		return
	}
	id, err := a.store.AddTranscodeResult(store.TranscodeResult{
		VideoID:            v.ID,
		RunnerJobID:        input.RunnerJobID,
		SourceSHA256:       input.SourceSHA256,
		OutputRelativePath: clean,
		Profile:            input.Profile,
		ProbeJSON:          string(probeJSON),
		Status:             "pending",
	})
	if err != nil {
		integrationError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	_ = a.store.AddAudit("integration:"+transRunnerName, "transcode_result_received", "video", v.ID, "success", map[string]any{"result_id": id, "profile": input.Profile})
	// Without apply_mode the source is never replaced; only a pending result is recorded.
	respond(w, http.StatusAccepted, map[string]any{"status": "validated", "id": id})
}

// transcodeOpKey derives the durable idempotency key. An explicit
// Idempotency-Key wins, otherwise the runner_job_id, otherwise a content hash.
func transcodeOpKey(jobID, idemKey, videoID, sourceSHA, outputSHA string) string {
	if k := strings.TrimSpace(idemKey); k != "" {
		return "idem:" + k
	}
	if j := strings.TrimSpace(jobID); j != "" {
		return "job:" + j
	}
	sum := sha256.Sum256([]byte(videoID + "|" + sourceSHA + "|" + outputSHA))
	return "opf:" + hex.EncodeToString(sum[:])
}

func transcodeContentHash(videoID, sourceSHA, outputSHA, relPath, profile string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{videoID, sourceSHA, outputSHA, relPath, profile}, "|")))
	return hex.EncodeToString(sum[:])
}

func mapApplyStatus(status string) (mapped string, applied bool) {
	switch status {
	case "applied":
		return "applied", true
	case "applying", "pending":
		return "in_progress", false
	case "failed":
		return "not_applied", false
	case "uncertain", "needs_recovery":
		return "uncertain", false
	default:
		return "not_applied", false
	}
}

// finalizeAppliedSource refreshes DB metadata from the (already replaced) source
// file; used to complete an apply that was interrupted after the file move.
func (a *App) finalizeAppliedSource(v store.Video, source string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	info, err := mediainfo.Probe(ctx, source)
	if err != nil {
		return err
	}
	if err := a.store.UpdateMediaInfo(v.ID, info.VideoCodec, info.AudioCodec, info.Container, info.Width, info.Height, info.FrameRate, info.BitRate, info.BitDepth); err != nil {
		return err
	}
	_ = a.store.SetVideoDuration(v.ID, int64(math.Round(info.DurationSeconds)))
	if size, quick, sha, hashErr := dedupe.FileHashes(source); hashErr == nil {
		_ = a.store.SaveFileHashes(v.ID, size, quick, sha)
	}
	return nil
}

// reconcileTranscodeApply inspects the filesystem to resolve an interrupted
// apply. It must never blindly re-replace: it compares the current source hash
// against the recorded source and output hashes.
func (a *App) reconcileTranscodeApply(op store.TranscodeApplyOp) (store.TranscodeApplyOp, error) {
	v, err := a.store.Video(op.VideoID)
	if err != nil {
		_ = a.store.UpdateTranscodeApply(op.OpKey, "needs_recovery", 0, "视频不存在，无法核对", "")
		return a.store.TranscodeApplyByKey(op.OpKey)
	}
	source := mediaPathOf(v)
	sourceSHA := ""
	if source != "" {
		if _, _, sha, hashErr := dedupe.FileHashes(source); hashErr == nil {
			sourceSHA = sha
		}
	}
	switch {
	case op.OutputSHA256 != "" && sourceSHA == op.OutputSHA256:
		if err := a.finalizeAppliedSource(v, source); err != nil {
			_ = a.store.UpdateTranscodeApply(op.OpKey, "needs_recovery", 0, "已替换但元数据刷新失败: "+err.Error(), source+".hwatch-pre-transcode")
			return a.store.TranscodeApplyByKey(op.OpKey)
		}
		id := op.ResultID
		if id == 0 {
			id, _ = a.store.AddTranscodeResult(store.TranscodeResult{VideoID: v.ID, RunnerJobID: op.RunnerJobID, SourceSHA256: op.SourceSHA256, OutputRelativePath: op.OutputRelativePath, Profile: op.Profile, Status: "applied"})
		}
		_ = os.Remove(source + ".hwatch-pre-transcode")
		_ = a.store.UpdateTranscodeApply(op.OpKey, "applied", id, "崩溃恢复：文件已替换", "")
	case op.SourceSHA256 != "" && sourceSHA == op.SourceSHA256:
		_ = a.store.UpdateTranscodeApply(op.OpKey, "failed", 0, "崩溃恢复：文件未替换，可重试", "")
	default:
		_ = a.store.UpdateTranscodeApply(op.OpKey, "uncertain", 0, "崩溃恢复：无法确认文件状态，需人工核查", "")
	}
	return a.store.TranscodeApplyByKey(op.OpKey)
}

// trTranscodeResultGet is the read-only idempotency lookup. It never triggers
// an apply and reports not_found rather than implying a task is safe to redo.
func (a *App) trTranscodeResultGet(w http.ResponseWriter, r *http.Request) {
	job := strings.TrimSpace(r.PathValue("runner_job_id"))
	if job == "" {
		integrationError(w, http.StatusBadRequest, "invalid_request", "缺少 runner_job_id")
		return
	}
	if op, err := a.store.LatestTranscodeApplyByJob(job); err == nil {
		status, applied := mapApplyStatus(op.Status)
		if op.Status == "applying" || op.Status == "pending" {
			op, _ = a.reconcileTranscodeApply(op)
			status, applied = mapApplyStatus(op.Status)
		}
		appliedAt := ""
		if applied {
			appliedAt = op.UpdatedAt
		}
		respond(w, http.StatusOK, map[string]any{
			"runner_job_id": job, "status": status, "applied": applied,
			"video_id": op.VideoID, "source_sha256": op.SourceSHA256, "output_sha256": op.OutputSHA256,
			"applied_at": appliedAt, "legacy": false,
		})
		return
	}
	if legacy, err := a.store.LatestTranscodeResultByJob(job); err == nil {
		applied := legacy.Status == "applied"
		appliedAt := ""
		status := "uncertain"
		switch legacy.Status {
		case "applied":
			status, appliedAt = "applied", legacy.CreatedAt
		case "pending":
			status = "not_applied"
		case "failed":
			status = "not_applied"
		}
		respond(w, http.StatusOK, map[string]any{
			"runner_job_id": job, "status": status, "applied": applied,
			"video_id": legacy.VideoID, "source_sha256": legacy.SourceSHA256, "output_sha256": "",
			"applied_at": appliedAt, "legacy": true,
			"note": "历史记录缺少输出哈希与幂等键，结果仅供只读参考，不得据此自动重试",
		})
		return
	}
	respond(w, http.StatusOK, map[string]any{
		"runner_job_id": job, "status": "not_found", "applied": false,
		"note": "没有该 job 的记录；not_found 不代表可以安全重复提交，尤其旧版任务需人工核查",
	})
}

// --- Admin token management ----------------------------------------------
func (a *App) integrationTokenGet(w http.ResponseWriter, r *http.Request) {
	token, err := a.store.ActiveIntegrationToken(transRunnerName)
	if err == sql.ErrNoRows {
		respond(w, http.StatusOK, map[string]any{"configured": false, "scopes": transRunnerDefaultScopes})
		return
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	respond(w, http.StatusOK, map[string]any{
		"configured":   true,
		"created_at":   token.CreatedAt,
		"last_used_at": token.LastUsedAt,
		"scopes":       parseScopes(token.ScopesJSON),
	})
}

func (a *App) integrationTokenCreate(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Rotate bool `json:"rotate"`
	}
	_ = decode(r, &input)
	if _, err := a.store.ActiveIntegrationToken(transRunnerName); err == nil && !input.Rotate {
		fail(w, http.StatusConflict, errors.New("已存在有效的集成令牌，请显式 rotate=true 轮换"))
		return
	}
	token, err := randomToken()
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	scopesJSON, _ := json.Marshal(transRunnerDefaultScopes)
	_, _ = a.store.RevokeIntegrationTokens(transRunnerName)
	if _, err = a.store.CreateIntegrationToken(transRunnerName, tokenHash(token), string(scopesJSON)); err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	_ = a.store.AddAudit("admin", "integration_token_created", "integration", transRunnerName, "success", map[string]any{"scopes": transRunnerDefaultScopes})
	respond(w, http.StatusCreated, map[string]any{"token": token, "scopes": transRunnerDefaultScopes})
}

func (a *App) integrationTokenDelete(w http.ResponseWriter, r *http.Request) {
	revoked, err := a.store.RevokeIntegrationTokens(transRunnerName)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	_ = a.store.AddAudit("admin", "integration_token_revoked", "integration", transRunnerName, "success", map[string]any{"revoked": revoked})
	respond(w, http.StatusOK, map[string]any{"revoked": true})
}
