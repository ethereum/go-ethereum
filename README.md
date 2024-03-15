# Shisui

![AppVeyor Build](https://img.shields.io/appveyor/build/GrapeBaBa/shisui)
[![Discord](https://img.shields.io/badge/discord-join%20chat-blue.svg)](https://discord.gg/HBAgaHCBuY)

Shisui is an [Ethereum portal client](https://github.com/ethereum/portal-network-specs) written in Go language based [go-ethereum](https://github.com/ethereum/go-ethereum) and [erigon](https://github.com/ledgerwatch/erigon).
The name is inspired by Uchiha Shisui from the anime Naruto, who is renowned as "Shisui of the Body Flicker".

> **Note:** Shisui is still **under heavy development** and is not yet ready for production use.

## Building the source

For prerequisites and detailed build instructions please read the [Installation Instructions](https://geth.ethereum.org/docs/getting-started/installing-geth).

Building `shisui` requires both a Go (version 1.19 or later) and a C compiler. You can install
them using your favourite package manager. Once the dependencies are installed, run

```shell
make shisui
```

Also, you can build the docker image by running

```shell
make shisui-image
```

## Running `shisui`

After building `shisui`, you can start the client by running

```shell
./build/bin/shisui
```

Alternatively, you can run the docker image by running

```shell
docker run -d -p 8545:8545 -p 9009:9009/udp ghcr.io/optimism-java/shisui:latest
```

### Hardware Requirements

Minimum:

* CPU with 2+ cores
* 4GB RAM
* 1TB free storage space to sync the Mainnet
* 8 MBit/sec download Internet service

Recommended:

* Fast CPU with 4+ cores
* 16GB+ RAM
* High-performance SSD with at least 1TB of free space
* 25+ MBit/sec download Internet service

