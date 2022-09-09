---
title: Proof-of-work mining with Ethash
description: Introduction to proof-of-work mining with Geth
---

{% include note.html content=" Proof-of-work mining is **no longer used to secure Ethereum Mainnet**. The information below is included because the Ethash code is still part of Geth and it could be used to create a private proof-of-work network or testnet. %}

Blockchains grow when nodes add blocks and distribute them to their peers. Nodes that add blocks are rewarded with ether payouts. On Ethereum Mainnet, the proof-of-stake consensus engine randomyl selects a node to produce each block. Under proof-of-work, however, block producers are selected by competition. The node that computes a certain value that can only be found using repeated random guesses wins the right to propose the next block. Only if a node can demonstrate that they have calculated this value, and therefore expended energy, will their block be accepted by other nodes. This secures the network. This process of creating blocks and securing them using proof-of-work is known as "mining".

Much more information about mining, including details about the specific algorithm ("Ethash") used by Ethereum nodes is available on [ethereum.org](https://ethereum.org/en/developers/docs/consensus-mechanisms/pow/mining).

## CPU vs GPU

Participating in Ethereum's PoW mining requires running an algorithm called ["Ethash"](https://ethereum.org/en/developers/docs/consensus-mechanisms/pow/mining-algorithms/ethash). Geth includes a CPU miner which runs Ethash within the Geth process. This might be useful for mining on testnets or private networks where competition for producing new blocks is small. When minign was used to secure Ethereum Mainnet, CPU mining was not viable because CPU miners were easily outcompeted by more efficient GPU miners. To mine using GPUs an additional piece of third-party software is required. The recommended GPU mining software is [Ethminer](https://github.com/ethereum-mining/ethminer).

Regardless of the mining method, the blockchain must be fully synced before mining is started, otherwise the miner will build on an outdated side chain, meaning block rewards will not be recognized by the main network.

## GPU Mining

### Installing Ethminer

The Ethminer software can be installed from a downloaded binary or built from source. The relevant downloads and installation instructions are available from the [Ethminer Github](https://github.com/ethereum-mining/ethminer/#build). Standalone executables are available for Linux, macOS and Windows.

### Using Ethminer with Geth

An account to receive block rewards must first be defined. The address of the account is all that is required to start mining - the mining rewards will be credited to that address. This can be an existing address or one that is newly created by Geth. More detailed instructions on creating and importing accounts are available on the [Account Management](/content/docs/fundamentals/account-management.md) page.

The account address can be provided to `--mining.etherbase` when Geth is started. This instructs Geth to direct any block rewards to this address. Once started, Geth will sync the blockchain. If Geth has not connected to this network before, or if the data directory has been deleted, this can take several days. Also, enable HTTP traffic with the `--http` command. To reiterate, mining does **not work on Ethereum, Mainnet** since the transition to proof-of-stake.

```shell
geth --http --miner.etherbase 0xC95767AC46EA2A9162F0734651d6cF17e5BfcF10
```

The progress of the blockchain syncing can be monitored by attaching a JavaScript console in another terminal. More detailed information about the console can be found on the [Javascript Console](/content/docs/interacting_with_geth/JavaScript-Console.md) page. To attach and open a console:

```shell
geth attach http://127.0.0.1:8545
```

Then in the console, to check the sync progress:

```shell
eth.syncing
```

If the sync is progressing correctly the output will look similar to the following:

```terminal
{
  currentBlock: 13891665,
  healedBytecodeBytes: 0,
  healedBytecodes: 0,
  healedTrienodeBytes: 0,
  healedTrienodes: 0,
  healingBytecode: 0,
  healingTrienodes: 0,
  highestBlock: 14640000,
  startingBlock: 13891665,
  syncedAccountBytes: 0,
  syncedAccounts: 0,
  syncedBytecodeBytes: 0,
  syncedBytecodes: 0,
  syncedStorage: 0,
  syncedStorageBytes: 0
}
```

Once the blockchain is sync'd, mining can begin. In order to begin mining, Ethminer must be run and connected to Geth in a new terminal. OpenCL can be used for a wide range of GPUs, CUDA can be used specifically for Nvidia GPUs:

```shell
#OpenCL
ethminer -v 9 -G -P http://127.0.0.1:8545
```

```shell
#CUDA
ethminer -v -U -P http://127.0.0.1:8545
```

Ethminer communicates with Geth on port 8545 (Geth's default RPC port) but this can be changed by providing a custom port to the `http.port` command. The corresponding port must also be configured in Ethminer by providing `-P http://127.0.0.1:<port-number>`. This is necessary when multiple instances of Geth/Ethminer will coexist on the same machine.

If using OpenCL and the default for `ethminer` does not work, specifying the device using the `--opencl--device X` command is a common fix. `X` is an integer `1`, `2`, `3` etc. The Ethminer `-M` (benchmark) command should display something that looks like:

```terminal
Benchmarking on platform: { "platform": "NVIDIA CUDA", "device": "GeForce GTX 750 Ti", "version": "OpenCL 1.1 CUDA" }

Benchmarking on platform: { "platform": "Apple", "device": "Intel(R) Xeon(R) CPU E5-1620 v2 @ 3.70GHz", "version": "OpenCL 1.2 " }
```

Note that the Geth command `miner.hashrate` only works for CPU mining - it always reports zero for GPU mining. To check the GPU mining hashrate, check the logs `ethminer` displays to its terminal. More verbose logs can be configured using `-v` and a value between 0-9.

The Ethash algorithm is [memory-hard](https://crypto.stackexchange.com/questions/84002/memory-hard-vs-memory-bound-functions) and requires a large dataset to be loaded into memory. Each GPU requires 4-5 GB of RAM. The error message `Error GPU mining. GPU memory fragmentation?` indicates that there is insufficient memory available.

## CPU Mining with Geth

When Geth is started it is not mining by default. Unless it is specifically instructed to mine, it acts only as a node, not a miner. Geth starts as a (CPU) miner if the `--mine` flag is provided. The `--miner.threads` parameter can be used to set the number parallel mining threads (defaulting to the total number of processor cores).

```shell
geth --mine --miner.threads=4
```

CPU mining can also be started and stopped at runtime using the [console](/docs/interface/javascript-console). The command `miner.start` takes an optional parameter for the number of miner threads.

```js
miner.start(8);
true;
miner.stop();
true;
```

Note that mining for real ether only makes sense if you are in sync with the network (since you mine on top of the consensus block). Therefore the Ethereum blockchain downloader/synchroniser will delay mining until syncing is complete, and after that mining automatically starts unless you cancel your intention with `miner.stop()`.

Like with GPU mining, an etherbase account must be set. This defaults to the primary account in the keystore but can be set to an alternative address using the `--miner.etherbase` command:

```shell
geth --miner.etherbase '0xC95767AC46EA2A9162F0734651d6cF17e5BfcF10' --mine
```

If there is no account available an account wil be created and automatically configured to be the coinbase. The Javascript console can be used to reset the etherbase account at runtime:

```shell
miner.setEtherbase(eth.accounts[2])
```

Note that your etherbase does not need to be an address of a local account, it just has to be set to an existing one.

There is an option to add extra data (32 bytes only) to the mined blocks. By convention this is interpreted as a unicode string, so it can be used to add a short vanity tag using `miner.setExtra` in the Javascript console.

```shell
miner.setExtra("ΞTHΞЯSPHΞЯΞ")
```

The console can also be used to check the current hashrate in units H/s (Hash operations per second):

```shell
eth.hashrate
 712000
```

After some blocks have been mined, the etherbase account balance with be >0. Assuming the etherbase is a local account:

```shell
eth.getBalance(eth.coinbase).toNumber();
 '34698870000000'
```

It is also possible to check which blocks were mined by a particular miner (address) using the following code snippet in the Javascript console:

```js
function minedBlocks(lastn, addr) {
  addrs = [];
  if (!addr) {
    addr = eth.coinbase;
  }
  limit = eth.blockNumber - lastn;
  for (i = eth.blockNumber; i >= limit; i--) {
    if (eth.getBlock(i).miner == addr) {
      addrs.push(i);
    }
  }
  return addrs;
}

// scans the last 1000 blocks and returns the blocknumbers of blocks mined by your coinbase
// (more precisely blocks the mining reward for which is sent to your coinbase).
minedBlocks(1000, eth.coinbase)[(352708, 352655, 352559)];
```

The etherbase balance will fluctuate because quite often a mined block may be re-org'd out of the canonical chain. This means that when the local Geth node includes the mined block in its own local blockchain the account balance appears higher because the block rewards are applied. When the node switches to another version of the chain due to information received from peers, that block may not be included and the block rewards are not applied.

The logs show locally mined blocks confirmed after 5 blocks.

## Summary

The page describes how to start Geth as a mining node. Mining can be done on CPUs - in which case Geth's built-in miner can be used - or on GPUs which requires third party software. Mining is no longer used on Ethereum Mainnet so this information is only applicable to private proof-of-work networks or testnets.
