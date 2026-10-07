package collector

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"r34watch/internal/store"
)

type Collector struct{}
type Link struct{ Name, URL, Image string }
type ModelResult struct {
	ID, Name, URL, AvatarURL string
	VideoCount               int64
}

var (
	videoRE         = regexp.MustCompile("href=[\"'](?:https?://rule34video\\.com)?(/video/(\\d+)/([^\"']+)/)[\"']")
	videoCardRE     = regexp.MustCompile(`(?i)data-video-card-id=["'](\d+)["']`)
	titleRE         = regexp.MustCompile("title=[\"']([^\"']+)[\"']")
	imageOriginalRE = regexp.MustCompile("(?i)(?:data-original|data-webp|data-src|data-rd-jpg|data-lazy|data-echo|data-thumb)=[\"']([^\"']+)[\"']")
	imageRE         = regexp.MustCompile("(?i)src=[\"']([^\"']+)[\"']")
	flashURLRE      = regexp.MustCompile("(?i)(?:video_(?:alt_)?url|video_url)\\s*:\\s*['\"]([^'\"]+)['\"]")
	flashQualityRE  = regexp.MustCompile("(?i)(?:video_(?:alt_)?url_text|video_url_text)\\s*:\\s*['\"]([^'\"]+)['\"]")
	downloadRE      = regexp.MustCompile("(?i)<a[^>]+href=[\"']([^\"']+)[\"'][^>]*>\\s*MP4\\s+(\\d+)p\\s*</a>")
	durationRE      = regexp.MustCompile("(?s)custom-time[\"'].*?<span>([^<]+)</span>")
	viewsRE         = regexp.MustCompile("(?s)custom-eye[\"'].*?<span>([^<]+)</span>")
	dateRE          = regexp.MustCompile("(?s)custom-calendar[\"'].*?<span>([^<]+)</span>")
	uploadDateRE    = regexp.MustCompile("[\"']uploadDate[\"']\\s*:\\s*[\"']([^\"']+)")
	ogImageRE       = regexp.MustCompile("(?i)<meta[^>]+property=[\"']og:image[\"'][^>]+content=[\"']([^\"']+)")
	detailThumbRE   = regexp.MustCompile("(?i)(https?://[^\"'\\s]+/videos_screenshots/[^\"'\\s]+/(?:preview\\.jpg|[0-9]+x[0-9]+/[0-9]+\\.jpg))")
	modelBlockRE    = regexp.MustCompile("(?s)data-suggest-type=[\"']model[\"'].*?<a[^>]+href=[\"']([^\"']+/models/[^\"']+/)[\"'][^>]*>(.*?)</a>")
	modelItemRE     = regexp.MustCompile("(?s)data-item-type=[\"']model[\"'][^>]*>.*?<a[^>]+href=[\"']([^\"']+/models/[^\"']+/)[\"'][^>]*>(.*?)</a>")
	uploaderBlockRE = regexp.MustCompile("(?s)<div class=[\"']label[\"']>Uploaded by</div>\\s*<a[^>]+href=[\"']([^\"']+)[\"'][^>]*>(.*?)</a>")
	tagRE           = regexp.MustCompile("(?s)<a[^>]+class=[\"']tag_item[\"'][^>]+href=[\"']([^\"']+/tags/[^\"']+)[\"'][^>]*>([^<]+)</a>")
	categoryRE      = regexp.MustCompile("(?s)data-item-type=[\"']category[\"'].*?<a[^>]+href=[\"']([^\"']+/categories/[^\"']+/)[\"'][^>]*>(.*?)</a>")
	externalRE      = regexp.MustCompile("https?://[^<\\s\"']+")
	externalBlockRE = regexp.MustCompile("(?s)<em>(.*?)</em>")
	pageTitleRE     = regexp.MustCompile("(?is)<h1[^>]*>(.*?)</h1>")
	stripRE         = regexp.MustCompile("<[^>]+>")
	numberRE        = regexp.MustCompile("[0-9][0-9,]*")
	nonSlugRE       = regexp.MustCompile(`[^a-z0-9]+`)
	pageRE          = regexp.MustCompile("from:0*([0-9]+)")
)

func (Collector) SearchModels(query, proxy string) ([]ModelResult, error) {
	endpoint := "https://rule34video.com/models_json.php?advanced_search=true&q=" + url.QueryEscape(strings.TrimSpace(query))
	raw, err := fetch(endpoint, proxy)
	if err != nil {
		return nil, err
	}
	var response struct {
		Items []struct{ ID, Title, Total string } `json:"items"`
	}
	if err = json.Unmarshal([]byte(raw), &response); err != nil {
		return nil, err
	}
	out := []ModelResult{}
	for _, item := range response.Items {
		slug := strings.Trim(nonSlugRE.ReplaceAllString(strings.ToLower(item.Title), "-"), "-")
		if slug == "" {
			continue
		}
		modelURL := "https://rule34video.com/models/" + slug + "/"
		page, pageErr := fetch(modelURL, proxy)
		if pageErr != nil {
			continue
		}
		avatarRE := regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']*/contents/models/` + regexp.QuoteMeta(item.ID) + `/[^"']+)["'][^>]*>`)
		avatar := ""
		if match := avatarRE.FindStringSubmatch(page); len(match) == 2 {
			avatar = absolute(match[1])
		}
		count, _ := strconv.ParseInt(item.Total, 10, 64)
		out = append(out, ModelResult{ID: item.ID, Name: item.Title, URL: modelURL, AvatarURL: avatar, VideoCount: count})
		if len(out) >= 12 {
			break
		}
	}
	return out, nil
}

func client(proxy string, timeout time.Duration) *http.Client {
	transport := &http.Transport{MaxIdleConns: 20, MaxIdleConnsPerHost: 10, IdleConnTimeout: 90 * time.Second, ResponseHeaderTimeout: 40 * time.Second}
	if proxy != "" {
		if p, err := url.Parse(proxy); err == nil {
			transport.Proxy = http.ProxyURL(p)
		}
	}
	return &http.Client{Transport: transport, Timeout: timeout}
}
func fetch(raw, proxy string) (string, error) {
	return fetchWithClient(raw, client(proxy, 45*time.Second))
}
func fetchWithClient(raw string, httpClient *http.Client) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= 4; attempt++ {
		req, err := http.NewRequest(http.MethodGet, raw, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; r34watch/2.0)")
		res, err := httpClient.Do(req)
		if err == nil {
			body, readErr := io.ReadAll(io.LimitReader(res.Body, 25<<20))
			res.Body.Close()
			if res.StatusCode >= 200 && res.StatusCode <= 299 && readErr == nil {
				return string(body), nil
			}
			if readErr != nil {
				lastErr = readErr
			} else {
				lastErr = fmt.Errorf("source returned HTTP %d", res.StatusCode)
			}
			if res.StatusCode < 500 && res.StatusCode != http.StatusTooManyRequests {
				return "", lastErr
			}
		} else {
			lastErr = err
		}
		if attempt < 4 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
	return "", fmt.Errorf("after 4 attempts: %w", lastErr)
}
func absolute(raw string) string {
	raw = html.UnescapeString(strings.ReplaceAll(raw, "\\/", "/"))
	if strings.HasPrefix(raw, "/") {
		return "https://rule34video.com" + raw
	}
	return raw
}
func clean(raw string) string {
	return strings.TrimSpace(html.UnescapeString(stripRE.ReplaceAllString(raw, " ")))
}

// usableImage normalizes a candidate image reference and rejects lazy-load
// placeholders (data:/javascript: URIs) that are not real images.
func usableImage(raw string) string {
	value := absolute(raw)
	low := strings.ToLower(value)
	if strings.HasPrefix(low, "data:") || strings.HasPrefix(low, "javascript:") {
		return ""
	}
	if !strings.HasPrefix(low, "http://") && !strings.HasPrefix(low, "https://") {
		return ""
	}
	return value
}

func firstImage(block string) string {
	if m := imageOriginalRE.FindStringSubmatch(block); len(m) == 2 {
		if value := usableImage(m[1]); value != "" {
			return value
		}
	}
	if m := imageRE.FindStringSubmatch(block); len(m) == 2 {
		if value := usableImage(m[1]); value != "" {
			return value
		}
	}
	return ""
}
func (Collector) List(modelURL, proxy string) ([]store.Video, error) {
	seen := map[string]bool{}
	out := []store.Video{}
	pageCount := 1
	httpClient := client(proxy, 45*time.Second)
	for pageNumber := 1; pageNumber <= pageCount; pageNumber++ {
		pageURL, err := paginationURL(modelURL, pageNumber)
		if err != nil {
			return nil, err
		}
		page, err := fetchWithClient(pageURL, httpClient)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNumber, err)
		}
		if pageNumber == 1 {
			for _, p := range pageRE.FindAllStringSubmatch(page, -1) {
				n, _ := strconv.Atoi(p[1])
				if n > pageCount && n <= 500 {
					pageCount = n
				}
			}
		}
		appendListing(page, seen, &out)
	}
	return out, nil
}

func paginationURL(modelURL string, page int) (string, error) {
	u, err := url.Parse(modelURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	if page <= 1 {
		q.Del("from")
	} else {
		q.Set("from", strconv.Itoa(page))
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func appendListing(page string, seen map[string]bool, out *[]store.Video) {
	cards := videoCardRE.FindAllStringSubmatchIndex(page, -1)
	for i, card := range cards {
		id := page[card[2]:card[3]]
		if seen[id] {
			continue
		}
		start := card[0]
		end := len(page)
		if i+1 < len(cards) {
			end = cards[i+1][0]
		}
		block := page[start:end]
		m := videoRE.FindStringSubmatch(block)
		if len(m) != 4 || m[2] != id {
			continue
		}
		seen[id] = true
		slug := m[3]
		title := slug
		if x := titleRE.FindStringSubmatch(block); len(x) == 2 {
			title = html.UnescapeString(x[1])
		}
		*out = append(*out, store.Video{ID: id, Title: title, PageURL: absolute(m[1]), ThumbnailURL: firstImage(block)})
	}
}
func (Collector) Detail(v store.Video, proxy string) (store.Video, error) {
	page, err := fetch(v.PageURL, proxy)
	if err != nil {
		return v, err
	}
	return parseRaw(v, page), nil
}
func parseRaw(v store.Video, page string) store.Video {
	// Raw clipboard imports live in the imports table; normalized fields keep
	// the main video table compact for long-running NAS installations.
	v.RawHTML = ""
	if m := pageTitleRE.FindStringSubmatch(page); len(m) == 2 && clean(m[1]) != "" {
		v.Title = clean(m[1])
	}
	if m := ogImageRE.FindStringSubmatch(page); len(m) == 2 {
		if value := usableImage(m[1]); value != "" {
			v.ThumbnailURL = value
		}
	}
	if v.ThumbnailURL == "" || strings.HasPrefix(strings.ToLower(v.ThumbnailURL), "data:") {
		if m := detailThumbRE.FindStringSubmatch(page); len(m) == 2 {
			v.ThumbnailURL = absolute(m[1])
		}
	}
	if m := durationRE.FindStringSubmatch(page); len(m) == 2 {
		v.DurationSeconds = parseDuration(clean(m[1]))
	}
	if m := viewsRE.FindStringSubmatch(page); len(m) == 2 {
		v.Views = parseCount(clean(m[1]))
	}
	if m := dateRE.FindStringSubmatch(page); len(m) == 2 {
		v.PublishedAt = clean(m[1])
	}
	if m := uploadDateRE.FindStringSubmatch(page); len(m) == 2 {
		v.PublishedAt = m[1]
	}
	creators := []Link{}
	creatorSeen := map[string]bool{}
	for _, expression := range []*regexp.Regexp{modelItemRE, modelBlockRE} {
		for _, m := range expression.FindAllStringSubmatch(page, -1) {
			creatorURL := absolute(m[1])
			if creatorSeen[creatorURL] {
				continue
			}
			creatorSeen[creatorURL] = true
			creators = append(creators, Link{Name: clean(m[2]), URL: creatorURL, Image: firstImage(m[2])})
		}
	}
	if len(creators) > 0 {
		v.CreatorURL = creators[0].URL
		v.CreatorName = creators[0].Name
		v.CreatorAvatar = creators[0].Image
	}
	v.CreatorsJSON = marshal(creators)
	if m := uploaderBlockRE.FindStringSubmatch(page); len(m) == 3 {
		v.UploaderURL = absolute(m[1])
		v.UploaderName = clean(m[2])
		v.UploaderAvatar = firstImage(m[2])
	}
	tags := []Link{}
	for _, m := range tagRE.FindAllStringSubmatch(page, -1) {
		tags = append(tags, Link{Name: clean(m[2]), URL: absolute(m[1])})
	}
	v.TagsJSON = marshal(tags)
	cats := []Link{}
	for _, m := range categoryRE.FindAllStringSubmatch(page, -1) {
		cats = append(cats, Link{Name: clean(m[2]), URL: absolute(m[1]), Image: firstImage(m[2])})
	}
	v.CategoriesJSON = marshal(cats)
	links := []Link{}
	externalSource := ""
	if m := externalBlockRE.FindStringSubmatch(page); len(m) == 2 {
		externalSource = m[1]
	}
	for _, raw := range externalRE.FindAllString(externalSource, -1) {
		raw = html.UnescapeString(raw)
		if !strings.Contains(raw, "rule34video.com") {
			links = appendUnique(links, Link{Name: host(raw), URL: raw})
		}
	}
	v.ExternalLinksJSON = marshal(links)
	return v
}
func (Collector) Highest(pageURL, proxy string) (string, int64, error) {
	page, err := fetch(pageURL, proxy)
	if err != nil {
		return "", 0, err
	}
	best := ""
	quality := -1
	for _, m := range downloadRE.FindAllStringSubmatch(page, -1) {
		q, _ := strconv.Atoi(m[2])
		if q > quality {
			best, quality = absolute(m[1]), q
		}
	}
	if best != "" {
		return best, int64(quality), nil
	}
	urls, labels := flashURLRE.FindAllStringSubmatch(page, -1), flashQualityRE.FindAllStringSubmatch(page, -1)
	for i, m := range urls {
		q := 0
		if i < len(labels) {
			q = int(parseCount(labels[i][1]))
		}
		if q > quality {
			best, quality = absolute(m[1]), q
		}
	}
	if best == "" {
		return "", 0, errors.New("no downloadable source found")
	}
	return best, int64(quality), nil
}
func (Collector) ParseImported(raw string) (store.Video, error) {
	id := ""
	if m := regexp.MustCompile("data-video-id=[\"'](\\d+)").FindStringSubmatch(raw); len(m) == 2 {
		id = m[1]
	}
	if id == "" {
		return store.Video{}, errors.New("HTML 中没有 video id")
	}
	v := store.Video{ID: id, RawHTML: raw, PageURL: "https://rule34video.com/video/" + id + "/", DiscoveredAt: store.Now()}
	return parseRaw(v, raw), nil
}
func parseDuration(v string) int64 {
	parts := strings.Split(v, ":")
	var total int64
	for _, p := range parts {
		n, _ := strconv.ParseInt(strings.TrimSpace(p), 10, 64)
		total = total*60 + n
	}
	return total
}
func parseCount(v string) int64 {
	matches := numberRE.FindAllString(v, -1)
	if len(matches) == 0 {
		return 0
	}
	candidate := matches[len(matches)-1]
	n, _ := strconv.ParseInt(strings.ReplaceAll(candidate, ",", ""), 10, 64)
	return n
}
func marshal(v any) string   { b, _ := json.Marshal(v); return string(b) }
func host(raw string) string { u, _ := url.Parse(raw); return u.Host }
func appendUnique(items []Link, item Link) []Link {
	for _, v := range items {
		if v.URL == item.URL {
			return items
		}
	}
	return append(items, item)
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
