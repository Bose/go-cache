package cache

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/Jim-Lambert-Bose/cache/persistence"
)

type cacheFactory func(*testing.T, time.Duration) persistence.CacheStore
type genericCacheFactory func(*testing.T, time.Duration) *GenericCache

// Test typical cache interactions
func typicalGetSet(t *testing.T, newCache cacheFactory) {
	var err error
	c := newCache(t, time.Hour)

	value := "foo"
	if err = c.(persistence.CacheStore).Set("value", value, persistence.DEFAULT); err != nil {
		t.Errorf("Error setting a value: %s", err)
	}

	var tValue string
	err = c.(persistence.CacheStore).Get("value", &tValue)
	value = ""
	err = c.(*GenericCache).Get("value", &value)
	if err != nil {
		t.Errorf("Error getting a value: %s", err)
	}
	if value != "foo" {
		t.Errorf("Expected to get foo back, got %s", value)
	}

	value = "encrypted-data"
	exp := 3 * time.Minute
	entry := c.(*GenericCache).NewGenericCacheEntry(value, exp)
	if err = c.(persistence.CacheStore).Set("value", entry, exp); err != nil {
		t.Errorf("Error setting a value: %s", err)
	}

	getEntry := GenericCacheEntry{}
	err = c.(persistence.CacheStore).Get("value", &getEntry)
	if err != nil {
		t.Errorf("Error getting a value: %s", err)
	}
	if getEntry.Data != "encrypted-data" {
		t.Errorf("Expected to get encrypted-data back, got %v", getEntry)
	}
	// f, e, err := cache.(*GenericCache).Exists("value")
	// fmt.Println(f, e, err)
}

// Test the increment-decrement cases
func incrDecr(t *testing.T, newCache cacheFactory) {
	var err error
	cache := newCache(t, time.Hour)

	// Normal increment / decrement operation.
	if err = cache.Set("int", 10, persistence.DEFAULT); err != nil {
		t.Errorf("Error setting int: %s", err)
	}
	newValue, err := cache.Increment("int", 50)
	if err != nil {
		t.Errorf("Error incrementing int: %s", err)
	}
	if newValue != 60 {
		t.Errorf("Expected 60, was %d", newValue)
	}

	if newValue, err = cache.Decrement("int", 50); err != nil {
		t.Errorf("Error decrementing: %s", err)
	}
	if newValue != 10 {
		t.Errorf("Expected 10, was %d", newValue)
	}

	// Increment wraparound
	newValue, err = cache.Increment("int", math.MaxUint64-5)
	if err != nil {
		t.Errorf("Error wrapping around: %s", err)
	}
	if newValue != 4 {
		t.Errorf("Expected wraparound 4, got %d", newValue)
	}

	// Decrement capped at 0
	newValue, err = cache.Decrement("int", 25)
	if err != nil {
		t.Errorf("Error decrementing below 0: %s", err)
	}
	if newValue != 0 {
		t.Errorf("Expected capped at 0, got %d", newValue)
	}
}

// Test the increment-decrement cases
func incrRedisAtomicAndExpireAt(t *testing.T, newStore genericCacheFactory) {
	var err error
	store := newStore(t, time.Hour)

	// Normal increment / decrement operation.
	if err = store.Set("int", 10, persistence.DEFAULT); err != nil {
		t.Errorf("Error setting int: %s", err)
	}

	newValue, err := store.Cache.(*persistence.RedisStore).IncrementAtomic("int", 50)
	if err != nil {
		t.Errorf("Error incrementing int: %s", err)
	}
	err = store.Cache.(*persistence.RedisStore).ExpireAt("int", uint64(time.Now().Unix()+30))
	if err != nil {
		t.Errorf("Error setting expire at for 'int': %s", err)
	}
	if newValue != 60 {
		t.Errorf("Expected 60, was %d", newValue)
	}
	newValue, err = store.Cache.(*persistence.RedisStore).IncrementCheckSet("int", 50)
	if err != nil {
		t.Errorf("Error incrementing int: %s", err)
	}
	if newValue != 110 {
		t.Errorf("Expected 110, was %d", newValue)
	}
	newValue, err = store.Cache.(*persistence.RedisStore).IncrementCheckSet("badkey", 50)
	if err != persistence.ErrCacheMiss {
		t.Errorf("Error incrementing badkey.. should have been ErrCacheMiss")
	}
	newValue, err = store.Cache.(*persistence.RedisStore).IncrementAtomic("newInt", 2)
	if err != nil {
		t.Errorf("Error incrementing int: %s", err)
	}
	if newValue != 2 {
		t.Errorf("Expected 2, was %d", newValue)
	}
	err = store.Cache.(*persistence.RedisStore).ExpireAt("newInt", uint64(time.Now().Unix()+1))
	if err != nil {
		t.Errorf("Error setting expire at for 'int': %s", err)
	}
	time.Sleep(2 * time.Second)
	err = store.Get("newInt", &newValue)
	if err != persistence.ErrCacheMiss {
		t.Errorf("Expected cache miss")
	}

}

func expiration(t *testing.T, newCache cacheFactory) {
	// memcached does not support expiration times less than 1 second.
	var err error
	cache := newCache(t, time.Second)
	// Test Set w/ DEFAULT
	value := 777
	cache.Set("int", value, time.Second*1)
	time.Sleep(2 * time.Second)
	v2 := 0
	err = cache.Get("int", &v2)
	if err != persistence.ErrCacheMiss {
		t.Errorf("Expected CacheMiss, but got: %s - %v", err, v2)
	}

	// Test Set w/ short time
	cache.Set("int", value, time.Second)
	time.Sleep(2 * time.Second)
	err = cache.Get("int", &v2)
	if err != persistence.ErrCacheMiss {
		t.Errorf("Expected CacheMiss, but got: %s", err)
	}

	// // set new default
	// cache = newCache(t, time.Hour)
	// cache.Set("int", value, 0) // test longer default
	// time.Sleep(2 * time.Second)
	// err = cache.Get("int", &v2)
	// if err != nil {
	// 	t.Errorf("Expected to get the value, but got: %s", err)
	// }

	// Test Set w/ longer time.
	cache.Set("int", value, time.Hour)
	time.Sleep(2 * time.Second)
	err = cache.Get("int", &v2)
	if err != nil {
		t.Errorf("Expected to get the value, but got: %s", err)
	}

	// Test Set w/ forever.
	cache.Set("int", value, persistence.FOREVER)
	time.Sleep(2 * time.Second)
	err = cache.Get("int", &v2)
	if err != nil {
		t.Errorf("Expected to get the value, but got: %s", err)
	}
}

func emptyCache(t *testing.T, newCache cacheFactory) {
	var err error
	cache := newCache(t, time.Hour)

	err = cache.Get("notexist", 0)
	if err == nil {
		t.Errorf("Error expected for non-existent key")
	}
	if err != persistence.ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss for non-existent key: %s", err)
	}

	err = cache.Delete("notexist")
	if err != persistence.ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss for non-existent key: %s", err)
	}

	_, err = cache.Increment("notexist", 1)
	if err != persistence.ErrCacheMiss {
		t.Errorf("Expected cache miss incrementing non-existent key: %s", err)
	}

	_, err = cache.Decrement("notexist", 1)
	if err != persistence.ErrCacheMiss {
		t.Errorf("Expected cache miss decrementing non-existent key: %s", err)
	}
}

func testReplace(t *testing.T, newCache cacheFactory) {
	var err error
	cache := newCache(t, time.Hour)

	// Replace in an empty cache.
	if err = cache.Replace("notexist", 1, persistence.FOREVER); err != persistence.ErrNotStored && err != persistence.ErrCacheMiss {
		t.Errorf("Replace in empty cache: expected ErrNotStored or ErrCacheMiss, got: %s", err)
	}

	// Set a value of 1, and replace it with 2
	if err = cache.Set("int", 1, time.Second); err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	if err = cache.Replace("int", 2, time.Second); err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	var i int
	if err = cache.Get("int", &i); err != nil {
		t.Errorf("Unexpected error getting a replaced item: %s", err)
	}
	if i != 2 {
		t.Errorf("Expected 2, got %d", i)
	}

	// Wait for it to expire and replace with 3 (unsuccessfully).
	time.Sleep(5 * time.Second)
	if err = cache.Replace("int", 3, time.Second); err != persistence.ErrNotStored && err != persistence.ErrCacheMiss {
		t.Errorf("Expected ErrNotStored or ErrCacheMiss, got: %s", err)
	}
	if err = cache.Get("int", &i); err != persistence.ErrCacheMiss {
		t.Errorf("Expected cache miss, got: %s", err)
	}
}

func testAdd(t *testing.T, newCache cacheFactory) {
	var err error
	cache := newCache(t, time.Hour)
	// Add to an empty cache.
	if err = cache.Add("int", 1, time.Second); err != nil {
		t.Errorf("Unexpected error adding to empty cache: %s", err)
	}

	// Try to add again. (fail)
	if err = cache.Add("int", 2, time.Second); err != persistence.ErrNotStored {
		t.Errorf("Expected ErrNotStored adding dupe to cache: %s", err)
	}

	// Wait for it to expire, and add again.
	time.Sleep(2 * time.Second)
	if err = cache.Add("int", 3, time.Second); err != nil {
		t.Errorf("Unexpected error adding to cache: %s", err)
	}

	// Get and verify the value.
	var i int
	if err = cache.Get("int", &i); err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if i != 3 {
		t.Errorf("Expected 3, got: %d", i)
	}
}

func testRedisGetExpiresIn(t *testing.T, newCache genericCacheFactory) {
	var err error
	key := fmt.Sprintf("get-expires-in-%s", time.Now())
	cache := newCache(t, time.Minute)
	// Add to an empty cache.
	if err = cache.Add(key, 1, time.Second); err != nil {
		t.Errorf("Unexpected error adding to empty cache: %s", err)
	}
	ttl, err := cache.RedisGetExpiresIn(key)
	if err != nil {
		t.Errorf("unexpected error == %s", err.Error())
	}
	if ttl < 500 || ttl > 1000 {
		t.Errorf("unexpected value for ttl == %d", ttl)
	}
	t.Logf("ttl == %d", ttl)
}
