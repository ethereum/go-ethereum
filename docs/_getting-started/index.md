---
title: Getting Started with Geth
permalink: docs/getting-started
sort_key: A
---

To use Geth, you need to install it first. You can install the geth software in a variety
of ways. These include installing it via your favorite package manager; downloading a
standalone pre-built binary; running as a docker container; or building it yourself.

Head over to the [install and build](/install-and-build/Installing-Geth) section and
follow the instructions for your operating system.

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

-   **Full**: Downloads all blocks (including headers, transactions and receipts) and generates the state of the blockchain incrementally by executing every block.
-   **Fast** (Default): Downloads all blocks (including headers, transactions and receipts), verifies all headers, and downloads the state and verifies it against the headers.
-   **Light**: Downloads all block headers, block data, and verifies some randomly. 

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

You can also enter the console directly when you start the node with the `console` command:

```shell
geth --syncmode "light" console
```
