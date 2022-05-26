---
title: The Merge
sort_key: A2
---

As an Execution-Layer (EL) client, Geth supports the transition of Ethereum from proof-of-work (PoW) to
proof-of-stake (PoS), a.k.a. [The Merge](https://ethereum.org/en/upgrades/merge/). In advance of the Mainnet
merge, several public testnets will transition from PoW to PoS. The first public testnet to merge will be 
Ropsten. As the Ropsten merge approaches, Geth users will need to prepare to ensure a smooth transition.

### What happens to Geth?

The merge swaps Ethereum's PoW consensus mechanism for a new PoS mechanism. At the moment of the merge, 
Geth switches off its mining algorithm and block gossip functions. Geth's role after the merge is 
to execute transactions, generate execution payloads and run the [EVM](https://ethereum.org/en/developers/docs/evm).

From a user perspective Geth will not change much at the merge. Responsibility for consensus logic and
block propagation are handed over to the consensus layer, but all of Geth's other functionality remains
intact. This means transactions, contract deployments and data queries can still be handled by Geth using
the same methods as before.

### Transition

The transition from PoW to PoS will happen when a pre-announced total difficulty is reached by the chain. 
This is unlike usual forks which occur at a certain scheduled block number.

The total difficulty threshold that triggers the Merge is also known as the [*Terminal
Total Difficulty* (TTD)](https://ethereum.org/en/glossary/#terminal-total-difficulty).

{% include note.html content="In case of an emergency delay, the TTD can be overriden using `--override.totalterminaldifficulty`." %}

### Consensus client

After the merge, Geth will no longer be able to follow the head of the chain unless it is connected to a second
piece of software known as a ["consensus client"](https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients)
which handles all of Ethereum's consensus logic. 

The consensus client communicates with Geth using the `engine` API over a local RPC connection. The consensus client 
is responsible for gossiping blocks, block proposal, attestation and fork choice. 

Geth must be connected to a consensus client in advance of the merge so that the two pieces of software can get in sync and
transition smoothly by coordinating together. There are several choices of [consensus client implementations][cl-list].

Note that CL clients are equipped with two modes. One for following the beacon chain (Beacon node), and another mode 
used for validators. **It is NOT required to run a validator or stake 32 ETH in order to follow the chain!**

### EL - CL communication

ELs and CLs communicate using a [JSON-RPC API][engineapi] under the namespace `engine` which is exposed by Geth.

The `engine` API is authenticated via [JWT](https://jwt.io). If a TTD is set for the given network, as is the 
case for Ropsten, Geth will:

- Generate a JWT secret under the path `<datadir>/geth/jwtsecret`. This secret is needed
  both by Geth and the CL client.

- Open HTTP and WS endpoints on the authenticated port 8551.

This is what it looks like by default:

```shell
geth --ropsten --datadir ~/.ropsten
```

```terminal
INFO [05-25|11:04:41.179] Starting Geth on Ropsten testnet...
...
WARN [05-25|11:04:41.520] Catalyst mode enabled                    protocol=eth
INFO [05-25|11:04:41.627] Generated JWT secret                     path=/home/user/.ropsten/geth/jwtsecret
INFO [05-25|11:04:41.628] WebSocket enabled                        url=ws://127.0.0.1:8551
INFO [05-25|11:04:41.628] HTTP server started                      endpoint=127.0.0.1:8551 auth=true prefix= cors=localhost vhosts=localhost
```

### Engine API Authentication

The listening address of the engine API is configurable. It is also possible to self-generate the JWT secret and feed the resulting file to Geth. To generate the secret, run:

```shell

openssl rand -hex 32 | tr -d "\n" > "/tmp/jwtsecret"

```

Now configure authentication using Geth flags:

```shell

geth --ropsten --datadir ~/.ropsten --authrpc.addr localhost --authrpc.port 8551 --authrpc.vhosts localhost --authrpc.jwtsecret /tmp/jwtsecret

```

[engineapi]: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md
[cl-list]: https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients
