# Boom Filters
[![Build Status](https://travis-ci.org/tylertreat/BoomFilters.svg?branch=master)](https://travis-ci.org/tylertreat/BoomFilters) [![GoDoc](https://godoc.org/github.com/tylertreat/BoomFilters?status.png)](https://godoc.org/github.com/tylertreat/BoomFilters)

**Boom Filters** are probabilistic data structures for [processing continuous, unbounded streams](http://www.bravenewgeek.com/stream-processing-and-probabilistic-methods/). This includes **Stable Bloom Filters**, **Scalable Bloom Filters**, **Counting Bloom Filters**, **Inverse Bloom Filters**, **Cuckoo Filters**, several variants of **traditional Bloom filters**, **HyperLogLog**, **Count-Min Sketch**, and **MinHash**.

Classic Bloom filters generally require a priori knowledge of the data set in order to allocate an appropriately sized bit array. This works well for offline processing, but online processing typically involves unbounded data streams. With enough data, a traditional Bloom filter "fills up", after which it has a false-positive probability of 1.

Boom Filters are useful for situations where the size of the data set isn't known ahead of time. For example, a Stable Bloom Filter can be used to deduplicate events from an unbounded event stream with a specified upper bound on false positives and minimal false negatives. Alternatively, an Inverse Bloom Filter is ideal for deduplicating a stream where duplicate events are relatively close together. This results in no false positives and, depending on how close together duplicates are, a small probability of false negatives. Scalable Bloom Filters place a tight upper bound on false positives while avoiding false negatives but require allocating memory proportional to the size of the data set. Counting Bloom Filters and Cuckoo Filters are useful for cases which require adding and removing elements to and from a set.

For large or unbounded data sets, calculating the exact cardinality is impractical. HyperLogLog uses a fraction of the memory while providing an accurate approximation. Similarly, Count-Min Sketch provides an efficient way to estimate event frequency for data streams, while Top-K tracks the top-k most frequent elements.

MinHash is a probabilistic algorithm to approximate the similarity between two sets. This can be used to cluster or compare documents by splitting the corpus into a bag of words.

## Installation 

```
$ go get github.com/tylertreat/BoomFilters
```

## Stable Bloom Filter

This is an implementation of Stable Bloom Filters as described by Deng and Rafiei in [Approximately Detecting Duplicates for Streaming Data using Stable Bloom Filters](http://webdocs.cs.ualberta.ca/~drafiei/papers/DupDet06Sigmod.pdf).

A Stable Bloom Filter (SBF) continuously evicts stale information so that it has room for more recent elements. Like traditional Bloom filters, an SBF has a non-zero probability of false positives, which is controlled by several parameters. Unlike the classic Bloom filter, an SBF has a tight upper bound on the rate of false positives while introducing a non-zero rate of false negatives. The false-positive rate of a classic Bloom filter eventually reaches 1, after which all queries result in a false positive. The stable-point property of an SBF means the false-positive rate asymptotically approaches a configurable fixed constant. A classic Bloom filter is actually a special case of SBF where the eviction rate is zero and the cell size is one, so this provides support for them as well (in addition to bitset-based Bloom filters).

Stable Bloom Filters are useful for cases where the size of the data set isn't known a priori and memory is bounded. For example, an SBF can be used to deduplicate events from an unbounded event stream with a specified upper bound on false positives and minimal false negatives.

### Usage

```go
package main

import (
    "fmt"
    "github.com/tylertreat/BoomFilters"
)

func main() {
    sbf := boom.NewDefaultStableBloomFilter(10000, 0.01)
    fmt.Println("stable point", sbf.StablePoint())
    
    sbf.Add([]byte(`a`))
    if sbf.Test([]byte(`a`)) {
        fmt.Println("contains a")
    }
    
    if !sbf.TestAndAdd([]byte(`b`)) {
        fmt.Println("doesn't contain b")
    }
    
    if sbf.Test([]byte(`b`)) {
        fmt.Println("now it contains b!")
    }
    
    // Restore to initial state.
    sbf.Reset()
}
```

## Scalable Bloom Filter

This is an implementation of a Scalable Bloom Filter as described by Almeida, Baquero, Preguica, and Hutchison in [Scalable Bloom Filters](http://gsd.di.uminho.pt/members/cbm/ps/dbloom.pdf).

A Scalable Bloom Filter (SBF) dynamically adapts to the size of the data set while enforcing a tight upper bound on the rate of false positives and a false-negative probability of zero. This works by adding Bloom filters with geometrically decreasing false-positive rates as filters become full. A tightening ratio, r, controls the filter growth. The compounded probability over the whole series converges to a target value, even accounting for an infinite series.

Scalable Bloom Filters are useful for cases where the size of the data set isn't known a priori and memory constraints aren't of particular concern. For situations where memory is bounded, consider using Inverse or Stable Bloom Filters.

The core parts of this implementation were originally written by Jian Zhen as discussed in [Benchmarking Bloom Filters and Hash Functions in Go](http://zhen.org/blog/benchmarking-bloom-filters-and-hash-functions-in-go/).

### Usage

```go
package main

import (
    "fmt"
    "github.com/tylertreat/BoomFilters"
)

func main() {
    sbf := boom.NewDefaultScalableBloomFilter(0.01)
    
    sbf.Add([]byte(`a`))
    if sbf.Test([]byte(`a`)) {
        fmt.Println("contains a")
    }
    
    if !sbf.TestAndAdd([]byte(`b`)) {
        fmt.Println("doesn't contain b")
    }
    
    if sbf.Test([]byte(`b`)) {
        fmt.Println("now it contains b!")
    }
    
    // Restore to initial state.
    sbf.Reset()
}
```

## Inverse Bloom Filter

An Inverse Bloom Filter, or "the opposite of a Bloom filter", is a concurrent, probabilistic data structure used to test whether an item has been observed or not. This implementation, [originally described and written by Jeff Hodges](http://www.somethingsimilar.com/2012/05/21/the-opposite-of-a-bloom-filter/), replaces the use of MD5 hashing with a non-cryptographic FNV-1 function.

The Inverse Bloom Filter may report a false negative but can never report a false positive. That is, it may report that an item has not been seen when it actually has, but it will never report an item as seen which it hasn't come across. This behaves in a similar manner to a fixed-size hashmap which does not handle conflicts.

This structure is particularly well-suited to streams in which duplicates are relatively close together. It uses a CAS-style approach, which makes it thread-safe.

### Usage

```go
package main

import (
    "fmt"
    "github.com/tylertreat/BoomFilters"
)

func main() {
    ibf := boom.NewInverseBloomFilter(10000)
    
    ibf.Add([]byte(`a`))
    if ibf.Test([]byte(`a`)) {
        fmt.Println("contains a")
    }
    
    if !ibf.TestAndAdd([]byte(`b`)) {
        fmt.Println("doesn't contain b")
    }
    
    if ibf.Test([]byte(`b`)) {
        fmt.Println("now it contains b!")
    }
}
```

## Counting Bloom Filter

This is an implementation of a Counting Bloom Filter as described by Fan, Cao, Almeida, and Broder in [Summary Cache: A Scalable Wide-Area Web Cache Sharing Protocol](http://pages.cs.wisc.edu/~jussara/papers/00ton.pdf).

A Counting Bloom Filter (CBF) provides a way to remove elements by using an array of n-bit buckets. When an element is added, the respective buckets are incremented. To remove an element, the respective buckets are decremented. A query checks that each of the respective buckets are non-zero. Because CBFs allow elements to be removed, they introduce a non-zero probability of false negatives in addition to the possibility of false positives.

Counting Bloom Filters are useful for cases where elements are both added and removed from the data set. Since they use n-bit buckets, CBFs use roughly n-times more memory than traditional Bloom filters.

See Deletable Bloom Filter for an alternative which avoids false negatives.

### Usage

```go
package main

import (
    "fmt"
    "github.com/tylertreat/BoomFilters"
)

func main() {
    bf := boom.NewDefaultCountingBloomFilter(1000, 0.01)
    
    bf.Add([]byte(`a`))
    if bf.Test([]byte(`a`)) {
        fmt.Println("contains a")
    }
    
    if !bf.TestAndAdd([]byte(`b`)) {
        fmt.Println("doesn't contain b")
    }
    
    if bf.TestAndRemove([]byte(`b`)) {
        fmt.Println("removed b")
    }
    
    // Restore to initial state.
    bf.Reset()
}
```

## Cuckoo Filter

This is an implementation of a Cuckoo Filter as described by Andersen, Kaminsky, and Mitzenmacher in [Cuckoo Filter: Practically Better Than Bloom](http://www.pdl.cmu.edu/PDL-FTP/FS/cuckoo-conext2014.pdf). The Cuckoo Filter is similar to the Counting Bloom Filter in that it supports adding and removing elements, but it does so in a way that doesn't significantly degrade space and performance.

It works by using a cuckoo hashing scheme for inserting items. Instead of storing the elements themselves, it stores their fingerprints which also allows for item removal without false negatives (if you don't attempt to remove an item not contained in the filter).

For applications that store many items and target moderately low false-positive rates, cuckoo filters have lower space overhead than space-optimized Bloom filters.

### Usage

```go
package main

import (
    "fmt"
    "github.com/tylertreat/BoomFilters"
)

func main() {
    cf := boom.NewCuckooFilter(1000, 0.01)
    
    cf.Add([]byte(`a`))
    if cf.Test([]byte(`a`)) {
        fmt.Println("contains a")
    }
    
    if contains, _ := cf.TestAndAdd([]byte(`b`)); !contains {
        fmt.Println("doesn't contain b")
    }
    
    if cf.TestAndRemove([]byte(`b`)) {
        fmt.Println("removed b")
    }
    
    // Restore to initial state.
    cf.Reset()
}
```

## Classic Bloom Filter

A classic Bloom filter is a special case of a Stable Bloom Filter whose eviction rate is zero and cell size is one. We call this special case an Unstable Bloom Filter. Because cells require more memory overhead, this package also provides two bitset-based Bloom filter variations. The first variation is the traditional implementation consisting of a single bit array. The second implementation is a partitioned approach which uniformly distributes the probability of false positives across all elements.

Bloom filters have a limited capacity, depending on the configured size. Once all bits are set, the probability of a false positive is 1. However, traditional Bloom filters cannot return a false negative.

A Bloom filter is ideal for cases where the data set is known a priori because the false-positive rate can be configured by the size and number of hash functions.

### Usage

```go
package main

import (
    "fmt"
    "github.com/tylertreat/BoomFilters"
)

func main() {
    // We could also use boom.NewUnstableBloomFilter or boom.NewPartitionedBloomFilter.
    bf := boom.NewBloomFilter(1000, 0.01)
    
    bf.Add([]byte(`a`))
    if bf.Test([]byte(`a`)) {
        fmt.Println("contains a")
    }
    
    if !bf.TestAndAdd([]byte(`b`)) {
        fmt.Println("doesn't contain b")
    }
    
    if bf.Test([]byte(`b`)) {
        fmt.Println("now it contains b!")
    }
    
    // Restore to initial state.
    bf.Reset()
}
```

## Count-Min Sketch

This is an implementation of a Count-Min Sketch as described by Cormode and Muthukrishnan in [An Improved Data Stream Summary: The Count-Min Sketch and its Applications](http://dimacs.rutgers.edu/~graham/pubs/papers/cm-full.pdf).

A Count-Min Sketch (CMS) is a probabilistic data structure which approximates the frequency of events in a data stream. Unlike a hash map, a CMS uses sub-linear space at the expense of a configurable error factor. Similar to Counting Bloom filters, items are hashed to a series of buckets, which increment a counter. The frequency of an item is estimated by taking the minimum of each of the item's respective counter values.

Count-Min Sketches are useful for counting the frequency of events in massive data sets or unbounded streams online. In these situations, storing the entire data set or allocating counters for every event in memory is impractical. It may be possible for offline processing, but real-time processing requires fast, space-efficient solutions like the CMS. For approximating set cardinality, refer to the HyperLogLog.

### Usage

```go
package main

import (
    "fmt"
    "github.com/tylertreat/BoomFilters"
)

func main() {
    cms := boom.NewCountMinSketch(0.001, 0.99)
    
    cms.Add([]byte(`alice`)).Add([]byte(`bob`)).Add([]byte(`bob`)).Add([]byte(`frank`))
    fmt.Println("frequency of alice", cms.Count([]byte(`alice`)))
    fmt.Println("frequency of bob", cms.Count([]byte(`bob`)))
    fmt.Println("frequency of frank", cms.Count([]byte(`frank`)))
    

    // Serialization example
    buf := new(bytes.Buffer)
    n, err := cms.WriteDataTo(buf)
    if err != nil {
       fmt.Println(err, n)
    }

    // Restore to initial state.
    cms.Reset()

    newCMS := boom.NewCountMinSketch(0.001, 0.99)
    n, err = newCMS.ReadDataFrom(buf)
    if err != nil {
       fmt.Println(err, n)
    }

    fmt.Println("frequency of frank", newCMS.Count([]byte(`frank`)))

   
}
```

## Top-K

Top-K uses a Count-Min Sketch and min-heap to track the top-k most frequent elements in a stream.

### Usage

```go
package main

import (
    "fmt"
    "github.com/tylertreat/BoomFilters"
)

func main() {
	topk := NewTopK(0.001, 0.99, 5)

	topk.Add([]byte(`bob`)).Add([]byte(`bob`)).Add([]byte(`bob`))
	topk.Add([]byte(`tyler`)).Add([]byte(`tyler`)).Add([]byte(`tyler`)).Add([]byte(`tyler`))
	topk.Add([]byte(`fred`))
	topk.Add([]byte(`alice`)).Add([]byte(`alice`)).Add([]byte(`alice`)).Add([]byte(`alice`))
	topk.Add([]byte(`james`))
	topk.Add([]byte(`fred`))
	topk.Add([]byte(`sara`)).Add([]byte(`sara`))
	topk.Add([]byte(`bill`))

	for i, element := range topk.Elements() {
		fmt.Println(i, string(element.Data), element.Freq)
	}
	
	// Restore to initial state.
	topk.Reset()
}
```

## HyperLogLog

This is an implementation of HyperLogLog as described by Flajolet, Fusy, Gandouet, and Meunier in [HyperLogLog: the analysis of a near-optimal cardinality estimation algorithm](http://algo.inria.fr/flajolet/Publications/FlFuGaMe07.pdf).

HyperLogLog is a probabilistic algorithm which approximates the number of distinct elements in a multiset. It works by hashing values and calculating the maximum number of leading zeros in the binary representation of each hash. If the maximum number of leading zeros is n, the estimated number of distinct elements in the set is 2^n. To minimize variance, the multiset is split into a configurable number of registers, the maximum number of leading zeros is calculated in the numbers in each register, and a harmonic mean is used to combine the estimates.

For large or unbounded data sets, calculating the exact cardinality is impractical. HyperLogLog uses a fraction of the memory while providing an accurate approximation.

This implementation was [originally written by Eric Lesh](https://github.com/eclesh/hyperloglog). Some small changes and additions have been made, including a way to construct a HyperLogLog optimized for a particular relative accuracy and adding FNV hashing. For counting element frequency, refer to the Count-Min Sketch.

### Usage

```go
package main

import (
    "fmt"
    "github.com/tylertreat/BoomFilters"
)

func main() {
    hll, err := boom.NewDefaultHyperLogLog(0.1)
    if err != nil {
        panic(err)
    }
    
    hll.Add([]byte(`alice`)).Add([]byte(`bob`)).Add([]byte(`bob`)).Add([]byte(`frank`))
    fmt.Println("count", hll.Count())

    // Serialization example
    buf := new(bytes.Buffer)
    _, err := hll.WriteDataTo(buf)
    if err != nil {
       fmt.Println(err)
    }
    
    // Restore to initial state.
    hll.Reset()

    newHll, err := boom.NewDefaultHyperLogLog(0.1)
    if err != nil {
       fmt.Println(err)
    }

    _, err := newHll.ReadDataFrom(buf)
    if err != nil {
       fmt.Println(err)
    }
    fmt.Println("count", newHll.Count())

}
```

## MinHash

This is a variation of the technique for estimating similarity between two sets as presented by Broder in [On the resemblance and containment of documents](http://gatekeeper.dec.com/ftp/pub/dec/SRC/publications/broder/positano-final-wpnums.pdf).

MinHash is a probabilistic algorithm which can be used to cluster or compare documents by splitting the corpus into a bag of words. MinHash returns the approximated similarity ratio of the two bags. The similarity is less accurate for very small bags of words.

### Usage

```go
package main

import (
    "fmt"
    "github.com/tylertreat/BoomFilters"
)

func main() {
    bag1 := []string{"bill", "alice", "frank", "bob", "sara", "tyler", "james"}
	bag2 := []string{"bill", "alice", "frank", "bob", "sara"}
	
	fmt.Println("similarity", boom.MinHash(bag1, bag2))
}
```

## References

- [Approximately Detecting Duplicates for Streaming Data using Stable Bloom Filters](http://webdocs.cs.ualberta.ca/~drafiei/papers/DupDet06Sigmod.pdf)
- [Scalable Bloom Filters](http://gsd.di.uminho.pt/members/cbm/ps/dbloom.pdf)
- [The Opposite of a Bloom Filter](http://www.somethingsimilar.com/2012/05/21/the-opposite-of-a-bloom-filter/)
- [Benchmarking Bloom Filters and Hash Functions in Go](http://zhen.org/blog/benchmarking-bloom-filters-and-hash-functions-in-go/)
- [Summary Cache: A Scalable Wide-Area Web Cache Sharing Protocol](http://pages.cs.wisc.edu/~jussara/papers/00ton.pdf)
- [An Improved Data Stream Summary: The Count-Min Sketch and its Applications](http://dimacs.rutgers.edu/~graham/pubs/papers/cm-full.pdf)
- [HyperLogLog: the analysis of a near-optimal cardinality estimation algorithm](http://algo.inria.fr/flajolet/Publications/FlFuGaMe07.pdf)
- [Package hyperloglog](https://github.com/eclesh/hyperloglog)
- [On the resemblance and containment of documents](http://gatekeeper.dec.com/ftp/pub/dec/SRC/publications/broder/positano-final-wpnums.pdf)
- [Cuckoo Filter: Practically Better Than Bloom](http://www.pdl.cmu.edu/PDL-FTP/FS/cuckoo-conext2014.pdf)
