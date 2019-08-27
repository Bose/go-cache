package cache

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

var benchInMemoryStore *GenericCache
var benchInMemoryStoreEncrypted *GenericCache

func init() {
	benchInMemoryStore = newBenchGenericStoreInMemory(time.Hour, false)
	benchInMemoryStoreEncrypted = newBenchGenericStoreInMemory(time.Hour, true)

}
func benchmarkTypicalGetSetInMemory(i int, b *testing.B) {

	for n := 0; n < b.N; n++ {
		benchTypicalGetSet(b, benchInMemoryStore)
	}
}

func benchmarkTypicalGetSetEncryptedInMemory(i int, b *testing.B) {
	for n := 0; n < b.N; n++ {
		benchTypicalGetSet(b, benchInMemoryStoreEncrypted)
	}
}
func BenchmarkGetSetInMemory1(b *testing.B)  { benchmarkTypicalGetSetInMemory(1, b) }
func BenchmarkGetSetInMemory2(b *testing.B)  { benchmarkTypicalGetSetInMemory(2, b) }
func BenchmarkGetSetInMemory3(b *testing.B)  { benchmarkTypicalGetSetInMemory(3, b) }
func BenchmarkGetSetInMemory10(b *testing.B) { benchmarkTypicalGetSetInMemory(10, b) }
func BenchmarkGetSetInMemory20(b *testing.B) { benchmarkTypicalGetSetInMemory(20, b) }
func BenchmarkGetSetInMemory40(b *testing.B) { benchmarkTypicalGetSetInMemory(40, b) }

func BenchmarkGetSetEncryptedInMemory1(b *testing.B)  { benchmarkTypicalGetSetEncryptedInMemory(1, b) }
func BenchmarkGetSetEncryptedInMemory2(b *testing.B)  { benchmarkTypicalGetSetEncryptedInMemory(2, b) }
func BenchmarkGetSetEncryptedInMemory3(b *testing.B)  { benchmarkTypicalGetSetEncryptedInMemory(3, b) }
func BenchmarkGetSetEncryptedInMemory10(b *testing.B) { benchmarkTypicalGetSetEncryptedInMemory(10, b) }
func BenchmarkGetSetEncryptedInMemory20(b *testing.B) { benchmarkTypicalGetSetEncryptedInMemory(20, b) }
func BenchmarkGetSetEncryptedInMemory40(b *testing.B) { benchmarkTypicalGetSetEncryptedInMemory(40, b) }

func newBenchGenericStoreInMemory(defaultExpiration time.Duration, encryptEntries bool) *GenericCache {
	logrus.SetLevel(logrus.ErrorLevel)
	logger := logrus.WithFields(logrus.Fields{
		"requestID": "unknown",
		"method":    "generic_test",
		"path":      "none",
	})
	cacheWritePool, err := NewInMemoryStore(100, defaultExpiration, defCleanupInterval, true, "")
	if err != nil {
		panic("can't create inmemory store: " + err.Error())
	}
	logger.Info("cacheWritePool initialized")
	c := NewCacheWithPool(cacheWritePool, Writable, L2, sharedSecret, defExpSeconds, []byte("test"), encryptEntries)
	c.Logger = logger
	return c
}
