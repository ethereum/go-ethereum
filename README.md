# MEV-geth

This is a fork of go-ethereum, [the original README is here](README.original.md).

Flashbots is a research and development organization formed to mitigate the negative externalities and existential risks posed by miner-extractable value (MEV) to smart-contract blockchains. We propose a permissionless, transparent, and fair ecosystem for MEV extraction that reinforce the Ethereum ideals.

## Quick start

<<<<<<< HEAD

## Building the source

For prerequisites and detailed build instructions please read the [Installation Instructions](https://geth.ethereum.org/docs/install-and-build/installing-geth).

Building `geth` requires both a Go (version 1.14 or later) and a C compiler. You can install
them using your favourite package manager. Once the dependencies are installed, run

```shell
=======
```

git clone https://github.com/flashbots/mev-geth
cd mev-geth

> > > > > > > dfdcfc666 (Add infra/CI and update README)
> > > > > > > make geth

````

See [here](https://geth.ethereum.org/docs/install-and-build/installing-geth#build-go-ethereum-from-source-code) for further info on building MEV-geth from source.

## MEV-Geth: a proof of concept

We have designed and implemented a proof of concept for permissionless MEV extraction called MEV-Geth. It is a sealed-bid block space auction mechanism for communicating transaction order preference. While our proof of concept has incomplete trust guarantees, we believe it's a significant improvement over the status quo. The adoption of MEV-Geth should relieve a lot of the network and chain congestion caused by frontrunning and backrunning bots.

| Guarantee            | PGA | Dark-txPool | MEV-Geth |
| -------------------- | :-: | :---------: | :------: |
| Permissionless       | ✅  |     ❌      |    ✅    |
| Efficient            | ❌  |     ❌      |    ✅    |
| Pre-trade privacy    | ❌  |     ✅      |    ✅    |
| Failed trade privacy | ❌  |     ❌      |    ✅    |
| Complete privacy     | ❌  |     ❌      |    ❌    |
| Finality             | ❌  |     ❌      |    ❌    |

<<<<<<< HEAD
|    Command    | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| :-----------: | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
|  **`geth`**   | Our main Ethereum CLI client. It is the entry point into the Ethereum network (main-, test- or private net), capable of running as a full node (default), archive node (retaining all historical state) or a light node (retrieving data live). It can be used by other processes as a gateway into the Ethereum network via JSON RPC endpoints exposed on top of HTTP, WebSocket and/or IPC transports. `geth --help` and the [CLI page](https://geth.ethereum.org/docs/interface/command-line-options) for command line options.          |
|   `clef`    | Stand-alone signing tool, which can be used as a backend signer for `geth`.  |
|   `devp2p`    | Utilities to interact with nodes on the networking layer, without running a full blockchain. |
|   `abigen`    | Source code generator to convert Ethereum contract definitions into easy to use, compile-time type-safe Go packages. It operates on plain [Ethereum contract ABIs](https://docs.soliditylang.org/en/develop/abi-spec.html) with expanded functionality if the contract bytecode is also available. However, it also accepts Solidity source files, making development much more streamlined. Please see our [Native DApps](https://geth.ethereum.org/docs/dapp/native-bindings) page for details. |
|  `bootnode`   | Stripped down version of our Ethereum client implementation that only takes part in the network node discovery protocol, but does not run any of the higher level application protocols. It can be used as a lightweight bootstrap node to aid in finding peers in private networks.                                                                                                                                                                                                                                                                 |
|     `evm`     | Developer utility version of the EVM (Ethereum Virtual Machine) that is capable of running bytecode snippets within a configurable environment and execution mode. Its purpose is to allow isolated, fine-grained debugging of EVM opcodes (e.g. `evm --code 60ff60ff --debug run`).                                                                                                                                                                                                                                                                     |
|   `rlpdump`   | Developer utility tool to convert binary RLP ([Recursive Length Prefix](https://eth.wiki/en/fundamentals/rlp)) dumps (data encoding used by the Ethereum protocol both network as well as consensus wise) to user-friendlier hierarchical representation (e.g. `rlpdump --hex CE0183FFFFFFC4C304050583616263`).                                                                                                                                                                                                                                 |
|   `puppeth`   | a CLI wizard that aids in creating a new Ethereum network.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
=======
### Why MEV-Geth?
>>>>>>> dfdcfc666 (Add infra/CI and update README)

We believe that without the adoption of neutral, public, open-source infrastructure for permissionless MEV extraction, MEV risks becoming an insiders' game. We commit as an organization to releasing reference implementations for participation in fair, ethical, and politically neutral MEV extraction. By doing so, we hope to prevent the properties of Ethereum from being eroded by trust-based dark pools or proprietary channels which are key points of security weakness. We thus release MEV-Geth with the dual goal of creating an ecosystem for MEV extraction that preserves Ethereum properties, as well as starting conversations with the community around our research and development roadmap.

### Design goals

- **Permissionless**
  A permissionless design implies there are no trusted intermediary which can censor transactions.
- **Efficient**
  An efficient design implies MEV extraction is performed without causing unnecessary network or chain congestion.
- **Pre-trade privacy**
  Pre-trade privacy implies transactions only become publicly known after they have been included in a block. Note, this type of privacy does not exclude privileged actors such as transaction aggregators / gateways / miners.
- **Failed trade privacy**
  Failed trade privacy implies loosing bids are never included in a block, thus never exposed to the public. Failed trade privacy is tightly coupled to extraction efficiency.
- **Complete privacy**
  Complete privacy implies there are no privileged actors such as transaction aggregators / gateways / miners who can observe incoming transactions.
- **Finality**
  Finality implies it is infeasible for MEV extraction to be reversed once included in a block. This would protect against time-bandit chain re-org attacks.

The MEV-Geth proof of concept relies on the fact that searchers can withhold bids from certain miners in order to disincentivize bad behavior like stealing a profitable strategy. We expect a complete privacy design to necessitate some sort of private computation solution like SGX, ZKP, or MPC to withhold the transaction content from miners until it is mined in a block. One of the core objective of the Flashbots organization is to incentivize and produce research in this direction.

The MEV-Geth proof of concept does not provide any finality guarantees. We expect the solution to this problem to require post-trade execution privacy through private chain state or strong economic infeasibility. The design of a system with strong finality is the second core objective of the MEV-Geth research effort.

<<<<<<< HEAD
This command will:
 * Start `geth` in fast sync mode (default, can be changed with the `--syncmode` flag),
   causing it to download more data in exchange for avoiding processing the entire history
   of the Ethereum network, which is very CPU intensive.
 * Start up `geth`'s built-in interactive [JavaScript console](https://geth.ethereum.org/docs/interface/javascript-console),
   (via the trailing `console` subcommand) through which you can interact using [`web3` methods](https://web3js.readthedocs.io/en/)
   (note: the `web3` version bundled within `geth` is very old, and not up to date with official docs),
   as well as `geth`'s own [management APIs](https://geth.ethereum.org/docs/rpc/server).
   This tool is optional and if you leave it out you can always attach to an already running
   `geth` instance with `geth attach`.

### A Full node on the Görli test network

Transitioning towards developers, if you'd like to play around with creating Ethereum
contracts, you almost certainly would like to do that without any real money involved until
you get the hang of the entire system. In other words, instead of attaching to the main
network, you want to join the **test** network with your node, which is fully equivalent to
the main network, but with play-Ether only.

```shell
$ geth --goerli console
````

=======

### How it works

> > > > > > > dfdcfc666 (Add infra/CI and update README)

MEV-Geth introduces the concepts of "searchers", "transaction bundles", and "block template" to Ethereum. Effectively, MEV-Geth provides a way for miners to delegate the task of finding and ordering transactions to third parties called "searchers". These searchers compete with each other to find the most profitable ordering and bid for its inclusion in the next block using a standardized template called a "transaction bundle". These bundles are evaluated in a sealed-bid auction hosted by miners to produce a "block template" which holds the [information about transaction order required to begin mining](https://ethereum.stackexchange.com/questions/268/ethereum-block-architecture).

![](https://hackmd.io/_uploads/B1fWz7rcD.png)

The MEV-Geth proof of concept is compatible with any regular Ethereum client. The Flashbots core devs are maintaining [a reference implementation](https://github.com/flashbots/mev-geth) for the go-ethereum client.

### Differences between MEV-Geth and [_vanilla_ geth](https://github.com/ethereum/go-ethereum)

The entire patch can be broken down into four modules:

1. bundle worker and `eth_sendBundle` rpc (commits [8104d5d7b0a54bd98b3a08479a1fde685eb53c29](https://github.com/flashbots/mev-geth/commit/8104d5d7b0a54bd98b3a08479a1fde685eb53c29) and [c2b5b4029b2b748a6f1a9d5668f12096f096563d](https://github.com/flashbots/mev-geth/commit/c2b5b4029b2b748a6f1a9d5668f12096f096563d))
2. profit switcher (commit [aa5840d22f4882f91ecba0eb20ef35a702b134d5](https://github.com/flashbots/mev-geth/commit/aa5840d22f4882f91ecba0eb20ef35a702b134d5))
3. `eth_callBundle` simulation rpc (commits [9199d2e13d484df7a634fad12343ed2b46d5d4c3](https://github.com/flashbots/mev-geth/commit/9199d2e13d484df7a634fad12343ed2b46d5d4c3) and [a99dfc198817dd171128cc22439c81896e876619](https://github.com/flashbots/mev-geth/commit/a99dfc198817dd171128cc22439c81896e876619))
4. Documentation (this file) and CI/infrastructure configuration (commit [035109807944f7a446467aa27ca8ec98d109a465](https://github.com/flashbots/mev-geth/commit/035109807944f7a446467aa27ca8ec98d109a465))

The entire changeset can be viewed inspecting the [diff](https://github.com/ethereum/go-ethereum/compare/master...flashbots:master).

In summary:

- Geth’s txpool is modified to also contain a `mevBundles` field, which stores a list of MEV bundles. Each MEV bundle is an array of transactions, along with a min/max timestamp for their inclusion.
- A new `eth_sendBundle` API is exposed which allows adding an MEV Bundle to the txpool. During the Flashbots Alpha, this is only called by MEV-relay.
  - The transactions submitted to the bundle are “eth_sendRawTransaction-style” RLP encoded signed transactions along with the min/max block of inclusion
  - This API is a no-op when run in light mode
- Geth’s miner is modified as follows:
  - While in the event loop, before adding all the pending txpool “normal” transactions to the block, it:
    - Finds the most profitable bundle
      - It picks the most profitable bundle by returning the one with the highest average gas price per unit of gas
        - computeBundleGas: Returns average gas price (\sum{gasprice_i\*gasused_i + (coinbase_after - coinbase_before)) / \sum{gasused_i})
    - Commits the bundle (remember: Bundle transactions are not ordered by nonce or gas price). For each transaction in the bundle, it:
      - `Prepare`’s it against the state
      - CommitsTransaction with trackProfit = true
        w.current.profit += coinbase_after_tx - coinbase_before_tx
        w.current.profit += gas \* gas_price
  - If a block is found where the w.current.profit is more than the previous profit, it switches mining to that block.
- A new `eth_callBundle` API is exposed that enables simulation of transaction bundles.
- Documentation and CI/infrastructure files are added.

### MEV-Geth for miners

Miners can start mining MEV blocks by running MEV-Geth, or by implementing their own fork that matches the specification.

While only the bundle worker and `eth_sendBundle` module (1) is necessary to mine flashbots blocks, we recommend also running the profit switcher module (2) to guarantee mining rewards are maximized. The `eth_callBundle` simulation rpc module (3) is not needed for the alpha. The suggested configuration is implemented in the `master` branch of this repository, which also includes the documentation module (4).

We issue and maintain [releases](https://github.com/flashbots/mev-geth/releases) for the recommended configuration for the current and immediately prior versions of geth.

In order to see the diff of the recommended patch, run:

```
 git diff master~4..master~1
```

Alternatively, the `master-barebones` branch includes only modules (1) and (4), leaving the profit switching logic to miners. While this usage is discouraged, it entails a much smaller change in the code.

At this stage, we recommend only receiving bundles via a relay, to prevent abuse via denial-of-service attacks. We have [implemented](https://github.com/flashbots/mev-relay) and currently run such relay. This relay performs basic rate limiting and miner profitability checks, but does otherwise not interfere with submitted bundles in any way, and is open for everybody to participate. We invite you to try the [Flashbots Alpha](https://github.com/flashbots/pm#flashbots-alpha) and start receiving MEV revenue by following these steps:

1. Fill out this [form](https://forms.gle/78JS52d22dwrgabi6) to indicate your interest in participating in the Alpha and be added to the MEV-Relay miner whitelist.
2. You will receive an onboarding email from Flashbots to help [set up](https://github.com/flashbots/mev-geth/blob/master/README.md#quick-start) your MEV-Geth node and protect it with a [reverse proxy](https://github.com/flashbots/mev-relay-js/blob/master/miner/proxy.js) to open the `eth_sendBundle` RPC.
3. Respond to Flashbots' email with your MEV-Geth node endpoint to be added to the Flashbots hosted [MEV-relay](https://github.com/flashbots/mev-relay-js) gateway. MEV-Relay is needed during the alpha to aggregate bundle requests from all users, prevent spam and DOS attacks on participating miner(s)/mining pool(s), and collect system health metrics.
4. After receiving a confirmation email that your MEV-Geth node's endpoint has been added to the relay, you will immediately start receiving Flashbots transaction bundles with associated MEV revenue paid to you.

### MEV-Geth for searchers

You do _not_ need to run MEV-Geth as a searcher, but, instead, to monitor the Ethereum state and transaction pool for MEV opportunities and produce transaction bundles that extract that MEV. Anyone can become a searcher. In fact, the bundles produced by searchers don't need to extract MEV at all, but we expect the most valuable bundles will.

An MEV-Geth bundle is a standard message template composed of an array of valid ethereum transactions, a blockheight, and an optional timestamp range over which the bundle is valid.

```json
{
    "signedTransactions": ['...'], // RLP encoded signed transaction array
    "blocknumber": "0x386526", // hex string
    "minTimestamp": 12345, // optional uint64
    "maxTimestamp": 12345 // optional uint64
}
```

The `signedTransactions` can be any valid ethereum transactions. Care must be taken to place transaction nonces in correct order.

The `blocknumber` defines the block height at which the bundle is to be included. A bundle will only be evaluated for the provided blockheight and immediately evicted if not selected.

The `minTimestamp` and `maxTimestamp` are optional conditions to further restrict bundle validity within a time range.

MEV-Geth miners select the most profitable bundle per unit of gas used and place it at the beginning of the list of transactions of the block template at a given blockheight. Miners determine the value of a bundle based on the following equation. _Note, the change in block.coinbase balance represents a direct transfer of ETH through a smart contract._

<img width="544" src="https://hackmd.io/_uploads/Bk6iQmr5P.png">

To submit a bundle, the searcher sends the bundle directly to the miner using the rpc method `eth_sendBundle`. Since MEV-Geth requires direct communication between searchers and miners, a searcher can configure the list of miners where they want to send their bundle.

### Feature requests and bug reports

If you are a user of MEV-Geth and have suggestions on how to make integration with your current setup easier, or would like to submit a bug report, we encourage you to open an issue in this repository with the `enhancement` or `bug` labels respectively. If you need help getting started, please ask in the dedicated [#⛏️miners](https://discord.gg/rcgADN9qFX) channel in our Discord.

### Moving beyond proof of concept

We provide the MEV-Geth proof of concept as a first milestone on the path to mitigating the negative externalities caused by MEV. We hope to discuss with the community the merits of adopting MEV-Geth in its current form. Our preliminary research indicates it could free at least 2.5% of the current chain congestion by eliminating the use of frontrunning and backrunning and provide uplift of up to 18% on miner rewards from Ethereum. That being said, we believe a sustainable solution to MEV existential risks requires complete privacy and finality, which the proof of concept does not address. We hope to engage community feedback throughout the development of this complete version of MEV-Geth.
