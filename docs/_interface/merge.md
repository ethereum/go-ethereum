---
title: The Merge
sort_key: B
---

As an Execution-Layer (EL) client, Geth will support the transition of Ethereum from PoW to PoS, i.e. [The Merge](https://ethereum.org/en/upgrades/merge/). As this milestone approaches, users running a Geth node will need to keep a few things in mind to ensure a smooth transition.

### Consensus client

Post-merge Ethereum's consensus logic will be handled by a separate piece of software called a Consensus-Layer (CL) client. This means, running Geth alone will not suffice to be able to follow the chain. In fact already from some time before the merge happens you'll need to set up a consensus client and configure it so it can talk to Geth. The two clients will coordinate the transition together. Here you can find a list of [consensus clients](https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients) you can choose from.

### EL-CL communication

ELs and CLs talk with oneanother via a new [API](https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md) under the namespace `engine` which needs to be exposed on Geth (as can be seen in the examples below). This communication is made secure via JWT. Hence, it is required to generate a secret which can be done as follows:

```console
$ openssl rand -hex 32 | tr -d "\n" > "/tmp/jwtsecret"
```

This secret is then fed into Geth on startup. For example to start Geth prepared for communication with a CL you can run the following:

```console
$ geth --http --http.api="engine,eth,web3,net" --authrpc.jwtsecret=/tmp/jwtsecret
```

### Transition

The transition will happen at a pre-determined and pre-announced total difficulty, unlike usual forks which occur at a certain block number. In case of an emergency delay the so-called total terminal difficulty can be overriden by the `--override.totalterminaldifficulty`.
