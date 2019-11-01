---
title: Dev mode
sort_key: B
---

Geth has a development mode which sets up a single node Ethereum test network with a number of options optimized for developing on local machines. You enable it with the `--dev` argument.

Starting geth in dev mode does the following:

-   Initializes the data directory with a testing genesis block
-   Sets max peers to 0
-   Turns off discovery by other nodes
-   Sets the gas price to 0
-   Uses the Clique PoA consensus engine with which allows blocks to be mined as-needed without excessive CPU and memory consumption
-   Uses on-demand block generation, producing blocks when there are transactions waiting to be mined

You can specify a data directory to maintain state between runs using the `--datadir` option, otherwise databases are ephemeral and in-memory:

```shell
$ mkdir test-chain-dir
$ geth --dev --datadir test-chain-dir console
```

Once geth is running in dev mode, you can interact with it in the same way as when geth is running in other ways.

For example, create a test account:

```shell
> personal.newAccount()
```

Then transfer ether from the coinbase to the new account:

```shell
> eth.sendTransaction({from:eth.coinbase, to:eth.accounts[1], value: web3.toWei(0.05, "ether")})
```

And check the balance of the account:

```shell
> eth.getBalance(eth.accounts[1])
```

If you want to test your dapps with a realistic block time use the `--dev.period` option when you start dev mode:

```shell
geth --dev --dev.period 14 console
```
