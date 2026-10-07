package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"r34watch/internal/store"
)

const (
	deleteModeOnly    = "delete_only"
	deleteModeExclude = "delete_and_exclude"
)

var errVideoExcluded = errors.New("该视频已加入“已排除”，请先恢复允许下载")

// deleteVideo removes a single video's media file. The mode decides whether a
// media_exclusions tombstone is also written. The media path is always resolved
// server-side from the DB and validated against the media root.
func (a *App) deleteVideo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var input struct {
		Mode string `json:"mode"`
	}
	_ = decode(r, &input)
	mode := strings.TrimSpace(input.Mode)
	if mode == "" {
		mode = deleteModeOnly
	}
	if mode != deleteModeOnly && mode != deleteModeExclude {
		fail(w, http.StatusUnprocessableEntity, errors.New("未知的删除模式"))
		return
	}
	v, err := a.store.Video(id)
	if err != nil {
		fail(w, http.StatusNotFound, errors.New("视频不存在"))
		return
	}
	cfg, _ := a.store.Settings()
	if !storagePathAvailable(cfg.DownloadDir, cfg.StorageMountRequired) {
		fail(w, http.StatusServiceUnavailable, errors.New("媒体存储当前不可用"))
		return
	}

	// Stop any in-flight downloader task for this video.
	if externalID := externalTaskID(v); externalID != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		_ = a.engineFor(v, cfg).Remove(ctx, externalID)
		cancel()
	}

	// Stage the media file (same directory) so a DB failure can roll back.
	staged := false
	abs := ""
	trash := ""
	if source := mediaPathOf(v); source != "" {
		if _, statErr := os.Stat(source); statErr == nil {
			if _, err := mediaRelativePath(cfg.DownloadDir, source); err != nil {
				fail(w, http.StatusConflict, errors.New("媒体路径不在媒体根目录内，拒绝删除"))
				return
			}
			rel, _ := mediaRelativePath(cfg.DownloadDir, source)
			abs = filepath.Join(cfg.DownloadDir, filepath.FromSlash(rel))
			trash = abs + ".hwatch-trash"
			_ = os.Remove(trash)
			if err := os.Rename(abs, trash); err != nil {
				fail(w, http.StatusInternalServerError, errors.New("暂存媒体文件失败"))
				return
			}
			staged = true
		}
	}

	if mode == deleteModeExclude {
		if err := a.store.AddMediaExclusion(store.MediaExclusion{VideoID: id, Source: v.Platform, Reason: "user_delete", TitleSnapshot: v.Title, OriginalURLSnapshot: v.PageURL, ThumbnailSnapshot: v.ThumbnailURL}); err != nil {
			if staged {
				_ = os.Rename(trash, abs)
			}
			fail(w, http.StatusInternalServerError, err)
			return
		}
	}
	if err := a.store.MarkVideoRemoved(id); err != nil {
		if mode == deleteModeExclude {
			_ = a.store.RemoveMediaExclusion(id)
		}
		if staged {
			_ = os.Rename(trash, abs)
		}
		fail(w, http.StatusInternalServerError, err)
		return
	}

	if staged {
		_ = os.Remove(trash)
		base := strings.TrimSuffix(abs, filepath.Ext(abs))
		_ = os.Remove(base + ".nfo")
		_ = os.Remove(base + ".jpg")
		_ = os.Remove(abs + ".aria2")
	}
	enabled := true
	_ = a.store.AddAudit("admin", "media_deleted", "video", id, "success", map[string]any{"mode": mode, "excluded": enabled && mode == deleteModeExclude})
	respond(w, http.StatusOK, map[string]any{"status": "deleted", "mode": mode})
}

func (a *App) mediaExclusions(w http.ResponseWriter, r *http.Request) {
	limit := intParam(r, "limit", 100, 1, 500)
	offset := intParam(r, "offset", 0, 0, 10000000)
	items, total, err := a.store.MediaExclusions(limit, offset, strings.TrimSpace(r.URL.Query().Get("q")), strings.TrimSpace(r.URL.Query().Get("source")))
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	respond(w, http.StatusOK, map[string]any{"items": items, "total": total, "limit": limit, "offset": offset})
}

func (a *App) removeMediaExclusion(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("video_id")
	if err := a.store.RemoveMediaExclusion(id); err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	_ = a.store.AddAudit("admin", "media_exclusion_removed", "video", id, "success", map[string]any{})
	respond(w, http.StatusOK, map[string]any{"restored": true})
}
