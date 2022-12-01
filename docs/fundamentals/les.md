---
title: Light client
description: Introduction to Geth's light sync mode
---

<Note>Light nodes do not currently work on proof-of-stake Ethereum, but new proof-of-stake light clients are expected to ship soon!</Note>

Running a full node is the most trustless, private, decentralized and censorship resistant way to interact with Ethereum. It is also the best choice for the health of the network, because a decentralized network relies on having many individual nodes that independently verify the head of the chain. In a full node a copy of the blockchain is stored locally enabling users to verify incoming data against a local source of truth. However, running a full node requires a lot of disk space and non-negligible CPU allocation and takes hours (for snap sync) or days (for full sync) to sync the blockchain from genesis. Geth also offers a light mode that overcomes these issues and provides some of the benefits of running a node but requires only a fraction of the resources.

Read more about the reasons to run nodes on [ethereum.org](https://ethereum.org/en/run-a-node/).

<Note>Geth light clients **do not currently work** on proof-of-stake Ethereum. New light clients that work with the proof-of-stake consensus engine are expected to ship soon!</Note>

## Light node vs full node {#light-node-vs-full-node}

Running Geth in light mode has the following advantages for users:

- Syncing takes minutes rather than hours/days
- Light mode uses significantly less storage
- Light mode is lighter on CPU and other resources
- Light mode is suitable for resource-constrained devices
- Light mode can catch up much quicker after having been offline for a while

However, the cost of this performance increase is that a light Geth node depends heavily on full-node peers that choose, for altruistic reasons, to run light servers. There is no monetary incentive for full nodes to run light servers and it is an opt-in, rather than opt-out function of a Geth full node. For those reasons light servers are rather rare and can quickly become overwhelmed by data requests from light clients. The result of this is that **Geth nodes run in light mode often struggle to find peers**.

A light client can be used to query data from Ethereum and submit transactions, acting as a locally-hosted Ethereum wallet. However they have different security guarantees than full nodes. Because they don't keep local copies of the Ethereum state, light nodes can't validate the blocks in the same way as the full nodes. Instead they fetch block headers by requesting them from full nodes and check their proof-of-work (PoW), assuming the heaviest chain is valid. This means that it is sensible to wait until a few additional blocks have been confirmed before trusting the validity of a recently-mined transaction.

### Running a light server {#running-light-server}

Full node operators that choose to enable light serving altruistically enable other users to run light clients. This is good for Ethereum because it makes it easier for a wider population of users to interact with Ethereum without using trusted intermediaries. However, there is naturally a limit to how much resource a node operator is able and willing to dedicate to serving light clients. Therefore, the command that enables light serving requires arguments that define the upper bound on resource allocation. The value given is in percent of a processing thread, for example `--light.serve 300` enables light-serving and dedicates three processing threads to it.

Recent versions of Geth (>`1.9.14`) unindex older transactions to save disk space. Indexing is required for looking up transactions in Geth's database. Therefore, unindexing limits the data that can be requested by light clients. This unindexing can be disabled by adding `--tx.txlookuplimit 0` to make the maximum data available to light clients.

The whole command for starting Geth with a light server could look as follows:

```sh
geth --light.serve 50 --txlookuplimit 0
```

### Running a light client {#running-light-client}

Running a light client simply requires Geth to be started in light mode. It is likely that a user would also want to interact with the light node using, for example, RPC. This can be enabled using the `--http` command.

```sh
geth --syncmode light --http --http.api "eth,debug"
```

Data can be requested from this light Geth instance in the same way as for a full node (i.e. using the [JSON-RPC-API](/docs/interacting-with-geth/rpc/) using tools such as [Curl](https://curl.se/) or Geth's [Javascript console](/docs/interacting-with-geth/javascript-console)). Instead of fetching the data from a local database as in a full node, the light Geth instance requests the data from full-node peers.

It's also possible to send transactions. However, light clients are not connected directly to Ethereum Mainnet but to a network of light servers that connect to Ethereum Mainnet. This means a transaction submitted by a light client is received first by a light server that then propagates it to full-node peers on the light-client's behalf. This reliance on honest light-servers is one of the trust compromises that comes along with running a light node instead of a full node.

### Ultra light clients {#ultra-light-client}

Geth has an even lighter sync mode called ultra light client (ULC). The difference between light mode and ultra-light mode is that a ULC doesn't check the PoW in block headers. There is an assumption that the ULC has access to one or more trusted light servers. This option has the greatest trust assumptions but the smallest resource requirement.

To start an ultra-light client, the enode addresses of the trusted light servers must be passed to the `--ulc.servers` command and the sync mode is `light`:

```sh
geth --syncmode light --ulc.servers "enode://...,enode://..." --http --http.api "eth,debug"
```

## Summary {#summary}

Running a full node is the most trustless way to interact with Ethereum. However, Geth provides a low-resource "light" mode that can be run on modest computers and requires much less disk space. The trade-offs are additional trust assumptions and a small pool of light-serving peers to connect to.
