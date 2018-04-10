go-libp2p-interface-pnet
==================

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](http://libp2p.io/)
[![](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](http://webchat.freenode.net/?channels=%23ipfs)

> An interface providing abstraction of swarm protection for libp2p.


## Table of Contents

- [Usage](#usage)
- [Contribute](#contribute)
- [License](#license)

## Usage

Core of this interface in `Protector` that is used to protect the swarm.
It makes decisions about which streams are allowed to pass.

This interface is accepted in multiple places in libp2p but most importantly in
go-libp2p-swarm `NewSwarmWithProtector` and `NewNetworkWithProtector`.

## Implementations:

 - [go-libp2p-pnet](//github.com/libp2p/go-libp2p-pnet) - simple PSK based Protector, using XSalsa20

## Contribute

PRs are welcome!

Small note: If editing the Readme, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

MIT © Jeromy Johnson
