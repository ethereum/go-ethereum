---
title: Sync modes
description: Introduction to the three ways to sync Geth
---

Syncing is the process by which Geth catches up to the latest Ethereum block and current global state. There are several ways to sync a Geth node that differ in their speed, storage requirements and trust assumptions. This page outlines three sync configurations for full nodes and one for light nodes.

## Full nodes

There are three types of full node with different sync modes:

### Full

A full sync generates the current state by executing every block starting from the genesis block. A full sync is the trust-minimized option since there it does not rely upon any information from any trusted source apart from the genesis configuration. All other information is indendently verified as the node re-executes the entire historical sequence of blocks. Only the most recent 128 blocks are stored in a full node - older blocks are pruned periodically and represented as a series of checkpoints from which any previous state can be regenerated on request. 128 blocks is about 25.6 minutes of history with a block time of 12 seconds. To create a full node pass `--syncmode full` at startup.

### Archive

An archive node is a full-sync'd node that retains all historical data right back to genesis. There is no need to regenerate any data from checkpoints because all data is directly available in the node's own storage. Archive nodes are therefore ideal for making fast queries about historical states. At the time of writing (August 2022) an archive node occupies nearly 12 TB of disk space (keep up with the current size on [Etherscan](https://etherscan.io/chartsync/chainarchive)). Archive nodes are created by configuring Geth's garbage collection so that old data is never deleted: `geth --syncmode full --gcmode archive`.

### Snap

A snap sync'd node holds the most recent 128 blocks in memory, so transactions in that range are always accessible. However, snap-sync only starts processing from a relatively recent block (as opposed to genesis for a full node). Between the initial sync block and the 128 most recent blocks, the node stores occasional checkpoints that can be used to rebuild the state on-the-fly. This means transactions can be traced back as far as the block that was used for the initial sync. Tracing a single transaction requires reexecuting all preceding transactions in the same block **and** all preceding blocks until the previous stored snapshot. Snap-sync'd nodes are therefore functionally equal to full nodes, but the initial synchronization required some trusted information (a checkpoint block) to sync from instead of independently verifying the chain all the way from genesis. Snap sync is much faster than full sync. To start a node with snap sync pass `--syncmode snap` at startup.

## Light nodes

A light node syncs very quickly and stores the bare minimum of blockchain data. Light nodes only process block headers, not entire blocks. This greatly reduces the computation time, storage and bandwidth required relative to a full node. This means light nodes are suitable for resource-constrained devices and can catch up to the head of the chain much faster when they are new or have been offline for a while. The trade-off is that light nodes rely heavily on data served by altruistic full nodes. A light client can be used to query data from Ethereum and submit transactions, acting as a locally-hosted Ethereum wallet. However, because they don't keep local copies of the Ethereum state, light nodes can't validate blocks in the same way as full nodes - they have to trust that the data they receive is honest. To start a node in light mode, pass `--syncmode light`. Be aware that full nodes serving light data are relative scarce so light nodes can struggle to find peers.

Read more about light nodes on our [LES page](/content/docs/fundamentals/les.md).

{% include note.html content="Light nodes do not currently work on proof-of-stake Ethereum, but they are expected to ship soon!" %}

## Summary

There are several ways to sync a Geth node. The default is to use snap sync to create a full node. This verifies all blocks starting at a recent checkpoint. A trust-minimized alternative is full-sync, which verifies every block since genesis. These modes prune the blockchain data older than 128 blocks, keeping only checkpoints that enable on-request regeneration of historical states. For rapid queries of historical data an archive node is required. Archive nodes keep local copies of all historical data right back to genesis - currently about 12 TB and growing. The opposite extreme is a light node that doesn't store any blockchain data - it requests everything from full nodes. These configurations are controlled by passing `full`, `snap` or `light` to `--syncmode` at startup. For an archive node, `--syncmode` should be `full` and `--gcmode` should be set to `archive`.
