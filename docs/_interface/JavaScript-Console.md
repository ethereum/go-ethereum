---
title: JavaScript Console
sort_key: D
---

Geth responds to instructions encoded as JSON objects as defined in the [JSON-RPC-API](../_rpc/). A Geth user can send these instructions directly, for example over HTTP using tools like [Curl](https://github.com/curl/curl). The code snippet below shows a request for an account balance sent to a local Geth node with the HTTP port `8545` exposed. 

```
curl --data '{"jsonrpc":"2.0","method":"eth_getBalance", "params": ["0x9b1d35635cc34752ca54713bb99d38614f63c955", "latest"], "id":2}' -H "Content-Type: application/json" localhost:8545

```

This returns a result which is also a JSON object, with values expressed as hexadecimal strings, for example:

```terminal

{"id":2,"jsonrpc":"2.0","result":"0x1639e49bba16280000"}

```

While this approach is valid, it is also a very low level and rather error-prone way to interact with Geth. Most developers prefer to use convenience libraries that abstract away some of the more tedious and awkward tasks such as converting values from hexadecimal strings into numbers, or converting between denominations of ether (Wei, Gwei, etc). One such library is [Web3.js](https://web3js.readthedocs.io/en/v1.7.3/). This is a collection of Javascript libraries for interacting with an Ethereum node at a higher level than sending raw JSON objects to the node. The purpose of Geth's Javascript console is to provide a built-in environment to use a subset of the Web3.js libraries to interact with a Geth node.

{% include note.html content="The web3.js version that comes bundled with Geth is not up to date with the official Web3.js documentation. There are several Web3.js libraries that are not available in the Geth Javascript Console. There are also administrative APIs included in the Geth console that are not documented in the Web3.jc documentation. The full list of libraries available in the Geth console is available on the [JSON-RPC API page](../_rpc/)" %}


## Starting the console

There are two ways to start an interactive session using Geth console. The first is to provide the `console` flag when Geth is started up. This starts the node and runs the console in the same terminal. It is therefore convenient to suppress the logs from the node to prevent them from obscuring the console. If the logs are not needed, they can be redirected to the `dev/null` path, effectively muting them. Alternatively, if the logs are required they can be redirected to a text file. The level of detail provided in the logs can be adjusted by providing a value between 1-6 to the `--verbosity` flag as in the example below:

```shell
# to mute logs
geth <other commands> console 2 > /dev/null

# to save logs to file
geth <other commands> --verbosity 3 2 > geth-logs.log

```

Alternatively, a Javascript console can be attached to an existing Geth instance (i.e. one that is running in another terminal or remotely). In this case, `geth attach` can be used to open a Javascript console connected to the Geth node. It is also necessary to define the method used to connect the console to the node. Geth supports websockets, HTTP or local IPC. To use HTTP or Websockets, these must be enabled at the node by providing the following flags at startup:

```shell

# enable websockets
geth <other commands> --ws 

# enable http

geth <other commands> --http

```

The commands above use default HTTP/WS endpoints and only enables the default JSON-RPC libraries. To update the Websockets or HTTP endpoints used, or to add support for additional libraries, the `.addr` `.port` and `.api` flags can be used as follows:

```shell

# define a custom http adress, custom http port and enable libraries
geth <other commands> --http --http.addr 192.60.52.21 --http.port 8552 --http.api eth,web3,admin

# define a custom Websockets address and enable libraries
geth <other commands> --ws --ws.addr 192.60.52.21 --ws.port 8552 --ws.api eth,web3,admin

```

It is important to note that by default some functionality, including account unlocking is forbidden when HTTP or Websockets access is enabled. This is because an attacker that manages to access the node via the externally-exposed HTTP/WS port then control the unlocked account. It is possible to force account unlock by including the `--allow-insecure-unlock` flag but this is not recommended if there is any chance of the node connecting to Ethereum Mainnet. This is not a hypothetical risk: **there are bots that continually scan for http-enabled Ethereum nodes to attack**"

The Javascript console can also be connected to a Geth node using IPC. When Geth is started, a `geth.ipc` file is automatically generated and saved to the data directory. This file, or a custom path to a specific ipc file can be passed to `geth attach` as follows:

```shell

geth attach datadir/geth.ipc

```






Geth has support to load custom JavaScript files into the console through the `--preload`
option. This can be used to load often used functions, or to setup web3 contract objects.

    geth console --preload "/my/scripts/folder/utils.js,/my/scripts/folder/contracts.js"

## Non-interactive Use: Script Mode

It's also possible to execute files to the JavaScript interpreter. The `console` and
`attach` subcommand accept the `--exec` argument which is a javascript statement.

    geth attach --exec "eth.blockNumber"

This prints the current block number of a running geth instance.

Or execute a local script with more complex statements on a remote node over http:

    geth attach http://geth.example.org:8545 --exec 'loadScript("/tmp/checkbalances.js")'
    geth attach http://geth.example.org:8545 --jspath "/tmp" --exec 'loadScript("checkbalances.js")'

Use the `--jspath <path/to/my/js/root>` to set a library directory for your js scripts.
Parameters to `loadScript()` with no absolute path will be understood relative to this
directory.

You can exit the console by typing `exit` or simply with `CTRL-C`.

## Caveats

go-ethereum now uses the [GoJa JS VM](https://github.com/dop251/goja) which is compatible with ECMAScript 5.1. There are some limitations though:

  * Promises and `async` won't work.

`web3.js` uses the [`bignumber.js`](https://github.com/MikeMcl/bignumber.js) library.
This library is auto-loaded into the console.

### Timers

In addition to the full functionality of JS (as per ECMA5), the ethereum JSRE is augmented
with various timers. It implements `setInterval`, `clearInterval`, `setTimeout`,
`clearTimeout` you may be used to using in browser windows. It also provides
implementation for `admin.sleep(seconds)` and a block based timer, `admin.sleepBlocks(n)`
which sleeps till the number of new blocks added is equal to or greater than `n`, think
"wait for n confirmations".
