package cache

import (
	"testing"
	"time"

	"github.com/BoseCorp/cache/persistence"
)

const (
	maxEntries         = 100
	defCleanupInterval = 1 * time.Second
)

var newInMemoryStore = func(_ *testing.T, defaultExpiration time.Duration) persistence.CacheStore {
	c, err := NewInMemoryStore(100, defaultExpiration, defCleanupInterval, true, "")
	if err != nil {
		panic("can't create inmemory store: " + err.Error())
	}
	return c
}

func TestInMemoryStore_TypicalGetSet(t *testing.T) {
	inMemTypicalGetSet(t, newInMemoryStore)
}

func TestInMemoryStore_Expiration(t *testing.T) {
	inMemExpiration(t, newInMemoryStore)
}

func TestInMemoryStore_IncrDecr(t *testing.T) {
	incrDecr(t, newInMemoryStore)
}
func TestInMemoryStore_EmptyCache(t *testing.T) {
	emptyCache(t, newInMemoryStore)
}

func TestInMemoryStore_Add(t *testing.T) {
	testAdd(t, newInMemoryStore)
}

func TestInMemoryStore_Len(t *testing.T) {
	var err error
	c := newInMemoryStore(t, time.Hour)

	value := "foo"
	if err = c.(persistence.CacheStore).Set("value", value, persistence.DEFAULT); err != nil {
		t.Errorf("Error setting a value: %s", err)
	}

	if c.(*InMemoryStore).Len() != 1 {
		t.Errorf("Error - there should be 1 key in the store: %v", c.(*InMemoryStore).Len())
	}
}

func TestInMemoryStore_Keys(t *testing.T) {
	var err error
	c := newInMemoryStore(t, time.Hour)

	value := "foo"
	if err = c.(persistence.CacheStore).Set("value", value, persistence.DEFAULT); err != nil {
		t.Errorf("Error setting a value: %s", err)
	}

	for _, v := range c.(*InMemoryStore).Keys() {
		if v != "value" {
			t.Errorf("Error - keys don't match: %v != %v", v, value)
		}
		var foundValue string
		err := c.(persistence.CacheStore).Get(v.(string), &foundValue)
		if err != nil {
			t.Errorf("error getting value: %v", err.Error())
		}
		if foundValue != value {
			t.Errorf("Error - values don't match: %v != %v", foundValue, value)
		}
	}
}

func inMemTypicalGetSet(t *testing.T, newCache cacheFactory) {
	var err error
	c := newCache(t, time.Hour)

	value := "foo"
	if err = c.(persistence.CacheStore).Set("value", value, persistence.DEFAULT); err != nil {
		t.Errorf("Error setting a value: %s", err)
	}

	var tValue string
	err = c.(persistence.CacheStore).Get("value", &tValue)
	value = ""
	err = c.(*InMemoryStore).Get("value", &value)
	if err != nil {
		t.Errorf("Error getting a value: %s", err)
	}
	if value != "foo" {
		t.Errorf("Expected to get foo back, got %s", value)
	}

	value = "encrypted-data"
	exp := 3 * time.Minute
	entry, err := c.(*InMemoryStore).NewGenericCacheEntry(value, exp)
	if err != nil {
		t.Errorf("Error creating entry: %s", err)
	}
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

func inMemExpiration(t *testing.T, newCache cacheFactory) {
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

	// set new default
	cache = newCache(t, time.Hour)
	cache.Set("int", value, 0) // test longer default
	time.Sleep(2 * time.Second)
	err = cache.Get("int", &v2)
	if err != nil {
		t.Errorf("Expected to get the value, but got: %s", err)
	}

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
