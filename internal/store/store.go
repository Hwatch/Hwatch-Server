package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB   *sql.DB
	Path string
}
type Translation struct{ EntityID, Language, Field, SourceHash, Text, Provider, UpdatedAt string }
type SubscriptionProgress struct {
	Total, Completed, Active, Waiting, Failed int64
	BytesDownloaded, TotalBytes               int64
}
type Subscription struct {
	ID                                                                             int64
	Slug, Name, URL, AvatarURL, ExternalLinks, CreatedAt, LastCheckedAt, LastError string
	Platform, Type, SourceID, AdapterConfig, Cursor                                string
	Enabled, BaselineComplete                                                      bool
}
type Video struct {
	ID, SubscriptionSlug, Title, PageURL, PublishedAt, DiscoveredAt, ThumbnailURL          string
	Platform, SourceID                                                                     string
	Views, DurationSeconds                                                                 int64
	CreatorName, CreatorURL, CreatorAvatar, UploaderName, UploaderURL, UploaderAvatar      string
	CategoriesJSON, TagsJSON, ExternalLinksJSON, CreatorsJSON, SourceRelationJSON, RawHTML string
	Status, AriaGID, FilePath, PlexPath, Error, DownloadedAt, UpdatedAt                    string
	VideoCodec, AudioCodec, Container, MediaProbedAt, TranscodeStatus                      string
	BytesDownloaded, TotalBytes, DownloadSpeed, BitRate                                    int64
	FrameRate                                                                              float64
	PlexSeason, PlexEpisode, Connections, Width, Height, BitDepth                          int
}
type Settings struct {
	HTTPProxy, DownloadDir, AriaRPCURL, AriaSecret, CheckTime, Timezone, PlexLanguage string
	BackupDir, BackupTime, BackupLastSuccess, BackupLastError                         string
	ExcludeTitleKeywords, ExcludeUploaders, ExcludeCategories, ExcludeTags            string
	MaxConcurrent, ConnectionsPerDownload                                             int
	BackupRetention                                                                   int
	MinFreeBytes, DownloadLimitKBPS                                                   int64
	MaxDurationSeconds                                                                int64
	RequireCreatorMatch, DownloadPaused                                               bool
}
type VideoSearch struct {
	Status, Query, Scope, Language, TagMode, Creator, Codec, Sort, Order string
	Tags, ExcludeTags                                                    []string
	MinHeight                                                            int
}
type VideoFacet struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

const schema = "PRAGMA journal_mode=WAL;" +
	"PRAGMA foreign_keys=ON;" +
	"CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY,value TEXT NOT NULL);" +
	"CREATE TABLE IF NOT EXISTS subscriptions (" +
	"id INTEGER PRIMARY KEY AUTOINCREMENT,slug TEXT NOT NULL UNIQUE,name TEXT NOT NULL,url TEXT NOT NULL UNIQUE," +
	"platform TEXT NOT NULL DEFAULT 'rule34video',subscription_type TEXT NOT NULL DEFAULT 'creator',source_id TEXT NOT NULL DEFAULT '',adapter_config TEXT NOT NULL DEFAULT '{}',cursor_json TEXT NOT NULL DEFAULT '{}'," +
	"avatar_url TEXT NOT NULL DEFAULT '',external_links TEXT NOT NULL DEFAULT '[]',enabled INTEGER NOT NULL DEFAULT 1," +
	"baseline_complete INTEGER NOT NULL DEFAULT 0,created_at TEXT NOT NULL,last_checked_at TEXT NOT NULL DEFAULT '',last_error TEXT NOT NULL DEFAULT '');" +
	"CREATE TABLE IF NOT EXISTS videos (" +
	"id TEXT PRIMARY KEY,subscription_slug TEXT NOT NULL,title TEXT NOT NULL,page_url TEXT NOT NULL,published_at TEXT NOT NULL DEFAULT ''," +
	"platform TEXT NOT NULL DEFAULT 'rule34video',source_id TEXT NOT NULL DEFAULT ''," +
	"discovered_at TEXT NOT NULL,thumbnail_url TEXT NOT NULL DEFAULT '',views INTEGER NOT NULL DEFAULT 0,duration_seconds INTEGER NOT NULL DEFAULT 0," +
	"creator_name TEXT NOT NULL DEFAULT '',creator_url TEXT NOT NULL DEFAULT '',creator_avatar TEXT NOT NULL DEFAULT ''," +
	"uploader_name TEXT NOT NULL DEFAULT '',uploader_url TEXT NOT NULL DEFAULT '',uploader_avatar TEXT NOT NULL DEFAULT ''," +
	"categories_json TEXT NOT NULL DEFAULT '[]',tags_json TEXT NOT NULL DEFAULT '[]',external_links_json TEXT NOT NULL DEFAULT '[]',creators_json TEXT NOT NULL DEFAULT '[]',source_relation_json TEXT NOT NULL DEFAULT '{}'," +
	"raw_html TEXT NOT NULL DEFAULT '',status TEXT NOT NULL DEFAULT 'discovered',aria_gid TEXT NOT NULL DEFAULT ''," +
	"file_path TEXT NOT NULL DEFAULT '',plex_path TEXT NOT NULL DEFAULT '',bytes_downloaded INTEGER NOT NULL DEFAULT 0,total_bytes INTEGER NOT NULL DEFAULT 0,download_speed INTEGER NOT NULL DEFAULT 0,connections INTEGER NOT NULL DEFAULT 0," +
	"video_codec TEXT NOT NULL DEFAULT '',audio_codec TEXT NOT NULL DEFAULT '',container TEXT NOT NULL DEFAULT '',width INTEGER NOT NULL DEFAULT 0,height INTEGER NOT NULL DEFAULT 0,frame_rate REAL NOT NULL DEFAULT 0,bit_rate INTEGER NOT NULL DEFAULT 0,bit_depth INTEGER NOT NULL DEFAULT 0,media_probed_at TEXT NOT NULL DEFAULT '',transcode_status TEXT NOT NULL DEFAULT ''," +
	"plex_season INTEGER NOT NULL DEFAULT 1,plex_episode INTEGER NOT NULL DEFAULT 0,error TEXT NOT NULL DEFAULT '',downloaded_at TEXT NOT NULL DEFAULT '',updated_at TEXT NOT NULL," +
	"FOREIGN KEY(subscription_slug) REFERENCES subscriptions(slug) ON DELETE CASCADE);" +
	"CREATE INDEX IF NOT EXISTS idx_videos_subscription ON videos(subscription_slug);" +
	"CREATE INDEX IF NOT EXISTS idx_videos_status ON videos(status);" +
	"CREATE TABLE IF NOT EXISTS traffic_events (id INTEGER PRIMARY KEY AUTOINCREMENT,video_id TEXT NOT NULL,bytes INTEGER NOT NULL,recorded_at TEXT NOT NULL);" +
	"CREATE TABLE IF NOT EXISTS imports (id INTEGER PRIMARY KEY AUTOINCREMENT,source TEXT NOT NULL,raw_html TEXT NOT NULL,video_id TEXT NOT NULL DEFAULT '',created_at TEXT NOT NULL,error TEXT NOT NULL DEFAULT '');" +
	"CREATE TABLE IF NOT EXISTS translations (entity_id TEXT NOT NULL,language TEXT NOT NULL,field TEXT NOT NULL,source_hash TEXT NOT NULL,translated_text TEXT NOT NULL,provider TEXT NOT NULL DEFAULT '',updated_at TEXT NOT NULL,PRIMARY KEY(entity_id,language,field));" +
	"CREATE TABLE IF NOT EXISTS playback_history (video_id TEXT PRIMARY KEY,play_count INTEGER NOT NULL DEFAULT 1,last_played_at TEXT NOT NULL,FOREIGN KEY(video_id) REFERENCES videos(id) ON DELETE CASCADE);" +
	"CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY AUTOINCREMENT,username TEXT NOT NULL UNIQUE,password_hash TEXT NOT NULL,created_at TEXT NOT NULL);" +
	"CREATE TABLE IF NOT EXISTS sessions (token_hash TEXT PRIMARY KEY,user_id INTEGER NOT NULL,csrf_token TEXT NOT NULL,created_at TEXT NOT NULL,expires_at TEXT NOT NULL,last_seen_at TEXT NOT NULL,FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE);" +
	"CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);"

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err = db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, err
	}
	s := &Store{DB: db, Path: path}
	// Additive migrations keep existing installations readable while new clean databases
	// get the same columns directly from schema.
	_, _ = db.Exec("ALTER TABLE videos ADD COLUMN download_speed INTEGER NOT NULL DEFAULT 0")
	_, _ = db.Exec("ALTER TABLE videos ADD COLUMN connections INTEGER NOT NULL DEFAULT 0")
	for _, statement := range []string{
		"ALTER TABLE subscriptions ADD COLUMN platform TEXT NOT NULL DEFAULT 'rule34video'",
		"ALTER TABLE subscriptions ADD COLUMN subscription_type TEXT NOT NULL DEFAULT 'creator'",
		"ALTER TABLE subscriptions ADD COLUMN source_id TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE subscriptions ADD COLUMN adapter_config TEXT NOT NULL DEFAULT '{}'",
		"ALTER TABLE subscriptions ADD COLUMN cursor_json TEXT NOT NULL DEFAULT '{}'",
		"ALTER TABLE videos ADD COLUMN platform TEXT NOT NULL DEFAULT 'rule34video'",
		"ALTER TABLE videos ADD COLUMN source_id TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE videos ADD COLUMN video_codec TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE videos ADD COLUMN audio_codec TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE videos ADD COLUMN container TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE videos ADD COLUMN width INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE videos ADD COLUMN height INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE videos ADD COLUMN frame_rate REAL NOT NULL DEFAULT 0",
		"ALTER TABLE videos ADD COLUMN bit_rate INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE videos ADD COLUMN bit_depth INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE videos ADD COLUMN media_probed_at TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE videos ADD COLUMN transcode_status TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE videos ADD COLUMN creators_json TEXT NOT NULL DEFAULT '[]'",
		"ALTER TABLE videos ADD COLUMN source_relation_json TEXT NOT NULL DEFAULT '{}'",
	} {
		_, _ = db.Exec(statement)
	}
	_, _ = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_source ON subscriptions(platform,subscription_type,source_id) WHERE source_id<>''")
	_, _ = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_videos_source ON videos(platform,source_id) WHERE source_id<>''")
	defaults := map[string]string{"http_proxy": "", "download_dir": "media", "aria_rpc_url": "http://127.0.0.1:6800/jsonrpc", "aria_secret": "", "check_time": "04:00", "timezone": "Asia/Shanghai", "plex_language": "zh-CN", "max_concurrent": "2", "connections_per_download": "4", "download_limit_kbps": "0", "min_free_bytes": "5368709120", "require_creator_match": "true", "download_paused": "false", "exclude_title_keywords": "compilation,collection,合集,集锦", "exclude_uploaders": "", "exclude_categories": "", "exclude_tags": "", "max_duration_seconds": "0", "backup_dir": "", "backup_time": "03:30", "backup_retention": "7", "backup_last_success": "", "backup_last_error": ""}
	for key, value := range defaults {
		_, _ = db.Exec("INSERT OR IGNORE INTO settings(key,value) VALUES(?,?)", key, value)
	}
	return s, nil
}
func Now() string             { return time.Now().UTC().Format(time.RFC3339) }
func (s *Store) Close() error { return s.DB.Close() }
func (s *Store) CleanupAutomaticRaw() error {
	var count int
	if err := s.DB.QueryRow("SELECT count(*) FROM videos WHERE raw_html<>''").Scan(&count); err != nil || count == 0 {
		return err
	}
	if _, err := s.DB.Exec("UPDATE videos SET raw_html=''"); err != nil {
		return err
	}
	_, err := s.DB.Exec("VACUUM")
	return err
}
func (s *Store) Settings() (Settings, error) {
	rows, err := s.DB.Query("SELECT key,value FROM settings")
	if err != nil {
		return Settings{}, err
	}
	defer rows.Close()
	values := map[string]string{}
	for rows.Next() {
		var k, v string
		if err = rows.Scan(&k, &v); err != nil {
			return Settings{}, err
		}
		values[k] = v
	}
	var cfg Settings
	cfg.HTTPProxy = values["http_proxy"]
	cfg.DownloadDir = values["download_dir"]
	if cfg.DownloadDir == "" {
		cfg.DownloadDir = values["media_root"]
	}
	cfg.AriaRPCURL = values["aria_rpc_url"]
	cfg.AriaSecret = values["aria_secret"]
	cfg.CheckTime = values["check_time"]
	cfg.Timezone = values["timezone"]
	cfg.PlexLanguage = values["plex_language"]
	cfg.BackupDir = values["backup_dir"]
	cfg.BackupTime = values["backup_time"]
	cfg.BackupLastSuccess = values["backup_last_success"]
	cfg.BackupLastError = values["backup_last_error"]
	cfg.ExcludeTitleKeywords = values["exclude_title_keywords"]
	cfg.ExcludeUploaders = values["exclude_uploaders"]
	cfg.ExcludeCategories = values["exclude_categories"]
	cfg.ExcludeTags = values["exclude_tags"]
	cfg.RequireCreatorMatch = strings.EqualFold(values["require_creator_match"], "true")
	cfg.DownloadPaused = strings.EqualFold(values["download_paused"], "true")
	_, _ = fmt.Sscan(values["max_concurrent"], &cfg.MaxConcurrent)
	_, _ = fmt.Sscan(values["connections_per_download"], &cfg.ConnectionsPerDownload)
	_, _ = fmt.Sscan(values["download_limit_kbps"], &cfg.DownloadLimitKBPS)
	_, _ = fmt.Sscan(values["min_free_bytes"], &cfg.MinFreeBytes)
	_, _ = fmt.Sscan(values["max_duration_seconds"], &cfg.MaxDurationSeconds)
	_, _ = fmt.Sscan(values["backup_retention"], &cfg.BackupRetention)
	return cfg, nil
}
func (s *Store) SetDownloadPaused(paused bool) error {
	_, err := s.DB.Exec("INSERT INTO settings(key,value) VALUES('download_paused',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", fmt.Sprint(paused))
	return err
}
func (s *Store) SaveSettings(cfg Settings) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	values := map[string]any{"http_proxy": cfg.HTTPProxy, "download_dir": cfg.DownloadDir, "aria_rpc_url": cfg.AriaRPCURL, "aria_secret": cfg.AriaSecret, "check_time": cfg.CheckTime, "timezone": cfg.Timezone, "plex_language": cfg.PlexLanguage, "max_concurrent": cfg.MaxConcurrent, "connections_per_download": cfg.ConnectionsPerDownload, "download_limit_kbps": cfg.DownloadLimitKBPS, "min_free_bytes": cfg.MinFreeBytes, "require_creator_match": cfg.RequireCreatorMatch, "exclude_title_keywords": cfg.ExcludeTitleKeywords, "exclude_uploaders": cfg.ExcludeUploaders, "exclude_categories": cfg.ExcludeCategories, "exclude_tags": cfg.ExcludeTags, "max_duration_seconds": cfg.MaxDurationSeconds, "backup_dir": cfg.BackupDir, "backup_time": cfg.BackupTime, "backup_retention": cfg.BackupRetention}
	for k, v := range values {
		if _, err = tx.Exec("INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", k, fmt.Sprint(v)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) SetBackupResult(success bool, message string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if success {
		_, err = tx.Exec("INSERT INTO settings(key,value) VALUES('backup_last_success',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", Now())
		if err == nil {
			_, err = tx.Exec("INSERT INTO settings(key,value) VALUES('backup_last_error','') ON CONFLICT(key) DO UPDATE SET value='' ")
		}
	} else {
		_, err = tx.Exec("INSERT INTO settings(key,value) VALUES('backup_last_error',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", message)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) AddSubscription(sub Subscription) error {
	_, err := s.DB.Exec("INSERT INTO subscriptions(slug,name,url,platform,subscription_type,source_id,adapter_config,cursor_json,avatar_url,external_links,enabled,baseline_complete,created_at,last_checked_at,last_error) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", sub.Slug, sub.Name, sub.URL, sub.Platform, sub.Type, sub.SourceID, sub.AdapterConfig, sub.Cursor, sub.AvatarURL, sub.ExternalLinks, boolInt(sub.Enabled), boolInt(sub.BaselineComplete), sub.CreatedAt, sub.LastCheckedAt, sub.LastError)
	return err
}
func (s *Store) Subscriptions() ([]Subscription, error) {
	rows, err := s.DB.Query("SELECT id,slug,name,url,platform,subscription_type,source_id,adapter_config,cursor_json,avatar_url,external_links,enabled,baseline_complete,created_at,last_checked_at,last_error FROM subscriptions ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Subscription{}
	for rows.Next() {
		var x Subscription
		var enabled, baseline int
		if err = rows.Scan(&x.ID, &x.Slug, &x.Name, &x.URL, &x.Platform, &x.Type, &x.SourceID, &x.AdapterConfig, &x.Cursor, &x.AvatarURL, &x.ExternalLinks, &enabled, &baseline, &x.CreatedAt, &x.LastCheckedAt, &x.LastError); err != nil {
			return nil, err
		}
		x.Enabled = enabled != 0
		x.BaselineComplete = baseline != 0
		out = append(out, x)
	}
	return out, rows.Err()
}
func (s *Store) Subscription(slug string) (Subscription, error) {
	var x Subscription
	var enabled, baseline int
	err := s.DB.QueryRow("SELECT id,slug,name,url,platform,subscription_type,source_id,adapter_config,cursor_json,avatar_url,external_links,enabled,baseline_complete,created_at,last_checked_at,last_error FROM subscriptions WHERE slug=?", slug).Scan(&x.ID, &x.Slug, &x.Name, &x.URL, &x.Platform, &x.Type, &x.SourceID, &x.AdapterConfig, &x.Cursor, &x.AvatarURL, &x.ExternalLinks, &enabled, &baseline, &x.CreatedAt, &x.LastCheckedAt, &x.LastError)
	x.Enabled = enabled != 0
	x.BaselineComplete = baseline != 0
	return x, err
}
func (s *Store) UpdateSubscriptionCheck(slug, lastError string) error {
	_, err := s.DB.Exec("UPDATE subscriptions SET last_checked_at=?,last_error=?,baseline_complete=1 WHERE slug=?", Now(), lastError, slug)
	return err
}
func (s *Store) UpdateSubscriptionMetadata(slug, name, avatar, links string) error {
	if name == "" {
		name = slug
	}
	_, err := s.DB.Exec("UPDATE subscriptions SET name=?,avatar_url=CASE WHEN ?='' THEN avatar_url ELSE ? END,external_links=CASE WHEN ?='' OR ?='[]' THEN external_links ELSE ? END WHERE slug=?", name, avatar, avatar, links, links, links, slug)
	return err
}
func (s *Store) SetSubscriptionEnabled(slug string, enabled bool) error {
	res, err := s.DB.Exec("UPDATE subscriptions SET enabled=? WHERE slug=?", boolInt(enabled), slug)
	if err == nil {
		n, _ := res.RowsAffected()
		if n == 0 {
			return sql.ErrNoRows
		}
	}
	return err
}
func (s *Store) DeleteSubscription(slug string) error {
	_, err := s.DB.Exec("DELETE FROM subscriptions WHERE slug=?", slug)
	return err
}
func (s *Store) UpsertDiscovered(v Video) (bool, error) {
	if v.DiscoveredAt == "" {
		v.DiscoveredAt = Now()
	}
	if v.UpdatedAt == "" {
		v.UpdatedAt = Now()
	}
	res, err := s.DB.Exec("INSERT OR IGNORE INTO videos(id,subscription_slug,platform,source_id,title,page_url,discovered_at,thumbnail_url,source_relation_json,status,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)", v.ID, v.SubscriptionSlug, v.Platform, v.SourceID, v.Title, v.PageURL, v.DiscoveredAt, v.ThumbnailURL, v.SourceRelationJSON, "discovered", v.UpdatedAt)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	if n == 0 && v.SourceRelationJSON != "" && v.SourceRelationJSON != "{}" {
		_, err = s.DB.Exec("UPDATE videos SET source_relation_json=?,updated_at=? WHERE id=?", v.SourceRelationJSON, Now(), v.ID)
	}
	return n > 0, nil
}
func scanVideo(scanner interface{ Scan(...any) error }) (Video, error) {
	var v Video
	err := scanner.Scan(&v.ID, &v.SubscriptionSlug, &v.Platform, &v.SourceID, &v.Title, &v.PageURL, &v.PublishedAt, &v.DiscoveredAt, &v.ThumbnailURL, &v.Views, &v.DurationSeconds, &v.CreatorName, &v.CreatorURL, &v.CreatorAvatar, &v.UploaderName, &v.UploaderURL, &v.UploaderAvatar, &v.CategoriesJSON, &v.TagsJSON, &v.ExternalLinksJSON, &v.CreatorsJSON, &v.SourceRelationJSON, &v.RawHTML, &v.Status, &v.AriaGID, &v.FilePath, &v.PlexPath, &v.BytesDownloaded, &v.TotalBytes, &v.DownloadSpeed, &v.Connections, &v.VideoCodec, &v.AudioCodec, &v.Container, &v.Width, &v.Height, &v.FrameRate, &v.BitRate, &v.BitDepth, &v.MediaProbedAt, &v.TranscodeStatus, &v.PlexSeason, &v.PlexEpisode, &v.Error, &v.DownloadedAt, &v.UpdatedAt)
	return v, err
}

const videoColumns = "id,subscription_slug,platform,source_id,title,page_url,published_at,discovered_at,thumbnail_url,views,duration_seconds,creator_name,creator_url,creator_avatar,uploader_name,uploader_url,uploader_avatar,categories_json,tags_json,external_links_json,creators_json,source_relation_json,raw_html,status,aria_gid,file_path,plex_path,bytes_downloaded,total_bytes,download_speed,connections,video_codec,audio_codec,container,width,height,frame_rate,bit_rate,bit_depth,media_probed_at,transcode_status,plex_season,plex_episode,error,downloaded_at,updated_at"

func (s *Store) Video(id string) (Video, error) {
	return scanVideo(s.DB.QueryRow("SELECT "+videoColumns+" FROM videos WHERE id=?", id))
}
func (s *Store) Videos(limit, offset int, status, query string) ([]Video, error) {
	where := " WHERE 1=1"
	args := []any{}
	if status != "" {
		where += " AND status=?"
		args = append(args, status)
	}
	if query != "" {
		where += " AND (title LIKE ? OR subscription_slug LIKE ?)"
		q := "%" + query + "%"
		args = append(args, q, q)
	}
	args = append(args, limit, offset)
	rows, err := s.DB.Query("SELECT "+videoColumns+" FROM videos"+where+" ORDER BY discovered_at DESC,id DESC LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Video{}
	for rows.Next() {
		v, e := scanVideo(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) SearchVideos(limit, offset int, opts VideoSearch) ([]Video, int64, error) {
	where, args := videoSearchWhere(opts)
	sortColumns := map[string]string{"downloaded_at": "v.downloaded_at", "published_at": "v.published_at", "views": "v.views", "duration": "v.duration_seconds", "size": "v.total_bytes", "resolution": "v.height", "bit_rate": "v.bit_rate", "title": "COALESCE(tt.translated_text,v.title)", "recent_played": "COALESCE(ph.last_played_at,'')", "play_count": "COALESCE(ph.play_count,0)"}
	sortColumn := sortColumns[opts.Sort]
	if sortColumn == "" {
		sortColumn = "v.downloaded_at"
	}
	direction := "DESC"
	if strings.EqualFold(opts.Order, "asc") {
		direction = "ASC"
	}
	joins := " LEFT JOIN translations tt ON tt.entity_id=v.id AND tt.field='title' AND tt.language=? LEFT JOIN playback_history ph ON ph.video_id=v.id"
	queryArgs := append([]any{opts.Language}, args...)
	rows, err := s.DB.Query("SELECT "+prefixedVideoColumns("v")+" FROM videos v"+joins+where+" ORDER BY "+sortColumn+" "+direction+",v.id DESC LIMIT ? OFFSET ?", append(queryArgs, limit, offset)...)
	items, err := scanVideos(rows, err)
	if err != nil {
		return nil, 0, err
	}
	var total int64
	err = s.DB.QueryRow("SELECT count(*) FROM videos v"+joins+where, queryArgs...).Scan(&total)
	return items, total, err
}
func videoSearchWhere(opts VideoSearch) (string, []any) {
	where := " WHERE 1=1"
	args := []any{}
	if opts.Status != "" {
		where += " AND v.status=?"
		args = append(args, opts.Status)
	}
	if q := strings.TrimSpace(opts.Query); q != "" {
		like := "%" + q + "%"
		original := "(v.title LIKE ? OR v.creator_name LIKE ? OR v.uploader_name LIKE ? OR v.categories_json LIKE ? OR v.tags_json LIKE ? OR v.id LIKE ?)"
		translated := "(tt.translated_text LIKE ? OR EXISTS(SELECT 1 FROM translations tq WHERE tq.entity_id=v.id AND tq.language=? AND tq.field='tags' AND tq.translated_text LIKE ?))"
		switch opts.Scope {
		case "original":
			where += " AND " + original
			for range 6 {
				args = append(args, like)
			}
		case "translated":
			where += " AND " + translated
			args = append(args, like, opts.Language, like)
		default:
			where += " AND (" + original + " OR " + translated + ")"
			for range 6 {
				args = append(args, like)
			}
			args = append(args, like, opts.Language, like)
		}
	}
	if opts.Creator != "" {
		where += " AND (lower(v.creator_name)=lower(?) OR lower(v.subscription_slug)=lower(?))"
		args = append(args, opts.Creator, opts.Creator)
	}
	if opts.Codec != "" {
		where += " AND lower(v.video_codec)=lower(?)"
		args = append(args, opts.Codec)
	}
	if opts.MinHeight > 0 {
		where += " AND v.height>=?"
		args = append(args, opts.MinHeight)
	}
	if len(opts.Tags) > 0 {
		parts := []string{}
		for _, tag := range opts.Tags {
			parts = append(parts, "(EXISTS(SELECT 1 FROM json_each(v.tags_json) j WHERE lower(COALESCE(json_extract(j.value,'$.Name'),json_extract(j.value,'$.name'),j.value))=lower(?)) OR EXISTS(SELECT 1 FROM translations tx,json_each(tx.translated_text) j WHERE tx.entity_id=v.id AND tx.language=? AND tx.field='tags' AND lower(j.value)=lower(?)))")
			args = append(args, tag, opts.Language, tag)
		}
		joiner := " AND "
		if opts.TagMode == "any" {
			joiner = " OR "
		}
		where += " AND (" + strings.Join(parts, joiner) + ")"
	}
	for _, tag := range opts.ExcludeTags {
		where += " AND NOT (EXISTS(SELECT 1 FROM json_each(v.tags_json) j WHERE lower(COALESCE(json_extract(j.value,'$.Name'),json_extract(j.value,'$.name'),j.value))=lower(?)) OR EXISTS(SELECT 1 FROM translations tx,json_each(tx.translated_text) j WHERE tx.entity_id=v.id AND tx.language=? AND tx.field='tags' AND lower(j.value)=lower(?)))"
		args = append(args, tag, opts.Language, tag)
	}
	return where, args
}
func (s *Store) VideoFacets() (map[string][]VideoFacet, error) {
	out := map[string][]VideoFacet{"creators": {}, "tags": {}, "codecs": {}}
	queries := map[string]string{
		"creators": "SELECT creator_name,count(*) FROM videos WHERE status='completed' AND creator_name<>'' GROUP BY creator_name ORDER BY count(*) DESC,creator_name LIMIT 200",
		"tags":     "SELECT COALESCE(json_extract(j.value,'$.Name'),json_extract(j.value,'$.name'),j.value),count(DISTINCT v.id) FROM videos v,json_each(v.tags_json) j WHERE v.status='completed' GROUP BY 1 ORDER BY 2 DESC,1 LIMIT 500",
		"codecs":   "SELECT video_codec,count(*) FROM videos WHERE status='completed' AND video_codec<>'' GROUP BY video_codec ORDER BY count(*) DESC",
	}
	for key, query := range queries {
		rows, err := s.DB.Query(query)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var f VideoFacet
			if err = rows.Scan(&f.Value, &f.Count); err != nil {
				rows.Close()
				return nil, err
			}
			out[key] = append(out[key], f)
		}
		rows.Close()
	}
	return out, nil
}
func (s *Store) CountVideos(status string) (int64, error) {
	var n int64
	if status == "" {
		return n, s.DB.QueryRow("SELECT count(*) FROM videos").Scan(&n)
	}
	return n, s.DB.QueryRow("SELECT count(*) FROM videos WHERE status=?", status).Scan(&n)
}
func (s *Store) QueueDiscovered(subscriptionSlug string) (int64, error) {
	query := "UPDATE videos SET status='queued',error='',updated_at=? WHERE status='discovered'"
	args := []any{Now()}
	if subscriptionSlug != "" {
		query += " AND subscription_slug=?"
		args = append(args, subscriptionSlug)
	}
	result, err := s.DB.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
func (s *Store) QueueVideo(id string) error {
	result, err := s.DB.Exec("UPDATE videos SET status='queued',aria_gid='',error='',updated_at=? WHERE id=? AND status IN ('discovered','failed','excluded')", Now(), id)
	if err != nil {
		return err
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		return errors.New("视频已经在下载队列中或已完成")
	}
	return nil
}
func (s *Store) RequeueTransientFailures() (int64, error) {
	result, err := s.DB.Exec("UPDATE videos SET status='queued',aria_gid='',updated_at=? WHERE status='failed' AND (error LIKE '%HTTP 429%' OR error LIKE '%status=429%' OR error LIKE '%deadline exceeded%' OR error LIKE '%Timeout%')", Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
func (s *Store) TaskVideos(limit, offset int, status string) ([]Video, error) {
	if status != "" {
		return s.Videos(limit, offset, status, "")
	}
	rows, err := s.DB.Query("SELECT "+videoColumns+" FROM videos WHERE status<>'discovered' ORDER BY CASE status WHEN 'active' THEN 0 WHEN 'waiting' THEN 1 WHEN 'queued' THEN 2 WHEN 'paused' THEN 3 WHEN 'failed' THEN 4 ELSE 5 END,updated_at DESC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Video{}
	for rows.Next() {
		v, e := scanVideo(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) RecentCompleted(limit int) ([]Video, error) {
	rows, err := s.DB.Query("SELECT "+videoColumns+" FROM videos WHERE status='completed' ORDER BY downloaded_at DESC,updated_at DESC LIMIT ?", limit)
	return scanVideos(rows, err)
}
func (s *Store) RecentlyPlayed(limit int) ([]Video, error) {
	rows, err := s.DB.Query("SELECT "+prefixedVideoColumns("v")+" FROM playback_history p JOIN videos v ON v.id=p.video_id WHERE v.status='completed' ORDER BY p.last_played_at DESC LIMIT ?", limit)
	return scanVideos(rows, err)
}
func scanVideos(rows *sql.Rows, err error) ([]Video, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Video{}
	for rows.Next() {
		v, e := scanVideo(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func prefixedVideoColumns(prefix string) string {
	parts := strings.Split(videoColumns, ",")
	for i := range parts {
		parts[i] = prefix + "." + parts[i]
	}
	return strings.Join(parts, ",")
}
func (s *Store) RecordPlayback(videoID string) error {
	now := Now()
	_, err := s.DB.Exec("INSERT INTO playback_history(video_id,play_count,last_played_at) VALUES(?,1,?) ON CONFLICT(video_id) DO UPDATE SET play_count=play_count+CASE WHEN (julianday(excluded.last_played_at)-julianday(playback_history.last_played_at))*1440>=10 THEN 1 ELSE 0 END,last_played_at=excluded.last_played_at", videoID, now)
	return err
}
func (s *Store) CountTasks(status string) (int64, error) {
	if status != "" {
		return s.CountVideos(status)
	}
	var n int64
	return n, s.DB.QueryRow("SELECT count(*) FROM videos WHERE status<>'discovered'").Scan(&n)
}
func (s *Store) SubscriptionProgress(slug string) (SubscriptionProgress, error) {
	var p SubscriptionProgress
	err := s.DB.QueryRow("SELECT count(*),COALESCE(SUM(status='completed'),0),COALESCE(SUM(status='active'),0),COALESCE(SUM(status IN ('waiting','queued')),0),COALESCE(SUM(status='failed'),0),COALESCE(SUM(bytes_downloaded),0),COALESCE(SUM(total_bytes),0) FROM videos WHERE subscription_slug=?", slug).Scan(&p.Total, &p.Completed, &p.Active, &p.Waiting, &p.Failed, &p.BytesDownloaded, &p.TotalBytes)
	return p, err
}
func (s *Store) UpdateDetails(v Video) error {
	_, err := s.DB.Exec("UPDATE videos SET title=?,published_at=?,thumbnail_url=?,views=?,duration_seconds=?,creator_name=?,creator_url=?,creator_avatar=?,uploader_name=?,uploader_url=?,uploader_avatar=?,categories_json=?,tags_json=?,external_links_json=?,creators_json=?,source_relation_json=CASE WHEN ?='' OR ?='{}' THEN source_relation_json ELSE ? END,raw_html=?,updated_at=? WHERE id=?", v.Title, v.PublishedAt, v.ThumbnailURL, v.Views, v.DurationSeconds, v.CreatorName, v.CreatorURL, v.CreatorAvatar, v.UploaderName, v.UploaderURL, v.UploaderAvatar, v.CategoriesJSON, v.TagsJSON, v.ExternalLinksJSON, v.CreatorsJSON, v.SourceRelationJSON, v.SourceRelationJSON, v.SourceRelationJSON, v.RawHTML, Now(), v.ID)
	return err
}
func (s *Store) RequeueTrustedCreatorMismatches(slug string) (int64, error) {
	res, err := s.DB.Exec("UPDATE videos SET status='queued',aria_gid='',error='',updated_at=? WHERE subscription_slug=? AND status='excluded' AND error LIKE '实际创作者与订阅作者不一致:%' AND source_relation_json<>'{}'", Now(), slug)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
func (s *Store) UpdateTask(id, status, gid, path, plexPath, errText string, done, total int64) error {
	_, err := s.DB.Exec("UPDATE videos SET status=?,aria_gid=?,file_path=?,plex_path=?,error=?,bytes_downloaded=?,total_bytes=?,download_speed=CASE WHEN ? IN ('completed','failed','discovered') THEN 0 ELSE download_speed END,connections=CASE WHEN ? IN ('completed','failed','discovered') THEN 0 ELSE connections END,downloaded_at=CASE WHEN ?='completed' THEN ? ELSE downloaded_at END,updated_at=? WHERE id=?", status, gid, path, plexPath, errText, done, total, status, status, status, Now(), Now(), id)
	return err
}
func (s *Store) UpdateTransfer(id string, speed int64, connections int) error {
	_, err := s.DB.Exec("UPDATE videos SET download_speed=?,connections=? WHERE id=?", speed, connections, id)
	return err
}
func (s *Store) UpdateMediaInfo(id, videoCodec, audioCodec, container string, width, height int, frameRate float64, bitRate int64, bitDepth int) error {
	_, err := s.DB.Exec("UPDATE videos SET video_codec=?,audio_codec=?,container=?,width=?,height=?,frame_rate=?,bit_rate=?,bit_depth=?,media_probed_at=?,transcode_status=CASE WHEN transcode_status='' THEN 'not_requested' ELSE transcode_status END,updated_at=? WHERE id=?", videoCodec, audioCodec, container, width, height, frameRate, bitRate, bitDepth, Now(), Now(), id)
	return err
}
func (s *Store) NextEpisode(slug string) (int, error) {
	var n int
	err := s.DB.QueryRow("SELECT COALESCE(MAX(plex_episode),0)+1 FROM videos WHERE subscription_slug=?", slug).Scan(&n)
	return n, err
}
func (s *Store) AssignEpisode(id string, season, episode int) error {
	_, err := s.DB.Exec("UPDATE videos SET plex_season=?,plex_episode=?,updated_at=? WHERE id=?", season, episode, Now(), id)
	return err
}
func (s *Store) AddTraffic(id string, bytes int64) error {
	if bytes <= 0 {
		return nil
	}
	_, err := s.DB.Exec("INSERT INTO traffic_events(video_id,bytes,recorded_at) VALUES(?,?,?)", id, bytes, Now())
	return err
}
func (s *Store) Stats() (traffic, library int64, err error) {
	err = s.DB.QueryRow("SELECT COALESCE(SUM(bytes),0) FROM traffic_events").Scan(&traffic)
	if err != nil {
		return
	}
	err = s.DB.QueryRow("SELECT COALESCE(SUM(total_bytes),0) FROM videos WHERE status='completed'").Scan(&library)
	return
}
func (s *Store) TransferStats() (speed int64, connections int, err error) {
	err = s.DB.QueryRow("SELECT COALESCE(SUM(download_speed),0),COALESCE(SUM(connections),0) FROM videos WHERE status='active'").Scan(&speed, &connections)
	return
}
func (s *Store) SaveImport(source, raw, videoID, errorText string) error {
	_, err := s.DB.Exec("INSERT INTO imports(source,raw_html,video_id,created_at,error) VALUES(?,?,?,?,?)", source, raw, videoID, Now(), errorText)
	return err
}
func (s *Store) Translations(language string) (map[string]Translation, error) {
	rows, err := s.DB.Query("SELECT entity_id,language,field,source_hash,translated_text,provider,updated_at FROM translations WHERE language=?", language)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]Translation{}
	for rows.Next() {
		var x Translation
		if err = rows.Scan(&x.EntityID, &x.Language, &x.Field, &x.SourceHash, &x.Text, &x.Provider, &x.UpdatedAt); err != nil {
			return nil, err
		}
		out[x.EntityID+"\x00"+x.Field] = x
	}
	return out, rows.Err()
}
func (s *Store) SaveTranslation(x Translation) error {
	_, err := s.DB.Exec("INSERT INTO translations(entity_id,language,field,source_hash,translated_text,provider,updated_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(entity_id,language,field) DO UPDATE SET source_hash=excluded.source_hash,translated_text=excluded.translated_text,provider=excluded.provider,updated_at=excluded.updated_at", x.EntityID, x.Language, x.Field, x.SourceHash, x.Text, x.Provider, Now())
	return err
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

var ErrNotFound = errors.New("not found")
var ErrInvalidSource = errors.New("无效的订阅来源地址")
