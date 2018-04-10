go-maddr-filter
==================

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](http://libp2p.io/)
[![](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](http://webchat.freenode.net/?channels=%23ipfs)
[![Coverage Status](https://coveralls.io/repos/github/libp2p/go-maddr-filter/badge.svg?branch=master)](https://coveralls.io/github/libp2p/go-maddr-filter?branch=master)
[![Travis CI](https://travis-ci.org/libp2p/go-maddr-filter.svg?branch=master)](https://travis-ci.org/libp2p/go-maddr-filter)

> A library to perform filtering of multiaddrs.


## Table of Contents

- [Install](#install)
- [Usage](#usage)
- [API](#api)
- [Contribute](#contribute)
- [License](#license)

## Install

```sh
make install
```

## Examples

```go
// make a new filterset
f := NewFilters()

// filter out addresses on the 192.168 subnet
_, ipnet, _ := net.ParseCIDR("192.168.0.0/16")
f.AddDialFilter(ipnet)


// check if an address is blocked
lanaddr, _ := ma.NewMultiaddr("/ip4/192.168.0.17/tcp/4050")
fmt.Println(f.AddrBlocked(lanaddr))
```

## Contribute

PRs are welcome!

Small note: If editing the Readme, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

MIT © Jeromy Johnson
