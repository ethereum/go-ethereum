// Copyright 2022 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package lru implements generically-typed LRU caches.
package lru

// BasicLRU is a simple LRU cache.
//
// This type is not safe for concurrent use.
// The zero value is not valid, instances must be created using NewCache.
type BasicLRU[K comparable, V any] struct {
	list  dlist[K]
	items map[K]cacheItem[K, V]
	cap   int
}

type cacheItem[K any, V any] struct {
	value V
	node  *dlistNode[K]
}

// NewBasicLRU creates a new LRU cache.
func NewBasicLRU[K comparable, V any](capacity int) BasicLRU[K, V] {
	if capacity < 0 {
		capacity = 1
	}
	return BasicLRU[K, V]{
		items: make(map[K]cacheItem[K, V]),
		cap:   capacity,
	}
}

// Add adds a value to the cache. Returns true if an item was evicted to store the new item.
func (c *BasicLRU[K, V]) Add(key K, value V) (evicted bool) {
	item, ok := c.items[key]
	if ok {
		// Already exists in cache.
		item.value = value
		c.list.moveToFront(item.node)
		return false
	}

	if c.Len() >= c.cap {
		// Evict an item.
		node := c.list.removeLast()
		delete(c.items, node.v)
		evicted = true
	}

	// Store the new item.
	item = cacheItem[K, V]{value: value, node: c.list.push(key)}
	c.items[key] = item
	return evicted
}

// Contains reports whether the given key exists in the cache.
func (c *BasicLRU[K, V]) Contains(key K) bool {
	_, ok := c.items[key]
	return ok
}

// Get retrieves a value from the cache. This marks the key as recently used.
func (c *BasicLRU[K, V]) Get(key K) (value V, ok bool) {
	item, ok := c.items[key]
	if !ok {
		return value, false
	}
	c.list.moveToFront(item.node)
	return item.value, true
}

// GetOldest retrieves the least-recently-used item.
// Note that this does not update the item's recency.
func (c *BasicLRU[K, V]) GetOldest() (key K, value V, ok bool) {
	if c.list.tail == nil {
		return key, value, false
	}
	key = c.list.tail.v
	item := c.items[key]
	return key, item.value, true
}

// Len returns the current number of items in the cache.
func (c *BasicLRU[K, V]) Len() int {
	return len(c.items)
}

// Peek retrieves a value from the cache, but does not mark the key as recently used.
func (c *BasicLRU[K, V]) Peek(key K) (value V, ok bool) {
	item, ok := c.items[key]
	if !ok {
		return value, false
	}
	return item.value, true
}

// Purge empties the cache.
func (c *BasicLRU[K, V]) Purge() {
	c.list.init()
	for k := range c.items {
		delete(c.items, k)
	}
}

// Remove drops an item from the cache. Returns true if the key was present in cache.
func (c *BasicLRU[K, V]) Remove(key K) bool {
	item, ok := c.items[key]
	if ok {
		delete(c.items, key)
		c.list.remove(item.node)
	}
	return ok
}

// RemoveOldest drops the least recently used item.
func (c *BasicLRU[K, V]) RemoveOldest() (key K, value V, ok bool) {
	if c.list.tail == nil {
		return key, value, false
	}
	key = c.list.tail.v
	item := c.items[key]
	delete(c.items, key)
	c.list.remove(c.list.tail)
	return key, item.value, true
}

// Keys returns all keys in the cache.
func (c *BasicLRU[K, V]) Keys() []K {
	keys := make([]K, 0, len(c.items))
	for node := c.list.head; node != nil; node = node.next {
		keys = append(keys, node.v)
	}
	return keys
}

// dlist is a doubly-linked list holding items of type T.
type dlist[T any] struct {
	head *dlistNode[T]
	tail *dlistNode[T]
}

type dlistNode[T any] struct {
	v    T
	next *dlistNode[T]
	prev *dlistNode[T]
}

// init reinitializes the list, making it empty.
func (l *dlist[T]) init() {
	l.head, l.tail = nil, nil
}

// push adds a new item to the front of the list and returns the
func (l *dlist[T]) push(item T) *dlistNode[T] {
	node := &dlistNode[T]{v: item}
	l.pushNode(node)
	return node
}

func (l *dlist[T]) pushNode(node *dlistNode[T]) {
	if l.head == nil {
		// List is empty, new node is head and tail.
		l.head = node
		l.tail = node
	} else {
		node.next = l.head
		l.head.prev = node
		l.head = node
	}
}

// moveToFront makes 'node' the head of the list.
func (l *dlist[T]) moveToFront(node *dlistNode[T]) {
	l.remove(node)
	l.pushNode(node)
}

// remove removes an element from the list.
func (l *dlist[T]) remove(node *dlistNode[T]) {
	if node.next != nil {
		node.next.prev = node.prev
	}
	if node.prev != nil {
		node.prev.next = node.next
	}
	if l.head == node {
		l.head = node.next
	}
	if l.tail == node {
		l.tail = node.prev
	}
	node.next, node.prev = nil, nil
}

// removeLast removes the last element of the list.
func (l *dlist[T]) removeLast() *dlistNode[T] {
	last := l.tail
	if last != nil {
		l.remove(last)
	}
	return last
}
