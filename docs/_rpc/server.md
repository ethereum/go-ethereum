---
title: JSON-RPC Server
sort_key: A
---


Geth supports all standard web3 JSON-RPC APIs. You can find documentation for these APIs on the [Ethereum Wiki JSON-RPC page][web3-rpc] page.
JSON-RPC is provided on multiple transports. For example, Geth supports JSON-RPC over HTTP, WebSocket, and Unix Domain Sockets (IPC). Transports must be enabled through command-line flags.

The Ethereum JSON-RPC APIs use the namespace system. RPC methods are grouped into several categories depending on their purpose.
All method names are composed of the namespace, an underscore, and the actual method name within the namespace. For example, 
the `eth_call` method resides in the `eth` namespace.

To access the RPC methods, you enable it per-namespace basis. Find documentation for individual namespaces in the sidebar.

### HTTP Server

To enable the HTTP server, use the `--http` flag.

    geth --http


By default, HTTP-RPC server listens to port (8545). However, 
you can change the port to which geth listens by adding
this flag `--http.port`.

    geth --http --http.port 8080 


By default, geth accepts connections from the loopback interface (127.0.0.1). 
However, you can customize the address to which geth listens 
by adding this flag `--http.addr`.
  
    geth --http --http.addr http://127.0.0.1


JSON-RPC method namespaces must be whitelisted to be available through the HTTP server. 
An RPC error with error code -32602 is generated if you call a namespace that isn’t whitelisted. 
The default whitelist allows access to the “eth” and “shh” namespaces.
To enable access to other APIs like account management (“personal”) and debugging (“debug”), 
they must be configured via the `--http.api` flag. However, 
we do not recommend enabling such APIs over HTTP since access to these methods increases the attack surface.

    geth --http --http.api personal,eth,net,web3


Since the HTTP server is reachable from any local application, additional 
protection is built into the server to prevent misuse of the API from web pages. 
If you want to enable access to the API from a web page, 
you must configure the server to accept Cross-Origin requests 
with the `--http.corsdomain` flag. Also, Also, you can add more than one domain
to accept cross-origin requests by separating them with a comma (,).

> you can use `--http.corsdomain '*'` to enable access from any origin, but it is highly insecure and not recommended.
 
 Example: if you want to use [Remix][remix] with geth, it will allow requests from the
remix domain.

    geth --http --http.corsdomain https://remix.ethereum.org


To enable access to the virtual hostnames from a server, 
you will configure geth http server to accept requests from 
that hostname with the `--http.vhosts` flag. Also, you can add 
more than one hostname by separating them with a comma (,).

    geth --http --http.vhosts http://localhost:9002/

### WebSocket Server

The configuration of the WebSocket endpoint is similar to the HTTP transport. 
This `--ws` flag enables the web socket RPC API Server in geth 
and listens to `ws://127.0.0.1:8546` as the default IP and port.

    geth --ws


Geth web socket listens to localhost by default (127.0.0.1).
However, by modifying the value with this flag `--ws.addr`, 
you can alter the address that geth listens to via the 
geth websocket. 

    geth --ws --ws.addr ws://190.0.0.1


If you want to enable access to the server from a web page, 
you must configure the server to accept Cross-Origin requests 
with the `--ws.origins` flag. Also, you can add more than 
one domain to accept cross-origin requests by separating them with a comma (,).

> As with `--http.corsdomain`, using `--ws.origins '*'` allows access from any origin.


    geth --ws --ws.origins chrome-extension://fgponpodhbmadfljofbimhhlengambbn
  

Geth web socket listens to port 8546 by default.
However, by setting the value of the flag `--ws.port`, 
you can alter the address that geth listens to via the WS-RPC server . 
    
     geth --ws --ws.port 8080
   

The default whitelist allows access to the “eth” and “shh” namespaces. 
To enable access to other APIs like account management (“personal”) 
and debugging (“debug”), they must be configured via the `--ws.api` flag.
However, we do not recommend enabling such APIs over WebSocket since 
access to these methods increases the attack surface.

    geth --ws --ws.port 3334 --ws.api eth,net,web3

### RPC Server

  
To limit gas used via the RPC APIs in eth_call, 
you will use this  `--rpc.gascap` flag to set global 
gas cap. The default amount of gas capped is 50,000,000.

      geth --http --rpc.gascap 100

  
To limit the transaction fee (in ether)
sent via the RPC APIs, you will use this `--rpc.txfeecap` flag  

      geth --http --rpc.txfeecap 100

### IPC Server

JSON-RPC APIs are also provided on a UNIX domain socket. This server is enabled
by default and has access to all JSON-RPC namespaces.

The listening socket is placed into the data directory by default. On **Linux** and **macOS**,
the default location of the geth socket is

    ~/.ethereum/geth.ipc

On **Windows**, IPC is provided via named pipes. The default location of the geth pipe is:

    \\.\pipe\geth.ipc
    

You can configure the location of the socket using the `--ipcpath` flag.

    geth --ipcpath

You can disable the IPC interface with this flag. This will close the IPC endpoint, thereby refusing you access to start the JavaScript console.

    geth --ipcdisable

[web3-rpc]: https://github.com/ethereum/execution-apis
[remix]: https://remix.ethereum.org
