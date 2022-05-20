---
title: Connecting To The Network
sort_key: B
---

The default behaviour for Geth is to connect to Ethereum Mainnet. However, Geth can also connect to public testnets, [private networks](/docs/getting-started/private-net) and [local testnets](/docs/getting-started/dev-mode). Command line flags are provided for connecting to the popular public testnets:

- `--ropsten`, Ropsten proof-of-work test network
- `--rinkeby`, Rinkeby proof-of-authority test network
- `--goerli`, Goerli proof-of-authority test network
- `--sepolia` Sepolia proof-of-work test network

Providing these flags at startup instructs Geth to connect to the specific public testnet instead of Ethereum Mainnet. Because these are public testnets that have been running for several years, Geth has to download the historical blockchain data from genesis, just the same as for Ethereum Mainnet.

**Note:** network selection is not persisted in the config file. To connect to a pre-defined network you must always enable it explicitly, even when using the `--config` flag to load other configuration values. For example:

```shell 

# Generate desired config file. You must specify testnet here.
geth --goerli --syncmode "full" ... dumpconfig > goerli.toml

# Start geth with given config file. Here too the testnet must be specified.
geth --goerli --config goerli.toml

```

## Finding peers

Geth continuously attempts to connect to other nodes on the network until it has enough peers. If UPnP (Universal Plug and Play) is enabled at the router or Ethereum is run on an Internet-facing server, it will also accept connections from other nodes. Geth finds peers using the [discovery protocol](https://ethereum.org/en/developers/docs/networking-layer/#discovery). In the discovery protocol, nodes exchange connectivity details and then establish sessions ([RLPx](https://github.com/ethereum/devp2p/blob/master/rlpx.md)). If the nodes support compatible sub-protocols they can start exchanging Ethereum data [on the wire](https://ethereum.org/en/developers/docs/networking-layer/#wire-protocol). 

A new node entering the network for the first time gets introduced to a set of peers by a bootstrap node ("bootnode") whose sole purpose is to connect new nodes to peers. The endpoints for these bootnodes are hardcoded into Geth, but they can also be specified by providing the `--bootnode` flag along with comma-separated bootnode addresses in the form of [enodes](https://ethereum.org/en/developers/docs/networking-layer/network-addresses/#enode) on startup. For example:

```shell

geth --bootnodes enode://pubkey1@ip1:port1,enode://pubkey2@ip2:port2,enode://pubkey3@ip3:port3

```

There are scenarios where disabling the discovery process is useful, for example for running a local test node or an experimental test network with known, fixed nodes. This can be achieved by passing the `--nodiscover` flag to Geth at startup.


## Connectivity problems

There are occasions when Geth simply fails to connect to peers. The common reasons for this are:

- Local time might be incorrect. An accurate clock is required to participate in the Ethereum network. The local clock can be resynchronized using commands such as `sudo ntpdate -s time.nist.gov` (this will vary depending on operating system).

- Some firewall configurations can prohibit UDP traffic. The static nodes feature or `admin.addPeer()` on the console can be used to configure connections manually.

- Running Geth in [light mode](/docs/interface/les) often leads to connectivity issues because there are few nodes running light servers. There is no easy fix for this except to switch Geth out of light mode.

- The public test network Geth is connecting to might be deprecated or have a low number of active nodes that are hard to find. In this case, the best action is to switch to an alternative test network.


## Checking Connectivity

The `net` module has two attributes that enable checking node connectivity from the [interactive Javascript console](/docs/interface/javascript-console). These are `net.listening` which reports whether the Geth node is listening for inbound requests, and `peerCount` which returns the number of active peers the node is connected to.


```javascript

> net.listening
true

> net.peerCount
4

```

Functions in the `admin` module provide more information about the connected peers, including their IP address, port number, supported protocols etc. Calling `admin.peers` returns this information for all connected peers.

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

The `admin` module also includes functions for gathering information about the local node rather than its peers. For example, `admin.nodeInfo` returns the name and connectivity details for the local node.

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

It is often useful for developers to connect to private test networks rather than public testnets or Etheruem mainnet. These sandbox environments allow block creation without competing against other miners, easy minting of test ether and give freedom to break things without real-world consequences. A private network is started by providing a value to `--networkid` that is not used by any other existing public network ([Chainlist](https://chainlist.org)) and creating a custom `genesis.json` file. Detailed instructions for this are available on the [Private Networks page](/docs/interface/private-network).


## Static nodes

Geth also supports static nodes. Static nodes are specific peers that are always connected to. Geth reconnects to these peers automatically when it is restarted. Specific nodes are defined to be static nodes by saving their enode addresses to a json file which must be stored in `datadir/geth/static-nodes.json`. The content of `static-nodes.json` should be formatted as follows:

```javascript
[
  "enode://f4642fa65af50cfdea8fa7414a5def7bb7991478b768e296f5e4a54e8b995de102e0ceae2e826f293c481b5325f89be6d207b003382e18a8ecba66fbaf6416c0@33.4.2.1:30303",
  "enode://pubkey@ip:port"
]
```

Static nodes can also be added at runtime in the Javascript console by passing an enode address to `admin.addPeer()`:

```javascript

admin.addPeer("enode://f4642fa65af50cfdea8fa7414a5def7bb7991478b768e296f5e4a54e8b995de102e0ceae2e826f293c481b5325f89be6d207b003382e18a8ecba66fbaf6416c0@33.4.2.1:30303")

```

## Peer limit

It is sometimes desirable to cap the number of peers Geth will connect to in order to limit on the computational and bandwidth cost associated with running a node. By default, the limit is 50 peers, however, this can be updated by passing a value to `--maxpeers`:

```shell

geth <otherflags> --maxpeers 15

```

## Trusted nodes

Geth supports trusted nodes that are always allowed to reconnect, even if the peer limit is reached. They can be added persistently via a config file `<datadir>/geth/trusted-nodes.json` or temporarily using the Javascript console. The format for the config file is identical to the one used for static nodes.

Nodes can be added using the `admin.addTrustedPeer()` call in the Javascript console and removed using `admin.removeTrustedPeer()` call.

```javascript
admin.addTrustedPeer("enode://f4642fa65af50cfdea8fa7414a5def7bb7991478b768e296f5e4a54e8b995de102e0ceae2e826f293c481b5325f89be6d207b003382e18a8ecba66fbaf6416c0@33.4.2.1:30303")
```


## Summary

Geth connects to Ethereum Mainnet by default. However, this behaviour can be changed using combinations of command line flags and files. This page has described the various options available for connecting a Geth node to Ethereum, public testnets and private networks.
