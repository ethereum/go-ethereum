---
title: The Merge
sort_key: A2
---

As an Execution-Layer (EL) client, Geth supports the transition of Ethereum from PoW to
PoS, a.k.a. [The Merge](https://ethereum.org/en/upgrades/merge/). As this milestone
approaches for the Ropsten testnet, users running a Geth node will need to keep a few
things in mind to ensure a smooth transition.

### Transition

The transition will happen when a pre-announced total difficulty is reached by the chain.
This is unlike usual forks which occur at a certain scheduled block number.

The total difficulty threshold that triggers the Merge is also known as the *Terminal
Total Difficulty* (TTD).

{% include note.html content="In case of an emergency delay, the TTD can be overriden using `--override.totalterminaldifficulty`." %}

### Consensus client

After the Merge, Ethereum's Proof-Of-Stake consensus logic is handled by a separate piece
of software called a Consensus-Layer (CL) client.

This means running Geth alone will not suffice to be able to follow the chain. You must
set up a consensus client and configure it so it can talk to Geth. This has to be done
before the Merge event happens.

The two clients will coordinate the transition together.

You can choose one of several [Consensus Client implementations][cl-list].

Note that CL clients are equipped with two modes. One for following the beacon chain
(beacon node), and another mode used for validators. **It is NOT required to run a
validator or stake 32 ETH in order to follow the chain!**

### EL - CL communication

ELs and CLs communicate using a [JSON-RPC API][engineapi] under the namespace `engine`
which is exposed by Geth.

The `engine` API is authenticated via [JWT](https://jwt.io). If a TTD is set for the given
network, as is the case for Ropsten, Geth will:

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

The listening address of the engine API is configurable. It is also possible to
self-generate the JWT secret and feed the resulting file to Geth. To generate the secret,
run:

```shell
openssl rand -hex 32 | tr -d "\n" > "/tmp/jwtsecret"
```

Now configure authentication using Geth flags:

```shell
geth --ropsten --datadir ~/.ropsten --authrpc.addr localhost --authrpc.port 8551 --authrpc.vhosts localhost --authrpc.jwtsecret /tmp/jwtsecret
```

[engineapi]: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md
[cl-list]: https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients
