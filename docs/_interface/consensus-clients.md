---
title: Connecting to Consensus Clients
sort_key: A3
---

Geth is an [execution client][ex-client-link]. Historically, an execution client alone has been 
enough to run a full Ethereum node. However, since Ethereum swapped its consensus mechanism from 
[proof-of-work][pow-link] (PoW) to [proof-of-stake][pos-link] (PoS) Geth is no longer able to track 
the Ethereum chain on its own. 

Instead, Geth needs to be coupled to another piece of software called a ["consensus client"][con-client-link]. 
There are five consensus clients available, all of which connect to Geth in the same way. 

This page will provide a general outline for how Geth can be set up with a consensus client.

## Configuring Geth

Geth can be downloaded and installed according to the instructions on the 
[Installing Geth](/docs/install-and-build/installing-geth) page. In order to connect to a consensus client,
Geth must expose a port for the inter-client RPC connection. 

The RPC connection must be authenticated using a `jwtsecret` file. This is created and saved 
to `<datadir>/geth/jwtsecret` by default but can also be created and saved to a custom location or it can be
self-generated and provided to Geth by passing the file path to `--authrpc.jwtsecret`. The `jwtsecret` file 
is required by both Geth and the consensus client.

The authorization must then be applied to a specific address/port. This is achieved by passing an address to
`--authrpc.addr` and a port number to `--authrpc.port`. It is also safe to provide either `localhost` or a wildcard
`*` to `--authrpc.vhosts` so that incoming requests from virtual hosts are accepted by Geth because it only 
applies to the port authenticated using `jwtsecret`. 

A complete command to start Geth so that it can connect to a consensus client looks as follows:

```shell
geth --authrpc.addr localhost --authrpc.port 8551 --authrpc.vhosts localhost --authrpc.jwtsecret /tmp/jwtsecret
```


## Consensus clients

There are currently five consensus clients that can be run alongside Geth. These are:
 
[Lighthouse](https://lighthouse-book.sigmaprime.io/): written in Rust
 
[Nimbus](https://nimbus.team/): written in Nim
 
[Prysm](https://docs.prylabs.network/docs/install/install-with-script): written in Go
 
[Teku](https://pegasys.tech/teku): written in Java

[Lodestar](https://github.com/ChainSafe/lodestar): written in Typescript
 
It is recommended to consider [client diversity][client-div-link] when choosing a consensus client. 
Instructions for installing each client are provided in the documentation linked in the list above.

The consensus client must be started with the right port configuration to establish an RPC connection 
to the local Geth instance. In the example above, `localhost:8551` was authorized 
for this purpose. The consensus clients all have a command similar to `--http-webprovider` that 
takes the exposed Geth port as an argument.

The consensus client also needs the path to Geth's `jwt-secret` in order to authenticate the RPC connection between them.
Each consensus client has a command similar to `--jwt-secret` that takes the file path as an argument. This must
be consistent with the `--authrpc.jwtsecret` path provided to Geth.

The consensus clients all expose a [Beacon API][beacon-api-link] that can be used to check the status
of the Beacon client or download blocks and consensus data by sending requests using tools such as [Curl](https://curl.se).
More information on this can be found in the documentation for each consensus client.

## Validators

Validators are responsible for securing the Ethereum blockchain. Validators are node operators that have staked at least 
32 ETH into a deposit contract and run validator software. Each of the consensus clients have their own validator software 
that is described in detail in their respective documentation. The easiest way to handle staking and validator key generation 
is to use the Ethereum Foundation [Staking Launchpad][launchpad-link].

## Syncing

Geth cannot sync until the connected consensus client is synced. The fastest way to sync a consensus client is 
using checkpoint sync. To do this, a checkpoint or a url to a checkpoint provider can be provided to the consensus 
client on startup. There are several sources for these checkpoints. The ideal scenario is to get one from a 
trusted node operator, organized out-of-band, and verified against a third node or a block explorer or checkpoint 
provider. Some clients also allow checkpoint syncing by HTTP API access to an existing Beacon node. 
There are also several [public checkpoint sync endpoints](https://eth-clients.github.io/checkpoint-sync-endpoints/).

Please see the pages on [syncing](/docs/interface/sync-modes.md) for more detail. For troubleshooting, 
please see the `Syncing` section on the [console log messages](/docs/interface/logs.md) page.

## Summary

Geth requires a connection to a consensus client in order to follow the Ethereum blockchain. There are five consensus clients 
to choose from. This page provided an overview of how to choose a consensus client and configure Geth to connect to it. More
information can be found on the clients' respective documentation sites or in numerous 
[online guides](https://github.com/SomerEsat/ethereum-staking-guides).


[pow-link]:https://ethereum.org/en/developers/docs/consensus-mechanisms/pow
[pos-link]:https://ethereum.org/en/developers/docs/consensus-mechanisms/pos
[con-client-link]:https://ethereum.org/en/glossary/#consensus-client
[ex-client-link]:https://ethereum.org/en/glossary/#execution-client
[beacon-api-link]:https://ethereum.github.io/beacon-APIs
[engine-api-link]: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md
[client-div-link]:https://ethereum.org/en/developers/docs/nodes-and-clients/client-diversity
[execution-clients-link]: https://ethereum.org/en/developers/docs/nodes-and-clients/client-diversity/#execution-clients
[launchpad-link]:https://launchpad.ethereum.org/
[prater-launchpad-link]:https://prater.launchpad.ethereum.org/
[e-org-link]: https://ethereum.org/en/developers/docs/nodes-and-clients/run-a-node/
[checklist-link]:https://launchpad.ethereum.org/en/merge-readiness
