---
title: Creating a local cluster
---

<!-- TODO: Redirects -->

> In the first link, the first section "Setting up multiple nodes" does not necessarily imply that those nodes will be connected in any way. They could literally be separate nodes that have no knowledge of each other. Perhaps serving as an array of nodes for some local testers, for example.
>
> It seems that it is trying to describe how to run multiple nodes locally, without those nodes treading on each other's toes and corrupting each other's accounts.
>
> This does not mean they will know about each other and 'form a p2p network'

* * *

This page describes how to set up a local cluster of nodes, and connect it to the [eth-netstat](https://github.com/cubedro/eth-netstats) network monitoring app.

An Ethereum cluster under your control is useful as a backend for network integration testing. For example, core developers working on issues related to networking/blockchain synching/message propagation, or dapp developers testing multi-block and multi-user scenarios.

## Setting up multiple nodes

To run multiple ethereum nodes locally, you have to make sure:

-   each instance has a separate data directory (`--datadir`)
-   each instance runs on a different port (both ETH and RPC) (`--port and --rpcport`)
-   in case of a cluster the instances must know about each other
-   the IPC endpoint is unique or the IPC interface is disabled (`--ipcpath or --ipcdisable`)

Create the temporary directories needed by geth to store data and save logs:

<!-- TODO: Why `60` -->

```shell
mkdir -p /tmp/eth/60/01
mkdir -p /tmp/eth/60/02
```

Start the first node, making the port explicit and disabling the IPC interface:

```shell
geth --datadir="/tmp/eth/60/01" -verbosity 6 --ipcdisable --port 30301 --rpcport 8101 console 2>> /tmp/eth/60/01.log
```

The node started with the console, so you can get the enode url that other nodes use to connect to it:

```shell
> admin.nodeInfo.enode

enode://{PUBLIC_KEY}@{IP_ADDRESS}:30301
```

If your nodes are on a local network check each individual host machine and find your ip with `ifconfig` (on Linux and MacOS):

```bash
$ ifconfig|grep netmask|awk '{print $2}'
127.0.0.1
192.168.1.97
```

If your peers are not on the local network, you need to know your external IP address (use a service) to construct the enode url.

Now you can launch a second node with:

```bash
geth --datadir="/tmp/eth/60/02" --verbosity 6 --ipcdisable --port 30302 --rpcport 8102 console 2>> /tmp/eth/60/02.log
```

To connect this instance to the first node you can add it as a peer from the console with `admin.addPeer(enodeUrlOfFirstInstance)`.

You can test the connection with the following commands:

```javascript
> net.listening
true
> net.peerCount
2
> admin.peers
[{
    caps: ["eth/62", "eth/63"],
    enode: "enode://{PUBLIC_KEY}@{IP_ADDRESS}:30301",
    id: "638c0193168646dd18136c3de7f5f4342b879b274f9bec90c60dfc1a04c50051",
    name: "Geth/main.jnode.network/v1.8.22-stable-7fa3509e/linux-amd64/go1.11.5",
    network: {
      inbound: false,
      localAddress: "{IP_ADDRESS}:55364",
      remoteAddress: "{IP_ADDRESS}:30301",
      static: false,
      trusted: false
    },
    protocols: {
      eth: {
        difficulty: 1.2651073550501831e+22,
        head: "0xa366e1198576fba10fe767199c7a330679cd37eab4f40fd8ac6eb2c84b7d3d0b",
        version: 63
      }
    }
}]
```

### Scripting cluster creation

You can automate the creation a local cluster of nodes, including account creation which is needed for mining. See the [`gethcluster.sh`](https://github.com/ethersphere/eth-utils/blob/master/gethcluster.sh) script, and the [README](https://github.com/ethersphere/eth-utils/blob/master/README.md) for usage and examples.

### Monitoring your nodes

<!-- TODO: Bring in? -->

[This guide](https://github.com/ethereum/wiki/wiki/Network-Status) describes how to use the [The Ethereum (centralised) network status monitor (known sometimes as "eth-netstats")](http://stats.ethdev.com) to monitor your nodes.

[This page](Setting-up-monitoring-on-local-cluster) or [this README](https://github.com/ethersphere/eth-utils)
describes how you set up your own monitoring service for a (private or public) local cluster.
