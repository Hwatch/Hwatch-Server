package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"r34watch/internal/collector"
	"r34watch/internal/store"
)

// Hanime1 supports both a playlist (series) and a user's uploaded videos.
// Signed media URLs are deliberately never persisted: ResolveDownload refreshes
// the watch page immediately before the task is handed to the downloader.
type Hanime1 struct{}

func (Hanime1) ID() string   { return "hanime1" }
func (Hanime1) Name() string { return "Hanime1" }
func (Hanime1) MatchURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Hostname(), "hanime1.me") {
		return false
	}
	return (u.Path == "/playlist" && u.Query().Get("list") != "") || hanimeUserPathRE.MatchString(u.Path)
}

var (
	hanimeUserPathRE      = regexp.MustCompile(`^/user/([0-9]+)(?:/uploaded)?/?$`)
	hanimeWatchRE         = regexp.MustCompile(`(?i)href=["'](?:https?://hanime1\.me)?/watch\?[^"']*\bv=([0-9]+)[^"']*["']`)
	hanimeCardRE          = regexp.MustCompile(`(?i)<div[^>]+class=["'][^"']*(?:playlist-video-card|video-item-container)[^"']*["']`)
	hanimeTitleRE         = regexp.MustCompile(`(?is)<h4[^>]*class=["'][^"']*video-title[^"']*["'][^>]*>.*?<a[^>]*>(.*?)</a>`)
	hanimeDivTitleRE      = regexp.MustCompile(`(?is)<div[^>]*class=["'][^"']*title[^"']*["'][^>]*>(.*?)</div>`)
	hanimeAttrTitleRE     = regexp.MustCompile(`(?is)^<div[^>]+title=["']([^"']+)["']`)
	hanimeImageRE         = regexp.MustCompile(`(?is)<img[^>]+class=["'][^"']*main-thumb[^"']*["'][^>]+src=["']([^"']+)["']`)
	hanimeDurationRE      = regexp.MustCompile(`(?is)<div[^>]+class=["'][^"']*duration[^"']*["'][^>]*>(.*?)</div>`)
	hanimeViewsRE         = regexp.MustCompile(`(?is)<div[^>]+class=["'][^"']*stat-item[^"']*["'][^>]*>\s*([0-9.,]+)\s*([万萬亿億]?)次`)
	hanimeH1RE            = regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
	hanimeH3RE            = regexp.MustCompile(`(?is)<h3[^>]*>(.*?)</h3>`)
	hanimeProfileAvatarRE = regexp.MustCompile(`(?is)<div[^>]+class=["'][^"']*profile-avatar-wrapper[^"']*["'][^>]*>.*?<img[^>]+src=["']([^"']+)["']`)
	hanimeSourceRE        = regexp.MustCompile(`(?is)<source[^>]+src=["']([^"']+)["'][^>]*size=["']?([0-9]+)["']?`)
	hanimeDownloadURLRE   = regexp.MustCompile(`(?is)data-url=["'](https?://[^"']+\.mp4\?[^"']*)["']`)
	hanimeQualityRE       = regexp.MustCompile(`-(\d{2,5})-(\d{2,5})-\d+-h264\.mp4`)
	hanimeHLSRE           = regexp.MustCompile(`(?is)source\s*=\s*["'][^"']*\.m3u8`)
	hanimePosterRE        = regexp.MustCompile(`(?is)<video[^>]+(?:data-poster|poster)=["']([^"']+)["']`)
	hanimeMetaRE          = regexp.MustCompile(`(?is)<meta[^>]+(?:name|property)=["']([^"']+)["'][^>]+content=["']([^"']*)["']`)
	hanimeUserRE          = regexp.MustCompile(`(?is)<a[^>]+href=["'](?:https?://hanime1\.me)?(/user/([0-9]+))/?["'][^>]*>(.*?)</a>`)
	hanimeAltRE           = regexp.MustCompile(`(?is)<img[^>]+alt=["']([^"']+)["']`)
	hanimeGenreRE         = regexp.MustCompile(`(?is)<a[^>]+href=["']([^"']*/search\?genre=[^"']+)["'][^>]*>(.*?)</a>`)
	hanimeTagRE           = regexp.MustCompile(`(?is)<div[^>]+class=["'][^"']*single-video-tag[^"']*["'][^>]*>\s*<a[^>]+href=["']([^"']*/search\?(?:query|tags[^=]*)=[^"']+)["'][^>]*>(.*?)</a>`)
	hanimeDescriptionRE   = regexp.MustCompile(`(?is)<div[^>]+class=["'][^"']*video-caption-text[^"']*["'][^>]*>(.*?)</div>`)
	hanimeDetailViewsRE   = regexp.MustCompile(`(?:观看次数|觀看次數)[:：]\s*([0-9.,]+)\s*([万萬亿億]?)次`)
	hanimeLikeRE          = regexp.MustCompile(`(?is)id=["']video-like-btn["'].*?thumb_up</i>\s*([0-9.]+)%.*?<span>\s*\(([0-9,]+)\)\s*</span>`)
)

func (Hanime1) SearchCreators(_ context.Context, _ string, _ string) ([]CreatorResult, error) {
	return []CreatorResult{}, nil
}

func (Hanime1) ParseSubscription(_ context.Context, raw, requestedType string) (store.Subscription, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !strings.EqualFold(u.Hostname(), "hanime1.me") {
		return store.Subscription{}, store.ErrInvalidSource
	}
	var kind, sourceID string
	if u.Path == "/playlist" && u.Query().Get("list") != "" {
		kind, sourceID = "playlist", u.Query().Get("list")
	} else if m := hanimeUserPathRE.FindStringSubmatch(u.Path); len(m) == 2 {
		kind, sourceID = "creator", m[1]
	} else {
		return store.Subscription{}, store.ErrInvalidSource
	}
	if requestedType != "" && requestedType != kind {
		return store.Subscription{}, fmt.Errorf("订阅类型与地址不匹配，应为 %s", kind)
	}
	canonical := "https://hanime1.me/playlist?list=" + url.QueryEscape(sourceID) + "&sort=latest"
	if kind == "creator" {
		canonical = "https://hanime1.me/user/" + sourceID + "/uploaded?sort=latest"
	}
	name := "播放清单 " + sourceID
	if kind == "creator" {
		name = "用户 " + sourceID
	}
	return store.Subscription{Slug: "hanime1:" + kind + ":" + sourceID, Platform: "hanime1", Type: kind, SourceID: sourceID, Name: name, URL: canonical, Enabled: true, ExternalLinks: "[]", AdapterConfig: "{}", Cursor: "{}"}, nil
}

func (Hanime1) EnrichSubscription(ctx context.Context, sub store.Subscription, proxy string) store.Subscription {
	if page, fetchErr := hanimeFetch(ctx, sub.URL, proxy); fetchErr == nil {
		if m := hanimeH1RE.FindStringSubmatch(page); len(m) == 2 && hanimeClean(m[1]) != "" {
			sub.Name = hanimeClean(m[1])
		}
		if sub.Type == "creator" {
			if m := hanimeProfileAvatarRE.FindStringSubmatch(page); len(m) == 2 {
				sub.AvatarURL = hanimeAbsolute(m[1])
			}
		}
	}
	return sub
}

func (Hanime1) ListVideos(ctx context.Context, sub store.Subscription, proxy string) ([]store.Video, error) {
	page, err := hanimeFetch(ctx, sub.URL, proxy)
	if err != nil {
		return nil, err
	}
	return hanimeParseListing(page, sub)
}

func hanimeParseListing(page string, sub store.Subscription) ([]store.Video, error) {
	starts := hanimeCardRE.FindAllStringIndex(page, -1)
	out := make([]store.Video, 0, len(starts))
	seen := map[string]bool{}
	for i, pos := range starts {
		end := len(page)
		if i+1 < len(starts) {
			end = starts[i+1][0]
		}
		block := page[pos[0]:end]
		wm := hanimeWatchRE.FindStringSubmatch(block)
		if len(wm) != 2 || seen[wm[1]] {
			continue
		}
		seen[wm[1]] = true
		title := "Hanime1 " + wm[1]
		if m := hanimeTitleRE.FindStringSubmatch(block); len(m) == 2 {
			title = hanimeClean(m[1])
		} else if m := hanimeDivTitleRE.FindStringSubmatch(block); len(m) == 2 {
			title = hanimeClean(m[1])
		} else if m := hanimeAttrTitleRE.FindStringSubmatch(block); len(m) == 2 {
			title = hanimeClean(m[1])
		}
		thumbnail := ""
		if m := hanimeImageRE.FindStringSubmatch(block); len(m) == 2 {
			thumbnail = hanimeAbsolute(m[1])
		}
		duration := int64(0)
		if m := hanimeDurationRE.FindStringSubmatch(block); len(m) == 2 {
			duration = hanimeDuration(hanimeClean(m[1]))
		}
		views := int64(0)
		if m := hanimeViewsRE.FindStringSubmatch(block); len(m) == 3 {
			views = hanimeCount(m[1], m[2])
		}
		relation, _ := json.Marshal(map[string]string{"type": sub.Type + "_listing", "platform": "hanime1", "subscription_slug": sub.Slug, "subscription_url": sub.URL, "source_id": sub.SourceID})
		out = append(out, store.Video{ID: "hanime1:" + wm[1], Platform: "hanime1", SourceID: wm[1], SubscriptionSlug: sub.Slug, Title: title, PageURL: "https://hanime1.me/watch?v=" + wm[1], ThumbnailURL: thumbnail, DurationSeconds: duration, Views: views, SourceRelationJSON: string(relation)})
	}
	if len(out) == 0 {
		if collector.LooksLikeAntiBot(page) {
			return nil, &collector.TemplateError{Platform: "hanime1", Reason: "来源返回人机校验/反爬页面，未获得真实列表"}
		}
		return nil, &collector.TemplateError{Platform: "hanime1", Reason: "未解析出任何视频卡片（playlist-video-card / video-item-container 结构可能改变）"}
	}
	return out, nil
}

func (Hanime1) FetchDetails(ctx context.Context, video store.Video, proxy string) (store.Video, error) {
	page, err := hanimeFetch(ctx, video.PageURL, proxy)
	if err != nil {
		return video, err
	}
	if m := hanimeH3RE.FindStringSubmatch(page); len(m) == 2 {
		video.Title = hanimeClean(m[1])
	}
	meta := map[string]string{}
	for _, m := range hanimeMetaRE.FindAllStringSubmatch(page, -1) {
		meta[strings.ToLower(m[1])] = html.UnescapeString(m[2])
	}
	if v := meta["og:image"]; v != "" {
		video.ThumbnailURL = hanimeAbsolute(v)
	} else if m := hanimePosterRE.FindStringSubmatch(page); len(m) == 2 {
		video.ThumbnailURL = hanimeAbsolute(m[1])
	}
	if n, _ := strconv.ParseInt(meta["og:video:duration"], 10, 64); n > 0 {
		video.DurationSeconds = n
	}
	description := meta["description"]
	if m := hanimeDescriptionRE.FindStringSubmatch(page); len(m) == 2 {
		description = hanimeMultilineText(m[1])
	}
	video.Description = description
	if m := regexp.MustCompile(`(?i)Release\s*/[^:]*:\s*([0-9]{4})/([0-9]{2})/([0-9]{2})`).FindStringSubmatch(description); len(m) == 4 {
		video.PublishedAt = m[1] + "-" + m[2] + "-" + m[3] + "T00:00:00Z"
	}
	if m := hanimeDetailViewsRE.FindStringSubmatch(hanimeClean(page)); len(m) == 3 {
		video.Views = hanimeCount(m[1], m[2])
	}
	if m := hanimeLikeRE.FindStringSubmatch(page); len(m) == 3 {
		video.LikeRate, _ = strconv.ParseFloat(m[1], 64)
		video.Likes, _ = strconv.ParseInt(strings.ReplaceAll(m[2], ",", ""), 10, 64)
	}
	if m := hanimeUserRE.FindStringSubmatch(page); len(m) == 4 {
		name := hanimeClean(m[3])
		if alt := hanimeAltRE.FindStringSubmatch(m[3]); len(alt) == 2 {
			name = hanimeClean(alt[1])
		}
		video.UploaderName, video.UploaderURL = name, "https://hanime1.me"+m[1]
		video.CreatorName, video.CreatorURL = video.UploaderName, video.UploaderURL
	}
	video.CategoriesJSON = hanimeLinksJSON(page, hanimeGenreRE)
	video.TagsJSON = hanimeLinksJSON(page, hanimeTagRE)
	video.CreatorsJSON, _ = marshalHanimeNames(video.CreatorName, video.CreatorURL)
	video.ExternalLinksJSON = hanimeExternalLinks(description)
	video.RawHTML = ""
	return video, nil
}

func (Hanime1) ResolveDownload(ctx context.Context, video store.Video, proxy string) (DownloadSource, error) {
	// Hanime1's dedicated download page exposes signed direct MP4 URLs
	// (data-url="...-<w>-<h>-<kbps>-h264.mp4?...") for each quality, while the
	// watch page now only carries an HLS master.m3u8. Prefer the MP4 page.
	id := hanimeNumericID(video)
	if id != "" {
		downloadURL := "https://hanime1.me/download?v=" + id
		if page, err := hanimeFetch(ctx, downloadURL, proxy); err == nil {
			if src, ok := hanimeDirectMP4(page, downloadURL); ok {
				return src, nil
			}
		}
	}
	// Fallback: legacy watch pages still expose <source src=... size=...>.
	page, err := hanimeFetch(ctx, video.PageURL, proxy)
	if err != nil {
		return DownloadSource{}, err
	}
	sources := hanimeSourceRE.FindAllStringSubmatch(page, -1)
	if len(sources) == 0 {
		if hanimeHLSRE.MatchString(page) {
			return DownloadSource{}, errors.New("Hanime1 该视频下载页未提供直链 MP4（仅 HLS 流），页面可能已变更")
		}
		return DownloadSource{}, errors.New("Hanime1 播放页没有可用 MP4 下载源")
	}
	type choice struct {
		raw     string
		quality int64
	}
	items := make([]choice, 0, len(sources))
	for _, m := range sources {
		q, _ := strconv.ParseInt(m[2], 10, 64)
		items = append(items, choice{hanimeAbsolute(m[1]), q})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].quality > items[j].quality })
	return DownloadSource{URL: items[0].raw, Quality: items[0].quality, Headers: map[string]string{"Referer": video.PageURL}}, nil
}

// hanimeNumericID returns the hanime1 video id for a record, from SourceID or
// the watch URL.
func hanimeNumericID(video store.Video) string {
	if video.SourceID != "" {
		return video.SourceID
	}
	if m := regexp.MustCompile(`[?&]v=(\d+)`).FindStringSubmatch(video.PageURL); len(m) == 2 {
		return m[1]
	}
	return ""
}

// hanimeDirectMP4 extracts the best signed direct MP4 URL from a /download
// page, choosing the highest resolution variant advertised via data-url.
func hanimeDirectMP4(page, referer string) (DownloadSource, bool) {
	matches := hanimeDownloadURLRE.FindAllStringSubmatch(page, -1)
	if len(matches) == 0 {
		return DownloadSource{}, false
	}
	var bestURL string
	var bestArea, bestHeight int64
	for _, m := range matches {
		raw := html.UnescapeString(m[1])
		area, height := int64(0), int64(0)
		if qm := hanimeQualityRE.FindStringSubmatch(raw); len(qm) == 3 {
			w, _ := strconv.ParseInt(qm[1], 10, 64)
			h, _ := strconv.ParseInt(qm[2], 10, 64)
			area, height = w*h, h
		}
		if bestURL == "" || area > bestArea {
			bestURL, bestArea, bestHeight = raw, area, height
		}
	}
	if bestURL == "" {
		return DownloadSource{}, false
	}
	return DownloadSource{URL: bestURL, Quality: bestHeight, Headers: map[string]string{"Referer": referer}}, true
}

func (Hanime1) ParseImported(string) (store.Video, error) {
	return store.Video{}, errors.New("Hanime1 暂不支持剪贴板 HTML 导入")
}

func hanimeFetch(ctx context.Context, raw, proxy string) (string, error) {
	transport := &http.Transport{MaxIdleConns: 10, MaxIdleConnsPerHost: 5, IdleConnTimeout: 90 * time.Second, ResponseHeaderTimeout: 40 * time.Second}
	if proxy != "" {
		if p, err := url.Parse(proxy); err == nil {
			transport.Proxy = http.ProxyURL(p)
		}
	}
	client := &http.Client{Transport: transport, Timeout: 50 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.7")
	if cookie := strings.TrimSpace(os.Getenv("HWATCH_HANIME1_COOKIE")); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	res, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if fallback, fallbackErr := hanimeWget(ctx, raw, proxy); fallbackErr == nil {
			return fallback, nil
		}
		return "", err
	}
	defer res.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(res.Body, 20<<20))
	if readErr != nil {
		return "", readErr
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if res.StatusCode == http.StatusForbidden || res.StatusCode == http.StatusServiceUnavailable || res.StatusCode == http.StatusTooManyRequests {
			if fallback, fallbackErr := hanimeWget(ctx, raw, proxy); fallbackErr == nil {
				return fallback, nil
			}
			return "", fmt.Errorf("Hanime1 访问验证未通过（HTTP %d）；请确认系统 wget 可用，或为服务设置 HWATCH_HANIME1_COOKIE", res.StatusCode)
		}
		return "", fmt.Errorf("Hanime1 返回 HTTP %d", res.StatusCode)
	}
	return string(body), nil
}

func hanimeWget(ctx context.Context, raw, proxy string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Hostname(), "hanime1.me") {
		return "", errors.New("拒绝通过回退下载器访问非 Hanime1 地址")
	}
	cmd := exec.CommandContext(ctx, "wget", "--quiet", "--output-document=-", "--timeout=45", "--tries=3", "--user-agent=Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36", raw)
	cmd.Env = os.Environ()
	if proxy != "" {
		cmd.Env = append(cmd.Env, "HTTPS_PROXY="+proxy, "https_proxy="+proxy)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err = cmd.Start(); err != nil {
		return "", err
	}
	body, readErr := io.ReadAll(io.LimitReader(stdout, 20<<20))
	waitErr := cmd.Wait()
	if readErr != nil {
		return "", readErr
	}
	if waitErr != nil {
		return "", waitErr
	}
	if len(body) == 0 {
		return "", errors.New("wget returned an empty Hanime1 page")
	}
	return string(body), nil
}

func hanimeClean(raw string) string {
	raw = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(raw, " ")
	return strings.Join(strings.Fields(html.UnescapeString(raw)), " ")
}
func hanimeMultilineText(raw string) string {
	raw = regexp.MustCompile(`(?i)<br\s*/?>`).ReplaceAllString(raw, "\n")
	raw = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(raw, " ")
	raw = html.UnescapeString(raw)
	lines := make([]string, 0)
	for _, line := range strings.Split(raw, "\n") {
		if line = strings.Join(strings.Fields(line), " "); line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}
func hanimeAbsolute(raw string) string {
	raw = html.UnescapeString(strings.ReplaceAll(raw, `\/`, "/"))
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	if strings.HasPrefix(raw, "/") {
		return "https://hanime1.me" + raw
	}
	return raw
}
func hanimeDuration(raw string) int64 {
	parts := strings.Split(raw, ":")
	var total int64
	for _, part := range parts {
		n, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil {
			return 0
		}
		total = total*60 + n
	}
	return total
}
func hanimeCount(number, unit string) int64 {
	n, _ := strconv.ParseFloat(strings.ReplaceAll(number, ",", ""), 64)
	switch unit {
	case "万", "萬":
		n *= 10000
	case "亿", "億":
		n *= 100000000
	}
	return int64(n)
}
func hanimeLinksJSON(page string, re *regexp.Regexp) string {
	type link struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	out, seen := []link{}, map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(page, -1) {
		name := hanimeClean(m[2])
		name = strings.TrimSpace(strings.TrimPrefix(name, "#"))
		name = regexp.MustCompile(`\s*\([0-9,]+\)\s*$`).ReplaceAllString(name, "")
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, link{Name: name, URL: hanimeAbsolute(m[1])})
	}
	b, _ := json.Marshal(out)
	return string(b)
}
func marshalHanimeNames(name, rawURL string) (string, error) {
	if name == "" {
		return "[]", nil
	}
	return stringMustJSON([]map[string]string{{"name": name, "url": rawURL}}), nil
}
func stringMustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }
func hanimeExternalLinks(description string) string {
	type link struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	re := regexp.MustCompile(`https?://[^\s<>"']+`)
	out := []link{}
	for _, raw := range re.FindAllString(description, -1) {
		u, err := url.Parse(strings.TrimRight(raw, ").,;"))
		if err != nil {
			continue
		}
		out = append(out, link{Name: u.Hostname(), URL: u.String()})
	}
	b, _ := json.Marshal(out)
	return string(b)
}
