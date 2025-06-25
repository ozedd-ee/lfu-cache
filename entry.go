package lfu

import (
	"container/list"
	"time"
)

// entry represents a cache item.
type entry[K comparable, V any] struct {
	key       K
	value     V
	frequency int
	node      *list.Element
	createdAt time.Time
}

// freqList maintains a list of entries for a particular frequency.
type freqList[K comparable, V any] struct {
	items *list.List // list of *entry[K, V]
}

func newFreqList[K comparable, V any]() *freqList[K, V] {
	return &freqList[K, V]{items: list.New()}
}

func (f *freqList[K, V]) pushFront(e *entry[K, V]) {
	e.node = f.items.PushFront(e)
}

func (f *freqList[K, V]) remove(e *entry[K, V]) {
	f.items.Remove(e.node)
}

func (f *freqList[K, V]) removeOldest() *entry[K, V] {
	elem := f.items.Back()
	if elem == nil {
		return nil
	}
	f.items.Remove(elem)
	return elem.Value.(*entry[K, V])
}

func (f *freqList[K, V]) isEmpty() bool {
	return f.items.Len() == 0
}
