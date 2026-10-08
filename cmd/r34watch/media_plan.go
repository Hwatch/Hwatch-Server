package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"r34watch/internal/dedupe"
	"r34watch/internal/store"
)

// TransRunner Phase 1.2 duplicate-deletion plan execution.
//
// A plan is a request, never proof: every entry is re-verified server-side
// against the database and the filesystem before anything is touched. Physical
// removal is implemented as a move into a controlled quarantine directory on the
// same filesystem, and execution state is persisted so a crash can be
// reconciled.

const transRunnerPlanKind = "transrunner-duplicate-deletion-plan"
const transRunnerPlanVersion = 1

const (
	itemPending         = "pending"
	itemVerified        = "verified"
	itemRejected        = "rejected"
	itemAlreadyExcluded = "already_excluded"
	itemQuarantined     = "quarantined"
	itemDBExcluded      = "db_excluded"
	itemFailed          = "failed"
	itemNeedsRecovery   = "needs_recovery"
)

const quarantineDirName = ".hwatch-quarantine"

type transRunnerPlanItem struct {
	VideoID        string          `json:"video_id"`
	KeptVideoID    string          `json:"kept_video_id"`
	OriginalPath   string          `json:"original_path"`
	ExpectedSize   int64           `json:"expected_size"`
	ExpectedSHA256 string          `json:"expected_sha256"`
	Reason         string          `json:"reason"`
	Evidence       json.RawMessage `json:"evidence"`
}

type transRunnerPlan struct {
	Kind             string                `json:"kind"`
	PlanVersion      int                   `json:"plan_version"`
	ScanID           string                `json:"scan_id"`
	SubscriptionSlug string                `json:"subscription_slug"`
	GeneratedAt      string                `json:"generated_at"`
	Executable       bool                  `json:"executable"`
	Stale            bool                  `json:"stale"`
	UserConfirmed    bool                  `json:"user_confirmed"`
	Items            []transRunnerPlanItem `json:"items"`
	Deletions        []transRunnerPlanItem `json:"deletions"`
}

func (p transRunnerPlan) entries() []transRunnerPlanItem {
	if len(p.Items) > 0 {
		return p.Items
	}
	return p.Deletions
}

type planItemView struct {
	VideoID        string `json:"video_id"`
	KeptVideoID    string `json:"kept_video_id"`
	OriginalPath   string `json:"original_path"`
	ExpectedSize   int64  `json:"expected_size"`
	ExpectedSHA256 string `json:"expected_sha256"`
	Status         string `json:"status"`
	Reason         string `json:"reason,omitempty"`
	FreedBytes     int64  `json:"freed_bytes,omitempty"`
}

type planPreview struct {
	Kind             string         `json:"kind"`
	PlanVersion      int            `json:"plan_version"`
	ScanID           string         `json:"scan_id"`
	SubscriptionSlug string         `json:"subscription_slug"`
	Valid            bool           `json:"valid"`
	Problems         []string       `json:"problems"`
	Total            int            `json:"total"`
	Ready            int            `json:"ready"`
	Rejected         int            `json:"rejected"`
	AlreadyExcluded  int            `json:"already_excluded"`
	NeedsRecovery    int            `json:"needs_recovery"`
	EstimatedFreed   int64          `json:"estimated_freed"`
	Items            []planItemView `json:"items"`
}

type planExecuteResult struct {
	PlanID         string         `json:"plan_id"`
	DryRun         bool           `json:"dry_run"`
	Executed       int            `json:"executed"`
	Skipped        int            `json:"skipped"`
	Failed         int            `json:"failed"`
	NeedsRecovery  int            `json:"needs_recovery"`
	EstimatedFreed int64          `json:"estimated_freed"`
	Items          []planItemView `json:"items"`
}

func parseTransRunnerPlan(raw []byte) (transRunnerPlan, error) {
	var plan transRunnerPlan
	if err := json.Unmarshal(raw, &plan); err != nil {
		return plan, errors.New("删除计划 JSON 格式无效")
	}
	return plan, nil
}

func planHeaderProblems(plan transRunnerPlan) []string {
	problems := []string{}
	if plan.Kind != transRunnerPlanKind {
		problems = append(problems, "kind 不是 "+transRunnerPlanKind)
	}
	if plan.PlanVersion != transRunnerPlanVersion {
		problems = append(problems, "plan_version 不受支持")
	}
	if !plan.Executable {
		problems = append(problems, "计划不是 executable")
	}
	if plan.Stale {
		problems = append(problems, "计划被标记为 stale")
	}
	if !plan.UserConfirmed {
		problems = append(problems, "计划未经用户确认")
	}
	if strings.TrimSpace(plan.SubscriptionSlug) == "" {
		problems = append(problems, "缺少 subscription_slug")
	}
	if len(plan.entries()) == 0 {
		problems = append(problems, "计划没有条目")
	}
	return problems
}

func rejectItem(item transRunnerPlanItem, status, reason string) planItemView {
	return planItemView{VideoID: item.VideoID, KeptVideoID: item.KeptVideoID, OriginalPath: item.OriginalPath, ExpectedSize: item.ExpectedSize, ExpectedSHA256: item.ExpectedSHA256, Status: status, Reason: reason}
}

// verifyPlanItem re-checks every business and filesystem premise. It never
// trusts the plan's executable/stale flags or the supplied absolute path.
func (a *App) verifyPlanItem(plan transRunnerPlan, item transRunnerPlanItem, cfg store.Settings) planItemView {
	if strings.TrimSpace(item.VideoID) == "" || strings.TrimSpace(item.KeptVideoID) == "" {
		return rejectItem(item, itemRejected, "缺少 video_id 或 kept_video_id")
	}
	if item.VideoID == item.KeptVideoID {
		return rejectItem(item, itemRejected, "删除项与保留项相同")
	}
	v, err := a.store.Video(item.VideoID)
	if err != nil {
		return rejectItem(item, itemRejected, "删除视频不在数据库中")
	}
	kept, err := a.store.Video(item.KeptVideoID)
	if err != nil {
		return rejectItem(item, itemRejected, "保留视频不在数据库中")
	}
	if v.SubscriptionSlug != plan.SubscriptionSlug || kept.SubscriptionSlug != plan.SubscriptionSlug {
		return rejectItem(item, itemRejected, "删除项或保留项不属于指定订阅")
	}
	if a.store.IsMediaExcluded(v.ID) {
		return rejectItem(item, itemAlreadyExcluded, "删除项已加入已排除")
	}
	if v.Status != "completed" {
		return rejectItem(item, itemRejected, "删除项状态为 "+v.Status)
	}
	if v.TranscodeStatus == "pending" {
		return rejectItem(item, itemRejected, "删除项正在转码")
	}
	if kept.Status != "completed" {
		return rejectItem(item, itemRejected, "保留项状态为 "+kept.Status)
	}
	deletePath := mediaPathOf(v)
	if deletePath == "" {
		return rejectItem(item, itemRejected, "删除项没有媒体路径")
	}
	if _, err := mediaRelativePath(cfg.DownloadDir, deletePath); err != nil {
		return rejectItem(item, itemRejected, "删除路径不在受管媒体根目录内或越界")
	}
	if item.OriginalPath != "" && normalizeMediaPath(cfg.DownloadDir, item.OriginalPath) != normalizeMediaPath(cfg.DownloadDir, deletePath) {
		return rejectItem(item, itemRejected, "计划路径与数据库不一致")
	}
	if _, statErr := os.Stat(mediaPathOf(kept)); statErr != nil {
		return rejectItem(item, itemRejected, "保留文件不存在或不可访问")
	}
	info, statErr := os.Stat(deletePath)
	if statErr != nil {
		return rejectItem(item, itemNeedsRecovery, "删除文件不存在，需核对是否已隔离")
	}
	if !info.Mode().IsRegular() {
		return rejectItem(item, itemRejected, "删除路径不是普通文件")
	}
	shared, err := a.store.CountVideosWithPath(deletePath, v.ID)
	if err != nil {
		return rejectItem(item, itemNeedsRecovery, "无法确认文件共享引用")
	}
	if shared > 0 {
		return rejectItem(item, itemRejected, "同一物理文件被多条媒体记录引用")
	}
	size, _, sha, hashErr := dedupe.FileHashes(deletePath)
	if hashErr != nil {
		return rejectItem(item, itemNeedsRecovery, "无法读取文件哈希: "+hashErr.Error())
	}
	if item.ExpectedSize > 0 && size != item.ExpectedSize {
		return rejectItem(item, itemRejected, "文件大小与计划不一致")
	}
	if item.ExpectedSHA256 != "" && !strings.EqualFold(sha, item.ExpectedSHA256) {
		return rejectItem(item, itemRejected, "SHA-256 与计划不一致")
	}
	return planItemView{VideoID: v.ID, KeptVideoID: kept.ID, OriginalPath: deletePath, ExpectedSize: size, ExpectedSHA256: sha, Status: itemVerified, FreedBytes: size}
}

func (a *App) previewDeletionPlan(plan transRunnerPlan) planPreview {
	cfg, _ := a.store.Settings()
	preview := planPreview{
		Kind: plan.Kind, PlanVersion: plan.PlanVersion, ScanID: plan.ScanID, SubscriptionSlug: plan.SubscriptionSlug,
		Problems: planHeaderProblems(plan), Items: []planItemView{},
	}
	preview.Valid = len(preview.Problems) == 0
	seen := map[string]bool{}
	for _, item := range plan.entries() {
		if seen[item.VideoID] {
			continue
		}
		seen[item.VideoID] = true
		view := a.verifyPlanItem(plan, item, cfg)
		switch view.Status {
		case itemVerified:
			preview.Ready++
			preview.EstimatedFreed += view.FreedBytes
		case itemAlreadyExcluded:
			preview.AlreadyExcluded++
		case itemNeedsRecovery:
			preview.NeedsRecovery++
		default:
			preview.Rejected++
		}
		preview.Items = append(preview.Items, view)
	}
	preview.Total = len(preview.Items)
	if !storagePathAvailable(cfg.DownloadDir, cfg.StorageMountRequired) {
		preview.Valid = false
		preview.Problems = append(preview.Problems, "媒体存储当前不可用")
	}
	return preview
}

func planID(plan transRunnerPlan) string {
	if strings.TrimSpace(plan.ScanID) != "" {
		return plan.ScanID
	}
	return "plan-" + store.Now()
}

func quarantineTarget(root, planID, originalPath string) string {
	dir := filepath.Join(root, quarantineDirName, planID)
	base := filepath.Base(originalPath)
	return filepath.Join(dir, base)
}

func (a *App) excludeQuarantined(v store.Video) error {
	return a.store.RemoveAndExclude(v.ID, store.MediaExclusion{
		VideoID:             v.ID,
		Source:              v.Platform,
		Reason:              "transrunner_duplicate",
		TitleSnapshot:       v.Title,
		OriginalURLSnapshot: v.PageURL,
		ThumbnailSnapshot:   v.ThumbnailURL,
	})
}

// executeDeletionPlan verifies, quarantines and excludes. confirm=false is a
// pure Dry Run. Each item is persisted before/after its filesystem step so a
// crash leaves a recoverable state; repeat executions are idempotent.
func (a *App) executeDeletionPlan(plan transRunnerPlan, confirm bool) (planExecuteResult, error) {
	id := planID(plan)
	result := planExecuteResult{PlanID: id, DryRun: !confirm, Items: []planItemView{}}
	preview := a.previewDeletionPlan(plan)
	result.EstimatedFreed = preview.EstimatedFreed
	result.Items = preview.Items
	if !confirm {
		return result, nil
	}
	if !preview.Valid {
		return result, errors.New(strings.Join(preview.Problems, "; "))
	}
	cfg, _ := a.store.Settings()
	if !storagePathAvailable(cfg.DownloadDir, cfg.StorageMountRequired) {
		return result, errors.New("媒体存储当前不可用，已取消执行")
	}
	if err := a.store.UpsertDeletionPlan(store.DeletionPlan{PlanID: id, Kind: plan.Kind, PlanVersion: plan.PlanVersion, SubscriptionSlug: plan.SubscriptionSlug, ScanID: plan.ScanID, Status: "pending"}); err != nil {
		return result, err
	}
	executed, skipped, failed, recovering := 0, 0, 0, 0
	for _, view := range preview.Items {
		persist := store.DeletionPlanItem{PlanID: id, VideoID: view.VideoID, KeptVideoID: view.KeptVideoID, ExpectedSize: view.ExpectedSize, ExpectedSHA256: view.ExpectedSHA256, OriginalPath: view.OriginalPath, Status: itemPending}
		if view.Status != itemVerified {
			persist.Status = view.Status
			persist.Reason = view.Reason
			_ = a.store.SaveDeletionPlanItem(persist)
			if view.Status == itemAlreadyExcluded {
				skipped++
			} else if view.Status == itemNeedsRecovery {
				recovering++
			} else {
				skipped++
			}
			continue
		}
		v, err := a.store.Video(view.VideoID)
		if err != nil {
			failed++
			continue
		}
		qPath := quarantineTarget(cfg.DownloadDir, id, view.OriginalPath)
		_, origErr := os.Stat(view.OriginalPath)
		origExists := origErr == nil
		_, qErr := os.Stat(qPath)
		qExists := qErr == nil
		switch {
		case !origExists && qExists:
			persist.Status = itemQuarantined
			persist.QuarantinePath = qPath
			persist.Reason = "文件已在隔离区"
			_ = a.store.SaveDeletionPlanItem(persist)
		case origExists && !qExists:
			if err := os.MkdirAll(filepath.Dir(qPath), 0o755); err != nil {
				persist.Status = itemFailed
				persist.Reason = "无法创建隔离目录: " + err.Error()
				_ = a.store.SaveDeletionPlanItem(persist)
				failed++
				continue
			}
			persist.Status = itemVerified
			_ = a.store.SaveDeletionPlanItem(persist)
			if err := moveFile(view.OriginalPath, qPath); err != nil {
				persist.Status = itemFailed
				persist.Reason = "隔离文件失败: " + err.Error()
				_ = a.store.SaveDeletionPlanItem(persist)
				failed++
				continue
			}
			persist.Status = itemQuarantined
			persist.QuarantinePath = qPath
			_ = a.store.SaveDeletionPlanItem(persist)
		default:
			persist.Status = itemNeedsRecovery
			persist.Reason = "原文件与隔离文件同时存在或同时缺失，需人工核对"
			_ = a.store.SaveDeletionPlanItem(persist)
			recovering++
			continue
		}
		if err := a.excludeQuarantined(v); err != nil {
			persist.Status = itemNeedsRecovery
			persist.QuarantinePath = qPath
			persist.Reason = "已隔离但数据库更新失败: " + err.Error()
			_ = a.store.SaveDeletionPlanItem(persist)
			recovering++
			continue
		}
		persist.Status = itemDBExcluded
		persist.QuarantinePath = qPath
		_ = a.store.SaveDeletionPlanItem(persist)
		executed++
		_ = a.store.AddAudit("admin", "transrunner_plan_cleanup", "video", v.ID, "success", map[string]any{"plan_id": id})
	}
	result.Executed, result.Skipped, result.Failed, result.NeedsRecovery = executed, skipped, failed, recovering
	_ = a.store.SetDeletionPlanStatus(id, "completed")
	return result, nil
}

// recoverDeletionPlan finishes items that were quarantined but not yet synced to
// the database (e.g. after a crash).
func (a *App) recoverDeletionPlan(id string) (planExecuteResult, error) {
	cfg, _ := a.store.Settings()
	if !storagePathAvailable(cfg.DownloadDir, cfg.StorageMountRequired) {
		return planExecuteResult{PlanID: id}, errors.New("媒体存储当前不可用")
	}
	items, err := a.store.DeletionPlanItems(id)
	if err != nil {
		return planExecuteResult{PlanID: id}, err
	}
	result := planExecuteResult{PlanID: id, Items: []planItemView{}}
	for _, item := range items {
		if item.Status != itemQuarantined && item.Status != itemNeedsRecovery {
			continue
		}
		v, err := a.store.Video(item.VideoID)
		if err != nil {
			result.Failed++
			continue
		}
		_, origErr := os.Stat(item.OriginalPath)
		origExists := origErr == nil
		_, qErr := os.Stat(item.QuarantinePath)
		qExists := qErr == nil
		if origExists {
			result.Skipped++
			continue
		}
		if !qExists {
			result.NeedsRecovery++
			result.Items = append(result.Items, planItemView{VideoID: item.VideoID, Status: itemNeedsRecovery, Reason: "隔离文件缺失，无法恢复"})
			continue
		}
		if err := a.excludeQuarantined(v); err != nil {
			result.NeedsRecovery++
			result.Items = append(result.Items, planItemView{VideoID: item.VideoID, Status: itemNeedsRecovery, Reason: err.Error()})
			continue
		}
		_ = a.store.UpdateDeletionPlanItemStatus(id, item.VideoID, itemDBExcluded, item.QuarantinePath, "恢复完成")
		result.Executed++
	}
	return result, nil
}

func (a *App) mediaIntegrityPlanPreview(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Raw string `json:"raw"`
	}
	if err := decode(r, &input); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	plan, err := parseTransRunnerPlan([]byte(input.Raw))
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	respond(w, http.StatusOK, a.previewDeletionPlan(plan))
}

func (a *App) mediaIntegrityPlanExecute(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Raw     string `json:"raw"`
		Confirm bool   `json:"confirm"`
	}
	if err := decode(r, &input); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	plan, err := parseTransRunnerPlan([]byte(input.Raw))
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	result, err := a.executeDeletionPlan(plan, input.Confirm)
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	result.Items = nil
	respond(w, http.StatusOK, result)
}

func (a *App) mediaIntegrityPlans(w http.ResponseWriter, r *http.Request) {
	limit := intParam(r, "limit", 100, 1, 500)
	offset := intParam(r, "offset", 0, 0, 10000000)
	items, total, err := a.store.DeletionPlans(limit, offset)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	respond(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

func (a *App) mediaIntegrityPlanDetail(w http.ResponseWriter, r *http.Request) {
	plan, err := a.store.DeletionPlan(r.PathValue("id"))
	if err != nil {
		fail(w, http.StatusNotFound, errors.New("计划不存在"))
		return
	}
	items, err := a.store.DeletionPlanItems(plan.PlanID)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	respond(w, http.StatusOK, map[string]any{"plan": plan, "items": items})
}

func (a *App) mediaIntegrityPlanRecover(w http.ResponseWriter, r *http.Request) {
	result, err := a.recoverDeletionPlan(r.PathValue("id"))
	if err != nil {
		fail(w, http.StatusServiceUnavailable, err)
		return
	}
	result.Items = nil
	respond(w, http.StatusOK, result)
}
