package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"r34watch/internal/store"
	"r34watch/internal/transrunner"
)

func (a *App) transRunnerClient(cfg store.Settings) *transrunner.Client {
	return transrunner.New(cfg.TransRunnerURL, cfg.TransRunnerAPIToken)
}

func summarizeTransRunnerCapabilities(raw []byte) map[string]any {
	summary := map[string]any{}
	var parsed map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &parsed) != nil {
		return summary
	}
	for _, key := range []string{"videotoolbox", "video_toolbox", "h264", "hevc", "media_duplicate", "video_signature", "platform", "version"} {
		if value, ok := parsed[key]; ok {
			summary[key] = value
		}
	}
	if profiles, ok := parsed["profiles"]; ok {
		summary["profiles"] = profiles
	}
	return summary
}

// integrationTransRunnerTest verifies URL reachability and token validity by
// calling /health and then the authenticated /api/v1/capabilities.
func (a *App) integrationTransRunnerTest(w http.ResponseWriter, r *http.Request) {
	cfg, _ := a.store.Settings()
	if strings.TrimSpace(cfg.TransRunnerURL) == "" {
		fail(w, http.StatusConflict, errors.New("尚未配置 TransRunner 地址"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	result, err := a.transRunnerClient(cfg).TestConnection(ctx)
	if err != nil {
		_ = a.store.AddAudit("admin", "transrunner_tested", "integration", "transrunner", "failed", map[string]any{"error": transRunnerErrorMessage(err)})
		transRunnerFailure(w, err)
		return
	}
	_ = a.store.AddAudit("admin", "transrunner_tested", "integration", "transrunner", "success", map[string]any{"version": result.Version})
	respond(w, http.StatusOK, map[string]any{
		"ok":           true,
		"version":      result.Version,
		"platform":     result.Platform,
		"capabilities": summarizeTransRunnerCapabilities(result.Capabilities),
	})
}

// integrationTransRunnerSubmitTranscode submits a transcode job to TransRunner
// using an HWatch-relative input. HWatch only records that the job was sent; it
// does not mirror TransRunner's queue, progress or state.
func (a *App) integrationTransRunnerSubmitTranscode(w http.ResponseWriter, r *http.Request) {
	var input struct {
		VideoID string `json:"video_id"`
		Profile string `json:"profile"`
		Force   bool   `json:"force"`
	}
	if err := decode(r, &input); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	cfg, _ := a.store.Settings()
	if !cfg.EnableTransRunner {
		fail(w, http.StatusConflict, errors.New("TransRunner 未启用"))
		return
	}
	if strings.TrimSpace(cfg.TransRunnerURL) == "" {
		fail(w, http.StatusConflict, errors.New("尚未配置 TransRunner 地址"))
		return
	}
	v, err := a.store.Video(input.VideoID)
	if err != nil {
		fail(w, http.StatusNotFound, errors.New("视频不存在"))
		return
	}
	if v.Status != "completed" {
		fail(w, http.StatusConflict, errors.New("只有已下载媒体可以提交到 TransRunner"))
		return
	}
	profile := strings.TrimSpace(input.Profile)
	if profile == "" {
		profile = cfg.TransRunnerDefaultProfile
	}
	if profile == "" {
		fail(w, http.StatusUnprocessableEntity, errors.New("缺少转码 profile"))
		return
	}
	rel, err := mediaRelativePath(cfg.DownloadDir, mediaPathOf(v))
	if err != nil {
		fail(w, http.StatusConflict, errors.New("媒体路径不在媒体根目录内"))
		return
	}
	if !input.Force {
		if existing, e := a.store.LatestTransRunnerSubmission(v.ID, "transcode", profile); e == nil && existing.ID != 0 {
			respond(w, http.StatusConflict, map[string]any{"code": "already_submitted", "error": "该任务已发送到 TransRunner", "runner_job_id": existing.RunnerJobID})
			return
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	job, err := a.transRunnerClient(cfg).CreateJob(ctx, transrunner.CreateJobRequest{
		Input:   transrunner.HWInput{Type: "hwatch", VideoID: v.ID, MediaRelativePath: rel},
		Profile: profile,
	})
	if err != nil {
		_ = a.store.AddAudit("admin", "transrunner_job_submitted", "video", v.ID, "failed", map[string]any{"operation": "transcode", "profile": profile, "error": transRunnerErrorMessage(err)})
		transRunnerFailure(w, err)
		return
	}
	submissionID, err := a.store.AddTransRunnerSubmission(v.ID, "transcode", profile, job.ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	_ = a.store.AddAudit("admin", "transrunner_job_submitted", "video", v.ID, "success", map[string]any{"operation": "transcode", "profile": profile, "runner_job_id": job.ID})
	respond(w, http.StatusAccepted, map[string]any{
		"status":        "submitted",
		"runner_job_id": job.ID,
		"profile":       profile,
		"submission_id": submissionID,
	})
}

// integrationTransRunnerSubmitDedupe submits a media-duplicate job to
// TransRunner using an HWatch-relative input. TransRunner fetches the
// server-filtered candidates from HWatch itself; HWatch sends no candidates,
// absolute paths or algorithm settings.
func (a *App) integrationTransRunnerSubmitDedupe(w http.ResponseWriter, r *http.Request) {
	var input struct {
		VideoID string `json:"video_id"`
		Force   bool   `json:"force"`
	}
	if err := decode(r, &input); err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	cfg, _ := a.store.Settings()
	if !cfg.EnableTransRunner {
		fail(w, http.StatusConflict, errors.New("TransRunner 未启用"))
		return
	}
	if strings.TrimSpace(cfg.TransRunnerURL) == "" {
		fail(w, http.StatusConflict, errors.New("尚未配置 TransRunner 地址"))
		return
	}
	v, err := a.store.Video(input.VideoID)
	if err != nil {
		fail(w, http.StatusNotFound, errors.New("视频不存在"))
		return
	}
	if v.Status != "completed" {
		fail(w, http.StatusConflict, errors.New("只有已下载媒体可以提交到 TransRunner"))
		return
	}
	rel, err := mediaRelativePath(cfg.DownloadDir, mediaPathOf(v))
	if err != nil {
		fail(w, http.StatusConflict, errors.New("媒体路径不在媒体根目录内"))
		return
	}
	if !input.Force {
		if existing, e := a.store.LatestTransRunnerSubmission(v.ID, "media_duplicate", ""); e == nil && existing.ID != 0 {
			respond(w, http.StatusConflict, map[string]any{"code": "already_submitted", "error": "该任务已发送到 TransRunner", "runner_job_id": existing.RunnerJobID})
			return
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	job, err := a.transRunnerClient(cfg).CreateJob(ctx, transrunner.CreateJobRequest{
		Kind:  "media_duplicate",
		Input: transrunner.HWInput{Type: "hwatch", VideoID: v.ID, MediaRelativePath: rel},
	})
	if err != nil {
		_ = a.store.AddAudit("admin", "transrunner_job_submitted", "video", v.ID, "failed", map[string]any{"operation": "media_duplicate", "error": transRunnerErrorMessage(err)})
		transRunnerFailure(w, err)
		return
	}
	submissionID, err := a.store.AddTransRunnerSubmission(v.ID, "media_duplicate", "", job.ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	_ = a.store.AddAudit("admin", "transrunner_job_submitted", "video", v.ID, "success", map[string]any{"operation": "media_duplicate", "runner_job_id": job.ID})
	respond(w, http.StatusAccepted, map[string]any{
		"status":        "submitted",
		"runner_job_id": job.ID,
		"operation":     "media_duplicate",
		"submission_id": submissionID,
	})
}

func transRunnerErrorMessage(err error) string {
	var target *transrunner.Error
	if errors.As(err, &target) {
		return target.Message
	}
	return err.Error()
}

func transRunnerFailure(w http.ResponseWriter, err error) {
	var target *transrunner.Error
	if !errors.As(err, &target) {
		fail(w, http.StatusBadGateway, errors.New("无法连接 TransRunner"))
		return
	}
	switch target.Kind {
	case transrunner.ErrUnauthorized:
		fail(w, http.StatusBadGateway, errors.New("TransRunner API Token 无效"))
	case transrunner.ErrForbidden:
		fail(w, http.StatusBadGateway, errors.New("TransRunner 拒绝该请求"))
	case transrunner.ErrBadRequest:
		fail(w, http.StatusBadGateway, fmt.Errorf("TransRunner API 版本不兼容或被拒绝：%s", target.Message))
	case transrunner.ErrServer:
		fail(w, http.StatusBadGateway, fmt.Errorf("TransRunner 服务错误：%s", target.Message))
	default:
		fail(w, http.StatusBadGateway, errors.New("无法连接 TransRunner"))
	}
}
