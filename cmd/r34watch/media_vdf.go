package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"r34watch/internal/store"
)

// Native VDF GUI duplicate-deletion export support.
//
// VDF GUI emits Groups of RemoveItems/KeepItems identified only by path, with
// no video_id and no content hash. HWatch therefore maps each path to a managed
// media row (server path, then two-component suffix, then unique basename),
// refuses ambiguous matches, and only ever executes items that pass the same
// quarantine + transactional exclusion path used by the TransRunner plan mode.

const vdfGUIPlanKind = "vdf-gui-deletion"

const (
	importFormatLegacy      = "legacy_deletion_list"
	importFormatTransRunner = "transrunner_plan"
	importFormatVDFGUI      = "vdf_gui"
	importFormatUnknown     = "unknown"
)

type vdfGUIItem struct {
	Path        string `json:"Path"`
	SizeBytes   int64  `json:"SizeBytes"`
	Resolution  string `json:"Resolution"`
	DateCreated string `json:"DateCreated"`
}

type vdfGUIGroup struct {
	GroupID     string       `json:"GroupId"`
	Reason      string       `json:"Reason"`
	RemoveItems []vdfGUIItem `json:"RemoveItems"`
	KeepItems   []vdfGUIItem `json:"KeepItems"`
}

type vdfGUIExport struct {
	CreatedAt                  string        `json:"CreatedAt"`
	EstimatedTotalSavingsBytes int64         `json:"EstimatedTotalSavingsBytes"`
	Groups                     []vdfGUIGroup `json:"Groups"`
}

// detectImportFormat chooses the parser by explicit schema markers so a pending
// TransRunner plan is never treated as an already-deleted list.
func detectImportFormat(raw []byte) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return importFormatUnknown
	}
	if strings.HasPrefix(trimmed, "[") {
		return importFormatLegacy
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return importFormatUnknown
	}
	if _, ok := probe["kind"]; ok {
		return importFormatTransRunner
	}
	if _, ok := probe["Groups"]; ok {
		return importFormatVDFGUI
	}
	if _, ok := probe["groups"]; ok {
		return importFormatVDFGUI
	}
	if _, ok := probe["deletions"]; ok {
		return importFormatLegacy
	}
	return importFormatUnknown
}

func parseVDFGUIExport(raw []byte) (vdfGUIExport, error) {
	var export vdfGUIExport
	if err := json.Unmarshal(raw, &export); err != nil {
		return export, err
	}
	if len(export.Groups) == 0 {
		// tolerate lowercase "groups"
		var alt struct {
			Groups []vdfGUIGroup `json:"groups"`
		}
		if err := json.Unmarshal(raw, &alt); err == nil {
			export.Groups = alt.Groups
		}
	}
	if len(export.Groups) == 0 {
		return export, fmt.Errorf("VDF 导出没有 Groups")
	}
	return export, nil
}

type mediaPathIndex struct {
	byNorm map[string]store.Video
	byBase map[string][]store.Video
	byTail map[string][]store.Video
}

func appendUniqueVideo(list []store.Video, v store.Video) []store.Video {
	for _, existing := range list {
		if existing.ID == v.ID {
			return list
		}
	}
	return append(list, v)
}

func lastComponents(rel string, n int) []string {
	parts := strings.Split(strings.Trim(rel, "/"), "/")
	if len(parts) <= n {
		return parts
	}
	return parts[len(parts)-n:]
}

func (a *App) buildMediaPathIndex(cfg store.Settings) mediaPathIndex {
	idx := mediaPathIndex{byNorm: map[string]store.Video{}, byBase: map[string][]store.Video{}, byTail: map[string][]store.Video{}}
	videos, _ := a.store.CompletedVideos(1000000, 0)
	for _, v := range videos {
		for _, p := range []string{v.PlexPath, v.FilePath} {
			if strings.TrimSpace(p) == "" {
				continue
			}
			norm := normalizeMediaPath(cfg.DownloadDir, p)
			if norm == "" {
				continue
			}
			if _, ok := idx.byNorm[norm]; !ok {
				idx.byNorm[norm] = v
			}
			base := strings.ToLower(filepath.Base(p))
			idx.byBase[base] = appendUniqueVideo(idx.byBase[base], v)
			tail := strings.ToLower(strings.Join(lastComponents(norm, 2), "/"))
			idx.byTail[tail] = appendUniqueVideo(idx.byTail[tail], v)
		}
	}
	return idx
}

// matchVideoPath resolves a VDF path to a single managed media row. It tolerates
// a macOS path prefix via a two-component suffix and a unique basename, but
// never accepts an ambiguous result.
func matchVideoPath(idx mediaPathIndex, cfg store.Settings, rawPath string) (store.Video, string) {
	norm := normalizeMediaPath(cfg.DownloadDir, filepath.FromSlash(rawPath))
	if norm == "" {
		return store.Video{}, "unmatched"
	}
	if v, ok := idx.byNorm[norm]; ok {
		return v, "matched"
	}
	tail := strings.ToLower(strings.Join(lastComponents(norm, 2), "/"))
	if vs := idx.byTail[tail]; len(vs) == 1 {
		return vs[0], "matched"
	} else if len(vs) > 1 {
		return store.Video{}, "ambiguous"
	}
	base := strings.ToLower(filepath.Base(norm))
	if vs := idx.byBase[base]; len(vs) == 1 {
		return vs[0], "matched"
	} else if len(vs) > 1 {
		return store.Video{}, "ambiguous"
	}
	return store.Video{}, "unmatched"
}

type vdfItemView struct {
	Path             string `json:"path"`
	SizeBytes        int64  `json:"size_bytes"`
	Resolution       string `json:"resolution,omitempty"`
	MatchStatus      string `json:"match_status"`
	VideoID          string `json:"video_id,omitempty"`
	SubscriptionSlug string `json:"subscription_slug,omitempty"`
	DBPath           string `json:"db_path,omitempty"`
	FileSize         int64  `json:"file_size,omitempty"`
	Detail           string `json:"detail,omitempty"`
}

type vdfGroupView struct {
	GroupID          string        `json:"group_id"`
	Reason           string        `json:"reason"`
	Status           string        `json:"status"`
	SubscriptionSlug string        `json:"subscription_slug"`
	Problems         []string      `json:"problems"`
	RemoveItems      []vdfItemView `json:"remove_items"`
	KeepItems        []vdfItemView `json:"keep_items"`
	EstimatedFreed   int64         `json:"estimated_freed"`
}

type vdfPreview struct {
	Format           string         `json:"format"`
	StorageAvailable bool           `json:"storage_available"`
	Groups           int            `json:"groups"`
	ReadyGroups      int            `json:"ready_groups"`
	RejectedGroups   int            `json:"rejected_groups"`
	RemoveItems      int            `json:"remove_items"`
	MatchedItems     int            `json:"matched_items"`
	EstimatedFreed   int64          `json:"estimated_freed"`
	Items            []vdfGroupView `json:"items"`
}

func (a *App) verifyVDFItem(item vdfGUIItem, idx mediaPathIndex, cfg store.Settings, allowMissing bool) vdfItemView {
	view := vdfItemView{Path: item.Path, SizeBytes: item.SizeBytes, Resolution: item.Resolution}
	v, status := matchVideoPath(idx, cfg, item.Path)
	if status != "matched" {
		view.MatchStatus = status
		if status == "ambiguous" {
			view.Detail = "同名/同后缀存在多个媒体记录，无法唯一匹配"
		} else {
			view.Detail = "未在已完成的 HWatch 媒体中找到"
		}
		return view
	}
	view.VideoID = v.ID
	view.SubscriptionSlug = v.SubscriptionSlug
	view.DBPath = mediaPathOf(v)
	view.MatchStatus = "matched"
	if v.Status != "completed" {
		view.MatchStatus = "not_completed"
		view.Detail = "状态为 " + v.Status
		return view
	}
	info, err := os.Stat(view.DBPath)
	if err != nil {
		if allowMissing && os.IsNotExist(err) {
			// The remove file was already deleted externally; still a valid
			// exclude-only candidate (no quarantine needed).
			view.MatchStatus = "absent"
			view.Detail = "文件已不在磁盘（外部删除）"
			return view
		}
		view.MatchStatus = "missing_file"
		view.Detail = "文件不存在或不可访问"
		return view
	}
	view.FileSize = info.Size()
	if item.SizeBytes > 0 && info.Size() != item.SizeBytes {
		view.MatchStatus = "size_mismatch"
		view.Detail = "文件大小与 VDF 记录不一致"
	}
	return view
}

func (a *App) previewVDFGUI(export vdfGUIExport) vdfPreview {
	cfg, _ := a.store.Settings()
	preview := vdfPreview{Format: importFormatVDFGUI, Items: []vdfGroupView{}}
	preview.StorageAvailable = storagePathAvailable(cfg.DownloadDir, cfg.StorageMountRequired)
	idx := a.buildMediaPathIndex(cfg)
	preview.Groups = len(export.Groups)
	for _, group := range export.Groups {
		view := vdfGroupView{GroupID: group.GroupID, Reason: group.Reason, Status: "ready", Problems: []string{}, RemoveItems: []vdfItemView{}, KeepItems: []vdfItemView{}}
		slugs := map[string]bool{}
		removeIDs := map[string]bool{}
		keepIDs := map[string]bool{}
		for _, item := range group.RemoveItems {
			iv := a.verifyVDFItem(item, idx, cfg, true)
			view.RemoveItems = append(view.RemoveItems, iv)
			preview.RemoveItems++
			if iv.MatchStatus == "matched" || iv.MatchStatus == "absent" {
				preview.MatchedItems++
				if iv.MatchStatus == "matched" {
					preview.EstimatedFreed += iv.FileSize
					view.EstimatedFreed += iv.FileSize
				}
				slugs[iv.SubscriptionSlug] = true
				removeIDs[iv.VideoID] = true
			} else {
				view.Problems = append(view.Problems, "删除项无法安全匹配: "+item.Path+" ("+iv.MatchStatus+")")
			}
		}
		for _, item := range group.KeepItems {
			iv := a.verifyVDFItem(item, idx, cfg, false)
			view.KeepItems = append(view.KeepItems, iv)
			if iv.MatchStatus == "matched" {
				preview.MatchedItems++
				slugs[iv.SubscriptionSlug] = true
				keepIDs[iv.VideoID] = true
			} else {
				view.Problems = append(view.Problems, "保留项无法安全匹配: "+item.Path+" ("+iv.MatchStatus+")")
			}
		}
		if len(group.KeepItems) == 0 {
			view.Problems = append(view.Problems, "VDF 组缺少 KeepItems")
		}
		if len(group.RemoveItems) == 0 {
			view.Problems = append(view.Problems, "VDF 组缺少 RemoveItems")
		}
		if len(slugs) > 1 {
			view.Problems = append(view.Problems, "删除项与保留项跨订阅，拒绝执行")
		}
		for id := range removeIDs {
			if keepIDs[id] {
				view.Problems = append(view.Problems, "删除项与保留项指向同一 HWatch 媒体")
			}
		}
		if len(slugs) == 1 {
			for s := range slugs {
				view.SubscriptionSlug = s
			}
		}
		if !preview.StorageAvailable {
			view.Problems = append(view.Problems, "媒体存储当前不可用")
		}
		if len(view.Problems) > 0 {
			view.Status = "rejected"
			preview.RejectedGroups++
		} else {
			preview.ReadyGroups++
		}
		preview.Items = append(preview.Items, view)
	}
	return preview
}

// executeVDFGUIGroups runs the ready groups through quarantine + transactional
// exclusion. confirm=false is a pure Dry Run.
func (a *App) executeVDFGUIGroups(export vdfGUIExport, confirm bool) (planExecuteResult, error) {
	preview := a.previewVDFGUI(export)
	result := planExecuteResult{DryRun: !confirm, EstimatedFreed: preview.EstimatedFreed, Items: []planItemView{}}
	planIDValue := vdfPlanID(export)
	result.PlanID = planIDValue
	if !confirm {
		return result, nil
	}
	if !preview.StorageAvailable {
		return result, fmt.Errorf("媒体存储当前不可用，已取消执行")
	}
	if err := a.store.UpsertDeletionPlan(store.DeletionPlan{PlanID: planIDValue, Kind: vdfGUIPlanKind, PlanVersion: 1, SubscriptionSlug: "", ScanID: planIDValue, Status: "pending"}); err != nil {
		return result, err
	}
	executed, skipped, failed, recovering := 0, 0, 0, 0
	for _, group := range preview.Items {
		if group.Status != "ready" {
			skipped++
			continue
		}
		for _, item := range group.RemoveItems {
			if item.VideoID == "" || (item.MatchStatus != "matched" && item.MatchStatus != "absent") {
				skipped++
				continue
			}
			v, err := a.store.Video(item.VideoID)
			if err != nil {
				failed++
				continue
			}
			persist := store.DeletionPlanItem{PlanID: planIDValue, VideoID: v.ID, KeptVideoID: firstKeepID(group), ExpectedSize: item.SizeBytes, OriginalPath: item.DBPath, Status: itemPending}
			_ = a.store.SaveDeletionPlanItem(persist)
			var status, qpath, detail string
			if item.MatchStatus == "absent" {
				// Already deleted externally: exclude only, nothing to quarantine.
				if err := a.excludeWithReason(v, "vdf_gui_duplicate"); err != nil {
					status, detail = itemFailed, "数据库更新失败: "+err.Error()
				} else {
					status, detail = itemDBExcluded, "文件已不存在，仅写入排除"
				}
			} else {
				status, qpath, detail = a.quarantineExcludeOne(planIDValue, "vdf_gui_duplicate", v, item.DBPath)
			}
			persist.Status, persist.QuarantinePath, persist.Reason = status, qpath, detail
			_ = a.store.SaveDeletionPlanItem(persist)
			switch status {
			case itemDBExcluded:
				executed++
				_ = a.store.AddAudit("admin", "vdf_gui_cleanup", "video", v.ID, "success", map[string]any{"plan_id": planIDValue})
			case itemFailed:
				failed++
			case itemNeedsRecovery:
				recovering++
			default:
				skipped++
			}
		}
	}
	result.Executed, result.Skipped, result.Failed, result.NeedsRecovery = executed, skipped, failed, recovering
	_ = a.store.SetDeletionPlanStatus(planIDValue, "completed")
	return result, nil
}

func firstKeepID(group vdfGroupView) string {
	for _, k := range group.KeepItems {
		if k.VideoID != "" {
			return k.VideoID
		}
	}
	return ""
}

func vdfPlanID(export vdfGUIExport) string {
	seed := export.CreatedAt + "|" + fmt.Sprintf("%d", export.EstimatedTotalSavingsBytes)
	for _, g := range export.Groups {
		seed += "|" + g.GroupID
	}
	sum := sha256.Sum256([]byte(seed))
	return "vdf-gui-" + fmt.Sprintf("%x", sum[:8])
}

// quarantineExcludeOne moves a single media file into the quarantine area and
// then excludes it. The filesystem move and the DB write are intentionally
// separate; a DB failure leaves a recoverable quarantined state.
func (a *App) quarantineExcludeOne(planID, reason string, v store.Video, originalPath string) (status, quarantinePath, detail string) {
	qPath := quarantineTarget(a.mediaRootForQuarantine(), planID, originalPath)
	_, origErr := os.Stat(originalPath)
	origExists := origErr == nil
	_, qErr := os.Stat(qPath)
	qExists := qErr == nil
	switch {
	case !origExists && qExists:
	case origExists && !qExists:
		if err := os.MkdirAll(filepath.Dir(qPath), 0o755); err != nil {
			return itemFailed, "", "无法创建隔离目录: " + err.Error()
		}
		if err := moveFile(originalPath, qPath); err != nil {
			return itemFailed, "", "隔离文件失败: " + err.Error()
		}
	default:
		return itemNeedsRecovery, qPath, "原文件与隔离文件同时存在或同时缺失，需人工核对"
	}
	if err := a.excludeWithReason(v, reason); err != nil {
		return itemNeedsRecovery, qPath, "已隔离但数据库更新失败: " + err.Error()
	}
	return itemDBExcluded, qPath, ""
}

func (a *App) mediaRootForQuarantine() string {
	cfg, _ := a.store.Settings()
	return cfg.DownloadDir
}

func (a *App) mediaIntegrityVDFExecute(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Raw     string `json:"raw"`
		Confirm bool   `json:"confirm"`
	}
	if err := decode(r, &input); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	export, err := parseVDFGUIExport([]byte(input.Raw))
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	result, err := a.executeVDFGUIGroups(export, input.Confirm)
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	result.Items = nil
	respond(w, http.StatusOK, result)
}
