package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"r34watch/internal/store"
)

// Thumbnail refresh statuses returned by the single-video API and maintenance
// runner.
const (
	thumbUpdated   = "updated"
	thumbUnchanged = "unchanged"
	thumbMissing   = "missing"
	thumbFailed    = "failed"
)

type thumbnailRefresh struct {
	Status string
	Before string
	After  string
}

var thumbnailDimRE = regexp.MustCompile(`(?i)(\d{1,5})x(\d{1,5})`)

// isUsableThumbnail reports whether a stored/returned value is a real remote
// image reference rather than empty data or a lazy-load placeholder.
func isUsableThumbnail(raw string) bool {
	value := strings.TrimSpace(raw)
	if value == "" {
		return false
	}
	low := strings.ToLower(value)
	if strings.HasPrefix(low, "data:") || strings.HasPrefix(low, "javascript:") {
		return false
	}
	return strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") || strings.HasPrefix(low, "//")
}

// thumbnailArea parses a WxH hint (e.g. Rule34Video .../320x180/7.jpg) so we
// can avoid downgrading a larger cover. It returns false when no hint exists.
func thumbnailArea(raw string) (int, bool) {
	m := thumbnailDimRE.FindStringSubmatch(raw)
	if len(m) != 3 {
		return 0, false
	}
	w, h := 0, 0
	for _, c := range m[1] {
		w = w*10 + int(c-'0')
	}
	for _, c := range m[2] {
		h = h*10 + int(c-'0')
	}
	if w <= 0 || h <= 0 {
		return 0, false
	}
	return w * h, true
}

func thumbnailHTTPClient(proxy string) *http.Client {
	transport := &http.Transport{}
	if proxy != "" {
		if p, err := url.Parse(proxy); err == nil {
			transport.Proxy = http.ProxyURL(p)
		}
	}
	return &http.Client{Transport: transport, Timeout: 15 * time.Second}
}

// thumbnailNeedsRefresh implements mode=missing: only covers that are empty,
// placeholder data, unreachable, or served with a non-image content type are
// treated as missing. Valid covers trigger no network request here.
func thumbnailNeedsRefresh(ctx context.Context, client *http.Client, raw string) bool {
	if !isUsableThumbnail(raw) {
		return true
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, raw, nil)
	if err != nil {
		return true
	}
	req.Header.Set("User-Agent", userAgent)
	res, err := client.Do(req)
	if err != nil {
		return true
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusMethodNotAllowed || res.StatusCode == http.StatusNotImplemented {
		return false
	}
	if res.StatusCode < 200 || res.StatusCode >= 400 {
		return true
	}
	return !strings.HasPrefix(strings.ToLower(res.Header.Get("Content-Type")), "image/")
}

// chooseThumbnail decides whether the freshly resolved cover should replace the
// stored one. It never downgrades a valid cover to a lower-resolution cover
// when both carry a comparable dimension hint, and never replaces a valid cover
// with an empty/placeholder value. When dimensions can't be compared it trusts
// the adapter's explicit priority instead of issuing extra image requests.
func chooseThumbnail(mode, current, candidate string) (string, bool) {
	candidate = strings.TrimSpace(candidate)
	if !isUsableThumbnail(candidate) {
		return current, false
	}
	if candidate == current {
		return current, false
	}
	if mode == "force" && isUsableThumbnail(current) {
		currentArea, currentKnown := thumbnailArea(current)
		candidateArea, candidateKnown := thumbnailArea(candidate)
		if currentKnown && candidateKnown && candidateArea <= currentArea {
			return current, false
		}
	}
	return candidate, true
}

func (a *App) invalidateThumbnailCache(id string, cfg store.Settings) {
	cachePath := filepath.Join(cfg.DownloadDir, ".hwatch-thumbnails", fmt.Sprintf("%x", sha256.Sum256([]byte(id)))+".jpg")
	if _, err := os.Stat(cachePath); err == nil {
		_ = os.Remove(cachePath)
	}
}

// refreshThumbnail resolves the current cover for one video and updates only
// thumbnail_url. It never touches the media file, path, codec, hash, lifecycle
// status, exclusions or downloader state.
func (a *App) refreshThumbnail(ctx context.Context, v store.Video, mode string, cfg store.Settings, client *http.Client) thumbnailRefresh {
	result := thumbnailRefresh{Status: thumbUnchanged, Before: v.ThumbnailURL, After: v.ThumbnailURL}
	if mode != "force" && !thumbnailNeedsRefresh(ctx, client, v.ThumbnailURL) {
		return result
	}
	adapter, err := a.adapters.ByID(v.Platform)
	if err != nil {
		result.Status = thumbFailed
		return result
	}
	fetchCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	detailed, err := adapter.FetchDetails(fetchCtx, v, cfg.HTTPProxy)
	if err != nil {
		result.Status = thumbFailed
		return result
	}
	after, changed := chooseThumbnail(mode, v.ThumbnailURL, detailed.ThumbnailURL)
	if !changed {
		if !isUsableThumbnail(v.ThumbnailURL) {
			result.Status = thumbMissing
		}
		return result
	}
	if err := a.store.UpdateThumbnail(v.ID, after); err != nil {
		result.Status = thumbFailed
		return result
	}
	a.invalidateThumbnailCache(v.ID, cfg)
	result.Status = thumbUpdated
	result.After = after
	return result
}

// refreshThumbnailHandler backs POST /api/videos/{id}/refresh-thumbnail. It is a
// normal management endpoint: it requires a logged-in session and is not
// exposed to TransRunner service tokens (which are read-only via a separate
// path).
func (a *App) refreshThumbnailHandler(w http.ResponseWriter, r *http.Request) {
	v, err := a.store.Video(r.PathValue("id"))
	if err != nil {
		fail(w, http.StatusNotFound, errors.New("视频不存在"))
		return
	}
	cfg, _ := a.store.Settings()
	client := thumbnailHTTPClient(cfg.HTTPProxy)
	result := a.refreshThumbnail(r.Context(), v, "force", cfg, client)
	respond(w, http.StatusOK, map[string]any{"status": result.Status, "video_id": v.ID, "before": result.Before, "after": result.After})
}

type thumbRefreshOptions struct {
	Source      string
	Mode        string
	Limit       int
	Concurrency int
	DelayMS     int
}

func runMaintenance(s *store.Store, args []string) error {
	if len(args) == 0 {
		return errors.New("用法: r34watch maintenance refresh-thumbnails [--source ... --mode missing|force --limit N --concurrency 1 --delay-ms 500]")
	}
	switch args[0] {
	case "refresh-thumbnails":
		return runThumbnailRefreshCommand(s, args[1:])
	default:
		return fmt.Errorf("未知的 maintenance 命令: %s", args[0])
	}
}

func runThumbnailRefreshCommand(s *store.Store, args []string) error {
	fs := flag.NewFlagSet("refresh-thumbnails", flag.ContinueOnError)
	source := fs.String("source", "", "只处理指定平台，例如 rule34video")
	status := fs.String("status", "downloaded", "只支持 downloaded（正式已下载媒体）")
	mode := fs.String("mode", "missing", "missing 只补缺失/失效封面；force 重新解析并更新")
	limit := fs.Int("limit", 200, "本次最多扫描的视频数")
	concurrency := fs.Int("concurrency", 1, "并发数，最大 2")
	delayMS := fs.Int("delay-ms", 500, "每次请求后的间隔毫秒")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *mode != "missing" && *mode != "force" {
		return errors.New("--mode 只支持 missing 或 force")
	}
	if *status != "" && *status != "downloaded" {
		return errors.New("--status 目前只支持 downloaded")
	}
	if *concurrency < 1 {
		*concurrency = 1
	}
	if *concurrency > 2 {
		*concurrency = 2
	}
	a := New(s)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return a.runThumbnailRefresh(ctx, thumbRefreshOptions{Source: *source, Mode: *mode, Limit: *limit, Concurrency: *concurrency, DelayMS: *delayMS})
}

func (a *App) runThumbnailRefresh(ctx context.Context, o thumbRefreshOptions) error {
	cfg, err := a.store.Settings()
	if err != nil {
		return err
	}
	videos, err := a.store.DownloadedVideos(o.Source, o.Limit, 0)
	if err != nil {
		return err
	}
	client := thumbnailHTTPClient(cfg.HTTPProxy)
	var scanned, processed, updated, unchanged, missing, failed int64
	scanned = int64(len(videos))
	delay := time.Duration(o.DelayMS) * time.Millisecond

	var mu sync.Mutex
	var penaltyMS atomic.Int64
	report := func() {
		mu.Lock()
		defer mu.Unlock()
		fmt.Fprintf(os.Stdout, "\r%d/%d updated=%d unchanged=%d missing=%d failed=%d", processed, scanned, updated, unchanged, missing, failed)
	}
	report()

	jobs := make(chan store.Video)
	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()
		for v := range jobs {
			if ctx.Err() != nil {
				return
			}
			if p := penaltyMS.Load(); p > 0 {
				select {
				case <-time.After(time.Duration(p) * time.Millisecond):
				case <-ctx.Done():
					return
				}
			}
			result := a.refreshThumbnail(ctx, v, o.Mode, cfg, client)
			mu.Lock()
			processed++
			switch result.Status {
			case thumbUpdated:
				updated++
			case thumbMissing:
				missing++
			case thumbFailed:
				failed++
			default:
				unchanged++
			}
			mu.Unlock()
			if result.Status == thumbFailed {
				next := penaltyMS.Load()*2 + 1000
				if next > 30000 {
					next = 30000
				}
				penaltyMS.Store(next)
			} else {
				penaltyMS.Store(0)
			}
			report()
			if delay > 0 {
				select {
				case <-time.After(delay):
				case <-ctx.Done():
					return
				}
			}
		}
	}
	for i := 0; i < o.Concurrency; i++ {
		wg.Add(1)
		go worker()
	}
feed:
	for _, v := range videos {
		select {
		case jobs <- v:
		case <-ctx.Done():
			break feed
		}
	}
	close(jobs)
	wg.Wait()
	fmt.Fprintf(os.Stdout, "\n%d/%d updated=%d unchanged=%d missing=%d failed=%d\n", processed, scanned, updated, unchanged, missing, failed)
	if ctx.Err() != nil {
		fmt.Fprintln(os.Stdout, "已中断，剩余未处理项将在下次运行时继续（幂等）。")
	}
	return nil
}
