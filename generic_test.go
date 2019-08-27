package cache

import (
	"os"
	"testing"
	"time"

	"github.com/Jim-Lambert-Bose/cache/persistence"
	"github.com/sirupsen/logrus"
)

// These tests require redis server running on localhost:6379 (the default)
const redisTestServer = "redis://localhost:6379"
const sentinelTestServer = "redis://localhost:26379"
const redisMasterIdentifier = "mymaster"
const sharedSecret = "test-secret-must"
const useSentinel = true
const defExpSeconds = 0
const myEnv = "local"
const maxConnectionsAllowed = 5

var newGenericStoreRedis = func(t *testing.T, defaultExpiration time.Duration) persistence.CacheStore {
	logrus.SetLevel(logrus.ErrorLevel)
	logger := logrus.WithFields(logrus.Fields{
		"requestID": "unknown",
		"method":    "generic_test",
		"path":      "none",
	})

	os.Setenv("NO_REDIS_PASSWORD", "true")
	// os.Setenv("REDIS_READ_ONLY_ADDRESS","redis://localhost:6379")
	// os.Setenv("REDIS_SENTINEL_ADDRESS", "redis://localhost:26379")
	selectDatabase := 3
	cacheWritePool, err := InitRedisCache(useSentinel, defExpSeconds, nil, defExpSeconds, defExpSeconds, selectDatabase, logger)
	if err != nil {
		t.Errorf("couldn't connect to redis on %s", redisTestServer)
		t.FailNow()
		panic("")
	}
	logger.Info("cacheWritePool initialized")
	readOnlyPool, err := InitReadOnlyRedisCache("redis://localhost:6379", "", defExpSeconds, 0, 0, maxConnectionsAllowed, selectDatabase, logger)
	if err != nil {
		t.Errorf("couldn't connect to redis on %s", redisTestServer)
		t.FailNow()
		panic("")
	}
	logger.Info("cacheReadPool initialized")
	c := NewCacheWithMultiPools(cacheWritePool, readOnlyPool, L2, sharedSecret, defExpSeconds, []byte("test"), false)
	// c.Logger = logger
	return c
}

func connWithSentinel(withSentinel bool, defaultExpiration time.Duration) persistence.CacheStore {
	logrus.SetLevel(logrus.ErrorLevel)
	logger := logrus.WithFields(logrus.Fields{
		"requestID": "unknown",
		"method":    "generic_test",
		"path":      "none",
	})

	selectDatabase := 3
	connInfo := RedisConnectionInfo{
		MasterIdentifier:              "",
		Password:                      "",
		SentinelURL:                   "",
		RedisURL:                      "",
		UseSentinel:                   withSentinel,
		DefaultExpSeconds:             defExpSeconds,
		ConnectionTimeoutMilliseconds: defExpSeconds,
		ReadWriteTimeoutMilliseconds:  defExpSeconds,
		SelectDatabase:                selectDatabase,
	}
	if withSentinel {
		connInfo.SentinelURL = sentinelTestServer
	}
	if !withSentinel {
		connInfo.RedisURL = redisTestServer
	}
	cacheWritePool, err := connInfo.New(true, logger)
	if err != nil {
		logrus.Fatalf("couldn't connect to redis on %s", redisTestServer)
	}
	logger.Info("cacheWritePool initialized")
	readOnlyPool, err := InitReadOnlyRedisCache("redis://localhost:6379", "", defExpSeconds, 0, 0, maxConnectionsAllowed, selectDatabase, logger)
	if err != nil {
		logrus.Fatalf("couldn't connect to redis on %s", redisTestServer)
	}
	logger.Info("cacheReadPool initialized")
	c := NewCacheWithMultiPools(cacheWritePool, readOnlyPool, L2, sharedSecret, defExpSeconds, []byte("test"), false)
	// c.Logger = logger
	return c
}

var newFromConnInfoWithoutSentinel = func(t *testing.T, defaultExpiration time.Duration) persistence.CacheStore {
	return connWithSentinel(false, defaultExpiration)
}
var newFromConnInfoWithSentinel = func(t *testing.T, defaultExpiration time.Duration) persistence.CacheStore {
	return connWithSentinel(true, defaultExpiration)
}
var newGenericCache = func(t *testing.T, defaultExpiration time.Duration) *GenericCache {
	c := newGenericStoreRedis(t, defaultExpiration)
	return c.(*GenericCache)
}

var newGenericStoreInMemory = func(t *testing.T, defaultExpiration time.Duration) persistence.CacheStore {
	logrus.SetLevel(logrus.ErrorLevel)
	logger := logrus.WithFields(logrus.Fields{
		"requestID": "unknown",
		"method":    "generic_test",
		"path":      "none",
	})
	storeExp := time.Second
	// just in memory cache 5min for JWKs... nothing else can be cached.
	store := persistence.NewInMemoryStore(storeExp)
	logger.Infof("Using cache expiration: %s", storeExp)
	logger.Info("cacheReadPool initialized")
	c := NewCacheWithPool(store, Writable, L2, sharedSecret, defExpSeconds, []byte("test"), false)
	// c.Logger = logger
	return c
}

var newGenericStoreRedisEncrypted = func(t *testing.T, defaultExpiration time.Duration) persistence.CacheStore {
	logrus.SetLevel(logrus.ErrorLevel)
	logger := logrus.WithFields(logrus.Fields{
		"requestID": "unknown",
		"method":    "generic_test",
		"path":      "none",
	})

	os.Setenv("NO_REDIS_PASSWORD", "true")
	os.Setenv("REDIS_SENTINEL_ADDRESS", sentinelTestServer)     // used by InitRedisCache for the sentinel address
	os.Setenv("REDIS_MASTER_IDENTIFIER", redisMasterIdentifier) // used by InitRedisCache for the master identifier

	selectDatabase := 0
	cacheWritePool, err := InitRedisCache(useSentinel, defExpSeconds, nil, defExpSeconds, defExpSeconds, selectDatabase, logger)
	if err != nil {
		t.Errorf("couldn't connect to redis on %s", redisTestServer)
		t.FailNow()
		panic("")
	}
	logger.Info("cacheWritePool initialized")
	readOnlyPool, err := InitReadOnlyRedisCache("redis://localhost:6379", "", defExpSeconds, 0, 0, maxConnectionsAllowed, selectDatabase, logger)
	if err != nil {
		t.Errorf("couldn't connect to redis on %s", redisTestServer)
		t.FailNow()
		panic("")
	}
	logger.Info("cacheReadPool initialized")
	c := NewCacheWithMultiPools(cacheWritePool, readOnlyPool, L2, sharedSecret, defExpSeconds, []byte("test"), true)
	// c.Logger = logger
	return c
}

var newExpiryLRUInMemoryStore = func(t *testing.T, defaultExpiration time.Duration) persistence.CacheStore {
	// logrus.SetLevel(logrus.ErrorLevel)
	// logger := logrus.WithFields(logrus.Fields{
	// 	"requestID": "unknown",
	// 	"method":    "generic_test",
	// 	"path":      "none",
	// })
	storeExp := time.Second
	store, err := NewInMemoryStore(100, storeExp, defCleanupInterval, true, "")
	if err != nil {
		panic("can't create inmemory store: " + err.Error())
	}
	c := NewCacheWithPool(store, Writable, L2, sharedSecret, defExpSeconds, []byte("test"), false)
	// c.Logger = logger
	return c
}

func TestGenericCache_TypicalGetSet(t *testing.T) {
	typicalGetSet(t, newGenericStoreRedis)
	typicalGetSet(t, newGenericStoreInMemory)
	typicalGetSet(t, newGenericStoreRedisEncrypted)
	typicalGetSet(t, newExpiryLRUInMemoryStore)
	// don't need to run these for every test... just need to make sure the init works
	typicalGetSet(t, newFromConnInfoWithSentinel)
	// typicalGetSet(t, newFromConnInfoWithoutSentinel) // you can not run this because it writes and we're using sentinel... so you likely won't get master
}

func TestRedisCache_IncrDecr(t *testing.T) {
	incrDecr(t, newGenericStoreRedis)
	incrDecr(t, newGenericStoreInMemory)
	incrDecr(t, newGenericStoreRedisEncrypted)
	incrDecr(t, newExpiryLRUInMemoryStore)
}
func TestRedisCache_IncrAtomic(t *testing.T) {
	incrRedisAtomicAndExpireAt(t, newGenericCache)
}
func TestRedisCache_Expiration(t *testing.T) {
	expiration(t, newGenericStoreRedis)
}
func TestRedisCache_ExpirationEncrypted(t *testing.T) {
	expiration(t, newGenericStoreRedisEncrypted)
}
func Test_ExpirationInMemStore(t *testing.T) {
	expiration(t, newGenericStoreInMemory)
}

func Test_ExpirationLRU(t *testing.T) {
	expiration(t, newExpiryLRUInMemoryStore)
}
func TestRedisCache_EmptyCache(t *testing.T) {
	emptyCache(t, newGenericStoreRedis)
	emptyCache(t, newGenericStoreInMemory)
	emptyCache(t, newGenericStoreRedisEncrypted)
	emptyCache(t, newGenericStoreRedisEncrypted)
}

func TestRedisCache_Replace(t *testing.T) {
	testReplace(t, newGenericStoreRedis)
	testReplace(t, newGenericStoreInMemory)
	testReplace(t, newExpiryLRUInMemoryStore)
}

func TestRedisCache_Add(t *testing.T) {
	testAdd(t, newGenericStoreRedis)
	testAdd(t, newGenericStoreInMemory)
}

func TestRedis_RedisGetExpiresIn(t *testing.T) {
	testRedisGetExpiresIn(t, newGenericCache)
}
