---
title: FAQ
permalink: docs/faq
sort_key: C
---


#### I noticed my peercount slowly decreasing, and now it is at 0.  Restarting doesn't get any peers.

Check and sync your clock with ntp. For example, you can [force a clock update using ntp](https://askubuntu.com/questions/254826/how-to-force-a-clock-update-using-ntp) like so:

```sh
sudo ntpdate -s time.nist.gov
```

#### I would like to run multiple geth instances but got the error "Fatal: blockchain db err: resource temporarily unavailable".

Geth uses a datadir to store the blockchain, accounts and some additional information. This directory cannot be shared between running instances. If you would like to run multiple instances follow [these](getting-started/private-net) instructions.

#### When I try to use the --password command line flag, I get the error "Could not decrypt key with given passphrase" but the password is correct. Why does this error appear?

Especially if the password file was created on Windows, it may have a Byte Order Mark or other special encoding that the go-ethereum client doesn't currently recognize.  You can change this behavior with a PowerShell command like `echo "mypasswordhere" | out-file test.txt -encoding ASCII`.  Additional details and/or any updates on more robust handling are at <https://github.com/ethereum/go-ethereum/issues/19905>.

#### I'm trying to sync my node, but it seems to be stuck at 64 blocks behind mainnet?!

If you see that you are 64 blocks behind mainnet, you aren't yet fully synchronized. You are just done with the block download phase and still haven't finished all other syncing phases.

#### How does Ethereum syncing work?

The current default syncing mode used by Geth is called [snap sync](https://github.com/ethereum/devp2p/blob/master/caps/snap.md). Instead of starting from the genesis block and processing all the transactions that ever occurred (which could take weeks), snap sync downloads the blocks, and only verifies the associated proof-of-works. Downloading all the blocks is a straightforward and fast procedure and will relatively quickly reassemble the entire chain.

Many people falsely assume that because they have the blocks, they are in sync. Unfortunately this is not the case, since no transaction was executed, so we do not have any account state available (ie. balances, nonces, smart contract code and data). These need to be downloaded separately and cross-checked with the latest blocks. This phase is called the state trie download phase. Snap sync tries to hasten this process by downloading contiguous chunks of useful state data, instead of doing so one-by-one, as in previous synchronization methods.

#### So, what's the state trie?

In the Ethereum mainnet, there are a ton of accounts already, which track the balance, nonce, etc of each user/contract. The accounts themselves are however insufficient to run a node, they need to be cryptographically linked to each block so that nodes can actually verify that the accounts are not tampered with.

This cryptographic linking is done by creating a tree-like data structure, where each leaf corresponds to an account, and each intermediary level aggregates the layer below it into an ever smaller layer, until you reach a single root. This gigantic data structure containing all the accounts and the intermediate cryptographic proofs is called the state trie.

#### Why is the state trie download phase require a special syncing mode?

The trie data structure is an intricate interlink of hundreds of millions of tiny cryptographic proofs (trie nodes). To truly have a synchronized node, you need to download all the account data, as well as all the tiny cryptographic proofs to verify that no one in the network is trying to cheat you. This itself is already a crazy number of data items.

The part where it gets even messier is that this data is constantly morphing: at every block (roughly 13s), about 1000 nodes are deleted from this trie and about 2000 new ones are added. This means your node needs to synchronize a dataset that is changing 230 times per second. Until you actually do gather all the data, your local node is not usable since it cannot cryptographically prove anything about any accounts.

#### What was the default syncing mode before "snap"?

Snap syncing was introduced by version [1.10.0](https://blog.ethereum.org/2021/03/03/geth-v1-10-0/) and was adopted as the default mode in version [1.10.4](https://github.com/ethereum/go-ethereum/releases/tag/v1.10.4). Before that, the default was the "fast" syncing mode, which requested each node of the state trie one-by-one.

The snap synchronization protocol is not intended to completely replace the fast syncing mode, as it relies on a special data structure that is not inherent to the Ethereum protocol. This data structure is called a [snapshot](https://blog.ethereum.org/2020/07/17/ask-about-geth-snapshot-acceleration/), and it contains a complete view of the Ethereum state at any given block. Although this allows faster syncing, maintaining this data structure requires additional data to be computed and stored.

Continuing to maintain both the "fast" and "snap" syncing protocols allows [other Ethereum clients](https://ethereum.org/en/developers/docs/nodes-and-clients) to choose not to utilize snapshots without hindering their capacity to participate in the eth protocol. Some clients even developed other synchronization methods, for example OpenEthereum's [Warp Sync](https://openethereum.github.io/Warp-Sync).

#### I have good bandwidth, so why does downloading the state take so long when using fast sync?

State sync is mostly limited by disk IO, not bandwidth.

The state trie in Ethereum contains hundreds of millions of nodes, most of which take the form of a single hash referencing up to 16 other hashes. This is a horrible way to store data on a disk, because there's almost no structure in it, just random numbers referencing even more random numbers. This makes any underlying database weep, as it cannot optimize storing and looking up the data in any meaningful way. Snap sync solves this issue by adopting the Snapshot data structure.

#### Wait, so I can't use fast sync on an HDD?

Doing a "fast" sync on an HDD will take more time than you're willing to wait, because the data structures used are not optimized for HDDs. Even if you do wait it out, an HDD will not be able to keep up with the read/write requirements of transaction processing on mainnet. You however should be able to run a light client on an HDD with minimal impact on system resources.
