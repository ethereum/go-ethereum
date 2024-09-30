# Shisui

![AppVeyor Build (with branch)](https://ci.appveyor.com/api/projects/status/github/optimism-java/shisui?branch=portal&svg=true)
[![Discord](https://img.shields.io/badge/discord-join%20chat-blue.svg)](https://discord.gg/HBAgaHCBuY)

Shisui is an [Ethereum portal client](https://github.com/ethereum/portal-network-specs) written in Go language based
on [go-ethereum](https://github.com/ethereum/go-ethereum).
The name is inspired by Uchiha Shisui from the anime Naruto, who is renowned as "Shisui of the Body Flicker".

> **Note:** Shisui is still **under heavy development** and is not yet ready for production use.

## Building the source

For prerequisites and detailed build instructions please read
the [Installation Instructions](https://geth.ethereum.org/docs/getting-started/installing-geth).

Building `shisui` requires both a Go (version 1.22 or later) and a C compiler. You can install
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

### supported options

* `--rpc.addr` HTTP-RPC server listening addr
* `--rpc.port` HTTP-RPC server listening port(default: `8545`)
* `--data.dir` data dir of where the data file located(default: `./`)
* `--data.capacity` the capacity of the data stored, the unit is MB(default: `10GB`)
* `--nat` p2p address(default `none`)
    * `none`, find local address
    * `any` uses the first auto-detected mechanism
    * `extip:77.12.33.4` will assume the local machine is reachable on the given IP
    * `upnp`               uses the Universal Plug and Play protocol
    * `pmp`                uses NAT-PMP with an auto-detected gateway address
    * `pmp:192.168.0.1`    uses NAT-PMP with the given gateway address
* `--udp.addr` protocol UDP server listening port(default: `9009`)
* `--loglevel` loglevel of portal network, `1` to `5`, from `error` to `trace`(default: `1`)
* `--private.key` private key of p2p node, hex format without `0x` prifix
* `--bootnodes` bootnode of p2p network with ENR format, use `none` to config empty bootnodes
* `--networks` portal sub networks: history, beacon, state

all the options above can be set with envs.

the env is prefixed with `SHISUI` and change the `.` to `_`.

eg `--rpc.add` can be replaced with env `SHISUI_RPC_ADDR`

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

