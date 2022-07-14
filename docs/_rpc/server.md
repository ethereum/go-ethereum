---
title: JSON-RPC Server
sort_key: A
---

Interacting with Geth requires sending requests to specific JSON-RPC API 
methods. Geth supports all standard Web3 [JSON-RPC API][web3-rpc] endpoints. 
The RPC requests must be sent to the node and the response returned to the client 
using some transport protocol. This page outlines the available transport protocols 
in Geth, providing the information users require to choose a transport protocol for 
a specific user scenario.

{:toc}
-   this will be removed by the toc

## Introduction

JSON-RPC is provided on multiple transports. Geth supports JSON-RPC over HTTP,
WebSocket and Unix Domain Sockets. Transports must be enabled through
command-line flags.

Ethereum JSON-RPC APIs use a name-space system. RPC methods are grouped into
several categories depending on their purpose. All method names are composed of
the namespace, an underscore, and the actual method name within the namespace.
For example, the `eth_call` method resides in the `eth` namespace.

Access to RPC methods can be enabled on a per-namespace basis. Find
documentation for individual namespaces in the sidebar.

## RPC

[RPC (Remote Procedure Calls)][rpc] are communication mechanisms that enable programs to execute a procedure in an address
space other than their own. Address spaces can be separate processes on a single
machine or different machines. RPC is a client-server interaction where a client sends 
a request that includes a payload that invokes some method in the server and returns
the result in a response to the client. The RPC has to be implemented using some transport 
system - for Geth this transport system can be IPC, HTTP or Websockets. HTTP and Websockets are individually
enabled using command line flags when Geth is started (IPC is enabled by default).

Regardless of the transport protocol, the RPC address determines which ports are exposed for incoming
RPC requests to Geth. The default for Geth is to expose `localhost:8545` (equivalent to `127.0.0.1:8545`) 
for RPC communication for HTTP and 8546 for Websocket. To restrict RPC access to processes running on the 
local machine, a firewall needs to be configured to block port 8545/8546 (or whatever custom port has been defined). 
Leaving the RPC port open allows access to the Geth RPC to any computers on the internet. Clearly, this is 
a **massive security risk**, so it is recommended to configure Geth with the RPC port blocked to all non-local 
traffic in most scenarios, and where remote access is required a firewall must be configured so that it blocks 
inbound requests from all ip addresses apart from ones that are explicitly whitelisted.

## Transport protocols

There are three transport protocols available in Geth: IPC, HTTP and Websockets. 

### HTTP Server

[HTTP](https://developer.mozilla.org/en-US/docs/Web/HTTP) is a unidirectional transport protocol
that connects a client and server. The client sends a request to the server, and the server
returns a response back to the client. Each response is associated with a specific request
and after a response is sent the connection between client and server is closed. Each new
HTTP request creates a new connection between the client and server that lasts until the response
is sent. HTTP requests are composed of the protocol version, method, headers, host information
and the request data being sent to the server. 

HTTP is very widely used and understood and this familiarity makes it a good default RPC transport
protocol. It also has some specific advantages over Websockets in some scenarios, such as when a
client wishes to retrieve the current state of some resource that does not continually update.
Historical values such as finalized states of the Ethereum blockchain, or infrequently
updating values such as most account balances, are ideally served over HTTP. The idempotency and
safety properties of HTTP are well suited to scenarios that must be resilient to message delivery
failures. Interactions with Geth fall into this category, and are therefore well served by HTTP.

To start Geth using HTTP as the transport protocol for the RPC connection, include the `--http` flag:

```sh
geth --http
```

If no other commands are provided, Geth falls back to its default behaviour of accepting connections 
from the local loopback interface (127.0.0.1). The default listening port is 8545. The ip address and 
listening port can be customized using the `--http.addr` and `--http.port` flags:

```sh
geth --http --http.port 3334
```

Not all of the JSON-RPC method namespaces are enabled for HTTP requests by default. 
Instead, they have to be whitelisted explicitly when Geth is started. Calling non-whitelisted 
RPC namespaces returns an RPC error with code `-32602`. 

The default whitelist allows access to the `eth`, `net` and `web3` namespaces. To enable access 
to other APIs like account management (`personal`) and debugging (`debug`), they must be configured 
using the `--http.api` flag. Enabling these APIs over HTTP is **not recommended** because access 
to these methods increases the attack surface.

```sh
geth --http --http.api personal,eth,net,web3
```

Since the HTTP server is reachable from any local application, additional protection is built into 
the server to prevent misuse of the API from web pages. To enable access to the API from a web page
(for example to use the online IDE, [Remix](https://remix.ethereum.org)), the server needs to be
configured to accept Cross-Origin requests. This is achieved using the `--http.corsdomain` flag.

```sh
geth --http --http.corsdomain https://remix.ethereum.org
```

The `--http.corsdomain` command also acceptsd wildcards that enable access to the RPC from any
origin: 

```sh
--http.corsdomain '*'
```

### WebSocket Server

Websocket is a bidirectional transport protocol. Unlike HTTP, Websocket is *stateful*, meaning
it maintains a connection between client and server until it is explicitly terminated by one
or other party. The persistent connection between the client and server is known as a 'Websocket'
that enables message exchange. The connection is established using a 'handshake' procedure that 
configures both parties.

This continuous messaging format works well for real-time applications, where data is continuously 
streamed from the server and ingested by the client. For Geth specifically, Websocket is required
to subscribe to events, since events are pushed from server to client over an open Websocket. 
This could not work over HTTP because actions can only be initiated by request. Websocket
is also probably preferential wherever large numbers of requetss are expected, since the overhead 
per message is lower for Websocket compared to HTTP.

Configuration of the WebSocket endpoint in Geth follows the same pattern as the HTTP transport. 
WebSocket access can be enabled using the `--ws` flag. If no additional information is provided, 
Geth falls back to its default behaviour which is to establish the Websocket on port 8546.
The `--ws.addr`, `--ws.port` and `--ws.api` flags can be used to customize settings
for the WebSocket server. For example, to start Geth with a Websocket connection for RPC using
the custom port 3334 and whitelisting the `eth`, `net` and `web3` namespaces:

```sh
geth --ws --ws.port 3334 --ws.api eth,net,web3
```

Cross-Origin request protection also applies to the WebSocket server. The
`--ws.origins` flag can be used to allow access to the server from web pages:

```sh
geth --ws --ws.origins http://myapp.example.com
```

As with `--http.corsdomain`, using the wildcard `--ws.origins '*'` allows access from any origin.

{% include note.html content=" By default, **account unlocking is forbidden when HTTP or 
Websocket access is enabled** (i.e. by passing `--http` or `ws` flag). This is because an 
attacker that manages to access the node via the externally-exposed HTTP/WS port can then 
control the unlocked account. It is possible to force account unlock by including the 
`--allow-insecure-unlock` flag but this is unsafe and **not recommended** except for expert 
users that completely understand how it can be used safely. 
This is not a hypothetical risk: **there are bots that continually scan for http-enabled 
Ethereum nodes to attack**" %}


### IPC Server

IPC is normally available for use in local environments where the node and the console
exist on the same machine. Geth creates a pipe in the computers local file system 
(at `ipcpath`) that configures a connection between node and console. The `geth.ipc` file can
also be used by other processes on the same machine to interact with Geth.

On UNIX-based systems (Linux, OSX) the IPC is a UNIX domain socket. On Windows IPC is
provided using named pipes. The IPC server is enabled by default and has access to all 
JSON-RPC namespaces.

The listening socket is placed into the data directory by default. On Linux and macOS,
the default location of the geth socket is

```sh
~/.ethereum/geth.ipc
```

On Windows, IPC is provided via named pipes. The default location of the geth pipe is:

```sh
\\.\pipe\geth.ipc
```

The location of the socket can be customized using the `--ipcpath` flag. IPC can be disabled 
using the `--ipcdisable` flag.

## Choosing a transport protocol

The following table summarizes the relative strengths and weaknesses of each transport
protocol so that users can make informed decisions about which to use.

|                                     |     HTTP    |     WS   |   IPC   |
| :----------------------------------:|:-----------:|:--------:|:-------:|
| Unidirectional                      |    **Y**    |     N    |     N   |
| Bidirectional                       |      N      |   **Y**  |   **Y** |
| Real-time data                      |      N      |   **Y**  |   **Y** |
| event subscription                  |      N      |   **Y**  |   **Y** |
| RESTful                             |    **Y**    |     N    |     N   |
| Closes connection between requests  |    **Y**    |     N    |     N   |
| Maintains continuous connections    |      N      |   **Y**  |   **Y** |

As a general rule IPC is most secure because it is limited to interactions on the 
local machine and cannot be exposed to external traffic. It can also be used
to subscribe to events. HTTP is a familiar and idempotent transport that closes 
connections between requests and can therefore have lower overall overheads if the number 
of requests is fairly low. Websockets provides a continuous open channel that can enable
event subscriptions and streaming and handle large volumes of requests with smaller per-message
overheads.


## Summary

RPC requests to a Geth node can be made using three different transport protocols. The 
protocols are enabled at startup using their respective flags. The right choice of transport
protocol depends on the specific use case.


[web3-rpc]: https://github.com/ethereum/execution-apis
[remix]: https://remix.ethereum.org
[rpc]: https://www.ibm.com/docs/en/aix/7.1?topic=concepts-remote-procedure-call
