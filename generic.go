package cache

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/Bose/go-cache/galapagos_gin/cache"
	"github.com/Jim-Lambert-Bose/cache/persistence"
	"github.com/sirupsen/logrus"
)

func init() {
	gob.Register(GenericCacheEntry{})
}

// GenericCache - represents the cache
//  - Cache: empty interface to a persistent cache pool (could be writable if cType == Writable)
//  - ReadCache: empty interface to a persistent read-only cache pool (cType == ReadOnly)
//  - sharedSecret: used by GetKey() for generating signatures to be used as an entries primary key
//  - DefaultExp: the default expiry for entries
//  - cType: Writable or ReadOnly
//  - cLevel: L1 (level 1) or L2 (level 2)
//  - KeyPrefix: a prefex added to each key that's generated by GetKey()
//  - Logger: the logger to use when writing logs
type GenericCache struct {
	Cache        interface{}
	ReadCache    *GenericCache
	sharedSecret []byte
	DefaultExp   time.Duration
	cType        Type
	cLevel       Level
	KeyPrefix    []byte
	Logger       *logrus.Entry
	EncryptData  bool
}

// GenericCacheEntry - represents a cached entry...
//   - Data: the entries data represented as an empty interface
//   - TimeAdded: epoc at the time of addtion
//   - ExpiresAd: epoc at the time of expiry
type GenericCacheEntry struct {
	Data      interface{}
	TimeAdded int64
	ExpiresAt int64
}

// NewCacheWithPool - creates a new generic cache for microservices using a Pool for connecting (this cache should be read/write)
func NewCacheWithPool(cachePool interface{}, cType Type, cLevel Level, sharedSecret string, expirySeconds int, keyPrefix []byte, encryptData bool) *GenericCache {
	storeExp := time.Duration(expirySeconds) * time.Second
	return &GenericCache{
		Cache:        cachePool,
		sharedSecret: []byte(sharedSecret),
		DefaultExp:   storeExp,
		cType:        cType,
		cLevel:       cLevel,
		KeyPrefix:    keyPrefix,
		EncryptData:  encryptData,
	}
}

// NewCacheWithMultiPools - creates a new generic cache for microservices using two Pools.  One pool for writes and a separate pool for reads
func NewCacheWithMultiPools(writeCachePool interface{}, readCachePool interface{}, cLevel Level, sharedSecret string, expirySeconds int, keyPrefix []byte, encryptData bool) *GenericCache {
	c := NewCacheWithPool(writeCachePool, Writable, cLevel, sharedSecret, expirySeconds, keyPrefix, encryptData)
	c.ReadCache = NewCacheWithPool(readCachePool, ReadOnly, cLevel, sharedSecret, expirySeconds, keyPrefix, encryptData)
	return c
}

// entrySignature - used for the cache primary key
func entrySignature(entry []byte, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(entry)
	sig := mac.Sum(nil)
	return hex.EncodeToString(sig)
}

func keyAndIV(secret []byte) (key []byte, IV []byte, err error) {
	if len(secret) < 16 {
		return nil, nil, fmt.Errorf("cache.keyAndIV: secret is too short - must be a min len of 16")
	}
	tmp := make([]byte, len(secret))
	copy(tmp, secret)
	key = tmp[:16]
	IV = tmp[:16]
	for i := len(IV)/2 - 1; i >= 0; i-- {
		opp := len(IV) - 1 - i
		IV[i], IV[opp] = IV[opp], IV[i]
	}
	return key, IV, nil
}

// encryptByteArray - symmetrically encrypt data with secret
func encryptByteArray(data []byte, secret []byte) ([]byte, error) {
	key, IV, err := keyAndIV(secret)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, len(data))
	mode := cipher.NewCBCEncrypter(block, IV)
	mode.CryptBlocks(ciphertext, data)
	return ciphertext[:], nil
}

// decryptByteArray - symmetrically decrypts data using secret
func decryptByteArray(data []byte, secret []byte) ([]byte, error) {
	key, IV, err := keyAndIV(secret)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, len(data))
	mode := cipher.NewCBCDecrypter(block, IV)
	mode.CryptBlocks(ciphertext, data)
	// log.Debug("Decrypted data: ", ciphertext)
	return ciphertext[:], nil
}

func (c *GenericCache) encryptEntry(data []byte) ([]byte, error) {
	// return data, nil
	if c.EncryptData {
		// fmt.Println("encrypt input: ", data)
		paddedData := PKCS7.Padding([]byte(data), 16)
		// fmt.Println("encrypt padded: ", paddedData)
		// fmt.Println("encrypt padded len: ", len(paddedData))
		// fmt.Println("encrypt secret: ", c.SharedSecret)
		encrypted, cryptErr := encryptByteArray(paddedData, c.sharedSecret)
		if cryptErr != nil {
			err := fmt.Errorf("GenericCache.encryptEntry: can't encrypt data: %s", cryptErr.Error())
			c.logError(err.Error())
			return nil, err
		}
		// fmt.Println("encrypt encrypted: ", encrypted)
		return []byte(encrypted), nil
	}
	return data, nil
}
func (c *GenericCache) decryptEntry(data []byte) ([]byte, error) {
	if c.EncryptData {
		// fmt.Println("decrypt encrypted: ", data)
		// fmt.Println("encrypt secret: ", c.SharedSecret)
		decryptedData, cryptErr := decryptByteArray(data, c.sharedSecret)
		// fmt.Println("decrypt decrypted: ", decryptedData)
		if cryptErr != nil {
			err := fmt.Errorf("GenericCache.decryptEntry: can't decrypt data: %s", cryptErr.Error())
			c.logError(err.Error())
			return nil, err
		}
		unpaddedData, cryptErr := PKCS7.Unpadding([]byte(decryptedData), 16)
		if cryptErr != nil {
			err := fmt.Errorf("GenericCache.decryptEntry: can't unpadding error: %s", cryptErr.Error())
			c.logError(err.Error())
			return nil, err
		}
		return unpaddedData, nil
	}
	return data, nil
}

// Expired - is the entry expired?
func (e *GenericCacheEntry) Expired() bool {
	if e.ExpiresAt == 0 {
		return false
	}
	t := time.Unix(e.ExpiresAt, 0)
	return t.Before(time.Now())
}

// NewGenericCacheEntry creates an entry with the data and all the time attribs set
func (c *GenericCache) NewGenericCacheEntry(data interface{}, exp time.Duration) GenericCacheEntry {
	var t time.Duration
	if exp != 0 {
		t = exp
	} else {
		t = c.DefaultExp
	}
	now := time.Now().Unix()
	expiresAt := now + int64(t/time.Second) // convert from nanoseconds
	return GenericCacheEntry{Data: data, TimeAdded: now, ExpiresAt: expiresAt}
}

// GetKey - return a key for the entryData
func (c *GenericCache) GetKey(entryData []byte) string {
	if c.KeyPrefix != nil {
		return fmt.Sprintf("%s::%s", c.KeyPrefix, entrySignature(entryData, c.sharedSecret))
	}
	return entrySignature(entryData, c.sharedSecret)
}

// logDebug - send an entry to the debug stream if the logger is defined
func (c *GenericCache) logDebug(entry string) {
	if c.Logger != nil {
		c.Logger.Debug(entry)
		return
	}
}

// logError - send an entry to the error stream
func (c *GenericCache) logError(entry string) {
	if c.Logger != nil {
		c.Logger.Error(entry)
		return
	}
	fmt.Fprintln(os.Stderr, entry)
}

// debugEntry - spit out some debug logs for the cache entry
func (c *GenericCache) debugEntry(key string, e GenericCacheEntry) {
	added := time.Unix(e.TimeAdded, 0).Format(time.RFC3339)
	expAt := time.Unix(e.ExpiresAt, 0).Format(time.RFC3339)
	c.logDebug(fmt.Sprintf("GenericCache.debugEntry: L%v/T%v, Key == %s, TimeAdded == %s, ExpiresAt == %s", c.cLevel, c.cType, key, added, expAt))
}

// AddExistingEntry -
func (c *GenericCache) AddExistingEntry(key string, entry GenericCacheEntry, expiresAt int64) error {
	//	return nil // disable the L1 cache
	c.logDebug(fmt.Sprintf("GenericCache.AddExistingEntry: L%v/T%v key == %s", c.cLevel, c.cType, key))
	expCacheAt := time.Duration(expiresAt-time.Now().Unix()) * time.Second
	if err := c.Cache.(persistence.CacheStore).Set(key, entry, expCacheAt); err != nil {
		c.logError(fmt.Sprintf("GenericCache.AddExistingEntry: L%v/T%v error == %s", c.cLevel, c.cType, err.Error()))
		return err
	}
	return nil
}

// Add - adds an entry to the cache
func (c *GenericCache) Add(key string, data interface{}, exp time.Duration) (err error) {
	if c.Cache == nil {
		err := fmt.Errorf("GenericCache.Add: error - no L%v/T%v cache intialized", c.cLevel, c.cType)
		c.logError(err.Error())
		return err
	}
	c.logDebug(fmt.Sprintf("GenericCache.Add: L%v/T%v key == %s", c.cLevel, c.cType, key))
	var t time.Duration
	if exp != 0 {
		t = exp
	} else {
		t = c.DefaultExp
	}
	if err := c.Cache.(persistence.CacheStore).Add(key, data, t); err != nil {
		c.logError(fmt.Sprintf("GenericCache.Add: L%v/T%v error == %s", c.cLevel, c.cType, err.Error()))
		return err
	}
	return nil
}

// Delete - deletes an entry in the cache
func (c *GenericCache) Delete(key string) (err error) {
	if c.Cache == nil {
		err := fmt.Errorf("GenericCache.Delete: error - no L%v/T%v cache intialized", c.cLevel, c.cType)
		c.logError(err.Error())
		return err
	}
	c.logDebug(fmt.Sprintf("GenericCache.Delete: L%v/T%v key == %s", c.cLevel, c.cType, key))
	if err := c.Cache.(persistence.CacheStore).Delete(key); err != nil {
		if err.Error() != persistence.ErrCacheMiss.Error() {
			c.logError(fmt.Sprintf("GenericCache.Delete: L%v/T%v error == %s", c.cLevel, c.cType, err.Error()))
			return err
		}
		return persistence.ErrCacheMiss
	}
	return nil
}

// Exists - searches the cache for an entry
func (c *GenericCache) Exists(key string) (found bool, entry GenericCacheEntry, err error) {
	if c.ReadCache != nil {
		c.ReadCache.Logger = c.Logger
		return c.ReadCache.Exists(key)
	}
	if c.Cache == nil {
		err := fmt.Errorf("GenericCache.Add: error - no L%v/T%v cache intialized", c.cLevel, c.cType)
		c.logError(err.Error())
		return false, entry, err
	}
	c.logDebug(fmt.Sprintf("GenericCache.Exists: L%v/T%v looking for key %s and encryption %v", c.cLevel, c.cType, key, c.EncryptData))
	//	if err := c.Cache.(persistence.CacheStore).Get(key, &entry); err != nil {
	if err := c.Get(key, &entry); err != nil {
		switch c.cLevel {
		case L1:
			c.logDebug(fmt.Sprintf("GenericCache::Exist: L1 not found - %s", err.Error()))
		case L2:
			c.logDebug(fmt.Sprintf("GenericCache::Exist: L2 not found - %s", err.Error()))
		}
		return false, entry, err
	}
	c.debugEntry(key, entry)
	return true, entry, nil
}

// Set - Set a key in the cache (over writting any existing entry)
func (c *GenericCache) Set(key string, data interface{}, exp time.Duration) (err error) {
	if c.Cache == nil {
		err := fmt.Errorf("GenericCache.Set: error - no L%v/T%v cache intialized", c.cLevel, c.cType)
		c.logError(err.Error())
		return err
	}
	c.logDebug(fmt.Sprintf("GenericCache.Set: L%v/T%v key == %s", c.cLevel, c.cType, key))
	var t time.Duration
	if exp != 0 {
		t = exp
	} else {
		t = c.DefaultExp
	}
	if c.EncryptData {
		valueType := fmt.Sprintf("%T", data)
		c.logDebug(fmt.Sprintf("GenericCache.Get: L%v/T%v key == %s and entry type == %s", c.cLevel, c.cType, key, valueType))
		switch valueType {
		case "GenericCacheEntry", "cache.GenericCacheEntry":
			byt := []byte("")
			b := bytes.NewBuffer(byt)
			encoder := gob.NewEncoder(b)
			entryData := data.(GenericCacheEntry).Data
			if err = encoder.Encode(&entryData); err != nil {
				return err
			}
			encryptedData, err := c.encryptEntry(b.Bytes())
			if err != nil {
				c.logError(fmt.Sprintf("GenericCache.Set: L%v/T%v error == %s", c.cLevel, c.cType, err.Error()))
				return err
			}
			data = c.NewGenericCacheEntry(encryptedData, exp)
		}
	}
	if err := c.Cache.(persistence.CacheStore).Set(key, data, t); err != nil {
		c.logError(fmt.Sprintf("GenericCache.Set: L%v/T%v error == %s", c.cLevel, c.cType, err.Error()))
		return err
	}
	return nil
}

// Replace - Replace an entry in the cache
func (c *GenericCache) Replace(key string, data interface{}, exp time.Duration) (err error) {
	if c.Cache == nil {
		err := fmt.Errorf("GenericCache.Replace: error - no L%v/T%v cache intialized", c.cLevel, c.cType)
		c.logError(err.Error())
		return err

	}
	c.logDebug(fmt.Sprintf("GenericCache.Replace: L%v/T%v key == %s and exp == %v", c.cLevel, c.cType, key, exp))
	var t time.Duration
	if exp != 0 {
		t = exp
	} else {
		t = c.DefaultExp
	}
	c.logDebug(fmt.Sprintf("GenericCache.Replace: L%v/T%v and t == %v", c.cLevel, c.cType, t))
	// now := time.Now().Unix()
	// expiresAt := now + int64(t/time.Second) // convert from nanoseconds
	// entry := GenericCacheEntry{Data: data, TimeAdded: now, ExpiresAt: expiresAt}
	// if err := c.Cache.(persistence.CacheStore).Replace(key, entry, t); err != nil {
	if err := c.Cache.(persistence.CacheStore).Replace(key, data, t); err != nil {
		c.logError(fmt.Sprintf("GenericCache.Replace: L%v/T%v error == %s", c.cLevel, c.cType, err.Error()))
		return err
	}
	return nil
}

// Increment - Increment an entry in the cache
func (c *GenericCache) Increment(key string, n uint64) (newValue uint64, err error) {
	if c.Cache == nil {
		err := fmt.Errorf("GenericCache.Increment: error - no L%v/T%v cache intialized", c.cLevel, c.cType)
		c.logError(err.Error())
		return 0, err

	}
	c.logDebug(fmt.Sprintf("GenericCache.Increment: L%v/T%v key == %s", c.cLevel, c.cType, key))
	newValue, err = c.Cache.(persistence.CacheStore).Increment(key, n)
	if err != nil {
		c.logError(fmt.Sprintf("GenericCache.Increment: L%v/T%v error == %s", c.cLevel, c.cType, err.Error()))
		return 0, err
	}
	return newValue, nil
}

// RedisExpireAt - get the TTL of an entry
func (c *GenericCache) RedisExpireAt(key string, epoc uint64) error {
	if c.Cache == nil {
		err := fmt.Errorf("GenericCache.RedisExpireAt: error - no L%v/T%v cache intialized", c.cLevel, c.cType)
		c.logError(err.Error())
		return err

	}
	c.logDebug(fmt.Sprintf("GenericCache.RedisExpireAt: L%v/T%v key == %s", c.cLevel, c.cType, key))
	err := c.Cache.(*persistence.RedisStore).ExpireAt(key, epoc)
	if err != nil {
		c.logError(fmt.Sprintf("GenericCache.RedisExpireAt: L%v/T%v error == %s", c.cLevel, c.cType, err.Error()))
		return err
	}
	return nil

}

func (c *GenericCache) RedisGetExpiresIn(key string) (int64, error) {
	if c.Cache == nil {
		err := fmt.Errorf("GenericCache.RedisGetExpiresIn: error - no L%v/T%v cache intialized", c.cLevel, c.cType)
		c.logError(err.Error())
		return 0, err
	}
	c.logDebug(fmt.Sprintf("GenericCache.RedisExpireAt: L%v/T%v key == %s", c.cLevel, c.cType, key))
	ttl, err := c.Cache.(*persistence.RedisStore).GetExpiresIn(key)
	if err != nil {
		c.logError(fmt.Sprintf("GenericCache.RedisExpireAt: L%v/T%v error == %s", c.cLevel, c.cType, err.Error()))
		return 0, err
	}
	return ttl, err
}

// RedisIncrementAtomic - Increment an entry in the cache
func (c *GenericCache) RedisIncrementAtomic(key string, n uint64) (newValue uint64, err error) {
	if c.Cache == nil {
		err := fmt.Errorf("GenericCache.IncrementAtomic: error - no L%v/T%v cache intialized", c.cLevel, c.cType)
		c.logError(err.Error())
		return 0, err

	}
	c.logDebug(fmt.Sprintf("GenericCache.IncrementAtomic: L%v/T%v key == %s", c.cLevel, c.cType, key))
	newValue, err = c.Cache.(*persistence.RedisStore).IncrementAtomic(key, n)
	if err != nil {
		c.logError(fmt.Sprintf("GenericCache.IncrementAtomic: L%v/T%v error == %s", c.cLevel, c.cType, err.Error()))
		return 0, err
	}
	return newValue, nil
}

// RedisIncrementCheckSet - Increment an entry in the cache
func (c *GenericCache) RedisIncrementCheckSet(key string, n uint64) (newValue uint64, err error) {
	if c.Cache == nil {
		err := fmt.Errorf("GenericCache.IncrementCheckSet: error - no L%v/T%v cache intialized", c.cLevel, c.cType)
		c.logError(err.Error())
		return 0, err

	}
	c.logDebug(fmt.Sprintf("GenericCache.IncrementCheckSet: L%v/T%v key == %s", c.cLevel, c.cType, key))
	newValue, err = c.Cache.(*persistence.RedisStore).IncrementCheckSet(key, n)
	if err != nil {
		c.logError(fmt.Sprintf("GenericCache.IncrementCheckSet: L%v/T%v error == %s", c.cLevel, c.cType, err.Error()))
		return 0, err
	}
	return newValue, nil
}

// Decrement - Decrement an entry in the cache
func (c *GenericCache) Decrement(key string, n uint64) (newValue uint64, err error) {
	if c.Cache == nil {
		err := fmt.Errorf("GenericCache.Decrement: error - no L%v/T%v cache intialized", c.cLevel, c.cType)
		c.logError(err.Error())
		return 0, err

	}
	c.logDebug(fmt.Sprintf("GenericCache.Decrement: L%v/T%v key == %s", c.cLevel, c.cType, key))
	newValue, err = c.Cache.(persistence.CacheStore).Decrement(key, n)
	if err != nil {
		c.logError(fmt.Sprintf("GenericCache.Decrement: L%v/T%v error == %s", c.cLevel, c.cType, err.Error()))
		return 0, err
	}
	return newValue, nil
}

// Flush  - Flush all the keys in the cache
func (c *GenericCache) Flush() error {
	if c.Cache == nil {
		err := fmt.Errorf("GenericCache.Flush: error - no L%v/T%v cache intialized", c.cLevel, c.cType)
		c.logError(err.Error())
		return err

	}
	c.logDebug(fmt.Sprintf("GenericCache.Flush: flushing all keys for L%v/T%v key", c.cLevel, c.cType))
	c.Cache.(persistence.CacheStore).Flush()
	return nil
}

// Get -  retrieves and entry from the cache
func (c *GenericCache) Get(key string, value interface{}) error {
	if c.Cache == nil {
		err := fmt.Errorf("GenericCache.Get: error - no L%v/T%v cache intialized", c.cLevel, c.cType)
		c.logError(err.Error())
		return err
	}
	valueType := fmt.Sprintf("%T", value)
	c.logDebug(fmt.Sprintf("GenericCache.Get: L%v/T%v key == %s and entry type == %s and encryption == %v", c.cLevel, c.cType, key, valueType, c.EncryptData))
	switch valueType {
	case "*cache.ResponseCache":
		entry := value.(*cache.ResponseCache)
		err := c.Cache.(persistence.CacheStore).Get(key, entry)
		if err != nil {
			if err.Error() != persistence.ErrCacheMiss.Error() {
				c.logError(fmt.Sprintf("GenericCache.Get: L%v/T%v error == %s", c.cLevel, c.cType, err.Error()))
				return err
			}
			return persistence.ErrCacheMiss
		}
		return nil
	case "*GenericCacheEntry", "*cache.GenericCacheEntry":
		entry := value.(*GenericCacheEntry)
		err := c.Cache.(persistence.CacheStore).Get(key, entry)
		if err != nil {
			if err.Error() != persistence.ErrCacheMiss.Error() {
				c.logError(fmt.Sprintf("GenericCache.Get: L%v/T%v error == %s", c.cLevel, c.cType, err.Error()))
				return err
			}
			return persistence.ErrCacheMiss
		}
		if c.EncryptData {
			decryptedData, err := c.decryptEntry(entry.Data.([]byte))
			if err != nil {
				c.logError(fmt.Sprintf("GenericCache.Get: L%v/T%v error == %s", c.cLevel, c.cType, err.Error()))
				return err
			}
			// fmt.Println("ENC: ", decryptedData)
			b := bytes.NewBuffer(decryptedData)
			decoder := gob.NewDecoder(b)
			if err = decoder.Decode(&entry.Data); err != nil {
				return err
			}
			// fmt.Println("GET entry.Data: ", entry.Data)
		}
		return nil
	case "*string":
		entry := value.(*string)
		err := c.Cache.(persistence.CacheStore).Get(key, entry)
		if err != nil {
			if err.Error() != persistence.ErrCacheMiss.Error() {
				c.logError(fmt.Sprintf("GenericCache.Get: L%v/T%v error == %s", c.cLevel, c.cType, err.Error()))
				return err
			}
			return persistence.ErrCacheMiss
		}
		return nil
	case "*int", "*int8", "*int16", "*int32", "*int64", "*uint", "*uint8", "*uint16", "*uint32", "*uint64":
		err := c.Cache.(persistence.CacheStore).Get(key, value)
		if err != nil {
			if err.Error() != persistence.ErrCacheMiss.Error() {
				c.logError(fmt.Sprintf("GenericCache.Get: L%v/T%v error == %s", c.cLevel, c.cType, err.Error()))
				return err
			}
			return persistence.ErrCacheMiss
		}
		return nil
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		err := c.Cache.(persistence.CacheStore).Get(key, &value)
		if err != nil {
			if err.Error() != persistence.ErrCacheMiss.Error() {
				c.logError(fmt.Sprintf("GenericCache.Get: L%v/T%v error == %s", c.cLevel, c.cType, err.Error()))
				return err
			}
			return persistence.ErrCacheMiss
		}
		return nil
	case "*float64":
		entry := value.(*float64)
		err := c.Cache.(persistence.CacheStore).Get(key, entry)
		if err != nil {
			if err.Error() != persistence.ErrCacheMiss.Error() {
				c.logError(fmt.Sprintf("GenericCache.Get: L%v/T%v error == %s", c.cLevel, c.cType, err.Error()))
				return err
			}
			return persistence.ErrCacheMiss
		}
		return nil
	}
	err := fmt.Errorf("GenericCache.Get: L%v/T%v error - not supported type - %s", c.cLevel, c.cType, valueType)
	c.logError(err.Error())
	return persistence.ErrNotSupport
}
