package mcache

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func assetEqual(t *testing.T, m string, expect, actual interface{}) {
	if actual != expect {
		t.Error(m, "expect:", expect, "actual:", actual)
	}
}

func assetGet(t *testing.T, c *MCache, k string, expect interface{}) {
	if x, ok := c.Get(k); !ok {
		t.Error("Get Error, can not get key:", k)
	} else if x != expect {
		t.Error("Get Error, cache value is incorrect:", k, "expect:", expect, "actual:", x)
	}
}

func TestAdd(t *testing.T) {
	cache := NewMemoryCache(true)
	key := "int"

	cache.Add("Add", "Add", time.Minute, AbsoluteExpiration)
	cache.Add("Add", "Add", time.Minute, AbsoluteExpiration)

	if ok := cache.Add(key, 11, time.Minute, AbsoluteExpiration); !ok {
		t.Errorf("Add error, should return true")
	}
	assetGet(t, cache, key, 11)

	if ok := cache.Add(key, 22, time.Minute, AbsoluteExpiration); ok {
		t.Errorf("Add error, should return false")
	}

}

func TestBasic(t *testing.T) {
	cache := NewMemoryCache(true)

	x, ok := cache.Get("int")
	if ok || x != nil {
		t.Error("Get Error, Key shouldn't exist:", "int")
	}

	cache.Put("int", 11, time.Minute, AbsoluteExpiration)
	cache.PutP("string", "string")
	cache.PutAbs("float", 3.14, time.Second)

	assetGet(t, cache, "int", 11)
	assetGet(t, cache, "string", "string")
	assetGet(t, cache, "float", 3.14)

	cache.Update("int", 22)
	cache.Update("string", "stringstring")
	cache.Update("float", 1.1)

	assetGet(t, cache, "int", 22)
	assetGet(t, cache, "string", "stringstring")
	assetGet(t, cache, "float", 1.1)

	assetEqual(t, "Count Error", 3, cache.Count())
	assetEqual(t, "Keys Error", 3, len(cache.Keys()))
	assetEqual(t, "Exists Error: int", true, cache.Exists("int"))

	cache.Delete("int")

	assetEqual(t, "Count Error", 2, cache.Count())
	assetEqual(t, "Keys Error", 2, len(cache.Keys()))
	assetEqual(t, "Exists Error: int", false, cache.Exists("int"))
	assetEqual(t, "Update Error: int", false, cache.Update("int", 1))

	cache.DeleteMulti([]string{"string", "float"})

	assetEqual(t, "Count Error", 0, cache.Count())
	assetEqual(t, "Keys Error", 0, len(cache.Keys()))
	assetEqual(t, "Exists Error: int", false, cache.Exists("string"))
	assetEqual(t, "Update Error: int", false, cache.Update("string", 1))

	cache.PutP("string", "string")
	cache.Clear()

	assetEqual(t, "Count Error", 0, cache.Count())
	assetEqual(t, "Keys Error", 0, len(cache.Keys()))

	cache = nil
}

func TestCas(t *testing.T) {
	cache := NewMemoryCache(true)

	key := "int"
	i := 0
	cache.PutP(key, i)
	i++
	cache.Update(key, i)

	if _, v, _ := cache.GetV(key); v != i {
		t.Errorf("GetV Error, version expect %d, actual %d", i, v)
	}

	cache.Update("int", i)
	if ok := cache.UpdateV(key, i, i); ok {
		t.Errorf("UpdateV Error, expect %d, actual %d", false, ok)
	}

	i++
	if ok := cache.UpdateV(key, i, i); !ok {
		t.Errorf("UpdateV Error, expect %d, actual %d", true, ok)
	}

}

func TestExpire(t *testing.T) {
	cache := NewMemoryCache(true)

	var interval = 10 * time.Millisecond

	cache.Put("a", 1, 2*interval, AbsoluteExpiration)

	time.Sleep(1 * interval)
	if _, ok := cache.Get("a"); !ok {
		t.Error("Expire error, cache a should exists")
	}

	//<-time.After(1 * time.Millisecond)
	time.Sleep(2 * interval)
	if _, ok := cache.Get("a"); ok {
		t.Error("Expire error, cache a should be expired")
	}

	// test absolute expiration
	cache.PutAbs("b", 1, 2*interval)

	time.Sleep(1 * interval)
	if !cache.Exists("b") {
		t.Error("Expire error, cache b should exists")
	}

	time.Sleep(2 * interval)
	if cache.Exists("b") {
		t.Error("Expire error, cache b should be expired")
	}

	// test sliding expiration
	cache.PutSlid("c", 1, 2*interval)

	for i := 0; i < 10; i++ {
		if _, ok := cache.Get("c"); !ok {
			t.Error("Expire error, cache c should exists", i)
		}
		time.Sleep(1 * interval)
	}

	time.Sleep(2 * interval)
	if cache.Exists("c") {
		t.Error("Expire error, cache c should be expired")
	}

}

// time.now() take time
func BenchmarkGet(b *testing.B) {
	var key = "a"
	b.StopTimer()
	cache := NewMemoryCache(true)
	cache.PutP(key, key)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		cache.Get(key)
	}
}

func BenchmarkGetM(b *testing.B) {
	var key = "key"
	count := 1000 * 1000
	b.StopTimer()
	cache := NewMemoryCache(true)
	for i := 0; i < count; i++ {
		cache.PutP(strconv.Itoa(i)+key, i)
	}
	key = "1000key"
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(key)
	}
}

func BenchmarkMapGet(b *testing.B) {
	var key = "a"
	b.StopTimer()
	m := map[string]string{
		key: key,
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_, _ = m[key]
	}
}

func BenchmarkMapGetM(b *testing.B) {
	var key = "key"
	count := 1000 * 1000
	b.StopTimer()
	m := make(map[string]int)
	for i := 0; i < count; i++ {
		m[strconv.Itoa(i)+key] = i
	}
	key = "1000key"
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_, _ = m[key]
	}
}

func BenchmarkMapMutexGet(b *testing.B) {
	var key = "a"
	b.StopTimer()
	m := map[string]string{
		key: key,
	}
	var lock sync.Mutex
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		lock.Lock()
		_, _ = m[key]
		lock.Unlock()
	}
}

func BenchmarkMapMutexGetM(b *testing.B) {
	var key = "key"
	count := 1000 * 1000
	b.StopTimer()
	m := make(map[string]int)
	for i := 0; i < count; i++ {
		m[strconv.Itoa(i)+key] = i
	}
	key = "1000key"
	var lock sync.Mutex
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		lock.Lock()
		_, _ = m[key]
		lock.Unlock()
	}
}

func BenchmarkMapRWMutexGet(b *testing.B) {
	var key = "a"
	b.StopTimer()
	m := map[string]string{
		key: key,
	}
	var lock sync.RWMutex
	b.StartTimer()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		lock.RLock()
		_, _ = m[key]
		lock.RUnlock()
	}
}

func BenchmarkCacheSet(b *testing.B) {
	var key = "a"
	b.StopTimer()
	cache := NewMemoryCache(true)

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		cache.Put(key, key, 0, AbsoluteExpiration)
	}
}

func BenchmarkMutexMapSet(b *testing.B) {
	var key = "a"
	b.StopTimer()
	m := map[string]string{}
	var lock sync.RWMutex
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		lock.Lock()
		m[key] = key
		lock.Unlock()
	}
}
