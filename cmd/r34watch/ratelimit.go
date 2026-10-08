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
	rateLimitBase           = 30 * time.Second
	rateLimitMax            = 15 * time.Minute
	rateLimitReset          = 10 * time.Minute
	rateLimitMaxRetryAfter  = 24 * time.Hour
	maxRateLimitRetries     = 5
	maxSourceExpiredRetries = 5

	transientThreshold = 3
	transientBase      = 60 * time.Second
	transientMax       = 10 * time.Minute
)

// downloaderOfflineRetry is how long the persistent queue worker waits before
// retrying after a transient downloader/DB error, instead of exiting and
// stalling every queued task until the next enqueue event.
var downloaderOfflineRetry = 15 * time.Second

const rateLimitExhaustedMessage = "rate_limit_retry_exhausted: 来源持续限速，已停止自动重试；可手动重新下载"

const sourceExpiredExhaustedMessage = "source_retry_exhausted: 下载签名持续失效或 CDN 拒绝，已停止自动重试并保留部分文件；可手动重试"

type rateLimitEntry struct {
	source      string
	host        string
	consecutive int
	transient   int
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
			transient:   row.Transient,
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
		Transient:         e.transient,
	})
	return time.Until(e.until)
}

// RecordTransient registers a connection-level failure (TLS/EOF/reset). A single
// blip is tolerated; the source is only paused after transientThreshold
// consecutive failures, with a gentler schedule than 429, so one flaky CDN
// window does not burn through the queue.
func (r *rateLimiter) RecordTransient(source, host string) time.Duration {
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
	e.transient++
	e.lastAt = now
	if host != "" {
		e.host = host
	}
	if e.transient >= transientThreshold {
		shift := e.transient - transientThreshold
		if shift > 5 {
			shift = 5
		}
		delay := transientBase * time.Duration(1<<uint(shift))
		if delay > transientMax {
			delay = transientMax
		}
		if until := now.Add(delay); until.After(e.until) {
			e.until = until
		}
	}
	_ = r.store.SaveRateLimit(store.RateLimitState{
		SourceKey: e.source, Host: e.host, Consecutive: e.consecutive,
		BackoffUntil: e.until.UTC().Format(time.RFC3339), Last429At: e.lastAt.UTC().Format(time.RFC3339),
		RetryAfterSeconds: int(e.retryAfter.Seconds()), Transient: e.transient,
	})
	if e.until.After(now) {
		return e.until.Sub(now)
	}
	return 0
}

// ClearTransient resets the transient-failure streak for a source after a
// successful transfer.
func (r *rateLimiter) ClearTransient(source string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e := r.states[source]
	if e == nil {
		return
	}
	e.transient = 0
	if e.consecutive == 0 && !e.until.After(time.Now()) {
		r.deleteLocked(source)
		return
	}
	_ = r.store.SaveRateLimit(store.RateLimitState{
		SourceKey: e.source, Host: e.host, Consecutive: e.consecutive,
		BackoffUntil: e.until.UTC().Format(time.RFC3339), Last429At: e.lastAt.UTC().Format(time.RFC3339),
		RetryAfterSeconds: int(e.retryAfter.Seconds()), Transient: 0,
	})
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
