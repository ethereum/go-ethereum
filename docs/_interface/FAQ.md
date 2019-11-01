---
title: FAQ
permalink: docs/faq
sort_key: C
---

**Q.**  I noticed my peercount slowly decrease, and now it is at 0.  Restarting doesn't get any peers.

**A.**  Check and sync your clock with ntp.  [Example](http://askubuntu.com/questions/254826/how-to-force-a-clock-update-using-ntp) `sudo ntpdate -s time.nist.gov`

---

**Q.**  I would like to run multiple geth instances but got the error "Fatal: blockchain db err: resource temporarily unavailable".

**A.**  Geth uses a datadir to store the blockchain, accounts and some additional information. This directory cannot be shared between running instances. If you would like to run multiple instances follow [these](../doc/setting-up-private-network-or-local-cluster) instructions.

---

**Q.** How do Ethereum syncing work?

**A.** The current default mode of sync for Geth is called fast sync. Instead of starting from the genesis block and reprocessing all the transactions that ever occurred (which could take weeks), fast sync downloads the blocks, and only verifies the associated proof-of-works. Downloading all the blocks is a straightforward and fast procedure and will relatively quickly reassemble the entire chain.

Many people falsely assume that because they have the blocks, they are in sync. Unfortunately this is not the case, since no transaction was executed, so we do not have any account state available (ie. balances, nonces, smart contract code and data). These need to be downloaded separately and cross checked with the latest blocks. This phase is called the state trie download and it actually runs concurrently with the block downloads; alas it take a lot longer nowadays than downloading the blocks.

So, what's the state trie? In the Ethereum mainnet, there are a ton of accounts already, which track the balance, nonce, etc of each user/contract. The accounts themselves are however insufficient to run a node, they need to be cryptographically linked to each block so that nodes can actually verify that the account's are not tampered with. This cryptographic linking is done by creating a tree data structure above the accounts, each level aggregating the layer below it into an ever smaller layer, until you reach the single root. This gigantic data structure containing all the accounts and the intermediate cryptographic proofs is called the state trie.

Ok, so why does this pose a problem? This trie data structure is an intricate interlink of hundreds of millions of tiny cryptographic proofs (trie nodes). To truly have a synchronized node, you need to download all the account data, as well as all the tiny cryptographic proofs to verify that noone in the network is trying to cheat you. This itself is already a crazy number of data items. The part where it gets even messier is that this data is constantly morphing: at every block (15s), about 1000 nodes are deleted from this trie and about 2000 new ones are added. This means your node needs to synchronize a dataset that is changing 200 times per second. The worst part is that while you are synchronizing, the network is moving forward, and state that you begun to download might disappear while you're downloading, so your node needs to constantly follow the network while trying to gather all the recent data. But until you actually do gather all the data, your local node is not usable since it cannot cryptographically prove anything about any accounts.

If you see that you are 64 blocks behind mainnet, you aren't yet synchronized, not even close. You are just done with the block download phase and still running the state downloads. You can see this yourself via the seemingly endless `Imported state entries [...]` stream of logs. You'll need to wait that out too before your node comes truly online.

---

**Q: The node just hangs on importing state enties?!**

**A**: The node doesn't hang, it just doesn't know how large the state trie is in advance so it keeps on going and going and going until it discovers and downloads the entire thing.

The reason is that a block in Ethereum only contains the state root, a single hash of the root node. When the node begins synchronizing, it knows about exactly 1 node and tries to download it. That node, can refer up to 16 new nodes, so in the next step, we'll know about 16 new nodes and try to download those. As we go along the download, most of the nodes will reference new ones that we didn't know about until then. This is why you might be tempted to think it's stuck on the same numbers. It is not, rather it's discovering and downloading the trie as it goes along.

---

**Q: I'm stuck at 64 blocks behind mainnet?!**

*A:* As explained above, you are not stuck, just finished with the block download phase, waiting for the state download phase to complete too. This latter phase nowadays take a lot longer than just getting the blocks.

---

**Q: Why does downloading the state take so long, I have good bandwidth?**

**A:** State sync is mostly limited by disk IO, not bandwidth.

The state trie in Ethereum contains hundreds of millions of nodes, most of which take the form of a single hash referencing up to 16 other hashes. This is a horrible way to store data on a disk, because there's almost no structure in it, just random numbers referencing even more random numbers. This makes any underlying database weep, as it cannot optimize storing and looking up the data in any meaningful way.

Not only is storing the data very suboptimal, but due to the 200 modification / second and pruning of past data, we cannot even download it is a properly pre-processed way to make it import faster without the underlying database shuffling it around too much. The end result is that even a fast sync nowadays incurs a huge disk IO cost, which is too much for a mechanical hard drive.

---

**Q: Wait, so I can't run a full node on an HDD?**

**A:** Unfortunately not. Doing a fast sync on an HDD will take more time than you're willing to wait with the current data schema. Even if you do wait it out, an HDD will not be able to keep up with the read/write requirements of transaction processing on mainnet.

You however should be able to run a light client on an HDD with minimal impact on system resources. If you wish to run a full node however, an SSD is your only option.
