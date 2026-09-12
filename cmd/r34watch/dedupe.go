package main

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"r34watch/internal/dedupe"
	"r34watch/internal/store"
)

func (a *App) duplicateBeforeDownload(v store.Video, cfg store.Settings) error {
	normalized := dedupe.NormalizeURL(v.PageURL)
	_ = a.store.UpsertFingerprint(v.ID, normalized, "", v.DurationSeconds)
	if match, err := a.store.IdentityDuplicate(v.ID, normalized); err == nil && match != "" && !a.store.DuplicateDismissed(v.ID, match, "page_url") {
		_ = a.store.AddDuplicate(v.ID, match, "page_url", 1, 0, true)
		return errors.New("duplicate: 详情页与已有记录一致")
	}
	if v.DurationSeconds <= 0 || v.ThumbnailURL == "" {
		return nil
	}
	hash, err := a.coverHash(v, cfg)
	if err != nil {
		return nil
	}
	_ = a.store.UpsertFingerprint(v.ID, normalized, hash, v.DurationSeconds)
	candidates, _ := a.store.CoverCandidates(v.ID, v.DurationSeconds)
	for _, candidate := range candidates {
		confidence := dedupe.Similarity(hash, candidate.CoverHash)
		if confidence >= .92 && !a.store.DuplicateDismissed(v.ID, candidate.VideoID, "cover_duration") {
			delta := v.DurationSeconds - candidate.DurationSeconds
			if delta < 0 {
				delta = -delta
			}
			_ = a.store.AddDuplicate(v.ID, candidate.VideoID, "cover_duration", confidence, delta, true)
			return errors.New("duplicate: 封面高度相似且时长相差不超过 5 秒")
		}
	}
	return nil
}
func (a *App) coverHash(v store.Video, cfg store.Settings) (string, error) {
	if path := firstExistingThumbnail(v); path != "" {
		f, e := os.Open(path)
		if e != nil {
			return "", e
		}
		defer f.Close()
		return dedupe.ImageHash(f)
	}
	transport := &http.Transport{}
	if cfg.HTTPProxy != "" {
		if p, e := url.Parse(cfg.HTTPProxy); e == nil {
			transport.Proxy = http.ProxyURL(p)
		}
	}
	client := &http.Client{Transport: transport, Timeout: 20 * time.Second}
	req, e := http.NewRequestWithContext(context.Background(), http.MethodGet, v.ThumbnailURL, nil)
	if e != nil {
		return "", e
	}
	req.Header.Set("User-Agent", userAgent)
	res, e := client.Do(req)
	if e != nil {
		return "", e
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", errors.New("thumbnail unavailable")
	}
	return dedupe.ImageHash(res.Body)
}
func (a *App) hashCompletedMedia(videoID, path string) {
	a.hashMu.Lock()
	if a.hashRunning[videoID] {
		a.hashMu.Unlock()
		return
	}
	a.hashRunning[videoID] = true
	a.hashMu.Unlock()
	go func() {
		defer func() { a.hashMu.Lock(); delete(a.hashRunning, videoID); a.hashMu.Unlock() }()
		size, quick, sha, err := dedupe.FileHashes(path)
		if err != nil {
			return
		}
		if a.store.SaveFileHashes(videoID, size, quick, sha) != nil {
			return
		}
		if match, e := a.store.ExactHashDuplicate(videoID, sha); e == nil && match != "" && !a.store.DuplicateDismissed(videoID, match, "sha256") {
			_ = a.store.AddDuplicate(videoID, match, "sha256", 1, 0, false)
		}
	}()
}

func (a *App) backfillCoverFingerprints() {
	for _, status := range []string{"completed", "discovered", "excluded"} {
		items, _ := a.store.Videos(100000, 0, status, "")
		for _, v := range items {
			hash := ""
			if path := firstExistingThumbnail(v); path != "" {
				if file, err := os.Open(path); err == nil {
					hash, _ = dedupe.ImageHash(file)
					_ = file.Close()
				}
			}
			_ = a.store.UpsertFingerprint(v.ID, dedupe.NormalizeURL(v.PageURL), hash, v.DurationSeconds)
		}
	}
}
func (a *App) hashNextCompletedMedia() {
	a.hashMu.Lock()
	busy := len(a.hashRunning) > 0
	a.hashMu.Unlock()
	if busy {
		return
	}
	v, err := a.store.MissingFileHash()
	if err != nil {
		return
	}
	path := v.PlexPath
	if path == "" {
		path = v.FilePath
	}
	if _, err = os.Stat(path); err == nil {
		a.hashCompletedMedia(v.ID, path)
	}
}

func (a *App) duplicates(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		items, err := a.store.DuplicateCandidates()
		if err != nil {
			fail(w, 500, err)
			return
		}
		respond(w, 200, map[string]any{"items": items})
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	action := r.PathValue("action")
	if err != nil || (action != "confirm" && action != "dismiss") {
		fail(w, 400, errors.New("无效的重复项操作"))
		return
	}
	videoID, err := a.store.ReviewDuplicate(id, action)
	if err != nil {
		fail(w, 500, err)
		return
	}
	if action == "dismiss" {
		a.startQueueWorker()
	}
	respond(w, 200, map[string]any{"video_id": videoID, "status": map[bool]string{true: "dismissed", false: "confirmed"}[action == "dismiss"]})
}
