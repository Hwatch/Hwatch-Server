package main

import (
	"testing"

	"r34watch/internal/store"
)

func TestExclusionReason(t *testing.T) {
	sub := store.Subscription{URL: "https://source.example/models/akt/"}
	cfg := store.Settings{RequireCreatorMatch: true, ExcludeTitleKeywords: "compilation,合集", ExcludeTags: "blocked-tag"}

	wrongCreator := store.Video{CreatorURL: "https://source.example/models/other/", CreatorName: "Other"}
	if got := exclusionReason(wrongCreator, sub, cfg); got == "" {
		t.Fatal("expected mismatched creator to be excluded")
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
