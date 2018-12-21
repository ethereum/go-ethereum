# BigCache [![Build Status](https://travis-ci.org/allegro/bigcache.svg?branch=master)](https://travis-ci.org/allegro/bigcache)&nbsp;[![Coverage Status](https://coveralls.io/repos/github/allegro/bigcache/badge.svg?branch=master)](https://coveralls.io/github/allegro/bigcache?branch=master)&nbsp;[![GoDoc](https://godoc.org/github.com/allegro/bigcache?status.svg)](https://godoc.org/github.com/allegro/bigcache)&nbsp;[![Go Report Card](https://goreportcard.com/badge/github.com/allegro/bigcache)](https://goreportcard.com/report/github.com/allegro/bigcache)

Fast, concurrent, evicting in-memory cache written to keep big number of entries without impact on performance.
BigCache keeps entries on heap but omits GC for them. To achieve that operations on bytes arrays take place,
therefore entries (de)serialization in front of the cache will be needed in most use cases.

## Usage

### Simple initialization

```go
import "github.com/allegro/bigcache"

cache, _ := bigcache.NewBigCache(bigcache.DefaultConfig(10 * time.Minute))

cache.Set("my-unique-key", []byte("value"))

entry, _ := cache.Get("my-unique-key")
fmt.Println(string(entry))
```

### Custom initialization

When cache load can be predicted in advance then it is better to use custom initialization because additional memory
allocation can be avoided in that way.

```go
import (
	"log"

	"github.com/allegro/bigcache"
)

config := bigcache.Config {
		// number of shards (must be a power of 2)
		Shards: 1024,
		// time after which entry can be evicted
		LifeWindow: 10 * time.Minute,
		// rps * lifeWindow, used only in initial memory allocation
		MaxEntriesInWindow: 1000 * 10 * 60,
		// max entry size in bytes, used only in initial memory allocation
		MaxEntrySize: 500,
		// prints information about additional memory allocation
		Verbose: true,
		// cache will not allocate more memory than this limit, value in MB
		// if value is reached then the oldest entries can be overridden for the new ones
		// 0 value means no size limit
		HardMaxCacheSize: 8192,
		// callback fired when the oldest entry is removed because of its expiration time or no space left
		// for the new entry, or because delete was called. A bitmask representing the reason will be returned.
		// Default value is nil which means no callback and it prevents from unwrapping the oldest entry.
		OnRemove: nil,
		// OnRemoveWithReason is a callback fired when the oldest entry is removed because of its expiration time or no space left
		// for the new entry, or because delete was called. A constant representing the reason will be passed through.
		// Default value is nil which means no callback and it prevents from unwrapping the oldest entry.
		// Ignored if OnRemove is specified.
		OnRemoveWithReason: nil,
	}

cache, initErr := bigcache.NewBigCache(config)
if initErr != nil {
	log.Fatal(initErr)
}

cache.Set("my-unique-key", []byte("value"))

if entry, err := cache.Get("my-unique-key"); err == nil {
	fmt.Println(string(entry))
}
```

## Benchmarks

Three caches were compared: bigcache, [freecache](https://github.com/coocood/freecache) and map.
Benchmark tests were made using an i7-6700K with 32GB of RAM on Windows 10.

### Writes and reads

```bash
cd caches_bench; go test -bench=. -benchtime=10s ./... -timeout 30m

BenchmarkMapSet-8                        3000000               569 ns/op             202 B/op          3 allocs/op
BenchmarkConcurrentMapSet-8              1000000              1592 ns/op             347 B/op          8 allocs/op
BenchmarkFreeCacheSet-8                  3000000               775 ns/op             355 B/op          2 allocs/op
BenchmarkBigCacheSet-8                   3000000               640 ns/op             303 B/op          2 allocs/op
BenchmarkMapGet-8                        5000000               407 ns/op              24 B/op          1 allocs/op
BenchmarkConcurrentMapGet-8              3000000               558 ns/op              24 B/op          2 allocs/op
BenchmarkFreeCacheGet-8                  2000000               682 ns/op             136 B/op          2 allocs/op
BenchmarkBigCacheGet-8                   3000000               512 ns/op             152 B/op          4 allocs/op
BenchmarkBigCacheSetParallel-8          10000000               225 ns/op             313 B/op          3 allocs/op
BenchmarkFreeCacheSetParallel-8         10000000               218 ns/op             341 B/op          3 allocs/op
BenchmarkConcurrentMapSetParallel-8      5000000               318 ns/op             200 B/op          6 allocs/op
BenchmarkBigCacheGetParallel-8          20000000               178 ns/op             152 B/op          4 allocs/op
BenchmarkFreeCacheGetParallel-8         20000000               295 ns/op             136 B/op          3 allocs/op
BenchmarkConcurrentMapGetParallel-8     10000000               237 ns/op              24 B/op          2 allocs/op
```

Writes and reads in bigcache are faster than in freecache.
Writes to map are the slowest.

### GC pause time

```bash
cd caches_bench; go run caches_gc_overhead_comparison.go

Number of entries:  20000000
GC pause for bigcache:  5.8658ms
GC pause for freecache:  32.4341ms
GC pause for map:  52.9661ms
```

Test shows how long are the GC pauses for caches filled with 20mln of entries.
Bigcache and freecache have very similar GC pause time.
It is clear that both reduce GC overhead in contrast to map
which GC pause time took more than 10 seconds.

## How it works

BigCache relies on optimization presented in 1.5 version of Go ([issue-9477](https://github.com/golang/go/issues/9477)).
This optimization states that if map without pointers in keys and values is used then GC will omit its content.
Therefore BigCache uses `map[uint64]uint32` where keys are hashed and values are offsets of entries.

Entries are kept in bytes array, to omit GC again.
Bytes array size can grow to gigabytes without impact on performance
because GC will only see single pointer to it.

## Bigcache vs Freecache

Both caches provide the same core features but they reduce GC overhead in different ways.
Bigcache relies on `map[uint64]uint32`, freecache implements its own mapping built on
slices to reduce number of pointers.

Results from benchmark tests are presented above.
One of the advantage of bigcache over freecache is that you don’t need to know
the size of the cache in advance, because when bigcache is full,
it can allocate additional memory for new entries instead of
overwriting existing ones as freecache does currently.
However hard max size in bigcache also can be set, check [HardMaxCacheSize](https://godoc.org/github.com/allegro/bigcache#Config).

## HTTP Server

This package also includes an easily deployable HTTP implementation of BigCache, which can be found in the [server](/server) package.

## More

Bigcache genesis is described in allegro.tech blog post: [writing a very fast cache service in Go](http://allegro.tech/2016/03/writing-fast-cache-service-in-go.html)

## License

BigCache is released under the Apache 2.0 license (see [LICENSE](LICENSE))
