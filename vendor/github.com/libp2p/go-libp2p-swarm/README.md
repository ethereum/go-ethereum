go-libp2p-swarm
==================

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](http://libp2p.io/)
[![](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](http://webchat.freenode.net/?channels=%23ipfs)
[![Coverage Status](https://coveralls.io/repos/github/libp2p/go-libp2p-swarm/badge.svg?branch=master)](https://coveralls.io/github/libp2p/go-libp2p-swarm?branch=master)
[![Travis CI](https://travis-ci.org/libp2p/go-libp2p-swarm.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-swarm)

> The libp2p swarm manages groups of connections to peers, and handles incoming and outgoing streams.

The libp2p swarm is the 'low level' interface for working with a given libp2p
network. It gives you more fine grained control over various aspects of the
system. Most applications don't need this level of access, so the `Swarm` is
generally wrapped in a `Host` abstraction that provides a more friendly
interface. See [the host interface](https://github.com/libp2p/go-libp2p-host)
for more info on that.

## Table of Contents

- [Install](#install)
- [Contribute](#contribute)
- [License](#license)

## Install

```sh
make install
```

## Usage

### Creating a swarm
To construct a swarm, you'll be calling `NewSwarm`. That function looks like this:
```go
swarm, err := NewSwarm(ctx, laddrs, pid, pstore, bwc)
```

It takes five items to fully construct a swarm, the first is a go
`context.Context`. This controls the lifetime of the swarm, and all swarm
processes have their lifespan derived from the given context. You can just use
`context.Background()` if you're not concerned with that.

The next argument is an array of multiaddrs that the swarm will open up
listeners for. Once started, the swarm will start accepting and handling
incoming connections on every given address. This argument is optional, you can
pass `nil` and the swarm will not listen for any incoming connections (but will
still be able to dial out to other peers).

After that, you'll need to give the swarm an identity in the form of a peer.ID.
If you're not wanting to enable secio (libp2p's transport layer encryption),
then you can pick any string for this value. For example `peer.ID("FooBar123")`
would work. Note that passing a random string ID will result in your node not
being able to communicate with other peers that have correctly generated IDs.
To see how to generate a proper ID, see the below section on "Identity
Generation".

The fourth argument is a peerstore. This is essentially a database that the
swarm will use to store peer IDs, addresses, public keys, protocol preferences
and more. You can construct one by importing
`github.com/libp2p/go-libp2p-peerstore` and calling `peerstore.NewPeerstore()`.

The final argument is a bandwidth metrics collector, This is used to track
incoming and outgoing bandwidth on connections managed by this swarm. It is
optional, and passing `nil` will simply result in no metrics for connections
being available.

#### Identity Generation
A proper libp2p identity is PKI based. We currently have support for RSA and ed25519 keys. To create a 'correct' ID, you'll need to either load or generate a new keypair. Here is an example of doing so:

```go
import (
	"fmt"
	"crypto/rand"

	ci "github.com/libp2p/go-libp2p-crypto"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	peer "github.com/libp2p/go-libp2p-peer"
)

func demo() {
	// First, select a source of entropy. We're using the stdlib's crypto reader here
	src := rand.Reader

	// Now create a 2048 bit RSA key using that
	priv, pub, err := ci.GenerateKeyPairWithReader(ci.RSA, 2048, src)
	if err != nil {
		panic(err) // oh no!
	}

	// Now that we have a keypair, lets create our identity from it
	pid, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		panic(err)
	}

	// Woo! Identity acquired!
	fmt.Println("I am ", pid)

	// Now, for the purposes of building a swarm, lets add this all to a peerstore.
	ps := pstore.NewPeerstore()
	ps.AddPubKey(pid, pub)
	ps.AddPrivKey(pid, priv)

	// Once you've got all that, creating a basic swarm can be as easy as
	ctx := context.Background()
	swarm, err := NewSwarm(ctx, nil, pid, ps, nil)

	// voila! A functioning swarm!
}
```

### Streams
The swarm is designed around using multiplexed streams to communicate with
other peers. When working with a swarm, you will want to set a function to
handle incoming streams from your peers:

```go
swrm.SetStreamHandler(func(s inet.Stream) {
	defer s.Close()
	fmt.Println("Got a stream from: ", s.SwarmConn().RemotePeer())
	fmt.Fprintln(s, "Hello Friend!")
})
```

Tip: Always make sure to close streams when you're done with them.

Opening streams is also pretty simple:
```go
s, err := swrm.NewStreamWithPeer(ctx, rpid)
if err != nil {
	panic(err)
}
defer s.Close()

io.Copy(os.Stdout, s) // pipe the stream to stdout
```

Just pass a context and the ID of the peer you want a stream to, and you'll get
back a stream to read and write on.


## Contribute

PRs are welcome!

Small note: If editing the Readme, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

MIT © Jeromy Johnson
