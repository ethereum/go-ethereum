---
title: FAQ
description: Frequently asked questions related to Geth
---

This page contains answers to common questions about Geth. The Geth team have also started to run AMA's on Reddit:

[Aug 2022 AMA](https://www.reddit.com/r/ethereum/comments/wpqmo1/ama_we_are_the_go_ethereum_geth_team_18_august/)

It is also recommended to search for 'Geth' and 'go-ethereum' on [ethereum.stackexchange](ethereum.stackexchange.com).


#### What are RPC and IPC?

IPC stands for Inter-Process Communications. Geth creates a `geth.ipc` file on startup that other processes on the same computer can use to communicate with Geth. 
RPC stands for Remote Procedure Call. RPC is a mode of communication between processes that may be running on different machines. Geth accepts RPC traffic over HTTP or Websockets. Geth functions are invoked by sending requests that are formatted according to the RPC-API to the node via either IPC or RPC.

#### What is `jwtsecret`?

The `jwtsecret` file is required to create an authenticated connection between Geth and a consensus client. JWT stands for JSON Web Token - it is signed using a secret key, proving each party's identity. Read about how to create `jwt-secret` in Geth on our [Connecting to consensus clients](/content/docs/getting_started/consensus-clients.md) page.

#### I noticed my peercount slowly decreasing, and now it is at 0.  Restarting doesn't get any peers.

This may be because your clock has fallen out of sync with other nodes. You can [force a clock update using ntp](https://askubuntu.com/questions/254826/how-to-force-a-clock-update-using-ntp) like so:

```sh
sudo ntpdate -s time.nist.gov
```

#### I would like to run multiple Geth instances but got the error "Fatal: blockchain db err: resource temporarily unavailable".

Geth uses a datadir to store the blockchain, accounts and some additional information. This directory cannot be shared between running instances. If you would like to run multiple instances follow [these](/content/docs/developers/geth-developer/Private-Network.md) instructions.

#### When I try to use the --password command line flag, I get the error "Could not decrypt key with given passphrase" but the password is correct. Why does this error appear?

Especially if the password file was created on Windows, it may have a Byte Order Mark or other special encoding that the go-ethereum client doesn't currently recognize. You can change this behavior with a PowerShell command like:

```sh
echo "mypasswordhere" | out-file test.txt -encoding ASCII
```

Additional details and/or any updates on more robust handling are at <https://github.com/ethereum/go-ethereum/issues/19905>.

#### How does Ethereum syncing work?

The current default syncing mode used by Geth is called [snap sync](https://github.com/ethereum/devp2p/blob/master/caps/snap.md). Instead of starting from the genesis block and processing all the transactions that ever occurred (which could take weeks), snap sync downloads the blocks, and only verifies the associated proof-of-works. Downloading all the blocks is a straightforward and fast procedure and will relatively quickly reassemble the entire chain.

Many people falsely assume that because they have the blocks, they are in sync. Unfortunately this is not the case. Since no transaction was executed, so we do not have any account state available (ie. balances, nonces, smart contract code and data). These need to be downloaded separately and cross-checked with the latest blocks. This phase is called the state trie download phase. Snap sync tries to hasten this process by downloading contiguous chunks of useful state data, instead of doing so one-by-one, as in previous synchronization methods.

#### What's the state trie?

In the Ethereum mainnet, there are a ton of accounts already, which track the balance, nonce, etc of each user/contract. The accounts themselves are however insufficient to run a node, they need to be cryptographically linked to each block so that nodes can actually verify that the accounts are not tampered with.

This cryptographic linking is done by creating a tree-like data structure, where each leaf corresponds to an account, and each intermediary level aggregates the layer below it into an ever smaller layer, until you reach a single root. This gigantic data structure containing all the accounts and the intermediate cryptographic proofs is called the state trie.

#### Why does the state trie download phase require a special syncing mode?

The trie data structure is an intricate interlink of hundreds of millions of tiny cryptographic proofs (trie nodes). To truly have a synchronized node, you need to download all the account data, as well as all the tiny cryptographic proofs to verify that no one in the network is trying to cheat you. This itself is already a crazy number of data items.

The part where it gets even messier is that this data is constantly morphing: at every block (roughly 13s), about 1000 nodes are deleted from this trie and about 2000 new ones are added. This means your node needs to synchronize a dataset that is changing more than 200 times per second. Until you actually do gather all the data, your local node is not usable since it cannot cryptographically prove anything about any accounts. But while you're syncing the network is moving forward and most nodes on the network keep the state for only a limited number of recent blocks. Any sync algorithm needs to consider this fact.

#### What happened to fast sync?

Snap syncing was introduced by version [1.10.0](https://blog.ethereum.org/2021/03/03/geth-v1-10-0/) and was adopted as the default mode in version [1.10.4](https://github.com/ethereum/go-ethereum/releases/tag/v1.10.4). Before that, the default was the "fast" syncing mode, which was dropped in version [1.10.14](https://github.com/ethereum/go-ethereum/releases/tag/v1.10.14). Even though support for fast sync was dropped, Geth still serves the relevant `eth` requests to other client implementations still relying on it. The reason being that snap sync relies on an alternative data structure called the [snapshot](https://blog.ethereum.org/2020/07/17/ask-about-geth-snapshot-acceleration/) which not all clients implement.

You can read more in the article posted above why snap sync replaced fast sync in Geth. Below is a table taken from the article summarising the benefits:

![snap-fast](https://user-images.githubusercontent.com/129561/109820169-6ee0af00-7c3d-11eb-9721-d8484eee4709.png)

#### When doing a fast sync, the node just hangs on importing state enties?!

The node doesn’t hang, it just doesn’t know how large the state trie is in advance so it keeps on going and going and going until it discovers and downloads the entire thing.

The reason is that a block in Ethereum only contains the state root, a single hash of the root node. When the node begins synchronizing, it knows about exactly 1 node and tries to download it. That node, can refer up to 16 new nodes, so in the next step, we’ll know about 16 new nodes and try to download those. As we go along the download, most of the nodes will reference new ones that we didn’t know about until then. This is why you might be tempted to think it’s stuck on the same numbers. It is not, rather it’s discovering and downloading the trie as it goes along.

During this phase you might see that your node is 64 blocks behind mainnet. You aren't actually synchronized. That's a side-effect of how fast sync works and you need to wait out until all state entries are downloaded.

#### I have good bandwidth, so why does downloading the state take so long when using fast sync?

State sync is mostly limited by disk IO, not bandwidth.

The state trie in Ethereum contains hundreds of millions of nodes, most of which take the form of a single hash referencing up to 16 other hashes. This is a horrible way to store data on a disk, because there's almost no structure in it, just random numbers referencing even more random numbers. This makes any underlying database weep, as it cannot optimize storing and looking up the data in any meaningful way. Snap sync solves this issue by adopting the Snapshot data structure.

#### Wait, so I can't use fast sync on an HDD?

Doing a "fast" sync on an HDD will take more time than you're willing to wait, because the data structures used are not optimized for HDDs. Even if you do wait it out, an HDD will not be able to keep up with the read/write requirements of transaction processing on mainnet. You however should be able to run a light client on an HDD with minimal impact on system resources.

#### What is wrong with my light client?

Light sync relies on full nodes that serve data to light clients. Historically, this has been hampered by the fact that serving light clients was turned off by default in geth full nodes and few nodes chose to turn it on. Therefore, light nodes often struggled to find peers. Since Ethereum switched to proof-of-stake, Geth light clients have stopped working altogether. Light clients for proof-of-stake Ethereum are expected to be implemented soon!
