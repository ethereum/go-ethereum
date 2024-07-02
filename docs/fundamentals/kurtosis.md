---
title: Private Networks via Kurtosis
description: Setting up private Ethereum networks, the easy way
---

This guide explains how to set up a private network of multiple Geth nodes along with their corresponding [consensus clients](/docs/getting-started/consensus-clients) using [Kurtosis](https://docs.kurtosis.com/basic-concepts), a tool that facilitates running containerized packages. An Ethereum network is private if the nodes are not connected to mainnet or any of the testnets. In this context private only means reserved or isolated, rather than protected or secure. A fully controlled, private Ethereum network is useful as a backend for core developers working on issues relating to networking/blockchain syncing etc. Private networks are also useful for Dapp developers testing multi-block and multi-user scenarios.

<Note>Geth only supports the Ethereum PoS consensus mechanism. This is a permissionless algorithm, meaning anyone who can access the private network and has enough ether (local to that network) can become a validator and propose blocks.</Note>

## Prerequisites {#prerequisites}

To follow the tutorial on this page it is necessary to have a working Kurtosis installation (instructions [here](https://docs.kurtosis.com/install)), as well as [Docker](https://docs.docker.com/get-docker/). It is also helpful to understand Geth fundamentals (see [Getting Started](/docs/getting-started)). 

## Private Networks {#private-networks}

A private network is composed of multiple Ethereum nodes that can only connect to each other. There are many details to setting up a fresh PoS network. To name a few: a genesis block must be generated for the execution as well as consensus client. The genesis will also contain the deposit contract which validators will use to stake on the network. Then ELs and CLs must be set-up in a concert off of the genesis files. The Kurtosis [ethereum-package](https://github.com/ethpandaops/ethereum-package) will handle all of that behind the scenes with the ability to costumize where needed.

### Choosing A Network ID {#choosing-network-id}

Ethereum Mainnet has Network ID = 1. There are also many other networks that Geth can connect to by providing alternative Chain IDs, some are testnets and others are alternative networks built from forks of the Geth source code. Providing a network ID that is not already being used by an existing network or testnet means the nodes using that network ID can only connect to each other, creating a private network. A list of current network IDs is available at [Chainlist.org](https://chainlist.org/).

### Basic configuration

Kurtosis runs based off Starlark configurations. Write the following content in a file named `network_params.yaml`:

```yaml
participants:
  - el_type: geth
    cl_type: lighthouse
    count: 2
  - el_type: geth
    cl_type: teku
network_params:
  network_id: "585858"
additional_services:
  - dora
```

This describes the structure of the network desired. The network will consist of 3 client pairs (execution and consensus). 2 of them running geth/lighthouse and 1 running geth/teku. Each pair would have an equal number of validators. They all will share a genesis block and will be peered together. It's best to specify a non-conflicting network ID for a private network. If no ID is indicated kurtosis will choose a default one (at the time of writing the default is 3151908).

### Spinning up the network

Once the config is written, it is straightforward to spin up the network. Run the following command:

```terminal
kurtosis run github.com/ethpandaops/ethereum-package --args-file ./network_params.yaml --image-download always
```

This indicates ethereum-package as a dependency which defines what the fields above mean. `--image-download always` makes sure the latest images are used always. Running it will produce an output such as the one below on a successful run:

```
INFO[2024-06-03T18:05:23+02:00] ===================================================
INFO[2024-06-03T18:05:23+02:00] ||          Created enclave: dusty-soil          ||
INFO[2024-06-03T18:05:23+02:00] ===================================================
Name:            dusty-soil
UUID:            1a33b911bfa4
Status:          RUNNING
Creation Time:   Mon, 03 Jun 2024 18:04:43 CEST
Flags:

========================================= Files Artifacts =========================================
UUID           Name
48ecd031ac60   1-lighthouse-geth-0-63-0
4d9057965009   2-lighthouse-geth-64-127-0
287a1079d7a7   3-teku-geth-128-191-0
760206ace8ae   dora-config
61bcf0e4a182   el_cl_genesis_data
72fa0877e1f0   final-genesis-timestamp
c30d6e459e5d   genesis-el-cl-env-file
3e1aa28cadf3   genesis_validators_root
41e32b09194d   jwt_file
3a555e3e1238   keymanager_file
1ffd63ba783c   prysm-password
a9eabb55db42   validator-ranges

========================================== User Services ==========================================
UUID           Name                                             Ports                                         Status
35dbe5e28986   cl-1-lighthouse-geth                             http: 4000/tcp -> http://127.0.0.1:54607      RUNNING
                                                                metrics: 5054/tcp -> http://127.0.0.1:54605
                                                                tcp-discovery: 9000/tcp -> 127.0.0.1:54606
                                                                udp-discovery: 9000/udp -> 127.0.0.1:56102
2758e9a955e3   cl-2-lighthouse-geth                             http: 4000/tcp -> http://127.0.0.1:54610      RUNNING
                                                                metrics: 5054/tcp -> http://127.0.0.1:54608
                                                                tcp-discovery: 9000/tcp -> 127.0.0.1:54609
                                                                udp-discovery: 9000/udp -> 127.0.0.1:55675
5e648790d930   cl-3-teku-geth                                   http: 4000/tcp -> http://127.0.0.1:54613      RUNNING
                                                                metrics: 8008/tcp -> 127.0.0.1:54611
                                                                tcp-discovery: 9000/tcp -> 127.0.0.1:54612
                                                                udp-discovery: 9000/udp -> 127.0.0.1:62286
1f961bcf0ef7   dora                                             http: 8080/tcp -> http://127.0.0.1:54628      RUNNING
f8a7764be245   el-1-geth-lighthouse                             engine-rpc: 8551/tcp -> 127.0.0.1:54586       RUNNING
                                                                metrics: 9001/tcp -> 127.0.0.1:54587
                                                                rpc: 8545/tcp -> http://127.0.0.1:54589
                                                                tcp-discovery: 30303/tcp -> 127.0.0.1:54588
                                                                udp-discovery: 30303/udp -> 127.0.0.1:51523
                                                                ws: 8546/tcp -> 127.0.0.1:54590
33a1aa3734f0   el-2-geth-lighthouse                             engine-rpc: 8551/tcp -> 127.0.0.1:54595       RUNNING
                                                                metrics: 9001/tcp -> 127.0.0.1:54596
                                                                rpc: 8545/tcp -> http://127.0.0.1:54598
                                                                tcp-discovery: 30303/tcp -> 127.0.0.1:54597
                                                                udp-discovery: 30303/udp -> 127.0.0.1:61026
                                                                ws: 8546/tcp -> 127.0.0.1:54599
22ec7e014303   el-3-geth-teku                                   engine-rpc: 8551/tcp -> 127.0.0.1:54602       RUNNING
                                                                metrics: 9001/tcp -> 127.0.0.1:54603
                                                                rpc: 8545/tcp -> http://127.0.0.1:54600
                                                                tcp-discovery: 30303/tcp -> 127.0.0.1:54604
                                                                udp-discovery: 30303/udp -> 127.0.0.1:60590
                                                                ws: 8546/tcp -> 127.0.0.1:54601
c4655f3e76da   validator-key-generation-cl-validator-keystore   <none>                                        RUNNING
349a3759d6c8   vc-1-geth-lighthouse                             metrics: 8080/tcp -> http://127.0.0.1:54621   RUNNING
deed7eacfd93   vc-2-geth-lighthouse                             metrics: 8080/tcp -> http://127.0.0.1:54623   RUNNING

```

That's it. Kurtosis has started all of the network components that was specified in the config in an enclave. The name of the enclave (i.e. `dusty-soil` for the run above) will be required to interact with the services. By now, the network should have started producing and validating new blocks. To get some insight into each of the clients it's possible to check the logs.

```
> kurtosis service logs dusty-soil el-1-geth-lighthouse

[el-1-geth-lighthouse] INFO [06-04|07:59:05.048] Chain head was updated                   number=495 hash=2f3200..673eee root=d3d92f..d3bd27 elapsed=3.429333ms
[el-1-geth-lighthouse] INFO [06-04|07:59:13.008] Starting work on payload                 id=0x03c53477e90934c9
[el-1-geth-lighthouse] INFO [06-04|07:59:13.008] Updated payload                          id=0x03c53477e90934c9 number=496 hash=e995db..f5310d txs=0 withdrawals=0 gas=0 fees=0 root=36638a..e3c9a9 elapsed="379.542µs"
[el-1-geth-lighthouse] INFO [06-04|07:59:17.007] Stopping work on payload                 id=0x03c53477e90934c9 reason=delivery
[el-1-geth-lighthouse] INFO [06-04|07:59:17.041] Imported new potential chain segment     number=496 hash=e995db..f5310d blocks=1 txs=0 mgas=0.000 elapsed=20.254ms     mgasps=0.000 snapdiffs=98.81KiB triediffs=454.03KiB triedirty=79.69KiB
[el-1-geth-lighthouse] INFO [06-04|07:59:17.047] Chain head was updated                   number=496 hash=e995db..f5310d root=36638a..e3c9a9 elapsed=2.198709ms
```

### Block explorer

You might have noticed in the configuration above an additional service called `dora` was requested. [Dora](https://github.com/ethpandaops/dora) is a lightweight block explorer. The kurtosis logs above indicate that dora was successfuly launched as a service and is available at `http://127.0.0.1:54628` to inspect the chain.

### Interacting with geth

The most straightforward to interact with any of the geth nodes is through JSON-RPC. They are started already with the RPC server running and kurtosis has exposed those ports to the host as indicated in the logs. E.g. First geth node can be accessed via `http://127.0.0.1:54589`. Therefor the current block number can be retrieved via:

```
> curl -X POST -H "Content-Type: application/json" --data '{"method":"eth_blockNumber","params":[],"id":1,"jsonrpc":"2.0"}' http://127.0.0.1:54589

{"jsonrpc":"2.0","id":1,"result":"0x332"}
```

In the end the kurtosis services are docker images. It is also possible to get shell access to them and poke around, e.g. load up the console. Kurtosis facilitates the shell access through a command:

```
> kurtosis service shell dusty-soil el-1-geth-lighthouse
No bash found on container; dropping down to sh shell...
/ # geth --datadir /data/geth/execution-data/ attach
Welcome to the Geth JavaScript console!

instance: Geth/v1.14.4-unstable-a6751d6f/linux-arm64/go1.22.3
at block: 830 (Tue Jun 04 2024 09:07:29 GMT+0000 (UTC))
 datadir: /data/geth/execution-data
  modules: admin:1.0 debug:1.0 engine:1.0 eth:1.0 miner:1.0 net:1.0 rpc:1.0 txpool:1.0 web3:1.0

  To exit, press ctrl-d or type exit
  >
```

## Further reading

This tutorial covered the basics of spinning up a network via Kurtosis. The [ethereum-package](https://github.com/ethpandaops/ethereum-package) has far more features and options than the scope of this tutorial. The [guide](https://ethpandaops.io/posts/kurtosis-deep-dive/) by ethPandaOps also goes over more advanced functionality such as deploying a MEV stack, shadowforking etc.
