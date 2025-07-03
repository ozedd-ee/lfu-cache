package lfu

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var ErrNotFound = errors.New("key not found")

type EvictionCallback[K comparable, V any] func(key K, value V)

type LFUCache[K comparable, V any] struct {
	capacity        int
	size            int
	ttl             time.Duration
	cleanupInterval time.Duration

	keyMap  map[K]*entry[K, V]
	freqMap map[int]*freqList[K, V]
	minFreq int

	mu      sync.RWMutex
	stop    chan struct{}
	onEvict EvictionCallback[K, V]

	hits      atomic.Int64
	misses    atomic.Int64
	evictions atomic.Int64
}

type CacheStats struct {
	Hits      int64
	Misses    int64
	Evictions int64
}

// Create a new LFU cache with the given capacity.
func New[K comparable, V any](
	capacity int,
	ttl time.Duration,
	cleanupInterval time.Duration,
	onEvict EvictionCallback[K, V],
) *LFUCache[K, V] {
	c := &LFUCache[K, V]{
		capacity:        capacity,
		ttl:             ttl,
		cleanupInterval: cleanupInterval,
		keyMap:          make(map[K]*entry[K, V]),
		freqMap:         make(map[int]*freqList[K, V]),
		stop:            make(chan struct{}), // to gracefully shutdown cleanup routine
		onEvict:         onEvict,
	}
	go c.startCleanupLoop()
	return c
}

func (c *LFUCache[K, V]) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return CacheStats{
		Hits: c.hits.Load(),
		Misses: c.misses.Load(),
		Evictions: c.evictions.Load(),
	}
}

// Retrieve a value and update its frequency.
func (c *LFUCache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	ent, ok := c.keyMap[key]
	c.mu.RUnlock()

	// Remove expired key if spotted to complement the CleanUpLoop
	if !ok || time.Since(ent.createdAt) > c.ttl {
		if ok {
			c.mu.Lock()
			c.deleteKey(key, ent) // Still O(1), so wouldn't hurt performance much
			c.mu.Unlock()
		}
		c.misses.Add(1)
		var zero V
		return zero, false
	}

	c.mu.Lock()
	c.increment(ent)
	c.mu.Unlock()
	c.hits.Add(1)
	return ent.value, true
}

// Insert or update a key-value pair.
func (c *LFUCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.capacity == 0 {
		return
	}

	if ent, ok := c.keyMap[key]; ok {
		ent.value = value
		ent.createdAt = time.Now()
		c.increment(ent)
		return
	}

	if c.size >= c.capacity {
		c.evict()
	}

	ent := &entry[K, V]{
		key:       key,
		value:     value,
		frequency: 1,
		createdAt: time.Now(),
	}
	c.keyMap[key] = ent

	if c.freqMap[1] == nil {
		c.freqMap[1] = newFreqList[K, V]()
	}
	c.freqMap[1].pushFront(ent)
	c.minFreq = 1
	c.size++
}

func (c *LFUCache[K, V]) increment(ent *entry[K, V]) {
	oldFreq := ent.frequency
	ent.frequency++

	// Remove from old freq list
	c.freqMap[oldFreq].remove(ent)
	if c.freqMap[oldFreq].isEmpty() {
		delete(c.freqMap, oldFreq)
		if c.minFreq == oldFreq {
			c.minFreq++
		}
	}

	// Add to new freq list
	if c.freqMap[ent.frequency] == nil {
		c.freqMap[ent.frequency] = newFreqList[K, V]()
	}
	c.freqMap[ent.frequency].pushFront(ent)
}

func (c *LFUCache[K, V]) evict() {
	list := c.freqMap[c.minFreq]
	if list == nil {
		return
	}
	evicted := list.removeOldest()
	if evicted != nil {
		delete(c.keyMap, evicted.key)
		c.size--
		c.evictions.Add(1)
		if list.isEmpty() {
			delete(c.freqMap, c.minFreq)
		}
		if c.onEvict != nil {
			c.onEvict(evicted.key, evicted.value)
		}
	}
}

func (c *LFUCache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.size
}

func (c *LFUCache[K, V]) deleteKey(key K, ent *entry[K, V]) {
	c.freqMap[ent.frequency].remove(ent)
	if c.freqMap[ent.frequency].isEmpty() {
		delete(c.freqMap, ent.frequency)
		if c.minFreq == ent.frequency {
			c.minFreq++
		}
	}
	delete(c.keyMap, key)
	c.size--
	c.evictions.Add(1)
	if c.onEvict != nil {
		c.onEvict(ent.key, ent.value)
	}
}

func (c *LFUCache[K, V]) startCleanupLoop() {
	ticker := time.NewTicker(c.cleanupInterval)
	for {
		select {
		case <-ticker.C:
			c.cleanupExpired()
		case <-c.stop:
			ticker.Stop()
			return
		}
	}
}

func (c *LFUCache[K, V]) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, ent := range c.keyMap {
		if now.Sub(ent.createdAt) > c.ttl {
			c.deleteKey(k, ent)
		}
	}
}

// Stop terminates the cleanup loop goroutine.
func (c *LFUCache[K, V]) Stop() {
	close(c.stop)
}
