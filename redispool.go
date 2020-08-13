package cache

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	sentinel "github.com/FZambia/sentinel"
	"github.com/Bose/cache/persistence"
	"github.com/ericchiang/k8s"
	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
)

// Type - define the type of cache: read or write
type Type int

const (
	// ReadOnly ...
	ReadOnly Type = iota
	// Writable ...
	Writable
)

// Level - define a level for the cache: L1, L2, etc
type Level int

const (
	// L1 ...
	L1 Level = iota + 1
	// L2 ...
	L2
)

const (
	network                          = "tcp"
	defConnectionTimeoutMilliseconds = 500
	defReadWriteTimeoutMilliseconds  = 50
)

// NewSentinelPool - create a new pool for Redis Sentinel
func NewSentinelPool(
	sentinelAddrs []string,
	masterIdentifier []byte,
	redisPassword []byte,
	connectionTimeoutMilliseconds int,
	readWriteTimeoutMilliseconds int,
	selectDatabase int,
	logger *logrus.Entry) *redis.Pool {
	if masterIdentifier == nil {
		masterIdentifier = []byte("mymaster")
	}
	connTimeout := time.Duration(connectionTimeoutMilliseconds) * time.Millisecond
	readWriteTimeout := time.Duration(readWriteTimeoutMilliseconds) * time.Millisecond
	sntnl := &sentinel.Sentinel{
		Addrs:      sentinelAddrs,
		MasterName: string(masterIdentifier),
		Dial: func(addr string) (redis.Conn, error) {
			c, err := redis.DialTimeout(network, addr, connTimeout, readWriteTimeout, readWriteTimeout)
			if err != nil {
				return nil, err
			}
			return c, nil
		},
	}

	return &redis.Pool{
		MaxIdle:     3,
		MaxActive:   64,
		Wait:        true,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			redisHostAddr, errHostAddr := sntnl.MasterAddr()
			if errHostAddr != nil {
				return nil, errHostAddr
			}
			// logger.Debugf("NewSentinelPool: using master addr %s", redisHostAddr)
			// logger.Debugf("NewSentinelPool: using addr %s", redisHostAddr)
			c, err := redis.DialTimeout(network, redisHostAddr, connTimeout, readWriteTimeout, readWriteTimeout)
			if err != nil {
				return nil, err
			}
			if redisPassword != nil { // auth first, before doing anything else
				// logger.Debugf("NewSentinelPool: authenticating")
				if _, err := c.Do("AUTH", string(redisPassword)); err != nil {
					c.Close()
					return nil, err
				}
			}
			if selectDatabase != 0 {
				// logger.Debugf("NewSentinelPool: select database %d", selectDatabase)
				if _, err := c.Do("SELECT", selectDatabase); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			if !sentinel.TestRole(c, "master") {
				return errors.New("Role check failed")
			}
			return nil
		},
	}
}

// inCluster - are we executing in the K8s cluster
func inCluster(logger *logrus.Entry) bool {
	// globals.InCluster = true
	// return
	namespace := os.Getenv("MY_POD_NAMESPACE")
	logger.Infof("inCluster: env == %s", namespace)
	// defer recoverInCluster() // maybe it won't panic, so we don't need this
	_, err := k8s.NewInClusterClient()
	// standard service account has no privs to list pods!!!
	// so we're done
	if err != nil {
		logger.Infof("inCluster: false")
		return false
	}
	logger.Infof("inCluster: true")
	return true
}

// RedisConnectionInfo - define all the things needed to manage a connection to redis (optionally using sentinel)
type RedisConnectionInfo struct {
	MasterIdentifier              string // Redis Master identifieer - should come out of ENV from DBaaS and  can be 0 len string (defaults to mymaster)
	Password                      string // Redis password - should be defined via an ENV and should default to 0 len string for no AUTH to Redis
	SentinelURL                   string // Sentinel URL - should be defined via an ENV from DBaaS and can be 0 len string when you don't want to use sentinel
	RedisURL                      string // Redis URL to use if we're not using sentinel
	UseSentinel                   bool   // should be set via a config var - turn on/off sentinel use
	DefaultExpSeconds             int    // default expiry for redis entries - should be set via config
	ConnectionTimeoutMilliseconds int    // Redis connection timeout - should be set via config
	ReadWriteTimeoutMilliseconds  int    // Redis read/write timeout - should be set via config
	SelectDatabase                int    // which Redis database to use - should be set via config
}

// New - Make a new cache interface using a RedisConnectionInfo for setup - it doesn't not rely on the ENV at all
func (connInfo *RedisConnectionInfo) New(testWriteRead bool, logger *logrus.Entry) (interface{}, error) {
	if len(connInfo.MasterIdentifier) == 0 {
		logger.Info("RedisConnectionInfo.New: master identifier is not defined, using mymaster")
		connInfo.MasterIdentifier = "mymaster"
	}
	logger.Infof("RedisConnectionInfo.New: masterIdentifier: %s", connInfo.MasterIdentifier)

	var redisPassword []byte
	if len(connInfo.Password) == 0 {
		logger.Info("RedisConnectionInfo.New: redis password is not defined, so no authentication used to validate connection to REDIS")
	} else {
		redisPassword = []byte(connInfo.Password)
		logger.Infof("RedisConnectionInfo.New: redisPassword len: %v", len(connInfo.Password))
	}
	sentinelHost := "localhost:26379" // default to localhost
	if len(connInfo.SentinelURL) != 0 {
		logger.Infof("RedisConnectionInfo.New: sentinel URL %s", connInfo.SentinelURL)

		// NOTE: The call to generate a client initially dials Redis to discover the topology and passes in the address(es)
		// from the sentinel to a dialFunction, we store the URL for the AUTH information
		realURL, err := url.Parse(connInfo.SentinelURL)
		if err != nil {
			err = fmt.Errorf("RedisConnectionInfo.New: Unable to parse sentinel URL %s - %s", connInfo.SentinelURL, err.Error())
			logger.Error(err)
			return nil, err
		}
		if len(realURL.Host) == 0 {
			err = fmt.Errorf("InitRedisReadOnlyPool: Unable to parse sentinel URL %s - url.Host is empty after parsing format should be: redis://<host-name>:6379", connInfo.SentinelURL)
			logger.Error(err)
			logger.Errorf("InitReadOnlyRedisCache: Can't initialize cache")
			return nil, err
		}
		sentinelHost = realURL.Host
	}
	logger.Infof("RedisConnectionInfo.New: using sentinel host %s", sentinelHost)

	var cache interface{}
	storeExp := time.Duration(connInfo.DefaultExpSeconds) * time.Second
	logger.Infof("RedisConnectionInfo.New: using default cache expiration: %s", storeExp)
	if !connInfo.UseSentinel {
		logger.Debugf("RedisConnectionInfo.New: setting up redis without sentinel")
		storeDNS := "redis:6379" // start with a historic default (redis was the service name in the namespace)
		if len(connInfo.RedisURL) != 0 {
			realURL, err := url.Parse(connInfo.RedisURL)
			if err != nil {
				err = fmt.Errorf("RedisConnectionInfo.New: Unable to parse redis URL %s - %s", connInfo.RedisURL, err.Error())
				logger.Error(err)
				logger.Errorf("RedisConnectionInfo.New: Can't initialize cache")
				return nil, err
			}
			if len(realURL.Host) == 0 {
				err = fmt.Errorf("InitRedisReadOnlyPool: Unable to parse redis URL %s - url.Host is empty after parsing format should be: redis://<host-name>:6379", connInfo.RedisURL)
				logger.Error(err)
				logger.Errorf("InitReadOnlyRedisCache: Can't initialize cache")
				return nil, err
			}
			storeDNS = realURL.Host
		}

		// if we're not in the cluster, then set a localhost override... not sure if this is right long term, but it works
		if !inCluster(logger) {
			storeDNS = "localhost:6379"
		}
		logger.Infof("RedisConnectionInfo.New: using cache at: %s", storeDNS)
		cache = persistence.NewRedisCache(storeDNS, connInfo.Password, storeExp) // if password is 0 len, then no AUTH is used for redis
		if testWriteRead {
			var v string
			var err error
			if v, err = testCache(cache, "testing", "1,2,3..", storeExp); err != nil {
				logger.Errorf("RedisConnectionInfo.New: cache test failed: %s", err.Error())
				return nil, err
			}
			logger.Infof("RedisConnectionInfo.New: cache test success: %s", v)
		}
		return cache, nil
	}
	addrs := []string{sentinelHost}
	sntlPool := NewSentinelPool(
		addrs,
		[]byte(connInfo.MasterIdentifier),
		redisPassword,
		connInfo.ConnectionTimeoutMilliseconds,
		connInfo.ReadWriteTimeoutMilliseconds,
		connInfo.SelectDatabase,
		logger)
	cache = persistence.NewRedisCacheWithPool(sntlPool, storeExp)
	if testWriteRead {
		if err := cache.(persistence.CacheStore).Set("test", "this", storeExp); err != nil {
			logger.Errorf("RedisConnectionInfo.New: cache test failed: %s", err.Error())
			return nil, err
		}
		var v string
		var err error
		if v, err = testCache(cache, "testing", "1,2,3..", storeExp); err != nil {
			logger.Errorf("RedisConnectionInfo.New: cache test failed: %s", err.Error())
			return cache, err
		}
		logger.Infof("RedisConnectionInfo.New: cache test success: %s", v)
	}
	return cache, nil
}

// InitRedisCache - used by microservices to init their redis cache.  Returns an interface{} so
// other caches could be swapped in - this uses ENV vars to figure out what to connect to.
// REDIS_MASTER_IDENTIFIER, REDIS_PASSWORD, REDIS_SENTINEL_ADDRESS
func InitRedisCache(
	useSentinel bool,
	defaultExpSeconds int,
	redisPassword []byte,
	connectionTimeoutMilliseconds int,
	readWriteTimeoutMilliseconds int,
	selectDatabase int,
	logger *logrus.Entry) (interface{}, error) {
	masterIdentifier := os.Getenv("REDIS_MASTER_IDENTIFIER")
	if len(masterIdentifier) == 0 {
		logger.Info("Env REDIS_MASTER_IDENTIFIER is not defined, using mymaster")
		masterIdentifier = "mymaster"
	}
	logger.Infof("masterIdentifier: %s", masterIdentifier)

	if redisPassword == nil {
		p := os.Getenv("REDIS_PASSWORD")
		if len(p) == 0 {
			logger.Info("Env REDIS_PASSWORD is not defined")
			logger.Info("No authentication used to validate connection to REDIS")
		} else {
			redisPassword = []byte(p)
			logger.Infof("Env REDIS_PASSWORD is set")
		}
	}
	logger.Infof("redisPassword len: %v", len(redisPassword))
	redisHost, ok := os.LookupEnv("REDIS_SENTINEL_ADDRESS")
	if ok {
		logger.Infof("Env REDIS_SENTINEL_ADDRESS: %s", redisHost)

		// NOTE: The call to generate a client initially dials Redis to discover the topology and passes in the address(es)
		// from the sentinel to a dialFunction, we store the URL for the AUTH information
		realURL, err := url.Parse(redisHost)
		if err != nil {
			err = fmt.Errorf("initCache: Unable to parse URL REDIS_SENTINEL_ADDRESS - %s", err.Error())
			logger.Error(err)
			return nil, err
		}
		redisHost = realURL.Host
	} else {
		logger.Info("InitCache: environment variable REDIS_SENTINEL_ADDRESS is not defined")
		redisHost = "localhost:26379"
	}
	logger.Infof("InitCache: redisHost: %s", redisHost)
	var cache interface{}
	storeExp := time.Duration(defaultExpSeconds) * time.Second
	logger.Infof("InitCache: using cache expiration: %s", storeExp)
	if !useSentinel {
		logger.Debugf("InitCache: setting up redis without sentinel")
		storeDNS := "redis:6379"
		if !inCluster(logger) {
			storeDNS = "localhost:6379"
		}
		logger.Infof("InitCache: using cache at: %s", storeDNS)
		cache = persistence.NewRedisCache(storeDNS, string(redisPassword), storeExp, persistence.WithSelectDatabase(selectDatabase))
		var v string
		var err error
		if v, err = testCache(cache, "testing", "1,2,3..", storeExp); err != nil {
			logger.Errorf("initCache: cache test failed: %s", err.Error())
			return nil, err
		}
		logger.Infof("initCache: cache test success: %s", v)
		return cache, nil
	}
	addrs := []string{redisHost}
	sntlPool := NewSentinelPool(addrs, []byte(masterIdentifier), redisPassword, connectionTimeoutMilliseconds, readWriteTimeoutMilliseconds, selectDatabase, logger)
	cache = persistence.NewRedisCacheWithPool(sntlPool, storeExp)
	if err := cache.(persistence.CacheStore).Set("test", "this", storeExp); err != nil {
		logger.Errorf("initCache: cache test failed: %s", err.Error())
		return nil, err
	}
	var v string
	var err error
	if v, err = testCache(cache, "testing", "1,2,3..", storeExp); err != nil {
		logger.Errorf("initCache: cache test failed: %s", err.Error())
		return cache, err
	}
	logger.Infof("initCache: cache test success: %s", v)

	return cache, nil
}

func testCache(c interface{}, k string, v string, exp time.Duration) (string, error) {
	if err := c.(persistence.CacheStore).Set(k, v, exp); err != nil {
		return "", err
	}
	var tValue string
	if err := c.(persistence.CacheStore).Get(k, &tValue); err != nil {
		return "", err
	}
	return tValue, nil
}

// InitReadOnlyRedisCache - used by microservices to init their read-only redis cache.  Returns an interface{} so
// other caches could be swapped in
func InitReadOnlyRedisCache(
	readOnlyCacheURL string,
	cachePassword string,
	connectionTimeoutMilliseconds int,
	readWriteTimeoutMilliseconds int,
	defaultExpMinutes int,
	maxConnections int,
	selectDatabase int,
	logger *logrus.Entry) (interface{}, error) {
	var cache interface{}
	logger.Debugf("InitReadOnlyRedisCache: trying to init read-only redis cache")
	defExpSeconds := int(60 * defaultExpMinutes)
	realURL, err := url.Parse(readOnlyCacheURL)
	if err != nil {
		err = fmt.Errorf("InitRedisReadOnlyPool: Unable to parse URL read-only cache address %s - %s", readOnlyCacheURL, err.Error())
		logger.Error(err)
		logger.Errorf("InitReadOnlyRedisCache: Can't initialize read-only redis cache")
		return cache, err
	}
	if len(realURL.Host) == 0 {
		err = fmt.Errorf("InitRedisReadOnlyPool: Unable to parse URL read-only cache address %s - url.Host is empty after parsing format should be: redis://<host-name>:6379", readOnlyCacheURL)
		logger.Error(err)
		logger.Errorf("InitReadOnlyRedisCache: Can't initialize read-only redis cache")
		return cache, err
	}

	if connectionTimeoutMilliseconds <= 0 {
		connectionTimeoutMilliseconds = defConnectionTimeoutMilliseconds // set a reasonable default
	}
	if readWriteTimeoutMilliseconds <= 0 {
		readWriteTimeoutMilliseconds = defReadWriteTimeoutMilliseconds // set a reasonable default
	}
	connTimeout := time.Duration(connectionTimeoutMilliseconds) * time.Millisecond
	readWriteTimeout := time.Duration(readWriteTimeoutMilliseconds) * time.Millisecond
	pool := &redis.Pool{
		MaxIdle:     maxConnections,
		MaxActive:   maxConnections,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			logger.Debugf("InitReadOnlyRedisCache: using addr %s", realURL.Host)
			c, err := redis.DialTimeout(network, realURL.Host, connTimeout, readWriteTimeout, readWriteTimeout)
			if err != nil {
				return nil, err
			}
			if len(cachePassword) > 0 { // auth first before doing anything else
				logger.Debugf("InitReadOnlyRedisCache: authenticating")
				if _, err := c.Do("AUTH", cachePassword); err != nil {
					c.Close()
					return nil, err
				}
			} else {
				// check with PING
				if _, err := c.Do("PING"); err != nil {
					c.Close()
					return nil, err
				}
			}
			if selectDatabase != 0 {
				logger.Debugf("InitReadOnlyRedisCache: select database %d", selectDatabase)
				if _, err := c.Do("SELECT", selectDatabase); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		// custom connection test method
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
		// // custom connection test method
		// TestOnBorrow: func(c redis.Conn, t time.Time) error {
		// 	defer timeTrack(time.Now(), "TestOnBorrow.PING", logger)
		// 	if _, err := c.Do("PING"); err != nil {
		// 		return err
		// 	}
		// 	return nil
		// },
	}
	return persistence.NewRedisCacheWithPool(pool, time.Duration(defExpSeconds)*time.Second), nil
}
