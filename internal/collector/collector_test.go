package collector

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"r34watch/internal/store"
)

func TestListFollowsAllDeclaredPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("from")
		if page == "" {
			page = "1"
		}
		fmt.Fprintf(w, `<a data-parameters="from:01"></a><a data-parameters="from:03"></a><div class="item thumb" data-video-card-id="%s"><a class="th" href="/video/%s/item-%s/" title="Item %s"><img src="item.jpg"></a></div>`, page, page, page, page)
	}))
	defer server.Close()
	videos, err := (Collector{}).List(server.URL+"/models/test/?sort=new", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(videos) != 3 || videos[0].ID != "1" || videos[2].ID != "3" {
		t.Fatalf("expected all three pages, got %+v", videos)
	}
}

func TestListIgnoresVideoLinksOutsideListingCards(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<a href="/video/999/advert-or-recommendation/">outside</a><div data-video-card-id="123"><a class="th" href="/video/123/real-item/" title="Real item"></a></div>`))
	}))
	defer server.Close()
	videos, err := (Collector{}).List(server.URL+"/models/test/", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(videos) != 1 || videos[0].ID != "123" {
		t.Fatalf("expected only the listing card, got %+v", videos)
	}
}

func TestFetchRetriesTemporaryServerFailure(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			http.Error(w, "temporary", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	body, err := fetchWithClient(server.URL, &http.Client{Timeout: 5 * time.Second})
	if err != nil || body != "ok" || attempts != 3 {
		t.Fatalf("body=%q attempts=%d err=%v", body, attempts, err)
	}
}

func TestParseImportedMetadata(t *testing.T) {
	raw := "<h1>Episode title</h1><div data-video-id=\"4419929\"></div>" +
		"<svg class=\"custom-calendar\"></svg><span>2026-06-01</span>" +
		"<svg class=\"custom-eye\"></svg><span>218K (218,417)</span>" +
		"<svg class=\"custom-time\"></svg><span>4:53</span>" +
		"<div data-suggest-type=\"model\"><a href=\"https://rule34video.com/models/akt/\"><img data-original=\"artist.jpg\"><span>Akt</span></a></div>" +
		"<div class=\"label\">Uploaded by</div><a href=\"https://rule34video.com/members/1/\">Uploader</a>" +
		"<a class=\"tag_item\" href=\"https://rule34video.com/tags/1/\">tag-a</a>"
	v, err := Collector{}.ParseImported(raw)
	if err != nil {
		t.Fatal(err)
	}
	if v.ID != "4419929" || v.Title != "Episode title" || v.Views != 218417 || v.DurationSeconds != 293 || v.CreatorName != "Akt" {
		t.Fatalf("unexpected metadata: %+v", v)
	}
	if v.CreatorsJSON == "" || v.CreatorsJSON == "[]" {
		t.Fatal("expected creator relationships")
	}
	if v.RawHTML != "" {
		t.Fatal("normalized video row must not duplicate raw import HTML")
	}
}
func TestParseCount(t *testing.T) {
	if got := parseCount("218K (218,417)"); got != 218417 {
		t.Fatalf("got %d", got)
	}
}

func TestModelVideoCountUsesPageTotal(t *testing.T) {
	page := `<h1>Theobrobine <span class="total_results">(147)</span></h1><div total="147" data-sort="post_date">`
	if got := modelVideoCount(page, 829); got != 147 {
		t.Fatalf("page total must win over models_json total, got %d", got)
	}
	if got := modelVideoCount(`<html>no count here</html>`, 829); got != 829 {
		t.Fatalf("fallback to models_json total, got %d", got)
	}
	// A model page whose real count happens to be zero should not be ignored.
	if got := modelVideoCount(`<span class="total_results">(0)</span>`, 5); got != 0 {
		t.Fatalf("zero count must be honored, got %d", got)
	}
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Now()
	if d, ok := parseRetryAfter("120", now); !ok || d != 120*time.Second {
		t.Fatalf("seconds: d=%s ok=%v", d, ok)
	}
	future := now.Add(90 * time.Second).UTC().Format(http.TimeFormat)
	if d, ok := parseRetryAfter(future, now); !ok || d < 80*time.Second || d > 95*time.Second {
		t.Fatalf("http-date: d=%s ok=%v", d, ok)
	}
	if _, ok := parseRetryAfter("-5", now); ok {
		t.Fatal("negative Retry-After must be rejected")
	}
	if _, ok := parseRetryAfter("garbage", now); ok {
		t.Fatal("garbage Retry-After must be rejected")
	}
	if d, ok := parseRetryAfter("999999999", now); !ok || d != 24*time.Hour {
		t.Fatalf("huge Retry-After must clamp to 24h: d=%s ok=%v", d, ok)
	}
}

func TestFetchReports429WithoutInternalRetry(t *testing.T) {
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	_, err := fetchWithClient(server.URL+"/page", server.Client())
	var rle *RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected RateLimitError, got %v", err)
	}
	if !rle.HasRetryAfter || rle.RetryAfter != 120*time.Second {
		t.Fatalf("Retry-After not captured: %+v", rle)
	}
	if hits != 1 {
		t.Fatalf("must not retry 429 internally, hits=%d", hits)
	}
}

func TestFirstImageRejectsLazyPlaceholder(t *testing.T) {
	if got := firstImage(`<img src="data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7">`); got != "" {
		t.Fatalf("placeholder must be rejected, got %q", got)
	}
}

func TestFirstImageUsesLazyAttribute(t *testing.T) {
	block := `<img class="thumb" src="data:image/gif;base64,AAA" data-rd-jpg="https://rule34video.com/contents/videos_screenshots/1/2/320x180/3.jpg">`
	want := "https://rule34video.com/contents/videos_screenshots/1/2/320x180/3.jpg"
	if got := firstImage(block); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDetailThumbnailFallbackWithoutOpenGraph(t *testing.T) {
	page := `<html><body><div class="img wrap_image"><img class="thumb" src="https://rule34video.com/contents/videos_screenshots/4629000/4629583/336x189/3.jpg"></div></body></html>`
	v := parseRaw(store.Video{ID: "1", ThumbnailURL: "data:image/gif;base64,AAA"}, page)
	want := "https://rule34video.com/contents/videos_screenshots/4629000/4629583/336x189/3.jpg"
	if v.ThumbnailURL != want {
		t.Fatalf("got %q want %q", v.ThumbnailURL, want)
	}
}

func TestDetailCandidatesPreferPreviewAndFilterRelated(t *testing.T) {
	page := `<img src="https://rule34video.com/contents/videos_screenshots/9999000/9999999/320x180/1.jpg">` +
		`<img src="https://rule34video.com/contents/videos_screenshots/3504000/3504948/320x180/3.jpg">` +
		`<img src="https://rule34video.com/contents/videos_screenshots/3504000/3504948/preview.jpg">` +
		`<img src="https://rule34video.com/contents/videos_screenshots/3504000/3504948/preview_360.mp4.jpg">`
	candidates := detailThumbnailCandidates(page, "3504948")
	if len(candidates) == 0 || !strings.HasSuffix(candidates[0], "/3504948/preview.jpg") {
		t.Fatalf("preview.jpg must rank first, got %v", candidates)
	}
	for _, c := range candidates {
		if strings.Contains(c, "9999999") {
			t.Fatalf("related-video asset leaked: %v", candidates)
		}
	}
}

func TestParseRawUpgradesListThumbnailToDetailPreview(t *testing.T) {
	list := "https://rule34video.com/contents/videos_screenshots/3504000/3504948/320x180/3.jpg"
	page := `<img src="` + list + `"><img src="https://rule34video.com/contents/videos_screenshots/3504000/3504948/preview.jpg">`
	v := parseRaw(store.Video{ID: "3504948", ThumbnailURL: list}, page)
	if !strings.HasSuffix(v.ThumbnailURL, "/3504948/preview.jpg") {
		t.Fatalf("list thumbnail was not upgraded: %q", v.ThumbnailURL)
	}
	kept := parseRaw(store.Video{ID: "3504948", ThumbnailURL: v.ThumbnailURL}, page)
	if kept.ThumbnailURL != v.ThumbnailURL {
		t.Fatalf("existing preview.jpg must not change: %q", kept.ThumbnailURL)
	}
}

func TestParseRawOpenGraphWins(t *testing.T) {
	page := `<meta property="og:image" content="https://cdn.example/cover.png"><img src="https://rule34video.com/contents/videos_screenshots/1/2/preview.jpg">`
	v := parseRaw(store.Video{ID: "2", ThumbnailURL: "https://rule34video.com/contents/videos_screenshots/1/2/320x180/1.jpg"}, page)
	if v.ThumbnailURL != "https://cdn.example/cover.png" {
		t.Fatalf("og:image should win, got %q", v.ThumbnailURL)
	}
}

func TestCreatorListingSnapshot(t *testing.T) {
	raw, err := os.ReadFile("testdata/creator_listing.html")
	if err != nil {
		t.Fatal(err)
	}
	out := []store.Video{}
	appendListing(string(raw), map[string]bool{}, &out)
	if len(out) != 1 || out[0].ID != "4429909" || out[0].Title != "Example & Video" || out[0].ThumbnailURL != "https://rule34video.com/thumbs/example.jpg" {
		t.Fatalf("listing parser contract changed: %+v", out)
	}
}

func TestVideoDetailSnapshot(t *testing.T) {
	raw, err := os.ReadFile("testdata/video_detail.html")
	if err != nil {
		t.Fatal(err)
	}
	v := parseRaw(store.Video{ID: "4429909"}, string(raw))
	if v.Title != "Snapshot Episode" || v.Views != 12345 || v.DurationSeconds != 201 || v.CreatorName != "Akt" || v.UploaderName != "Uploader" {
		t.Fatalf("detail parser contract changed: %+v", v)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(raw) }))
	defer server.Close()
	u, quality, err := (Collector{}).Highest(server.URL, "")
	if err != nil || quality != 2160 || u != "https://cdn.example/video-2160.mp4" {
		t.Fatalf("source=%q quality=%d err=%v", u, quality, err)
	}
}

func highestFromHTML(t *testing.T, page string) (string, int64) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(page)) }))
	defer server.Close()
	u, q, err := (Collector{}).Highest(server.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	return u, q
}

func downloadPanel(items ...string) string {
	page := ""
	for _, it := range items {
		page += `<a class="tag_item tag_item_download vp-sheet__item" href="https://rule34video.com/get_file/51/x/1/2/video_` + it + `p.mp4/?v-acctoken=token-` + it + `&download=true" data-attach-session="PHPSESSID" role="menuitem">MP4 ` + it + `p</a>`
	}
	return page
}

func TestHighestSelectsDownloadPanelBest(t *testing.T) {
	u, q := highestFromHTML(t, downloadPanel("360", "480", "720", "1080"))
	if q != 1080 || !strings.Contains(u, "video_1080p.mp4") {
		t.Fatalf("source=%q quality=%d, want 1080p download panel", u, q)
	}
}

func TestHighestIgnoresLowResPlayerWhenDownloadPanelPresent(t *testing.T) {
	page := `<script>video_url: 'https://rule34video.com/get_file/51/x/1/2/video_360.mp4', video_url_text: '360p', video_alt_url: 'https://rule34video.com/get_file/51/x/1/2/video_480.mp4', video_alt_url_text: '480p',</script>` + downloadPanel("1080")
	u, q := highestFromHTML(t, page)
	if q != 1080 || !strings.Contains(u, "video_1080p.mp4") {
		t.Fatalf("source=%q quality=%d, want 1080p over 270x480 player preview", u, q)
	}
}

func TestHighestVertical1080NotDownrankedByOrientation(t *testing.T) {
	page := `<img class="thumb" src="https://rule34video.com/contents/videos_screenshots/1/2/preview_270x480.mp4.jpg">` + downloadPanel("1080", "720")
	u, q := highestFromHTML(t, page)
	if q != 1080 {
		t.Fatalf("quality=%d source=%q, vertical 1080p must not be downranked", q, u)
	}
}

func TestHighestSelects4KWhenOffered(t *testing.T) {
	u, q := highestFromHTML(t, downloadPanel("1080", "2160"))
	if q != 2160 || !strings.Contains(u, "video_2160p.mp4") {
		t.Fatalf("source=%q quality=%d, want 2160p", u, q)
	}
}

func TestHighestFallsBackToLegacyParser(t *testing.T) {
	page := `<a href="https://rule34video.com/get_file/51/x/1/2/video_720.mp4">MP4 720p</a><script>video_url: 'https://rule34video.com/get_file/51/x/1/2/video_360.mp4', video_url_text: '360p'</script>`
	u, q := highestFromHTML(t, page)
	if q != 720 || u != "https://rule34video.com/get_file/51/x/1/2/video_720.mp4" {
		t.Fatalf("legacy fallback source=%q quality=%d", u, q)
	}
}

func TestHighestReadsIndexedAltRenditions(t *testing.T) {
	page := `<script>video_url: 'https://rule34video.com/get_file/51/x/1/2/video_360.mp4', video_url_text: '360p', video_alt_url: 'https://rule34video.com/get_file/51/x/1/2/video_480.mp4', video_alt_url_text: '480p', video_alt_url2: 'https://rule34video.com/get_file/51/x/1/2/video_720p.mp4', video_alt_url2_text: '720p', video_alt_url3: 'https://rule34video.com/get_file/51/x/1/2/video_1080p.mp4', video_alt_url3_text: '1080p', video_alt_url3_hd: '1'</script>`
	u, q := highestFromHTML(t, page)
	if q != 1080 || !strings.Contains(u, "video_1080p.mp4") {
		t.Fatalf("indexed alt renditions source=%q quality=%d, want 1080p", u, q)
	}
}

func TestHighestRankMixedNamingWithoutP(t *testing.T) {
	page := `<a class="tag_item_download" href="https://rule34video.com/get_file/51/x/1/2/video_360.mp4">MP4 360p</a>` +
		`<a class="tag_item_download" href="https://rule34video.com/get_file/51/x/1/2/video_1080p.mp4">MP4 1080p</a>`
	u, q := highestFromHTML(t, page)
	if q != 1080 || !strings.Contains(u, "video_1080p.mp4") {
		t.Fatalf("mixed naming source=%q quality=%d, want 1080p", u, q)
	}
}

func TestHighestPreservesSignedQuery(t *testing.T) {
	page := `<a class="tag_item_download" href="https://rule34video.com/get_file/51/x/1/2/video_1080p.mp4/?v-acctoken=SECRET&amp;download=true&amp;download_filename=clip_1080p.mp4">MP4 1080p</a>`
	u, q := highestFromHTML(t, page)
	if q != 1080 {
		t.Fatalf("quality=%d", q)
	}
	if strings.Contains(u, "&amp;") || !strings.Contains(u, "v-acctoken=SECRET&download=true&download_filename=clip_1080p.mp4") {
		t.Fatalf("signed query not preserved: %q", u)
	}
}

// TestListDetectsTemplateChange ensures a structurally changed or anti-bot
// page surfaces a loud TemplateError instead of a silent empty listing.
func TestListDetectsTemplateChange(t *testing.T) {
	cases := []struct{ name, body string }{
		{"recognized-container-no-cards", `<div class="thumbs"><div class="item "></div></div>`},
		{"anti-bot-challenge", `<html><head><title>Just a moment...</title></head><body><div id="cf-challenge-running"></div></body></html>`},
		{"unknown-layout", `<html><body><h1>Something else</h1><p>no listing here</p></body></html>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			_, err := (Collector{}).List(server.URL+"/models/x/", "")
			var te *TemplateError
			if !errors.As(err, &te) {
				t.Fatalf("expected TemplateError, got %v", err)
			}
			if !strings.Contains(err.Error(), "抓取模板检测失败") {
				t.Fatalf("template error message not explicit: %v", err)
			}
		})
	}
}

func TestLooksLikeAntiBot(t *testing.T) {
	if !LooksLikeAntiBot(`<title>Just a moment...</title>`) {
		t.Fatal("should detect Cloudflare interstitial")
	}
	if LooksLikeAntiBot(`<div data-video-card-id="1"></div>`) {
		t.Fatal("normal listing must not be flagged as anti-bot")
	}
}
