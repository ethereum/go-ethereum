---
title: API Reference

language_tabs: # must be one of https://git.io/vQNgJ
  - shell

toc_footers:
  - <a href='https://github.com/lord/slate'>Documentation Powered by Slate</a>

includes:
  - errors

search: true
---

# Introduction

### A Note From The Developers

Our goal was to avoid compromising the integrity of Geth and simply extend existing functionality to meet the specific needs of the Shyft Network. To our utmost ability we have documented, within the codebase, exactly where we have extended our functionality using the following notation:  NOTE:SHYFT. This document is meant to provide a high level overview of the changes made to Geth and to provide explanations, where needed, on the changes that were made. Another benefit of this document is to allow others to quickly see the changes that were made in order to get quicker feedback on a compromising line of code.

### Contributing To Shyft Geth

In order to successfully accept a PR the maintainers of the Shyft repositories require that this document must be updated, reflecting the changes made in the PR. Along with the documentation, we ask that contributors provide the NOTE:SHYFT. The tag could should contain a brief on the modified code. This will help with releases further down the road as we document what breaking changes have been made along the journey.

# Setup

### Build GETH

Before running any CLI options ensure you run `make geth` in the root directory.

### CLI

> In the root directory run `./shyft-geth.sh` with any of the following flags:

```shell
--setup              - Setups postgres and the shyft chain db.
--start              - Starts geth.
--reset              - Drops postgres and chain db, and instantiates both.
--js <web3 filename> - Executes web3 calls with a passed file name.
                       If the file name is sendTransactions.js:
                       ./shyft-geth.sh --js sendTransactions
```

For convenience a simple CLI was built using `shyft-geth.sh` as the executable file with some basic commands to get your environment setup.

This will create a new database for geth to use as well as all the necessary tables for the shyft blockexplorer.

### Govendor and Packages/Dependencies

> Download Go Vendor

```shell
go get -u github.com/kardianos/govendor
```

> To run govendor globally, have this in your bash_profile file:

```shell
export GOPATH=$HOME/go
export PATH=$PATH:$HOME/go/bin
```

> Then shyft_geth will need to be cloned to this directory:

```shell
$GOPATH/src/github.com/ethereum/go-ethereum
```

Geth uses govendor to manage packages/dependencies: [Go Vendor](https://github.com/kardianos/govendor)

This has some more information: [Ethereum Wiki](https://github.com/ethereum/go-ethereum/wiki/Developers'-Guide)

To add a new dependency, run govendor fetch <import-path> , and commit the changes to git. Then the deps will be accessible on other machines that pull from git.

<aside class="notice">
GOPATH is not strictly necessary however, for govendor it is much easier to use gopath as go will look for binaries in this directory ($GOPATH/bin). To set up GOPATH, read the govendor section.
</aside>

# Custom Shyft Constants

### Block Rewards

> ./consensus/ethash/consensus.go

```go
```

Shyft inflation is different than that of Ethereum, therefore the constants were changed in order to support this.

# Shyft Extended Functionality

## Database Functions

> ./eth/backend.go

```go
```

### Database instanitation

The local database is instantiated in the same function where geth instantiates the merkel tri database.

> ./core/blockchain.go

```go
```

### Writing Blocks

In our case, we use `WriteBlock()` to store all our data. So far, it contains all the data that we need to store to our local block explorer database. This may change in the future.

## Transaction Helper Functions

> ./core/types/transaction.go

```go
```

The existing transaction type in Geth did not allow the evm to call a helper function to retrieve the from address, essentially the sender. Therefore, we extended the functionality of the Transaction type to generate the from address through `*Transaction.From()`.

>  ./shyftdb/shyft_database_util.go

```go
```

## Database Getters And Setters

In order to store the block explorer database, a custom folder was created `./shyftdb` that contains all the necessary functions to read and write to the explorer database.

The main functions exist in `./shyftdb/shyft_database_util.go`.
