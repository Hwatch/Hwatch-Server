package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"r34watch/internal/mediainfo"
	"r34watch/internal/store"
)

const computeLease = 2 * time.Minute

type nodeCapabilities struct {
	Dedupe           bool     `json:"dedupe"`
	Transcode        bool     `json:"transcode"`
	MediaDuplicate   bool     `json:"media_duplicate,omitempty"`
	VideoSignature   bool     `json:"video_signature,omitempty"`
	VideoToolbox     bool     `json:"video_toolbox,omitempty"`
	Encoders         []string `json:"encoders,omitempty"`
	DedupeWorkers    int      `json:"max_dedupe_jobs,omitempty"`
	TranscodeWorkers int      `json:"max_transcode_jobs,omitempty"`
}

func newComputeID(prefix string) (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	if len(token) > 24 {
		token = token[:24]
	}
	return prefix + token, nil
}

func (a *App) computeNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		nodes, err := a.store.ComputeNodes()
		if err != nil {
			fail(w, 500, err)
			return
		}
		now := time.Now()
		items := make([]map[string]any, 0, len(nodes))
		for _, n := range nodes {
			status := n.Status
			if n.RevokedAt == "" {
				if seen, e := time.Parse(time.RFC3339, n.LastSeenAt); e != nil || now.Sub(seen) > 3*time.Minute {
					status = "offline"
				}
			}
			items = append(items, map[string]any{"id": n.ID, "name": n.Name, "permissions": jsonValue(n.PermissionsJSON), "capabilities": jsonValue(n.CapabilitiesJSON), "version": n.Version, "platform": n.Platform, "status": status, "last_seen_at": n.LastSeenAt, "created_at": n.CreatedAt, "revoked_at": n.RevokedAt})
		}
		respond(w, 200, map[string]any{"items": items})
		return
	}
	var input struct {
		Name         string            `json:"name"`
		Permissions  *nodeCapabilities `json:"permissions"`
		Capabilities *nodeCapabilities `json:"capabilities,omitempty"`
	}
	if err := decode(r, &input); err != nil {
		fail(w, 400, err)
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 100 {
		fail(w, 422, errors.New("节点名称长度必须为 1 到 100 个字符"))
		return
	}
	permissions := input.Permissions
	if permissions == nil {
		permissions = input.Capabilities
	}
	if permissions == nil || !permissions.Dedupe && !permissions.Transcode {
		fail(w, 422, errors.New("节点至少需要一种能力"))
		return
	}
	id, err := newComputeID("node_")
	if err != nil {
		fail(w, 500, err)
		return
	}
	token, err := randomToken()
	if err != nil {
		fail(w, 500, err)
		return
	}
	raw, _ := json.Marshal(permissions)
	if err = a.store.CreateComputeNode(id, input.Name, tokenHash(token), string(raw)); err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 201, map[string]any{"id": id, "name": input.Name, "token": token, "permissions": permissions, "warning": "令牌只显示一次，请立即保存"})
}

func (a *App) revokeComputeNode(w http.ResponseWriter, r *http.Request) {
	if err := a.store.RevokeComputeNode(r.PathValue("id")); err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]any{"revoked": true})
}

func (a *App) computeJobs(w http.ResponseWriter, r *http.Request) {
	limit := intParam(r, "limit", 100, 1, 500)
	offset := intParam(r, "offset", 0, 0, 1000000)
	items, err := a.store.ComputeJobs(limit, offset)
	if err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]any{"items": items, "limit": limit, "offset": offset})
}

func secureMediaRelative(root, path string) (string, os.FileInfo, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", nil, err
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return "", nil, err
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", nil, errors.New("媒体文件不在配置的媒体根目录内")
	}
	rootReal, rootErr := filepath.EvalSymlinks(rootAbs)
	pathReal, pathErr := filepath.EvalSymlinks(pathAbs)
	if rootErr != nil {
		return "", nil, fmt.Errorf("无法解析媒体根目录: %w", rootErr)
	}
	if pathErr != nil {
		return "", nil, fmt.Errorf("无法解析媒体路径: %w", pathErr)
	}
	realRel, realErr := filepath.Rel(rootReal, pathReal)
	if realErr != nil || realRel == ".." || strings.HasPrefix(realRel, ".."+string(filepath.Separator)) || filepath.IsAbs(realRel) {
		return "", nil, errors.New("媒体软链接指向媒体根目录之外")
	}
	info, err := os.Stat(pathAbs)
	if err != nil {
		return "", nil, err
	}
	if !info.Mode().IsRegular() {
		return "", nil, errors.New("媒体路径不是普通文件")
	}
	return filepath.ToSlash(rel), info, nil
}

func computeIdempotency(videoID string, info os.FileInfo, kind, version string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d\x00%d\x00%s\x00%s", videoID, info.Size(), info.ModTime().UnixNano(), kind, version)))
	return hex.EncodeToString(sum[:])
}

func (a *App) createComputeJob(w http.ResponseWriter, r *http.Request) {
	var input struct {
		VideoID          string          `json:"video_id"`
		Type             string          `json:"type"`
		AlgorithmVersion string          `json:"algorithm_version"`
		Options          json.RawMessage `json:"options"`
	}
	if err := decode(r, &input); err != nil {
		fail(w, 400, err)
		return
	}
	if input.Type != "dedupe" && input.Type != "transcode" {
		fail(w, 422, errors.New("任务类型必须是 dedupe 或 transcode"))
		return
	}
	if input.AlgorithmVersion == "" {
		if input.Type == "dedupe" {
			input.AlgorithmVersion = mediaDuplicateAlgorithm
		} else {
			input.AlgorithmVersion = "hevc-v1"
		}
	}
	v, err := a.store.Video(input.VideoID)
	if err == sql.ErrNoRows {
		fail(w, 404, errors.New("视频不存在"))
		return
	}
	if err != nil {
		fail(w, 500, err)
		return
	}
	if v.Status != "completed" {
		fail(w, 409, errors.New("只有已下载媒体可以建立外部计算任务"))
		return
	}
	cfg, _ := a.store.Settings()
	if input.Type == "dedupe" {
		job, created, err := a.createMediaDuplicateJob(v, cfg, false)
		if err != nil {
			fail(w, 422, err)
			return
		}
		status := http.StatusCreated
		if !created {
			status = http.StatusOK
		}
		respond(w, status, map[string]any{"job": job, "created": created})
		return
	}
	path := v.PlexPath
	if path == "" {
		path = v.FilePath
	}
	rel, info, err := secureMediaRelative(cfg.DownloadDir, path)
	if err != nil {
		fail(w, 422, err)
		return
	}
	if len(input.Options) == 0 {
		input.Options = json.RawMessage(`{}`)
	}
	if !json.Valid(input.Options) {
		fail(w, 400, errors.New("options 必须是 JSON"))
		return
	}
	key := computeIdempotency(v.ID, info, input.Type, input.AlgorithmVersion)
	sum := sha256.Sum256([]byte(key))
	id := "job_" + hex.EncodeToString(sum[:12])
	job, created, err := a.store.CreateComputeJob(store.ComputeJob{ID: id, Type: input.Type, VideoID: v.ID, IdempotencyKey: key, AlgorithmVersion: input.AlgorithmVersion, MediaRelativePath: rel, InputJSON: string(input.Options)})
	if err != nil {
		fail(w, 500, err)
		return
	}
	if input.Type == "transcode" && created {
		_ = a.store.SetTranscodeStatus(v.ID, "pending")
	}
	status := http.StatusCreated
	if !created {
		status = http.StatusOK
	}
	respond(w, status, map[string]any{"job": job, "created": created})
}

func (a *App) cancelComputeJob(w http.ResponseWriter, r *http.Request) {
	job, err := a.store.ComputeJob(r.PathValue("id"))
	if err == sql.ErrNoRows {
		fail(w, 404, errors.New("任务不存在"))
		return
	}
	if err != nil {
		fail(w, 500, err)
		return
	}
	if err = a.store.CancelComputeJob(job.ID); err != nil {
		fail(w, 500, err)
		return
	}
	if job.Type == "transcode" {
		_ = a.store.SetTranscodeStatus(job.VideoID, "cancelled")
	}
	respond(w, 200, map[string]any{"cancelled": true})
}

func safeComputeFilename(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, value)
}

// enqueueAutoTranscode creates an idempotent H.265 transcode job for a freshly
// downloaded video when automatic transcoding is enabled. It is a no-op when the
// media directory is unavailable or a matching job already exists.
func (a *App) enqueueAutoTranscode(v store.Video) {
	if v.Status != "completed" {
		return
	}
	path := v.PlexPath
	if path == "" {
		path = v.FilePath
	}
	cfg, _ := a.store.Settings()
	rel, info, err := secureMediaRelative(cfg.DownloadDir, path)
	if err != nil {
		return
	}
	const version = "hevc-v1"
	key := computeIdempotency(v.ID, info, "transcode", version)
	sum := sha256.Sum256([]byte(key))
	id := "job_" + hex.EncodeToString(sum[:12])
	_, created, err := a.store.CreateComputeJob(store.ComputeJob{ID: id, Type: "transcode", VideoID: v.ID, IdempotencyKey: key, AlgorithmVersion: version, MediaRelativePath: rel, InputJSON: "{}"})
	if err == nil && created {
		_ = a.store.SetTranscodeStatus(v.ID, "pending")
	}
}

func (a *App) commitTranscodeJob(w http.ResponseWriter, r *http.Request) {
	job, err := a.store.ComputeJob(r.PathValue("id"))
	if err == sql.ErrNoRows {
		fail(w, 404, errors.New("任务不存在"))
		return
	}
	if err != nil {
		fail(w, 500, err)
		return
	}
	if job.Type != "transcode" || job.Status != "verifying" {
		fail(w, 409, errors.New("转码任务不在待提交状态"))
		return
	}
	var input struct {
		Confirm      bool  `json:"confirm"`
		KeepOriginal *bool `json:"keep_original"`
	}
	if err = decode(r, &input); err != nil {
		fail(w, 400, err)
		return
	}
	if !input.Confirm {
		fail(w, 422, errors.New("必须明确确认提交转码产物"))
		return
	}
	keep := true
	if input.KeepOriginal != nil {
		keep = *input.KeepOriginal
	}
	var result struct {
		TempRelativePath string `json:"temp_relative_path"`
	}
	if json.Unmarshal([]byte(job.ResultJSON), &result) != nil || result.TempRelativePath == "" {
		fail(w, 422, errors.New("转码结果缺少临时文件路径"))
		return
	}
	if err = a.commitTranscodeArtifact(job, result.TempRelativePath, keep); err != nil {
		fail(w, 422, err)
		return
	}
	if err = a.store.CommitComputeJob(job.ID); err != nil {
		fail(w, 500, err)
		return
	}
	video, _ := a.store.Video(job.VideoID)
	respond(w, 200, map[string]any{"status": "completed", "video_id": video.ID, "video_codec": video.VideoCodec, "duration_seconds": video.DurationSeconds})
}

// commitTranscodeArtifact validates a node-provided transcode output and
// atomically replaces the source media. The caller moves the compute job to its
// terminal state. When keepOriginal is false the pre-transcode file is removed
// after a successful swap.
func (a *App) commitTranscodeArtifact(job store.ComputeJob, tempRelativePath string, keepOriginal bool) error {
	video, err := a.store.Video(job.VideoID)
	if err != nil {
		return err
	}
	cfg, _ := a.store.Settings()
	source := video.PlexPath
	if source == "" {
		source = video.FilePath
	}
	if source == "" {
		return errors.New("媒体路径为空")
	}
	sourceRel, _, err := secureMediaRelative(cfg.DownloadDir, source)
	if err != nil {
		return err
	}
	clean := filepath.Clean(filepath.FromSlash(tempRelativePath))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return errors.New("转码产物路径无效")
	}
	temp := filepath.Join(cfg.DownloadDir, clean)
	tempRel, _, err := secureMediaRelative(cfg.DownloadDir, temp)
	if err != nil {
		return err
	}
	if sourceRel == tempRel {
		return errors.New("转码产物不能与源文件相同")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	sourceInfo, err := mediainfo.Probe(ctx, source)
	if err != nil {
		return fmt.Errorf("源文件验证失败: %w", err)
	}
	outputInfo, err := mediainfo.Probe(ctx, temp)
	if err != nil {
		return fmt.Errorf("转码产物验证失败: %w", err)
	}
	tolerance := math.Max(2, sourceInfo.DurationSeconds*.01)
	if sourceInfo.DurationSeconds > 0 && outputInfo.DurationSeconds > 0 && math.Abs(sourceInfo.DurationSeconds-outputInfo.DurationSeconds) > tolerance {
		return fmt.Errorf("转码产物时长差异过大: %.2f 秒", math.Abs(sourceInfo.DurationSeconds-outputInfo.DurationSeconds))
	}
	backupDir := filepath.Join(cfg.DownloadDir, ".hwatch-originals")
	if err = os.MkdirAll(backupDir, 0700); err != nil {
		return err
	}
	backup := filepath.Join(backupDir, safeComputeFilename(job.ID)+filepath.Ext(source))
	if _, statErr := os.Stat(backup); statErr == nil {
		return errors.New("该任务的原文件备份已存在")
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	if err = os.Rename(source, backup); err != nil {
		return fmt.Errorf("暂存原文件失败: %w", err)
	}
	if err = os.Rename(temp, source); err != nil {
		_ = os.Rename(backup, source)
		return fmt.Errorf("提交转码产物失败: %w", err)
	}
	if err = a.store.UpdateMediaInfo(video.ID, outputInfo.VideoCodec, outputInfo.AudioCodec, outputInfo.Container, outputInfo.Width, outputInfo.Height, outputInfo.FrameRate, outputInfo.BitRate, outputInfo.BitDepth); err != nil {
		_ = os.Rename(source, temp)
		_ = os.Rename(backup, source)
		return err
	}
	if err = a.store.SetTranscodeStatus(video.ID, "completed"); err != nil {
		return err
	}
	_ = a.store.ClearFileHashes(video.ID)
	a.hashCompletedMedia(video.ID, source)
	if !keepOriginal {
		if err = os.Remove(backup); err != nil {
			_, _ = a.store.Notify("transcode_backup_cleanup", "warning", "转码成功，但原文件暂存副本清理失败", job.ID, 24*time.Hour)
		}
	}
	return nil
}

func (a *App) authenticateComputeNode(r *http.Request) (store.ComputeNode, error) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return store.ComputeNode{}, errors.New("缺少节点令牌")
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" {
		return store.ComputeNode{}, errors.New("缺少节点令牌")
	}
	return a.store.ComputeNodeByToken(tokenHash(token))
}

func capabilitiesAllow(raw, kind string) bool {
	var c nodeCapabilities
	if json.Unmarshal([]byte(raw), &c) != nil {
		return false
	}
	return kind == "dedupe" && c.Dedupe || kind == "transcode" && c.Transcode
}

func (a *App) nodeHeartbeat(w http.ResponseWriter, r *http.Request) {
	node, err := a.authenticateComputeNode(r)
	if err != nil {
		fail(w, 401, err)
		return
	}
	var input struct {
		Capabilities nodeCapabilities `json:"capabilities"`
		Version      string           `json:"version"`
		Platform     string           `json:"platform"`
	}
	if err = decode(r, &input); err != nil {
		fail(w, 400, err)
		return
	}
	raw, _ := json.Marshal(input.Capabilities)
	if err = a.store.HeartbeatComputeNode(node.ID, string(raw), input.Version, input.Platform); err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]any{"node_id": node.ID, "lease_seconds": int(computeLease.Seconds()), "server_time": store.Now()})
}

func (a *App) nodeClaim(w http.ResponseWriter, r *http.Request) {
	node, err := a.authenticateComputeNode(r)
	if err != nil {
		fail(w, 401, err)
		return
	}
	var input struct {
		Types []string `json:"types"`
	}
	if err = decode(r, &input); err != nil {
		fail(w, 400, err)
		return
	}
	allowed := []string{}
	subtypes := []string{}
	seen := map[string]bool{}
	for _, kind := range input.Types {
		if !seen[kind] && capabilitiesAllow(node.PermissionsJSON, kind) && capabilitiesAllow(node.CapabilitiesJSON, kind) {
			allowed = append(allowed, kind)
			seen[kind] = true
			if kind == "dedupe" && nodeCapabilityFlags(node.PermissionsJSON, node.CapabilitiesJSON, "media_duplicate") {
				subtypes = append(subtypes, mediaDuplicateSubtype)
			}
		}
	}
	job, err := a.store.ClaimComputeJob(node.ID, allowed, subtypes, computeLease)
	if errors.Is(err, store.ErrNoComputeJob) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]any{"job": job, "lease_seconds": int(computeLease.Seconds())})
}

func (a *App) nodeProgress(w http.ResponseWriter, r *http.Request) {
	node, err := a.authenticateComputeNode(r)
	if err != nil {
		fail(w, 401, err)
		return
	}
	var input struct {
		Status   string  `json:"status"`
		Progress float64 `json:"progress"`
	}
	if err = decode(r, &input); err != nil {
		fail(w, 400, err)
		return
	}
	if err = a.store.RenewComputeLease(r.PathValue("id"), node.ID, input.Status, input.Progress, computeLease); err != nil {
		fail(w, 409, err)
		return
	}
	respond(w, 200, map[string]any{"lease_seconds": int(computeLease.Seconds())})
}

type dedupeNodeMatch struct {
	VideoID          string          `json:"video_id"`
	Relationship     string          `json:"relationship,omitempty"`
	Classification   string          `json:"classification,omitempty"`
	Backend          string          `json:"backend,omitempty"`
	AlgorithmVersion string          `json:"algorithm_version,omitempty"`
	Confidence       float64         `json:"confidence"`
	DurationDelta    int64           `json:"duration_delta"`
	Coverage         float64         `json:"coverage"`
	Metrics          json.RawMessage `json:"metrics,omitempty"`
}

type dedupeNodeResult struct {
	Matches []dedupeNodeMatch `json:"matches"`
}

func (a *App) nodeComplete(w http.ResponseWriter, r *http.Request) {
	node, err := a.authenticateComputeNode(r)
	if err != nil {
		fail(w, 401, err)
		return
	}
	job, err := a.store.ComputeJob(r.PathValue("id"))
	if err != nil {
		fail(w, 404, errors.New("任务不存在"))
		return
	}
	if job.NodeID != node.ID {
		fail(w, 409, store.ErrComputeLease)
		return
	}
	if job.Status != "assigned" && job.Status != "analyzing" && job.Status != "matching" && job.Status != "transcoding" {
		fail(w, 409, store.ErrComputeLease)
		return
	}
	var input struct {
		Result json.RawMessage `json:"result"`
	}
	if err = decode(r, &input); err != nil || !json.Valid(input.Result) {
		if err == nil {
			err = errors.New("result 必须是 JSON")
		}
		fail(w, 400, err)
		return
	}
	finalStatus := "completed"
	if job.Type == "dedupe" {
		var result dedupeNodeResult
		if err = json.Unmarshal(input.Result, &result); err != nil {
			fail(w, 400, err)
			return
		}
		if job.Subtype == mediaDuplicateSubtype {
			if err = a.applyMediaDuplicateResult(job, result); err != nil {
				fail(w, 422, err)
				return
			}
		} else {
			allowed := map[string]bool{"exact": true, "reencoded": true, "contained": true, "partial_overlap": true, "compilation_contains": true}
			for _, match := range result.Matches {
				if !allowed[match.Relationship] || match.VideoID == job.VideoID || match.Confidence < 0 || match.Confidence > 1 {
					fail(w, 422, errors.New("去重匹配结果无效"))
					return
				}
				if _, e := a.store.Video(match.VideoID); e != nil {
					fail(w, 422, errors.New("匹配视频不存在"))
					return
				}
				if e := a.store.AddDuplicate(job.VideoID, match.VideoID, "external_"+match.Relationship, match.Confidence, match.DurationDelta, false); e != nil {
					fail(w, 500, e)
					return
				}
			}
		}
	} else {
		var result struct {
			TempRelativePath string `json:"temp_relative_path"`
		}
		if err = json.Unmarshal(input.Result, &result); err != nil || result.TempRelativePath == "" {
			fail(w, 422, errors.New("转码结果缺少 temp_relative_path"))
			return
		}
		clean := filepath.Clean(filepath.FromSlash(result.TempRelativePath))
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			fail(w, 422, errors.New("转码输出路径无效"))
			return
		}
		if filepath.ToSlash(clean) == job.MediaRelativePath {
			fail(w, 422, errors.New("转码临时产物不能覆盖输入文件"))
			return
		}
		// Transcoding is fully automatic: verify the artifact with ffprobe and
		// atomically replace the source without an administrator confirmation.
		if err = a.commitTranscodeArtifact(job, result.TempRelativePath, false); err != nil {
			_ = a.store.SetTranscodeStatus(job.VideoID, "failed")
			_ = a.store.FailComputeJob(job.ID, node.ID, "transcode_commit_failed", err.Error(), false)
			fail(w, 422, err)
			return
		}
	}
	if err = a.store.CompleteComputeJob(job.ID, node.ID, finalStatus, string(input.Result)); err != nil {
		fail(w, 409, err)
		return
	}
	respond(w, 200, map[string]any{"status": finalStatus})
}

func (a *App) nodeFail(w http.ResponseWriter, r *http.Request) {
	node, err := a.authenticateComputeNode(r)
	if err != nil {
		fail(w, 401, err)
		return
	}
	var input struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Retry   bool   `json:"retry"`
	}
	if err = decode(r, &input); err != nil {
		fail(w, 400, err)
		return
	}
	if len(input.Code) > 64 {
		fail(w, 422, errors.New("错误码过长"))
		return
	}
	if len(input.Message) > 1000 {
		input.Message = input.Message[:1000]
	}
	job, _ := a.store.ComputeJob(r.PathValue("id"))
	if err = a.store.FailComputeJob(r.PathValue("id"), node.ID, input.Code, input.Message, input.Retry); err != nil {
		fail(w, 409, err)
		return
	}
	if job.Type == "transcode" {
		status := "failed"
		if input.Retry {
			status = "pending"
		}
		_ = a.store.SetTranscodeStatus(job.VideoID, status)
	}
	respond(w, 200, map[string]any{"status": map[bool]string{true: "pending", false: "failed"}[input.Retry]})
}
