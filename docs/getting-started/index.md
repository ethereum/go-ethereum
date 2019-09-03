---
title: Getting Started with Geth
---

## Installing

You can install the Go implementation of Ethereum in a variety of ways. These include installing it via your favorite package manager; downloading a standalone pre-built binary; running as a docker container; or building it yourself. This section highlights the common options, but you can find others in the left hand menu, or in the [install and build](/install-and-build/Installing-Geth) section.

### Install on macOS via Homebrew

You can install go-ethereum on macOS using [our Homebrew tap](https://github.com/ethereum/homebrew-ethereum). If you don't have Homebrew, [install it first](http://brew.sh/).

Then run the following commands to add the tap and install geth:

```shell
brew tap ethereum/ethereum
brew install ethereum
```

_[Read this guide](/install-and-build/Installation-Instructions-for-Mac) further Homebrew options._

### Install on Ubuntu via PPAs

You can install go-ethereum on Ubuntu-based distributions using the built-in launchpad PPAs (Personal Package Archives). We provide a single PPA repository with both our stable and our development releases for Ubuntu versions `trusty`, `xenial`, `zesty` and `artful`.

Install dependencies first:

```shell
sudo apt-get install software-properties-common
```

To enable our launchpad repository run:

```shell
sudo add-apt-repository -y ppa:ethereum/ethereum
```

After that you can install the stable version of go-ethereum:

```shell
sudo apt-get update
sudo apt-get install ethereum
```

_[Read this guide](/install-and-build/Installation-Instructions-for-Ubuntu) for further Ubuntu options._

### Install on Windows

_Although we were shipping Chocolatey packages for a time after the Frontier release, the constant manual approval process led to us stopping distribution. We will try to negotiate trusted package status for go-ethereum so the Chocolatey option is available again._

Until then grab a Windows installer from our [downloads](https://geth.ethereum.org/downloads) page.

### Download standalone binary

We distribute all our stable releases and development builds as standalone binaries. These are useful for scenarios where you'd like to: a) install a specific version of our code (e.g., for reproducible environments); b) install on machines without internet access (e.g., air gapped computers); or c) do not like automatic updates and would rather manually install software.

We create the following standalone binaries:

-   32bit, 64bit, ARMv5, ARMv6, ARMv7 and ARM64 archives (`.tar.gz`) on Linux
-   64bit archives (`.tar.gz`) on macOS
-   32bit and 64bit archives (`.zip`) and installers (`.exe`) on Windows

For all binaries we provide two options, one containing only Geth, and another containing Geth along with all the developer tools from our repository (`abigen`, `bootnode`, `disasm`, `evm`, `rlpdump`). Read our [`README`](https://github.com/ethereum/go-ethereum#executables) for more information about these executables.

To download these binaries, head to the [Go Ethereum Downloads](https://geth.ethereum.org/downloads) page.

### Run inside docker container

We maintain a Docker image with recent snapshot builds from our `develop` branch on DockerHub. In addition to the container based on Ubuntu (158 MB), there is a smaller image using Alpine Linux (35 MB). To use the alpine [tag](https://hub.docker.com/r/ethereum/client-go/tags), replace `ethereum/client-go` with `ethereum/client-go:alpine` in the examples below.

To pull the image and start a node, run these commands:

```shell
docker pull ethereum/client-go
docker run -it -p 30303:30303 ethereum/client-go
```

_[Read this guide](/install-and-build/Installation-Instructions-for-Docker) for further Docker options._

## Starting a node

### Create an account

Before starting Geth you first need to create an account that represents a key pair. Use the following command to create a new account and set a password for that account:

```shell
geth account new
```

_[Read this guide](/interface/Managing-your-accounts) for more details on importing existing Ethereum accounts and other uses of the `account` command._

### Sync modes

Running Geth starts an Ethereum node that can join any existing network, or create a new one. You can start Geth in one of three different sync modes using the `--syncmode "{mode}"` argument that determines what sort of node it is in the network.

These are:

-   **Full**: Downloads all block headers, block data, and validates all transactions
-   **Fast** (Default): Downloads block headers and block data of the most recent transactions (1024) and validates them.
-   **Light**: Downloads all block headers, block data, but does not validate transactions.

For example:

```shell
geth --syncmode "light"
```

### Connect to node

Once you have an account and Geth is running, you can interact with it by opening another terminal and using the following command to open a JavaScript console:

```shell
geth attach
```

In the console you can issue any of the Geth commands, for example, to list all the accounts on the node, use:

```shell
eth.accounts
```

<!-- TODO: Read more -->

## Next steps
