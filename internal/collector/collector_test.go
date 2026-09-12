package collector

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
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
