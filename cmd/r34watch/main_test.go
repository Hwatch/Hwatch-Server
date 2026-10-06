package main

import (
	"testing"
	"time"

	"r34watch/internal/store"
)

func TestRateLimitBackoffIsExponentialAndCapped(t *testing.T) {
	a := &App{}
	want := []time.Duration{30 * time.Second, time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 15 * time.Minute, 15 * time.Minute}
	for i, expected := range want {
		if got := a.record429(); got != expected {
			t.Fatalf("attempt %d: got %s want %s", i+1, got, expected)
		}
	}
	if wait := a.rateLimitWait(); wait <= 0 || wait > 15*time.Minute {
		t.Fatalf("invalid active wait %s", wait)
	}
	a.rateMu.Lock()
	a.last429 = time.Now().Add(-11 * time.Minute)
	a.rateMu.Unlock()
	if wait := a.rateLimitWait(); wait != 0 {
		t.Fatalf("stale backoff was not reset: %s", wait)
	}
}

func TestExclusionReason(t *testing.T) {
	sub := store.Subscription{Slug: "source:akt", Platform: "source", URL: "https://source.example/models/akt/"}
	cfg := store.Settings{RequireCreatorMatch: true, ExcludeTitleKeywords: "compilation,合集", ExcludeTags: "blocked-tag"}

	wrongCreator := store.Video{CreatorURL: "https://source.example/models/other/", CreatorName: "Other"}
	if got := exclusionReason(wrongCreator, sub, cfg); got == "" {
		t.Fatal("expected mismatched creator to be excluded")
	}
	trustedListing := wrongCreator
	trustedListing.SourceRelationJSON = `{"type":"creator_listing","platform":"source","subscription_slug":"source:akt"}`
	if got := exclusionReason(trustedListing, sub, cfg); got != "" {
		t.Fatalf("trusted subscription listing must not be rejected by primary creator: %q", got)
	}
	titled := store.Video{CreatorURL: sub.URL, Title: "2026 Compilation"}
	if got := exclusionReason(titled, sub, cfg); got != "标题命中排除规则" {
		t.Fatalf("unexpected title result: %q", got)
	}
	tagged := store.Video{CreatorURL: sub.URL, TagsJSON: `[{"Name":"blocked-tag"}]`}
	if got := exclusionReason(tagged, sub, cfg); got != "标签命中排除规则" {
		t.Fatalf("unexpected tag result: %q", got)
	}
	allowed := store.Video{CreatorURL: sub.URL, Title: "Regular episode", TagsJSON: `[]`}
	if got := exclusionReason(allowed, sub, cfg); got != "" {
		t.Fatalf("expected allowed video, got %q", got)
	}
}
