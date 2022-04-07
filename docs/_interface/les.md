---
title: Light client
sort_key: B
---

*Note*: Light client is an experimental feature. Please consider this before using it with large funds.

In addition to the full client, geth supports a light mode which has several advantages for end users:

- Syncing takes minutes instead of hours (for snap sync) or days (for full sync)
- It uses significantly less storage, e.g. less than 1Gb for a node light-synced to mainnet
- It is lighter on CPU and possibly other resources
- It is hence suitable for resource-contrained devices
- It can catch up much quicker after having been offline for a while

What's the catch you might ask? They are heavily dependant on as of now altruistic light servers. These are full nodes that volunteer to serve data to light clients and which can be easily overwhelmed as the number of clients increase.

They also have slightly different security guarantees. Because they don't keep the Ethereum state, they can't validate the blocks in the same way as the full nodes. Instead they fetch block headers and check their Proof-of-Work, assuming the heaviest chain is valid. What this means for you is that if you want to be on the safe side it's advisable to wait for a few blocks worth of confirmations before trusting the validity of a recently mined transaction.

#### Light server

By enabling light serving on your full node you'll help the network and light clients. Naturally you might want to cap the amount of resources you dedicate to serving light clients. That's why the same flag that enables this feature also requires you to specify how much resources to dedicate to serving clients. `--light.serve <num>` takes a percentage as value. Number bigger than 100 indicate you want to dedicate more than one thread to this feature, e.g. `--light.serve 200` for 2 cores.

Something else you need to note is that since v1.9.14 geth unindexes old transactions to save space. This limits what clients can request. A node with unindexed transactions also can't serve lesv4 clients due to lack of support. Please keep this in mind and if possible disable unindexing by adding `--txlookuplimit 0`.

E.g. the whole command could see as follows:

```sh
geth --light.serve 50 --txlookuplimit 0
```

#### Light client

To run a light client you specify sync mode to be light. In addition to syncing you probably also want to interact with the node, say through RPC. So Let's enable that too:

```sh
geth --syncmode light --http --http.api "eth,debug"
```

Now you can request data from the RPC endpoint just like as in a full node and the light client will fetch the necessary data from servers in the background. You can also send transactions. But note that light clients do not join the main ethereum network (`eth`) so they can't propagate transactions themselves but rather give this to light servers which propagate it on their behalf.

##### Ultra light client

Geth has an even lighter sync mode called ultra light client (ULC). The difference is that an ULC doesn't check the Proof-of-Work in block headers. The assumption is that you have access to one or more light servers you are running yourself or that you trust. The command looks as follows:

```sh
geth --syncmode light --ulc.servers "enode://...,enode://..." --http --http.api "eth,debug"
```
