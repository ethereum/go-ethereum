---
title: JSON-RPC Server
sort_key: A
---

Geth supports all standard web3 JSON-RPC APIs. You can find documentation for
these APIs on the [Ethereum Wiki JSON-RPC page][web3-rpc].

JSON-RPC is provided on multiple transports. Geth supports JSON-RPC over HTTP,
WebSocket and Unix Domain Sockets. Transports must be enabled through
command-line flags.

Ethereum JSON-RPC APIs use a name-space system. RPC methods are grouped into
several categories depending on their purpose. All method names are composed of
the namespace, an underscore, and the actual method name within the namespace.
For example, the `eth_call` method resides in the `eth` namespace.

Access to RPC methods can be enabled on a per-namespace basis. Find
documentation for individual namespaces in the sidebar.

### HTTP Server

To enable the HTTP server, use the `--rpc` flag.

    geth --rpc

By default, geth accepts connections from the loopback interface (127.0.0.1).
The default listening port is 8545. You can customize address and port using the
`--rpcport` and `--rpcaddr` flags.

    geth --rpc --rpcport 3334

JSON-RPC method namespaces must be whitelisted in order to be available through
the HTTP server. An RPC error with error code `-32602` is generated if you call a
namespace that isn't whitelisted. The default whitelist allows access to the "eth"
and "shh" namespaces. To enable access to other APIs like account management ("personal")
and debugging ("debug"), they must be configured via the `--rpcapi` flag. We do
not recommend enabling such APIs over HTTP, however, since access to these
methods increases the attack surface.

    geth --rpc --rpcapi personal,eth,net,web3b

Since the HTTP server is reachable from any local application, additional
protection is built into the server to prevent misuse of the API from web pages.
If you want enable access to the API from a web page, you must configure the
server to accept Cross-Origin requests with the `--rpccorsdomain` flag.

Example: if you want to use [Remix][remix] with geth, allow requests from the
remix domain.

    geth --rpc --rpccorsdomain https://remix.ethereum.org

Use `--rpccorsdomain '*'` to enable access from any origin.

### WebSocket Server

Configuration of the WebSocket endpoint is similar to the HTTP transport. To
enable WebSocket access, use `--ws` flag. The default WebSocket port is 8546.
The `--wsaddr`, `--wsport` and `--wsapi` flags can be used to customize settings
for the WebSocket server.

    geth --ws --wsport 3334 --wsapi eth,net,web3

Cross-Origin request protection also applies to the WebSocket server. Use the
`--wsorigins` flag to allow access to the server from web pages:

    geth --ws --wsorigins http://myapp.example.com

As with `--rpccorsdomain`, using `--wsorigins '*'` allows access from any origin.

### IPC Server

JSON-RPC APIs are also provided on a UNIX domain socket. This server is enabled
by default and has access to all JSON-RPC namespaces.

The listening socket is placed into the data directory by default. On Linux and macOS,
the default location of the geth socket is

    ~/.ethereum/geth.ipc

On Windows, IPC is provided via named pipes. The default location of the geth pipe is:

    \\.\pipe\geth.ipc
    
You can configure the location of the socket using the `--ipcpath` flag. IPC can
be disabled using the `--ipcdisable` flag.

[web3-rpc]: https://github.com/ethereum/wiki/wiki/JSON-RPC
[remix]: https://remix.ethereum.org
