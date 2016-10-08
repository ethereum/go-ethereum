## Expanse Go

Expanse Go Client, by Christopher Franko (forked from Jeffrey Wilcke (and some other people)'s Expanse Go client).

          | Linux   | OSX | ARM | Windows | Tests
----------|---------|-----|-----|---------|------
develop   | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=Linux%20Go%20develop%20branch)](https://build.ethdev.com/builders/Linux%20Go%20develop%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=Linux%20Go%20develop%20branch)](https://build.ethdev.com/builders/OSX%20Go%20develop%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=ARM%20Go%20develop%20branch)](https://build.ethdev.com/builders/ARM%20Go%20develop%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=Windows%20Go%20develop%20branch)](https://build.ethdev.com/builders/Windows%20Go%20develop%20branch/builds/-1) | [![Buildr+Status](https://travis-ci.org/expanse-project/go-expanse.svg?branch=develop)](https://travis-ci.org/expanse/go-expanse) [![Coverage Status](https://coveralls.io/repos/expanse-project/go-expanse/badge.svg?branch=develop)](https://coveralls.io/r/expanse/go-expanse?branch=develop)
master    | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=Linux%20Go%20master%20branch)](https://build.ethdev.com/builders/Linux%20Go%20master%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=OSX%20Go%20master%20branch)](https://build.ethdev.com/builders/OSX%20Go%20master%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=ARM%20Go%20master%20branch)](https://build.ethdev.com/builders/ARM%20Go%20master%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=Windows%20Go%20master%20branch)](https://build.ethdev.com/builders/Windows%20Go%20master%20branch/builds/-1) | [![Buildr+Status](https://travis-ci.org/expanse-project/go-expanse.svg?branch=master)](https://travis-ci.org/expanse-project/go-expanse) [![Coverage Status](https://coveralls.io/repos/expanse-project/go-expanse/badge.svg?branch=master)](https://coveralls.io/r/expanse-project/go-expanse?branch=master)

[![Bugs](https://badge.waffle.io/expanse-project/go-expanse.png?label=bug&title=Bugs)](https://waffle.io/expanse/go-expanse)
[![Stories in Ready](https://badge.waffle.io/expanse-project/go-expanse.png?label=ready&title=Ready)](https://waffle.io/expanse/go-expanse)
[![Stories in Progress](https://badge.waffle.io/expanse-project/go-expanse.svg?label=in%20progress&title=In Progress)](http://waffle.io/expanse/go-expanse)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/expanse/go-expanse?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)
## Automated development builds

The following builds are build automatically by our build servers after each push to the [develop](https://github.com/expanse-project/go-expanse/tree/develop) branch.

* [Docker](https://registry.hub.docker.com/u/expanse/go-expanse/)
* [OS X](http://build.ethdev.com/builds/OSX%20Go%20develop%20branch/Mist-OSX-latest.dmg)
* Ubuntu
  [trusty](https://build.ethdev.com/builds/Linux%20Go%20develop%20deb%20i386-trusty/latest/) |
  [utopic](https://build.ethdev.com/builds/Linux%20Go%20develop%20deb%20i386-utopic/latest/)
* [Windows 64-bit](https://build.ethdev.com/builds/Windows%20Go%20develop%20branch/Gexp-Win64-latest.zip)
* [ARM](https://build.ethdev.com/builds/ARM%20Go%20develop%20branch/gexp-ARM-latest.tar.bz2)

## Building the source

For prerequisites and detailed build instructions please read the
[Installation Instructions](https://github.com/expanse-project/go-expanse/wiki/Building-Expanse)
on the wiki.

Building gexp requires both a Go and a C compiler.
You can install them using your favourite package manager.
Once the dependencies are installed, run

    make gexp

or, to build the full suite of utilities:

    make all

## Executables

Go Expanse comes with several wrappers/executables found in
[the `cmd` directory](https://github.com/expanse-project/go-expanse/tree/develop/cmd):

* `gexp` Expanse CLI (expanse command line interface client)
* `bootnode` runs a bootstrap node for the Discovery Protocol
* `exptest` test tool which runs with the [tests](https://github.com/expanse-project/tests) suite:
  `/path/to/test.json > exptest --test BlockTests --stdin`.
* `evm` is a generic Expanse Virtual Machine: `evm -code 60ff60ff -gas
  10000 -price 0 -dump`. See `-h` for a detailed description.
* `disasm` disassembles EVM code: `echo "6001" | disasm`
* `rlpdump` prints RLP structures

## Running geth

Going through all the possible command line flags is out of scope here (please consult our
[CLI Wiki page](https://github.com/expanse-project/go-expanse/wiki/Command-Line-Options)), but we've
enumerated a few common parameter combos to get you up to speed quickly on how you can run your
own  instance.

### Full node on the main Ethereum network

By far the most common scenario is people wanting to simply interact with the Ethereum network:
create accounts; transfer funds; deploy and interact with contracts. For this particular use-case
the user doesn't care about years-old historical data, so we can fast-sync quickly to the current
state of the network. To do so:

```
$ geth --fast --cache=512 console
```

This command will:

 * Start geth in fast sync mode (`--fast`), causing it to download more data in exchange for avoiding
   processing the entire history of the Ethereum network, which is very CPU intensive.
 * Bump the memory allowance of the database to 512MB (`--cache=512`), which can help significantly in
   sync times especially for HDD users. This flag is optional and you can set it as high or as low as
   you'd like, though we'd recommend the 512MB - 2GB range.
 * Start up 's built-in interactive [JavaScript console](https://github.com/expanse-project/go-expanse/wiki/JavaScript-Console),
   (via the trailing `console` subcommand) through which you can invoke all official [`web3` methods](https://github.com/ethereum/wiki/wiki/JavaScript-API)
   as well as 's own [management APIs](https://github.com/expanse-project/go-expanse/wiki/Management-APIs).
   This too is optional and if you leave it out you can always attach to an already running  instance
   with `geth --attach`.

### Full node on the Ethereum test network

Transitioning towards developers, if you'd like to play around with creating Ethereum contracts, you
almost certainly would like to do that without any real money involved until you get the hang of the
entire system. In other words, instead of attaching to the main network, you want to join the **test**
network with your node, which is fully equivalent to the main network, but with play-Ether only.

```
$ geth --testnet --fast --cache=512 console
```

The `--fast`, `--cache` flags and `console` subcommand have the exact same meaning as above and they
are equially useful on the testnet too. Please see above for their explanations if you've skipped to
here.

Specifying the `--testnet` flag however will reconfigure your  instance a bit:

 * Instead of using the default data directory (`~/.ethereum` on Linux for example),  will nest
   itself one level deeper into a `testnet` subfolder (`~/.ethereum/testnet` on Linux).
 * Instead of connecting the main Ethereum network, the client will connect to the test network,
   which uses different P2P bootnodes, different network IDs and genesis states.

*Note: Although there are some internal protective measures to prevent transactions from crossing
over between the main network and test network (different starting nonces), you should make sure to
always use separate accounts for play-money and real-money. Unless you manually move accounts, 
will by default correctly separate the two networks and will not make any accounts available between
them.*

### Programatically interfacing  nodes

As a developer, sooner rather than later you'll want to start interacting with  and the Ethereum
network via your own programs and not manually through the console. To aid this,  has built in
support for a JSON-RPC based APIs ([standard APIs](https://github.com/ethereum/wiki/wiki/JSON-RPC) and
[ specific APIs](https://github.com/expanse-project/go-expanse/wiki/Management-APIs)). These can be
exposed via HTTP, WebSockets and IPC (unix sockets on unix based platroms, and named pipes on Windows).

The IPC interface is enabled by default and exposes all the APIs supported by , whereas the HTTP
and WS interfaces need to manually be enabled and only expose a subset of APIs due to security reasons.
These can be turned on/off and configured as you'd expect.

HTTP based JSON-RPC API options:

  * `--rpc` Enable the HTTP-RPC server
  * `--rpcaddr` HTTP-RPC server listening interface (default: "localhost")
  * `--rpcport` HTTP-RPC server listening port (default: 8545)
  * `--rpcapi` API's offered over the HTTP-RPC interface (default: "eth,net,web3")
  * `--rpccorsdomain` Comma separated list of domains from which to accept cross origin requests (browser enforced)
  * `--ws` Enable the WS-RPC server
  * `--wsaddr` WS-RPC server listening interface (default: "localhost")
  * `--wsport` WS-RPC server listening port (default: 8546)
  * `--wsapi` API's offered over the WS-RPC interface (default: "eth,net,web3")
  * `--wsorigins` Origins from which to accept websockets requests
  * `--ipcdisable` Disable the IPC-RPC server
  * `--ipcapi` API's offered over the IPC-RPC interface (default: "admin,debug,eth,miner,net,personal,shh,txpool,web3")
  * `--ipcpath` Filename for IPC socket/pipe within the datadir (explicit paths escape it)

You'll need to use your own programming environments' capabilities (libraries, tools, etc) to connect
via HTTP, WS or IPC to a  node configured with the above flags and you'll need to speak [JSON-RPC](http://www.jsonrpc.org/specification)
on all transports. You can reuse the same connection for multiple requests!

**Note: Please understand the security implications of opening up an HTTP/WS based transport before
doing so! Hackers on the internet are actively trying to subvert Ethereum nodes with exposed APIs!
Further, all browser tabs can access locally running webservers, so malicious webpages could try to
subvert locally available APIs!**

### Operating a private network

Maintaining your own private network is more involved as a lot of configurations taken for granted in
the official networks need to be manually set up.

#### Defining the private genesis state

First, you'll need to create the genesis state of your networks, which all nodes need to be aware of
and agree upon. This consists of a small JSON file (e.g. call it `genesis.json`):

```json
{
  "alloc"      : {},
  "coinbase"   : "0x0000000000000000000000000000000000000000",
  "difficulty" : "0x20000",
  "extraData"  : "",
  "gasLimit"   : "0x2fefd8",
  "nonce"      : "0x0000000000000042",
  "mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
  "parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
  "timestamp"  : "0x00"
}
```

The above fields should be fine for most purposes, although we'd recommend changing the `nonce` to
some random value so you prevent unknown remote nodes from being able to connect to you. If you'd
like to pre-fund some accounts for easier testing, you can populate the `alloc` field with account
configs:

```json
"alloc": {
  "0x0000000000000000000000000000000000000001": {"balance": "111111111"},
  "0x0000000000000000000000000000000000000002": {"balance": "222222222"}
}
```

With the genesis state defined in the above JSON file, you'll need to initialize **every**  node
with it prior to starting it up to ensure all blockchain parameters are correctly set:

```
$ geth init path/to/genesis.json
```

#### Creating the rendezvous point

With all nodes that you want to run initialized to the desired genesis state, you'll need to start a
bootstrap node that others can use to find each other in your network and/or over the internet. The
clean way is to configure and run a dedicated bootnode:

```
$ bootnode --genkey=boot.key
$ bootnode --nodekey=boot.key
```

With the bootnode online, it will display an [`enode` URL](https://github.com/ethereum/wiki/wiki/enode-url-format)
that other nodes can use to connect to it and exchange peer information. Make sure to replace the
displayed IP address information (most probably `[::]`) with your externally accessible IP to get the
actual `enode` URL.

*Note: You could also use a full fledged  node as a bootnode, but it's the less recommended way.*

#### Starting up your member nodes

With the bootnode operational and externally reachable (you can try `telnet <ip> <port>` to ensure
it's indeed reachable), start every subsequent  node pointed to the bootnode for peer discovery
via the `--bootnodes` flag. It will probably also be desirable to keep the data directory of your
private network separated, so do also specify a custom `--datadir` flag.

```
$ geth --datadir=path/to/custom/data/folder --bootnodes=<bootnode-enode-url-from-above>
```

*Note: Since your network will be completely cut off from the main and test networks, you'll also
need to configure a miner to process transactions and create new blocks for you.*

#### Running a private miner

Mining on the public Ethereum network is a complex task as it's only feasible using GPUs, requiring
an OpenCL or CUDA enabled `ethminer` instance. For information on such a setup, please consult the
[EtherMining subreddit](https://www.reddit.com/r/EtherMining/) and the [Genoil miner](https://github.com/Genoil/cpp-ethereum)
repository.

In a private network setting however, a single CPU miner instance is more than enough for practical
purposes as it can produce a stable stream of blocks at the correct intervals without needing heavy
resources (consider running on a single thread, no need for multiple ones either). To start a 
instance for mining, run it with all your usual flags, extended by:

```
$ geth <usual-flags> --mine --minerthreads=1 --etherbase=0x0000000000000000000000000000000000000000
```

Which will start mining bocks and transactions on a single CPU thread, crediting all proceedings to
the account specified by `--etherbase`. You can further tune the mining by changing the default gas
limit blocks converge to (`--targetgaslimit`) and the price transactions are accepted at (`--gasprice`).

## Contribution

`gexp` can be configured via command line options, environment variables and config files.

If you'd like to contribute to go-expanse, please fork, fix, commit and send a pull request
for the maintainers to review and merge into the main code base. If you wish to submit more
complex changes though, please check up with the core devs first on [our gitter channel](https://gitter.im/expanse-project/go-expanse)
to ensure those changes are in line with the general philosophy of the project and/or get some
early feedback which can make both your efforts much lighter as well as our review and merge
procedures quick and simple.

Please make sure your contributions adhere to our coding guidelines:

 * Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
 * Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary) guidelines.
 * Pull requests need to be based on and opened against the `develop` branch.
 * Commit messages should be prefixed with the package(s) they modify.
   * E.g. "exp, rpc: make trace configs optional"


Please see the [Developers' Guide](https://github.com/expanse-project/go-expanse/wiki/Developers'-Guide)
for more details on configuring your environment, managing project dependencies and testing procedures.

## License

The go-expanse library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](http://www.gnu.org/licenses/lgpl-3.0.en.html), also
included in our repository in the `COPYING.LESSER` file.

The go-expanse binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](http://www.gnu.org/licenses/gpl-3.0.en.html), also included
in our repository in the `COPYING` file.


