---
title: Understanding Geth's dashboard
description: How to use a dashboard to understand a Geth node's performance
---

Out [dashboards page](/docs/monitoring/dashboards.md) explains how to set up a Grafana dashboard for monitoring your Geth node. This page explores the dashboard itself, explaining what the various metrics are and what they mean for the health of a node. Note that the raw data informing the dashboard can be viewed in JSON format in the browser by navigating to the ip address and port passed to `--metrics.addr` and `--metrics.port` (`127.0.0.1:6060` by default).

## What does the dashboard look like?

The Grafana dashboard looks as follows (note that there are many more panels on the actual page than in the snapshot below):

![The Grafana dashboard](/public/images/docs/grafana/dashboard.png)

Each panel in the dashboard tracks a different metric that can be used to understand some aspect of how a Geth node is behaving. There are three main categories of panel in the default dashboard: System, Network and Blockchain. The individual panels are explained in the following sections.

## What do the panels show?

### System

Panels in the System category track the impact of Geth on the local machine, including memory and CPU usage.

#### CPU

![The CPU panel](/public/images/docs/grafana/cpu.png)

The CPU panel shows how much CPU is being used as a percentage of one processing core (i.e. 100% means complete usage of one processing core, 200% means complete usage of two processing cores). There are three processes plotted on the figure. The total CPU usage by the entire system is plotted as `system`; the percentage of time that the CPUs are idle waiting for disk i/o operations is plotted as `iowait`; the CPU usage by the Geth process is plotted as `geth`.

#### Memory

![The Memory panel](/public/images/docs/grafana/memory.png)

Memory tracks the amount of RAM being used by Geth. Three metrics are plotted: the cache size, i.e. the total RAM reserved for Geth (default 1024 MB) is plotted as `held`; the amount of the cache actually being used by Geth is plotted as `used`; the memory allocations being made is plotted as `alloc`. 

#### Disk

Disk tracks the rate that data is written to (plotted as `write`) or read from (plotted as `read`) the hard disk in units of MB/s.

![The Disk panel](/public/images/docs/grafana/disk.png)

### Network

Panels in the Network category track the data flow in and out of the local node.

#### Traffic

The Traffic panel shows the rate of data ingress and egress for all subprotocols, measured in units of kB/s.

![The Traffic panel](/public/images/docs/grafana/traffic.png)

#### Peers

The Peers panel shows the number of individual peers the local node is connected to. The number of times the local node dials to find new peers and the number of times information is served from the local  node are also tracked in this panel.

![The Peers panel](/public/images/docs/grafana/peers.png)

#### ETH ingress data rate

Ingress is the process of data arriving at the local node from its peers. This panel shows the rate that data specifically using the eth subprotocol is arriving at the local node in units of kB/s (kilobytes per second). The data is subdivided into specific versions of the ETH subprotocol. Be aware that some dashboard templates might not yet include the latest subprotocol versions.

![The ETH ingress rate panel](/public/images/docs/grafana/eth-ingress-rate.png)

#### ETH egress data rate

Egress is the process of data leaving the local node and being transferred to its peers. This panel shows the rate that data specifically using the eth subprotocol is leaving the local node in units of kB/s (kilobytes per second). Be aware that some dashboard templates might not yet include the latest subprotocol versions.

![The ETH egress rate panel](/public/images/docs/grafana/eth-egress-rate.png)

#### ETH ingress traffic

Ingress is the process of data arriving at the local node from its peers. This panel shows a moment-by-moment snapshot of the amount of data that is arriving at the local node, specifically using the eth subprotocol, in units of GB (gigabytes). Be aware that some dashboard templates might not yet include the latest subprotocol versions.

![The ETH ingress traffic panel](/public/images/docs/grafana/eth-ingress-traffic.png)

#### ETH egress traffic

Egress is the process of data leaving the local node and being transferred to its peers. This panel shows a moment-by-moment snapshot of the amount of data that is leaving the local node, specifically using the eth subprotocol, in units of GB (gigabytes). Be aware that some dashboard templates might not yet include the latest subprotocol versions.

![The ETH egress traffic panel](/public/images/docs/grafana/eth-egress-traffic.png)

### Blockchain

Panels in the Blockchain category track the local node's view of the blockchain.

#### Chain head

The chain head simply tracks the latest block number that the local node is aware of.

![The Chain head panel](/public/images/docs/grafana/chain-head.png)

#### Transaction pool

Geth has a capacity for pending transactions defined by `--txpool.globalslots` (default is 5160). The number of slots filled with transactions is tracked as `slots`. The transactions in the pool are divided into pending transactions and queued transactions. Pending transactions are ready to be processed and included in a block, whereas queued transactions are those whose transaction nonces are out of sequence. Queued transactions can become pending transactions if transactions with the missing nonces become available. In the dashboard pending transactions are labelled as `executable` and queued transactions are labelled `gapped`. The subset of those global transactions that originated from the local node are tracked as `local`.

![The tx pool panel](/public/images/docs/grafana/tx-pool.png)

#### Block processing

The block processing panel tracks the time taken to complete the various tasks involved in processing each block, measured in microseconds or nanoseconds. Specifically, this includes:

- execution: time taken to execute the transactions in the block
- validation: time taken to compute a new state root and compare it to the one that arrived in the block
- commit: time taken to write the new block to the chain data
- account read: time taken to access account information from the state trie
- account update: time taken to update a leaf in the state trie
- account hash: time taken to generate a hash of an account's data
- account commit: time taken to write new account data into the state trie
- storage read: time taken to access smart contract storage data from the storage trie
- storage update: time taken to change a piece of smart contract storage data in the storage trie
- storage hash: time taken to generate a new hash for modified smart contract storage data
- storage commit: time taken to write modified smart contract storage data to the storage trie. 
- snapshot account read: time taken to read account data from a snapshot
- snapshot storage read: time taken to read storage data from a  snapshot
- snapshot commit: time taken to write data to a snapshot

![The block processing panel](/public/images/docs/grafana/block-processing.png)

#### Transaction processing

The transaction processing panel tracks the time taken to complete the various tasks involved in processing each block, measured as a mean rate of events per second:

- known: rate of new transactions arriving at the node.
- valid: rate that node marks known transactions as valid
- invalid: rate that node marks known transactions as invalid
- underpriced: rate that node marks transactions paying insufficient gas as invalid
- executable discard: rate that valid transactions are dropped from the transaction pool, e.g. because it is already known.
- executable replace: rate that valid transactions are replaced with a new one from same sender with same nonce but higher gas
- executable ratelimit: rate that valid transactions are dropped due to rate-limiting
- executable nofunds: rate that valid transations are dropped due to running out of ETH to pay gas
- gapped discard: rate that queued transactions are discarded from the transaction pool
- gapped replace: rate that queued transactions are replaced with a new one from same sender with same nonce but higher gas
- gapped ratelimit: rate that queued transactions are dropped due to rate limiting
- gapped nofunds: rate that queued transactions are dropped due to running out of ETH to pay gas

![The tx processing panel](/public/images/docs/grafana/tx-processing.png)

#### block propagation
#### transaction propagation
#### block forwarding
#### transaction fetcher peers
#### transaction fecther hashes
#### reorg meters
#### reorg total meters
#### Goroutines
#### ETh fetcher filter bodies
#### Eth fetcher filter headers
#### Data rate
#### Session totals
#### Persistent size

### Database
#### Compaction time
#### Compaction delay
#### Compaction count
