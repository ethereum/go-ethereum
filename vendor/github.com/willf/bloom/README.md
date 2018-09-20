Bloom filters
-------------

[![Master Build Status](https://secure.travis-ci.org/willf/bloom.png?branch=master)](https://travis-ci.org/willf/bloom?branch=master)
[![Coverage Status](https://coveralls.io/repos/github/willf/bloom/badge.svg?branch=master)](https://coveralls.io/github/willf/bloom?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/willf/bloom)](https://goreportcard.com/report/github.com/willf/bloom)
[![GoDoc](https://godoc.org/github.com/willf/bloom?status.svg)](http://godoc.org/github.com/willf/bloom)

A Bloom filter is a representation of a set of _n_ items, where the main
requirement is to make membership queries; _i.e._, whether an item is a
member of a set.

A Bloom filter has two parameters: _m_, a maximum size (typically a reasonably large multiple of the cardinality of the set to represent) and _k_, the number of hashing functions on elements of the set. (The actual hashing functions are important, too, but this is not a parameter for this implementation). A Bloom filter is backed by a [BitSet](https://github.com/willf/bitset); a key is represented in the filter by setting the bits at each value of the  hashing functions (modulo _m_). Set membership is done by _testing_ whether the bits at each value of the hashing functions (again, modulo _m_) are set. If so, the item is in the set. If the item is actually in the set, a Bloom filter will never fail (the true positive rate is 1.0); but it is susceptible to false positives. The art is to choose _k_ and _m_ correctly.

In this implementation, the hashing functions used is [murmurhash](github.com/spaolacci/murmur3), a non-cryptographic hashing function.

This implementation accepts keys for setting and testing as `[]byte`. Thus, to
add a string item, `"Love"`:

    n := uint(1000)
    filter := bloom.New(20*n, 5) // load of 20, 5 keys
    filter.Add([]byte("Love"))

Similarly, to test if `"Love"` is in bloom:

    if filter.Test([]byte("Love"))

For numeric data, I recommend that you look into the encoding/binary library. But, for example, to add a `uint32` to the filter:

    i := uint32(100)
    n1 := make([]byte, 4)
    binary.BigEndian.PutUint32(n1, i)
    filter.Add(n1)

Finally, there is a method to estimate the false positive rate of a particular
bloom filter for a set of size _n_:

    if filter.EstimateFalsePositiveRate(1000) > 0.001

Given the particular hashing scheme, it's best to be empirical about this. Note
that estimating the FP rate will clear the Bloom filter.

Discussion here: [Bloom filter](https://groups.google.com/d/topic/golang-nuts/6MktecKi1bE/discussion)

Godoc documentation: https://godoc.org/github.com/willf/bloom

## Installation

```bash
go get -u github.com/willf/bloom
```

## Contributing

If you wish to contribute to this project, please branch and issue a pull request against master ("[GitHub Flow](https://guides.github.com/introduction/flow/)")

This project include a Makefile that allows you to test and build the project with simple commands.
To see all available options:
```bash
make help
```

## Running all tests

Before committing the code, please check if it passes all tests using (note: this will install some dependencies):
```bash
make deps
make qa
```
