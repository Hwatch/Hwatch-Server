package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"r34watch/internal/store"
)

// Media Duplicate is a server-driven content comparison layer. HWatch selects
// candidates and records results; the external node only compares the exact
// target/candidate pairs it is given and never scans the library or deletes
// media.
const (
	mediaDuplicateAlgorithm          = "media-fingerprint-v1"
	mediaDuplicateSubtype            = "media_duplicate"
	mediaDuplicateMode               = "media_duplicate"
	mediaDuplicateMinToleranceSecond = 2
	mediaDuplicateScheduleBatch      = 5
	defaultMediaDuplicateCandidates  = 20
	defaultMediaDuplicateDurationPct = 1
)

// mediaDuplicateClassifications are the only classification values accepted from
// a node. Only non-"different" values produce a duplicate candidate.
var mediaDuplicateClassifications = map[string]bool{
	"whole-video":    true,
	"trimmed-video":  true,
	"possible-match": true,
	"different":      true,
}

type mediaDuplicateCandidate struct {
	VideoID           string `json:"video_id"`
	MediaRelativePath string `json:"media_relative_path"`
}

type mediaDuplicateOptions struct {
	Mode       string                    `json:"mode"`
	Candidates []mediaDuplicateCandidate `json:"candidates"`
}

// mediaDuplicateTolerance returns the allowed duration difference, centralised so
// the initial candidate rule is not scattered across the codebase.
func mediaDuplicateTolerance(duration int64, percent int) int64 {
	if percent < 1 {
		percent = defaultMediaDuplicateDurationPct
	}
	tolerance := duration * int64(percent) / 100
	if tolerance < mediaDuplicateMinToleranceSecond {
		tolerance = mediaDuplicateMinToleranceSecond
	}
	return tolerance
}

func computeMediaPath(v store.Video) string {
	if v.PlexPath != "" {
		return v.PlexPath
	}
	return v.FilePath
}

// buildMediaDuplicateOptions assembles the node payload for a target media:
// duration-proximate candidates, skipping exact SHA-256 duplicates, missing
// files and paths that escape the media root.
func (a *App) buildMediaDuplicateOptions(target store.Video, cfg store.Settings) (mediaDuplicateOptions, string, error) {
	options := mediaDuplicateOptions{Mode: mediaDuplicateMode, Candidates: []mediaDuplicateCandidate{}}
	sha, err := a.store.MediaDuplicateHash(target.ID)
	if err != nil {
		return options, "", err
	}
	limit := cfg.MediaDuplicateMaxCandidates
	if limit <= 0 {
		limit = defaultMediaDuplicateCandidates
	}
	tolerance := mediaDuplicateTolerance(target.DurationSeconds, cfg.MediaDuplicateDurationPercent)
	rows, err := a.store.MediaDuplicateCandidates(target.ID, sha, target.DurationSeconds, tolerance, limit*3)
	if err != nil {
		return options, sha, err
	}
	for _, candidate := range rows {
		if len(options.Candidates) >= limit {
			break
		}
		rel, _, err := secureMediaRelative(cfg.DownloadDir, computeMediaPath(candidate))
		if err != nil {
			continue
		}
		if info, err := os.Stat(filepath.Join(cfg.DownloadDir, filepath.FromSlash(rel))); err != nil || !info.Mode().IsRegular() {
			continue
		}
		options.Candidates = append(options.Candidates, mediaDuplicateCandidate{VideoID: candidate.ID, MediaRelativePath: rel})
	}
	return options, sha, nil
}

// mediaDuplicateJobID derives a deterministic, idempotent id/key from the target
// and its exact candidate set, so the same comparison is never queued twice.
func mediaDuplicateJobID(targetID, version string, candidateIDs []string) (string, string) {
	sorted := append([]string(nil), candidateIDs...)
	sort.Strings(sorted)
	sum := sha256.Sum256([]byte(targetID + "\x00" + version + "\x00" + strings.Join(sorted, ",")))
	key := hex.EncodeToString(sum[:])
	return "job_" + key[:24], key
}

// createMediaDuplicateJob builds and persists a media-duplicate job for one
// target. When skipEmpty is true (scheduler path) a target with no candidates
// produces no job.
func (a *App) createMediaDuplicateJob(target store.Video, cfg store.Settings, skipEmpty bool) (store.ComputeJob, bool, error) {
	options, _, err := a.buildMediaDuplicateOptions(target, cfg)
	if err != nil {
		return store.ComputeJob{}, false, err
	}
	if skipEmpty && len(options.Candidates) == 0 {
		return store.ComputeJob{}, false, nil
	}
	rel, _, err := secureMediaRelative(cfg.DownloadDir, computeMediaPath(target))
	if err != nil {
		return store.ComputeJob{}, false, err
	}
	ids := make([]string, 0, len(options.Candidates))
	for _, candidate := range options.Candidates {
		ids = append(ids, candidate.VideoID)
	}
	id, key := mediaDuplicateJobID(target.ID, mediaDuplicateAlgorithm, ids)
	raw, err := json.Marshal(options)
	if err != nil {
		return store.ComputeJob{}, false, err
	}
	return a.store.CreateComputeJob(store.ComputeJob{ID: id, Type: "dedupe", Subtype: mediaDuplicateSubtype, VideoID: target.ID, IdempotencyKey: key, AlgorithmVersion: mediaDuplicateAlgorithm, MediaRelativePath: rel, InputJSON: string(raw)})
}

func nodeCapabilityFlags(permissionsJSON, capabilitiesJSON, kind string) bool {
	parse := func(raw string) nodeCapabilities {
		var c nodeCapabilities
		_ = json.Unmarshal([]byte(raw), &c)
		return c
	}
	granted, reported := parse(permissionsJSON), parse(capabilitiesJSON)
	switch kind {
	case "media_duplicate":
		return granted.MediaDuplicate && reported.MediaDuplicate
	default:
		return false
	}
}

// mediaDuplicateNode reports whether an online node is both allowed and capable
// of media-duplicate work.
func (a *App) mediaDuplicateNode() (store.ComputeNode, bool) {
	nodes, err := a.store.ComputeNodes()
	if err != nil {
		return store.ComputeNode{}, false
	}
	now := time.Now()
	for _, n := range nodes {
		if n.RevokedAt != "" {
			continue
		}
		if seen, e := time.Parse(time.RFC3339, n.LastSeenAt); e != nil || now.Sub(seen) > 3*time.Minute {
			continue
		}
		if nodeCapabilityFlags(n.PermissionsJSON, n.CapabilitiesJSON, "media_duplicate") {
			return n, true
		}
	}
	return store.ComputeNode{}, false
}

// applyMediaDuplicateResult validates a node's media-duplicate comparison and
// records only the pairs the node was asked about. "different" never creates a
// candidate, and confidence is stored but never used to auto-delete media.
func (a *App) applyMediaDuplicateResult(job store.ComputeJob, result dedupeNodeResult) error {
	var options mediaDuplicateOptions
	if err := json.Unmarshal([]byte(job.InputJSON), &options); err != nil {
		return errors.New("任务候选数据无效")
	}
	allowed := make(map[string]bool, len(options.Candidates))
	for _, candidate := range options.Candidates {
		allowed[candidate.VideoID] = true
	}
	for _, match := range result.Matches {
		if !mediaDuplicateClassifications[match.Classification] {
			return fmt.Errorf("未知的重复分类 %q", match.Classification)
		}
		if match.VideoID == job.VideoID || !allowed[match.VideoID] {
			return errors.New("去重结果引用了任务候选之外的媒体")
		}
		if match.Confidence < 0 || match.Confidence > 1 {
			return errors.New("去重结果置信度无效")
		}
		if match.Classification == "different" {
			continue
		}
		backend := match.Backend
		if backend == "" {
			backend = mediaDuplicateAlgorithm
		}
		version := match.AlgorithmVersion
		if version == "" {
			version = job.AlgorithmVersion
		}
		metrics := "{}"
		if len(match.Metrics) > 0 && json.Valid(match.Metrics) {
			metrics = string(match.Metrics)
		}
		created, err := a.store.AddMediaDuplicate(job.VideoID, match.VideoID, match.Classification, backend, version, metrics, match.Confidence, match.DurationDelta)
		if err != nil {
			return err
		}
		if created {
			_ = a.store.AddAudit("node:"+job.NodeID, "media_duplicate_candidate", "video", job.VideoID, "success", map[string]any{"match_video_id": match.VideoID, "classification": match.Classification, "backend": backend, "algorithm_version": version})
		}
	}
	return nil
}

// scheduleMediaDuplicateJobs is called from the scheduler. It never blocks the
// download path, only queues a bounded batch, and does nothing when the feature
// is disabled, storage is unavailable, or no capable node is online.
func (a *App) scheduleMediaDuplicateJobs() {
	cfg, err := a.store.Settings()
	if err != nil || !cfg.EnableMediaDuplicate {
		return
	}
	if !storagePathAvailable(cfg.DownloadDir, cfg.StorageMountRequired) {
		return
	}
	if _, ok := a.mediaDuplicateNode(); !ok {
		return
	}
	targets, err := a.store.MediaDuplicateTargets(mediaDuplicateAlgorithm, mediaDuplicateScheduleBatch)
	if err != nil {
		return
	}
	for _, target := range targets {
		job, created, err := a.createMediaDuplicateJob(target, cfg, true)
		if err != nil || !created {
			continue
		}
		_ = a.store.AddAudit("scheduler", "media_duplicate_scheduled", "video", target.ID, "success", map[string]any{"job_id": job.ID, "algorithm_version": mediaDuplicateAlgorithm})
	}
}
