---
title: Dev mode
---

Geth has a development mode which sets up a single node Ethereum test network with a number of options optimized for developing on local machines. You enable it with the `--dev` argument.

Dev mode does the following:

-   Initializes the data directory with a testing genesis block
-   Sets max peers to 0
-   Turns off discovery by other nodes
-   Sets the gas price to 0
-   Uses a testing consensus algorithm (proof of work) which generates blocks quickly
-   Prevents the consensus (proof of work) difficulty from ever increasing

We recommend you create and specify a data directory when using dev mode.

```shell
$ mkdir test-chain-dir
$ geth --dev --datadir test-chain-dir console
```

<!-- TODO: The command above is a lot simpler, introduce elsewhere? -->

Once geth is running, create a test account.

<!-- TODO: What is the difference in these methods? -->

<!-- TODO: Fix next steps as per https://github.com/curvegrid/go-ethereum-wiki/pull/1 -->

```shell
> personal.newAccount()
Passphrase:
Repeat passphrase:
INFO [06-25|00:51:55] New wallet appeared                      url=keystore:///tmp… status=Locked
"0x42a3f741fa25e52c618854e7a002ff7c7985b044"
>
```

Then start the miner to mine some blocks. The coinbase will have automatically been set to the test account so the mined Ether will be deposited there. You can verify this by running `eth.coinbase`.

```sh
> miner.start()
INFO [06-25|00:52:45] Updated mining threads                   threads=0
INFO [06-25|00:52:45] Transaction pool price threshold updated price=0
INFO [06-25|00:52:45] Starting mining operation
null
> INFO [06-25|00:52:45] Commit new mining work                   number=1 txs=0 uncles=0 elapsed=103.893µs
INFO [06-25|00:52:45] Generating DAG in progress               epoch=0 percentage=0 elapsed=68.374µs
INFO [06-25|00:52:45] Generating DAG in progress               epoch=0 percentage=3 elapsed=137.968µs
...
INFO [06-25|00:52:51] 🔗 block reached canonical chain          number=3 hash=7049f9…22f775
INFO [06-25|00:52:51] 🔨 mined potential block                  number=8 hash=c84185…4b5994
INFO [06-25|00:52:51] Mining too far in the future             wait=2s
```

And after a few seconds, stop the miner.

```sh
> miner.stop()
INFO [06-25|00:52:53] Commit new mining work                   number=9 txs=0 uncles=0 elapsed=2.002s
true
>
```

Now check the balance in your test account.

```sh
> web3.fromWei(eth.getBalance(personal.listAccounts[0]), "ether")
40
>
```

Our test account has 40 Ether in it, which at 5 Ether per block means it mined 8 blocks in 8 seconds on a contemporary iMac, including generating the [DAG](https://ethereum.stackexchange.com/questions/1993/what-actually-is-a-dag).
