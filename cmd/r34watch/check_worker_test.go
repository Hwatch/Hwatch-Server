package main

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"r34watch/internal/store"
)

// TestSubscriptionCheckConcurrencyIsBounded verifies the subscription-check
// worker pool never runs more than maxConcurrentChecks checks at once, and
// that the queue drains and clears its "checking" markers afterwards.
func TestSubscriptionCheckConcurrencyIsBounded(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "check.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a := New(s)

	var mu sync.Mutex
	active, maxSeen := 0, 0
	release := make(chan struct{})
	a.checkOverride = func(string) {
		mu.Lock()
		active++
		if active > maxSeen {
			maxSeen = active
		}
		mu.Unlock()
		<-release
		mu.Lock()
		active--
		mu.Unlock()
	}

	for i := 0; i < 10; i++ {
		a.triggerCheck(fmt.Sprintf("slug-%d", i))
	}
	// Wait until the pool is saturated.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		seen := maxSeen
		mu.Unlock()
		if seen >= maxConcurrentChecks {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	seen := maxSeen
	mu.Unlock()
	if seen != maxConcurrentChecks {
		t.Fatalf("max concurrent checks = %d, want %d", seen, maxConcurrentChecks)
	}

	close(release)
	// Queue must fully drain and clear.
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if a.checkActive.Load() == 0 && len(a.checkQueue) == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if n := a.checkActive.Load(); n != 0 {
		t.Fatalf("checks still active: %d", n)
	}
	a.mu.Lock()
	pending := len(a.checking)
	a.mu.Unlock()
	if pending != 0 {
		t.Fatalf("checking markers not cleared: %d", pending)
	}
}
