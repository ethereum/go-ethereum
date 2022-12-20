---
title: JSON-RPC Server
description: Introduction to the JSON_RPC server
---

Interacting with Geth requires sending requests to specific JSON-RPC API methods. Geth supports all standard [JSON-RPC API](https://github.com/ethereum/execution-apis) endpoints.
The RPC requests must be sent to the node and the response returned to the client using some transport protocol. This page outlines the available transport protocols in Geth, providing the information users require to choose a transport protocol for a specific user scenario.

## Introduction {#introduction}

JSON-RPC is provided on multiple transports. Geth supports JSON-RPC over HTTP, WebSocket and Unix Domain Sockets. Transports must be enabled through
command-line flags.

Ethereum JSON-RPC APIs use a name-space system. RPC methods are grouped into several categories depending on their purpose. All method names are composed of
the namespace, an underscore, and the actual method name within the namespace. For example, the `eth_call` method resides in the `eth` namespace.

Access to RPC methods can be enabled on a per-namespace basis. Find documentation for individual namespaces in the sidebar.

## Transports {#transports}

There are three transport protocols available in Geth: IPC, HTTP and Websockets.

### HTTP Server {#http-server}

[HTTP](https://developer.mozilla.org/en-US/docs/Web/HTTP) is a unidirectional transport protocol that connects a client and server. The client sends a request to the server, and the server returns a response back to the client. An HTTP connection is closed after the response for a given request is sent.

HTTP is supported in every browser as well as almost all programming toolchains. Due to its ubiquity it has become the most widely used transport for interacting with Geth. To start a HTTP server in Geth, include the `--http` flag:

```sh
geth --http
```

If no other commands are provided, Geth falls back to its default behaviour of accepting connections from the local loopback interface (127.0.0.1). The default listening port is 8545. The ip address and listening port can be customized using the `--http.addr` and `--http.port` flags:

```sh
geth --http --http.port 3334
```

Not all of the JSON-RPC method namespaces are enabled for HTTP requests by default. Instead, they have to be whitelisted explicitly when Geth is started. Calling non-whitelisted RPC namespaces returns an RPC error with code `-32602`.

The default whitelist allows access to the `eth`, `net` and `web3` namespaces. To enable access to other APIs like debugging (`debug`), they must be configured using the `--http.api` flag. Enabling these APIs over HTTP is **not recommended** because access to these methods increases the attack surface.

```sh
geth --http --http.api eth,net,web3
```

Since the HTTP server is reachable from any local application, additional protection is built into the server to prevent misuse of the API from web pages. To enable access to the API from a web page (for example to use the online IDE, [Remix](https://remix.ethereum.org)), the server needs to be configured to accept Cross-Origin requests. This is achieved using the `--http.corsdomain` flag.

```sh
geth --http --http.corsdomain https://remix.ethereum.org
```

The `--http.corsdomain` command also acceptsd wildcards that enable access to the RPC from any origin:

```sh
--http.corsdomain '*'
```

### WebSocket Server {#websockets-server}

Websocket is a bidirectional transport protocol. A Websocket connection is maintained by client and server until it is explicitly terminated by one. Most modern browsers support Websocket which means it has good tooling.

Because Websocket is bidirectional, servers can push events to clients. That makes Websocket a good choice for use-cases involving [event subscription](/docs/interacting-with-geth/rpc/pubsub). Another benefit of Websocket is that after the handshake procedure, the overhead of individual messages is low,
making it good for sending high number of requests.

Configuration of the WebSocket endpoint in Geth follows the same pattern as the HTTP transport. WebSocket access can be enabled using the `--ws` flag. If no additional information is provided, Geth falls back to its default behaviour which is to establish the Websocket on port 8546. The `--ws.addr`, `--ws.port` and `--ws.api` flags can be used to customize settings for the WebSocket server. For example, to start Geth with a Websocket connection for RPC using
the custom port 3334 and whitelisting the `eth`, `net` and `web3` namespaces:

```sh
geth --ws --ws.port 3334 --ws.api eth,net,web3
```

Cross-Origin request protection also applies to the WebSocket server. The `--ws.origins` flag can be used to allow access to the server from web pages:

```sh
geth --ws --ws.origins http://myapp.example.com
```

As with `--http.corsdomain`, using the wildcard `--ws.origins '*'` allows access from any origin.

<Note>By default, **account unlocking is forbidden when HTTP or Websocket access is enabled** (i.e. by passing `--http` or `ws` flag). This is because an attacker that manages to access the node via the externally-exposed HTTP/WS port can then control the unlocked account. It is possible to force account unlock by including the `--allow-insecure-unlock` flag but this is unsafe and **not recommended** except for expert users that completely understand how it can be used safely. This is not a hypothetical risk: **there are bots that continually scan for http-enabled Ethereum nodes to attack**</Note>

### IPC Server {#ipc-server}

IPC is normally available for use in local environments where the node and the console exist on the same machine. Geth creates a pipe in the computers local file system (at `ipcpath`) that configures a connection between node and console. The `geth.ipc` file can also be used by other processes on the same machine to interact with Geth.

On UNIX-based systems (Linux, OSX) the IPC is a UNIX domain socket. On Windows IPC is provided using named pipes. The IPC server is enabled by default and has access to all JSON-RPC namespaces.

The listening socket is placed into the data directory by default. On Linux and macOS, the default location of the geth socket is

```sh
~/.ethereum/geth.ipc
```

On Windows, IPC is provided via named pipes. The default location of the geth pipe is:

```sh
\\.\pipe\geth.ipc
```

The location of the socket can be customized using the `--ipcpath` flag. IPC can be disabled
using the `--ipcdisable` flag.

## Choosing a transport protocol {#choosing-transport-protocol}

The following table summarizes the relative strengths and weaknesses of each transport protocol so that users can make informed decisions about which to use.

|                               | HTTP  |  WS   |  IPC  |
| :---------------------------: | :---: | :---: | :---: |
|      Event subscription       |   N   | **Y** | **Y** |
|       Remote connection       | **Y** | **Y** |   N   |
| Per-message metadata overhead | high  |  low  |  low  |

As a general rule IPC is most secure because it is limited to interactions on the local machine and cannot be exposed to external traffic. It can also be used
to subscribe to events. HTTP is a familiar and idempotent transport that closes connections between requests and can therefore have lower overall overheads if the number of requests is fairly low. Websockets provides a continuous open channel that can enable event subscriptions and streaming and handle large volumes of requests with smaller per-message overheads.

## Engine-API {#engine-api}

The Engine-API is a set of RPC methods that enable communication between Geth and the [consensus client](/docs/getting-started/consensus-clients). These are not designed to be exposed to the user - instead they are called automatically by the clients when they need to exchange information. The Engine API is enabled by default - the user is not required to pass any instruction to Geth to enable these methods.

Read more in the [Engine API spec](https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md).

## Summary {#summary}

RPC requests to a Geth node can be made using three different transport protocols. The protocols are enabled at startup using their respective flags. The right choice of transport protocol depends on the specific use case.
