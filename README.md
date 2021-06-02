## Coinex Smart Chain
CoinEx is committed to product development and service improvement, and contributes its share to the infrastructure of the blockchain world. `CSC` is a small part of the development prospect.

CoinEx Smart Chain (CSC) is a smart contract chain that supports up to 101 validators. Aside from shorter time for block generation and lower fees for transactions, `CSC` is also fully compatible with Ethereum virtual machine(EVM) and protocols while supporting high-performance transactions. And to achieve that, the easiest solution is to develop based on go-ethereum fork, as we respect the great work of Ethereum very much.

## CSC Features

* Decentralization: Permission-free, anyone can become a validator by staking `CET`.
* EVM compatibility: Fully compatible with Ethereum virtual machine(EVM), which means that almost all DApps, ecosystem components and tools on Ethereum can be migrated to `CSC` directly or with very small changes.
* High performance: 600+ TPS, 3s block time

## Native Token

`CET` on `CSC` runs the same way as `ETH`runs on Ethereum, and its main functions are:

* As block reward for validators
* To pay for the gas for transfers and contract calls on `CSC` 
* To pay for the transaction fees for deploying smart contracts on `CSC`
* To be delegated to the selected validators

## Building the source

For prerequisites and detailed build instructions please read the [Compile](https://docs.coinex.org/#/en-us/node_compile).

Building `cetd` requires both a Go (version 1.13 or later) and a C compiler. You can install
them using your favourite package manager. Once the dependencies are installed, run

```shell
make cetd
```

## Running `cetd`

Going through all the possible command line flags is out of scope here (please consult our
[CLI](https://github.com/ethereum/go-ethereum/wiki/Command-Line-Options)),
but we've enumerated a few common parameter combos to get you up to speed quickly
on how you can run your own `cetd` instance.

### Hardware Requirements

* 1T of SSD storage for mainnet, 500G of SSD storage for testnet.
* 16 cores of CPU and 32 gigabytes of memory (RAM) for mainnet.
* 4 cores of CPU and 8 gigabytes of memory (RAM) for testnet.
* A broadband Internet connection with upload/download speeds of at least 10 megabyte per second

### Full node on the testnet

By far the most common scenario is people wanting to simply interact with the `CSC`
network: create accounts; transfer funds; deploy and interact with contracts. For this
particular use-case the user doesn't care about years-old historical data, so we can
fast-sync quickly to the current state of the network. To do so:

```shell
$ cetd console
```

This command will:
 * Start `cetd` in fast sync mode (default, can be changed with the `--syncmode` flag),
   causing it to download more data in exchange for avoiding processing the entire history
   of the Ethereum network, which is very CPU intensive.
 * Start up `cetd`'s built-in interactive [JavaScript console](https://github.com/ethereum/go-ethereum/wiki/JavaScript-Console),
   (via the trailing `console` subcommand) through which you can invoke all official [`web3` methods](https://github.com/ethereum/wiki/wiki/JavaScript-API)
   as well as `cetd`'s own [management APIs](https://github.com/ethereum/go-ethereum/wiki/Management-APIs).
   This tool is optional and if you leave it out you can always attach to an already running
   `cetd` instance with `cetd attach`.


### Configuration

As an alternative to passing the numerous flags to the `cetd` binary, you can also pass a
configuration file via:

```shell
$ cetd --config /path/to/your_config.toml
```

### Programmatically interfacing `cetd` nodes

As a developer, sooner rather than later you'll want to start interacting with `cetd` and the
`CSC` network via your own programs and not manually through the console. To aid
this, `cetd` has built-in support for a JSON-RPC based APIs ([standard APIs](https://github.com/ethereum/wiki/wiki/JSON-RPC)
and [specific APIs](https://github.com/ethereum/go-ethereum/wiki/Management-APIs)).
These can be exposed via HTTP, WebSockets and IPC (UNIX sockets on UNIX based
platforms, and named pipes on Windows).

The IPC interface is enabled by default and exposes all the APIs supported by `cetd`,
whereas the HTTP and WS interfaces need to manually be enabled and only expose a
subset of APIs due to security reasons. These can be turned on/off and configured as
you'd expect.

HTTP based JSON-RPC API options:

  * `--http` Enable the HTTP-RPC server
  * `--http.addr` HTTP-RPC server listening interface (default: `localhost`)
  * `--http.port` HTTP-RPC server listening port (default: `8545`)
  * `--http.api` API's offered over the HTTP-RPC interface (default: `eth,net,web3`)
  * `--http.corsdomain` Comma separated list of domains from which to accept cross origin requests (browser enforced)
  * `--ws` Enable the WS-RPC server
  * `--ws.addr` WS-RPC server listening interface (default: `localhost`)
  * `--ws.port` WS-RPC server listening port (default: `8546`)
  * `--ws.api` API's offered over the WS-RPC interface (default: `eth,net,web3`)
  * `--ws.origins` Origins from which to accept websockets requests
  * `--ipcdisable` Disable the IPC-RPC server
  * `--ipcapi` API's offered over the IPC-RPC interface (default: `admin,debug,eth,miner,net,personal,shh,txpool,web3`)
  * `--ipcpath` Filename for IPC socket/pipe within the datadir (explicit paths escape it)

You'll need to use your own programming environments' capabilities (libraries, tools, etc) to
connect via HTTP, WS or IPC to a `cetd` node configured with the above flags and you'll
need to speak [JSON-RPC](https://www.jsonrpc.org/specification) on all transports. You
can reuse the same connection for multiple requests!

**Note: Please understand the security implications of opening up an HTTP/WS based
transport before doing so! Hackers on the internet are actively trying to subvert
`CSC` nodes with exposed APIs! Further, all browser tabs can access locally
running web servers, so malicious web pages could try to subvert locally available
APIs!**

## Contribution

Thank you for considering to help out with the source code! We welcome contributions
from anyone on the internet, and are grateful for even the smallest of fixes!

If you'd like to contribute to `csc`, please fork, fix, commit and send a pull request
for the maintainers to review and merge into the main code base. If you wish to submit
more complex changes though, please contact to `the core developer`(`developer@coinex.org`)
to ensure those changes are in line with the general philosophy of the project and/or get
some early feedback which can make both your efforts much lighter as well as our review
and merge procedures quick and simple.

Please make sure your contributions adhere to our coding guidelines:

 * Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting)
   guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
 * Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary)
   guidelines.
 * Pull requests need to be based on and opened against the `master` branch.
 * Commit messages should be prefixed with the package(s) they modify.
   * E.g. "eth, rpc: make trace configs optional"

Please see the [Developers' Guide](https://github.com/ethereum/go-ethereum/wiki/Developers'-Guide)
for more details on configuring your environment, managing project dependencies, and
testing procedures.

## License

The csc library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html),
also included in our repository in the `COPYING.LESSER` file.

The csc binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also
included in our repository in the `COPYING` file.
