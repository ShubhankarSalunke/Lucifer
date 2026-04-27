package datamodel

import (
	"math/rand"
	"sync"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
	"golang.org/x/sync/singleflight"
)

type cacheItem struct {
	data      interface{}
	expiresAt time.Time
	ttl       time.Duration
}

type QueryCache struct {
	mu          sync.RWMutex
	items       map[string]cacheItem
	bloomFilter *bloom.BloomFilter
	flightGroup singleflight.Group
}

func NewQueryCache() *QueryCache {
	return &QueryCache{
		items:       make(map[string]cacheItem),
		bloomFilter: bloom.NewWithEstimates(10000, 0.01), // 10k items, 1% false positive
	}
}

func (c *QueryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.bloomFilter.Add([]byte(key))

	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = cacheItem{
		data:      value,
		expiresAt: time.Now().Add(ttl),
		ttl:       ttl,
	}
}

func (c *QueryCache) Get(key string) (interface{}, bool) {
	// 1. Check Bloom Filter first (lock-free fast path)
	if !c.bloomFilter.Test([]byte(key)) {
		return nil, false // Definitely not here, skip map lock entirely
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	item, found := c.items[key]
	if !found {
		return nil, false
	}

	now := time.Now()
	if now.After(item.expiresAt) {
		return nil, false
	}

	// Probabilistic early expiration (X-Fetch) to avoid stampedes
	timeRemaining := item.expiresAt.Sub(now)
	if timeRemaining > 0 && timeRemaining < item.ttl/10 {
		if rand.Float32() < 0.10 { // 10% chance to early expire one requestor
			return nil, false
		}
	}

	return item.data, true
}

// Fetch executes a query function with singleflight protection
func (c *QueryCache) Fetch(key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	if val, found := c.Get(key); found {
		return val, nil
	}

	// Singleflight - if multiple requests hit this simultaneously, only one runs `fn()`
	v, err, _ := c.flightGroup.Do(key, func() (interface{}, error) {
		// Double check in case it was populated while we waited
		if val, found := c.Get(key); found {
			return val, nil
		}

		val, computeErr := fn()
		if computeErr != nil {
			return nil, computeErr
		}

		c.Set(key, val, ttl)
		return val, nil
	})

	return v, err
}

// ClearExpired removes all expired items from the cache
func (c *QueryCache) ClearExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, v := range c.items {
		if now.After(v.expiresAt) {
			delete(c.items, k)
		}
	}
}
