## Installing from PPA

```shell
sudo apt-get install software-properties-common
sudo add-apt-repository -y ppa:ethereum/ethereum
sudo apt-get update
sudo apt-get install ethereum
```

If you want to stay on the bleeding edge, install the `ethereum-unstable` package instead.

After installing, run `geth account new` to create an account on your node.

You should now be able to run `geth` and connect to the network.

Make sure to check the different options and commands with `geth --help`

You can alternatively install only the `geth` CLI with `apt-get install geth` if you don't want to install the other utilities (`bootnode`, `evm`, `disasm`, `rlpdump`, `ethtest`).

## Building from source

### Building Geth (command line client)

Clone the repository to a directory of your choosing:

```shell
git clone https://github.com/ethereum/go-ethereum
```
Install latest distribution of Go (v1.7) if you don't have it already:

[See instructions](https://github.com/ethereum/go-ethereum/wiki/Installing-Go#ubuntu-1404)

Building `geth` requires Go and C compilers to be installed:

```shell
sudo apt-get install -y build-essential golang
```

Finally, build the `geth` program using the following command.
```shell
cd go-ethereum
make geth
```

You can now run `build/bin/geth` to start your node.
