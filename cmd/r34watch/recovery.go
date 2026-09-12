package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"r34watch/internal/store"
)

type recoverySummary struct {
	CompletedAt        string `json:"completed_at"`
	SourceMediaRoot    string `json:"source_media_root"`
	CurrentMediaRoot   string `json:"current_media_root"`
	Subscriptions      int    `json:"subscriptions"`
	CompletedMedia     int    `json:"completed_media"`
	MissingMedia       int    `json:"missing_media"`
	PathMismatches     int    `json:"path_mismatches"`
	PathsRemapped      int    `json:"paths_remapped"`
	AriaTasksChecked   int    `json:"aria_tasks_checked"`
	AriaTasksKept      int    `json:"aria_tasks_kept"`
	AriaTasksRequeued  int    `json:"aria_tasks_requeued"`
	AriaUnavailable    bool   `json:"aria_unavailable"`
	ReconciliationDone bool   `json:"reconciliation_done"`
}

func pathWithin(root, path string) bool {
	if root == "" || path == "" {
		return false
	}
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func remapPath(path, sourceRoot, targetRoot string) (string, bool) {
	if !pathWithin(sourceRoot, path) || sourceRoot == targetRoot {
		return path, false
	}
	rel, err := filepath.Rel(sourceRoot, path)
	if err != nil {
		return path, false
	}
	return filepath.Join(targetRoot, rel), true
}

func missingAriaGID(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "gid") && strings.Contains(message, "not found")
}

func (a *App) reconcileRecovery() {
	a.recoveryMu.Lock()
	defer a.recoveryMu.Unlock()
	pending, _ := a.store.Setting("restore_pending")
	if pending != "true" {
		return
	}
	cfg, _ := a.store.Settings()
	sourceRoot, _ := a.store.Setting("restore_source_root")
	targetRoot, _ := a.store.Setting("restore_target_root")
	if targetRoot == "" {
		targetRoot = cfg.DownloadDir
	}
	report := recoverySummary{CompletedAt: store.Now(), SourceMediaRoot: sourceRoot, CurrentMediaRoot: targetRoot}
	subscriptions, _ := a.store.Subscriptions()
	report.Subscriptions = len(subscriptions)
	videos, _ := a.store.Videos(100000, 0, "", "")
	for _, video := range videos {
		filePath, fileChanged := remapPath(video.FilePath, sourceRoot, targetRoot)
		plexPath, plexChanged := remapPath(video.PlexPath, sourceRoot, targetRoot)
		if fileChanged || plexChanged {
			_ = a.store.UpdateMediaPaths(video.ID, filePath, plexPath)
			report.PathsRemapped++
		}
		if video.Status != "completed" {
			continue
		}
		report.CompletedMedia++
		path := plexPath
		if path == "" {
			path = filePath
		}
		if path == "" {
			report.MissingMedia++
			continue
		}
		if _, err := os.Stat(path); err != nil {
			report.MissingMedia++
		}
		if !pathWithin(targetRoot, path) {
			report.PathMismatches++
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	ariaOnline := a.aria(cfg).Health(ctx).Online
	cancel()
	report.AriaUnavailable = !ariaOnline
	transientFailure := !ariaOnline
	if ariaOnline {
		for _, state := range []string{"active", "waiting", "paused"} {
			items, _ := a.store.Videos(100000, 0, state, "")
			for _, video := range items {
				if video.DownloaderID != "aria2" && video.AriaGID == "" {
					continue
				}
				report.AriaTasksChecked++
				ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
				_, err := a.aria(cfg).Tell(ctx, externalTaskID(video))
				cancel()
				if err == nil {
					report.AriaTasksKept++
					continue
				}
				if !missingAriaGID(err) {
					transientFailure = true
					continue
				}
				_ = a.store.UpdateDownloaderTask(video.ID, "queued", "", "", video.FilePath, video.PlexPath, "恢复后原 aria2 GID 不存在，已重新排队", video.BytesDownloaded, video.TotalBytes)
				report.AriaTasksRequeued++
			}
		}
	}
	report.ReconciliationDone = !transientFailure
	raw, _ := json.Marshal(report)
	_ = a.store.SetSetting("restore_last_summary", string(raw))
	if report.ReconciliationDone {
		_ = a.store.SetSetting("restore_pending", "false")
	}
}

func (a *App) recoveryStatus(w http.ResponseWriter, _ *http.Request) {
	raw, _ := a.store.Setting("restore_last_summary")
	if raw == "" {
		respond(w, http.StatusOK, map[string]any{"available": false})
		return
	}
	var report recoverySummary
	_ = json.Unmarshal([]byte(raw), &report)
	respond(w, http.StatusOK, map[string]any{"available": true, "summary": report})
}
