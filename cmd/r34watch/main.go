package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
	"r34watch/internal/adapters"
	"r34watch/internal/downloader"
	"r34watch/internal/mediainfo"
	"r34watch/internal/plex"
	"r34watch/internal/store"
)

const userAgent = "Mozilla/5.0 (compatible; r34watch/2.0; NAS media manager)"

type App struct {
	store          *store.Store
	adapters       *adapters.Registry
	wget           *downloader.Wget
	mu             sync.Mutex
	queueMu        sync.Mutex
	bulkMu         sync.Mutex
	bulkRunning    bool
	thumbMu        sync.Mutex
	thumbWait      map[string]chan struct{}
	rateMu         sync.Mutex
	consecutive429 int
	backoffUntil   time.Time
	last429        time.Time
	nfoMu          sync.Mutex
	nfoRunning     bool
	hashMu         sync.Mutex
	hashRunning    map[string]bool
	backupMu       sync.Mutex
	recoveryMu     sync.Mutex
	maintenance    atomic.Bool
	authMu         sync.Mutex
	loginAttempts  map[string][]time.Time
	checking       map[string]bool
	lastSchedule   string
	lastBackupRun  string
	startedAt      time.Time
}

func New(s *store.Store) *App {
	return &App{store: s, adapters: adapters.NewRegistry(adapters.Rule34Video{}, adapters.Hanime1{}), wget: downloader.NewWget(), checking: map[string]bool{}, thumbWait: map[string]chan struct{}{}, hashRunning: map[string]bool{}, loginAttempts: map[string][]time.Time{}, startedAt: time.Now()}
}
func (a *App) reconcile() {
	items, _ := a.store.Videos(100000, 0, "completed", "")
	for _, v := range items {
		path := v.PlexPath
		if path == "" {
			path = v.FilePath
		}
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			_ = a.store.UpdateTask(v.ID, "failed", "", "", "", "媒体文件不存在，需要重新下载", 0, 0)
		} else if v.MediaProbedAt == "" {
			a.probeMedia(v.ID, path)
		}
	}
}

func (a *App) probeMedia(id, path string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	info, err := mediainfo.Probe(ctx, path)
	if err != nil {
		log.Printf("media probe %s: %v", id, err)
		return
	}
	if err = a.store.UpdateMediaInfo(id, info.VideoCodec, info.AudioCodec, info.Container, info.Width, info.Height, info.FrameRate, info.BitRate, info.BitDepth); err != nil {
		log.Printf("save media probe %s: %v", id, err)
	}
}
func (a *App) translatedMetadata(video store.Video, language string) store.Video {
	if language == "" {
		language = "zh-CN"
	}
	translations, _ := a.store.Translations(language)
	if title, ok := translations[video.ID+"\x00title"]; ok && strings.TrimSpace(title.Text) != "" {
		video.Title = title.Text
	}
	return video
}
func (a *App) writeTranslatedNFO(video store.Video, language string) error {
	path := video.PlexPath
	if path == "" {
		path = video.FilePath
	}
	if path == "" {
		return errors.New("媒体路径为空")
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}
	sub, err := a.store.Subscription(video.SubscriptionSlug)
	if err != nil {
		return err
	}
	cfg, _ := a.store.Settings()
	return plex.GenerateNFO(plex.Existing(cfg.DownloadDir, path), sub, a.translatedMetadata(video, language))
}
func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, err error) {
	codes := map[int]string{400: "invalid_request", 401: "unauthorized", 403: "forbidden", 404: "not_found", 409: "conflict", 422: "validation_failed", 429: "rate_limited", 500: "internal_error", 502: "upstream_unavailable"}
	code := codes[status]
	if code == "" {
		code = "request_failed"
	}
	respond(w, status, map[string]string{"code": code, "error": err.Error()})
}
func decode(r *http.Request, target any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 12<<20)).Decode(target)
}
func parseModelURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host != "rule34video.com" {
		return "", errors.New("请提供 rule34video.com 作者页")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 || parts[0] != "models" || parts[1] == "" {
		return "", errors.New("地址必须是 /models/<作者>/ 格式")
	}
	return parts[1], nil
}

func subMap(s store.Subscription) map[string]any {
	return map[string]any{"id": s.ID, "slug": s.Slug, "platform": s.Platform, "subscription_type": s.Type, "source_id": s.SourceID, "name": s.Name, "url": s.URL, "avatar_url": s.AvatarURL, "avatar_proxy_url": "/avatar/" + url.PathEscape(s.Slug), "external_links": jsonValue(s.ExternalLinks), "enabled": s.Enabled, "baseline_complete": s.BaselineComplete, "created_at": s.CreatedAt, "last_checked_at": s.LastCheckedAt, "last_error": s.LastError}
}
func (a *App) subscriptionMaps(items []store.Subscription) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, sub := range items {
		item := subMap(sub)
		p, _ := a.store.SubscriptionProgress(sub.Slug)
		item["progress"] = map[string]any{"total": p.Total, "completed": p.Completed, "active": p.Active, "waiting": p.Waiting, "failed": p.Failed, "bytes_downloaded": p.BytesDownloaded, "total_bytes": p.TotalBytes}
		out = append(out, item)
	}
	return out
}
func videoMap(v store.Video) map[string]any {
	failure := classifyDownloadMessage(v.Error)
	return map[string]any{"video_id": v.ID, "subscription_slug": v.SubscriptionSlug, "title": v.Title, "description": v.Description, "page_url": v.PageURL, "published_at": v.PublishedAt, "discovered_at": v.DiscoveredAt, "thumbnail_url": v.ThumbnailURL, "views": v.Views, "likes": v.Likes, "like_rate": v.LikeRate, "duration_seconds": v.DurationSeconds, "creator_name": v.CreatorName, "creator_url": v.CreatorURL, "creator_avatar": v.CreatorAvatar, "creators": jsonValue(v.CreatorsJSON), "source_relation": jsonValue(v.SourceRelationJSON), "uploader_name": v.UploaderName, "uploader_url": v.UploaderURL, "uploader_avatar": v.UploaderAvatar, "categories": jsonValue(v.CategoriesJSON), "tags": jsonValue(v.TagsJSON), "external_links": jsonValue(v.ExternalLinksJSON), "status": v.Status, "aria_gid": v.AriaGID, "downloader_id": v.DownloaderID, "external_task_id": v.ExternalID, "file_path": v.FilePath, "plex_path": v.PlexPath, "bytes_downloaded": v.BytesDownloaded, "total_bytes": v.TotalBytes, "download_speed": v.DownloadSpeed, "connections": v.Connections, "video_codec": v.VideoCodec, "audio_codec": v.AudioCodec, "container": v.Container, "width": v.Width, "height": v.Height, "frame_rate": v.FrameRate, "bit_rate": v.BitRate, "bit_depth": v.BitDepth, "media_probed_at": v.MediaProbedAt, "transcode_status": v.TranscodeStatus, "plex_season": v.PlexSeason, "plex_episode": v.PlexEpisode, "error": v.Error, "error_code": failure.Code, "error_retryable": failure.Retryable, "downloaded_at": v.DownloadedAt, "updated_at": v.UpdatedAt}
}
func translatedVideoMap(v store.Video, language string, translations map[string]store.Translation) map[string]any {
	item := videoMap(v)
	item["translation_language"], item["translated_title"], item["translated_tags"] = language, "", []string{}
	if x, ok := translations[v.ID+"\x00title"]; ok {
		item["translated_title"] = x.Text
	}
	if x, ok := translations[v.ID+"\x00tags"]; ok {
		var tags []string
		if json.Unmarshal([]byte(x.Text), &tags) == nil {
			item["translated_tags"] = tags
		}
	}
	return item
}
func translatedVideoMaps(items []store.Video, language string, translations map[string]store.Translation) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, translatedVideoMap(item, language, translations))
	}
	return out
}
func jsonValue(raw string) any {
	if raw == "" {
		return []any{}
	}
	var value any
	if json.Unmarshal([]byte(raw), &value) != nil {
		return []any{}
	}
	return value
}
func maps[T any](items []T, fn func(T) map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, fn(item))
	}
	return out
}

func (a *App) aria(cfg store.Settings) downloader.Engine {
	return downloader.NewAria(cfg.AriaRPCURL, cfg.AriaSecret)
}
func (a *App) selectedEngine(cfg store.Settings) downloader.Engine {
	if cfg.DownloaderMode == "wget" {
		return a.wget
	}
	if cfg.DownloaderMode == "auto" {
		ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
		defer cancel()
		if a.aria(cfg).Health(ctx).Online {
			return a.aria(cfg)
		}
		return a.wget
	}
	return a.aria(cfg)
}
func (a *App) engineFor(v store.Video, cfg store.Settings) downloader.Engine {
	if v.DownloaderID == "wget" {
		return a.wget
	}
	return a.aria(cfg)
}
func externalTaskID(v store.Video) string {
	if v.ExternalID != "" {
		return v.ExternalID
	}
	return v.AriaGID
}
func (a *App) ariaOnline(cfg store.Settings) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()
	return a.aria(cfg).Health(ctx).Online
}
func (a *App) downloaderHealth(ctx context.Context, cfg store.Settings) []downloader.Health {
	ariaCtx, ariaCancel := context.WithTimeout(ctx, 7*time.Second)
	ariaHealth := a.aria(cfg).Health(ariaCtx)
	ariaCancel()
	wgetCtx, wgetCancel := context.WithTimeout(ctx, 2*time.Second)
	wgetHealth := a.wget.Health(wgetCtx)
	wgetCancel()
	items := []downloader.Health{ariaHealth, wgetHealth}
	for i := range items {
		stats, _ := a.store.DownloaderStats(items[i].ID)
		items[i].Active, items[i].Connections, items[i].DownloadSpeed = stats.Active, stats.Connections, stats.Speed
		items[i].UptimeSeconds = int64(time.Since(a.startedAt).Seconds())
		if items[i].Error == "" {
			items[i].Error = stats.LastError
		}
	}
	return items
}
func (a *App) health(w http.ResponseWriter, r *http.Request) {
	cfg, _ := a.store.Settings()
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	respond(w, 200, a.healthSnapshot(ctx, cfg))
}
func (a *App) healthSnapshot(ctx context.Context, cfg store.Settings) map[string]any {
	warnings := []map[string]string{}
	dbErr := a.store.DB.PingContext(ctx)
	if dbErr != nil {
		warnings = append(warnings, map[string]string{"code": "database_unavailable", "severity": "error", "message": "SQLite 数据库不可用"})
	}
	total, free := diskStats(cfg.DownloadDir, cfg.StorageMountRequired)
	storageOnline := total > 0
	if !storageOnline {
		warnings = append(warnings, map[string]string{"code": "storage_unavailable", "severity": "error", "message": "媒体目录或 NFS 不可用"})
	}
	if storageOnline && cfg.MinFreeBytes > 0 && free < cfg.MinFreeBytes {
		warnings = append(warnings, map[string]string{"code": "storage_low", "severity": "error", "message": "可用空间低于最低保留空间，已暂停建立新任务"})
	}
	downloaders := a.downloaderHealth(ctx, cfg)
	selectedOnline := false
	for _, item := range downloaders {
		selected := item.ID == cfg.DownloaderMode || (cfg.DownloaderMode == "auto" && item.Online)
		if selected && item.Online {
			selectedOnline = true
		}
	}
	if !selectedOnline {
		warnings = append(warnings, map[string]string{"code": "downloader_unavailable", "severity": "error", "message": "当前下载模式没有可用下载器"})
	}
	failed, _ := a.store.CountVideos("failed")
	if failed > 0 {
		warnings = append(warnings, map[string]string{"code": "download_failures", "severity": "warning", "message": fmt.Sprintf("有 %d 个失败任务等待处理", failed)})
	}
	degraded := dbErr != nil || !storageOnline || (cfg.MinFreeBytes > 0 && free < cfg.MinFreeBytes) || !selectedOnline
	return map[string]any{"status": map[bool]string{true: "degraded", false: "ok"}[degraded], "warnings": warnings, "database": map[string]any{"online": dbErr == nil}, "storage": map[string]any{"online": storageOnline, "total_bytes": total, "free_bytes": free, "minimum_free_bytes": cfg.MinFreeBytes}, "queue": map[string]any{"paused": cfg.DownloadPaused}, "downloaders": downloaders, "uptime_seconds": int64(time.Since(a.startedAt).Seconds())}
}
func diskStats(path string, mountRequired bool) (total, free int64) {
	if path == "" {
		path = "."
	}
	if !storagePathAvailable(path, mountRequired) {
		return 0, 0
	}
	var stat unix.Statfs_t
	if unix.Statfs(path, &stat) == nil {
		return int64(stat.Blocks) * int64(stat.Bsize), int64(stat.Bavail) * int64(stat.Bsize)
	}
	return
}
func (a *App) dashboard(w http.ResponseWriter, r *http.Request) {
	subs, err := a.store.Subscriptions()
	if err != nil {
		fail(w, 500, err)
		return
	}
	videos, _ := a.store.RecentCompleted(12)
	recentlyPlayed, _ := a.store.RecentlyPlayed(12)
	language := r.URL.Query().Get("language")
	if language == "" {
		language = "zh-CN"
	}
	translations, _ := a.store.Translations(language)
	totalCount, _ := a.store.CountVideos("")
	activeCount, _ := a.store.CountVideos("active")
	waiting, _ := a.store.CountVideos("waiting")
	queued, _ := a.store.CountVideos("queued")
	submitting, _ := a.store.CountVideos("submitting")
	waiting += queued + submitting
	completed, _ := a.store.CountVideos("completed")
	discovered, _ := a.store.CountVideos("discovered")
	traffic, library, _ := a.store.Stats()
	downloadSpeed, connections, _ := a.store.TransferStats()
	cfg, _ := a.store.Settings()
	diskTotal, diskFree := diskStats(cfg.DownloadDir, cfg.StorageMountRequired)
	healthCtx, healthCancel := context.WithTimeout(r.Context(), 8*time.Second)
	health := a.healthSnapshot(healthCtx, cfg)
	healthCancel()
	downloaders, _ := health["downloaders"].([]downloader.Health)
	ariaOnline := false
	for _, item := range downloaders {
		if item.ID == "aria2" {
			ariaOnline = item.Online
		}
	}
	unreadNotifications, _ := a.store.UnreadNotifications()
	respond(w, 200, map[string]any{"subscriptions": a.subscriptionMaps(subs), "videos": translatedVideoMaps(videos, language, translations), "recently_played": translatedVideoMaps(recentlyPlayed, language, translations), "counts": map[string]any{"videos": totalCount, "active": activeCount, "waiting": waiting, "completed": completed, "discovered": discovered}, "traffic_bytes": traffic, "library_bytes": library, "current_download_speed": downloadSpeed, "current_connections": connections, "downloads_paused": cfg.DownloadPaused, "disk_total": diskTotal, "disk_free": diskFree, "aria2_online": ariaOnline, "downloader_mode": cfg.DownloaderMode, "downloaders": downloaders, "health_status": health["status"], "health_warnings": health["warnings"], "unread_notifications": unreadNotifications, "timezone": cfg.Timezone})
}
func (a *App) listVideos(w http.ResponseWriter, r *http.Request) {
	limit := intParam(r, "limit", 48, 1, 200)
	offset := intParam(r, "offset", 0, 0, 10000000)
	status := r.URL.Query().Get("status")
	values := r.URL.Query()
	language := values.Get("language")
	if language == "" {
		language = "zh-CN"
	}
	opts := store.VideoSearch{Status: status, Query: values.Get("q"), Scope: values.Get("scope"), Language: language, TagMode: values.Get("tag_mode"), Creator: values.Get("creator"), Codec: values.Get("codec"), Sort: values.Get("sort"), Order: values.Get("order"), Tags: csvValues(values.Get("tags")), ExcludeTags: csvValues(values.Get("exclude_tags")), MinHeight: intParam(r, "min_height", 0, 0, 10000)}
	items, total, err := a.store.SearchVideos(limit, offset, opts)
	if err != nil {
		fail(w, 500, err)
		return
	}
	translations, _ := a.store.Translations(language)
	respond(w, 200, map[string]any{"items": translatedVideoMaps(items, language, translations), "total": total, "limit": limit, "offset": offset})
}
func csvValues(raw string) []string {
	out := []string{}
	for _, value := range strings.Split(raw, ",") {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}
func (a *App) videoFacets(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.VideoFacets()
	if err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, items)
}

func (a *App) listTasks(w http.ResponseWriter, r *http.Request) {
	limit := intParam(r, "limit", 200, 1, 1000)
	offset := intParam(r, "offset", 0, 0, 10000000)
	status := r.URL.Query().Get("status")
	items, err := a.store.TaskVideos(limit, offset, status)
	if err != nil {
		fail(w, 500, err)
		return
	}
	total, _ := a.store.CountTasks(status)
	language := r.URL.Query().Get("language")
	if language == "" {
		language = "zh-CN"
	}
	translations, _ := a.store.Translations(language)
	respond(w, 200, map[string]any{"items": translatedVideoMaps(items, language, translations), "total": total, "limit": limit, "offset": offset})
}

type translationItem struct {
	Key          string         `json:"key"`
	Platform     string         `json:"platform"`
	SourceID     string         `json:"source_id"`
	SourceHash   string         `json:"source_hash"`
	Fields       map[string]any `json:"fields"`
	Translations map[string]any `json:"translations"`
}
type translationPackage struct {
	SchemaVersion  int               `json:"schema_version"`
	ExportedAt     string            `json:"exported_at,omitempty"`
	SourceLanguage string            `json:"source_language"`
	TargetLanguage string            `json:"target_language"`
	Provider       string            `json:"provider,omitempty"`
	Items          []translationItem `json:"items"`
}

func translationFields(v store.Video) map[string]any {
	tags := []string{}
	var links []struct{ Name string }
	_ = json.Unmarshal([]byte(v.TagsJSON), &links)
	for _, tag := range links {
		if tag.Name != "" {
			tags = append(tags, tag.Name)
		}
	}
	return map[string]any{"title": v.Title, "tags": tags}
}
func sourceHash(fields map[string]any) string {
	raw, _ := json.Marshal(fields)
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("sha256:%x", sum[:])
}
func (a *App) exportTranslations(w http.ResponseWriter, r *http.Request) {
	language := r.URL.Query().Get("language")
	if language == "" {
		language = "zh-CN"
	}
	platform, subscription := r.URL.Query().Get("platform"), r.URL.Query().Get("subscription")
	videos, err := a.store.Videos(100000, 0, "", "")
	if err != nil {
		fail(w, 500, err)
		return
	}
	existing, _ := a.store.Translations(language)
	pkg := translationPackage{SchemaVersion: 1, ExportedAt: store.Now(), SourceLanguage: "auto", TargetLanguage: language, Items: []translationItem{}}
	for _, video := range videos {
		if platform != "" && video.Platform != platform {
			continue
		}
		if subscription != "" && video.SubscriptionSlug != subscription {
			continue
		}
		fields := translationFields(video)
		translated := map[string]any{"title": "", "tags": []string{}}
		if x, ok := existing[video.ID+"\x00title"]; ok {
			translated["title"] = x.Text
		}
		if x, ok := existing[video.ID+"\x00tags"]; ok {
			var tags []string
			if json.Unmarshal([]byte(x.Text), &tags) == nil {
				translated["tags"] = tags
			}
		}
		pkg.Items = append(pkg.Items, translationItem{Key: video.ID, Platform: video.Platform, SourceID: video.SourceID, SourceHash: sourceHash(fields), Fields: fields, Translations: translated})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=r34watch-translation-"+language+".json")
	_ = json.NewEncoder(w).Encode(pkg)
}
func (a *App) importTranslations(w http.ResponseWriter, r *http.Request) {
	var pkg translationPackage
	if err := decode(r, &pkg); err != nil {
		fail(w, 400, errors.New("翻译 JSON 格式错误"))
		return
	}
	if pkg.SchemaVersion != 1 || pkg.TargetLanguage == "" || len(pkg.Items) > 100000 {
		fail(w, 400, errors.New("翻译包版本、语言或条目数量无效"))
		return
	}
	imported, skipped := 0, 0
	for _, item := range pkg.Items {
		video, err := a.store.Video(item.Key)
		if err != nil || sourceHash(translationFields(video)) != item.SourceHash {
			skipped++
			continue
		}
		if title, ok := item.Translations["title"].(string); ok && strings.TrimSpace(title) != "" {
			_ = a.store.SaveTranslation(store.Translation{EntityID: video.ID, Language: pkg.TargetLanguage, Field: "title", SourceHash: item.SourceHash, Text: strings.TrimSpace(title), Provider: pkg.Provider})
			imported++
		}
		if tags, ok := item.Translations["tags"].([]any); ok {
			values := []string{}
			for _, value := range tags {
				if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
					values = append(values, strings.TrimSpace(text))
				}
			}
			if len(values) > 0 {
				raw, _ := json.Marshal(values)
				_ = a.store.SaveTranslation(store.Translation{EntityID: video.ID, Language: pkg.TargetLanguage, Field: "tags", SourceHash: item.SourceHash, Text: string(raw), Provider: pkg.Provider})
				imported++
			}
		}
	}
	respond(w, 200, map[string]int{"imported_fields": imported, "skipped_items": skipped})
}
func (a *App) refreshNFO(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Language string `json:"language"`
	}
	_ = decode(r, &input)
	if input.Language == "" {
		input.Language = "zh-CN"
	}
	a.nfoMu.Lock()
	if a.nfoRunning {
		a.nfoMu.Unlock()
		fail(w, 409, errors.New("NFO 刷新任务正在运行"))
		return
	}
	items, err := a.store.Videos(100000, 0, "completed", "")
	if err != nil {
		a.nfoMu.Unlock()
		fail(w, 500, err)
		return
	}
	a.nfoRunning = true
	a.nfoMu.Unlock()
	go func(videos []store.Video, language string) {
		defer func() { a.nfoMu.Lock(); a.nfoRunning = false; a.nfoMu.Unlock() }()
		for _, video := range videos {
			if err := a.writeTranslatedNFO(video, language); err != nil {
				log.Printf("refresh NFO %s: %v", video.ID, err)
			}
		}
	}(items, input.Language)
	respond(w, 202, map[string]any{"accepted": len(items), "language": input.Language})
}

func (a *App) downloadAll(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimSpace(r.URL.Query().Get("subscription"))
	accepted, err := a.store.QueueDiscovered(slug)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	a.startQueueWorker()
	respond(w, http.StatusAccepted, map[string]any{"accepted": accepted})
}
func (a *App) toggleDownloads(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Paused bool `json:"paused"`
	}
	if decode(r, &input) != nil {
		fail(w, http.StatusBadRequest, errors.New("请求格式错误"))
		return
	}
	cfg, err := a.store.Settings()
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if input.Paused {
		if err = a.store.SetDownloadPaused(true); err != nil {
			fail(w, http.StatusInternalServerError, err)
			return
		}
		_ = a.aria(cfg).PauseAll(ctx)
		_ = a.wget.PauseAll(ctx)
	} else {
		_ = a.aria(cfg).ResumeAll(ctx)
		_ = a.wget.ResumeAll(ctx)
	}
	if !input.Paused {
		err = a.store.SetDownloadPaused(false)
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	if !input.Paused {
		a.startQueueWorker()
	}
	respond(w, http.StatusAccepted, map[string]any{"paused": input.Paused})
}
func (a *App) startQueueWorker() {
	if a.maintenance.Load() {
		return
	}
	a.bulkMu.Lock()
	if a.bulkRunning {
		a.bulkMu.Unlock()
		return
	}
	a.bulkRunning = true
	a.bulkMu.Unlock()
	go func() {
		defer func() {
			a.bulkMu.Lock()
			a.bulkRunning = false
			a.bulkMu.Unlock()
		}()
		for {
			if a.maintenance.Load() {
				return
			}
			cfg, cfgErr := a.store.Settings()
			if cfgErr != nil || cfg.DownloadPaused {
				return
			}
			if wait := a.rateLimitWait(); wait > 0 {
				log.Printf("download queue paused for %s after HTTP 429", wait.Round(time.Second))
				time.Sleep(wait)
			}
			engine := a.selectedEngine(cfg)
			if waiter, ok := engine.(downloader.CapacityWaiter); ok {
				waitCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				err := waiter.WaitForCapacity(waitCtx)
				cancel()
				if err != nil {
					continue
				}
			}
			videos, err := a.store.Videos(1, 0, "queued", "")
			if err != nil || len(videos) == 0 {
				return
			}
			video := videos[0]
			if err = a.queue(video.ID); err != nil {
				log.Printf("persistent queue %s: %v", video.ID, err)
				if err.Error() == "下载队列已暂停" {
					return
				}
				failure := classifyDownloadError(err)
				if failure.Code == "capacity_reached" {
					continue
				}
				if failure.Code == "downloader_offline" {
					return
				}
				if strings.HasPrefix(err.Error(), "excluded: ") {
					continue
				}
				if strings.HasPrefix(err.Error(), "duplicate: ") {
					continue
				}
				if failure.Retryable {
					_ = a.store.UpdateTask(video.ID, "queued", "", "", "", err.Error(), 0, 0)
					if failure.Code == "rate_limited" {
						a.record429()
					}
					if failure.Delay > 0 && failure.Code != "rate_limited" {
						time.Sleep(failure.Delay)
					}
					continue
				}
				_ = a.store.UpdateTask(video.ID, "failed", "", "", "", err.Error(), 0, 0)
			}
			time.Sleep(750 * time.Millisecond)
		}
	}()
}
func isRateLimited(message string) bool {
	return strings.Contains(message, "HTTP 429") || strings.Contains(message, "status=429")
}
func (a *App) record429() time.Duration {
	a.rateMu.Lock()
	defer a.rateMu.Unlock()
	a.consecutive429++
	a.last429 = time.Now()
	shift := a.consecutive429 - 1
	if shift > 5 {
		shift = 5
	}
	delay := 30 * time.Second * time.Duration(1<<shift)
	if delay > 15*time.Minute {
		delay = 15 * time.Minute
	}
	until := time.Now().Add(delay)
	if until.After(a.backoffUntil) {
		a.backoffUntil = until
	}
	return delay
}
func (a *App) rateLimitWait() time.Duration {
	a.rateMu.Lock()
	defer a.rateMu.Unlock()
	if !a.last429.IsZero() && time.Since(a.last429) > 10*time.Minute {
		a.consecutive429 = 0
		a.backoffUntil = time.Time{}
		a.last429 = time.Time{}
	}
	if wait := time.Until(a.backoffUntil); wait > 0 {
		return wait
	}
	return 0
}
func intParam(r *http.Request, key string, def, min, max int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return def
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
func (a *App) settings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		cfg, err := a.store.Settings()
		if err != nil {
			fail(w, 500, err)
			return
		}
		respond(w, 200, settingsMap(cfg))
		return
	}
	var input map[string]any
	if decode(r, &input) != nil {
		fail(w, 400, errors.New("请求格式错误"))
		return
	}
	cfg, err := a.store.Settings()
	if err != nil {
		fail(w, 500, err)
		return
	}
	previousDownloadDir := filepath.Clean(cfg.DownloadDir)
	previousBackupDir := filepath.Clean(cfg.BackupDir)
	assignString(input, "http_proxy", &cfg.HTTPProxy)
	assignString(input, "download_dir", &cfg.DownloadDir)
	assignString(input, "aria_rpc_url", &cfg.AriaRPCURL)
	assignString(input, "aria_secret", &cfg.AriaSecret)
	assignString(input, "downloader_mode", &cfg.DownloaderMode)
	assignString(input, "check_time", &cfg.CheckTime)
	assignString(input, "timezone", &cfg.Timezone)
	assignString(input, "plex_language", &cfg.PlexLanguage)
	assignString(input, "backup_dir", &cfg.BackupDir)
	assignString(input, "backup_time", &cfg.BackupTime)
	assignString(input, "exclude_title_keywords", &cfg.ExcludeTitleKeywords)
	assignString(input, "exclude_uploaders", &cfg.ExcludeUploaders)
	assignString(input, "exclude_categories", &cfg.ExcludeCategories)
	assignString(input, "exclude_tags", &cfg.ExcludeTags)
	assignBool(input, "require_creator_match", &cfg.RequireCreatorMatch)
	assignBool(input, "auto_transcode", &cfg.AutoTranscode)
	assignString(input, "transrunner_url", &cfg.TransRunnerURL)
	assignString(input, "transrunner_default_profile", &cfg.TransRunnerDefaultProfile)
	assignBool(input, "enable_transrunner", &cfg.EnableTransRunner)
	if value, ok := input["transrunner_api_token"].(string); ok && strings.TrimSpace(value) != "" {
		cfg.TransRunnerAPIToken = strings.TrimSpace(value)
	}
	if clear, ok := input["transrunner_api_token_clear"].(bool); ok && clear {
		cfg.TransRunnerAPIToken = ""
	}
	assignInt(input, "max_concurrent", &cfg.MaxConcurrent)
	assignInt(input, "connections_per_download", &cfg.ConnectionsPerDownload)
	assignInt64(input, "download_limit_kbps", &cfg.DownloadLimitKBPS)
	assignInt64(input, "min_free_bytes", &cfg.MinFreeBytes)
	assignInt64(input, "max_duration_seconds", &cfg.MaxDurationSeconds)
	assignInt(input, "backup_retention", &cfg.BackupRetention)
	if cfg.HTTPProxy != "" {
		u, e := url.Parse(cfg.HTTPProxy)
		if e != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			fail(w, 400, errors.New("HTTP 代理格式无效"))
			return
		}
	}
	if cfg.DownloadDir == "" {
		fail(w, 400, errors.New("下载目录不能为空"))
		return
	}
	if cfg.ConnectionsPerDownload < 1 || cfg.ConnectionsPerDownload > 16 {
		fail(w, 400, errors.New("单任务连接数必须在 1 到 16 之间"))
		return
	}
	if cfg.DownloadLimitKBPS < 0 {
		fail(w, 400, errors.New("下载限速不能小于 0"))
		return
	}
	if cfg.DownloaderMode != "aria2" && cfg.DownloaderMode != "wget" && cfg.DownloaderMode != "auto" {
		fail(w, 400, errors.New("下载器模式无效"))
		return
	}
	if strings.TrimSpace(cfg.TransRunnerURL) != "" {
		u, e := url.Parse(cfg.TransRunnerURL)
		if e != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			fail(w, 400, errors.New("TransRunner 地址格式无效"))
			return
		}
	}
	if cfg.BackupRetention < 1 || cfg.BackupRetention > 365 {
		fail(w, 400, errors.New("数据库备份保留数量必须在 1 到 365 之间"))
		return
	}
	if _, parseErr := time.Parse("15:04", cfg.BackupTime); parseErr != nil {
		fail(w, 400, errors.New("数据库备份时间格式无效"))
		return
	}
	if absolute, absErr := filepath.Abs(cfg.DownloadDir); absErr == nil {
		cfg.DownloadDir = absolute
	}
	if filepath.Clean(cfg.DownloadDir) != previousDownloadDir {
		// A newly selected media root must already exist. This avoids silently
		// creating a local directory when the intended NAS mount is offline.
		if info, statErr := os.Stat(cfg.DownloadDir); statErr != nil || !info.IsDir() {
			fail(w, 400, errors.New("新下载目录必须已存在且可访问"))
			return
		}
		cfg.StorageMountRequired = pathHasDistinctMount(cfg.DownloadDir) || requiresExternalMount(cfg.DownloadDir)
		if !storagePathAvailable(cfg.DownloadDir, cfg.StorageMountRequired) {
			fail(w, 400, errors.New("新下载目录的外部存储未挂载"))
			return
		}
	}
	if strings.TrimSpace(cfg.BackupDir) != "" {
		if absolute, absErr := filepath.Abs(cfg.BackupDir); absErr == nil {
			cfg.BackupDir = absolute
		}
		if filepath.Clean(cfg.BackupDir) != previousBackupDir {
			if info, statErr := os.Stat(cfg.BackupDir); statErr != nil || !info.IsDir() {
				fail(w, 400, errors.New("新备份目录必须已存在且可访问"))
				return
			}
			cfg.BackupMountRequired = pathHasDistinctMount(cfg.BackupDir) || requiresExternalMount(cfg.BackupDir)
			if !storagePathAvailable(cfg.BackupDir, cfg.BackupMountRequired) {
				fail(w, 400, errors.New("新备份目录的外部存储未挂载"))
				return
			}
		}
	} else {
		cfg.BackupMountRequired = false
	}
	if err = a.store.SaveSettings(cfg); err != nil {
		fail(w, 500, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	_ = a.aria(cfg).SetGlobal(ctx, cfg.MaxConcurrent, cfg.DownloadLimitKBPS)
	_ = a.wget.SetGlobal(ctx, cfg.MaxConcurrent, cfg.DownloadLimitKBPS)
	cancel()
	respond(w, 200, settingsMap(cfg))
}
func settingsMap(c store.Settings) map[string]any {
	return map[string]any{"http_proxy": c.HTTPProxy, "download_dir": c.DownloadDir, "aria_rpc_url": c.AriaRPCURL, "aria_secret": c.AriaSecret, "downloader_mode": c.DownloaderMode, "check_time": c.CheckTime, "timezone": c.Timezone, "plex_language": c.PlexLanguage, "max_concurrent": c.MaxConcurrent, "connections_per_download": c.ConnectionsPerDownload, "download_limit_kbps": c.DownloadLimitKBPS, "min_free_bytes": c.MinFreeBytes, "require_creator_match": c.RequireCreatorMatch, "auto_transcode": c.AutoTranscode, "enable_transrunner": c.EnableTransRunner, "transrunner_url": c.TransRunnerURL, "transrunner_default_profile": c.TransRunnerDefaultProfile, "transrunner_token_configured": c.TransRunnerAPIToken != "", "exclude_title_keywords": c.ExcludeTitleKeywords, "exclude_uploaders": c.ExcludeUploaders, "exclude_categories": c.ExcludeCategories, "exclude_tags": c.ExcludeTags, "max_duration_seconds": c.MaxDurationSeconds, "backup_dir": c.BackupDir, "backup_time": c.BackupTime, "backup_retention": c.BackupRetention, "backup_last_success": c.BackupLastSuccess, "backup_last_error": c.BackupLastError}
}
func assignString(m map[string]any, k string, v *string) {
	if x, ok := m[k].(string); ok {
		*v = strings.TrimSpace(x)
	}
}
func assignInt(m map[string]any, k string, v *int) {
	if x, ok := m[k].(float64); ok {
		*v = int(x)
	}
}
func assignInt64(m map[string]any, k string, v *int64) {
	if x, ok := m[k].(float64); ok {
		*v = int64(x)
	}
}
func assignBool(m map[string]any, k string, v *bool) {
	if x, ok := m[k].(bool); ok {
		*v = x
	}
}

func (a *App) subscribe(w http.ResponseWriter, r *http.Request) {
	var input struct {
		URL              string `json:"url"`
		Type             string `json:"type"`
		DownloadExisting bool   `json:"download_existing"`
	}
	if decode(r, &input) != nil {
		fail(w, 400, errors.New("请求格式错误"))
		return
	}
	raw := strings.TrimSpace(input.URL)
	adapter, err := a.adapters.Match(raw)
	if err != nil {
		fail(w, 400, err)
		return
	}
	cfg, _ := a.store.Settings()
	sub, err := adapter.ParseSubscription(r.Context(), raw, input.Type)
	if err != nil {
		fail(w, 400, err)
		return
	}
	if enricher, ok := adapter.(adapters.SubscriptionEnricher); ok {
		sub = enricher.EnrichSubscription(r.Context(), sub, cfg.HTTPProxy)
	}
	items, err := adapter.ListVideos(r.Context(), sub, cfg.HTTPProxy)
	if err != nil {
		fail(w, 502, err)
		return
	}
	sub.BaselineComplete, sub.CreatedAt, sub.LastCheckedAt = true, store.Now(), store.Now()
	if err = a.store.AddSubscription(sub); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			fail(w, 409, errors.New("该订阅已存在"))
		} else {
			fail(w, 500, err)
		}
		return
	}
	for _, v := range items {
		_, _ = a.store.UpsertDiscovered(v)
	}
	queued := int64(0)
	if input.DownloadExisting {
		queued, _ = a.store.QueueDiscovered(sub.Slug)
		a.startQueueWorker()
	}
	respond(w, 201, map[string]any{"slug": sub.Slug, "platform": sub.Platform, "baselined": len(items), "queued": queued})
	go a.enrich(sub.Slug, items)
}
func (a *App) enrich(slug string, items []store.Video) {
	cfg, _ := a.store.Settings()
	sub, _ := a.store.Subscription(slug)
	adapter, err := a.adapters.ByID(sub.Platform)
	if err != nil {
		return
	}
	for _, item := range items {
		detailed, err := adapter.FetchDetails(context.Background(), item, cfg.HTTPProxy)
		if err == nil {
			_ = a.store.UpdateDetails(detailed)
			if strings.TrimRight(detailed.CreatorURL, "/") == strings.TrimRight(sub.URL, "/") {
				_ = a.store.UpdateSubscriptionMetadata(slug, detailed.CreatorName, detailed.CreatorAvatar, detailed.ExternalLinksJSON)
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
}
func (a *App) subscriptions(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.Subscriptions()
	if err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]any{"items": a.subscriptionMaps(items)})
}
func (a *App) platforms(w http.ResponseWriter, r *http.Request) {
	items := []map[string]string{}
	for _, adapter := range a.adapters.All() {
		items = append(items, map[string]string{"id": adapter.ID(), "name": adapter.Name()})
	}
	respond(w, 200, map[string]any{"items": items})
}
func (a *App) searchCreators(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(query)) < 2 {
		fail(w, 400, errors.New("作者关键词至少需要 2 个字符"))
		return
	}
	platform := r.URL.Query().Get("platform")
	if platform == "" {
		platform = "rule34video"
	}
	adapter, err := a.adapters.ByID(platform)
	if err != nil {
		fail(w, 400, err)
		return
	}
	cfg, _ := a.store.Settings()
	items, err := adapter.SearchCreators(r.Context(), query, cfg.HTTPProxy)
	if err != nil {
		fail(w, 502, err)
		return
	}
	respond(w, 200, map[string]any{"items": items})
}
func (a *App) subscriptionAction(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	switch r.Method {
	case http.MethodPatch:
		var input map[string]any
		if decode(r, &input) != nil {
			fail(w, 400, errors.New("请求格式错误"))
			return
		}
		enabled, _ := input["enabled"].(bool)
		if err := a.store.SetSubscriptionEnabled(slug, enabled); err != nil {
			fail(w, 404, err)
			return
		}
		respond(w, 200, map[string]any{"enabled": enabled})
	case http.MethodDelete:
		// Prevent an in-flight submission from racing the subscription cascade.
		a.queueMu.Lock()
		defer a.queueMu.Unlock()
		items, listErr := a.store.SubscriptionVideos(slug)
		if listErr != nil {
			fail(w, 500, listErr)
			return
		}
		cfg, _ := a.store.Settings()
		for _, item := range items {
			if item.Status != "submitting" && item.Status != "waiting" && item.Status != "active" && item.Status != "paused" {
				continue
			}
			externalID := externalTaskID(item)
			if externalID == "" {
				continue
			}
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			removeErr := a.engineFor(item, cfg).Remove(ctx, externalID)
			cancel()
			if removeErr != nil && classifyDownloadError(removeErr).Code != "detached_task" && !strings.Contains(strings.ToLower(removeErr.Error()), "not found") {
				fail(w, http.StatusServiceUnavailable, fmt.Errorf("停止下载任务 %s 失败: %w", item.ID, removeErr))
				return
			}
		}
		if err := a.store.DeleteSubscription(slug); err != nil {
			fail(w, 500, err)
			return
		}
		respond(w, 200, map[string]bool{"deleted": true})
	}
}
func (a *App) triggerCheck(slug string) {
	if a.maintenance.Load() {
		return
	}
	a.mu.Lock()
	if a.checking[slug] {
		a.mu.Unlock()
		return
	}
	a.checking[slug] = true
	a.mu.Unlock()
	go func() { defer func() { a.mu.Lock(); delete(a.checking, slug); a.mu.Unlock() }(); a.check(slug) }()
}
func (a *App) check(slug string) {
	if a.maintenance.Load() {
		return
	}
	sub, err := a.store.Subscription(slug)
	if err != nil || !sub.Enabled {
		return
	}
	cfg, _ := a.store.Settings()
	adapter, err := a.adapters.ByID(sub.Platform)
	if err != nil {
		_ = a.store.UpdateSubscriptionCheck(slug, err.Error())
		return
	}
	if enricher, ok := adapter.(adapters.SubscriptionEnricher); ok {
		enriched := enricher.EnrichSubscription(context.Background(), sub, cfg.HTTPProxy)
		if enriched.Name != sub.Name || enriched.AvatarURL != sub.AvatarURL {
			_ = a.store.UpdateSubscriptionMetadata(sub.Slug, enriched.Name, enriched.AvatarURL, enriched.ExternalLinks)
			sub = enriched
		}
	}
	items, err := adapter.ListVideos(context.Background(), sub, cfg.HTTPProxy)
	if err != nil {
		_ = a.store.UpdateSubscriptionCheck(slug, err.Error())
		return
	}
	_ = a.store.UpdateSubscriptionCheck(slug, "")
	for _, item := range items {
		created, e := a.store.UpsertDiscovered(item)
		if e == nil {
			existing, _ := a.store.Video(item.ID)
			needsHanimeMetadata := sub.Platform == "hanime1" && existing.Description == ""
			if created || existing.DurationSeconds == 0 || needsHanimeMetadata {
				if detailed, e := adapter.FetchDetails(context.Background(), item, cfg.HTTPProxy); e == nil {
					detailed.SubscriptionSlug = slug
					_ = a.store.UpdateDetails(detailed)
					if strings.TrimRight(detailed.CreatorURL, "/") == strings.TrimRight(sub.URL, "/") {
						_ = a.store.UpdateSubscriptionMetadata(slug, detailed.CreatorName, detailed.CreatorAvatar, detailed.ExternalLinksJSON)
					}
				}
				if needsHanimeMetadata {
					time.Sleep(250 * time.Millisecond)
				}
			}
			if created {
				if a.store.QueueVideo(item.ID) == nil {
					a.startQueueWorker()
				}
			}
		}
	}
	if requeued, _ := a.store.RequeueTrustedCreatorMismatches(slug); requeued > 0 {
		a.startQueueWorker()
	}
}
func (a *App) checkHandler(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug != "" {
		a.triggerCheck(slug)
	} else {
		subs, _ := a.store.Subscriptions()
		for _, sub := range subs {
			if sub.Enabled {
				a.triggerCheck(sub.Slug)
			}
		}
	}
	respond(w, 202, map[string]string{"status": "started"})
}

func (a *App) queue(id string) error {
	if a.maintenance.Load() {
		return errors.New("服务正在执行数据库维护")
	}
	a.queueMu.Lock()
	defer a.queueMu.Unlock()
	if a.maintenance.Load() {
		return errors.New("服务正在执行数据库维护")
	}
	v, err := a.store.Video(id)
	if err != nil {
		return err
	}
	if v.Status == "active" || v.Status == "waiting" || v.Status == "paused" {
		return errors.New("任务已存在")
	}
	cfg, _ := a.store.Settings()
	if cfg.DownloadPaused {
		return errors.New("下载队列已暂停")
	}
	total, free := diskStats(cfg.DownloadDir, cfg.StorageMountRequired)
	if total == 0 {
		return errors.New("媒体目录不可用或挂载已断开")
	}
	if cfg.MinFreeBytes > 0 && free < cfg.MinFreeBytes {
		return errors.New("磁盘可用空间低于保留值")
	}
	sub, err := a.store.Subscription(v.SubscriptionSlug)
	if err != nil {
		return err
	}
	if v.DurationSeconds == 0 || (v.CreatorURL == "" && !trustedSubscriptionListing(v.SourceRelationJSON, sub)) {
		adapter, _ := a.adapters.ByID(v.Platform)
		if detail, e := adapter.FetchDetails(context.Background(), v, cfg.HTTPProxy); e == nil {
			v = detail
			_ = a.store.UpdateDetails(v)
		}
	}
	if err = a.duplicateBeforeDownload(v, cfg); err != nil {
		return err
	}
	if reason := exclusionReason(v, sub, cfg); reason != "" {
		_ = a.store.UpdateTask(id, "excluded", "", "", "", reason, 0, 0)
		return errors.New("excluded: " + reason)
	}
	if v.PlexEpisode == 0 {
		episode, e := a.store.NextEpisode(v.SubscriptionSlug)
		if e != nil {
			return e
		}
		if e = a.store.AssignEpisode(v.ID, 1, episode); e != nil {
			return e
		}
		v.PlexSeason = 1
		v.PlexEpisode = episode
	}
	adapter, err := a.adapters.ByID(v.Platform)
	if err != nil {
		return err
	}
	resolved, err := adapter.ResolveDownload(context.Background(), v, cfg.HTTPProxy)
	if err != nil {
		if classifyDownloadError(err).Code == "source_expired" {
			_ = a.refreshExpiredSource(v, cfg, err.Error())
			if fresh, freshErr := a.store.Video(v.ID); freshErr == nil {
				resolved, err = adapter.ResolveDownload(context.Background(), fresh, cfg.HTTPProxy)
			}
		}
	}
	if err != nil {
		return err
	}
	if latest, e := a.store.Settings(); e != nil || latest.DownloadPaused {
		return errors.New("下载队列已暂停")
	}
	layout := plex.Build(cfg.DownloadDir, sub, v)
	if err = os.MkdirAll(layout.SeasonDir, 0755); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	engine := a.selectedEngine(cfg)
	healthCtx, healthCancel := context.WithTimeout(context.Background(), 7*time.Second)
	health := engine.Health(healthCtx)
	healthCancel()
	if !health.Online {
		return fmt.Errorf("downloader unavailable: %s", engine.ID())
	}
	plannedID := downloadSubmissionID(engine.ID(), v.ID)
	preflightCtx, preflightCancel := context.WithTimeout(context.Background(), 3*time.Second)
	preflight, preflightErr := engine.Tell(preflightCtx, plannedID)
	preflightCancel()
	if preflightErr == nil && (preflight.State == "error" || preflight.State == "removed") {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 3*time.Second)
		if cleaner, ok := engine.(downloader.ResultCleaner); ok {
			_ = cleaner.CleanResult(cleanupCtx, plannedID)
		} else {
			_ = engine.Remove(cleanupCtx, plannedID)
		}
		cleanupCancel()
		preflightErr = errors.New("terminal downloader result cleaned")
	}
	if err = a.store.BeginSubmission(id, engine.ID(), plannedID, layout.FullPath); err != nil {
		return err
	}
	if preflightErr == nil {
		return a.store.BindDownloader(id, engine.ID(), plannedID, "waiting", layout.FullPath)
	}
	externalID, err := engine.Add(ctx, downloader.Request{URL: resolved.URL, Directory: layout.SeasonDir, FileName: layout.FileName, Referer: v.PageURL, UserAgent: userAgent, Proxy: cfg.HTTPProxy, ExternalID: plannedID, Connections: cfg.ConnectionsPerDownload})
	if err != nil {
		// A timed-out RPC can still have accepted the deterministic ID. Recover
		// that task instead of submitting a second writer for the same path.
		recoverCtx, recoverCancel := context.WithTimeout(context.Background(), 3*time.Second)
		recovered, recoverErr := engine.Tell(recoverCtx, plannedID)
		recoverCancel()
		if recoverErr == nil && recovered.State != "error" && recovered.State != "removed" {
			return a.store.BindDownloader(id, engine.ID(), plannedID, "waiting", layout.FullPath)
		}
		if errors.Is(err, downloader.ErrCapacityReached) {
			_ = a.store.UpdateDownloaderTask(id, "queued", "", "", v.FilePath, v.PlexPath, "", v.BytesDownloaded, v.TotalBytes)
			return err
		}
		_ = a.store.UpdateDownloaderTask(id, "failed", "", "", "", "", err.Error(), 0, 0)
		return err
	}
	if err = a.store.BindDownloader(id, engine.ID(), externalID, "waiting", layout.FullPath); err != nil {
		removeCtx, removeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = engine.Remove(removeCtx, externalID)
		removeCancel()
		_ = a.store.UpdateDownloaderTask(id, "queued", "", "", v.FilePath, v.PlexPath, "下载器绑定失败，任务已撤销并重新排队", v.BytesDownloaded, v.TotalBytes)
		return err
	}
	return nil
}

func downloadSubmissionID(engineID, videoID string) string {
	sum := sha256.Sum256([]byte(engineID + "\x00" + videoID))
	if engineID == "aria2" {
		return fmt.Sprintf("%x", sum[:8])
	}
	return "hwatch-" + fmt.Sprintf("%x", sum[:12])
}

func exclusionReason(v store.Video, sub store.Subscription, cfg store.Settings) string {
	if trustedSubscriptionListing(v.SourceRelationJSON, sub) {
		return contentExclusionReason(v, cfg)
	}
	creatorURL := strings.TrimRight(strings.ToLower(v.CreatorURL), "/")
	subURL := strings.TrimRight(strings.ToLower(sub.URL), "/")
	if cfg.RequireCreatorMatch && creatorURL != "" && subURL != "" && creatorURL != subURL {
		return "实际创作者与订阅作者不一致: " + v.CreatorName
	}
	return contentExclusionReason(v, cfg)
}
func contentExclusionReason(v store.Video, cfg store.Settings) string {
	if matchedRule(v.Title, cfg.ExcludeTitleKeywords) {
		return "标题命中排除规则"
	}
	if matchedRule(v.UploaderName, cfg.ExcludeUploaders) {
		return "上传者命中排除规则"
	}
	if matchedJSONNames(v.CategoriesJSON, cfg.ExcludeCategories) {
		return "分类命中排除规则"
	}
	if matchedJSONNames(v.TagsJSON, cfg.ExcludeTags) {
		return "标签命中排除规则"
	}
	if cfg.MaxDurationSeconds > 0 && v.DurationSeconds > cfg.MaxDurationSeconds {
		return "视频时长超过限制"
	}
	return ""
}
func trustedSubscriptionListing(raw string, sub store.Subscription) bool {
	var relation struct {
		Type             string `json:"type"`
		Platform         string `json:"platform"`
		SubscriptionSlug string `json:"subscription_slug"`
	}
	return json.Unmarshal([]byte(raw), &relation) == nil && (relation.Type == "creator_listing" || relation.Type == "playlist_listing") && relation.Platform == sub.Platform && relation.SubscriptionSlug == sub.Slug
}
func exclusionRules(raw string) []string {
	return strings.FieldsFunc(strings.ToLower(raw), func(r rune) bool { return r == ',' || r == '\n' || r == ';' || r == '，' })
}
func matchedRule(value, rawRules string) bool {
	value = strings.ToLower(value)
	for _, rule := range exclusionRules(rawRules) {
		if rule = strings.TrimSpace(rule); rule != "" && strings.Contains(value, rule) {
			return true
		}
	}
	return false
}
func matchedJSONNames(rawJSON, rawRules string) bool {
	var items []struct{ Name string }
	if json.Unmarshal([]byte(rawJSON), &items) != nil {
		return false
	}
	for _, item := range items {
		if matchedRule(item.Name, rawRules) {
			return true
		}
	}
	return false
}
func (a *App) videoAction(w http.ResponseWriter, r *http.Request) {
	id, action := r.PathValue("id"), r.PathValue("action")
	v, err := a.store.Video(id)
	if err != nil {
		fail(w, 404, err)
		return
	}
	cfg, _ := a.store.Settings()
	client := a.engineFor(v, cfg)
	externalID := externalTaskID(v)
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	switch action {
	case "download", "retry":
		err = a.store.QueueVideo(id)
		if err == nil {
			a.startQueueWorker()
		}
	case "pause":
		err = client.Pause(ctx, externalID)
		if err == nil {
			err = a.store.UpdateDownloaderTask(id, "paused", client.ID(), externalID, v.FilePath, v.PlexPath, "", v.BytesDownloaded, v.TotalBytes)
		}
	case "resume":
		err = client.Resume(ctx, externalID)
		if err != nil && v.DownloaderID == "wget" {
			err = a.store.UpdateDownloaderTask(id, "queued", "", "", v.FilePath, v.PlexPath, "", v.BytesDownloaded, v.TotalBytes)
			if err == nil {
				a.startQueueWorker()
			}
		}
	case "delete":
		if externalID != "" {
			_ = client.Remove(ctx, externalID)
		}
		if v.PlexPath != "" {
			_ = os.Remove(v.PlexPath)
			base := strings.TrimSuffix(v.PlexPath, filepath.Ext(v.PlexPath))
			_ = os.Remove(base + ".nfo")
			_ = os.Remove(base + ".jpg")
			_ = os.Remove(v.PlexPath + ".aria2")
		}
		err = a.store.UpdateDownloaderTask(id, "discovered", "", "", "", "", "", 0, 0)
	default:
		err = errors.New("未知操作")
	}
	if err != nil {
		fail(w, 400, err)
		return
	}
	respond(w, 202, map[string]string{"status": "accepted"})
}
func (a *App) importHTML(w http.ResponseWriter, r *http.Request) {
	var input map[string]string
	if decode(r, &input) != nil {
		fail(w, 400, errors.New("请求格式错误"))
		return
	}
	raw := input["html"]
	adapter, _ := a.adapters.ByID("rule34video")
	v, err := adapter.ParseImported(raw)
	if err != nil {
		_ = a.store.SaveImport("paste", raw, "", err.Error())
		fail(w, 400, err)
		return
	}
	existing, e := a.store.Video(v.ID)
	if e != nil {
		sub, parseErr := adapter.ParseSubscription(r.Context(), v.CreatorURL, "creator")
		if parseErr != nil {
			_ = a.store.SaveImport("paste", raw, v.ID, parseErr.Error())
			fail(w, 409, errors.New("HTML 缺少可识别的创作者页面"))
			return
		}
		if _, subErr := a.store.Subscription(sub.Slug); subErr != nil {
			name := v.CreatorName
			if name == "" {
				name = sub.SourceID
			}
			sub.Name, sub.AvatarURL, sub.ExternalLinks, sub.Enabled, sub.BaselineComplete, sub.CreatedAt, sub.LastCheckedAt = name, v.CreatorAvatar, v.ExternalLinksJSON, true, true, store.Now(), store.Now()
			if addErr := a.store.AddSubscription(sub); addErr != nil {
				fail(w, 500, addErr)
				return
			}
		}
		v.SubscriptionSlug = sub.Slug
		_, _ = a.store.UpsertDiscovered(v)
		existing, _ = a.store.Video(v.ID)
	}
	v.SubscriptionSlug = existing.SubscriptionSlug
	if v.Title == "" {
		v.Title = existing.Title
	}
	if existing.PageURL != "" {
		v.PageURL = existing.PageURL
	}
	v.DiscoveredAt = existing.DiscoveredAt
	_ = a.store.UpdateDetails(v)
	_ = a.store.SaveImport("paste", raw, v.ID, "")
	saved, _ := a.store.Video(v.ID)
	respond(w, 200, videoMap(saved))
}

func (a *App) monitor(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.pollTasks(ctx)
		}
	}
}
func (a *App) pollTasks(ctx context.Context) {
	if a.maintenance.Load() {
		return
	}
	cfg, _ := a.store.Settings()
	for _, state := range []string{"waiting", "active", "paused"} {
		// Poll the complete persisted set. A fixed first page silently starved
		// older tasks when an imported aria2 session contained more than 500.
		items, _ := a.store.Videos(100000, 0, state, "")
		for _, v := range items {
			client := a.engineFor(v, cfg)
			externalID := externalTaskID(v)
			callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			status, err := client.Tell(callCtx, externalID)
			cancel()
			if err != nil {
				failure := classifyDownloadError(err)
				if failure.Code == "detached_task" {
					healthCtx, healthCancel := context.WithTimeout(ctx, 2*time.Second)
					online := client.Health(healthCtx).Online
					healthCancel()
					if online {
						_ = a.store.UpdateDownloaderTask(v.ID, "queued", "", "", v.FilePath, v.PlexPath, "原下载任务已脱离，已自动重新排队", v.BytesDownloaded, v.TotalBytes)
						a.startQueueWorker()
					}
				}
				continue
			}
			mapped := status.State
			if mapped == "complete" {
				mapped = "completed"
			}
			if mapped == "error" || mapped == "removed" {
				mapped = "failed"
			}
			delta := status.Completed - v.BytesDownloaded
			if delta > 0 {
				_ = a.store.AddTraffic(v.ID, delta)
			}
			path := v.FilePath
			if len(status.Files) > 0 {
				path = status.Files[0]
			}
			errText := status.ErrorMessage
			if errText == "" && status.ErrorCode != "" && mapped == "failed" {
				errText = "aria2 error " + status.ErrorCode
			}
			failure := classifyDownloaderStatus(status.ErrorCode, errText)
			if mapped == "failed" && failure.Retryable {
				removeCtx, removeCancel := context.WithTimeout(ctx, 5*time.Second)
				_ = client.Remove(removeCtx, externalID)
				removeCancel()
				if failure.Code == "source_expired" {
					errText = a.refreshExpiredSource(v, cfg, errText)
				} else if failure.Code == "rate_limited" {
					a.record429()
				}
				_ = a.store.UpdateDownloaderTask(v.ID, "queued", "", "", path, v.PlexPath, errText, status.Completed, status.Total)
				a.startQueueWorker()
				continue
			}
			_ = a.store.UpdateDownloaderTask(v.ID, mapped, client.ID(), externalID, path, v.PlexPath, errText, status.Completed, status.Total)
			_ = a.store.UpdateTransfer(v.ID, status.DownloadSpeed, status.Connections)
			if mapped == "completed" {
				fresh, _ := a.store.Video(v.ID)
				sub, _ := a.store.Subscription(v.SubscriptionSlug)
				layout := plex.Build(cfg.DownloadDir, sub, fresh)
				translated := a.translatedMetadata(fresh, cfg.PlexLanguage)
				_ = plex.Generate(layout, sub, translated, cfg.HTTPProxy)
				a.probeMedia(v.ID, path)
				a.hashCompletedMedia(v.ID, path)
				if cleaner, ok := client.(downloader.ResultCleaner); ok {
					cleanCtx, cleanCancel := context.WithTimeout(ctx, 5*time.Second)
					_ = cleaner.CleanResult(cleanCtx, externalID)
					cleanCancel()
				}
			}
		}
	}
}

func (a *App) refreshExpiredSource(video store.Video, cfg store.Settings, cause string) string {
	adapter, err := a.adapters.ByID(video.Platform)
	if err == nil {
		refreshCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		fresh, refreshErr := adapter.FetchDetails(refreshCtx, video, cfg.HTTPProxy)
		cancel()
		if refreshErr == nil {
			fresh.SubscriptionSlug = video.SubscriptionSlug
			_ = a.store.UpdateDetails(fresh)
			cause = "下载源已失效，详情已刷新并重新排队"
		}
	}
	return cause
}
func (a *App) scheduler(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if a.maintenance.Load() {
				continue
			}
			a.reconcileRecovery()
			a.hashNextCompletedMedia()
			notifyCtx, notifyCancel := context.WithTimeout(ctx, 5*time.Second)
			a.evaluateNotifications(notifyCtx)
			notifyCancel()
			cfg, _ := a.store.Settings()
			loc, err := time.LoadLocation(cfg.Timezone)
			if err != nil {
				loc = time.Local
			}
			key := time.Now().In(loc).Format("2006-01-02 15:04")
			if strings.HasSuffix(key, " "+cfg.CheckTime) && key != a.lastSchedule {
				a.lastSchedule = key
				subs, _ := a.store.Subscriptions()
				for _, s := range subs {
					if s.Enabled {
						a.triggerCheck(s.Slug)
					}
				}
			}
			if strings.HasSuffix(key, " "+cfg.BackupTime) && key != a.lastBackupRun {
				a.lastBackupRun = key
				go func() {
					if _, backupErr := a.createBackup(false); backupErr != nil {
						log.Printf("database backup failed: %v", backupErr)
					}
				}()
			}
		}
	}
}

func (a *App) media(w http.ResponseWriter, r *http.Request) {
	v, err := a.store.Video(r.PathValue("id"))
	path := v.PlexPath
	if path == "" {
		path = v.FilePath
	}
	if err != nil || path == "" {
		http.NotFound(w, r)
		return
	}
	path, err = filepath.Abs(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if v.Status == "completed" {
		_ = a.store.RecordPlayback(v.ID)
	}
	http.ServeFile(w, r, path)
}
func (a *App) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			items := []store.Video{}
			for _, state := range []string{"waiting", "active", "paused"} {
				part, _ := a.store.Videos(200, 0, state, "")
				items = append(items, part...)
			}
			speed, connections, _ := a.store.TransferStats()
			raw, _ := json.Marshal(map[string]any{"tasks": maps(items, videoMap), "download_speed": speed, "connections": connections})
			_, _ = fmt.Fprintf(w, "data: %s\n\n", raw)
			flusher.Flush()
		}
	}
}
func (a *App) thumbnail(w http.ResponseWriter, r *http.Request) {
	v, err := a.store.Video(r.PathValue("id"))
	if err != nil || v.ThumbnailURL == "" {
		http.NotFound(w, r)
		return
	}
	cfg, _ := a.store.Settings()
	w.Header().Set("Cache-Control", "public, max-age=86400")
	if mediaPath := firstExistingThumbnail(v); mediaPath != "" {
		http.ServeFile(w, r, mediaPath)
		return
	}
	cacheDir := filepath.Join(cfg.DownloadDir, ".hwatch-thumbnails")
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(v.ID)))
	cachePath := filepath.Join(cacheDir, digest+".jpg")
	if _, statErr := os.Stat(cachePath); statErr == nil {
		http.ServeFile(w, r, cachePath)
		return
	}
	a.thumbMu.Lock()
	if pending, exists := a.thumbWait[v.ID]; exists {
		a.thumbMu.Unlock()
		select {
		case <-pending:
			if _, statErr := os.Stat(cachePath); statErr == nil {
				http.ServeFile(w, r, cachePath)
				return
			}
		case <-r.Context().Done():
			return
		}
		a.thumbnailPlaceholder(w)
		return
	}
	pending := make(chan struct{})
	a.thumbWait[v.ID] = pending
	a.thumbMu.Unlock()
	defer func() {
		a.thumbMu.Lock()
		delete(a.thumbWait, v.ID)
		close(pending)
		a.thumbMu.Unlock()
	}()
	transport := &http.Transport{}
	if cfg.HTTPProxy != "" {
		if p, e := url.Parse(cfg.HTTPProxy); e == nil {
			transport.Proxy = http.ProxyURL(p)
		}
	}
	client := &http.Client{Transport: transport, Timeout: 20 * time.Second}
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, v.ThumbnailURL, nil)
	req.Header.Set("User-Agent", userAgent)
	res, err := client.Do(req)
	if err != nil {
		a.thumbnailPlaceholder(w)
		return
	}
	defer res.Body.Close()
	kind := res.Header.Get("Content-Type")
	if res.StatusCode < 200 || res.StatusCode >= 300 || !strings.HasPrefix(kind, "image/") {
		a.thumbnailPlaceholder(w)
		return
	}
	if err = os.MkdirAll(cacheDir, 0755); err != nil {
		a.thumbnailPlaceholder(w)
		return
	}
	tmp, err := os.CreateTemp(cacheDir, ".thumb-*")
	if err != nil {
		a.thumbnailPlaceholder(w)
		return
	}
	tmpPath := tmp.Name()
	_, copyErr := io.Copy(tmp, io.LimitReader(res.Body, 20<<20))
	closeErr := tmp.Close()
	if copyErr != nil || closeErr != nil || os.Rename(tmpPath, cachePath) != nil {
		_ = os.Remove(tmpPath)
		a.thumbnailPlaceholder(w)
		return
	}
	http.ServeFile(w, r, cachePath)
}
func firstExistingThumbnail(v store.Video) string {
	for _, mediaPath := range []string{v.PlexPath, v.FilePath} {
		if mediaPath == "" {
			continue
		}
		candidate := strings.TrimSuffix(mediaPath, filepath.Ext(mediaPath)) + ".jpg"
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}
func (a *App) thumbnailPlaceholder(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, `<svg xmlns="http://www.w3.org/2000/svg" width="320" height="180"><rect width="100%" height="100%" fill="#17191f"/><text x="50%" y="50%" fill="#9199a7" text-anchor="middle" dominant-baseline="middle" font-family="sans-serif" font-size="15">缩略图暂不可用</text></svg>`)
}

func (a *App) subscriptionAvatar(w http.ResponseWriter, r *http.Request) {
	sub, err := a.store.Subscription(r.PathValue("slug"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	cfg, _ := a.store.Settings()
	cacheDir := filepath.Join(cfg.DownloadDir, ".hwatch-avatars")
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(sub.Slug)))
	cachePath := filepath.Join(cacheDir, digest+".img")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	if _, statErr := os.Stat(cachePath); statErr == nil {
		http.ServeFile(w, r, cachePath)
		return
	}
	if sub.AvatarURL == "" {
		a.avatarPlaceholder(w, sub.Name)
		return
	}
	transport := &http.Transport{}
	if cfg.HTTPProxy != "" {
		if p, parseErr := url.Parse(cfg.HTTPProxy); parseErr == nil {
			transport.Proxy = http.ProxyURL(p)
		}
	}
	client := &http.Client{Transport: transport, Timeout: 20 * time.Second}
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, sub.AvatarURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/140 Safari/537.36")
	req.Header.Set("Referer", sub.URL)
	res, fetchErr := client.Do(req)
	if fetchErr != nil {
		a.avatarPlaceholder(w, sub.Name)
		return
	}
	defer res.Body.Close()
	kind := res.Header.Get("Content-Type")
	if res.StatusCode < 200 || res.StatusCode >= 300 || !strings.HasPrefix(kind, "image/") {
		a.avatarPlaceholder(w, sub.Name)
		return
	}
	if err = os.MkdirAll(cacheDir, 0755); err != nil {
		a.avatarPlaceholder(w, sub.Name)
		return
	}
	tmp, err := os.CreateTemp(cacheDir, ".avatar-*")
	if err != nil {
		a.avatarPlaceholder(w, sub.Name)
		return
	}
	tmpPath := tmp.Name()
	_, copyErr := io.Copy(tmp, io.LimitReader(res.Body, 8<<20))
	closeErr := tmp.Close()
	if copyErr != nil || closeErr != nil || os.Rename(tmpPath, cachePath) != nil {
		_ = os.Remove(tmpPath)
		a.avatarPlaceholder(w, sub.Name)
		return
	}
	w.Header().Set("Content-Type", kind)
	http.ServeFile(w, r, cachePath)
}

func (a *App) avatarPlaceholder(w http.ResponseWriter, name string) {
	initial := "H"
	if value := []rune(strings.TrimSpace(name)); len(value) > 0 {
		initial = html.EscapeString(string(value[0]))
	}
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `<svg xmlns="http://www.w3.org/2000/svg" width="128" height="128"><rect width="128" height="128" rx="64" fill="#e5b951"/><text x="64" y="68" fill="#241f14" text-anchor="middle" dominant-baseline="middle" font-family="sans-serif" font-size="52" font-weight="700">%s</text></svg>`, initial)
}
func staticHandler(dist string) http.Handler {
	files := http.FileServer(http.Dir(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			target := filepath.Join(dist, filepath.Clean(r.URL.Path))
			if info, err := os.Stat(target); err == nil && !info.IsDir() {
				kind := mime.TypeByExtension(filepath.Ext(target))
				if filepath.Ext(target) == ".webmanifest" {
					kind = "application/manifest+json; charset=utf-8"
				}
				if kind != "" {
					w.Header().Set("Content-Type", kind)
				}
				files.ServeHTTP(w, r)
				return
			}
		}
		http.ServeFile(w, r, filepath.Join(dist, "index.html"))
	})
}
func (a *App) router(dist string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /openapi.yaml", openAPISpec)
	mux.HandleFunc("GET /docs", openAPIDocs)
	mux.HandleFunc("GET /api/auth/status", a.authStatus)
	mux.HandleFunc("POST /api/auth/setup", a.authSetup)
	mux.HandleFunc("POST /api/auth/login", a.authLogin)
	mux.HandleFunc("POST /api/auth/logout", a.authLogout)
	mux.HandleFunc("GET /api/auth/me", a.authMe)
	mux.HandleFunc("GET /api/dashboard", a.dashboard)
	mux.HandleFunc("GET /api/health", a.health)
	mux.HandleFunc("GET /api/notifications", a.notifications)
	mux.HandleFunc("POST /api/notifications/read-all", a.notifications)
	mux.HandleFunc("GET /api/duplicates", a.duplicates)
	mux.HandleFunc("POST /api/duplicates/{id}/{action}", a.duplicates)
	mux.HandleFunc("POST /api/duplicates/batch", a.duplicateBatch)
	mux.HandleFunc("POST /api/exclusions/preview", a.exclusionPreview)
	mux.HandleFunc("GET /api/audit", a.auditLogs)
	mux.HandleFunc("GET /api/tasks/{id}/history", a.taskHistory)
	mux.HandleFunc("GET /api/integrations/transrunner/token", a.integrationTokenGet)
	mux.HandleFunc("POST /api/integrations/transrunner/token", a.integrationTokenCreate)
	mux.HandleFunc("DELETE /api/integrations/transrunner/token", a.integrationTokenDelete)
	mux.HandleFunc("POST /api/integrations/transrunner/test", a.integrationTransRunnerTest)
	mux.HandleFunc("POST /api/integrations/transrunner/submit/transcode", a.integrationTransRunnerSubmitTranscode)
	mux.HandleFunc("GET /api/transrunner/v1/media/{video_id}", a.requireIntegrationScope("read_media", a.trGetMedia))
	mux.HandleFunc("GET /api/transrunner/v1/media/{video_id}/dedupe-candidates", a.requireIntegrationScope("read_dedupe_candidates", a.trDedupeCandidates))
	mux.HandleFunc("POST /api/transrunner/v1/duplicate-results", a.requireIntegrationScope("write_duplicate_results", a.trDuplicateResults))
	mux.HandleFunc("POST /api/transrunner/v1/transcode-results", a.requireIntegrationScope("write_transcode_results", a.trTranscodeResults))
	mux.HandleFunc("GET /api/videos", a.listVideos)
	mux.HandleFunc("GET /api/videos/facets", a.videoFacets)
	mux.HandleFunc("GET /api/tasks", a.listTasks)
	mux.HandleFunc("GET /api/translations/export", a.exportTranslations)
	mux.HandleFunc("POST /api/translations/import", a.importTranslations)
	mux.HandleFunc("POST /api/plex/refresh-nfo", a.refreshNFO)
	mux.HandleFunc("POST /api/videos/download-all", a.downloadAll)
	mux.HandleFunc("POST /api/downloads/toggle", a.toggleDownloads)
	mux.HandleFunc("GET /api/settings", a.settings)
	mux.HandleFunc("PUT /api/settings", a.settings)
	mux.HandleFunc("GET /api/backups", a.backups)
	mux.HandleFunc("GET /api/recovery/latest", a.recoveryStatus)
	mux.HandleFunc("POST /api/backups", a.backups)
	mux.HandleFunc("POST /api/backups/{name}/restore", a.restoreBackup)
	mux.HandleFunc("GET /api/subscriptions", a.subscriptions)
	mux.HandleFunc("GET /api/platforms", a.platforms)
	mux.HandleFunc("GET /api/creators/search", a.searchCreators)
	mux.HandleFunc("POST /api/subscriptions", a.subscribe)
	mux.HandleFunc("PATCH /api/subscriptions/{slug}", a.subscriptionAction)
	mux.HandleFunc("DELETE /api/subscriptions/{slug}", a.subscriptionAction)
	mux.HandleFunc("POST /api/check", a.checkHandler)
	mux.HandleFunc("POST /api/check/{slug}", a.checkHandler)
	mux.HandleFunc("POST /api/videos/{id}/{action}", a.videoAction)
	mux.HandleFunc("POST /api/import/html", a.importHTML)
	mux.HandleFunc("GET /media/{id}", a.media)
	mux.HandleFunc("GET /thumb/{id}", a.thumbnail)
	mux.HandleFunc("GET /avatar/{slug}", a.subscriptionAvatar)
	mux.HandleFunc("GET /api/events", a.events)
	mux.Handle("/", staticHandler(dist))
	return requestLog(a.requireAuth(mux))
}
func openAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	http.ServeFile(w, r, "openapi.yaml")
}
func openAPIDocs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>HWatch API</title><style>body{max-width:960px;margin:40px auto;padding:0 20px;background:#101116;color:#e8e9ed;font:15px/1.6 system-ui}a{color:#e5b951}pre{overflow:auto;padding:18px;border:1px solid #30333c;border-radius:12px;background:#181a21;white-space:pre-wrap}</style></head><body><h1>HWatch API V1</h1><p>规范文件：<a href="/openapi.yaml">openapi.yaml</a>。本页只加载本机资源，不依赖第三方 CDN。</p><pre id="spec">正在加载…</pre><script>fetch('/openapi.yaml').then(r=>r.text()).then(t=>document.getElementById('spec').textContent=t).catch(()=>document.getElementById('spec').textContent='无法加载规范文件')</script></body></html>`)
}
func requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
func main() {
	dbPath := env("R34WATCH_DB", "r34watch.db")
	dist := env("R34WATCH_WEB", "web/dist")
	addr := env("R34WATCH_ADDR", ":8787")
	s, err := store.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()
	_ = s.CleanupAutomaticRaw()
	cfg, _ := s.Settings()
	if absolute, absErr := filepath.Abs(cfg.DownloadDir); absErr == nil && absolute != cfg.DownloadDir {
		cfg.DownloadDir = absolute
		_ = s.SaveSettings(cfg)
	}
	if mediaRoot := os.Getenv("R34WATCH_MEDIA_ROOT"); mediaRoot != "" {
		cfg, _ = s.Settings()
		previousRoot := filepath.Clean(cfg.DownloadDir)
		cfg.DownloadDir = filepath.Clean(mediaRoot)
		if absolute, absErr := filepath.Abs(cfg.DownloadDir); absErr == nil {
			cfg.DownloadDir = absolute
		}
		if filepath.Clean(cfg.DownloadDir) != previousRoot {
			cfg.StorageMountRequired = pathHasDistinctMount(cfg.DownloadDir) || requiresExternalMount(cfg.DownloadDir)
		}
		if rpc := os.Getenv("R34WATCH_ARIA_RPC"); rpc != "" {
			cfg.AriaRPCURL = rpc
		}
		if secret := os.Getenv("R34WATCH_ARIA_SECRET"); secret != "" {
			cfg.AriaSecret = secret
		}
		_ = s.SaveSettings(cfg)
	}
	if _, settingErr := s.Setting("storage_mount_required"); settingErr != nil {
		cfg.StorageMountRequired = pathHasDistinctMount(cfg.DownloadDir)
		_ = s.SetSetting("storage_mount_required", strconv.FormatBool(cfg.StorageMountRequired))
	}
	if cfg.BackupDir != "" {
		if _, settingErr := s.Setting("backup_mount_required"); settingErr != nil {
			cfg.BackupMountRequired = pathHasDistinctMount(cfg.BackupDir)
			_ = s.SetSetting("backup_mount_required", strconv.FormatBool(cfg.BackupMountRequired))
		}
	}
	if err = ensureStorageDirectory(cfg.DownloadDir, 0755, cfg.StorageMountRequired); err != nil {
		log.Printf("media storage unavailable: %v", err)
	}
	app := New(s)
	go app.backfillCoverFingerprints()
	app.hashNextCompletedMedia()
	notifyCtx, notifyCancel := context.WithTimeout(context.Background(), 5*time.Second)
	app.evaluateNotifications(notifyCtx)
	notifyCancel()
	app.reconcileRecovery()
	app.reconcile()
	_, _ = s.RequeueTransientFailures()
	_, _ = s.RequeueSubmitting()
	_, _ = s.RequeueDetachedWget()
	startupCfg, _ := s.Settings()
	startupCtx, startupCancel := context.WithTimeout(context.Background(), 3*time.Second)
	_ = app.aria(startupCfg).SetGlobal(startupCtx, startupCfg.MaxConcurrent, startupCfg.DownloadLimitKBPS)
	_ = app.wget.SetGlobal(startupCtx, startupCfg.MaxConcurrent, startupCfg.DownloadLimitKBPS)
	startupCancel()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	go app.monitor(ctx)
	go app.scheduler(ctx)
	app.startQueueWorker()
	server := &http.Server{Addr: addr, Handler: app.router(dist), ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, WriteTimeout: 0}
	go func() {
		log.Printf("R34Watch listening on %s", addr)
		if e := server.ListenAndServe(); e != nil && !errors.Is(e, http.ErrServerClosed) {
			log.Fatal(e)
		}
	}()
	<-ctx.Done()
	shutdownCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
	defer stop()
	_ = server.Shutdown(shutdownCtx)
}
func env(key, def string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return def
}
