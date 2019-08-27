

# cache
`import "github.com/Bose/go-cache"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)
* [Subdirectories](#pkg-subdirectories)

## <a name="pkg-overview">Overview</a>



## <a name="pkg-index">Index</a>
* [Variables](#pkg-variables)
* [func InitReadOnlyRedisCache(readOnlyCacheURL string, cachePassword string, defaultExpMinutes int, maxConnections int, selectDatabase int, logger *logrus.Entry) (interface{}, error)](#InitReadOnlyRedisCache)
* [func InitRedisCache(useSentinel bool, defaultExpSeconds int, redisPassword []byte, timeoutMilliseconds int, selectDatabase int, logger *logrus.Entry) (interface{}, error)](#InitRedisCache)
* [func NewSentinelPool(sentinelAddrs []string, masterIdentifier []byte, redisPassword []byte, timeoutMilliseconds int, selectDatabase int, logger *logrus.Entry) *redis.Pool](#NewSentinelPool)
* [type GenericCache](#GenericCache)
  * [func NewCacheWithMultiPools(writeCachePool interface{}, readCachePool interface{}, cLevel Level, sharedSecret string, expirySeconds int, keyPrefix []byte, encryptData bool) *GenericCache](#NewCacheWithMultiPools)
  * [func NewCacheWithPool(cachePool interface{}, cType Type, cLevel Level, sharedSecret string, expirySeconds int, keyPrefix []byte, encryptData bool) *GenericCache](#NewCacheWithPool)
  * [func (c *GenericCache) Add(key string, data interface{}, exp time.Duration) (err error)](#GenericCache.Add)
  * [func (c *GenericCache) AddExistingEntry(key string, entry GenericCacheEntry, expiresAt int64) error](#GenericCache.AddExistingEntry)
  * [func (c *GenericCache) Decrement(key string, n uint64) (newValue uint64, err error)](#GenericCache.Decrement)
  * [func (c *GenericCache) Delete(key string) (err error)](#GenericCache.Delete)
  * [func (c *GenericCache) Exists(key string) (found bool, entry GenericCacheEntry, err error)](#GenericCache.Exists)
  * [func (c *GenericCache) Flush() error](#GenericCache.Flush)
  * [func (c *GenericCache) Get(key string, value interface{}) error](#GenericCache.Get)
  * [func (c *GenericCache) GetKey(entryData []byte) string](#GenericCache.GetKey)
  * [func (c *GenericCache) Increment(key string, n uint64) (newValue uint64, err error)](#GenericCache.Increment)
  * [func (c *GenericCache) NewGenericCacheEntry(data interface{}, exp time.Duration) GenericCacheEntry](#GenericCache.NewGenericCacheEntry)
  * [func (c *GenericCache) Replace(key string, data interface{}, exp time.Duration) (err error)](#GenericCache.Replace)
  * [func (c *GenericCache) Set(key string, data interface{}, exp time.Duration) (err error)](#GenericCache.Set)
* [type GenericCacheEntry](#GenericCacheEntry)
  * [func (e *GenericCacheEntry) Expired() bool](#GenericCacheEntry.Expired)
* [type InMemoryStore](#InMemoryStore)
  * [func NewInMemoryStore(maxEntries int, defaultExpiration, cleanupInterval time.Duration) (*InMemoryStore, error)](#NewInMemoryStore)
  * [func (c *InMemoryStore) Add(key string, value interface{}, exp time.Duration) error](#InMemoryStore.Add)
  * [func (c *InMemoryStore) Decrement(key string, n uint64) (uint64, error)](#InMemoryStore.Decrement)
  * [func (c *InMemoryStore) Delete(key string) error](#InMemoryStore.Delete)
  * [func (c InMemoryStore) DeleteExpired()](#InMemoryStore.DeleteExpired)
  * [func (c *InMemoryStore) Flush() error](#InMemoryStore.Flush)
  * [func (c *InMemoryStore) Get(key string, value interface{}) error](#InMemoryStore.Get)
  * [func (c *InMemoryStore) Increment(key string, n uint64) (uint64, error)](#InMemoryStore.Increment)
  * [func (c *InMemoryStore) NewGenericCacheEntry(data interface{}, exp time.Duration) (newEntry GenericCacheEntry, err error)](#InMemoryStore.NewGenericCacheEntry)
  * [func (c *InMemoryStore) Replace(key string, value interface{}, exp time.Duration) error](#InMemoryStore.Replace)
  * [func (c *InMemoryStore) Set(key string, value interface{}, exp time.Duration) error](#InMemoryStore.Set)
* [type Level](#Level)
* [type Type](#Type)


#### <a name="pkg-files">Package files</a>
[generic.go](/src/github.com/Bose/go-cache/generic.go) [generic_test_store.go](/src/github.com/Bose/go-cache/generic_test_store.go) [inmemory.go](/src/github.com/Bose/go-cache/inmemory.go) [pkcs.go](/src/github.com/Bose/go-cache/pkcs.go) [redispool.go](/src/github.com/Bose/go-cache/redispool.go) 



## <a name="pkg-variables">Variables</a>
``` go
var (
    // ErrPaddingSize - represents padding errors
    ErrPaddingSize = errors.New("padding size error")
)
```
``` go
var (
    // PKCS5 represents pkcs5 struct
    PKCS5 = &pkcs5{}
)
```
``` go
var (
    // PKCS7 - difference with pkcs5 only block must be 8
    PKCS7 = &pkcs5{}
)
```


## <a name="InitReadOnlyRedisCache">func</a> [InitReadOnlyRedisCache](/src/target/redispool.go?s=6447:6627#L208)
``` go
func InitReadOnlyRedisCache(readOnlyCacheURL string, cachePassword string, defaultExpMinutes int, maxConnections int, selectDatabase int, logger *logrus.Entry) (interface{}, error)
```
InitReadOnlyRedisCache - used by microservices to init their read-only redis cache.  Returns an interface{} so
other caches could be swapped in



## <a name="InitRedisCache">func</a> [InitRedisCache](/src/target/redispool.go?s=3061:3231#L121)
``` go
func InitRedisCache(useSentinel bool, defaultExpSeconds int, redisPassword []byte, timeoutMilliseconds int, selectDatabase int, logger *logrus.Entry) (interface{}, error)
```
InitRedisCache - used by microservices to init their redis cache.  Returns an interface{} so
other caches could be swapped in



## <a name="NewSentinelPool">func</a> [NewSentinelPool](/src/target/redispool.go?s=599:769#L40)
``` go
func NewSentinelPool(sentinelAddrs []string, masterIdentifier []byte, redisPassword []byte, timeoutMilliseconds int, selectDatabase int, logger *logrus.Entry) *redis.Pool
```
NewSentinelPool - create a new pool for Redis Sentinel




## <a name="GenericCache">type</a> [GenericCache](/src/target/generic.go?s=906:1144#L33)
``` go
type GenericCache struct {
    Cache     interface{}
    ReadCache *GenericCache

    DefaultExp time.Duration

    KeyPrefix   []byte
    Logger      *logrus.Entry
    EncryptData bool
    // contains filtered or unexported fields
}

```
GenericCache - represents the cache


	- Cache: empty interface to a persistent cache pool (could be writable if cType == Writable)
	- ReadCache: empty interface to a persistent read-only cache pool (cType == ReadOnly)
	- sharedSecret: used by GetKey() for generating signatures to be used as an entries primary key
	- DefaultExp: the default expiry for entries
	- cType: Writable or ReadOnly
	- cLevel: L1 (level 1) or L2 (level 2)
	- KeyPrefix: a prefex added to each key that's generated by GetKey()
	- Logger: the logger to use when writing logs







### <a name="NewCacheWithMultiPools">func</a> [NewCacheWithMultiPools](/src/target/generic.go?s=2159:2345#L70)
``` go
func NewCacheWithMultiPools(writeCachePool interface{}, readCachePool interface{}, cLevel Level, sharedSecret string, expirySeconds int, keyPrefix []byte, encryptData bool) *GenericCache
```
NewCacheWithMultiPools - creates a new generic cache for microservices using two Pools.  One pool for writes and a separate pool for reads


### <a name="NewCacheWithPool">func</a> [NewCacheWithPool](/src/target/generic.go?s=1575:1735#L56)
``` go
func NewCacheWithPool(cachePool interface{}, cType Type, cLevel Level, sharedSecret string, expirySeconds int, keyPrefix []byte, encryptData bool) *GenericCache
```
NewCacheWithPool - creates a new generic cache for microservices using a Pool for connecting (this cache should be read/write)





### <a name="GenericCache.Add">func</a> (\*GenericCache) [Add](/src/target/generic.go?s=7811:7898#L240)
``` go
func (c *GenericCache) Add(key string, data interface{}, exp time.Duration) (err error)
```
Add - adds an entry to the cache




### <a name="GenericCache.AddExistingEntry">func</a> (\*GenericCache) [AddExistingEntry](/src/target/generic.go?s=7232:7331#L228)
``` go
func (c *GenericCache) AddExistingEntry(key string, entry GenericCacheEntry, expiresAt int64) error
```
AddExistingEntry -




### <a name="GenericCache.Decrement">func</a> (\*GenericCache) [Decrement](/src/target/generic.go?s=13073:13156#L387)
``` go
func (c *GenericCache) Decrement(key string, n uint64) (newValue uint64, err error)
```
Decrement - Decrement an entry in the cache




### <a name="GenericCache.Delete">func</a> (\*GenericCache) [Delete](/src/target/generic.go?s=8474:8527#L261)
``` go
func (c *GenericCache) Delete(key string) (err error)
```
Delete - deletes an entry in the cache




### <a name="GenericCache.Exists">func</a> (\*GenericCache) [Exists](/src/target/generic.go?s=9030:9120#L276)
``` go
func (c *GenericCache) Exists(key string) (found bool, entry GenericCacheEntry, err error)
```
Exists - searches the cache for an entry




### <a name="GenericCache.Flush">func</a> (\*GenericCache) [Flush](/src/target/generic.go?s=13700:13736#L404)
``` go
func (c *GenericCache) Flush() error
```
Flush  - Flush all the keys in the cache




### <a name="GenericCache.Get">func</a> (\*GenericCache) [Get](/src/target/generic.go?s=14104:14167#L417)
``` go
func (c *GenericCache) Get(key string, value interface{}) error
```
Get -  retrieves and entry from the cache




### <a name="GenericCache.GetKey">func</a> (\*GenericCache) [GetKey](/src/target/generic.go?s=6223:6277#L196)
``` go
func (c *GenericCache) GetKey(entryData []byte) string
```
GetKey - return a key for the entryData




### <a name="GenericCache.Increment">func</a> (\*GenericCache) [Increment](/src/target/generic.go?s=12443:12526#L370)
``` go
func (c *GenericCache) Increment(key string, n uint64) (newValue uint64, err error)
```
Increment - Increment an entry in the cache




### <a name="GenericCache.NewGenericCacheEntry">func</a> (\*GenericCache) [NewGenericCacheEntry](/src/target/generic.go?s=5827:5925#L183)
``` go
func (c *GenericCache) NewGenericCacheEntry(data interface{}, exp time.Duration) GenericCacheEntry
```
NewGenericCacheEntry creates an entry with the data and all the time attribs set




### <a name="GenericCache.Replace">func</a> (\*GenericCache) [Replace](/src/target/generic.go?s=11377:11468#L343)
``` go
func (c *GenericCache) Replace(key string, data interface{}, exp time.Duration) (err error)
```
Replace - Replace an entry in the cache




### <a name="GenericCache.Set">func</a> (\*GenericCache) [Set](/src/target/generic.go?s=10005:10092#L302)
``` go
func (c *GenericCache) Set(key string, data interface{}, exp time.Duration) (err error)
```
Set - Set a key in the cache (over writting any existing entry)




## <a name="GenericCacheEntry">type</a> [GenericCacheEntry](/src/target/generic.go?s=1353:1443#L49)
``` go
type GenericCacheEntry struct {
    Data      interface{}
    TimeAdded int64
    ExpiresAt int64
}

```
GenericCacheEntry - represents a cached entry...


	- Data: the entries data represented as an empty interface
	- TimeAdded: epoc at the time of addtion
	- ExpiresAd: epoc at the time of expiry










### <a name="GenericCacheEntry.Expired">func</a> (\*GenericCacheEntry) [Expired](/src/target/generic.go?s=5593:5635#L174)
``` go
func (e *GenericCacheEntry) Expired() bool
```
Expired - is the entry expired?




## <a name="InMemoryStore">type</a> [InMemoryStore](/src/target/inmemory.go?s=187:232#L13)
``` go
type InMemoryStore struct {
    // contains filtered or unexported fields
}

```
InMemoryStore - an in memory LRU store with expiry







### <a name="NewInMemoryStore">func</a> [NewInMemoryStore](/src/target/inmemory.go?s=849:960#L40)
``` go
func NewInMemoryStore(maxEntries int, defaultExpiration, cleanupInterval time.Duration) (*InMemoryStore, error)
```
NewInMemoryStore - create a new in memory cache





### <a name="InMemoryStore.Add">func</a> (\*InMemoryStore) [Add](/src/target/inmemory.go?s=2830:2913#L108)
``` go
func (c *InMemoryStore) Add(key string, value interface{}, exp time.Duration) error
```
Add - add an entry




### <a name="InMemoryStore.Decrement">func</a> (\*InMemoryStore) [Decrement](/src/target/inmemory.go?s=3926:3997#L150)
``` go
func (c *InMemoryStore) Decrement(key string, n uint64) (uint64, error)
```
Decrement (see CacheStore interface)




### <a name="InMemoryStore.Delete">func</a> (\*InMemoryStore) [Delete](/src/target/inmemory.go?s=3284:3332#L124)
``` go
func (c *InMemoryStore) Delete(key string) error
```
Delete - delete an entry




### <a name="InMemoryStore.DeleteExpired">func</a> (InMemoryStore) [DeleteExpired](/src/target/inmemory.go?s=4564:4603#L177)
``` go
func (c InMemoryStore) DeleteExpired()
```
DeleteExpired - Delete all expired items from the cache.




### <a name="InMemoryStore.Flush">func</a> (\*InMemoryStore) [Flush](/src/target/inmemory.go?s=4434:4471#L171)
``` go
func (c *InMemoryStore) Flush() error
```
Flush (see CacheStore interface)




### <a name="InMemoryStore.Get">func</a> (\*InMemoryStore) [Get](/src/target/inmemory.go?s=1588:1652#L64)
``` go
func (c *InMemoryStore) Get(key string, value interface{}) error
```
Get - Get an entry




### <a name="InMemoryStore.Increment">func</a> (\*InMemoryStore) [Increment](/src/target/inmemory.go?s=3481:3552#L133)
``` go
func (c *InMemoryStore) Increment(key string, n uint64) (uint64, error)
```
Increment (see CacheStore interface)




### <a name="InMemoryStore.NewGenericCacheEntry">func</a> (\*InMemoryStore) [NewGenericCacheEntry](/src/target/inmemory.go?s=399:520#L24)
``` go
func (c *InMemoryStore) NewGenericCacheEntry(data interface{}, exp time.Duration) (newEntry GenericCacheEntry, err error)
```
NewGenericCacheEntry - create a new in memory cache entry




### <a name="InMemoryStore.Replace">func</a> (\*InMemoryStore) [Replace](/src/target/inmemory.go?s=3056:3143#L116)
``` go
func (c *InMemoryStore) Replace(key string, value interface{}, exp time.Duration) error
```
Replace - replace an entry




### <a name="InMemoryStore.Set">func</a> (\*InMemoryStore) [Set](/src/target/inmemory.go?s=2683:2766#L103)
``` go
func (c *InMemoryStore) Set(key string, value interface{}, exp time.Duration) error
```
Set - set an entry




## <a name="Level">type</a> [Level](/src/target/redispool.go?s=444:458#L28)
``` go
type Level int
```
Level - define a level for the cache: L1, L2, etc


``` go
const (
    // L1 ...
    L1 Level = iota + 1
    // L2 ...
    L2
)
```









## <a name="Type">type</a> [Type](/src/target/redispool.go?s=299:312#L18)
``` go
type Type int
```
Type - define the type of cache: read or write


``` go
const (
    // ReadOnly ...
    ReadOnly Type = iota
    // Writable ...
    Writable
)
```













- - -
Generated by [godoc2md](http://godoc.org/github.com/davecheney/godoc2md)
