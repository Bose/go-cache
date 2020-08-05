package cache

import (
	"fmt"
	"runtime"
	"time"

	"github.com/Bose/go-cache/galapagos_gin/cache"
	"github.com/BoseCorp/cache/persistence"
	lru "github.com/hashicorp/golang-lru"
)

// InMemoryStore - an in memory LRU store with expiry
type InMemoryStore struct {
	*inMemoryStore
}

type inMemoryStore struct {
	lru        *lru.ARCCache
	DefaultExp time.Duration
	janitor    *janitor
}

// NewGenericCacheEntry - create a new in memory cache entry
func (c *InMemoryStore) NewGenericCacheEntry(data interface{}, exp time.Duration) (newEntry GenericCacheEntry, err error) {
	var t time.Duration
	if exp != 0 {
		t = exp
	} else {

		t = c.DefaultExp
	}
	now := time.Now().Unix()
	expiresAt := now + int64(t/time.Second) // convert from nanoseconds

	entry := GenericCacheEntry{Data: data, TimeAdded: now, ExpiresAt: expiresAt}
	return entry, nil
}

// NewInMemoryStore - create a new in memory cache
func NewInMemoryStore(maxEntries int, defaultExpiration, cleanupInterval time.Duration, createMetric bool, metricLabel string) (*InMemoryStore, error) {
	lru, err := lru.NewARC(maxEntries)
	if err != nil {
		return nil, err
	}
	c := &inMemoryStore{
		lru:        lru,
		DefaultExp: defaultExpiration,
	}

	if createMetric {
		label := "inmemory_cache_total_items_cnt"
		if len(metricLabel) != 0 {
			label = metricLabel
		}
		// setup metrics
		initGaugeWithFunc(
			func() float64 {
				return float64(lru.Len())
			},
			label,
			fmt.Sprintf("Total count the number of items in the in-memory cache for %s", label))
	}

	// This trick ensures that the janitor goroutine (which--granted it
	// was enabled--is running DeleteExpired on c forever) does not keep
	// the returned C object from being garbage collected. When it is
	// garbage collected, the finalizer stops the janitor goroutine, after
	// which c can be collected.
	C := &InMemoryStore{c}
	if cleanupInterval > 0 {
		runJanitor(c, cleanupInterval)
		runtime.SetFinalizer(C, stopJanitor)
	}
	return C, nil
}

// Get - Get an entry
func (c *InMemoryStore) Get(key string, value interface{}) error {
	if val, ok := c.lru.Get(key); ok {
		entry := val.(GenericCacheEntry)
		if entry.Expired() {
			c.lru.Remove(key)
			return persistence.ErrCacheMiss
		}
		valueType := fmt.Sprintf("%T", value)
		switch valueType {
		case "*string":
			*value.(*string) = entry.Data.(string)
			return nil
		case "*int":
			*value.(*int) = entry.Data.(int)
			return nil
		case "*cache.GenericCacheEntry":
			*value.(*GenericCacheEntry) = entry
			return nil
		case "*cache.ResponseCache":
			*value.(*cache.ResponseCache) = entry.Data.(cache.ResponseCache)
			return nil
		}
		return persistence.ErrNotSupport
	}
	return persistence.ErrCacheMiss
}
func (c *InMemoryStore) doAddSet(key string, value interface{}, exp time.Duration) error {
	valueType := fmt.Sprintf("%T", value)
	now := time.Now().Unix()
	var expiresAt int64
	if exp == persistence.FOREVER {
		expiresAt = 0
	} else if exp != 0 {
		expiresAt = now + int64(exp/time.Second)
	} else {
		expiresAt = now + int64(c.DefaultExp/time.Second)
	}
	if valueType != "cache.GenericCacheEntry" {
		e := GenericCacheEntry{Data: value, ExpiresAt: expiresAt, TimeAdded: now}
		c.lru.Add(key, e)
		return nil
	}
	e := GenericCacheEntry{Data: value.(GenericCacheEntry).Data, ExpiresAt: expiresAt, TimeAdded: now}
	c.lru.Add(key, e)
	return nil
}

// Keys - get all the keys
func (c *InMemoryStore) Keys() []interface{} {
	return c.lru.Keys()
}

// Len - get the current count of entries in the cache
func (c *InMemoryStore) Len() int {
	return c.lru.Len()
}

// Set - set an entry
func (c *InMemoryStore) Set(key string, value interface{}, exp time.Duration) error {
	return c.doAddSet(key, value, exp)
}

// Add - add an entry
func (c *InMemoryStore) Add(key string, value interface{}, exp time.Duration) error {
	if _, ok := c.lru.Get(key); ok {
		return persistence.ErrNotStored
	}
	return c.doAddSet(key, value, exp)
}

// Replace - replace an entry
func (c *InMemoryStore) Replace(key string, value interface{}, exp time.Duration) error {
	if _, ok := c.lru.Get(key); ok {
		return c.doAddSet(key, value, exp)
	}
	return persistence.ErrNotStored
}

// Update - update an entry
func (c *InMemoryStore) Update(key string, entry GenericCacheEntry) error {
	if _, ok := c.lru.Get(key); ok {
		c.lru.Add(key, entry)
		return nil
	}
	return persistence.ErrNotStored
}

// Delete - delete an entry
func (c *InMemoryStore) Delete(key string) error {
	if _, ok := c.lru.Get(key); ok {
		c.lru.Remove(key)
		return nil
	}
	return persistence.ErrCacheMiss
}

// Increment (see CacheStore interface)
func (c *InMemoryStore) Increment(key string, n uint64) (uint64, error) {
	v, ok := c.lru.Get(key)
	if !ok {
		return 0, persistence.ErrCacheMiss
	}
	entry := v.(GenericCacheEntry)
	valueType := fmt.Sprintf("%T", entry.Data)
	switch valueType {
	case "int":
		entry.Data = entry.Data.(int) + int(n)
		c.lru.Add(key, entry)
		return uint64(entry.Data.(int)), nil
	}
	return 0, persistence.ErrNotSupport
}

// Decrement (see CacheStore interface)
func (c *InMemoryStore) Decrement(key string, n uint64) (uint64, error) {
	v, ok := c.lru.Get(key)
	if !ok {
		return 0, persistence.ErrCacheMiss
	}
	entry := v.(GenericCacheEntry)
	valueType := fmt.Sprintf("%T", entry.Data)
	switch valueType {
	case "int":
		if int(n) > entry.Data.(int) {
			entry.Data = 0
		} else {
			entry.Data = entry.Data.(int) - int(n)
		}
		c.lru.Add(key, entry)
		return uint64(entry.Data.(int)), nil
	}
	return 0, persistence.ErrNotSupport
}

// Flush (see CacheStore interface)
func (c *InMemoryStore) Flush() error {
	c.lru.Purge()
	return nil
}

// DeleteExpired - Delete all expired items from the cache.
func (c *inMemoryStore) DeleteExpired() {
	keys := c.lru.Keys()
	for _, key := range keys {
		if entry, ok := c.lru.Get(key); ok {
			e := entry.(GenericCacheEntry)
			if e.Expired() {
				c.lru.Remove(key)
			}
		}
	}
}

type janitor struct {
	Interval time.Duration
	stop     chan bool
}

func (j *janitor) Run(c *inMemoryStore) {
	j.stop = make(chan bool)
	tick := time.Tick(j.Interval)
	for {
		select {
		case <-tick:
			c.DeleteExpired()
		case <-j.stop:
			return
		}
	}
}

func stopJanitor(c *InMemoryStore) {
	c.janitor.stop <- true
}

func runJanitor(c *inMemoryStore, ci time.Duration) {
	j := &janitor{
		Interval: ci,
	}
	c.janitor = j
	go j.Run(c)
}
