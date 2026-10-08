package main

import (
	"path/filepath"
	"testing"
	"time"

	"r34watch/internal/store"
)

func newRateLimitTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "rl.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestRateLimitBackoffIsExponentialAndCapped(t *testing.T) {
	r := newRateLimiter(newRateLimitTestStore(t))
	want := []time.Duration{30 * time.Second, time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 15 * time.Minute, 15 * time.Minute}
	for i, expected := range want {
		got := r.Record("rule34video", "rule34video.com", 0, false)
		if diff := got - expected; diff < -2*time.Second || diff > 2*time.Second {
			t.Fatalf("attempt %d: got %s want ~%s", i+1, got, expected)
		}
	}
	if wait := r.Wait("rule34video"); wait <= 0 || wait > 15*time.Minute {
		t.Fatalf("invalid active wait %s", wait)
	}
	// A different source is unaffected by rule34video's backoff.
	if wait := r.Wait("hanime1"); wait != 0 {
		t.Fatalf("hanime1 must not be rate limited: %s", wait)
	}
	// Quiet for longer than the reset window clears the counter.
	r.states["rule34video"].lastAt = time.Now().Add(-11 * time.Minute)
	if wait := r.Wait("rule34video"); wait != 0 {
		t.Fatalf("stale backoff was not reset: %s", wait)
	}
}

func TestRateLimitHonorsRetryAfterBeyondCap(t *testing.T) {
	r := newRateLimiter(newRateLimitTestStore(t))
	// Server asks for 20 minutes; the local cap is 15, but the server wins.
	got := r.Record("rule34video", "rule34video.com", 20*time.Minute, true)
	if got < 19*time.Minute {
		t.Fatalf("Retry-After not honored: %s", got)
	}
	if wait := r.Wait("rule34video"); wait < 19*time.Minute {
		t.Fatalf("wait=%s", wait)
	}
}

func TestRateLimitPersistsAcrossRestart(t *testing.T) {
	s := newRateLimitTestStore(t)
	first := newRateLimiter(s)
	first.Record("rule34video", "rule34video.com", 10*time.Minute, true)
	// Simulate a service restart: a fresh limiter loads state from SQLite.
	second := newRateLimiter(s)
	if wait := second.Wait("rule34video"); wait < 9*time.Minute {
		t.Fatalf("persisted backoff not restored: %s", wait)
	}
	if wait := second.Wait("hanime1"); wait != 0 {
		t.Fatalf("unrelated source limited after restore: %s", wait)
	}
}

func TestPickRunnableSkipsLimitedSource(t *testing.T) {
	s := newRateLimitTestStore(t)
	a := New(s)
	a.rateLimits.Record("rule34video", "rule34video.com", 10*time.Minute, true)
	candidates := []store.Video{
		{ID: "r1", Platform: "rule34video"},
		{ID: "r2", Platform: "rule34video"},
		{ID: "h1", Platform: "hanime1"},
		{ID: "h2", Platform: "hanime1"},
	}
	picked, _ := a.pickRunnable(candidates)
	if picked == nil || picked.ID != "h1" {
		t.Fatalf("expected to skip the limited source and pick h1, got %+v", picked)
	}
	// With only limited sources, no task is picked and a wait is returned.
	limited := []store.Video{{ID: "r3", Platform: "rule34video"}}
	picked, wait := a.pickRunnable(limited)
	if picked != nil || wait <= 0 {
		t.Fatalf("expected wait for limited-only queue, got picked=%v wait=%s", picked, wait)
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
