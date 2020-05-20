## Disclaimer

**The software and related documentation are under active development, 
all subjected to potential future change without notification and not ready for production use. 
The code and security audit have not been fully completed and not ready for any bug bounty.
We advise you to be careful and experiment on the network at your own risk. Stay safe out there.**

## Binance Smart Chain

Binance Smart Chain client is based on the go-ethereum fork so that it stays compatible with all the existing smart contracts and Ethereum tooling.

Binance Smart Chain is an innovative solution to bring programmability and interoperability to Binance Chain. Binance Smart Chain relies on a system of 21 validators with Proof of Staked Authority (PoSA) consensus that can support short block time and lower fees. The most bonded validator candidates of staking will become validators and produce blocks. The double-sign detection and other slashing logic guarantee security, stability, and chain finality.

The Binance Smart Chain also supports EVM-compatible smart contracts and protocols. Cross-chain transfer and other communication are possible due to native support of interoperability. Binance DEX remains a liquid venue of the exchange of assets on both chains. This dual-chain architecture will be ideal for users to take advantage of the fast trading on one side and build their decentralized apps on the other side. **The Binance Smart Chain** will be:

- **A self-sovereign blockchain**: Provides security and safety with elected validators.
- **EVM-compatible**: Supports all the existing Ethereum tooling along with faster finality and cheaper transaction fees.
- **Interoperable**: Comes with efficient native dual chain communication; Optimized for scaling high-performance dApps that require fast and smooth user experience.
- **Distributed with on-chain governance**: Proof of Staked Authority brings in decentralization and community participants. As the native token, BNB will serve as both the gas of smart contract execution and tokens for staking.

More details in [White Paper](http://binance.org/en#smartChain).

## Building the source

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
|  **`geth`**   | Our main Binance Smart Chain client. It is the entry point into the BSC network (main-, test- or private net), capable of running as a full node (default), archive node (retaining all historical state) or a light node (retrieving data live). It can be used by other processes as a gateway into the BSC network via JSON RPC endpoints exposed on top of HTTP, WebSocket and/or IPC transports. `geth --help` and the [CLI Wiki page](https://github.com/ethereum/go-ethereum/wiki/Command-Line-Options) for command line options.          |
|   `abigen`    | Source code generator to convert Ethereum contract definitions into easy to use, compile-time type-safe Go packages. It operates on plain [Ethereum contract ABIs](https://github.com/ethereum/wiki/wiki/Ethereum-Contract-ABI) with expanded functionality if the contract bytecode is also available. However, it also accepts Solidity source files, making development much more streamlined. Please see our [Native DApps](https://github.com/ethereum/go-ethereum/wiki/Native-DApps:-Go-bindings-to-Ethereum-contracts) wiki page for details. |
|  `bootnode`   | Stripped down version of our Binance Smart Chain client implementation that only takes part in the network node discovery protocol, but does not run any of the higher level application protocols. It can be used as a lightweight bootstrap node to aid in finding peers in private networks.                                                                                                                                                                                                                                                                 |
|     `evm`     | Developer utility version of the EVM (Ethereum Virtual Machine) that is capable of running bytecode snippets within a configurable environment and execution mode. Its purpose is to allow isolated, fine-grained debugging of EVM opcodes (e.g. `evm --code 60ff60ff --debug run`).                                                                                                                                                                                                                                                                     |
| `gethrpctest` | Developer utility tool to support our [ethereum/rpc-test](https://github.com/ethereum/rpc-tests) test suite which validates baseline conformity to the [Ethereum JSON RPC](https://github.com/ethereum/wiki/wiki/JSON-RPC) specs. Please see the [test suite's readme](https://github.com/ethereum/rpc-tests/blob/master/README.md) for details.                                                                                                                                                                                                     |
|   `rlpdump`   | Developer utility tool to convert binary RLP ([Recursive Length Prefix](https://github.com/ethereum/wiki/wiki/RLP)) dumps (data encoding used by the Ethereum protocol both network as well as consensus wise) to user-friendlier hierarchical representation (e.g. `rlpdump --hex CE0183FFFFFFC4C304050583616263`).                                                                                                                                                                                                                                 |
|   `puppeth`   | a CLI wizard that aids in creating a new network.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |

## Running `geth`

Going through all the possible command line flags is out of scope here (please consult our
[CLI Wiki page](https://github.com/ethereum/go-ethereum/wiki/Command-Line-Options)).

### Minimum Requirements

The hardware must meet certain requirements to run a full node.
- VPS running recent versions of Mac OS X or Linux.
- 500 GB of free disk space
- 8 cores of CPU and 16 gigabytes of memory (RAM) for mainnet.
- 4 cores of CPU and 8 gigabytes of memory (RAM) for testnet.
- A broadband Internet connection with upload/download speeds of at least 1 megabyte per second

### A Full node on the Rialto test network

Transitioning towards developers, if you'd like to play around with creating smart
contracts, you almost certainly would like to do that without any real money involved until
you get the hang of the entire system. In other words, instead of attaching to the main
network, you want to join the **test** network with your node, which is fully equivalent to
the main network, but with play-BNB only.

Steps:

1. Download the config and genesis files. You need to have [genesis.json](https://github.com/binance-chain/smart-chain-binary/blob/pre-release/bsc/fullnode/config/genesis.json) and [config.toml](https://github.com/binance-chain/smart-chain-binary/blob/pre-release/bsc/fullnode/config/config.toml)
2. Init genesis state: `./geth --datadir node init genesis.json`.
3. Start your fullnode: `./geth --config ./config.toml --datadir ./node`.
4. Or start a validator node: `./geth --config ./config.toml --datadir ./node -unlock ${validatorAddr} --mine --allow-insecure-unlock`. The ${validatorAddr} is the wallet account address of your running validator node. 

More details about [running a node](https://github.com/binance-chain/docs-site/blob/add-bsc/docs/smart-chain/developer/fullnode.md) and [becoming a validator](https://github.com/binance-chain/docs-site/blob/add-bsc/docs/guides/concepts/bc-staking.md).

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
