---
title: Sync-modes
sort-key: L
---

Syncing is the process by which Geth catches up to the latest Ethereum block and current global state. 
There are several ways to sync a Geth node that differ in their speed, storage requirements and trust 
assumptions. Now that Ethereum uses proof-of-stake based consensus, a consensus client is required for 
Geth to sync. 

## Full nodes

There are two types of full node that use different mechanisms to sync up to the head of the chain:

### Snap (default)

A snap sync'd node holds the most recent 128 block states in memory, so transactions in that range are always quickly 
accessible. However, snap-sync only starts processing from a relatively recent block (as opposed to genesis 
for a full node). Between the initial sync block and the 128 most recent blocks, the node stores occasional 
checkpoints that can be used to rebuild the state on-the-fly. This means transactions can be traced back as 
far as the block that was used for the initial sync. Tracing a single transaction requires reexecuting all 
preceding transactions in the same block **and** all preceding blocks until the previous stored snapshot. 
Snap-sync'd nodes are therefore full nodes, with the only difference being the initial synchronization required 
a checkpoint block to sync from instead of independently verifying the chain all the way from genesis. 
Snap sync then only verifies the proof-of-work and ancestor-child block progression and assumes that the 
state transitions are correct rather than re-executing the transactions in each block to verify the state 
changes. Snap sync is much faster than block-by-block sync. To start a node with snap sync pass `--syncmode snap` at 
startup.

Snap sync starts by downloading the headers for a chunk of blocks. Once the headers have been verified, the block
bodies and receipts for those blocks are downloaded. In parallel, Geth also sync begins state-sync. In state-sync, Geth first downloads the 
leaves of the state trie for each block without the intermediate nodes along with a range proof. The state trie is 
then regenerated locally. The state download is the part of the snap-sync that takes the most time to complete 
and the progress can be monitored using the ETA values in the log messages. However, the blockchain is also 
progressing at the same time and invalidating some of the regenerated state data. This means it is also necessary 
to have a 'healing' phase where errors in the state are fixed. It is not possible to monitor the progress of 
the state heal because the extent of the errors cannot be known until the current state has already been regenerated.

Geth regularly reports `Syncing, state heal in progress` during state heal - this informs the user that 
state heal has not finished. It is also possible to confirm this using `eth.syncing` - if this command 
returns `false` then the node is in sync. If it returns anything other than `false` then syncing is still in progress. 


```sh
# this log message indicates that state healing is still in progress
INFO [10-20|20:20:09.510] State heal in progress                   accounts=313,309@17.95MiB slots=363,525@28.77MiB codes=7222@50.73MiB nodes=49,616,912@12.67GiB pending=29805
```

```sh
# this indicates that the node is in sync, any other response indicates that syncing has not finished
eth.syncing
>> false
```

The healing has to outpace the growth of the blockchain, otherwise the node will never catch up to the current state. 
There are some hardware factors that determine the speed of the state healing (speed of disk read/write and internet 
connection) and also the total gas used in each block (more gas means more changes to the state that have to be 
handled).

To summarize, snap sync progresses in the following sequence:
- download and verify headers
- download block bodies and receipts.In parallel, download raw state data and build state trie
- heal state trie to account for newly arriving data

**Note** Snap sync is the default behaviour, so if the `--syncmode` value is not passed to Geth at startup, 
Geth will use snap sync. A node that is started using `snap` will switch to block-by-block sync once it has 
caught up to the head of the chain.

### Full

A full sync generates the current state by executing every block starting from the genesis block. A full sync 
indendently verifies proof-of-work and block provenance as well as all state transitions by re-executing the 
transactions in the entire historical sequence of blocks. Only the most recent 128 block states are stored in a full 
node - older block states are pruned periodically and represented as a series of checkpoints from which any previous 
state can be regenerated on request. 128 blocks is about 25.6 minutes of history with a block time of 12 seconds. 
To create a full node pass `--syncmode full` at startup.

## Archive nodes

An archive node is a node that retains all historical data right back to genesis. There is no need to regenerate 
any data from checkpoints because all data is directly available in the node's own storage. Archive nodes are 
therefore ideal for making fast queries about historical states. At the time of writing (September 2022) a full 
archive node that stores all data since genesis occupies nearly 12 TB of disk space (keep up with the current 
size on [Etherscan](https://etherscan.io/chartsync/chainarchive)). Archive nodes are created by configuring Geth's 
garbage collection so that old data is never deleted: `geth --syncmode full --gcmode archive`. 

It is also possible to create a partial/recent archive node where the node was synced using `snap` but the state 
is never pruned. This creates an archive node that saves all state data from the point that the node first syncs. 
This is configured by starting Geth with `--syncmode snap --gcmode archive`.

## Light nodes

A light node syncs very quickly and stores the bare minimum of blockchain data. Light nodes only process block
headers, not entire blocks. This greatly reduces the computation time, storage and bandwidth required relative to a 
full node. This means light nodes are suitable for resource-constrained devices and can catch up to the head of the
chain much faster when they are new or have been offline for a while. The trade-off is that light nodes rely heavily 
on data served by altruistic full nodes. A light client can be used to query data from Ethereum and submit transactions, 
acting as a locally-hosted Ethereum wallet. However, because they don't keep local copies of the Ethereum state, light 
nodes can't validate blocks in the same way as full nodes - they receive a proof from the full node and verify it against their local header chain. 
To start a node in light mode, pass `--syncmode light`. Be aware that full nodes serving light data are relative scarce 
so light nodes can struggle to find peers. **Light nodes are not currently working on proof-of-stake Ethereum**.

Read more about light nodes on our [LES page](/docs/interface/les.md).

## Consensus layer syncing

Now that Ethereum has switched to proof-of-stake, all consensus logic and block propagation is handled by consensus clients. 
This means that syncing the blockchain is a process shared between the consensus and execution clients. Blocks are 
downloaded by the consensus client and verified by the execution client. In order for Geth to sync, it requires a header from
its connected consensus client. Geth does not import any data until it is instructed to by the consensus client. 
**Geth cannot sync without being connected to a consensus client**. This includes block-by-block syncing from genesis.

Once a header is available to use as a syncing target, Geth retrieves all headers between that target header and the 
local header chain in reverse chronological order. These headers show that the sequence of blocks is correct because
the parenthashes link one block to the next right up to the target block. Eventually, the sync will reach a block held 
in the local database, at which point the local data and the target data are considered 'linked' and there is a very high 
chance the node is syncing the correct chain. The block bodies are then downloaded and then the state data. The consensus
client can update the target header - as long as the syncing outpaces the growth of the blockchain then the node will eventually
get in sync.

There are two ways for the consensus client to find a block header that Geth can use as a sync target: optimistic syncing and 
checkpoint syncing:

### Optimistic sync

Optimistic sync downloads blocks before the execution client has validated them. In optimistic sync the node assumes 
the data it receives from its peers is correct during the downloading phase but then retroactively verifies each 
downloaded block. Nodes are not allowed to attest or propose blocks while they are still 'optimistic' because they 
can't yet guarantee their view of the head of the chain is correct.

Read more in the [optimistic sync specs](https://github.com/ethereum/consensus-specs/blob/dev/sync/optimistic.md).

### Checkpoint sync

Alternatively, the consensus client can grab a checkpoint from a trusted source which provides a target state to sync 
up to, before switching to full sync and verifying each block in turn. In this mode, the node trusts that the checkpoint 
is correct. There are many possible sources for this checkpoint - the gold standard would be to get it out-of-band 
from another trusted friend, but it could also come from block explorers or [public APIs/web apps](https://eth-clients.github.io/checkpoint-sync-endpoints/).

Please see the pages on [syncing](/docs/interface/sync-modes.md) for more detail. For troubleshooting, 
please see the `Syncing` section on the [console log messages](/docs/interface/logs.md) page.

**Note** it is not currently possible to use a Geth light node as an execution client on proof-of-stake Ethereum.

## Summary

There are several ways to sync a Geth node. The default is to use snap sync to create a full node. This verifies all 
blocks using some recent block that is old enough to be safe from re-orgs as a sync target. A trust-minimized alternative 
is full-sync, which verifies every block since genesis. These modes drop state data older than 128 blocks, keeping only 
checkpoints that enable on-request regeneration of historical states. For rapid queries of historical data an archive node 
is required. Archive nodes keep local copies of all historical data right back to genesis - currently about 12 TB and growing. 
The opposite extreme is a light node that doesn't store any blockchain data - it requests everything from full nodes. 
These configurations are controlled by passing `full`, `snap` or `light` to `--syncmode` at startup. For an archive node,
`--syncmode` should be `full` and `--gcmode` should be set to `archive`. Currently, due to the transition to proof-of-stake, 
`syncmode=light` does not work (new light client protocols are being developed). 
