package lfu

import (
	"errors"
)

var ErrNotFound = errors.New("key not found")

type LFUCache[K comparable, V any] struct {
	capacity int
	size     int

	keyMap map[K]*entry[K, V]
	freqMap map[int]*freqList[K, V]
	minFreq int
}

// Create a new LFU cache with the given capacity.
func New[K comparable, V any](capacity int) *LFUCache[K, V] {
	return &LFUCache[K, V]{
		capacity: capacity,
		keyMap:   make(map[K]*entry[K, V]),
		freqMap:  make(map[int]*freqList[K, V]),
	}
}

// Retrieve a value and update its frequency.
func (c *LFUCache[K, V]) Get(key K) (V, bool) {
	if ent, ok := c.keyMap[key]; ok {
		c.increment(ent)
		return ent.value, true
	}
	var zero V
	return zero, false
}

// Insert or update a key-value pair.
func (c *LFUCache[K, V]) Set(key K, value V) {
	if c.capacity == 0 {
		return
	}

	if ent, ok := c.keyMap[key]; ok {
		ent.value = value
		c.increment(ent)
		return
	}

	if c.size >= c.capacity {
		c.evict()
	}

	// Insert new entry
	ent := &entry[K, V]{
		key:      key,
		value:    value,
		frequency: 1,
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
		if list.isEmpty() {
			delete(c.freqMap, c.minFreq)
		}
	}
}

func (c *LFUCache[K, V]) Len() int {
	return c.size
}
