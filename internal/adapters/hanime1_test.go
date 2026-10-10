package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"r34watch/internal/collector"
	"r34watch/internal/store"
)

func TestHanime1ParseSubscriptionKinds(t *testing.T) {
	a := Hanime1{}
	playlist, err := a.ParseSubscription(context.Background(), "https://hanime1.me/playlist?list=890459&sort=latest", "")
	if err != nil || playlist.Slug != "hanime1:playlist:890459" || playlist.Type != "playlist" {
		t.Fatalf("playlist = %#v, %v", playlist, err)
	}
	creator, err := a.ParseSubscription(context.Background(), "https://hanime1.me/user/1234893/uploaded", "")
	if err != nil || creator.Slug != "hanime1:creator:1234893" || creator.Type != "creator" {
		t.Fatalf("creator = %#v, %v", creator, err)
	}
}

func TestHanime1DetailMetadataPatterns(t *testing.T) {
	page := `<div class="video-description-panel"><div class="hidden-xs">观看次数：37.7万次&nbsp;&nbsp;2026-06-25</div><div class="video-caption-text caption-ellipsis">Title / タイトル: Demo
Brand / ブランド: Studio
Release / 販売日: 2026/06/24
https://www.patreon.com/Studio/posts/demo</div></div>
<button id="video-like-btn"><i>thumb_up</i>99%&nbsp;&nbsp;<span>(1866)</span></button>
<div class="video-tags-wrapper">
<div class="single-video-tag"><a href="/search?query=Re:Zero"><span>#</span>&nbsp;Re:Zero</a></div>
<div class="single-video-tag"><a href="/search?tags%5B%5D=百合">百合&nbsp;<span>(1)</span></a></div>
</div>`
	descriptionMatch := hanimeDescriptionRE.FindStringSubmatch(page)
	if len(descriptionMatch) != 2 || hanimeMultilineText(descriptionMatch[1]) == "" {
		t.Fatal("description was not parsed")
	}
	views := hanimeDetailViewsRE.FindStringSubmatch(hanimeClean(page))
	if len(views) != 3 || hanimeCount(views[1], views[2]) != 377000 {
		t.Fatalf("views = %#v", views)
	}
	likes := hanimeLikeRE.FindStringSubmatch(page)
	if len(likes) != 3 || likes[1] != "99" || likes[2] != "1866" {
		t.Fatalf("likes = %#v", likes)
	}
	var tags []map[string]string
	if err := json.Unmarshal([]byte(hanimeLinksJSON(page, hanimeTagRE)), &tags); err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 || tags[0]["name"] != "Re:Zero" || tags[1]["name"] != "百合" {
		t.Fatalf("tags = %#v", tags)
	}
}

func TestHanime1ParseListing(t *testing.T) {
	page := `<div class="playlist-video-card video-item-container"><a href="https://hanime1.me/watch?v=408216"><img class="main-thumb" src="https://cdn.example/408216l.jpg"><div class="duration">06:37</div><div class="stat-item">1.4万次</div></a><h4 class="video-title"><a href="https://hanime1.me/watch?v=408216">第 8 集</a></h4></div>`
	sub := store.Subscription{Slug: "hanime1:playlist:890459", Platform: "hanime1", Type: "playlist", SourceID: "890459", URL: "https://hanime1.me/playlist?list=890459&sort=latest"}
	items, err := hanimeParseListing(page, sub)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "hanime1:408216" || items[0].Title != "第 8 集" || items[0].DurationSeconds != 397 || items[0].Views != 14000 {
		t.Fatalf("items = %#v", items)
	}
	if items[0].SourceRelationJSON == "" {
		t.Fatal("source relation is empty")
	}
}

// TestHanime1ListingTemplateChange ensures a changed or anti-bot page yields a
// loud TemplateError rather than a silent empty listing.
func TestHanime1ListingTemplateChange(t *testing.T) {
	sub := store.Subscription{Slug: "hanime1:playlist:1", Platform: "hanime1", Type: "playlist", SourceID: "1"}
	for _, page := range []string{
		`<html><body><h1>Something new</h1><div class="totally-different-card"></div></body></html>`,
		`<html><head><title>Just a moment...</title></head><body><div id="cf-challenge-running"></div></body></html>`,
	} {
		_, err := hanimeParseListing(page, sub)
		var te *collector.TemplateError
		if !errors.As(err, &te) {
			t.Fatalf("expected TemplateError for %q, got %v", page, err)
		}
	}
}

func TestHanime1DirectMP4ChoosesBestAndUnescapes(t *testing.T) {
	page := `<a class="download" data-url="https://cdne-mobile.youjizz.com/videos/a/9/9/6/3/x-240-360-865-h264.mp4?validfrom=1&amp;hash=AAA%3D" download="t">240</a>` +
		`<a class="download" data-url="https://cdne-mobile.youjizz.com/videos/a/9/9/6/3/x-1080-1620-4637-h264.mp4?validfrom=1&amp;hash=BBB%3D" download="t">1080</a>`
	src, ok := hanimeDirectMP4(page, "https://hanime1.me/download?v=1")
	if !ok {
		t.Fatal("expected a direct MP4 to be found")
	}
	if src.Quality != 1620 {
		t.Fatalf("quality=%d want 1620", src.Quality)
	}
	if !strings.Contains(src.URL, "-1080-1620-4637-h264.mp4") || !strings.Contains(src.URL, "&hash=BBB%3D") {
		t.Fatalf("picked/unescaped URL wrong: %s", src.URL)
	}
	if src.Headers["Referer"] != "https://hanime1.me/download?v=1" {
		t.Fatalf("referer not set: %v", src.Headers)
	}
}

func TestHanime1NumericID(t *testing.T) {
	if got := hanimeNumericID(store.Video{SourceID: "39837"}); got != "39837" {
		t.Fatalf("SourceID id=%q", got)
	}
	if got := hanimeNumericID(store.Video{PageURL: "https://hanime1.me/watch?v=39915"}); got != "39915" {
		t.Fatalf("URL id=%q", got)
	}
	if got := hanimeNumericID(store.Video{PageURL: "https://hanime1.me/playlist?list=1"}); got != "" {
		t.Fatalf("expected empty id, got %q", got)
	}
}
