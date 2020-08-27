## Binance Smart Chain

The goal of Binance Smart Chain is to bring programmability and interoperability to Binance Chain. In order to embrace the existing popular community and advanced technology, it will bring huge benefits by staying compatible with all the existing smart contracts on Ethereum and Ethereum tooling. And to achieve that, the easiest solution is to develop based on go-ethereum fork, as we respect the great work of Ethereum very much.

Binance Smart Chain starts its development based on go-ethereum fork. So you may see many toolings, binaries and also docs are based on Ethereum ones, such as the name “geth”.

But from that baseline of EVM compatible, Binance Smart Chain introduces  a system of 21 validators with Proof of Staked Authority (PoSA) consensus that can support short block time and lower fees. The most bonded validator candidates of staking will become validators and produce blocks. The double-sign detection and other slashing logic guarantee security, stability, and chain finality.

Cross-chain transfer and other communication are possible due to native support of interoperability. Relayers and on-chain contracts are developed to support that. Binance DEX remains a liquid venue of the exchange of assets on both chains. This dual-chain architecture will be ideal for users to take advantage of the fast trading on one side and build their decentralized apps on the other side. **The Binance Smart Chain** will be:

- **A self-sovereign blockchain**: Provides security and safety with elected validators.
- **EVM-compatible**: Supports all the existing Ethereum tooling along with faster finality and cheaper transaction fees.
- **Interoperable**: Comes with efficient native dual chain communication; Optimized for scaling high-performance dApps that require fast and smooth user experience.
- **Distributed with on-chain governance**: Proof of Staked Authority brings in decentralization and community participants. As the native token, BNB will serve as both the gas of smart contract execution and tokens for staking.

More details in [White Paper](http://binance.org/en#smartChain).

## Key features

### Proof of Staked Authority 
Although Proof-of-Work (PoW) has been approved as a practical mechanism to implement a decentralized network, it is not friendly to the environment and also requires a large size of participants to maintain the security. 

Proof-of-Authority(PoA) provides some defense to 51% attack, with improved efficiency and tolerance to certain levels of Byzantine players (malicious or hacked). 
Meanwhile, the PoA protocol is most criticized for being not as decentralized as PoW, as the validators, i.e. the nodes that take turns to produce blocks, have all the authorities and are prone to corruption and security attacks.

Other blockchains, such as EOS and Cosmos both, introduce different types of Deputy Proof of Stake (DPoS) to allow the token holders to vote and elect the validator set. It increases the decentralization and favors community governance. 

To combine DPoS and PoA for consensus, Binance Smart Chain implement a novel consensus engine called Parlia that:

1. Blocks are produced by a limited set of validators.
2. Validators take turns to produce blocks in a PoA manner, similar to Ethereum's Clique consensus engine.
3. Validator set are elected in and out based on a staking based governance on Binance Chain.
4. The validator set change is relayed via a cross-chain communication mechanism.
5. Parlia consensus engine will interact with a set of [system contracts](https://github.com/binance-chain/docs-site/blob/add-bsc/docs/smart-chain/guides/concepts/system-contract.md) to achieve liveness slash, revenue distributing and validator set renewing func.

 
### Light Client of Binance Chain  

To achieve the cross-chain communication from Binance Chain to Binance Smart Chain, need introduce a on-chain light client verification algorithm.
It contains two parts:

1. [Stateless Precompiled contracts](https://github.com/binance-chain/bsc/blob/master/core/vm/contracts_lightclient.go) to do tendermint header verification and Merkle Proof verification.
2. [Stateful solidity contracts](https://github.com/binance-chain/bsc-genesis-contract/blob/master/contracts/TendermintLightClient.sol) to store validator set and trusted appHash.  

## Native Token

BNB will run on Binance Smart Chain in the same way as ETH runs on Ethereum so that it remains as `native token` for BSC. This means, 
BNB will be used to:

1. pay `gas` to deploy or invoke Smart Contract on BSC
2. perform cross-chain operations, such as transfer token assets across Binance Smart Chain and Binance Chain.

## Building the source

Many of the below are the same as or similar to go-ethereum.

For prerequisites and detailed build instructions please read the [Installation Instructions](https://github.com/ethereum/go-ethereum/wiki/Building-Ethereum) on the wiki.

Building `geth` requires both a Go (version 1.13 or later) and a C compiler. You can install
them using your favourite package manager. Once the dependencies are installed, run

```shell
make geth
```

or, to build the full suite of utilities:

```shell
make all
```

## Executables

The bsc project comes with several wrappers/executables found in the `cmd`
directory.

|    Command    | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| :-----------: | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
|  **`geth`**   | Main Binance Smart Chain client binary. It is the entry point into the BSC network (main-, test- or private net), capable of running as a full node (default), archive node (retaining all historical state) or a light node (retrieving data live). It has the same and more RPC and other interface as go-ethereum and can be used by other processes as a gateway into the BSC network via JSON RPC endpoints exposed on top of HTTP, WebSocket and/or IPC transports. `geth --help` and the [CLI Wiki page](https://github.com/ethereum/go-ethereum/wiki/Command-Line-Options) for command line options.          |
|   `abigen`,`bootnode`,`evm`, `gethrpctest`,`rlpdump`,`puppeth`     | **These binaries are exactly the same as the binaries built in go-ethereum repo.**|

## Running `geth`

Going through all the possible command line flags is out of scope here (please consult our
[CLI Wiki page](https://github.com/ethereum/go-ethereum/wiki/Command-Line-Options)).

### Hardware Requirements

The hardware must meet certain requirements to run a full node.
- VPS running recent versions of Mac OS X or Linux.
- 500 GB of free disk space
- 8 cores of CPU and 16 gigabytes of memory (RAM) for mainnet.
- 4 cores of CPU and 8 gigabytes of memory (RAM) for testnet.
- A broadband Internet connection with upload/download speeds of at least 1 megabyte per second

### A Full node on the Rialto test network

Steps:

1. Download the binary, config and genesis files from [release](https://github.com/binance-chain/bsc/releases/download/v1.0.0-alpha.0/binary.zip), or compile the binary by `make geth`. 
2. Init genesis state: `./geth --datadir node init genesis.json`.
3. Start your fullnode: `./geth --config ./config.toml --datadir ./node`.
4. Or start a validator node: `./geth --config ./config.toml --datadir ./node -unlock ${validatorAddr} --mine --allow-insecure-unlock`. The ${validatorAddr} is the wallet account address of your running validator node. 

*Note: The default p2p port is 30311 and the RPC port is 8575 which is different from Ethereum.*

More details about [running a node](https://docs.binance.org/smart-chain/developer/fullnode.html) and [becoming a validator](https://docs.binance.org/smart-chain/validator/candidate.html).

*Note: Although there are some internal protective measures to prevent transactions from
crossing over between the main network and test network, you should make sure to always
use separate accounts for play-money and real-money. Unless you manually move
accounts, `geth` will by default correctly separate the two networks and will not make any
accounts available between them.*

### Programmatically interfacing `geth` nodes

As a developer, sooner rather than later you'll want to start interacting with `geth` and the
Binance Smart Chain network via your own programs and not manually through the console. To aid
this, `geth` has built-in support for a JSON-RPC based APIs ([standard APIs](https://github.com/ethereum/wiki/wiki/JSON-RPC)
and [`geth` specific APIs](https://github.com/ethereum/go-ethereum/wiki/Management-APIs)).
These can be exposed via HTTP, WebSockets and IPC (UNIX sockets on UNIX based
platforms, and named pipes on Windows).

The IPC interface is enabled by default and exposes all the APIs supported by `geth`,
whereas the HTTP and WS interfaces need to manually be enabled and only expose a
subset of APIs due to security reasons. These can be turned on/off and configured as
you'd expect.

HTTP based JSON-RPC API options:

  * `--rpc` Enable the HTTP-RPC server
  * `--rpcaddr` HTTP-RPC server listening interface (default: `localhost`)
  * `--rpcport` HTTP-RPC server listening port (default: `8545`)
  * `--rpcapi` API's offered over the HTTP-RPC interface (default: `eth,net,web3`)
  * `--rpccorsdomain` Comma separated list of domains from which to accept cross origin requests (browser enforced)
  * `--ws` Enable the WS-RPC server
  * `--wsaddr` WS-RPC server listening interface (default: `localhost`)
  * `--wsport` WS-RPC server listening port (default: `8546`)
  * `--wsapi` API's offered over the WS-RPC interface (default: `eth,net,web3`)
  * `--wsorigins` Origins from which to accept websockets requests
  * `--ipcdisable` Disable the IPC-RPC server
  * `--ipcapi` API's offered over the IPC-RPC interface (default: `admin,debug,eth,miner,net,personal,shh,txpool,web3`)
  * `--ipcpath` Filename for IPC socket/pipe within the datadir (explicit paths escape it)

You'll need to use your own programming environments' capabilities (libraries, tools, etc) to
connect via HTTP, WS or IPC to a `geth` node configured with the above flags and you'll
need to speak [JSON-RPC](https://www.jsonrpc.org/specification) on all transports. You
can reuse the same connection for multiple requests!

**Note: Please understand the security implications of opening up an HTTP/WS based
transport before doing so! Hackers on the internet are actively trying to subvert
BSC nodes with exposed APIs! Further, all browser tabs can access locally
running web servers, so malicious web pages could try to subvert locally available
APIs!**

## License

The bsc library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html),
also included in our repository in the `COPYING.LESSER` file.

The bsc binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also
included in our repository in the `COPYING` file.