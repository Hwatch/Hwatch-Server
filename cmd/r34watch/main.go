package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
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
	"syscall"
	"time"

	"golang.org/x/sys/unix"
	"r34watch/internal/adapters"
	"r34watch/internal/aria2"
	"r34watch/internal/mediainfo"
	"r34watch/internal/plex"
	"r34watch/internal/store"
)

const userAgent = "Mozilla/5.0 (compatible; r34watch/2.0; NAS media manager)"

type App struct {
	store          *store.Store
	adapters       *adapters.Registry
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
	checking       map[string]bool
	lastSchedule   string
}

func New(s *store.Store) *App {
	return &App{store: s, adapters: adapters.NewRegistry(adapters.Rule34Video{}), checking: map[string]bool{}, thumbWait: map[string]chan struct{}{}}
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
	respond(w, status, map[string]string{"error": err.Error()})
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
	return map[string]any{"id": s.ID, "slug": s.Slug, "platform": s.Platform, "subscription_type": s.Type, "source_id": s.SourceID, "name": s.Name, "url": s.URL, "avatar_url": s.AvatarURL, "external_links": jsonValue(s.ExternalLinks), "enabled": s.Enabled, "baseline_complete": s.BaselineComplete, "created_at": s.CreatedAt, "last_checked_at": s.LastCheckedAt, "last_error": s.LastError}
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
	return map[string]any{"video_id": v.ID, "subscription_slug": v.SubscriptionSlug, "title": v.Title, "page_url": v.PageURL, "published_at": v.PublishedAt, "discovered_at": v.DiscoveredAt, "thumbnail_url": v.ThumbnailURL, "views": v.Views, "duration_seconds": v.DurationSeconds, "creator_name": v.CreatorName, "creator_url": v.CreatorURL, "creator_avatar": v.CreatorAvatar, "uploader_name": v.UploaderName, "uploader_url": v.UploaderURL, "uploader_avatar": v.UploaderAvatar, "categories": jsonValue(v.CategoriesJSON), "tags": jsonValue(v.TagsJSON), "external_links": jsonValue(v.ExternalLinksJSON), "status": v.Status, "aria_gid": v.AriaGID, "file_path": v.FilePath, "plex_path": v.PlexPath, "bytes_downloaded": v.BytesDownloaded, "total_bytes": v.TotalBytes, "download_speed": v.DownloadSpeed, "connections": v.Connections, "video_codec": v.VideoCodec, "audio_codec": v.AudioCodec, "container": v.Container, "width": v.Width, "height": v.Height, "frame_rate": v.FrameRate, "bit_rate": v.BitRate, "bit_depth": v.BitDepth, "media_probed_at": v.MediaProbedAt, "transcode_status": v.TranscodeStatus, "plex_season": v.PlexSeason, "plex_episode": v.PlexEpisode, "error": v.Error, "downloaded_at": v.DownloadedAt, "updated_at": v.UpdatedAt}
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

func (a *App) aria(cfg store.Settings) *aria2.Client {
	return aria2.New(cfg.AriaRPCURL, cfg.AriaSecret)
}
func (a *App) ariaOnline(cfg store.Settings) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return a.aria(cfg).Version(ctx) == nil
}
func diskStats(path string) (total, free int64) {
	if path == "" {
		path = "."
	}
	_ = os.MkdirAll(path, 0755)
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
	waiting += queued
	completed, _ := a.store.CountVideos("completed")
	discovered, _ := a.store.CountVideos("discovered")
	traffic, library, _ := a.store.Stats()
	downloadSpeed, connections, _ := a.store.TransferStats()
	cfg, _ := a.store.Settings()
	diskTotal, diskFree := diskStats(cfg.DownloadDir)
	respond(w, 200, map[string]any{"subscriptions": a.subscriptionMaps(subs), "videos": translatedVideoMaps(videos, language, translations), "recently_played": translatedVideoMaps(recentlyPlayed, language, translations), "counts": map[string]any{"videos": totalCount, "active": activeCount, "waiting": waiting, "completed": completed, "discovered": discovered}, "traffic_bytes": traffic, "library_bytes": library, "current_download_speed": downloadSpeed, "current_connections": connections, "disk_total": diskTotal, "disk_free": diskFree, "aria2_online": a.ariaOnline(cfg), "timezone": cfg.Timezone})
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
func (a *App) startQueueWorker() {
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
			if wait := a.rateLimitWait(); wait > 0 {
				log.Printf("download queue paused for %s after HTTP 429", wait.Round(time.Second))
				time.Sleep(wait)
			}
			videos, err := a.store.Videos(1, 0, "queued", "")
			if err != nil || len(videos) == 0 {
				return
			}
			video := videos[0]
			if err = a.queue(video.ID); err != nil {
				log.Printf("persistent queue %s: %v", video.ID, err)
				if strings.HasPrefix(err.Error(), "excluded: ") {
					continue
				}
				if isRateLimited(err.Error()) || strings.Contains(err.Error(), "deadline exceeded") || strings.Contains(err.Error(), "Timeout") {
					_ = a.store.UpdateTask(video.ID, "queued", "", "", "", err.Error(), 0, 0)
					if isRateLimited(err.Error()) {
						a.record429()
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
	assignString(input, "http_proxy", &cfg.HTTPProxy)
	assignString(input, "download_dir", &cfg.DownloadDir)
	assignString(input, "aria_rpc_url", &cfg.AriaRPCURL)
	assignString(input, "aria_secret", &cfg.AriaSecret)
	assignString(input, "check_time", &cfg.CheckTime)
	assignString(input, "timezone", &cfg.Timezone)
	assignString(input, "plex_language", &cfg.PlexLanguage)
	assignString(input, "exclude_title_keywords", &cfg.ExcludeTitleKeywords)
	assignString(input, "exclude_uploaders", &cfg.ExcludeUploaders)
	assignString(input, "exclude_categories", &cfg.ExcludeCategories)
	assignString(input, "exclude_tags", &cfg.ExcludeTags)
	assignBool(input, "require_creator_match", &cfg.RequireCreatorMatch)
	assignInt(input, "max_concurrent", &cfg.MaxConcurrent)
	assignInt(input, "connections_per_download", &cfg.ConnectionsPerDownload)
	assignInt64(input, "download_limit_kbps", &cfg.DownloadLimitKBPS)
	assignInt64(input, "min_free_bytes", &cfg.MinFreeBytes)
	assignInt64(input, "max_duration_seconds", &cfg.MaxDurationSeconds)
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
	if absolute, absErr := filepath.Abs(cfg.DownloadDir); absErr == nil {
		cfg.DownloadDir = absolute
	}
	if err = a.store.SaveSettings(cfg); err != nil {
		fail(w, 500, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	_ = a.aria(cfg).SetGlobal(ctx, cfg.MaxConcurrent, cfg.DownloadLimitKBPS)
	cancel()
	respond(w, 200, settingsMap(cfg))
}
func settingsMap(c store.Settings) map[string]any {
	return map[string]any{"http_proxy": c.HTTPProxy, "download_dir": c.DownloadDir, "aria_rpc_url": c.AriaRPCURL, "aria_secret": c.AriaSecret, "check_time": c.CheckTime, "timezone": c.Timezone, "plex_language": c.PlexLanguage, "max_concurrent": c.MaxConcurrent, "connections_per_download": c.ConnectionsPerDownload, "download_limit_kbps": c.DownloadLimitKBPS, "min_free_bytes": c.MinFreeBytes, "require_creator_match": c.RequireCreatorMatch, "exclude_title_keywords": c.ExcludeTitleKeywords, "exclude_uploaders": c.ExcludeUploaders, "exclude_categories": c.ExcludeCategories, "exclude_tags": c.ExcludeTags, "max_duration_seconds": c.MaxDurationSeconds}
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
		if err := a.store.DeleteSubscription(slug); err != nil {
			fail(w, 500, err)
			return
		}
		respond(w, 200, map[string]bool{"deleted": true})
	}
}
func (a *App) triggerCheck(slug string) {
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
			if created || existing.DurationSeconds == 0 {
				if detailed, e := adapter.FetchDetails(context.Background(), item, cfg.HTTPProxy); e == nil {
					detailed.SubscriptionSlug = slug
					_ = a.store.UpdateDetails(detailed)
					if strings.TrimRight(detailed.CreatorURL, "/") == strings.TrimRight(sub.URL, "/") {
						_ = a.store.UpdateSubscriptionMetadata(slug, detailed.CreatorName, detailed.CreatorAvatar, detailed.ExternalLinksJSON)
					}
				}
			}
			if created {
				if a.store.QueueVideo(item.ID) == nil {
					a.startQueueWorker()
				}
			}
		}
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
	a.queueMu.Lock()
	defer a.queueMu.Unlock()
	v, err := a.store.Video(id)
	if err != nil {
		return err
	}
	if v.Status == "active" || v.Status == "waiting" || v.Status == "paused" {
		return errors.New("任务已存在")
	}
	cfg, _ := a.store.Settings()
	_, free := diskStats(cfg.DownloadDir)
	if cfg.MinFreeBytes > 0 && free < cfg.MinFreeBytes {
		return errors.New("磁盘可用空间低于保留值")
	}
	if v.DurationSeconds == 0 || v.CreatorURL == "" {
		adapter, _ := a.adapters.ByID(v.Platform)
		if detail, e := adapter.FetchDetails(context.Background(), v, cfg.HTTPProxy); e == nil {
			v = detail
			_ = a.store.UpdateDetails(v)
		}
	}
	sub, err := a.store.Subscription(v.SubscriptionSlug)
	if err != nil {
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
		return err
	}
	layout := plex.Build(cfg.DownloadDir, sub, v)
	if err = os.MkdirAll(layout.SeasonDir, 0755); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	gid, err := a.aria(cfg).Add(ctx, resolved.URL, layout.SeasonDir, layout.FileName, v.PageURL, userAgent, cfg.HTTPProxy, cfg.ConnectionsPerDownload)
	if err != nil {
		_ = a.store.UpdateTask(id, "failed", "", "", "", err.Error(), 0, 0)
		return err
	}
	return a.store.UpdateTask(id, "waiting", gid, "", layout.FullPath, "", 0, 0)
}
func exclusionReason(v store.Video, sub store.Subscription, cfg store.Settings) string {
	creatorURL := strings.TrimRight(strings.ToLower(v.CreatorURL), "/")
	subURL := strings.TrimRight(strings.ToLower(sub.URL), "/")
	if cfg.RequireCreatorMatch && creatorURL != "" && subURL != "" && creatorURL != subURL {
		return "实际创作者与订阅作者不一致: " + v.CreatorName
	}
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
	client := a.aria(cfg)
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	switch action {
	case "download", "retry":
		err = a.store.QueueVideo(id)
		if err == nil {
			a.startQueueWorker()
		}
	case "pause":
		err = client.Pause(ctx, v.AriaGID)
		if err == nil {
			err = a.store.UpdateTask(id, "paused", v.AriaGID, v.FilePath, v.PlexPath, "", v.BytesDownloaded, v.TotalBytes)
		}
	case "resume":
		err = client.Resume(ctx, v.AriaGID)
	case "delete":
		if v.AriaGID != "" {
			_ = client.Remove(ctx, v.AriaGID)
		}
		if v.PlexPath != "" {
			_ = os.Remove(v.PlexPath)
			base := strings.TrimSuffix(v.PlexPath, filepath.Ext(v.PlexPath))
			_ = os.Remove(base + ".nfo")
			_ = os.Remove(base + ".jpg")
			_ = os.Remove(v.PlexPath + ".aria2")
		}
		err = a.store.UpdateTask(id, "discovered", "", "", "", "", 0, 0)
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
	cfg, _ := a.store.Settings()
	client := a.aria(cfg)
	for _, state := range []string{"waiting", "active", "paused"} {
		items, _ := a.store.Videos(500, 0, state, "")
		for _, v := range items {
			callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			status, err := client.Tell(callCtx, v.AriaGID)
			cancel()
			if err != nil {
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
			if mapped == "failed" && isRateLimited(errText) {
				removeCtx, removeCancel := context.WithTimeout(ctx, 5*time.Second)
				_ = client.Remove(removeCtx, v.AriaGID)
				removeCancel()
				a.record429()
				_ = a.store.UpdateTask(v.ID, "queued", "", path, v.PlexPath, errText, status.Completed, status.Total)
				a.startQueueWorker()
				continue
			}
			_ = a.store.UpdateTask(v.ID, mapped, v.AriaGID, path, v.PlexPath, errText, status.Completed, status.Total)
			_ = a.store.UpdateTransfer(v.ID, status.DownloadSpeed, status.Connections)
			if mapped == "completed" {
				fresh, _ := a.store.Video(v.ID)
				sub, _ := a.store.Subscription(v.SubscriptionSlug)
				layout := plex.Build(cfg.DownloadDir, sub, fresh)
				translated := a.translatedMetadata(fresh, cfg.PlexLanguage)
				_ = plex.Generate(layout, sub, translated, cfg.HTTPProxy)
				a.probeMedia(v.ID, path)
			}
		}
	}
}
func (a *App) scheduler(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
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
func staticHandler(dist string) http.Handler {
	files := http.FileServer(http.Dir(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			target := filepath.Join(dist, filepath.Clean(r.URL.Path))
			if info, err := os.Stat(target); err == nil && !info.IsDir() {
				if kind := mime.TypeByExtension(filepath.Ext(target)); kind != "" {
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
	mux.HandleFunc("GET /api/dashboard", a.dashboard)
	mux.HandleFunc("GET /api/videos", a.listVideos)
	mux.HandleFunc("GET /api/videos/facets", a.videoFacets)
	mux.HandleFunc("GET /api/tasks", a.listTasks)
	mux.HandleFunc("GET /api/translations/export", a.exportTranslations)
	mux.HandleFunc("POST /api/translations/import", a.importTranslations)
	mux.HandleFunc("POST /api/plex/refresh-nfo", a.refreshNFO)
	mux.HandleFunc("POST /api/videos/download-all", a.downloadAll)
	mux.HandleFunc("GET /api/settings", a.settings)
	mux.HandleFunc("PUT /api/settings", a.settings)
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
	mux.HandleFunc("GET /api/events", a.events)
	mux.Handle("/", staticHandler(dist))
	return requestLog(mux)
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
		cfg.DownloadDir = mediaRoot
		if rpc := os.Getenv("R34WATCH_ARIA_RPC"); rpc != "" {
			cfg.AriaRPCURL = rpc
		}
		if secret := os.Getenv("R34WATCH_ARIA_SECRET"); secret != "" {
			cfg.AriaSecret = secret
		}
		_ = s.SaveSettings(cfg)
	}
	app := New(s)
	app.reconcile()
	_, _ = s.RequeueTransientFailures()
	startupCfg, _ := s.Settings()
	startupCtx, startupCancel := context.WithTimeout(context.Background(), 3*time.Second)
	_ = app.aria(startupCfg).SetGlobal(startupCtx, startupCfg.MaxConcurrent, startupCfg.DownloadLimitKBPS)
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
