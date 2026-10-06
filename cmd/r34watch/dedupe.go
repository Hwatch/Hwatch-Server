package main

import (
	"errors"
	"net/http"
	"os"
	"strconv"

	"r34watch/internal/dedupe"
)

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
