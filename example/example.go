package main

import (
	"encoding/gob"
	"os"
	"time"

	goCache "github.com/Bose/go-cache"
	ginCache "github.com/Bose/go-cache/galapagos_gin/cache"
	ginprometheus "github.com/zsais/go-gin-prometheus"

	"github.com/BoseCorp/cache/persistence"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// These tests require redis server running on localhost:6379 (the default)
const redisTestServer = "redis://localhost:6379"
const sentinelTestServer = "redis://localhost:26379" // will default to this if not set
const redisMasterIdentifier = "mymaster"             // will default to this if not set
const sharedSecret = "secret-must-be-min-16chars"
const useSentinel = true // if false, the master will default to redis:6379 if running in k8s, other it defaults to localhost:6379
const defExpSeconds = 30
const defTimeoutMilliseconds = 500
const maxConnectionsAllowed = 50 // max connections allowed to the read-only redis cluster from this service
const maxEntries = 1000          // max entries allowed in the LRU + expiry in memory store

var genericRedisCache interface{}
var genericInMemoryCache interface{}
var lruExpiryCache interface{}

func init() {
	logger := logrus.WithFields(logrus.Fields{
		"requestID": "unknown",
		"method":    "generic_test",
		"path":      "none",
	})
	selectDatabase := 2 // select database #2 in the redis - use 0 if you're connected to a sharded cluster
	if len(os.Getenv("REDIS_SENTINEL_ADDRESS")) == 0 {
		os.Setenv("REDIS_SENTINEL_ADDRESS", sentinelTestServer) // used by InitRedisCache for the sentinel address
	}
	os.Setenv("REDIS_MASTER_IDENTIFIER", redisMasterIdentifier) // used by InitRedisCache for the master identifier
	cacheWritePool, err := goCache.InitRedisCache(useSentinel, defExpSeconds, nil, defTimeoutMilliseconds, defTimeoutMilliseconds, selectDatabase, logger)
	if err != nil {
		logger.Fatalf("couldn't connect to redis on %s", os.Getenv("REDIS_SENTINEL_ADDRESS"))
	}
	logger.Info("cacheWritePool initialized")
	redisAddr := os.Getenv("REDIS_ADDRESS")
	if len(redisAddr) == 0 {
		redisAddr = redisTestServer
	}
	readOnlyPool, err := goCache.InitReadOnlyRedisCache(redisTestServer, "", 0, 0, defExpSeconds, maxConnectionsAllowed, selectDatabase, logger)
	if err != nil {
		logger.Fatalf("couldn't connect to redis on %s", redisTestServer)
	}
	logger.Info("cacheReadPool initialized")
	genericRedisCache = goCache.NewCacheWithMultiPools(cacheWritePool, readOnlyPool, goCache.L2, sharedSecret, defExpSeconds, []byte("test"), true)
	genericRedisCache.(*goCache.GenericCache).Logger = logger

	gob.Register(TestEntry{})

	inMemoryStore := persistence.NewInMemoryStore(defExpSeconds)
	genericInMemoryCache = goCache.NewCacheWithPool(inMemoryStore, goCache.Writable, goCache.L1, sharedSecret, defExpSeconds, []byte("mem"), false)

	inMemoryLRUWExpiryStore, err := goCache.NewInMemoryStore(maxEntries, defExpSeconds, 1*time.Minute, true, "exampe_inmemory_store")
	if err != nil {
		panic("Unable to allocate LRU w/expiry store")
	}
	lruExpiryCache = goCache.NewCacheWithPool(inMemoryLRUWExpiryStore, goCache.Writable, goCache.L1, sharedSecret, defExpSeconds, []byte("mem"), false)
}
func main() {
	// use the JSON formatter
	// logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.DebugLevel)

	r := gin.Default()
	r.Use(gin.Recovery()) // add Recovery middleware
	p := ginprometheus.NewPrometheus("go_cache_example")
	p.Use(r)

	r.GET("/hello", func(c *gin.Context) {
		c.String(200, "Hello world!\n")
	})

	r.GET("/cached-page", ginCache.CachePage(genericRedisCache.(persistence.CacheStore), 5*time.Minute, func(c *gin.Context) {
		c.Writer.Header().Set("x-okay-to-cache", "true") // signal middleware that it's okay to cache this page
		c.String(200, "Cached Hello world!\n")
	}))

	r.GET("/cached-encrypted-entry", func(c *gin.Context) {
		cache := genericRedisCache.(*goCache.GenericCache)
		key := cache.GetKey([]byte("cached-encrypted-entry")) // Data will be decrypted automatically when Getting
		entry := goCache.GenericCacheEntry{}
		err := cache.Get(key, &entry) // get a symmetrical signature to use an entry key
		if err != nil {
			logrus.Errorf("Error getting a value: %s", err)
			exp := 3 * time.Minute // override the default expiry for this entry
			entry = cache.NewGenericCacheEntry(getNewEntry(), exp)

			// why make the client wait... just do this set concurrently
			go func() {
				if err := cache.Set(key, entry, exp); err != nil { // Data will be encrypted automatically when Setting
					logrus.Errorf("Error setting a value: %s", err)
				}
			}()
		}
		c.JSON(200, gin.H{"entry": entry})
		return
	})

	r.GET("/cached-in-memory-not-encrypted-entry", func(c *gin.Context) {
		cache := genericInMemoryCache.(*goCache.GenericCache)
		key := cache.GetKey([]byte("cached-in-memory-not-encrypted-entry")) // Data will be decrypted automatically when Getting
		entry := goCache.GenericCacheEntry{}
		err := cache.Get(key, &entry) // get a symmetrical signature to use an entry key
		if err != nil {
			logrus.Errorf("Error getting a value: %s", err)
			exp := 3 * time.Minute // override the default expiry for this entry
			entry = cache.NewGenericCacheEntry(getNewEntry(), exp)

			// why make the client wait... just do this set concurrently
			go func() {
				if err := cache.Set(key, entry, exp); err != nil { // Data will be encrypted automatically when Setting
					logrus.Errorf("Error setting a value: %s", err)
				}
			}()
		}
		c.JSON(200, gin.H{"entry": entry})
		return
	})

	r.GET("/cached-in-memory-lru-with-expiry", func(c *gin.Context) {
		cache := lruExpiryCache.(*goCache.GenericCache)
		key := cache.GetKey([]byte("cached-in-memory-lru-with-expiry")) // Data will be decrypted automatically when Getting
		entry := goCache.GenericCacheEntry{}
		err := cache.Get(key, &entry) // get a symmetrical signature to use an entry key
		if err != nil {
			logrus.Errorf("Error getting a value: %s", err)
			exp := 3 * time.Minute // override the default expiry for this entry
			entry = cache.NewGenericCacheEntry(getNewEntry(), exp)

			// why make the client wait... just do this set concurrently
			go func() {
				if err := cache.Set(key, entry, exp); err != nil { // Data will be encrypted automatically when Setting
					logrus.Errorf("Error setting a value: %s", err)
				}
			}()
		}
		c.JSON(200, gin.H{"entry": entry})
		return
	})

	r.Run(":9090")
}

// TestEntry - just a sampe of what you can store in a GenericCacheEntry
type TestEntry struct {
	TestValue string
	TestBool  bool
}

func getNewEntry() TestEntry {
	return TestEntry{TestValue: "Hi Mom!", TestBool: true}
}
