---
title: Installation instructions for FreeBSD
---
## Building from source

### Installing binary package

Binary packages tend not to be up to date (1.8.9 at the time of writing) with the latest version (1.8.16 at the time of writing). It is recommended that you use ports or compile it yourself.

```shell
pkg install go-ethereum
```

The `geth` command is then available on your system in `/usr/local/bin/geth`, you can start it e.g. on the testnet by typing:

```shell
geth -rinkeby
```

### Using ports

Go to the `net-p2p/go-ethereum` ports directory:

```shell
cd /usr/ports/net-p2p/go-ethereum
```
Then build it the standard way (as root):

```shell
make install
```

### Building Geth (command line client)

Ports are slightly more up to date (1.8.14 at the time of writing)

Clone the repository to a directory of your choosing:

```shell
git clone https://github.com/ethereum/go-ethereum
```

Building `geth` requires the Go compiler:

```shell
pkg install go
```

If your golang version is >= 1.5, build the `geth` program using the following command.
```shell
cd go-ethereum
make geth
```
If your golang version is < 1.5 (quarterly packages, for example), use the following command instead.
```shell
cd go-ethereum
CC=clang make geth
```

You can now run `build/bin/geth` to start your node.
