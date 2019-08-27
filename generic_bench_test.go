package cache

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Jim-Lambert-Bose/cache/persistence"
	"github.com/sirupsen/logrus"
)

var genericRedisCache *GenericCache
var genericRedisCacheEncrypted *GenericCache

func initPools() {
	if genericRedisCache == nil {
		genericRedisCache = newBenchGenericStoreRedis(time.Hour, false)
	}
	if genericRedisCacheEncrypted == nil {
		genericRedisCacheEncrypted = newBenchGenericStoreRedis(time.Hour, true)
	}
}

func benchmarkTypicalGetSet(i int, b *testing.B) {
	initPools()
	for n := 0; n < b.N; n++ {
		benchTypicalGetSet(b, genericRedisCache)
	}
}

func benchmarkTypicalGetSetEncrypted(i int, b *testing.B) {
	initPools()
	for n := 0; n < b.N; n++ {
		benchTypicalGetSet(b, genericRedisCacheEncrypted)
	}
}
func BenchmarkGetSet1(b *testing.B)  { benchmarkTypicalGetSet(1, b) }
func BenchmarkGetSet2(b *testing.B)  { benchmarkTypicalGetSet(2, b) }
func BenchmarkGetSet3(b *testing.B)  { benchmarkTypicalGetSet(3, b) }
func BenchmarkGetSet10(b *testing.B) { benchmarkTypicalGetSet(10, b) }
func BenchmarkGetSet20(b *testing.B) { benchmarkTypicalGetSet(20, b) }
func BenchmarkGetSet40(b *testing.B) { benchmarkTypicalGetSet(40, b) }

func BenchmarkGetSetEncrypted1(b *testing.B)  { benchmarkTypicalGetSetEncrypted(1, b) }
func BenchmarkGetSetEncrypted2(b *testing.B)  { benchmarkTypicalGetSetEncrypted(2, b) }
func BenchmarkGetSetEncrypted3(b *testing.B)  { benchmarkTypicalGetSetEncrypted(3, b) }
func BenchmarkGetSetEncrypted10(b *testing.B) { benchmarkTypicalGetSetEncrypted(10, b) }
func BenchmarkGetSetEncrypted20(b *testing.B) { benchmarkTypicalGetSetEncrypted(20, b) }
func BenchmarkGetSetEncrypted40(b *testing.B) { benchmarkTypicalGetSetEncrypted(40, b) }

func newBenchGenericStoreRedis(defaultExpiration time.Duration, encryptEntries bool) *GenericCache {
	logrus.SetLevel(logrus.ErrorLevel)
	logger := logrus.WithFields(logrus.Fields{
		"requestID": "unknown",
		"method":    "generic_test",
		"path":      "none",
	})

	logger.Info(os.Setenv("NO_REDIS_PASSWORD", "true"))
	selectDatabase := 3
	cacheWritePool, err := InitRedisCache(useSentinel, defExpSeconds, nil, defExpSeconds, defExpSeconds, selectDatabase, logger)
	if err != nil {
		logger.Errorf("couldn't connect to redis on %s", redisTestServer)
		panic("")
	}
	logger.Info("cacheWritePool initialized")
	readOnlyPool, err := InitReadOnlyRedisCache("redis://localhost:6379", "", 0, 0, defExpSeconds, maxConnectionsAllowed, selectDatabase, logger)
	if err != nil {
		logger.Errorf("couldn't connect to redis on %s", redisTestServer)
		panic("")
	}
	logger.Info("cacheReadPool initialized")
	c := NewCacheWithMultiPools(cacheWritePool, readOnlyPool, L2, sharedSecret, defExpSeconds, []byte("test"), encryptEntries)
	c.Logger = logger
	return c
}

type benchCacheFactory func(*testing.B, time.Duration) persistence.CacheStore

func benchTypicalGetSet(b *testing.B, c *GenericCache) {
	logrus.SetLevel(logrus.ErrorLevel)
	logger := logrus.WithFields(logrus.Fields{
		"requestID": "unknown",
		"method":    "generic_test",
		"path":      "none",
	})

	uniqUUID := fmt.Sprintf("%s", uuid.New())
	var err error
	value := "foo:" + uniqUUID
	key := "value:" + uniqUUID
	if err = c.Set(key, value, persistence.DEFAULT); err != nil {
		b.Errorf("Error setting a value: %s", err)
	}

	var tValue string
	err = c.Get(key, &tValue)
	logger.Debug("err: ", err)
	logger.Debug("found: ", tValue)
	getValue := ""
	err = c.Get(key, &getValue)
	if err != nil {
		b.Errorf("Error getting a value: %s", err)
	}
	if getValue != value {
		b.Errorf("Expected to get foo back, got %s", value)
	}

	value = "generic-entry-data"
	exp := 3 * time.Minute
	entry := c.NewGenericCacheEntry(value, exp)
	if err = c.Set(key, entry, exp); err != nil {
		b.Errorf("Error setting a value: %s", err)
	}

	getEntry := GenericCacheEntry{}
	err = c.Get(key, &getEntry)
	if err != nil {
		b.Errorf("Error getting a value: %s", err)
	}
	logger.Debug("found: ", getEntry)
	if getEntry.Data != value {
		b.Errorf("Expected to get encrypted-data back, got %v", getEntry)
	}
	// f, e, err := cache.(*GenericCache).Exists("value")
	// fmt.Println(f, e, err)
}
