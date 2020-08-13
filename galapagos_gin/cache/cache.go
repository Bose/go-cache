/*

	This was copied out of Jim-Lambert-Bose/cache and was "enhanced" to understand
	when honor a cache request by looking at the header: x-okay-to-cache
	in func CachePage
*/

package cache

import (
	"bytes"
	// #nosec G505 - this is just for encoding url (not crypto)
	"crypto/sha1"
	"encoding/gob"
	"io"
	"log"
	"net/http"
	"net/url"
	"runtime"
	"sync"
	"time"

	"github.com/Bose/cache/persistence"
	"github.com/gin-gonic/gin"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

const (
	CACHE_MIDDLEWARE_KEY = "gincontrib.cache"
)

var (
	PageCachePrefix = "gincontrib.page.cache"
)

type ResponseCache struct {
	Status int
	Header http.Header
	Data   []byte
}

type cachedWriter struct {
	gin.ResponseWriter
	status  int
	written bool
	store   persistence.CacheStore
	expire  time.Duration
	key     string
}

// RegisterResponseCacheGob registers the responseCache type with the encoding/gob package
func RegisterResponseCacheGob() {
	gob.Register(ResponseCache{})
}

var _ gin.ResponseWriter = &cachedWriter{}

func urlEscape(prefix string, u string) string {
	key := url.QueryEscape(u)
	if len(key) > 200 {
		// #nosec G401 - this is just for encoding url (not crypto)
		h := sha1.New()
		if _, err := io.WriteString(h, u); err != nil {
			log.Println(err.Error())
		}
		key = string(h.Sum(nil))
	}
	var buffer bytes.Buffer
	buffer.WriteString(prefix)
	buffer.WriteString(":")
	buffer.WriteString(key)
	return buffer.String()
}

func newCachedWriter(store persistence.CacheStore, expire time.Duration, writer gin.ResponseWriter, key string) *cachedWriter {
	return &cachedWriter{writer, 0, false, store, expire, key}
}

func (w *cachedWriter) WriteHeader(code int) {
	w.status = code
	w.written = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *cachedWriter) Status() int {
	return w.ResponseWriter.Status()
}

func (w *cachedWriter) Written() bool {
	return w.ResponseWriter.Written()
}

func (w *cachedWriter) Write(data []byte) (int, error) {
	ret, err := w.ResponseWriter.Write(data)
	if err == nil {
		store := w.store
		// jlambert - Aug 31, 2018
		// stopped appending the cache.Data slice to the data slice.. this causes dup JSON data in the cache
		// var cache responseCache
		// if err := store.Get(w.key, &cache); err == nil {
		// 	data = append(cache.Data, data...)
		// }

		//cache response
		val := ResponseCache{
			w.status,
			w.Header(),
			data,
		}

		// scope of this is private... jlambert Nov 2018
		storeErr := store.Set(w.key, val, w.expire)
		if storeErr != nil {
			// currently a no-op...what would we do anywho?
			// need logger
		}
	}
	return ret, err
}

func (w *cachedWriter) WriteString(data string) (n int, err error) {
	ret, err := w.ResponseWriter.WriteString(data)
	if err == nil {
		//cache response
		store := w.store
		val := ResponseCache{
			w.status,
			w.Header(),
			[]byte(data),
		}
		if err := store.Set(w.key, val, w.expire); err != nil {
			return 0, err
		}
	}
	return ret, err
}

// Cache Middleware
func Cache(store *persistence.CacheStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(CACHE_MIDDLEWARE_KEY, store)
		c.Next()
	}
}

func SiteCache(store persistence.CacheStore, expire time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cache ResponseCache
		url := c.Request.URL
		key := urlEscape(PageCachePrefix, url.RequestURI())
		if err := store.Get(key, &cache); err != nil {
			c.Next()
		} else {
			c.Writer.WriteHeader(cache.Status)
			for k, vals := range cache.Header {
				for _, v := range vals {
					c.Writer.Header().Add(k, v)
				}
			}
			if _, err := c.Writer.Write(cache.Data); err != nil {
				log.Println(err.Error())
			}
		}
	}
}

// CachePage Decorator
func CachePage(store persistence.CacheStore, expire time.Duration, handle gin.HandlerFunc) gin.HandlerFunc {

	return func(c *gin.Context) {

		var span opentracing.Span
		if cspan, ok := c.Get("tracing-context"); ok {
			span = startSpanWithParent(cspan.(opentracing.Span).Context(), "api-request-cache-page", c.Request.Method, c.Request.URL.Path)

		} else {
			span = startSpanWithHeader(&c.Request.Header, "api-request-cache-page", c.Request.Method, c.Request.URL.Path)
		}
		defer span.Finish()
		c.Set("tracing-context", span) // add the span to the context so it can be used for the duration of the request.

		var cache ResponseCache
		url := c.Request.URL
		key := urlEscape(PageCachePrefix, url.RequestURI())
		if err := store.Get(key, &cache); err != nil {
			// not in cache path
			log.Println(err.Error())
			// replace writer
			writer := newCachedWriter(store, expire, c.Writer, key)
			c.Writer = writer
			handle(c)
		} else {
			// silly signal... this is the custom mod that looks for x-okay-to-cache.
			if len(cache.Header.Get("x-okay-to-cache")) == 0 {
				// in cache, but not okay to cache
				log.Println("not okay to cache")
				writer := newCachedWriter(store, expire, c.Writer, key)
				c.Writer = writer
				handle(c)
				return
			}
			// end of custom mod for x-okay-to-cache
			// okay to cache and we found it in the cache
			c.Writer.WriteHeader(cache.Status)
			for k, vals := range cache.Header {
				for _, v := range vals {
					c.Writer.Header().Add(k, v)
				}
			}
			if _, err := c.Writer.Write(cache.Data); err != nil {
				log.Println(err.Error())
			}
		}
	}
}

// function to be used to cache pages in thread-safe manner
func CachePageAtomic(store persistence.CacheStore, expire time.Duration, handle gin.HandlerFunc) gin.HandlerFunc {
	var m sync.Mutex
	p := CachePage(store, expire, handle)
	return func(c *gin.Context) {
		m.Lock()
		defer m.Unlock()
		p(c)
	}
}

// StartSpanWithParent will start a new span with a parent span.
// example:
//      span:= StartSpanWithParent(c.Get("tracing-context"),
func startSpanWithParent(parent opentracing.SpanContext, operationName, method, path string) opentracing.Span {
	options := []opentracing.StartSpanOption{
		opentracing.Tag{Key: ext.SpanKindRPCServer.Key, Value: ext.SpanKindRPCServer.Value},
		opentracing.Tag{Key: string(ext.HTTPMethod), Value: method},
		opentracing.Tag{Key: string(ext.HTTPUrl), Value: path},
		opentracing.Tag{Key: "current-goroutines", Value: runtime.NumGoroutine()},
	}

	if parent != nil {
		options = append(options, opentracing.ChildOf(parent))
	}

	return opentracing.StartSpan(operationName, options...)
}

// StartSpanWithHeader will look in the headers to look for a parent span before starting the new span.
// example:
//  func handleGet(c *gin.Context) {
//     span := StartSpanWithHeader(&c.Request.Header, "api-request", method, path)
//     defer span.Finish()
//     c.Set("tracing-context", span) // add the span to the context so it can be used for the duration of the request.
//     personID := c.Param("personID")
//     span.SetTag("personID", personID)
//
func startSpanWithHeader(header *http.Header, operationName, method, path string) opentracing.Span {
	var wireContext opentracing.SpanContext
	if header != nil {
		wireContext, _ = opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(*header))
	}
	span := startSpanWithParent(wireContext, operationName, method, path)
	return span
	// return StartSpanWithParent(wireContext, operationName, method, path)
}
