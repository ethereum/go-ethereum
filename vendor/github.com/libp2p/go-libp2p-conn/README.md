go-libp2p-conn
==================

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](http://libp2p.io/)
[![](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](http://webchat.freenode.net/?channels=%23ipfs)
[![Coverage Status](https://coveralls.io/repos/github/libp2p/go-libp2p-conn/badge.svg?branch=master)](https://coveralls.io/github/libp2p/go-libp2p-conn?branch=master)
[![Travis CI](https://travis-ci.org/libp2p/go-libp2p-conn.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-conn)

> A library providing 'Connection' objects for libp2p.

This package offers wrappers for `go-libp2p-transport` raw types,
exposing `go-libp2p-interface-conn` types.

It negotiates either plaintext or secio over the raw connection
using `go-multistream`.

## Table of Contents

- [Install](#install)
- [Usage](#usage)
- [Protocol overview](#protocol-overview)
- [Contribute](#contribute)
- [License](#license)

## Install

```sh
make deps
```

## Usage

On the server side, a `go-libp2p-transport` Listener is wrapped in a `go-libp2p-interface-conn` Listener with `WrapTransportListener`. Such `iconn.Listener` has a peer identity: an ID and a secret key. These are only used when connections are encrypted, and a missing secret key forces plaintext connections.

On the client side, a `Dialer` creates `go-libp2p-interface-conn` connections using a set of `go-libp2p-transport` Dialers. Like with Listener, a Dialer has an ID and private key identity to be used to negotiate encrypted connections. Dial also checks the peer identity if encryption is enabled by specifying a secret key in Dialer.

Encryption is forced on when `go-libp2p-interface-conn.EncryptConnections` is true and the Dialer/Listener has a secret key, and forced off otherwise.

## Protocol overview

The protocol is fairly straightforward: upon opening a connection, `go-multistream` is used to agree on plaintext (`"/plaintext/1.0.0"`) or encrypted (`"/secio/1.0.0"`). Plaintext will only be negotiated iff both peers have `go-libp2p-interface-conn.EncryptConnections` set to `false` or haven't constructed their Listeners/Dialers with secret keys.

If plaintext is selected, the connection is used as-is for the rest of its lifetime.

If encrypted is selected, `go-libp2p-secio` is used to negotiate a transparent encrypted tunnel. The negotiation happens before the connection is made available to the library consumer.

## Contribute

PRs are welcome!

Small note: If editing the Readme, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

### Tests

```sh
make deps
go test
```

## License

MIT © Jeromy Johnson
