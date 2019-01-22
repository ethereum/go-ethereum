---
title: Mining
---
* [Introduction to Ethereum mining](https://github.com/ethereum/wiki/wiki/Mining#introduction) _(main wiki)_

# CPU Mining with Geth

At Frontier, the first release of Ethereum, you'll just need a) a GPU and b) an Ethereum client, Geth. CPU mining will be possible but too inefficient to hold any value.

At the moment, Geth only includes a CPU miner, and the team is testing a [GPU miner branch](https://github.com/ethereum/go-ethereum/tree/gpu_miner), but this won't be part of Frontier.

The C++ implementation of Ethereum also offers a GPU miner, both as part of Eth (its CLI), AlethZero (its GUI) and EthMiner (the standalone miner). 

_**NOTE:** Ensure your blockchain is fully synchronised with the main chain before starting to mine, otherwise you will not be mining on the main chain._

When you start up your ethereum node with `geth` it is not mining by default. To start it in mining mode, you use the `--mine` [command line option](Command-Line-Options). The `-minerthreads` parameter can be used to set the number parallel mining threads (defaulting to the total number of processor cores). 

`geth --mine --minerthreads=4`

You can also start and stop CPU mining at runtime using the [console](JavaScript-Console#adminminerstart). `miner.start` takes an optional parameter for the number of miner threads. 

```
> miner.start(8)
true
> miner.stop()
true
```

Note that mining for real ether only makes sense if you are in sync with the network (since you mine on top of the consensus block). Therefore the eth blockchain downloader/synchroniser will delay mining until syncing is complete, and after that mining automatically starts unless you cancel your intention with `miner.stop()`.

In order to earn ether you must have your **etherbase** (or **coinbase**) address set. This etherbase defaults to your [primary account](Managing-your-accounts). If you don't have an etherbase address, then `geth --mine` will not start up.

You can set your etherbase on the command line:

```
geth --etherbase 1 --mine  2>> geth.log // 1 is index: second account by creation order OR
geth --etherbase '0xa4d8e9cae4d04b093aac82e6cd355b6b963fb7ff' --mine 2>> geth.log
```

You can reset your etherbase on the console too:
```
miner.setEtherbase(eth.accounts[2])
```

Note that your etherbase does not need to be an address of a local account, just an existing one. 

There is an option [to add extra Data](JavaScript-Console#adminminersetextra) (32 bytes only) to your mined blocks. By convention this is interpreted as a unicode string, so you can set your short vanity tag.

```
miner.setExtra("ΞTHΞЯSPHΞЯΞ")
...
debug.printBlock(131805)
BLOCK(be465b020fdbedc4063756f0912b5a89bbb4735bd1d1df84363e05ade0195cb1): Size: 531.00 B TD: 643485290485 {
NoNonce: ee48752c3a0bfe3d85339451a5f3f411c21c8170353e450985e1faab0a9ac4cc
Header:
[
...
        Coinbase:           a4d8e9cae4d04b093aac82e6cd355b6b963fb7ff
        Number:             131805
        Extra:              ΞTHΞЯSPHΞЯΞ
...
}
```

See also [this proposal](https://github.com/ethereum/wiki/wiki/Extra-Data)

You can check your hashrate with [miner.hashrate](JavaScript-Console#adminminerhashrate), the result is in H/s (Hash operations per second). 

```
> miner.hashrate
712000
```

After you successfully mined some blocks, you can check the ether balance of your etherbase account. Now assuming your etherbase is a local account:

```
> eth.getBalance(eth.coinbase).toNumber();
'34698870000000' 
```

In order to spend your earnings you will need to have this account unlocked.

```
> personal.unlockAccount(eth.coinbase)
Password
true
```

You can check which blocks are mined by a particular miner (address) with the following code snippet on the console:

```
function minedBlocks(lastn, addr) {
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
minedBlocks(1000, eth.coinbase);
//[352708, 352655, 352559]
```

Note that it will happen often that you find a block yet it never makes it to the canonical chain. This means when you locally include your mined block, the current state will show the mining reward credited to your account, however, after a while, the better chain is discovered and we switch to a chain in which your block is not included and therefore no mining reward is credited. Therefore it is quite possible that as a miner monitoring their coinbase balance will find that it may fluctuate quite a bit. 

The logs show locally mined blocks confirmed after 5 blocks. At the moment you may find it easier and faster to generate the list of your mined blocks from these logs.

Mining success depends on the set block difficulty. Block difficulty dynamically adjusts each block in order to regulate the network hashing power to produce a 12 second blocktime. Your chances of finding a block therefore follows from your hashrate relative to difficulty. The time you need to wait you are expected to find a block can be estimated with the following code:

**INCORRECT...CHECKING**
```
etm = eth.getBlock("latest").difficulty/miner.hashrate; // estimated time in seconds
Math.floor(etm / 3600.) + "h " + Math.floor((etm % 3600)/60) + "m " +  Math.floor(etm % 60) + "s";
// 1h 3m 30s
```

Given a difficulty of 3 billion, a typical CPU with 800KH/s is expected to find a block every ....?


# GPU mining

***

## Hardware

The algorithm is memory hard and in order to fit the DAG into memory, it needs 1-2GB of RAM on each GPU. If you get ` Error GPU mining. GPU memory fragmentation?` you havent got enough memory.

The GPU miner is implemented in OpenCL, so AMD GPUs will be 'faster' than same-category NVIDIA GPUs.

ASICs and FPGAs are relatively inefficient and therefore discouraged. 

To get openCL for your chipset and platform, try:
* [AMD SDK openCL](http://developer.amd.com/tools-and-sdks/opencl-zone/amd-accelerated-parallel-processing-app-sdk)
* [NVIDIA CUDA openCL](https://developer.nvidia.com/cuda-downloads)

## On Ubuntu
### AMD

* http://developer.amd.com/tools-and-sdks/opencl-zone/amd-accelerated-parallel-processing
* http://developer.amd.com/tools-and-sdks/graphics-development/display-library-adl-sdk/

download: `ADL_SDK8.zip ` and `AMD-APP-SDK-v2.9-1.599.381-GA-linux64.sh`

```
./AMD-APP-SDK-v2.9-1.599.381-GA-linux64.sh
ln -s /opt/AMDAPPSDK-2.9-1 /opt/AMDAPP
ln -s /opt/AMDAPP/include/CL /usr/include
ln -s /opt/AMDAPP/lib/x86_64/* /usr/lib/
ldconfig
reboot
```

```
apt-get install fglrx-updates
// wget, tar, opencl
sudo aticonfig --adapter=all --initial
sudo aticonfig --list-adapters
* 0. 01:00.0 AMD Radeon R9 200 Series

* - Default adapter
```

### Nvidia
The following instructions are, for the most part, relevant to any system with Ubuntu 14.04 and a Nvidia GPU.
[Setting up an EC2 instance for mining](https://forum.ethereum.org/discussion/comment/8889/#Comment_8889)

## On MacOSx

```
wget http://developer.download.nvidia.com/compute/cuda/7_0/Prod/local_installers/cuda_7.0.29_mac.pkg
sudo installer -pkg ~/Desktop/cuda_7.0.29_mac.pkg -target /
brew update
brew tap ethereum/ethereum
brew reinstall cpp-ethereum --with-gpu-mining --devel --headless --build-from-source
```

You check your cooling status:

    aticonfig --adapter=0 --od-gettemperature

## Mining Software

The official Frontier release of `geth` only supports a CPU miner natively. We are working on a [GPU miner](https://github.com/ethereum/go-ethereum/tree/gpuminer), but it may not be available for the Frontier release. Geth however can be used in conjunction with `ethminer`, using the standalone miner as workers and `geth` as scheduler communicating via [JSON-RPC](https://github.com/ethereum/wiki/wiki/JSON-RPC). 

The [C++ implementation of Ethereum](https://github.com/ethereum/cpp-ethereum/) (not officially released) however has a GPU miner. It can be used from `eth`, `AlethZero` (GUI) and `ethMiner` (the standalone miner). 

[You can install this](https://github.com/ethereum/cpp-ethereum/wiki/Installing-clients) via ppa on linux, brew tap on MacOS or from source. 

On MacOS:
```
brew install cpp-ethereum --with-gpu-mining --devel --build-from-source
```

On Linux:
```
apt-get install cpp-ethereum 
```

On Windows: 
https://github.com/ethereum/cpp-ethereum/wiki/Building-on-Windows

## GPU mining with ethminer 
To mine with `eth`:

```
eth -m on -G -a <coinbase> -i -v 8 //
```

To install `ethminer` from source:

```
cd cpp-ethereum
cmake -DETHASHCL=1 -DGUI=0
make -j4
make install
```

To set up GPU mining you need a coinbase account. It can be an account created locally or remotely. 

### Using ethminer with geth

```
geth account new
geth --rpc --rpccorsdomain localhost 2>> geth.log &
ethminer -G  // -G for GPU, -M for benchmark
tail -f geth.log
```

`ethminer` communicates with geth on port 8545 (the default RPC port in geth). You can change this by giving the [`--rpcport` option](https://github.com/ethereum/go-ethereum/Command-Line-Options) to `geth`.
Ethminer will find get on any port. Note that you need to set the CORS header with `--rpccorsdomain localhost`. You can also set port on `ethminer` with `-F http://127.0.0.1:3301`. Setting the ports is necessary if you want several instances mining on the same computer,  although this is somewhat pointless. If you are testing on a private cluster, we recommend you use CPU mining instead. 

Also note that you do **not** need to give `geth` the `--mine` option or start the miner in the console unless you want to do CPU mining on TOP of GPU mining. 

If the default for `ethminer` does not work try to specify the OpenCL device with: `--opencl-device X` where X is 0, 1, 2, etc.
When running `ethminer` with `-M` (benchmark), you should see something like:

    Benchmarking on platform: { "platform": "NVIDIA CUDA", "device": "GeForce GTX 750 Ti", "version": "OpenCL 1.1 CUDA" }


    Benchmarking on platform: { "platform": "Apple", "device": "Intel(R) Xeon(R) CPU E5-1620 v2 @ 3.70GHz", "version": "OpenCL 1.2 " }

To debug `geth`:

```
geth  --rpccorsdomain "localhost" --verbosity 6 2>> geth.log
```

To debug the miner: 

```
make -DCMAKE_BUILD_TYPE=Debug -DETHASHCL=1 -DGUI=0
gdb --args ethminer -G -M
```

**Note** hashrate info is not available in `geth` when GPU mining. Check your hashrate with `ethminer`, `miner.hashrate` will always report 0. 


### ethminer and eth

`ethminer` can be used in conjunction with `eth` via rpc

```
eth -i -v 8 -j // -j for rpc
ethminer -G -M // -G for GPU, -M for benchmark
tail -f geth.log
```

or you can use `eth` to GPU mine by itself:

```
eth -m on -G -a <coinbase> -i -v 8 //
```

# Further Resources:

* [ether-proxy, a web interface for mining rigs](https://github.com/sammy007/ether-proxy)
  (supports solo and pool mining proxy with web interface and rigs availability monitoring)
* [ethereum forum mining FAQ live update](https://forum.ethereum.org/discussion/197/mining-faq-live-updates)
* [yates randall mining video](https://www.youtube.com/watch?v=CnKnclkkbKg)
* https://blog.ethereum.org/2014/07/05/stake/
* https://blog.ethereum.org/2014/10/03/slasher-ghost-developments-proof-stake/
* https://blog.ethereum.org/2014/06/19/mining/
* https://github.com/ethereum/wiki/wiki/Ethash
* [Benchmarking results for GPU mining](https://forum.ethereum.org/discussion/2134/gpu-mining-is-out-come-and-let-us-know-of-your-bench-scores)
* [historic moment](https://twitter.com/gavofyork/status/586623875577937922)
* [live mining statistic](https://etherapps.info/stats/mining)
* [netstat ethereum network monitor](https://stats.ethdev.com)
