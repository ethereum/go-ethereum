---
title: History Pruning
description: Guide to pruning history on geth to decrease space consumption
---

Running a full Geth node consumes significant disk space over time. Much of this storage is occupied by historical block bodies and receipts that are rarely accessed. Geth provides a history pruning tool to safely remove this old history while preserving full chain validation.

## Pruning prerequisites {#prerequisites}

Before running the pruning command, you must ensure:
- Geth is fully stopped
- No other process is accessing the database

Running history pruning while Geth is online will fail and may corrupt data.

## Running the pruning command {#running}

Once Geth is stopped, run:

```sh
geth prune-history --datadir /path/to/your/geth/datadir
```

This command will scan your database and delete historical block bodies and receipts up to the merge block (i.e. historical data from the proof-of-work chain that is no longer needed for consensus).

The state trie and all headers are preserved. Your node remains fully functional for consensus.

After pruning is completed, you may continue running geth!

## Measuring the storage savings {#measuring}

Before and after pruning you can inspect your database to measure storage savings with. This should take around 30 minutes to run, and will provide a detailed measure of your storage:

```sh
geth db inspect --datadir /mnt/history_expiry_vol_data/gdata
```

Example output before pruning (mainnet):
```sh
+-----------------------+-----------------------------+------------+------------+
|       DATABASE        |          CATEGORY           |    SIZE    |   ITEMS    |
+-----------------------+-----------------------------+------------+------------+
| Key-Value store       | Headers                     | 59.92 KiB  |         90 |
| Key-Value store       | Bodies                      | 7.63 MiB   |         90 |
| Key-Value store       | Receipt lists               | 7.65 MiB   |         90 |
| Key-Value store       | Difficulties (deprecated)   | 0.00 B     |          0 |
| Key-Value store       | Block number->hash          | 3.69 KiB   |         90 |
| Key-Value store       | Block hash->number          | 889.35 MiB |   22745224 |
| Key-Value store       | Transaction index           | 13.85 GiB  |  401894661 |
| Key-Value store       | Log index filter-map rows   | 13.31 GiB  |  136688603 |
| Key-Value store       | Log index last-block-of-map | 2.79 MiB   |      61014 |
| Key-Value store       | Log index block-lv          | 45.18 MiB  |    2368590 |
| Key-Value store       | Log bloombits (deprecated)  | 0.00 B     |          0 |
| Key-Value store       | Contract codes              | 10.16 GiB  |    1673786 |
| Key-Value store       | Hash trie nodes             | 0.00 B     |          0 |
| Key-Value store       | Path trie state lookups     | 229.86 KiB |       5741 |
| Key-Value store       | Path trie account nodes     | 47.29 GiB  |  410145969 |
| Key-Value store       | Path trie storage nodes     | 180.07 GiB | 1791897413 |
| Key-Value store       | Verkle trie nodes           | 0.00 B     |          0 |
| Key-Value store       | Verkle trie state lookups   | 0.00 B     |          0 |
| Key-Value store       | Trie preimages              | 0.00 B     |          0 |
| Key-Value store       | Account snapshot            | 13.76 GiB  |  299492286 |
| Key-Value store       | Storage snapshot            | 95.46 GiB  | 1323423323 |
| Key-Value store       | Beacon sync headers         | 654.00 B   |          1 |
| Key-Value store       | Clique snapshots            | 0.00 B     |          0 |
| Key-Value store       | Singleton metadata          | 302.46 MiB |         19 |
| Ancient store (Chain) | Hashes                      | 824.28 MiB |   22745135 |
| Ancient store (Chain) | Bodies                      | 655.94 GiB |   22745135 |
| Ancient store (Chain) | Receipts                    | 253.70 GiB |   22745135 |
| Ancient store (Chain) | Headers                     | 10.93 GiB  |   22745135 |
| Ancient store (State) | History.Meta                | 442.68 KiB |       5738 |
| Ancient store (State) | Account.Index               | 61.89 MiB  |       5738 |
| Ancient store (State) | Storage.Index               | 80.13 MiB  |       5738 |
| Ancient store (State) | Account.Data                | 68.90 MiB  |       5738 |
| Ancient store (State) | Storage.Data                | 27.08 MiB  |       5738 |
+-----------------------+-----------------------------+------------+------------+
|                                    TOTAL            |  1.27 TIB  |            |
+-----------------------+-----------------------------+------------+------------+
```

Example output after pruning (mainnet):
```sh
+-----------------------+-----------------------------+------------+------------+
|       DATABASE        |          CATEGORY           |    SIZE    |   ITEMS    |
+-----------------------+-----------------------------+------------+------------+
| Key-Value store       | Headers                     | 59.92 KiB  |         90 |
| Key-Value store       | Bodies                      | 7.63 MiB   |         90 |
| Key-Value store       | Receipt lists               | 7.65 MiB   |         90 |
| Key-Value store       | Difficulties (deprecated)   | 0.00 B     |          0 |
| Key-Value store       | Block number->hash          | 3.69 KiB   |         90 |
| Key-Value store       | Block hash->number          | 889.35 MiB |   22745224 |
| Key-Value store       | Transaction index           | 13.85 GiB  |  401894661 |
| Key-Value store       | Log index filter-map rows   | 13.31 GiB  |  136688603 |
| Key-Value store       | Log index last-block-of-map | 2.79 MiB   |      61014 |
| Key-Value store       | Log index block-lv          | 45.18 MiB  |    2368590 |
| Key-Value store       | Log bloombits (deprecated)  | 0.00 B     |          0 |
| Key-Value store       | Contract codes              | 10.16 GiB  |    1673786 |
| Key-Value store       | Hash trie nodes             | 0.00 B     |          0 |
| Key-Value store       | Path trie state lookups     | 229.86 KiB |       5741 |
| Key-Value store       | Path trie account nodes     | 47.29 GiB  |  410145969 |
| Key-Value store       | Path trie storage nodes     | 180.07 GiB | 1791897413 |
| Key-Value store       | Verkle trie nodes           | 0.00 B     |          0 |
| Key-Value store       | Verkle trie state lookups   | 0.00 B     |          0 |
| Key-Value store       | Trie preimages              | 0.00 B     |          0 |
| Key-Value store       | Account snapshot            | 13.76 GiB  |  299492286 |
| Key-Value store       | Storage snapshot            | 95.46 GiB  | 1323423323 |
| Key-Value store       | Beacon sync headers         | 654.00 B   |          1 |
| Key-Value store       | Clique snapshots            | 0.00 B     |          0 |
| Key-Value store       | Singleton metadata          | 302.46 MiB |         19 |
| Ancient store (Chain) | Bodies                      | 412.23 GiB |    7207742 |
| Ancient store (Chain) | Receipts                    | 137.01 GiB |    7207742 |
| Ancient store (Chain) | Headers                     | 10.93 GiB  |    7207742 |
| Ancient store (Chain) | Hashes                      | 824.28 MiB |    7207742 |
| Ancient store (State) | History.Meta                | 442.68 KiB |       5738 |
| Ancient store (State) | Account.Index               | 61.89 MiB  |       5738 |
| Ancient store (State) | Storage.Index               | 80.13 MiB  |       5738 |
| Ancient store (State) | Account.Data                | 68.90 MiB  |       5738 |
| Ancient store (State) | Storage.Data                | 27.08 MiB  |       5738 |
+-----------------------+-----------------------------+------------+------------+
|                                    TOTAL            | 936.34 GIB |            |
+-----------------------+-----------------------------+------------+------------+
```








