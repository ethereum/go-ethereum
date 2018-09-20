SipHash (Go)
============

[![Build Status](https://travis-ci.org/dchest/siphash.svg)](https://travis-ci.org/dchest/siphash)

Go implementation of SipHash-2-4, a fast short-input PRF created by
Jean-Philippe Aumasson and Daniel J. Bernstein (http://131002.net/siphash/).


## Installation

    $ go get github.com/dchest/siphash

## Usage

    import "github.com/dchest/siphash"

There are two ways to use this package.
The slower one is to use the standard hash.Hash64 interface:

    h := siphash.New(key)
    h.Write([]byte("Hello"))
    sum := h.Sum(nil) // returns 8-byte []byte

or

    sum64 := h.Sum64() // returns uint64

The faster one is to use Hash() function, which takes two uint64 parts of
16-byte key and a byte slice, and returns uint64 hash:

    sum64 := siphash.Hash(key0, key1, []byte("Hello"))

The keys and output are little-endian.


## Functions

### func Hash(k0, k1 uint64, p []byte) uint64

Hash returns the 64-bit SipHash-2-4 of the given byte slice with two
64-bit parts of 128-bit key: k0 and k1.

### func Hash128(k0, k1 uint64, p []byte) (uint64, uint64)

Hash128 returns the 128-bit SipHash-2-4 of the given byte slice with two
64-bit parts of 128-bit key: k0 and k1.

Note that 128-bit SipHash is considered experimental by SipHash authors at this time.

### func New(key []byte) hash.Hash64

New returns a new hash.Hash64 computing SipHash-2-4 with 16-byte key.

### func New128(key []byte) hash.Hash

New128 returns a new hash.Hash computing SipHash-2-4 with 16-byte key and 16-byte output.

Note that 16-byte output is considered experimental by SipHash authors at this time.


## Public domain dedication

Written by Dmitry Chestnykh and Damian Gryski.

To the extent possible under law, the authors have dedicated all copyright
and related and neighboring rights to this software to the public domain
worldwide. This software is distributed without any warranty.
http://creativecommons.org/publicdomain/zero/1.0/
