---
title: Connecting To The Network
description: Guide to connecting Geth to a peer-to-peer network
---

The default behaviour for Geth is to connect to Ethereum Mainnet. However, Geth can also connect to public testnets, [private networks](/docs/fundamentals/private-network) and [local testnets](/docs/developers/geth-developer/dev-mode). For convenience, the two public testnets with long term support, Goerli and Sepolia, have their own command line flag. Geth can connect to these testnets simply by passing:

- `--goerli`, Goerli proof-of-authority test network
- `--sepolia` Sepolia proof-of-work test network

These testnets started as proof-of-work and proof-of-authority testnets, but they were transitioned to proof-of-stake in 2022 in preparation for doing the same to Ethereum Mainnet. This means that to run a node on Goerli or Sepolia it is now necessary to run a consensus client connected to Geth. This is also true for Ethereum Mainnet. **Geth does not work on proof-of-stake networks without a consensus client**! The remainder of this page will assume that Geth is connected to a consensus client that is synced to the desired network. For instructions on how to set up a consensus client please see the [Consensus Clients](/docs/getting-started/consensus-clients) page.

**Note:** Network selection is not persisted from a config file. To connect to a pre-defined network you must always enable it explicitly, even when using the `--config` flag to load other configuration values. For example:

```sh
# Generate desired config file. You must specify testnet here.
geth --goerli --syncmode "full" ... dumpconfig > goerli.toml

# Start geth with given config file. Here too the testnet must be specified.
geth --goerli --config goerli.toml
```

## Finding peers {#finding-peers}

Geth continuously attempts to connect to other nodes on the network until it has enough peers. If UPnP (Universal Plug and Play) is enabled at the router or Ethereum is run on an Internet-facing server, it will also accept connections from other nodes. Geth finds peers using the [discovery protocol](https://ethereum.org/en/developers/docs/networking-layer/#discovery). In the discovery protocol, nodes exchange connectivity details and then establish sessions ([RLPx](https://github.com/ethereum/devp2p/blob/master/rlpx.md)). If the nodes support compatible sub-protocols they can start exchanging Ethereum data [on the wire](https://ethereum.org/en/developers/docs/networking-layer/#wire-protocol).

A new node entering the network for the first time gets introduced to a set of peers by a bootstrap node ("bootnode") whose sole purpose is to connect new nodes to peers. The endpoints for these bootnodes are hardcoded into Geth, but they can also be specified by providing the `--bootnode` flag along with comma-separated bootnode addresses in the form of [enodes](https://ethereum.org/en/developers/docs/networking-layer/network-addresses/#enode) on startup. For example:

```sh
geth --bootnodes enode://pubkey1@ip1:port1,enode://pubkey2@ip2:port2,enode://pubkey3@ip3:port3
```

There are scenarios where disabling the discovery process is useful, for example for running a local test node or an experimental test network with known, fixed nodes. This can be achieved by passing the `--nodiscover` flag to Geth at startup.

## Connectivity problems {#connectivity-problems}

There are occasions when Geth simply fails to connect to peers. The common reasons for this are:

- Local time might be incorrect. An accurate clock is required to participate in the Ethereum network. The local clock can be resynchronized using commands such as `sudo ntpdate -s time.nist.gov` (this will vary depending on operating system).

- Some firewall configurations can prohibit UDP traffic. The static nodes feature or `admin.addPeer()` on the console can be used to configure connections manually.

- Running Geth in [light mode](/docs/fundamentals/les) often leads to connectivity issues because there are few nodes running light servers. There is no easy fix for this except to switch Geth out of light mode. **Note that light mode does not currently work on proof-of-stake networks**.

- The public test network Geth is connecting to might be deprecated or have a low number of active nodes that are hard to find. In this case, the best action is to switch to an alternative test network.

## Checking Connectivity {#checking-connectivity}

The `net` module has two attributes that enable checking node connectivity from the [interactive Javascript console](/docs/interacting-with-geth/javascript-console). These are `net.listening` which reports whether the Geth node is listening for inbound requests, and `peerCount` which returns the number of active peers the node is connected to.

```js
> net.listening
true

> net.peerCount
4
```

Functions in the `admin` module provide more information about the connected peers, including their IP address, port number, supported protocols etc. Calling `admin.peers` returns this information for all connected peers.

```sh
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

```sh
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

## Custom Networks {#custom-networks}

It is often useful for developers to connect to private test networks rather than public testnets or Ethereum mainnet. These sandbox environments allow block creation without competing against other miners, easy minting of test ether and give freedom to break things without real-world consequences. A private network is started by providing a value to `--networkid` that is not used by any other existing public network ([Chainlist](https://chainlist.org)) and creating a custom `genesis.json` file. Detailed instructions for this are available on the [Private Networks page](/docs/fundamentals/private-network).

## Static nodes {#static-nodes}

Geth also supports static nodes. Static nodes are specific peers that are always connected to. Geth reconnects to these peers automatically when it is restarted. Specific nodes are defined to be static nodes by adding their enode addresses to a config file. The easiest way to create this config file is to run:

```sh
geth --datadir <datadir> dumpconfig > config.toml
```

This will create `config.toml` in the current directory. The enode addresses for static nodes can then be added as a list to the `StaticNodes` field of the `Node.P2P` section in `config.toml`. When Geth is started, pass `--config config.toml`. The relevant line in `config.toml` looks as follows:

```toml
StaticNodes = ["enode://f4642fa65af50cfdea8fa7414a5def7bb7991478b768e296f5e4a54e8b995de102e0ceae2e826f293c481b5325f89be6d207b003382e18a8ecba66fbaf6416c0@33.4.2.1:30303"]
```

Ensure the other lines in `config.toml` are also set correctly before starting Geth, as passing `--config` instructs Geth to get its configuration values from this file. An example of a complete `config.toml` file can be found [here](https://gist.github.com/jmcook1186/16db2f0feddb4bd0581ebb9ba867a47a).

Static nodes can also be added at runtime in the Javascript console by passing an enode address to `admin.addPeer()`:

```js
admin.addPeer(
  'enode://f4642fa65af50cfdea8fa7414a5def7bb7991478b768e296f5e4a54e8b995de102e0ceae2e826f293c481b5325f89be6d207b003382e18a8ecba66fbaf6416c0@33.4.2.1:30303'
);
```

## Peer limit {#peer-limit}

It is sometimes desirable to cap the number of peers Geth will connect to in order to limit on the computational and bandwidth cost associated with running a node. By default, the limit is 50 peers, however, this can be updated by passing a value to `--maxpeers`:

```sh
geth <otherflags> --maxpeers 15
```

## Trusted nodes {#trusted-nodes}

Trusted nodes can be added to `config.toml` in the same way as for static nodes. Add the trusted node's enode address to the `TrustedNodes` field in `config.toml` before starting Geth with `--config config.toml`.

Nodes can be added using the `admin.addTrustedPeer()` call in the Javascript console and removed using `admin.removeTrustedPeer()` call.

```js
admin.addTrustedPeer(
  'enode://f4642fa65af50cfdea8fa7414a5def7bb7991478b768e296f5e4a54e8b995de102e0ceae2e826f293c481b5325f89be6d207b003382e18a8ecba66fbaf6416c0@33.4.2.1:30303'
);
```

## Summary {#summary}

Geth connects to Ethereum Mainnet by default. However, this behaviour can be changed using combinations of command line flags and files. This page has described the various options available for connecting a Geth node to Ethereum, public testnets and private networks. Remember that to connect to a proof-of-stake network (e.g. Ethereum Mainnet, Goerli, Sepolia) a consensus client is also required.
