---
title: Mining
sort_key: B
---

This document explains how to set up geth for mining. The Ethereum wiki also has a [page
about mining](eth-wiki-mining), be sure to check that one as well.

Mining is the process through which new blocks are created. Geth actually creates new
blocks all the time, but these blocks need to be secured through proof-of-work so they
will be accepted by other nodes. Mining is all about creating these proof-of-work values.

The proof-of-work computation can be performed in multiple ways. Geth includes a CPU
miner, which does mining within the geth process. We discourage using the CPU miner with
the Ethereum mainnet. If you want to mine real ether, use GPU mining. Your best option for
doing that is the [ethminer](ethminer) software.

Always ensure your blockchain is fully synchronised with the chain before starting to
mine, otherwise you will not be mining on the correct chain and your block rewards will
not be valueable.

## GPU mining

The ethash algorithm is memory hard and in order to fit the DAG into memory, it needs
1-2GB of RAM on each GPU. If you get `Error GPU mining. GPU memory fragmentation?` you
don't have enough memory.

### Installing ethminer

To get ethminer, you need to install the ethminer binary package or build it from source.
See <https://github.com/ethereum-mining/ethminer/#build> for the official ethminer
build/install instructions. At the time of writing, ethminer only provides a binary for
Microsoft Windows.

### Using ethminer with geth

First create an account to hold your block rewards.

    geth account new

Follow the prompts and enter a good password. **DO NOT FORGET YOUR PASSWORD**. Also take
note of the public Ethereum address which is printed at the end of the account creation
process. In the following examples, we will use 0xC95767AC46EA2A9162F0734651d6cF17e5BfcF10
as the example address.

Now start geth and wait for it to sync the blockchain. This will take quite a while.

    geth --rpc --etherbase 0xC95767AC46EA2A9162F0734651d6cF17e5BfcF10

Now we're ready to start mining. In a new terminal session, run ethminer and connect it to geth:

    ethminer -G -P http://127.0.0.1:8545

`ethminer` communicates with geth on port 8545 (the default RPC port in geth). You can
change this by giving the [`--rpcport` option](../rpc/index) to `geth`. Ethminer will find
get on any port. You also need to set the port on `ethminer` with `-P
http://127.0.0.1:3301`. Setting up custom ports is necessary if you want several instances
mining on the same computer. If you are testing on a private cluster, we recommend you use
CPU mining instead.

If the default for `ethminer` does not work try to specify the OpenCL device with:
`--opencl-device X` where X is 0, 1, 2, etc. When running `ethminer` with `-M`
(benchmark), you should see something like:

    Benchmarking on platform: { "platform": "NVIDIA CUDA", "device": "GeForce GTX 750 Ti", "version": "OpenCL 1.1 CUDA" }

    Benchmarking on platform: { "platform": "Apple", "device": "Intel(R) Xeon(R) CPU E5-1620 v2 @ 3.70GHz", "version": "OpenCL 1.2 " }

**Note** hashrate info is not available in `geth` when GPU mining. Check your hashrate
with `ethminer`, `miner.hashrate` will always report 0.

## CPU Mining with Geth

When you start up your ethereum node with `geth` it is not mining by default. To start it
in mining mode, you use the `--mine` command-line flag. The `--minerthreads` parameter can
be used to set the number parallel mining threads (defaulting to the total number of
processor cores).

    geth --mine --minerthreads=4

You can also start and stop CPU mining at runtime using the
[console](../interface/javascript-console). `miner.start` takes an optional parameter for
the number of miner threads.

    > miner.start(8)
    true
    > miner.stop()
    true

Note that mining for real ether only makes sense if you are in sync with the network
(since you mine on top of the consensus block). Therefore the eth blockchain
downloader/synchroniser will delay mining until syncing is complete, and after that mining
automatically starts unless you cancel your intention with `miner.stop()`.

In order to earn ether you must have your **etherbase** (or **coinbase**) address set.
This etherbase defaults to your [primary account](../interface/managing-your-accounts). If
you don't have an etherbase address, then `geth --mine` will not start up.

You can set your etherbase on the command line:

    geth --etherbase '0xC95767AC46EA2A9162F0734651d6cF17e5BfcF10' --mine 2>> geth.log

You can reset your etherbase on the console too:

    > miner.setEtherbase(eth.accounts[2])

Note that your etherbase does not need to be an address of a local account, just an
existing one.

There is an option [to add extra data](../interface/javascript-console) (32 bytes only) to
your mined blocks. By convention this is interpreted as a unicode string, so you can set
your short vanity tag.

    > miner.setExtra("ΞTHΞЯSPHΞЯΞ")

You can check your hashrate with [miner.hashrate](../interface/javascript-console), the
result is in H/s (Hash operations per second).

    > miner.hashrate
    712000

After you successfully mined some blocks, you can check the ether balance of your
etherbase account. Now assuming your etherbase is a local account:

    > eth.getBalance(eth.coinbase).toNumber();
    '34698870000000'

You can check which blocks are mined by a particular miner (address) with the following
code snippet on the console:

    > function minedBlocks(lastn, addr) {
        addrs = [];
        if (!addr) {
            addr = eth.coinbase
        }
        limit = eth.blockNumber - lastn
        for (i = eth.blockNumber; i >= limit; i--) {
            if (eth.getBlock(i).miner == addr) {
                addrs.push(i)
            }
        }
        return addrs
    }
    // scans the last 1000 blocks and returns the blocknumbers of blocks mined by your coinbase
    // (more precisely blocks the mining reward for which is sent to your coinbase).
    > minedBlocks(1000, eth.coinbase)
    [352708, 352655, 352559]

Note that it will happen often that you find a block yet it never makes it to the
canonical chain. This means when you locally include your mined block, the current state
will show the mining reward credited to your account, however, after a while, the better
chain is discovered and we switch to a chain in which your block is not included and
therefore no mining reward is credited. Therefore it is quite possible that as a miner
monitoring their coinbase balance will find that it may fluctuate quite a bit.

The logs show locally mined blocks confirmed after 5 blocks. At the moment you may find it
easier and faster to generate the list of your mined blocks from these logs.

[eth-wiki-mining]: https://github.com/ethereum/wiki/wiki/Mining
[ethminer]: https://github.com/ethereum-mining/ethminer
