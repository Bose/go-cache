# go-cache
[![](https://godoc.org/github.com/Bose/go-cache?status.svg)](https://github.com/Bose/go-cache/blob/master/godoc.md) 

GenericCache allows you to interface with cache via either one write/read connection (pool) or two separate write and read connections (pools).  Once initialized the GenericCache will just do the right thing when interacting with the pools, based on what you're trying to do.

GenericCache (implements Jim-Lambert-Bose/cache interface) which makes it compatible with persistent caches included in [gin-contrib/cache](https://github.com/gin-contrib/cache).   Also, a GenericCache can also be used to cache results from a Gin application.
 

The current list of persistent caches from gin-contrib includes:
* Redis 
* In memory
* memcached

## Redis Pools
This package also includes factories to create Redis pools.
* InitRedisCache: creates an interface to a Redis master via a Sentinel pool.  
* InitReadOnlyRedisCache: creates an interface to a read-only Redis pool. 

## Encrypting Cache Entries
The GenericCache supports using symmetrical signatures for cache entry keys and symmetrical encryption for storing/retrieving entry data.   Once the cache is initialized, these crypto operations are very transparent, requiring to intervention or knowledge to utilize. 

## In Memory LRU with Expiry
This package includes InMemoryStore which implements an LRU cache that includes time based expiration of entries.  InMemoryStore is built on top of github.com/hashicorp/golang-lru which provides an open source LRU implementation by HashiCorp, and this package adds time based expiration of entries to that implementation. 

This type of InMemoryStore exports a prometheus metric gauge for the total number of entries in the store: `go_cache_inmemory_cache_total_items_cnt`


## Installation

`$ go get github.com/Bose/go-cache`

You'll also want to install gin-contrib/cache If you want to use it with other peristent cache stores (in memory or memcached):

`$ go get github.com/gin-contrib/cache`

## Benchmarks

```bash
$ ./run-benchmarks.sh 
goos: darwin
goarch: amd64
pkg: github.com/Bose/go-cache
BenchmarkGetSet1-8                     	    3000	    400840 ns/op
BenchmarkGetSet2-8                     	    5000	    400080 ns/op
BenchmarkGetSet3-8                     	    3000	    398985 ns/op
BenchmarkGetSet10-8                    	    3000	    402394 ns/op
BenchmarkGetSet20-8                    	    5000	    400553 ns/op
BenchmarkGetSet40-8                    	    5000	    415157 ns/op
BenchmarkGetSetEncrypted1-8            	    3000	    415584 ns/op
BenchmarkGetSetEncrypted2-8            	    3000	    411826 ns/op
BenchmarkGetSetEncrypted3-8            	    3000	    419667 ns/op
BenchmarkGetSetEncrypted10-8           	    3000	    421709 ns/op
BenchmarkGetSetEncrypted20-8           	    3000	    421568 ns/op
BenchmarkGetSetEncrypted40-8           	    3000	    425709 ns/op
BenchmarkGetSetInMemory1-8             	  200000	      8972 ns/op
BenchmarkGetSetInMemory2-8             	  200000	      8783 ns/op
BenchmarkGetSetInMemory3-8             	  200000	      8802 ns/op
BenchmarkGetSetInMemory10-8            	  200000	      8840 ns/op
BenchmarkGetSetInMemory20-8            	  200000	      8733 ns/op
BenchmarkGetSetInMemory40-8            	  200000	      8883 ns/op
BenchmarkGetSetEncryptedInMemory1-8    	  100000	     18686 ns/op
BenchmarkGetSetEncryptedInMemory2-8    	  100000	     18812 ns/op
BenchmarkGetSetEncryptedInMemory3-8    	  100000	     18686 ns/op
BenchmarkGetSetEncryptedInMemory10-8   	  100000	     18666 ns/op
BenchmarkGetSetEncryptedInMemory20-8   	  100000	     18646 ns/op
BenchmarkGetSetEncryptedInMemory40-8   	  100000	     18738 ns/op
PASS
ok  	github.com/Bose/go-cache	41.418s
```

## Usage


```go
package main

import (
	"encoding/gob"
	"os"
	"time"

	goCache "github.com/Bose/go-cache"
	ginCache "github.com/Bose/go-cache/galapagos_gin/cache"
	ginprometheus "github.com/zsais/go-gin-prometheus"

	"github.com/gin-contrib/cache/persistence"
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
	selectDatabase := 2                                         // select database #2 in the redis - use 0 if you're connected to a sharded cluster
	os.Setenv("REDIS_SENTINEL_ADDRESS", sentinelTestServer)     // used by InitRedisCache for the sentinel address
	os.Setenv("REDIS_MASTER_IDENTIFIER", redisMasterIdentifier) // used by InitRedisCache for the master identifier
	cacheWritePool, err := goCache.InitRedisCache(useSentinel, defExpSeconds, nil, defExpSeconds, selectDatabase, logger)
	if err != nil {
		logger.Fatalf("couldn't connect to redis on %s", redisTestServer)
	}
	logger.Info("cacheWritePool initialized")
	readOnlyPool, err := goCache.InitReadOnlyRedisCache(redisTestServer, "", defExpSeconds, maxConnectionsAllowed, selectDatabase, logger)
	if err != nil {
		logger.Fatalf("couldn't connect to redis on %s", redisTestServer)
	}
	logger.Info("cacheReadPool initialized")
	genericRedisCache = goCache.NewCacheWithMultiPools(cacheWritePool, readOnlyPool, goCache.L2, sharedSecret, defExpSeconds, []byte("test"), true)
	genericRedisCache.(*goCache.GenericCache).Logger = logger

	gob.Register(TestEntry{})

	inMemoryStore := persistence.NewInMemoryStore(defExpSeconds)
	genericInMemoryCache = goCache.NewCacheWithPool(inMemoryStore, goCache.Writable, goCache.L1, sharedSecret, defExpSeconds, []byte("mem"), false)

	inMemoryLRUWExpiryStore, err := goCache.NewInMemoryStore(maxEntries, defExpSeconds, 1*time.Minute)
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

```

See also: 
* [example/README.md](https://github.com/Bose/go-cache/blob/master/example/README.md)
* [example.go](https://github.com/Bose/go-cache/blob/master/example/example.go)

