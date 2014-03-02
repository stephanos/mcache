// Copyright 2013 by sdm. All rights reserved.

/*
mcache is a package to provide an in-memory key/value cache:
 	- thread safe
	- expiration
	- CAS update
*/
package mcache

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// ExpirationKind is the kind of cache entry expiration
type ExpirationKind int

const (
	// SlidingExpiration means cache entry should be evicted if it has not been accessed in a given span of time.
	SlidingExpiration ExpirationKind = 0

	// AbsoluteExpiration means cache entry should be evicted after a specified duration.
	AbsoluteExpiration ExpirationKind = 1
)

const (
	// _minTickInterval is the min interval duration to run expiration check process
	_minTickInterval time.Duration = time.Second

	// _minExpiration is the min duration of cache entry expiration
	_minExpiration time.Duration = time.Microsecond

	// _noExpiration means cache entry will not be expirated
	_noExpiration time.Duration = 1000 * 1000 * time.Hour
)

// TickInterval is the the interval duration of expiration check
var TickInterval time.Duration = time.Minute

// item is cache entry item
type item struct {
	Key        string
	Value      interface{}
	Version    int
	Kind       ExpirationKind
	Expiration time.Duration
	ExpAt      time.Time
}

// MCache is cache in memory
type MCache struct {
	*mcache
}

// https://groups.google.com/forum/?fromgroups=#!topic/golang-nuts/1ItNOOj8yW8
type mcache struct {
	sync.RWMutex
	items map[string]*item
	stop  chan bool
	tick  <-chan time.Time
}

func NewMemoryCache(expire bool) *MCache {
	cache := &mcache{
		items: map[string]*item{},
		stop:  make(chan bool),
	}
	c := &MCache{cache}

	if expire {
		go cache.startTick()
		runtime.SetFinalizer(c, stopTick)
	}

	return c
}

// PutP set a cache entry with very long expiration time
func (self *mcache) PutP(key string, value interface{}) {
	self.Put(key, value, 0, AbsoluteExpiration)
}

// PutAbs set a cache entry with AbsoluteExpiration
func (self *mcache) PutAbs(key string, value interface{}, expire time.Duration) {
	self.Put(key, value, expire, AbsoluteExpiration)
}

// PutSlid set a cache entry with SlidingExpiration
func (self *mcache) PutSlid(key string, value interface{}, expire time.Duration) {
	self.Put(key, value, expire, SlidingExpiration)
}

// Put set a cache entry with expire time span and kind
func (self *mcache) Put(key string, value interface{}, expire time.Duration, kind ExpirationKind) {
	self.Lock()
	defer self.Unlock()

	self.put(key, value, expire, kind)
}

// Get return a cached value, it return false if key doesn't exist
func (self *mcache) Get(key string) (interface{}, bool) {
	x, ok := self.get(key)
	if !ok {
		return nil, false
	}

	x.touch()
	return x.Value, true
}

// GetV return cached value and it's version
func (self *mcache) GetV(key string) (interface{}, int, bool) {
	x, ok := self.get(key)
	if !ok {
		return nil, 0, false
	}

	x.touch()
	return x.Value, x.Version, true
}

// Add insert a cache entry, it return false if key exist
func (self *mcache) Add(key string, value interface{}, expire time.Duration, kind ExpirationKind) bool {
	self.Lock()
	defer self.Unlock()

	x, ok := self.items[key]
	if !ok {
		self.put(key, value, expire, kind)
		return true
	}

	if x.Expiration >= _minExpiration && x.expired() {
		self.put(key, value, expire, kind)
		return true
	}

	return false
}

// Update update cache entry, it return false if key doesn't exist
func (self *mcache) Update(key string, value interface{}) bool {
	return self.update(key, -1, value)
}

// UpdateV update cache entry when version match
func (self *mcache) UpdateV(key string, version int, value interface{}) bool {
	return self.update(key, version, value)
}

// Delete delete cache entry from the cache
func (self *mcache) Delete(key string) {
	self.delete(key)
}

// DeleteMulti delete some keys from cache
func (self *mcache) DeleteMulti(keys []string) {
	if keys == nil || len(keys) == 0 {
		return
	}

	self.Lock()
	defer self.Unlock()

	for _, k := range keys {
		delete(self.items, k)
	}
}

// Clear deletes everything from the cache
func (self *mcache) Clear() {
	self.Lock()
	defer self.Unlock()
	self.items = map[string]*item{}
}

// Count return number of cache entry, maybe include expired
func (self *mcache) Count() int {
	self.Lock()
	defer self.Unlock()

	n := len(self.items)
	return n
}

// Exists return whether the key exist
func (self *mcache) Exists(key string) bool {
	_, ok := self.get(key)
	return ok
}

// Keys return all cache keys
func (self *mcache) Keys() []string {
	self.RLock()
	defer self.RUnlock()

	keys := make([]string, 0, 255)

	for k, v := range self.items {
		if !v.expired() {
			keys = append(keys, k)
		}
	}

	return keys
}

// Stat return MCache stat information
func (self *mcache) Stat() string {
	self.RLock()
	defer self.RUnlock()

	var buf bytes.Buffer
	buf.WriteString("start stat \n")
	buf.WriteString(fmt.Sprintf("Len=%d \n", len(self.items)))
	for k, v := range self.items {
		buf.WriteString(fmt.Sprintf("key=%s; value=%v; ExpAt=%v; \n", k, v.Value, v.ExpAt))
	}
	buf.WriteString("end stat \n")
	return buf.String()
}

func (self *mcache) update(key string, version int, value interface{}) bool {
	x, ok := self.get(key)
	if !ok {
		return false
	}

	self.Lock()
	defer self.Unlock()

	if version >= 0 && x.Version != version {
		return false
	}

	x.Value = value
	x.Version++
	x.touch()

	return true
}

// expired return cache entry expired or not
func (item *item) expired() bool {
	//return time.Now().UnixNano() > item.ExpAtN
	return time.Now().After(item.ExpAt)
}

// touch can refresh cache entry expiration time
func (item *item) touch() {
	if item.Kind != SlidingExpiration {
		return
	}

	if item.Expiration >= _minExpiration {
		item.ExpAt = time.Now().Add(item.Expiration)
	}
}

func (self *mcache) put(key string, value interface{}, expire time.Duration, kind ExpirationKind) {
	var expAt time.Time
	if expire < _minExpiration {
		expire = 0
		expAt = time.Now().Add(_noExpiration)
	} else {
		expAt = time.Now().Add(expire)
	}

	self.items[key] = &item{
		Key:        key,
		Value:      value,
		Version:    0,
		Kind:       kind,
		Expiration: expire,
		ExpAt:      expAt,
	}
	return
}

func (self *mcache) get(key string) (*item, bool) {
	self.RLock()
	x, ok := self.items[key]
	self.RUnlock()

	if !ok {
		return nil, false
	}

	if x.Expiration < _minExpiration {
		return x, ok
	}
	if x.expired() {
		//self.delete(key)
		return nil, false
	}

	return x, ok
}

func (self *mcache) delete(key string) {
	self.Lock()
	defer self.Unlock()
	delete(self.items, key)
}
