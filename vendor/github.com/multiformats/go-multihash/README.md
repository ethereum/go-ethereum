# go-multihash

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-multiformats-blue.svg?style=flat-square)](https://github.com/multiformats/multiformats)
[![](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](https://webchat.freenode.net/?channels=%23ipfs)
[![](https://img.shields.io/badge/readme%20style-standard-brightgreen.svg?style=flat-square)](https://github.com/RichardLitt/standard-readme)
[![GoDoc](https://godoc.org/github.com/multiformats/go-multihash?status.svg)](https://godoc.org/github.com/multiformats/go-multihash)
[![Travis CI](https://img.shields.io/travis/multiformats/go-multihash.svg?style=flat-square&branch=master)](https://travis-ci.org/multiformats/go-multihash)
[![codecov.io](https://img.shields.io/codecov/c/github/multiformats/go-multihash.svg?style=flat-square&branch=master)](https://codecov.io/github/multiformats/go-multihash?branch=master)

> [multihash](https://github.com/multiformats/multihash) implementation in Go

## Table of Contents

- [Install](#install)
- [Usage](#usage)
- [Maintainers](#maintainers)
- [Contribute](#contribute)
- [License](#license)

## Install

`go-multihash` is a standard Go module which can be installed with:

```sh
go get github.com/multiformats/go-multihash
```

## Usage


### Example

This example takes a standard hex-encoded data and uses `EncodeName` to calculate the SHA1 multihash value for the buffer.

The resulting hex-encoded data corresponds to: `<hash function code><digest size><hash function output>`, which could be re-parsed
with `Multihash.FromHexString()`.


```go
package main

import (
	"encoding/hex"
	"fmt"

	"github.com/multiformats/go-multihash"
)

func main() {
	// ignores errors for simplicity.
	// don't do that at home.
	// Decode a SHA1 hash to a binary buffer
	buf, _ := hex.DecodeString("0beec7b5ea3f0fdbc95d0dd47f3c5bc275da8a33")

	// Create a new multihash with it.
	mHashBuf, _ := multihash.EncodeName(buf, "sha1")
	// Print the multihash as hex string
	fmt.Printf("hex: %s\n", hex.EncodeToString(mHashBuf))

	// Parse the binary multihash to a DecodedMultihash
	mHash, _ := multihash.Decode(mHashBuf)
	// Convert the sha1 value to hex string
	sha1hex := hex.EncodeToString(mHash.Digest)
	// Print all the information in the multihash
	fmt.Printf("obj: %v 0x%x %d %s\n", mHash.Name, mHash.Code, mHash.Length, sha1hex)
}
```

To run, copy to [example/foo.go](example/foo.go) and:

```
> cd example/
> go build
> ./example
hex: 11140beec7b5ea3f0fdbc95d0dd47f3c5bc275da8a33
obj: sha1 0x11 20 0beec7b5ea3f0fdbc95d0dd47f3c5bc275da8a33
```

## Contribute

Contributions welcome. Please check out [the issues](https://github.com/multiformats/go-multihash/issues).

Check out our [contributing document](https://github.com/multiformats/multiformats/blob/master/contributing.md) for more information on how we work, and about contributing in general. Please be aware that all interactions related to multiformats are subject to the IPFS [Code of Conduct](https://github.com/ipfs/community/blob/master/code-of-conduct.md).

Small note: If editing the README, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

[MIT](LICENSE) © 2014 Juan Batiz-Benet
