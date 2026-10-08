package main

import (
	"errors"
	"log"
	"sort"
	"sync"
	"time"

	"r34watch/internal/collector"
	"r34watch/internal/store"
)

const (
	rateLimitBase          = 30 * time.Second
	rateLimitMax           = 15 * time.Minute
	rateLimitReset         = 10 * time.Minute
	rateLimitMaxRetryAfter = 24 * time.Hour
	maxRateLimitRetries    = 5
)

const rateLimitExhaustedMessage = "rate_limit_retry_exhausted: 来源持续限速，已停止自动重试；可手动重新下载"

type rateLimitEntry struct {
	source      string
	host        string
	consecutive int
	until       time.Time
	lastAt      time.Time
	retryAfter  time.Duration
}

// rateLimiter tracks HTTP 429 backoff per source (adapter/platform) so a limit
// on one platform no longer stalls downloads from another. State is persisted
// in SQLite so a restart keeps respecting an unexpired backoff.
type rateLimiter struct {
	mu     sync.Mutex
	store  *store.Store
	states map[string]*rateLimitEntry
}

func newRateLimiter(s *store.Store) *rateLimiter {
	r := &rateLimiter{store: s, states: map[string]*rateLimitEntry{}}
	r.load()
	return r
}

func (r *rateLimiter) load() {
	rows, err := r.store.RateLimits()
	if err != nil {
		log.Printf("rate limit load: %v", err)
		return
	}
	now := time.Now()
	for _, row := range rows {
		until, _ := time.Parse(time.RFC3339, row.BackoffUntil)
		lastAt, _ := time.Parse(time.RFC3339, row.Last429At)
		if !until.IsZero() && !until.After(now) {
			_ = r.store.DeleteRateLimit(row.SourceKey)
			continue
		}
		r.states[row.SourceKey] = &rateLimitEntry{
			source:      row.SourceKey,
			host:        row.Host,
			consecutive: row.Consecutive,
			until:       until,
			lastAt:      lastAt,
			retryAfter:  time.Duration(row.RetryAfterSeconds) * time.Second,
		}
	}
}

// Wait returns the remaining backoff for a source, or 0 when it may run.
func (r *rateLimiter) Wait(source string) time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	e := r.states[source]
	if e == nil {
		return 0
	}
	now := time.Now()
	if !e.lastAt.IsZero() && now.Sub(e.lastAt) > rateLimitReset {
		r.deleteLocked(source)
		return 0
	}
	if e.until.After(now) {
		return e.until.Sub(now)
	}
	r.deleteLocked(source)
	return 0
}

func (r *rateLimiter) deleteLocked(source string) {
	delete(r.states, source)
	_ = r.store.DeleteRateLimit(source)
}

// Record registers a 429 for a source and returns the new wait duration. A
// valid Retry-After is honored even beyond the local exponential cap so HWatch
// never retries earlier than the server asked.
func (r *rateLimiter) Record(source, host string, retryAfter time.Duration, hasRetryAfter bool) time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	if source == "" {
		source = "unknown"
	}
	e := r.states[source]
	if e == nil {
		e = &rateLimitEntry{source: source}
		r.states[source] = e
	}
	now := time.Now()
	e.consecutive++
	e.lastAt = now
	if host != "" {
		e.host = host
	}
	var delay time.Duration
	if hasRetryAfter && retryAfter > 0 {
		delay = retryAfter
		if delay > rateLimitMaxRetryAfter {
			delay = rateLimitMaxRetryAfter
		}
		e.retryAfter = delay
	} else {
		shift := e.consecutive - 1
		if shift > 5 {
			shift = 5
		}
		delay = rateLimitBase * time.Duration(1<<uint(shift))
		if delay > rateLimitMax {
			delay = rateLimitMax
		}
		e.retryAfter = 0
	}
	if until := now.Add(delay); until.After(e.until) {
		e.until = until
	}
	_ = r.store.SaveRateLimit(store.RateLimitState{
		SourceKey:         e.source,
		Host:              e.host,
		Consecutive:       e.consecutive,
		BackoffUntil:      e.until.UTC().Format(time.RFC3339),
		Last429At:         e.lastAt.UTC().Format(time.RFC3339),
		RetryAfterSeconds: int(e.retryAfter.Seconds()),
	})
	return time.Until(e.until)
}

// Snapshot lists sources currently in backoff, sorted by source key.
func (r *rateLimiter) Snapshot() []rateLimitEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	out := []rateLimitEntry{}
	for _, e := range r.states {
		if e.until.After(now) {
			out = append(out, *e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].source < out[j].source })
	return out
}

func sourceRateLimitFromError(err error) (host string, retryAfter time.Duration, hasRetryAfter bool) {
	var rle *collector.RateLimitError
	if errors.As(err, &rle) {
		return rle.Host, rle.RetryAfter, rle.HasRetryAfter
	}
	return "", 0, false
}
