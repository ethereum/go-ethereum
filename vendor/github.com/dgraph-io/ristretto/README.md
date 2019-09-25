# Ristretto

[![GoDoc](https://img.shields.io/badge/api-reference-blue.svg)](https://godoc.org/github.com/dgraph-io/ristretto)
[![Go Report Card](https://img.shields.io/badge/go%20report-A%2B-green.svg)](https://goreportcard.com/report/github.com/dgraph-io/ristretto)

Ristretto is a fast, concurrent cache library using a [TinyLFU](https://arxiv.org/abs/1512.00727)
admission policy and Sampled LFU eviction policy.

The motivation to build Ristretto comes from the need for a contention-free
cache in [Dgraph][].

[Dgraph]: https://github.com/dgraph-io/dgraph

## Example

```go
package main

import (
	"fmt"
	"time"

	"github.com/dgraph-io/ristretto"
)

func main() {
	// create a cache instance
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000000 * 10,
		MaxCost:     1000000,
		BufferItems: 64,
	})
	if err != nil {
		panic(err)
	}

	// set a value
	cache.Set("key", "value", 1)

	// wait for value to pass through buffers
	time.Sleep(time.Second / 100)

	// get a value, given a key
	value, found := cache.Get("key")
	if !found {
		panic("missing value")
	}

	fmt.Println(value)

	// delete a value, given a key
	cache.Del("key")
}
```

### Benchmarks

The benchmarks can be found in https://github.com/dgraph-io/benchmarks/tree/master/cachebench/ristretto

### Hit Ratios

#### Search

This trace is described as "disk read accesses initiated by a large commercial
search engine in response to various web search requests."

![](https://raw.githubusercontent.com/karlmcguire/karlmcguire.com/master/docs/Hit%20Ratios%20-%20Search%20(ARC-S3).svg?sanitize=true)

#### Database

This trace is described as "a database server running at a commercial site
running an ERP application on top of a commercial database."

![](https://raw.githubusercontent.com/karlmcguire/karlmcguire.com/master/docs/Hit%20Ratios%20-%20Database%20(ARC-DS1).svg?sanitize=true)

#### Looping

This trace demonstrates a looping access pattern.

![](https://raw.githubusercontent.com/karlmcguire/karlmcguire.com/master/docs/Hit%20Ratios%20-%20Glimpse%20(LIRS-GLI).svg?sanitize=true)

#### CODASYL

This trace is described as "references to a CODASYL database for a one hour
period."

![](https://raw.githubusercontent.com/karlmcguire/karlmcguire.com/master/docs/Hit%20Ratios%20-%20CODASYL%20(ARC-OLTP).svg?sanitize=true)

### Throughput

All throughput benchmarks were ran on an Intel Core i7-8700K (3.7GHz) with 16gb
of RAM.

#### Mixed

![](https://raw.githubusercontent.com/karlmcguire/karlmcguire.com/master/docs/Throughput%20-%20Mixed.svg?sanitize=true)

#### Read

![](https://raw.githubusercontent.com/karlmcguire/karlmcguire.com/master/docs/Throughput%20-%20Read%20(Zipfian).svg?sanitize=true)

#### Write

![](https://raw.githubusercontent.com/karlmcguire/karlmcguire.com/master/docs/Throughput%20-%20Write%20(Zipfian).svg?sanitize=true)
