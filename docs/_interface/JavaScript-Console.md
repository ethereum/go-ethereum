---
title: JavaScript Console
sort_key: B
---

The Geth JavaScript console exposes the full [web3 JavaScript Dapp
API](https://github.com/ethereum/wiki/wiki/JavaScript-API) and further administrative
APIs.

## Interactive Use: The Console

The geth JavaScript console is started with the `console` or `attach` geth sub-commands.
The `console` subcommands starts the geth node and then opens the console. The `attach`
subcommand attaches to the console to an already-running geth instance.

    geth console
    geth attach

Attach mode accepts an endpoint in case the geth node is running with a non default
ipc endpoint or you would like to connect over the rpc interface.

    geth attach /some/custom/path.ipc
    geth attach http://191.168.1.1:8545
    geth attach ws://191.168.1.1:8546

Note that by default the geth node doesn't start the HTTP and WebSocket servers and not
all functionality is provided over these interfaces for security reasons. These defaults
can be overridden with the `--rpcapi` and `--wsapi` arguments when the geth node is
started, or with [admin.startRPC](../rpc/ns-admin#admin_startrpc) and
[admin.startWS](../rpc/ns-admin#admin_startws).

If you need log information, start with:

    geth console --verbosity 5 2>> /tmp/eth.log

Otherwise mute your logs, so that it does not pollute your console:

    geth console 2> /dev/null

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

go-ethereum uses the [Otto JS VM](https://github.com/robertkrimen/otto) which has some
limitations:

* `"use strict"` will parse, but does nothing.
* The regular expression engine (re2/regexp) is not fully compatible with the ECMA5
  specification.

`web3.js` uses the [`bignumber.js`](https://github.com/MikeMcl/bignumber.js) library.
This library is auto-loaded into the console.

### Timers

In addition to the full functionality of JS (as per ECMA5), the ethereum JSRE is augmented
with various timers. It implements `setInterval`, `clearInterval`, `setTimeout`,
`clearTimeout` you may be used to using in browser windows. It also provides
implementation for `admin.sleep(seconds)` and a block based timer, `admin.sleepBlocks(n)`
which sleeps till the number of new blocks added is equal to or greater than `n`, think
"wait for n confirmations".
