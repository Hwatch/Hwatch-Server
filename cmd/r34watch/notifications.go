package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/sys/unix"
)

func (a *App) evaluateNotifications(ctx context.Context) {
	cfg, err := a.store.Settings()
	if err != nil {
		return
	}
	total, free := diskStats(cfg.DownloadDir)
	fsType := filesystemType(cfg.DownloadDir)
	previousFS, _ := a.store.Setting("media_filesystem_type")
	previousPath, _ := a.store.Setting("media_filesystem_path")
	sameConfiguredPath := previousPath == cfg.DownloadDir
	filesystemChanged := sameConfiguredPath && previousFS != "" && fsType != "" && previousFS != fsType
	if total == 0 || filesystemChanged {
		_, _ = a.store.Notify("storage_unavailable", "error", "媒体目录或 NFS 当前不可用", "media-root", 6*time.Hour)
	} else if cfg.MinFreeBytes > 0 && free < cfg.MinFreeBytes {
		_, _ = a.store.Notify("storage_low", "error", fmt.Sprintf("媒体目录可用空间低于保留值，当前剩余 %.1f GB", float64(free)/(1<<30)), "media-root", 6*time.Hour)
	}
	if (!sameConfiguredPath || previousFS == "") && fsType != "" {
		_ = a.store.SetSetting("media_filesystem_type", fsType)
		_ = a.store.SetSetting("media_filesystem_path", cfg.DownloadDir)
	}
	failures, _ := a.store.ConsecutiveFailures(3)
	if failures >= 3 {
		_, _ = a.store.Notify("consecutive_download_failures", "warning", "最近连续 3 个下载任务失败，请检查下载器、代理或下载源", "downloads", 6*time.Hour)
	}
	health := a.selectedEngine(cfg).Health(ctx)
	if !health.Online {
		_, _ = a.store.Notify("downloader_unavailable", "error", "当前选择的下载器不可用，持久队列已停止建立新任务", cfg.DownloaderMode, 6*time.Hour)
	}
}

func filesystemType(path string) string {
	var stat unix.Statfs_t
	if unix.Statfs(path, &stat) != nil {
		return ""
	}
	return fmt.Sprintf("%x", stat.Type)
}

func (a *App) notifications(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := a.store.Notifications(intParam(r, "limit", 50, 1, 200))
		if err != nil {
			fail(w, 500, err)
			return
		}
		unread, _ := a.store.UnreadNotifications()
		respond(w, 200, map[string]any{"items": items, "unread": unread})
	case http.MethodPost:
		if err := a.store.MarkNotificationsRead(); err != nil {
			fail(w, 500, err)
			return
		}
		respond(w, 200, map[string]any{"read": true})
	}
}
