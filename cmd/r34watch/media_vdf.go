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

// matchVideoPath resolves a VDF path to a single managed media row and reports
// how it matched. A basename-only match is returned distinctly so callers can
// refuse to act on it.
func matchVideoPath(idx mediaPathIndex, cfg store.Settings, rawPath string) (store.Video, string, string) {
	norm := normalizeMediaPath(cfg.DownloadDir, filepath.FromSlash(rawPath))
	if norm == "" {
		return store.Video{}, "unmatched", ""
	}
	if v, ok := idx.byNorm[norm]; ok {
		return v, "matched", "exact"
	}
	tail := strings.ToLower(strings.Join(lastComponents(norm, 2), "/"))
	if vs := idx.byTail[tail]; len(vs) == 1 {
		return vs[0], "matched", "suffix"
	} else if len(vs) > 1 {
		return store.Video{}, "ambiguous", ""
	}
	base := strings.ToLower(filepath.Base(norm))
	if vs := idx.byBase[base]; len(vs) == 1 {
		return vs[0], "basename_only", "basename"
	} else if len(vs) > 1 {
		return store.Video{}, "ambiguous", ""
	}
	return store.Video{}, "unmatched", ""
}

type vdfItemView struct {
	Path             string   `json:"path"`
	SizeBytes        int64    `json:"size_bytes"`
	Resolution       string   `json:"resolution,omitempty"`
	MatchStatus      string   `json:"match_status"`
	VideoID          string   `json:"video_id,omitempty"`
	SubscriptionSlug string   `json:"subscription_slug,omitempty"`
	DBPath           string   `json:"db_path,omitempty"`
	FileSize         int64    `json:"file_size,omitempty"`
	Detail           string   `json:"detail,omitempty"`
	Groups           []string `json:"groups,omitempty"`
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

type vdfConflict struct {
	Kind    string   `json:"kind"`
	VideoID string   `json:"video_id,omitempty"`
	Path    string   `json:"path,omitempty"`
	Groups  []string `json:"groups"`
	Detail  string   `json:"detail"`
}

type vdfPreview struct {
	Format              string         `json:"format"`
	StorageAvailable    bool           `json:"storage_available"`
	HasConflicts        bool           `json:"has_conflicts"`
	Groups              int            `json:"groups"`
	ReadyGroups         int            `json:"ready_groups"`
	AlreadyDoneGroups   int            `json:"already_done_groups"`
	ConflictGroups      int            `json:"conflict_groups"`
	RejectedGroups      int            `json:"rejected_groups"`
	RemoveTotal         int            `json:"remove_total"`
	RemoveMatched       int            `json:"remove_matched"`
	RemoveAlreadyDone   int            `json:"remove_already_done"`
	RemoveUnmatched     int            `json:"remove_unmatched"`
	KeepTotal           int            `json:"keep_total"`
	KeepMatched         int            `json:"keep_matched"`
	KeepMissing         int            `json:"keep_missing"`
	CrossGroupConflicts int            `json:"cross_group_conflicts"`
	EstimatedFreed      int64          `json:"estimated_freed"`
	Conflicts           []vdfConflict  `json:"conflicts"`
	Items               []vdfGroupView `json:"items"`
}

const (
	vdfStatusReady       = "ready"
	vdfStatusAlreadyDone = "already_done"
	vdfStatusConflict    = "conflict"
	vdfStatusRejected    = "rejected"
)

func (a *App) verifyVDFItem(item vdfGUIItem, idx mediaPathIndex, cfg store.Settings, allowMissing bool) vdfItemView {
	view := vdfItemView{Path: item.Path, SizeBytes: item.SizeBytes, Resolution: item.Resolution}
	v, status, _ := matchVideoPath(idx, cfg, item.Path)
	if status != "matched" {
		view.MatchStatus = status
		switch status {
		case "ambiguous":
			view.Detail = "同名/同后缀存在多个媒体记录，无法唯一匹配"
		case "basename_only":
			view.MatchStatus = "basename_only"
			view.Detail = "仅文件名匹配（无路径后缀证据），需人工确认"
		default:
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

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func pathSuffixKey(cfg store.Settings, raw string) string {
	norm := normalizeMediaPath(cfg.DownloadDir, filepath.FromSlash(raw))
	if norm == "" {
		return ""
	}
	return strings.ToLower(strings.Join(lastComponents(norm, 2), "/"))
}

// alreadyDoneRemoveVideo returns the canonical video_id only when there is
// reliable evidence a remove target was previously applied: a db_excluded
// deletion_plan_items row with the same two-component path suffix whose video is
// now removed and has a media_exclusions tombstone. Filename-only or
// file-absence-only evidence is never accepted.
func (a *App) alreadyDoneRemoveVideo(cfg store.Settings, excluded []store.DeletionPlanItem, rawPath string) (string, bool) {
	key := pathSuffixKey(cfg, rawPath)
	if key == "" {
		return "", false
	}
	found := ""
	for _, it := range excluded {
		if pathSuffixKey(cfg, it.OriginalPath) != key {
			continue
		}
		v, err := a.store.Video(it.VideoID)
		if err != nil || v.Status != "removed" || !a.store.IsMediaExcluded(it.VideoID) {
			continue
		}
		if found != "" && found != it.VideoID {
			return "", false // ambiguous evidence
		}
		found = it.VideoID
	}
	return found, found != ""
}

// previewVDFGUI performs a full, plan-global safety review: it never mutates the
// database or filesystem, and it blocks the whole plan when any cross-group
// identity conflict exists.
func (a *App) previewVDFGUI(export vdfGUIExport) vdfPreview {
	cfg, _ := a.store.Settings()
	p := vdfPreview{Format: importFormatVDFGUI, Items: []vdfGroupView{}, Conflicts: []vdfConflict{}}
	p.StorageAvailable = storagePathAvailable(cfg.DownloadDir, cfg.StorageMountRequired)
	idx := a.buildMediaPathIndex(cfg)
	excluded, _ := a.store.DeletionPlanItemsByStatus(itemDBExcluded)
	p.Groups = len(export.Groups)

	views := make([]vdfGroupView, 0, len(export.Groups))
	rejectGroup := map[string]bool{}

	removePathGroups := map[string][]string{}
	keepPathGroups := map[string][]string{}
	removeVideoGroups := map[string][]string{}
	keepVideoGroups := map[string][]string{}
	removePathVideo := map[string]string{}
	pathVideoIDs := map[string]map[string]bool{}
	type spHit struct{ gid, vid, path string }
	spHits := []spHit{}

	for _, group := range export.Groups {
		view := vdfGroupView{GroupID: group.GroupID, Reason: group.Reason, Status: vdfStatusReady, Problems: []string{}, RemoveItems: []vdfItemView{}, KeepItems: []vdfItemView{}}
		slugs := map[string]bool{}
		removeIDs := map[string]bool{}
		keepIDs := map[string]bool{}
		hasReject := false

		for _, item := range group.RemoveItems {
			iv := a.verifyVDFItem(item, idx, cfg, true)
			p.RemoveTotal++
			key := pathSuffixKey(cfg, item.Path)
			switch iv.MatchStatus {
			case "matched", "absent":
				p.RemoveMatched++
				if iv.MatchStatus == "matched" {
					p.EstimatedFreed += iv.FileSize
					view.EstimatedFreed += iv.FileSize
					if shared, _ := a.store.CountVideosWithPath(iv.DBPath, iv.VideoID); shared > 0 {
						spHits = append(spHits, spHit{group.GroupID, iv.VideoID, iv.DBPath})
					}
				}
				slugs[iv.SubscriptionSlug] = true
				removeIDs[iv.VideoID] = true
				removeVideoGroups[iv.VideoID] = append(removeVideoGroups[iv.VideoID], group.GroupID)
				if key != "" {
					removePathGroups[key] = append(removePathGroups[key], group.GroupID)
					removePathVideo[key] = iv.VideoID
					if pathVideoIDs[key] == nil {
						pathVideoIDs[key] = map[string]bool{}
					}
					pathVideoIDs[key][iv.VideoID] = true
				}
			default:
				if vid, ok := a.alreadyDoneRemoveVideo(cfg, excluded, item.Path); ok {
					iv.MatchStatus = "already_done"
					iv.VideoID = vid
					iv.Detail = "此前已成功清理并加入排除"
					p.RemoveAlreadyDone++
					removeVideoGroups[vid] = append(removeVideoGroups[vid], group.GroupID)
					if v, e := a.store.Video(vid); e == nil {
						iv.SubscriptionSlug = v.SubscriptionSlug
						slugs[iv.SubscriptionSlug] = true
					}
				} else {
					p.RemoveUnmatched++
					hasReject = true
					view.Problems = append(view.Problems, "删除项无法安全匹配: "+item.Path+" ("+iv.MatchStatus+")")
				}
			}
			view.RemoveItems = append(view.RemoveItems, iv)
		}

		for _, item := range group.KeepItems {
			iv := a.verifyVDFItem(item, idx, cfg, false)
			p.KeepTotal++
			key := pathSuffixKey(cfg, item.Path)
			if iv.MatchStatus == "matched" {
				p.KeepMatched++
				slugs[iv.SubscriptionSlug] = true
				keepIDs[iv.VideoID] = true
				keepVideoGroups[iv.VideoID] = append(keepVideoGroups[iv.VideoID], group.GroupID)
				if key != "" {
					keepPathGroups[key] = append(keepPathGroups[key], group.GroupID)
				}
			} else {
				p.KeepMissing++
				hasReject = true
				if vid, ok := a.alreadyDoneRemoveVideo(cfg, excluded, item.Path); ok {
					iv.MatchStatus = "already_removed"
					iv.VideoID = vid
					iv.Detail = "保留项对应媒体已被此前清理删除"
					keepVideoGroups[vid] = append(keepVideoGroups[vid], group.GroupID)
					if key != "" {
						keepPathGroups[key] = append(keepPathGroups[key], group.GroupID)
					}
				}
				view.Problems = append(view.Problems, "保留项无法安全匹配: "+item.Path+" ("+iv.MatchStatus+")")
			}
			view.KeepItems = append(view.KeepItems, iv)
		}

		if len(group.KeepItems) == 0 {
			view.Problems = append(view.Problems, "VDF 组缺少 KeepItems")
			hasReject = true
		}
		if len(group.RemoveItems) == 0 {
			view.Problems = append(view.Problems, "VDF 组缺少 RemoveItems")
			hasReject = true
		}
		if len(slugs) > 1 {
			view.Problems = append(view.Problems, "删除项与保留项跨订阅，拒绝执行")
			hasReject = true
		}
		for id := range removeIDs {
			if keepIDs[id] {
				view.Problems = append(view.Problems, "删除项与保留项指向同一 HWatch 媒体")
				hasReject = true
			}
		}
		if len(slugs) == 1 {
			for s := range slugs {
				view.SubscriptionSlug = s
			}
		}
		if !p.StorageAvailable {
			view.Problems = append(view.Problems, "媒体存储当前不可用")
			hasReject = true
		}
		if hasReject {
			rejectGroup[group.GroupID] = true
		}
		views = append(views, view)
	}

	conflictGroups := map[string]bool{}
	addConflict := func(kind, videoID, path string, groups []string, detail string) {
		groups = uniqueStrings(groups)
		p.Conflicts = append(p.Conflicts, vdfConflict{Kind: kind, VideoID: videoID, Path: path, Groups: groups, Detail: detail})
		for _, gid := range groups {
			conflictGroups[gid] = true
		}
	}
	for vid, rg := range removeVideoGroups {
		if kg, ok := keepVideoGroups[vid]; ok {
			addConflict("keep_remove_video", vid, "", append(append([]string{}, rg...), kg...), "同一媒体既在 RemoveItems 又在 KeepItems")
		}
	}
	for key, rg := range removePathGroups {
		if kg, ok := keepPathGroups[key]; ok {
			addConflict("keep_remove_path", removePathVideo[key], key, append(append([]string{}, rg...), kg...), "同一路径既在 RemoveItems 又在 KeepItems")
		}
	}
	for vid, groups := range removeVideoGroups {
		if len(uniqueStrings(groups)) > 1 {
			addConflict("duplicate_remove", vid, "", groups, "同一媒体在多个组中被列为 RemoveItems")
		}
	}
	for key, groups := range removePathGroups {
		if len(uniqueStrings(groups)) > 1 {
			addConflict("duplicate_remove_path", removePathVideo[key], key, groups, "同一路径在多个组中被列为 RemoveItems")
		}
	}
	for key, set := range pathVideoIDs {
		if len(set) > 1 {
			addConflict("shared_path", "", key, removePathGroups[key], "不同 video_id 映射同一个物理媒体路径")
		}
	}
	for _, h := range spHits {
		addConflict("shared_path", h.vid, h.path, []string{h.gid}, "该物理文件被多条媒体记录引用")
	}
	for _, view := range views {
		for _, it := range view.KeepItems {
			switch it.MatchStatus {
			case "already_removed":
				addConflict("keep_already_removed", it.VideoID, it.Path, []string{view.GroupID}, "保留项对应媒体已被清理")
			case "matched":
				// ok
			default:
				addConflict("keep_missing", it.VideoID, it.Path, []string{view.GroupID}, "保留项文件缺失、不可访问或无法唯一匹配")
			}
		}
	}

	for i := range views {
		v := &views[i]
		switch {
		case conflictGroups[v.GroupID]:
			v.Status = vdfStatusConflict
			p.ConflictGroups++
		case rejectGroup[v.GroupID]:
			v.Status = vdfStatusRejected
			p.RejectedGroups++
		default:
			allDone := true
			for _, it := range v.RemoveItems {
				if it.MatchStatus != "already_done" {
					allDone = false
					break
				}
			}
			if allDone && p.RemoveAlreadyDone > 0 {
				v.Status = vdfStatusAlreadyDone
				p.AlreadyDoneGroups++
			} else {
				v.Status = vdfStatusReady
				p.ReadyGroups++
			}
		}
		p.Items = append(p.Items, *v)
	}
	p.HasConflicts = len(p.Conflicts) > 0
	p.CrossGroupConflicts = len(p.Conflicts)
	return p
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
	// Backend hard block: never execute a plan with any cross-group identity
	// conflict (re-checked here, not only in the UI preview).
	if preview.HasConflicts {
		return result, fmt.Errorf("VDF 计划存在跨组身份冲突（%d 处），已阻止整个计划的执行，未做任何文件或数据库操作；请人工核对", preview.CrossGroupConflicts)
	}
	if !preview.StorageAvailable {
		return result, fmt.Errorf("媒体存储当前不可用，已取消执行")
	}
	if err := a.store.UpsertDeletionPlan(store.DeletionPlan{PlanID: planIDValue, Kind: vdfGUIPlanKind, PlanVersion: 1, SubscriptionSlug: "", ScanID: planIDValue, Status: "pending"}); err != nil {
		return result, err
	}
	executed, skipped, failed, recovering := 0, 0, 0, 0
	for _, group := range preview.Items {
		if group.Status != vdfStatusReady {
			skipped++
			continue
		}
		for _, item := range group.RemoveItems {
			if item.MatchStatus == "already_done" {
				skipped++
				continue
			}
			if item.VideoID == "" || (item.MatchStatus != "matched" && item.MatchStatus != "absent") {
				skipped++
				continue
			}
			v, err := a.store.Video(item.VideoID)
			if err != nil {
				failed++
				continue
			}
			persist := store.DeletionPlanItem{PlanID: planIDValue, GroupID: group.GroupID, VideoID: v.ID, KeptVideoID: firstKeepID(group), ExpectedSize: item.SizeBytes, OriginalPath: item.DBPath, Status: itemPending}
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
