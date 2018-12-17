This page describes how to set up a local cluster of nodes, advise how to make it private, and how to hook up your nodes on the eth-netstat network monitoring app. 
A fully controlled ethereum network is useful as a backend for network integration testing (core developers working on issues related to networking/blockchain synching/message propagation, etc or DAPP developers testing multi-block and multi-user scenarios).

We assume you are able to build `geth` following the [build instructions](https://github.com/ethereum/go-ethereum/wiki/Building-Ethereum)

## Setting up multiple nodes

In order to run multiple ethereum nodes locally, you have to make sure:
- each instance has a separate data directory (`--datadir`)
- each instance runs on a different port (both eth and rpc) (`--port and --rpcport`)
- in case of a cluster the instances must know about each other
- the ipc endpoint is unique or the ipc interface is disabled (`--ipcpath or --ipcdisable`)

You start the first node (let's make port explicit and disable ipc interface)
```bash
geth --datadir="/tmp/eth/60/01" -verbosity 6 --ipcdisable --port 30301 --rpcport 8101 console 2>> /tmp/eth/60/01.log
```

We started the node with the console, so that we can grab the enode url for instance:

```
> admin.nodeInfo.enode
enode://8c544b4a07da02a9ee024def6f3ba24b2747272b64e16ec5dd6b17b55992f8980b77938155169d9d33807e501729ecb42f5c0a61018898c32799ced152e9f0d7@9[::]:30301
```

`[::]` will be parsed as localhost (`127.0.0.1`). If your nodes are on a local network check each individual host machine and find your ip with `ifconfig` (on Linux and MacOS):

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

If you want to connect this instance to the previously started node you can add it as a peer from the console with `admin.addPeer(enodeUrlOfFirstInstance)`.

You can test the connection  by typing in geth console:

```javascript
> net.listening
true
> net.peerCount 
1
> admin.peers
...
```

## Local cluster

As an extention of the above, you can spawn a local cluster of nodes easily. It can also be scripted including account creation which is needed for mining. 
See [`gethcluster.sh`](https://github.com/ethersphere/eth-utils) script, and the README there for usage and examples.

## Private network 

See [[the Private Network Page|Private network]] for more information.

### Setup bootnode

The first time a node connects to the network it uses one of the predefined [bootnodes](https://github.com/ethereum/go-ethereum/blob/master/params/bootnodes.go). Through these bootnodes a node can join the network and find other nodes. In the case of a private cluster these predefined bootnodes are not of much use. Therefore go-ethereum offers a bootnode implementation that can be configured and run in your private network.

It can be run through the command.
```
> bootnode
Fatal: Use -nodekey or -nodekeyhex to specify a private key
``` 

As can be seen the bootnode asks for a key. Each ethereum node, including a bootnode is identified by an enode identifier. These identifiers are derived from a key. Therefore you will need to give the bootnode such key. Since we currently don't have one we can instruct the bootnode to generate a key (and store it in a file) before it starts.

```
> bootnode -genkey bootnode.key
I0216 09:53:08.076155 p2p/discover/udp.go:227] Listening, enode://890b6b5367ef6072455fedbd7a24ebac239d442b18c5ab9d26f58a349dad35ee5783a0dd543e4f454fed22db9772efe28a3ed6f21e75674ef6203e47803da682@[::]:30301
``` 

(exit with CTRL-C)

The stored key can be seen with:
```
> cat bootnode.key
dc90f8f7324f1cc7ba52c4077721c939f98a628ed17e51266d01c9cd0294033a
```

To instruct geth nodes to use our own bootnode(s) use the `--bootnodes` flag. This is a comma separated list of bootnode enode identifiers.

```
geth --bootnodes "enode://890b6b5367ef6072455fedbd7a24ebac239d442b18c5ab9d26f58a349dad35ee5783a0dd543e4f454fed22db9772efe28a3ed6f21e75674ef6203e47803da682@[::]:30301"
```
(what [::] means is explained previously)

Since it is convenient to start the bootnode each time with the same enode we can give the bootnode program the just generated key on the next time it is started.

```
bootnode -nodekey bootnode.key
I0216 10:01:19.125600 p2p/discover/udp.go:227] Listening, enode://890b6b5367ef6072455fedbd7a24ebac239d442b18c5ab9d26f58a349dad35ee5783a0dd543e4f454fed22db9772efe28a3ed6f21e75674ef6203e47803da682@[::]:30301
```

or

```
bootnode -nodekeyhex dc90f8f7324f1cc7ba52c4077721c939f98a628ed17e51266d01c9cd0294033a
I0216 10:01:40.094089 p2p/discover/udp.go:227] Listening, enode://890b6b5367ef6072455fedbd7a24ebac239d442b18c5ab9d26f58a349dad35ee5783a0dd543e4f454fed22db9772efe28a3ed6f21e75674ef6203e47803da682@[::]:30301
```


## Monitoring your nodes

[This page](https://github.com/ethereum/wiki/wiki/Network-Status) describes how to use the [The Ethereum (centralised) network status monitor (known sometimes as "eth-netstats")](http://stats.ethdev.com) to monitor your nodes.

[This page](https://github.com/ethereum/go-ethereum/wiki/Setting-up-monitoring-on-local-cluster) or [this README](https://github.com/ethersphere/eth-utils) 
describes how you set up your own monitoring service for a (private or public) local cluster.