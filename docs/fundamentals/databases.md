---
title: Databases
description: Overview of Geth's database architecture
---

Since v1.9.0, Geth has divided its database into two parts. Recent blocks and state data are kept in quick-access storage, but older blocks and receipts ("ancients") are stored in a "freezer" database. The point of this separation is to minimize the dependency on expensive, sensitive SSDs, and instead push the less frequently-accessed data into a database that can be supported by cheaper and more durable drives. Storing less data in the faster LevelDB database also enables faster compactions and improves the database performance by allowing more state trie nodes to be held in active memory for a given cache-size.

# Recent blocks

Geth stores recent blocks in a LevelDB database. This is a persistent key-value store that can be queried very quickly. The LevelDB database is supposed to be run on top of a fast SSD hard disk so that the disk IO is not bottlenecked by the underlying hardware. In addition to basic storage, the LevelDB database supports batch writes and iterations over the keyspace in binary-alphabetical order.
The database is periodically compacted to reduce the operational cost of accessing indivdual items. This is achieved by flattening the underlying data store for a given range of keys. Any deleted or overwritten items in that key range are removed and the surviving data is reorganized for efficiency.

Geth also tracks several performance metrics for the LevelDB database that can be monitored via the metrics subsystem. These are:

| meter                | function                                                                |
| -------------------- | ----------------------------------------------------------------------- |
| `compTimeMeter`      | Meter for measuring the total time spent in database compaction         |
| `compReadMeter`      | Meter for measuring the data read during compaction                     |
| `compWriteMeter`     | Meter for measuring the data written during compaction                  |
| `writeDelayNMeter`   | Meter for measuring the write delay number due to database compaction   |
| `writeDelayMeter`    | Meter for measuring the write delay duration due to database compaction |
| `diskSizeGauge`      | Gauge for tracking the size of all the levels in the database           |
| `diskReadMeter`      | Meter for measuring the effective amount of data read                   |
| `diskWriteMeter`     | Meter for measuring the effective amount of data written                |
| `memCompGauge`       | Gauge for tracking the number of memory compaction                      |
| `level0CompGauge`    | Gauge for tracking the number of table compaction in level0             |
| `nonlevel0CompGauge` | Gauge for tracking the number of table compaction in non0 level         |
| `seekCompGauge`      | Gauge for tracking the number of table compaction caused by read opt    |

## Freezer/ancients

Older segments of the chain are moved out of the LevelDB database and into a freezer database. Nodes rarely need to access these files so IO speed is less important and the bulk of the chain data can be stored on a cheaper HDD. Once blocks pass some threshold age (90,000 blocks behind the head by default) the block and receipt data is flattened and saved as a raw binary blob of data along with an index entry file used for identification.

Geth also tracks some basic metrics relating to the ancients database that can be monitored:

| metric       | function                                                   |
| ------------ | ---------------------------------------------------------- |
| `readMeter`  | Meter for measuring the effective amount of data read      |
| `writeMeter` | Meter for measuring the effective amount of data written   |
| `sizeGauge`  | Gauge for tracking the combined size of all freezer tables |

The ancients data is saved entirely separately from the fast-access recent data, meaning it can be stored in a different location. The default location for the ancient chain segments is inside the `chaindata` directory, which is inside `datadir`, but it can be defined by passing `--datadir.ancient <path>` to Geth on startup. The freezer is designed to have a read operation complexity of O(1), involving only a read for index items (6 bytes) and a read for the data. This design makes the freezer performant enough to run on a slow HDD disk, permitting people to run Ethereum nodes without requiring a huge SSD. The ancient data can also be moved later by manually copying the directory to a new location and then starting Geth passing the new path to `--datadir.ancient`.

## Using the freezer

If Geth cannot find the freezer, either because the database was deleted or because Geth received an incorrect path, then the node becomes unstable. It is explicitly forbidden to start Geth with an invalid path to the freezer. However, if the state database goes missing Geth can rebuild all its indices based upon data from the freezer and then do a snap-sync on top of it to fill in the missing state data.

This can be used to deliberately clean up a node. Passing `--datadir --removedb` starts the process. Geth will ask whether it should delete the state database, the ancient database and the LES database. Choosing to delete the state database only leaves the block bodies, receipts, headers etc intact in the freezer, meaning rebuilding the state will not include re-downloading ~400GB of data from the network. Geth will then rebuild the state from the freezer reusing that existing block and receipt data. In doing so, unused data and accumulated junk data will be pruned from the state database. This process can take an hour or more.

## Unclean shutdowns

If Geth stops unexpectedly the database can be corrupted. This is known as an "unclean shutdown" and it can lead to a variety of problems for the node when it is restarted. It is always best to shut down Geth gracefully, i.e. using a shutdown command such as `ctrl-c`, `docker stop -t 300 <container ID>` or `systemctl stop` (although please note that `systemctl stop` has a default timeout of 90s - if Geth takes longer than this to gracefully shut down it will quit forcefully. Update the `TimeoutSecs` variable in `systemd.service` to override this value to something larger, at least 300s). This way, Geth knows to write all relevant information into the database to allow the node to restart properly later. This can involve >1GB of information being written to the LevelDB database which can take several minutes.

If an unexpected shutdown does occur, the `removedb` subcommand can be used to delete the state database and resync it from the ancient database. This should get the database back up and running.

## Pebble

It is possible to configure Geth to use [Pebble](https://www.cockroachlabs.com/blog/pebble-rocksdb-kv-store/) instead of LevelDB for recent data. The main reason to include the alternative database is that the Go implementation of LevelDB is not actively maintained; Pebble is an actively maintained replacement that should offer a better long term option. There is no urgent reason to switch the database type yet - LevelDB works well and Pebble is still being evaluated for potentially replacing LevelDB as the default option some time in the future. However, if you wish to experiment with running Geth with Pebble, add the following flag to your Geth startup commands:

```sh
--db.engine=pebble
```

This also requires resyncing from scratch in a fresh data directory, because if an existing LevelDB database is detected on startup, Geth will default to using that, overriding the `--db.engine=pebble` flag. Pebble only works on 64 bit systems.
