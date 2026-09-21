package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
)

func (a *App) auditLogs(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	items, err := a.store.AuditLogs(limit, offset)
	if err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]any{"items": items, "limit": limit, "offset": offset})
}

func (a *App) taskHistory(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.TaskEvents(r.PathValue("id"), 100)
	if err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]any{"items": items})
}

func (a *App) exclusionPreview(w http.ResponseWriter, r *http.Request) {
	cfg, err := a.store.Settings()
	if err != nil {
		fail(w, 500, err)
		return
	}
	// A preview may use an unsaved rule draft. Only exclusion fields are accepted.
	var draft struct {
		ExcludeTitleKeywords string `json:"exclude_title_keywords"`
		ExcludeUploaders     string `json:"exclude_uploaders"`
		ExcludeCategories    string `json:"exclude_categories"`
		ExcludeTags          string `json:"exclude_tags"`
		MaxDurationSeconds   int64  `json:"max_duration_seconds"`
		RequireCreatorMatch  *bool  `json:"require_creator_match"`
	}
	if r.Body != nil && r.ContentLength != 0 {
		if err = decode(r, &draft); err != nil {
			fail(w, 400, err)
			return
		}
		cfg.ExcludeTitleKeywords = draft.ExcludeTitleKeywords
		cfg.ExcludeUploaders = draft.ExcludeUploaders
		cfg.ExcludeCategories = draft.ExcludeCategories
		cfg.ExcludeTags = draft.ExcludeTags
		cfg.MaxDurationSeconds = draft.MaxDurationSeconds
		if draft.RequireCreatorMatch != nil {
			cfg.RequireCreatorMatch = *draft.RequireCreatorMatch
		}
	}
	videos, err := a.store.Videos(100000, 0, "", "")
	if err != nil {
		fail(w, 500, err)
		return
	}
	type item struct {
		ID               string `json:"id"`
		Title            string `json:"title"`
		SubscriptionSlug string `json:"subscription_slug"`
		Reason           string `json:"reason"`
		Status           string `json:"status"`
	}
	out := []item{}
	for _, v := range videos {
		if v.Status == "completed" {
			continue
		}
		sub, e := a.store.Subscription(v.SubscriptionSlug)
		if e != nil {
			continue
		}
		if reason := exclusionReason(v, sub, cfg); reason != "" {
			out = append(out, item{v.ID, v.Title, v.SubscriptionSlug, reason, v.Status})
		}
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 500 {
		limit = 200
	}
	total := len(out)
	if len(out) > limit {
		out = out[:limit]
	}
	respond(w, 200, map[string]any{"items": out, "total": total, "truncated": total > len(out)})
}

func (a *App) duplicateBatch(w http.ResponseWriter, r *http.Request) {
	var input struct {
		IDs    []int64 `json:"ids"`
		Action string  `json:"action"`
	}
	if err := decode(r, &input); err != nil {
		fail(w, 400, err)
		return
	}
	if len(input.IDs) == 0 || len(input.IDs) > 500 || (input.Action != "confirm" && input.Action != "dismiss") {
		fail(w, 400, errors.New("无效的批量审查请求"))
		return
	}
	count, bytes, err := a.store.ReviewDuplicates(input.IDs, input.Action)
	if err != nil {
		fail(w, 500, err)
		return
	}
	if input.Action == "dismiss" {
		a.startQueueWorker()
	}
	respond(w, 200, map[string]any{"reviewed": count, "estimated_bytes": bytes, "action": strings.ToLower(input.Action)})
}
