---
title: Pruning
description: Instructions for pruning a Geth node
---

<Note>Offline pruning is only for the hash-based state scheme. Soon, we will have a path-based state scheme which enables the pruning by default. Once the hash-based state scheme is no longer supported, offline pruning will be deprecated.</Note>

A snap-sync'd Geth node currently requires more than 650 GB of disk space to store the historic blockchain data. With default cache size the database grows by about 14 GB/week. This means that Geth users will rapidly run out of space on 1TB hard drives. To solve this problem without needing to purchase additional hardware, Geth can be pruned. Pruning is the process of erasing older data to save disk space. Since Geth `v1.10`, users have been able to trigger a snapshot offline prune to bring the total storage back down to the original ~650 GB in about 4-5 hours. This has to be done periodically to keep the total disk storage
within the bounds of the local hardware (e.g. every month or so for a 1TB disk).

To prune a Geth node at least 40 GB of free disk space is recommended. This means pruning cannot be used to save a hard drive that has been completely filled. A good rule of thumb is to prune before the node fills ~80% of the available disk space.

## Pruning rules {#pruning-rules}

1. Do not try to prune an archive node. Archive nodes need to maintain ALL historic data by definition.
2. Ensure there is at least 40 GB of storage space still available on the disk that will be pruned. Failures have been reported with ~25GB of free space.
3. Geth is at least `v1.10` ideally > `v1.10.3`
4. Geth is fully sync'd
5. Geth has finished creating a snapshot that is at least 128 blocks old. This is true when "state snapshot generation" is no longer reported in the logs.

With these rules satisfied, Geth's database can be pruned.

## How pruning works {#how-pruning-works}

Pruning uses snapshots of the state database as an indicator to determine which nodes in the state trie can be kept and which ones are stale and can be discarded. Geth identifies the target state trie based on a stored snapshot layer which has at least 128 block confirmations on top (for surviving reorgs) data that isn't part of the target state trie or genesis state.

Geth prunes the database in three stages:

1. Iterating state snapshot: Geth iterates the bottom-most snapshot layer and constructs a bloom filter set for identifying the target trie nodes.
2. Pruning state data: Geth deletes stale trie nodes from the database which are not in the bloom filter set.
3. Compacting database: Geth tidies up the new database to reclaim free space.

There may be a period of >1 hour during the Compacting Database stage with no log messages at all. This is normal, and the pruning should be left to run until finally a log message containing the phrase `State pruning successful` appears (i.e. do not restart Geth yet!). That message indicates that the pruning is complete and Geth can be started.

## Pruning command {#pruning-command}

For a normal Geth node, Geth should be stopped and the following command executed to start a offline state prune:

```sh
geth snapshot prune-state
```

For a Geth node run using `systemd`:

```sh
sudo systemctl stop geth # stop geth, wait >3mins to ensure clean shutdown
tmux # tmux enables pruning to keep running even if you disconnect
sudo -u <user> geth --datadir <path> snapshot prune-state # wait for pruning to finish
sudo systemctl start geth # restart geth
```

The pruning could take 4-5 hours to complete. Once finished, restart Geth.

## Troubleshooting {#troubleshooting}

Messages about "state snapshot generation" indicate that a snapshot is not fully generated. This suggests either the `--datadir` is not correct or Geth ran out of time to complete the snapshot generation and the pruning began before the snapshot was completed. In either case, the best course of action is to stop Geth, run it normally again (no pruning) until the snapshot is definitely complete and at least 128 blocks exist on top of it, then try pruning again.

## Further Reading {#further-reading}

[Ethereum Foundation blog post for Geth v1.10.0](https://blog.ethereum.org/2021/03/03/geth-v1-10-0/)

[Pruning Geth guide (@yorickdowne)](https://gist.github.com/yorickdowne/3323759b4cbf2022e191ab058a4276b2)

[Pruning Geth in a RocketPool node](https://docs.rocketpool.net/guides/node/pruning.html)
