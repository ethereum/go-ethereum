# Set [![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://godoc.org/gopkg.in/fatih/set.v0) [![Build Status](http://img.shields.io/travis/fatih/set.svg?style=flat-square)](https://travis-ci.org/fatih/set)

Set is a basic and simple, hash-based, **Set** data structure implementation
in Go (Golang).

Set provides both threadsafe and non-threadsafe implementations of a generic
set data structure. The thread safety encompasses all operations on one set.
Operations on multiple sets are consistent in that the elements of each set
used was valid at exactly one point in time between the start and the end of
the operation. Because it's thread safe, you can use it concurrently with your
goroutines.

For usage see examples below or click on the godoc badge.

## Install and Usage

Install the package with:

```bash
go get gopkg.in/fatih/set.v0
```

Import it with:

```go
import "gopkg.in/fatih/set.v0"
```

and use `set` as the package name inside the code.

## Examples

#### Initialization of a new Set

```go

// create a set with zero items
s := set.New()
s := set.NewNonTS() // non thread-safe version

// ... or with some initial values
s := set.New("istanbul", "frankfurt", 30.123, "san francisco", 1234)
s := set.NewNonTS("kenya", "ethiopia", "sumatra")

```

#### Basic Operations

```go
// add items
s.Add("istanbul")
s.Add("istanbul") // nothing happens if you add duplicate item

// add multiple items
s.Add("ankara", "san francisco", 3.14)

// remove item
s.Remove("frankfurt")
s.Remove("frankfurt") // nothing happes if you remove a nonexisting item

// remove multiple items
s.Remove("barcelona", 3.14, "ankara")

// removes an arbitary item and return it
item := s.Pop()

// create a new copy
other := s.Copy()

// remove all items
s.Clear()

// number of items in the set
len := s.Size()

// return a list of items
items := s.List()

// string representation of set
fmt.Printf("set is %s", s.String())

```

#### Check Operations

```go
// check for set emptiness, returns true if set is empty
s.IsEmpty()

// check for a single item exist
s.Has("istanbul")

// ... or for multiple items. This will return true if all of the items exist.
s.Has("istanbul", "san francisco", 3.14)

// create two sets for the following checks...
s := s.New("1", "2", "3", "4", "5")
t := s.New("1", "2", "3")


// check if they are the same
if !s.IsEqual(t) {
    fmt.Println("s is not equal to t")
}

// if s contains all elements of t
if s.IsSubset(t) {
	fmt.Println("t is a subset of s")
}

// ... or if s is a superset of t
if t.IsSuperset(s) {
	fmt.Println("s is a superset of t")
}


```

#### Set Operations


```go
// let us initialize two sets with some values
a := set.New("ankara", "berlin", "san francisco")
b := set.New("frankfurt", "berlin")

// creates a new set with the items in a and b combined.
// [frankfurt, berlin, ankara, san francisco]
c := set.Union(a, b)

// contains items which is in both a and b
// [berlin]
c := set.Intersection(a, b)

// contains items which are in a but not in b
// [ankara, san francisco]
c := set.Difference(a, b)

// contains items which are in one of either, but not in both.
// [frankfurt, ankara, san francisco]
c := set.SymmetricDifference(a, b)

```

```go
// like Union but saves the result back into a.
a.Merge(b)

// removes the set items which are in b from a and saves the result back into a.
a.Separate(b)

```

#### Multiple Set Operations

```go
a := set.New("1", "3", "4", "5")
b := set.New("2", "3", "4", "5")
c := set.New("4", "5", "6", "7")

// creates a new set with items in a, b and c
// [1 2 3 4 5 6 7]
u := set.Union(a, b, c)

// creates a new set with items in a but not in b and c
// [1]
u := set.Difference(a, b, c)

// creates a new set with items that are common to a, b and c
// [5]
u := set.Intersection(a, b, c)
```

#### Helper methods

The Slice functions below are a convenient way to extract or convert your Set data
into basic data types.


```go
// create a set of mixed types
s := set.New("ankara", "5", "8", "san francisco", 13, 21)


// convert s into a slice of strings (type is []string)
// [ankara 5 8 san francisco]
t := set.StringSlice(s)


// u contains a slice of ints (type is []int)
// [13, 21]
u := set.IntSlice(s)

```

#### Concurrent safe usage

Below is an example of a concurrent way that uses set. We call ten functions
concurrently and wait until they are finished. It basically creates a new
string for each goroutine and adds it to our set.

```go
package main

import (
	"fmt"
	"github.com/fatih/set"
	"strconv"
	"sync"
)

func main() {
	var wg sync.WaitGroup // this is just for waiting until all goroutines finish

	// Initialize our thread safe Set
	s := set.New()

	// Add items concurrently (item1, item2, and so on)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			item := "item" + strconv.Itoa(i)
			fmt.Println("adding", item)
			s.Add(item)
			wg.Done()
		}(i)
	}

	// Wait until all concurrent calls finished and print our set
	wg.Wait()
	fmt.Println(s)
}
```

## Credits

 * [Fatih Arslan](https://github.com/fatih)
 * [Arne Hormann](https://github.com/arnehormann)
 * [Sam Boyer](https://github.com/sdboyer)
 * [Ralph Loizzo](https://github.com/friartech)

## License

The MIT License (MIT) - see LICENSE.md for more details

