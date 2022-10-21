---
title: Hardware requirements
sort-key: M
---

The hardware requirements for running a Geth node depend upon the node configuration and can change over time as upgrades to the network are implemented. Ethereum nodes can be run on low power, resource-constrained devices such as Raspberry Pi's. Prebuilt, dedicated staking machines are available from several companies - these might be good choices for users who want plug-and-play hardware specifically designed for Ethereum. However, many users will choose to run nodes on laptop or desktop computers. 

## Processor

It is preferable to use a quad-core (or dual-core hyperthreaded) CPU. Geth is released for a wide range of architectures.

## Memory

It is recommended to use at least 16GB RAM.

## Disk space

Disk space is usually the primary bottleneck for node operators. At the time of writing (September 2022) a 2TB SSD is recommended for a full node running Geth and a consensus client. Geth itself requires >650GB of disk space for a snap-synced full node and, with the default cache size, grows about 14GB/week. Pruning brings the total storage back down to the original 650GB.
Archive nodes require additional space. A "full" archive node that keeps all state back to genesis requires more than 12TB of space. Partial archive nodes can also be created by turning off the garbage collector after some initial sync - the storage requirement depends how much state is saved.


As well as storage capacity, Geth nodes rely on fast read and write operations. This means HDDs and cheaper HDDS can sometimes struggle to sync the blockchain. A list of SSD models that users report being able and unable to sync Geth is available in this [Github Gist](https://gist.github.com/yorickdowne/f3a3e79a573bf35767cd002cc977b038). Please note that the list has *not* been verified by the Geth team.


## Bandwidth

It is important to have a stable and reliable internet connection, especially for running a validator because downtime can result in missed rewards or penalties. It is recommended to have at least 25Mbps download speed to run a node. Running a node also requires a lot of data to be uploaded and downloaded so it is better to use an ISP that does not have a capped data allowance.