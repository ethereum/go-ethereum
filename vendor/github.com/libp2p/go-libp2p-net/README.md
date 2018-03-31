# go-libp2p-net

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-libp2p-blue.svg?style=flat-square)](http://github.com/libp2p/libp2p)
[![](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](http://webchat.freenode.net/?channels=%23ipfs)

> Network interfaces for go-libp2p

The IPFS Network package handles all of the peer-to-peer networking. It connects to other hosts, it encrypts communications, it muxes messages between the network's client services and target hosts. It has multiple subcomponents:

- `Conn` - a connection to a single Peer
  - `MultiConn` - a set of connections to a single Peer
  - `SecureConn` - an encrypted (tls-like) connection
- `Swarm` - holds connections to Peers, multiplexes from/to each `MultiConn`
- `Muxer` - multiplexes between `Services` and `Swarm`. Handles `Request/Reply`.
  - `Service` - connects between an outside client service and Network.
  - `Handler` - the client service part that handles requests

It looks a bit like this:

[![](https://docs.google.com/drawings/d/1FvU7GImRsb9GvAWDDo1le85jIrnFJNVB_OTPXC15WwM/pub?h=480)]

## Install

```sh
go get libp2p/go-libp2p-net
```

## Contribute

Feel free to join in. All welcome. Open an [issue](https://github.com/ipfs/go-libp2p-net/issues)!

Check out our [contributing document](https://github.com/libp2p/community/blob/master/CONTRIBUTE.md) for more information on how we work, and about contributing in general. Please be aware that all interactions related to libp2p are subject to the IPFS [Code of Conduct](https://github.com/ipfs/community/blob/master/code-of-conduct.md).

Small note: If editing the README, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

### Want to hack on IPFS?

[![](https://cdn.rawgit.com/jbenet/contribute-ipfs-gif/master/img/contribute.gif)](https://github.com/ipfs/community/blob/master/contributing.md)

## License

[MIT](LICENSE) © 2016 Jeromy Johnson
