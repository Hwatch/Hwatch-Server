package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"r34watch/internal/store"
)

// Media integrity scan + external VDF deletion-list reconciliation.
//
// The scan is read-only. Cleanup is only ever applied to entries that an
// external deletion list explicitly confirms by canonical video_id, and only
// after the media storage is verified mounted and the file is verified absent.
// Unknown missing files are reported, never auto-excluded.

const (
	scanStatusPresent     = "present"
	scanStatusMissing     = "missing"
	scanStatusMoved       = "moved"
	scanStatusPermission  = "permission_denied"
	scanStatusPathInvalid = "path_invalid"
	scanStatusUnreadable  = "unreadable"
	scanStatusExcluded    = "excluded"

	cleanupCandidate       = "candidate"
	cleanupAlreadyExcluded = "already_excluded"
	cleanupAlreadyRemoved  = "already_removed"
	cleanupUnknownVideo    = "unknown_video"
	cleanupNotCompleted    = "not_completed"
	cleanupActive          = "active"
	cleanupPathMismatch    = "path_mismatch"
	cleanupFilePresent     = "file_present"
	cleanupPathInvalid     = "path_invalid"
)

type mediaScanItem struct {
	VideoID  string `json:"video_id"`
	Platform string `json:"platform"`
	Title    string `json:"title"`
	Path     string `json:"path"`
	Status   string `json:"status"`
	Detail   string `json:"detail,omitempty"`
}

type mediaScanResult struct {
	StorageAvailable bool            `json:"storage_available"`
	Total            int             `json:"total"`
	Present          int             `json:"present"`
	Missing          int             `json:"missing"`
	Moved            int             `json:"moved"`
	PermissionDenied int             `json:"permission_denied"`
	PathInvalid      int             `json:"path_invalid"`
	Unreadable       int             `json:"unreadable"`
	Items            []mediaScanItem `json:"items"`
	ScannedAt        string          `json:"scanned_at"`
}

// vdfDeletion is one entry of an external deletion list. Only VideoID and
// OriginalPath are required; the rest is metadata/evidence.
type vdfDeletion struct {
	VideoID      string `json:"video_id"`
	OriginalPath string `json:"original_path"`
	Size         int64  `json:"size"`
	DeletedAt    string `json:"deleted_at"`
	Reason       string `json:"reason"`
	KeptVideoID  string `json:"kept_video_id"`
	Evidence     string `json:"evidence"`
	Reference    string `json:"reference"`
}

type vdfList struct {
	Version     int           `json:"version"`
	GeneratedAt string        `json:"generated_at"`
	Deletions   []vdfDeletion `json:"deletions"`
}

type cleanupPreviewItem struct {
	VideoID  string `json:"video_id"`
	Platform string `json:"platform"`
	Title    string `json:"title"`
	Path     string `json:"path"`
	Status   string `json:"status"`
	Detail   string `json:"detail,omitempty"`
}

type cleanupPreview struct {
	Format           string               `json:"format,omitempty"`
	StorageAvailable bool                 `json:"storage_available"`
	Deletions        int                  `json:"deletions"`
	Candidates       int                  `json:"candidates"`
	AlreadyExcluded  int                  `json:"already_excluded"`
	Skipped          int                  `json:"skipped"`
	Items            []cleanupPreviewItem `json:"items"`
}

func normalizeMediaPath(root, p string) string {
	if strings.TrimSpace(p) == "" {
		return ""
	}
	if rel, err := mediaRelativePath(root, p); err == nil {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(filepath.Clean(p))
}

// locateMovedSibling detects a renamed-but-same-stem file beside the expected
// path, so a moved file is not mistaken for an external deletion.
func locateMovedSibling(path string) string {
	dir := filepath.Dir(path)
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == filepath.Base(path) {
			continue
		}
		if strings.TrimSuffix(name, filepath.Ext(name)) == stem {
			return filepath.Join(dir, name)
		}
	}
	return ""
}

// scanMediaLibrary is read-only: it never mutates the database and refuses to
// classify anything as missing while the media storage is unavailable.
func (a *App) scanMediaLibrary(limit int) mediaScanResult {
	cfg, _ := a.store.Settings()
	result := mediaScanResult{ScannedAt: store.Now(), Items: []mediaScanItem{}}
	if !storagePathAvailable(cfg.DownloadDir, cfg.StorageMountRequired) {
		result.StorageAvailable = false
		return result
	}
	result.StorageAvailable = true
	videos, err := a.store.CompletedVideos(limit, 0)
	if err != nil {
		return result
	}
	result.Total = len(videos)
	for _, v := range videos {
		path := mediaPathOf(v)
		item := mediaScanItem{VideoID: v.ID, Platform: v.Platform, Title: v.Title, Path: path}
		if path == "" {
			item.Status = scanStatusPathInvalid
			item.Detail = "数据库没有媒体路径"
			result.PathInvalid++
			result.Items = append(result.Items, item)
			continue
		}
		info, statErr := os.Stat(path)
		switch {
		case statErr == nil && info.Mode().IsRegular():
			result.Present++
		case statErr == nil:
			item.Status = scanStatusPathInvalid
			item.Detail = "路径不是普通文件"
			result.PathInvalid++
			result.Items = append(result.Items, item)
		case os.IsNotExist(statErr):
			if moved := locateMovedSibling(path); moved != "" {
				item.Status = scanStatusMoved
				item.Detail = "同目录发现同名文件: " + moved
				result.Moved++
			} else {
				item.Status = scanStatusMissing
				result.Missing++
			}
			result.Items = append(result.Items, item)
		case os.IsPermission(statErr):
			item.Status = scanStatusPermission
			item.Detail = statErr.Error()
			result.PermissionDenied++
			result.Items = append(result.Items, item)
		default:
			item.Status = scanStatusUnreadable
			item.Detail = statErr.Error()
			result.Unreadable++
			result.Items = append(result.Items, item)
		}
	}
	return result
}

func parseVDFDeletionList(raw []byte) ([]vdfDeletion, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, errors.New("删除清单为空")
	}
	if strings.HasPrefix(trimmed, "[") {
		var items []vdfDeletion
		if err := json.Unmarshal([]byte(trimmed), &items); err != nil {
			return nil, errors.New("删除清单 JSON 格式无效")
		}
		return items, nil
	}
	var list vdfList
	if err := json.Unmarshal([]byte(trimmed), &list); err != nil {
		return nil, errors.New("删除清单 JSON 格式无效")
	}
	return list.Deletions, nil
}

// previewMediaCleanup verifies each deletion against the database and the
// filesystem. It is a pure read: nothing is changed.
func (a *App) previewMediaCleanup(deletions []vdfDeletion) cleanupPreview {
	cfg, _ := a.store.Settings()
	preview := cleanupPreview{Format: importFormatLegacy, Deletions: len(deletions), Items: []cleanupPreviewItem{}}
	preview.StorageAvailable = storagePathAvailable(cfg.DownloadDir, cfg.StorageMountRequired)
	for _, d := range deletions {
		item := cleanupPreviewItem{VideoID: d.VideoID}
		v, err := a.store.Video(d.VideoID)
		if err != nil {
			item.Status = cleanupUnknownVideo
			item.Detail = "数据库中不存在该 video_id"
			preview.Skipped++
			preview.Items = append(preview.Items, item)
			continue
		}
		item.Platform, item.Title = v.Platform, v.Title
		path := mediaPathOf(v)
		item.Path = path
		switch {
		case a.store.IsMediaExcluded(v.ID):
			item.Status = cleanupAlreadyExcluded
		case v.Status == "removed":
			item.Status = cleanupAlreadyRemoved
		case v.Status == "queued" || v.Status == "submitting" || v.Status == "waiting" || v.Status == "active" || v.Status == "paused":
			item.Status = cleanupActive
			item.Detail = "任务处于 " + v.Status
		case v.Status != "completed":
			item.Status = cleanupNotCompleted
			item.Detail = "状态为 " + v.Status
		case path == "":
			item.Status = cleanupPathInvalid
		case d.OriginalPath != "" && normalizeMediaPath(cfg.DownloadDir, d.OriginalPath) != normalizeMediaPath(cfg.DownloadDir, path):
			item.Status = cleanupPathMismatch
			item.Detail = "清单路径与数据库不一致"
		default:
			if _, statErr := os.Stat(path); statErr == nil {
				item.Status = cleanupFilePresent
				item.Detail = "文件仍存在，拒绝排除"
			} else if os.IsNotExist(statErr) {
				item.Status = cleanupCandidate
			} else {
				item.Status = cleanupPathInvalid
				item.Detail = statErr.Error()
			}
		}
		if item.Status == cleanupCandidate {
			preview.Candidates++
		} else if item.Status != cleanupAlreadyExcluded && item.Status != cleanupAlreadyRemoved {
			preview.Skipped++
		}
		if item.Status == cleanupAlreadyExcluded {
			preview.AlreadyExcluded++
		}
		preview.Items = append(preview.Items, item)
	}
	return preview
}

type cleanupApplyResult struct {
	Executed   int                  `json:"executed"`
	Skipped    int                  `json:"skipped"`
	Failed     int                  `json:"failed"`
	Candidates int                  `json:"candidates"`
	Items      []cleanupPreviewItem `json:"items"`
}

// applyMediaCleanup re-verifies every entry and only then atomically writes the
// exclusion + removal per video. A single failure never affects other entries
// (each is its own transaction).
func (a *App) applyMediaCleanup(deletions []vdfDeletion, confirm bool) (cleanupApplyResult, error) {
	preview := a.previewMediaCleanup(deletions)
	apply := cleanupApplyResult{Candidates: preview.Candidates, Items: preview.Items}
	if !confirm {
		return apply, nil
	}
	cfg, _ := a.store.Settings()
	if !storagePathAvailable(cfg.DownloadDir, cfg.StorageMountRequired) {
		return apply, errors.New("媒体存储当前不可用，已取消清理")
	}
	for _, item := range preview.Items {
		if item.Status != cleanupCandidate {
			apply.Skipped++
			continue
		}
		v, err := a.store.Video(item.VideoID)
		if err != nil {
			apply.Failed++
			continue
		}
		// Re-check under the apply path so a file that reappeared or a task that
		// became active between preview and apply is not touched.
		if v.Status != "completed" || a.store.IsMediaExcluded(v.ID) {
			apply.Skipped++
			continue
		}
		path := mediaPathOf(v)
		if path == "" {
			apply.Failed++
			continue
		}
		if _, statErr := os.Stat(path); statErr == nil {
			apply.Skipped++
			continue
		} else if !os.IsNotExist(statErr) {
			apply.Failed++
			continue
		}
		if err := a.store.RemoveAndExclude(v.ID, store.MediaExclusion{
			VideoID:             v.ID,
			Source:              v.Platform,
			Reason:              "vdf_duplicate",
			TitleSnapshot:       v.Title,
			OriginalURLSnapshot: v.PageURL,
			ThumbnailSnapshot:   v.ThumbnailURL,
		}); err != nil {
			apply.Failed++
			_ = a.store.AddAudit("admin", "media_integrity_cleanup", "video", v.ID, "failed", map[string]any{"reason": err.Error()})
			continue
		}
		apply.Executed++
		_ = a.store.AddAudit("admin", "media_integrity_cleanup", "video", v.ID, "success", map[string]any{"reason": "vdf_duplicate"})
	}
	return apply, nil
}

func (a *App) mediaIntegrityScan(w http.ResponseWriter, r *http.Request) {
	limit := intParam(r, "limit", 100000, 1, 1000000)
	respond(w, http.StatusOK, a.scanMediaLibrary(limit))
}

// mediaIntegrityImport detects the deletion JSON schema (external deleted list
// or native VDF GUI export) and returns the matching preview. A pending
// TransRunner plan is never treated as an already-deleted list.
func (a *App) mediaIntegrityImport(w http.ResponseWriter, r *http.Request) {
	raw := make([]byte, 0, 4096)
	buf := make([]byte, 4096)
	for {
		n, err := r.Body.Read(buf)
		if n > 0 {
			raw = append(raw, buf[:n]...)
			if len(raw) > 8<<20 {
				fail(w, http.StatusRequestEntityTooLarge, errors.New("删除清单过大"))
				return
			}
		}
		if err != nil {
			break
		}
	}
	switch detectImportFormat(raw) {
	case importFormatVDFGUI:
		export, err := parseVDFGUIExport(raw)
		if err != nil {
			fail(w, http.StatusUnprocessableEntity, err)
			return
		}
		respond(w, http.StatusOK, a.previewVDFGUI(export))
	case importFormatTransRunner:
		fail(w, http.StatusUnprocessableEntity, errors.New("这是 TransRunner 删除计划，请使用“模式二”导入"))
	case importFormatLegacy:
		deletions, err := parseVDFDeletionList(raw)
		if err != nil {
			fail(w, http.StatusUnprocessableEntity, err)
			return
		}
		preview := a.previewMediaCleanup(deletions)
		preview.Format = importFormatLegacy
		respond(w, http.StatusOK, preview)
	default:
		fail(w, http.StatusUnprocessableEntity, errors.New("无法识别的删除清单格式"))
	}
}

func (a *App) mediaIntegrityApply(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Deletions []vdfDeletion `json:"deletions"`
		Raw       string        `json:"raw"`
		Confirm   bool          `json:"confirm"`
	}
	if err := decode(r, &input); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	deletions := input.Deletions
	if len(deletions) == 0 && strings.TrimSpace(input.Raw) != "" {
		parsed, err := parseVDFDeletionList([]byte(input.Raw))
		if err != nil {
			fail(w, http.StatusUnprocessableEntity, err)
			return
		}
		deletions = parsed
	}
	if len(deletions) == 0 {
		fail(w, http.StatusUnprocessableEntity, errors.New("没有可处理的删除记录"))
		return
	}
	result, err := a.applyMediaCleanup(deletions, input.Confirm)
	if err != nil {
		fail(w, http.StatusServiceUnavailable, err)
		return
	}
	result.Items = nil
	respond(w, http.StatusOK, map[string]any{
		"dry_run":    !input.Confirm,
		"candidates": result.Candidates,
		"executed":   result.Executed,
		"skipped":    result.Skipped,
		"failed":     result.Failed,
	})
}
