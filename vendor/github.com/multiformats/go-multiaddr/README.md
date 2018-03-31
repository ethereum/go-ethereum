# go-multiaddr

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-multiformats-blue.svg?style=flat-square)](https://github.com/multiformats/multiformats)
[![](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](https://webchat.freenode.net/?channels=%23ipfs)
[![](https://img.shields.io/badge/readme%20style-standard-brightgreen.svg?style=flat-square)](https://github.com/RichardLitt/standard-readme)
[![Travis CI](https://img.shields.io/travis/multiformats/go-multiaddr.svg?style=flat-square&branch=master)](https://travis-ci.org/multiformats/go-multiaddr)
[![codecov.io](https://img.shields.io/codecov/c/github/multiformats/go-multiaddr.svg?style=flat-square&branch=master)](https://codecov.io/github/multiformats/go-multiaddr?branch=master)

> [multiaddr](https://github.com/multiformats/multiaddr) implementation in go

Multiaddr is a standard way to represent addresses that:

- Support any standard network protocols.
- Self-describe (include protocols).
- Have a binary packed format.
- Have a nice string representation.
- Encapsulate well.

## Table of Contents

- [Install](#install)
- [Usage](#usage)
  - [Example](#example)
    - [Simple](#simple)
    - [Protocols](#protocols)
    - [En/decapsulate](#endecapsulate)
    - [Tunneling](#tunneling)
- [Maintainers](#maintainers)
- [Contribute](#contribute)
- [License](#license)

## Install

```sh
go get github.com/multiformats/go-multiaddr
```

## Usage

### Example

#### Simple

```go
import ma "github.com/multiformats/go-multiaddr"

// construct from a string (err signals parse failure)
m1, err := ma.NewMultiaddr("/ip4/127.0.0.1/udp/1234")

// construct from bytes (err signals parse failure)
m2, err := ma.NewMultiaddrBytes(m1.Bytes())

// true
strings.Equal(m1.String(), "/ip4/127.0.0.1/udp/1234")
strings.Equal(m1.String(), m2.String())
bytes.Equal(m1.Bytes(), m2.Bytes())
m1.Equal(m2)
m2.Equal(m1)
```

#### Protocols

```go
// get the multiaddr protocol description objects
m1.Protocols()
// []Protocol{
//   Protocol{ Code: 4, Name: 'ip4', Size: 32},
//   Protocol{ Code: 17, Name: 'udp', Size: 16},
// }
```

#### En/decapsulate

```go
m.Encapsulate(ma.NewMultiaddr("/sctp/5678"))
// <Multiaddr /ip4/127.0.0.1/udp/1234/sctp/5678>
m.Decapsulate(ma.NewMultiaddr("/udp")) // up to + inc last occurrence of subaddr
// <Multiaddr /ip4/127.0.0.1>
```

#### Tunneling

Multiaddr allows expressing tunnels very nicely.

```js
printer, _ := ma.NewMultiaddr("/ip4/192.168.0.13/tcp/80")
proxy, _ := ma.NewMultiaddr("/ip4/10.20.30.40/tcp/443")
printerOverProxy := proxy.Encapsulate(printer)
// /ip4/10.20.30.40/tcp/443/ip4/192.168.0.13/tcp/80

proxyAgain := printerOverProxy.Decapsulate(printer)
// /ip4/10.20.30.40/tcp/443
```

## Maintainers

Captain: [@whyrusleeping](https://github.com/whyrusleeping).

## Contribute

Contributions welcome. Please check out [the issues](https://github.com/multiformats/go-multiaddr/issues).

Check out our [contributing document](https://github.com/multiformats/multiformats/blob/master/contributing.md) for more information on how we work, and about contributing in general. Please be aware that all interactions related to multiformats are subject to the IPFS [Code of Conduct](https://github.com/ipfs/community/blob/master/code-of-conduct.md).

Small note: If editing the README, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

[MIT](LICENSE) © 2014 Juan Batiz-Benet
