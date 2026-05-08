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
		bloomFilter: bloom.NewWithEstimates(10000, 0.01), 
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
	if !c.bloomFilter.Test([]byte(key)) {
		return nil, false
	}

	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}

	if time.Now().After(item.expiresAt) {
		// Probabilistic early expiration to avoid stampede
		if rand.Float64() < 0.1 {
			return nil, false
		}
		if time.Now().After(item.expiresAt.Add(item.ttl / 2)) {
			return nil, false
		}
	}

	return item.data, true
}

func (c *QueryCache) Fetch(key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	if val, ok := c.Get(key); ok {
		return val, nil
	}

	// Use singleflight to ensure only one fetch happens for the same key
	val, err, _ := c.flightGroup.Do(key, func() (interface{}, error) {
		// Check again in case another goroutine just filled the cache
		if val, ok := c.Get(key); ok {
			return val, nil
		}

		data, err := fn()
		if err != nil {
			return nil, err
		}

		c.Set(key, data, ttl)
		return data, nil
	})

	return val, err
}
