package lfu

import (
	"sync"
	"testing"
	"time"
)

// helper: create a short TTL LFU cache with fixed cleanup interval
func newTestCache[K comparable, V any](
	cap int,
	ttl time.Duration,
	evictCb EvictionCallback[K, V],
) *LFUCache[K, V] {
	return New(cap, ttl, 50*time.Millisecond, evictCb)
}

// Test basic Set and Get
func TestSetAndGet(t *testing.T) {
	cache := newTestCache[string, int](2, time.Minute, nil)

	cache.Set("a", 1)
	cache.Set("b", 2)
 
	if v, ok := cache.Get("a"); !ok || v != 1 {
		t.Errorf("Expected a=1, got %v", v)
	}
	if v, ok := cache.Get("b"); !ok || v != 2 {
		t.Errorf("Expected b=2, got %v", v)
	}
}

// Test eviction when capacity is full
func TestEviction(t *testing.T) {
	var evicted []string
	var mu sync.Mutex

	cache := newTestCache( 2, time.Minute, func(k string, v int) {
		mu.Lock()
		evicted = append(evicted, k)
		mu.Unlock()
	})

	cache.Set("a", 1)
	cache.Set("b", 2)
	cache.Set("c", 3) // should evict least-frequent (a or b)

	if cache.Len() != 2 {
		t.Errorf("Expected length 2, got %d", cache.Len())
	}

	mu.Lock()
	if len(evicted) != 1 {
		t.Errorf("Expected 1 eviction, got %d", len(evicted))
	}
	mu.Unlock()
}

// Test that LFU is honored during eviction
func TestLFUEvictionOrder(t *testing.T) {
	cache := newTestCache[string, int](2, time.Minute, nil)

	cache.Set("a", 1)
	cache.Set("b", 2)
	_, _ = cache.Get("a") // increase a's frequency

	cache.Set("c", 3) // should evict b

	if _, ok := cache.Get("b"); ok {
		t.Errorf("Expected b to be evicted")
	}
	if _, ok := cache.Get("a"); !ok {
		t.Errorf("Expected a to remain")
	}
	if _, ok := cache.Get("c"); !ok {
		t.Errorf("Expected c to remain")
	}
}

// Test TTL expiration on Get
func TestExpirationOnGet(t *testing.T) {
	cache := newTestCache[string, int](2, 50*time.Millisecond, nil)

	cache.Set("x", 42)
	time.Sleep(80 * time.Millisecond)

	if _, ok := cache.Get("x"); ok {
		t.Errorf("Expected x to be expired")
	}
}

// Test cleanup loop expires keys
func TestCleanupLoop(t *testing.T) {
	cache := newTestCache[string, int](2, 50*time.Millisecond, nil)

	cache.Set("x", 100)
	time.Sleep(100 * time.Millisecond)

	// Wait for background loop to clean it
	time.Sleep(100 * time.Millisecond)

	if cache.Len() != 0 {
		t.Errorf("Expected item to be cleaned up, got length %d", cache.Len())
	}
}

// Test eviction callback on expiration
func TestEvictionCallback(t *testing.T) {
	var called bool
	cache := newTestCache(1, 50*time.Millisecond, func(k string, v int) {
		called = true
	})

	cache.Set("x", 1)
	time.Sleep(100 * time.Millisecond)
	_, _ = cache.Get("x") // triggers deleteKey()

	if !called {
		t.Errorf("Expected eviction callback to be called")
	}
}

func TestCacheStats(t *testing.T) {
	cache := newTestCache[string, int](2, time.Minute, nil)

	cache.Set("a", 1)
	cache.Set("b", 2)

	_, _ = cache.Get("a")
	_, _ = cache.Get("b")

	_, _ = cache.Get("c")

	cache.Set("c", 3)

	stats := cache.Stats()

	if stats.Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}
	if stats.Evictions != 1 {
		t.Errorf("Expected 1 eviction, got %d", stats.Evictions)
	}
}
