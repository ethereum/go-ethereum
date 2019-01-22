---
title: Installation instructions for Mac
---
## Installing with Homebrew

By far the easiest way to install go-ethereum is to use our
Homebrew tap. If you don't have Homebrew, [install it first](http://brew.sh).

Then run the following commands to add the tap and install `geth`:

```shell
brew tap ethereum/ethereum
brew install ethereum
```

You can install the develop branch by running `--devel`:

```shell
brew install ethereum --devel
```

After installing, run `geth account new` to create an account on your node.

You should now be able to run `geth` and connect to the network.

Make sure to check the different options and commands with `geth --help`

For options and patches, see: https://github.com/ethereum/homebrew-ethereum

## Building from source

### Building Geth (command line client)

Clone the repository to a directory of your choosing:

```shell
git clone https://github.com/ethereum/go-ethereum
```

Building `geth` requires the Go compiler:

```shell
brew install go
```

Finally, build the `geth` program using the following command.
```shell
cd go-ethereum
make geth
```

If you see some errors related to header files of Mac OS system library, install XCode Command Line Tools, and try again.

```shell
xcode-select --install
```

You can now run `build/bin/geth` to start your node.
