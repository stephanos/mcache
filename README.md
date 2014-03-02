mcache [![Build Status](https://secure.travis-ci.org/101loops/mcache.png)](https://travis-ci.org/101loops/mcache)
=========

This Go package provides an in-memory key/value cache:
 - thread safe
 - support expiration after absolute or sliding span of time
 - support CAS Update

### Installation
`go get github.com/101loops/mcache`

### Documentation
[godoc.org](http://godoc.org/github.com/101loops/mcache)

### Credit
Based on the source code from https://github.com/sdming/mcache.

### License
Apache License 2.0 (see LICENSE).

### Usage

#### Getting Started

	cache := mcache.NewMCache()
	cache.SetAbs("float", 3.14, time.Second)
	cache.Update("float", 1.1)
	fmt.Println(cache.Get("float"))
	<-time.After(time.Second)
	fmt.Println(cache.Get("float"))


#### Example

	// new cache
	cache := mcache.NewMCache()

	// set a cache entry with expire time span and kind
	cache.Set("Set", "Set", time.Minute, mcache.AbsoluteExpiration)
	fmt.Println(cache.Get("Set"))

	// set a cache entry with very long expiration time
	cache.SetP("SetP", "SetP")
	fmt.Println(cache.Get("SetP"))

	// set a cache entry with AbsoluteExpiration
	cache.SetAbs("SetAbs", "SetAbs", time.Second)
	fmt.Println(cache.Get("SetAbs"))
	<-time.After(2 * time.Second)
	fmt.Println(cache.Get("SetAbs"))

	// test a cache entry with SlidingExpiration
	cache.SetSlid("SetSlid", "SetSlid", time.Second)
	for i := 0; i < 10; i++ {
		fmt.Println(cache.Get("SetSlid"))
		time.Sleep(time.Second / 2)
	}
	<-time.After(time.Second)
	fmt.Println(cache.Get("SetSlid"))

	// count of cache, include expired
	fmt.Println(cache.Count())

	// all keys
	fmt.Println(cache.Keys())

	// key exists?
	cache.SetP("key", "key")
	fmt.Println(cache.Exists("key"))

	// delete
	cache.Delete("key")
	fmt.Println(cache.Exists("key"))

	// stat
	fmt.Println(cache.Stat())

	// Add
	fmt.Println(cache.Add("Add", "Add", time.Minute, AbsoluteExpiration))
	fmt.Println(cache.Add("Add", "Add", time.Minute, AbsoluteExpiration))



#### Cas Update

mcache increase version when update a cache entry, GetV can return this version.

	cache := mcache.NewMCache()

	key := "key"
	i := 0
	cache.SetP(key, key+strconv.Itoa(i))
	cache.Update(key, key+strconv.Itoa(i))
	fmt.Println(cache.GetV(key))

	fmt.Println(cache.UpdateV(key, i, key+strconv.Itoa(i)))
	i++
	fmt.Println(cache.UpdateV(key, i, key+strconv.Itoa(i)))


#### Options

You can set TickInterval to adjust the interval of expiration check

#### Benchmark

	go test -bench .*
