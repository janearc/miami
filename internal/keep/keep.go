// Package keep holds a few expensive results by key.
//
// One result is enough while there is one camera. Two cameras taking turns on
// one kept result evict each other every frame, and a thing that was worked
// out once a minute is suddenly worked out twice a frame -- for a black hole's
// rays that is 1.3 seconds a frame rather than 18 milliseconds.
//
// So: a handful of the most recent, and the oldest goes when the handful is
// full. Not a general cache. There is no time in it, no cost model and no
// statistics; it is a small fixed number of slots, and what it is for is that
// the number is more than one.
package keep

import "sync"

// Cache holds up to a fixed number of values, the most recently put.
type Cache[K comparable, V any] struct {
	mu   sync.Mutex
	max  int
	keys []K
	vals []V
}

// New is a cache of at most max values. A max under one is one.
func New[K comparable, V any](max int) *Cache[K, V] {
	return &Cache[K, V]{max: max}
}

// Get is the value under a key, and whether it was there.
func (c *Cache[K, V]) Get(k K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, key := range c.keys {
		if key == k {
			return c.vals[i], true
		}
	}
	var zero V
	return zero, false
}

// Put stores a value, replacing what was under that key, and drops the oldest
// if there is no room.
func (c *Cache[K, V]) Put(k K, v V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, key := range c.keys {
		if key == k {
			c.vals[i] = v
			return
		}
	}
	if c.max < 1 {
		c.max = 1
	}
	if len(c.keys) >= c.max {
		c.keys, c.vals = c.keys[1:], c.vals[1:]
	}
	c.keys, c.vals = append(c.keys, k), append(c.vals, v)
}

// Clear drops everything, for a caller whose inputs changed underneath it.
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	c.keys, c.vals = nil, nil
	c.mu.Unlock()
}

// Len is how many values are held.
func (c *Cache[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.keys)
}
