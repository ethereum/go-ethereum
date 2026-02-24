---
title: Archive mode
description: Running an Archive Node in Geth
---

Geth supports two types of archive nodes that allow querying historical Ethereum state: hash-based and path-based. This document outlines both approaches.

## Hash-Based Archive Node (Legacy)

A hash-based archive node stores the entire historical Ethereum state using Merkle Patricia Tries. This method retains every account and storage slot including all corresponding trie nodes, at every block. As a result, it fully supports merkle proofs (via `eth_getProof`) and state access for all historical blocks.

However, synchronizing a hash-based archive node from genesis is a resource-intensive process that may take months. The challenges arise from the significant database compaction overhead, the database size for Ethereum mainnet can exceed 20TB.

It is worth noting that enabling hash-based archive mode isn't always required. You can switch to archive mode at any point, after which Geth will preserve all subsequent historical data. This flexibility unlocks the potential to build the archive node "cluster", where different nodes are responsible for maintaining different segments of the historical state.

```sh
#!/bin/bash
geth --state.scheme=hash --gcmode=archive --syncmode=full
```

## Path-Based Archive Node (Recommended)

Geth v1.16.0 introduces a new approach to archive nodes using path-based state storage. This design is significantly more efficient in disk usage and allows flexible control over how many historical states to retain.

### Key Advantages

- **Low disk usage**: an archive node on Ethereum mainnet with full flat state history requires around 2 TB of storage. If it stores full flat states alongside the historical trie data, the requirement increases to approximately 6.5 TB.
- **Configurable retention**: users can set how many historical states to keep
- **Faster boostrap**: synchronizing a path-based archive node takes around 2 weeks
- **HDD support**: the state histories can be placed even on the HDD, which is more friendly to home operators

Notably, in archive mode on v1.16.x, historical Merkle proofs via `eth_getProof` are not supported, as this requires storing historical trie nodes. Starting from v1.17.x, `eth_getProof` for historical blocks is supported if `history.trienode = N` is configured, which enables retention of the required historical trie nodes.

### How to Run a Path-Based archive node

```sh
#!/bin/bash

geth \
    --history.state=0 \
    --gcmode archive \
    --syncmode full \
```

`--history.state=0` is the key flag. When combined with `--syncmode=full` to perform a full sync from genesis, Geth will retain all the historical states.

`--gcmode archive` enables historical state indexing. Historical states become accessible only after indexing is fully completed.

`--datadir.ancient` can be set if you want to place the state history on a cheaper device (e.g., HDD).

`--history.state=N` can be set if you only want to retain a limited number of recent states. Older state data will be pruned automatically. Note: Once pruned, historical state cannot be recovered, so choose this value carefully.

**Historical Trie Node Retention**

`--history.trienode=N` controls whether historical trie nodes are retained. By default, `--history.trienode=-1`, which disables retention of historical trie data.

If you want to support historical Merkle proofs (e.g., via `eth_getProof` for historical blocks), you must explicitly set `--history.trienode=N` to retain trie node history. Without this flag, historical flat states are available, but historical trie data (and therefore historical proofs) are not.

Notably, `--gcmode archive` does not need to be enabled during the initial full sync. In fact, enabling it from the start may slow down synchronization. For better efficiency, we recommend completing the full sync first, and then enabling `--gcmode archive` afterward for historical state indexing.

One final note: make sure that historical states have been fully indexed by checking the `eth_syncing` endpoint.

```sh
Welcome to the Geth JavaScript console!

instance: Geth/v1.15.12-unstable-4c47b22f-20250626/linux-amd64/go1.24.2
at block: 22788923 (Thu Jun 26 2025 21:29:47 GMT+0800 (CST))
datadir: /home/gary/mount/geth
modules: admin:1.0 debug:1.0 engine:1.0 eth:1.0 miner:1.0 net:1.0 rpc:1.0 txpool:1.0 web3:1.0

To exit, press ctrl-d or type exit
> eth.syncing
false
```

### How it works

With 'path-based' state storage, Geth keeps exactly one full state in the database. Specifically, the persistent state is 128 blocks in the past. For newer blocks up to the head, forward diffs are kept in memory. In order to support rolling back to blocks older than head-128, Geth also keeps 'state history' in the form of reverse key-value diffs. When the chain is reset to an old block, these diffs are applied to the persistent state, going backwards one diff at a time until the target block is reached.

A reverse state diff contains the previous values of all modified accounts and storage locations for a specific block. There is a reverse diff for each historical block. This makes reverse diffs suitable for storage into the 'freezer', i.e. they do not need to live within Pebble/LevelDB.

The new archive mode is built on the realization that reverse state diffs contain all necessary data to serve historical state queries. For example, in order to get the historical balance of an account X at block B, we can search forward through diffs starting at block B until we find a block where account X is modified. This diff will contain the balance of the account, since it stores the all modified pre-values.

To accelerate the search for a suitable diff, Geth creates a database index storing the block numbers in which an account was modified. This index is small compared to the overall state history, but it is crucial for correct operation of the archive node. The state index is stored in PebbleDB and will be generated automatically while geth is syncing the chain. It takes ~30h to build the archive state index for mainnet, and historical state will only be available when the index is fully built. Geth will report a syncing status through `eth_syncing` while the indexing happens.

### Database inspection on Ethereum mainnet

```sh
+-----------------------+-----------------------------+------------+------------+
|       DATABASE        |          CATEGORY           |    SIZE    |   ITEMS    |
+-----------------------+-----------------------------+------------+------------+
| Key-Value store       | Headers                     | 84.63 KiB  |        127 |
| Key-Value store       | Bodies                      | 11.44 MiB  |        127 |
| Key-Value store       | Receipt lists               | 10.64 MiB  |        127 |
| Key-Value store       | Difficulties (deprecated)   | 0.00 B     |          0 |
| Key-Value store       | Block number->hash          | 5.21 KiB   |        127 |
| Key-Value store       | Block hash->number          | 891.06 MiB |   22788924 |
| Key-Value store       | Transaction index           | 96.96 GiB  | 2865039365 |
| Key-Value store       | Log index filter-map rows   | 13.34 GiB  |  137082712 |
| Key-Value store       | Log index last-block-of-map | 2.80 MiB   |      61163 |
| Key-Value store       | Log index block-lv          | 45.16 MiB  |    2367672 |
| Key-Value store       | Log bloombits (deprecated)  | 0.00 B     |          0 |
| Key-Value store       | Contract codes              | 10.29 GiB  |    1712766 |
| Key-Value store       | Hash trie nodes             | 0.00 B     |          0 |
| Key-Value store       | Path trie state lookups     | 890.98 MiB |   22786783 |
| Key-Value store       | Path trie account nodes     | 47.47 GiB  |  411692001 |
| Key-Value store       | Path trie storage nodes     | 180.44 GiB | 1795510203 |
| Key-Value store       | Path state history indexes  | 297.23 GiB | 4124368811 |
| Key-Value store       | Verkle trie nodes           | 0.00 B     |          0 |
| Key-Value store       | Verkle trie state lookups   | 0.00 B     |          0 |
| Key-Value store       | Trie preimages              | 524.04 MiB |    7758083 |
| Key-Value store       | Account snapshot            | 13.81 GiB  |  300568494 |
| Key-Value store       | Storage snapshot            | 95.66 GiB  | 1326130982 |
| Key-Value store       | Beacon sync headers         | 52.82 KiB  |         83 |
| Key-Value store       | Clique snapshots            | 0.00 B     |          0 |
| Key-Value store       | Singleton metadata          | 373.19 MiB |         16 |
| Ancient store (Chain) | Receipts                    | 254.76 GiB |   22788798 |
| Ancient store (Chain) | Headers                     | 10.95 GiB  |   22788798 |
| Ancient store (Chain) | Hashes                      | 825.86 MiB |   22788798 |
| Ancient store (Chain) | Bodies                      | 657.81 GiB |   22788798 |
| Ancient store (State) | History.Meta                | 1.68 GiB   |   22786782 |
| Ancient store (State) | Account.Index               | 142.04 GiB |   22786782 |
| Ancient store (State) | Storage.Index               | 204.55 GiB |   22786782 |
| Ancient store (State) | Account.Data                | 142.15 GiB |   22786782 |
| Ancient store (State) | Storage.Data                | 50.63 GiB  |   22786782 |
+-----------------------+-----------------------------+------------+------------+
|                                    TOTAL            |  2.17 TIB  |            |
+-----------------------+-----------------------------+------------+------------+
```
