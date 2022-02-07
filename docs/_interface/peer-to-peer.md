---
title: Connecting To The Network
sort_key: B
---

If you start geth without any flags, it will connect to the Ethereum mainnet. In addition to
the mainnet, geth recognizes a few testnets which you can connect to via the respective flags:

- `--ropsten`, Ropsten proof-of-work test network
- `--rinkeby`, Rinkeby proof-of-authority test network
- `--goerli`, Goerli proof-of-authority test network

**Note:** network selection is not persisted in the config file. To connect to a pre-defined network
you must always enable it explicitly, even when using the `--config` flag to load other configuration values.
For example:

```sh
# Generate desired config file. You must specify testnet here.
geth --goerli --syncmode "full" ... dumpconfig > goerli.toml

# Start geth with given config file. Here too the testnet must be specified.
geth --goerli --config goerli.toml
```

## How Peers Are Found

Geth continuously attempts to connect to other nodes on the network until it has peers. If
you have UPnP enabled on your router or run ethereum on an Internet-facing server, it will
also accept connections from other nodes.

Geth finds peers through something called the discovery protocol. In the discovery
protocol, nodes are gossipping with each other to find out about other nodes on the
network. In order to get going initially, geth uses a set of bootstrap nodes whose
endpoints are recorded in the source code.

To change the bootnodes on startup, use the `--bootnodes` option and separate the nodes by
commas. For example:

    geth --bootnodes enode://pubkey1@ip1:port1,enode://pubkey2@ip2:port2,enode://pubkey3@ip3:port3

## Common Problems With Connectivity

Sometimes you just can't get connected. The most common reasons are as follows:

- Your local time might be incorrect. An accurate clock is required to participate in the
  Ethereum network. Check your OS for how to resync your clock (example `sudo ntpdate -s
  time.nist.gov`) because even 12 seconds too fast can lead to 0 peers.
- Some firewall configurations can prevent UDP traffic from flowing. You can use the
  static nodes feature or `admin.addPeer()` on the console to configure connections by
  hand.

To start geth without the discovery protocol, you can use the `--nodiscover` parameter.
You only want this if you are running a test node or an experimental test network with
fixed nodes.

## Checking Connectivity

To check how many peers the client is connected to in the interactive console, the `net`
module has two attributes that give you info about the number of peers and whether you
are a listening node.

```js
> net.listening
true
> net.peerCount
4
```

To get more information about the connected peers, such as IP address and port number,
supported protocols, use the `peers()` function of the `admin` object. `admin.peers()`
returns the list of currently connected peers.

```
> admin.peers
[{
  ID: 'a4de274d3a159e10c2c9a68c326511236381b84c9ec52e72ad732eb0b2b1a2277938f78593cdbe734e6002bf23114d434a085d260514ab336d4acdc312db671b',
  Name: 'Geth/v0.9.14/linux/go1.4.2',
  Caps: 'eth/60',
  RemoteAddress: '5.9.150.40:30301',
  LocalAddress: '192.168.0.28:39219'
}, {
  ID: 'a979fb575495b8d6db44f750317d0f4622bf4c2aa3365d6af7c284339968eef29b69ad0dce72a4d8db5ebb4968de0e3bec910127f134779fbcb0cb6d3331163c',
  Name: 'Geth/v0.9.15/linux/go1.4.2',
  Caps: 'eth/60',
  RemoteAddress: '52.16.188.185:30303',
  LocalAddress: '192.168.0.28:50995'
}, {
  ID: 'f6ba1f1d9241d48138136ccf5baa6c2c8b008435a1c2bd009ca52fb8edbbc991eba36376beaee9d45f16d5dcbf2ed0bc23006c505d57ffcf70921bd94aa7a172',
  Name: 'pyethapp_dd52/v0.9.13/linux2/py2.7.9',
  Caps: 'eth/60, p2p/3',
  RemoteAddress: '144.76.62.101:30303',
  LocalAddress: '192.168.0.28:40454'
}, {
  ID: 'f4642fa65af50cfdea8fa7414a5def7bb7991478b768e296f5e4a54e8b995de102e0ceae2e826f293c481b5325f89be6d207b003382e18a8ecba66fbaf6416c0',
  Name: '++eth/Zeppelin/Rascal/v0.9.14/Release/Darwin/clang/int',
  Caps: 'eth/60, shh/2',
  RemoteAddress: '129.16.191.64:30303',
  LocalAddress: '192.168.0.28:39705'
} ]

```

To check the ports used by geth and also find your enode URI run:

```
> admin.nodeInfo
{
  Name: 'Geth/v0.9.14/darwin/go1.4.2',
  NodeUrl: 'enode://3414c01c19aa75a34f2dbd2f8d0898dc79d6b219ad77f8155abf1a287ce2ba60f14998a3a98c0cf14915eabfdacf914a92b27a01769de18fa2d049dbf4c17694@[::]:30303',
  NodeID: '3414c01c19aa75a34f2dbd2f8d0898dc79d6b219ad77f8155abf1a287ce2ba60f14998a3a98c0cf14915eabfdacf914a92b27a01769de18fa2d049dbf4c17694',
  IP: '::',
  DiscPort: 30303,
  TCPPort: 30303,
  Td: '2044952618444',
  ListenAddr: '[::]:30303'
}
```

## Custom Networks

Sometimes you might not need to connect to the live public network, you can instead choose
to create your own private testnet. This is very useful if you don't need to test external
contracts and want just to test the technology, because you won't have to compete with
other miners and will easily generate a lot of test ether to play around (replace 12345
with any non-negative number):

	geth -—networkid="12345" console

It is also possible to run geth with a custom genesis block from a JSON file by supplying
the `--genesis` flag. The genesis JSON file should have the following format:

```js
{
  "alloc": {
    "dbdbdb2cbd23b783741e8d7fcf51e459b497e4a6": { 
        "balance": "1606938044258990275541962092341162602522202993782792835301376"
    },
    "e6716f9544a56c530d868e4bfbacb172315bdead": {
        "balance": "1606938044258990275541962092341162602522202993782792835301376"
    },
    ...
  },
  "nonce": "0x000000000000002a",
  "difficulty": "0x020000",
  "mixhash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "coinbase": "0x0000000000000000000000000000000000000000",
  "timestamp": "0x00",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "extraData": "0x",
  "gasLimit": "0x2fefd8"
}
``` 

## Static nodes

Geth also supports a feature called static nodes if you have certain peers you always want
to connect to. Static nodes are re-connected on disconnects. You can configure permanent
static nodes by putting something like the following into
`<datadir>/geth/static-nodes.json`:

```js
[
  "enode://f4642fa65af50cfdea8fa7414a5def7bb7991478b768e296f5e4a54e8b995de102e0ceae2e826f293c481b5325f89be6d207b003382e18a8ecba66fbaf6416c0@33.4.2.1:30303",
  "enode://pubkey@ip:port"
]
```

You can also add static nodes at runtime via the js console using
`admin.addPeer()`:

```js
admin.addPeer("enode://f4642fa65af50cfdea8fa7414a5def7bb7991478b768e296f5e4a54e8b995de102e0ceae2e826f293c481b5325f89be6d207b003382e18a8ecba66fbaf6416c0@33.4.2.1:30303")
```

## Trusted nodes

Geth supports trusted nodes that are always allowed to reconnect, even if the peer limit is reached.
They can be added permanently via a config file `<datadir>/geth/trusted-nodes.json` or temporary via RPC call.
The format for the config file is identical to the one used for static nodes.
Nodes can be added using the `admin.addTrustedPeer()` RPC-call over the js console and removed using the `admin.removeTrustedPeer()` call.

```js
admin.addTrustedPeer("enode://f4642fa65af50cfdea8fa7414a5def7bb7991478b768e296f5e4a54e8b995de102e0ceae2e826f293c481b5325f89be6d207b003382e18a8ecba66fbaf6416c0@33.4.2.1:30303")
```
