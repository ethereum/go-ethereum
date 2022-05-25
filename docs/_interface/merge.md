---
title: The Merge
sort_key: B
---

As an Execution-Layer (EL) client, Geth will support the transition of Ethereum from PoW to PoS, i.e. [The Merge](https://ethereum.org/en/upgrades/merge/). As this milestone approaches for the Ropsten testnet, users running a Geth node will need to keep a few things in mind to ensure a smooth transition.

### Transition

The transition will happen at a [pre-announced](https://github.com/ethereum/go-ethereum/pull/24876) total difficulty (TTD, or total terminal difficulty). This is unlike usual forks which occur at a certain block number.

{% include note.html content="In case of an emergency delay the TTD can be overriden by the `--override.totalterminaldifficulty`" %}

### Consensus client

Post-merge Ethereum's consensus logic will be handled by a separate piece of software called a Consensus-Layer (CL) client. This means, running Geth alone will not suffice to be able to follow the chain. In fact already from some time before the merge happens you'll need to set up a consensus client and configure it so it can talk to Geth. The two clients will coordinate the transition together. Here is a list of [consensus clients](https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients) to pick from.

Note that CL clients are equipped with two modes. One for following the beacon chain (beacon node), and one which is used for validators. It is NOT required to run a validator or stake 32 ETH in order to follow the chain!

### EL-CL communication

ELs and CLs talk via a new [API](https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md) under the namespace `engine` which needs to be exposed on Geth. This communication is authenticated via [JWT](https://jwt.io). If the TTD is set for the given network as is the case for Ropsten in the most recent release, Geth will by default:

- Generate a JWT secret under the path `<datadir>/geth/jwtsecret`. **This secret is needed both by Geth and the CL client.**
- Open HTTP and WS endpoints on the authenticated port 8551

As you can see in the logs below:

```console
user@pc:~$ geth --ropsten --datadir ~/.ropsten
INFO [05-25|11:04:41.179] Starting Geth on Ropsten testnet...
[...]
INFO [05-25|11:04:41.518] Initialised chain configuration          config="{ChainID: 3 Homestead: 0 DAO: <nil> DAOSupport: true EIP150: 0
EIP155: 10 EIP158: 10 Byzantium: 1700000 Constantinople: 4230000 Petersburg: 4939394 Istanbul: 6485846, Muir Glacier: 7117117, Berlin: 981
2189, London: 10499401, Arrow Glacier: <nil>, MergeFork: <nil>, Terminal TD: 43531756765713534, Engine: ethash}"
[...]
WARN [05-25|11:04:41.520] Catalyst mode enabled                    protocol=eth
INFO [05-25|11:04:41.627] Generated JWT secret                     path=/home/user/.ropsten/geth/jwtsecret
INFO [05-25|11:04:41.628] WebSocket enabled                        url=ws://127.0.0.1:8551
INFO [05-25|11:04:41.628] HTTP server started                      endpoint=127.0.0.1:8551 auth=true prefix= cors=localhost vhosts=localhost
```

The address of this endpoint as well as the port and vhosts are configurable. It is also possible to self-generate the JWT secret and feed the resulting file to Geth. First to generate the secret, run:

```console
user@pc:~$ openssl rand -hex 32 | tr -d "\n" > "/tmp/jwtsecret"
```

Then all of these parameters can be passed to Geth:

```console
user@pc:~$ geth --ropsten --datadir ~/.ropsten --authrpc.addr localhost --authrpc.port 8551 --authrpc.vhosts localhost --authrpc.jwtsecret /tmp/jwtsecret
```
