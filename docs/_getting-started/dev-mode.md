---
title: Dev mode
sort_key: B
---

Geth has a development mode that sets up a single node Ethereum test network with options optimized for developing on local machines. You enable it with the `--dev` argument.

Starting geth in dev mode does the following:

-   Initializes the data directory with a testing genesis block
-   Sets max peers to 0
-   Turns off discovery by other nodes
-   Sets the gas price to 0
-   Uses the Clique PoA consensus engine with which allows blocks to be mined as-needed without excessive CPU and memory consumption
-   Uses on-demand block generation, producing blocks when transactions are waiting to be mined

## Start Geth in Dev Mode

You can specify a data directory to maintain state between runs using the `--datadir` option, otherwise, databases are ephemeral and in-memory:

```shell
mkdir test-chain-dir
```

For this guide, start geth in dev mode, and enable [RPC](../_rpc/server.md) so you can connect other applications to geth. For this guide, we use Remix, the web-based Ethereum IDE, so also allow its domains to accept cross-origin requests.

```shell
geth --datadir test-chain-dir --rpc --dev --rpccorsdomain "https://remix.ethereum.org,http://remix.ethereum.org"
```

Connect to the IPC console on the node from another terminal window:

```shell
geth attach <IPC_LOCATION>
```

Once geth is running in dev mode, you can interact with it in the same way as when geth is running in other ways.

For example, create a test account:

```shell
> personal.newAccount()
```

And transfer ether from the coinbase to the new account:

```shell
> eth.sendTransaction({from:eth.coinbase, to:eth.accounts[1], value: web3.toWei(0.05, "ether")})
```

And check the balance of the account:

```shell
> eth.getBalance(eth.accounts[1])
```

If you want to test your dapps with a realistic block time use the `--dev.period` option when you start dev mode with the `--dev.period 14` argument.

## Connect Remix to Geth

With geth now running, open <https://remix.ethereum.org>. Compile the contract as normal, but when you deploy and run a contract, select _Web3 Provider_ from the _Environment_ dropdown, and add "http://127.0.0.1:8545" to the popup box. Click _Deploy_, and interact with the contract. You should see contract creation, mining, and transaction activity.