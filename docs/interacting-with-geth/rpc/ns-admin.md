---
title: admin Namespace
description: Documentation for the JSON-RPC API "admin" namespace
---

The `admin` API gives access to several non-standard RPC methods, which allows fine grained control over a Geth instance, including but not limited to network peer and RPC endpoint management.

## admin_addPeer {#admin-addpeer}

The `addPeer` administrative method requests adding a new remote node to the list of tracked static nodes. The node will try to maintain connectivity to these nodes at all times, reconnecting every once in a while if the remote connection goes down.

The method accepts a single argument, the [`enode`](https://ethereum.org/en/developers/docs/networking-layer/network-addresses/#enode) URL of the remote peer to start tracking and returns a `BOOL` indicating whether the peer was accepted for tracking or some error occurred.

| Client  | Method invocation                              |
| :------ | ---------------------------------------------- |
| Go      | `admin.AddPeer(url string) (bool, error)`      |
| Console | `admin.addPeer(url)`                           |
| RPC     | `{"method": "admin_addPeer", "params": [url]}` |

**Example:**

```js
> admin.addPeer("enode://a979fb575495b8d6db44f750317d0f4622bf4c2aa3365d6af7c284339968eef29b69ad0dce72a4d8db5ebb4968de0e3bec910127f134779fbcb0cb6d3331163c@52.16.188.185:30303")
true
```

## admin_addTrustedPeer {#admin-addtrustedpeer}

Adds the given node to a reserved trusted list which allows the node to always connect, even if the slots are full. It returns a `BOOL` to indicate whether the peer was successfully added to the list.

| Client  | Method invocation                                     |
| :------ | ----------------------------------------------------- |
| Console | `admin.addTrustedPeer(url)`                           |
| RPC     | `{"method": "admin_addTrustedPeer", "params": [url]}` |

## admin_datadir {#admin-datadir}

The `datadir` administrative property can be queried for the absolute path the running Geth node currently uses to store all its databases.

| Client  | Method invocation                 |
| :------ | --------------------------------- |
| Go      | `admin.Datadir() (string, error`) |
| Console | `admin.datadir`                   |
| RPC     | `{"method": "admin_datadir"}`     |

**Example:**

```js
> admin.datadir
"/home/john/.ethereum"
```

## admin_exportChain {#admin-exportchain}

Exports the current blockchain into a local file. It optionally takes a first and last block number, in which case it exports only that range of blocks. It returns a boolean indicating whether the operation succeeded.

| Client  | Method invocation                                                     |
| :------ | --------------------------------------------------------------------- |
| Console | `admin.exportChain(file, first, last)`                                |
| RPC     | `{"method": "admin_exportChain", "params": [string, uint64, uint64]}` |

## admin_importChain {#admin-importchain}

Imports an exported list of blocks from a local file. Importing involves processing the blocks and inserting them into the canonical chain. The state from the parent block of this range is required. It returns a boolean indicating whether the operation succeeded.

| Client  | Method invocation                                     |
| :------ | ----------------------------------------------------- |
| Console | `admin.importChain(file)`                             |
| RPC     | `{"method": "admin_importChain", "params": [string]}` |

## admin_nodeInfo {#admin-nodeinfo}

The `nodeInfo` administrative property can be queried for all the information known about the running Geth node at the networking granularity. These include general information about the node itself as a participant of the [ÐΞVp2p](https://github.com/ethereum/devp2p/blob/master/caps/eth.md) P2P overlay protocol, as well as specialized information added by each of the running application protocols (e.g. `eth`, `les`, `shh`, `bzz`).

| Client  | Method invocation                         |
| :------ | ----------------------------------------- |
| Go      | `admin.NodeInfo() (*p2p.NodeInfo, error`) |
| Console | `admin.nodeInfo`                          |
| RPC     | `{"method": "admin_nodeInfo"}`            |

**Example:**

```js
> admin.nodeInfo
{
  enode: "enode://3d876252880e32116fdb52ea56a78ee2b9789e55b4413de910db69702ce93a7ff9a0b7c647a010a5e1e079c0aca146331083009644e12dc03510b9de9f50b9ef@156.146.56.131:30303?discport=39261",
  enr: "enr:-K64QL0-CI9BofDkirpulbV1OOOgqf5HLRMHr9iaziZInqI9HmeGOGZv2hs6J7olLu32LUMeYHTCjNBu3De_zlkI1fSGAY_h5ivyg2V0aMrJhPxk7ASDEYwwgmlkgnY0gmlwhJySOIOJc2VjcDI1NmsxoQM9h2JSiA4yEW_bUupWp47iuXieVbRBPekQ22lwLOk6f4RzbmFwwIN0Y3CCdl-DdWRwgpldhHVkcDaCdl8",
  id: "b7b61ea54ad081258a13a6d82920ce6719301f4670c458f64f0035e3463ec2df",
  ip: "156.146.56.131",
  listenAddr: "[::]:30303",
  name: "Geth/v1.14.4-unstable-51327686/linux-arm64/go1.22.3",
  ports: {
    discovery: 39261,
    listener: 30303
  },
  protocols: {
    eth: {
      config: {
        arrowGlacierBlock: 13773000,
        berlinBlock: 12244000,
        byzantiumBlock: 4370000,
        cancunTime: 1710338135,
        chainId: 1,
        constantinopleBlock: 7280000,
        daoForkBlock: 1920000,
        daoForkSupport: true,
        eip150Block: 2463000,
        eip155Block: 2675000,
        eip158Block: 2675000,
        ethash: {},
        grayGlacierBlock: 15050000,
        homesteadBlock: 1150000,
        istanbulBlock: 9069000,
        londonBlock: 12965000,
        muirGlacierBlock: 9200000,
        petersburgBlock: 7280000,
        shanghaiTime: 1681338455,
        terminalTotalDifficulty: 5.875e+22,
        terminalTotalDifficultyPassed: true
      },
      difficulty: 17179869184,
      genesis: "0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3",
      head: "0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3",
      network: 1
    },
    snap: {}
  }
}
```

## admin_peerEvents {#admin-peerevents}

PeerEvents creates an [RPC subscription](/docs/interacting-with-geth/rpc/pubsub) which receives peer events from the node's p2p server. The type of events emitted by the server are as follows:

- `add`: emitted when a peer is added
- `drop`: emitted when a peer is dropped
- `msgsend`: emitted when a message is successfully sent to a peer
- `msgrecv`: emitted when a message is received from a peer

## admin_peers {#admin-peers}

The `peers` administrative property can be queried for all the information known about the connected remote nodes at the networking granularity. These include general information about the nodes themselves as participants of the [ÐΞVp2p](https://github.com/ethereum/devp2p/blob/master/caps/eth.md) P2P overlay protocol, as well as specialized information added by each of the running application protocols (e.g. `eth`, `les`, `shh`, `bzz`).

| Client  | Method invocation                        |
| :------ | ---------------------------------------- |
| Go      | `admin.Peers() ([]*p2p.PeerInfo, error`) |
| Console | `admin.peers`                            |
| RPC     | `{"method": "admin_peers"}`              |

**Example:**

```js
> admin.peers
[{
    caps: ["eth/68", "snap/1"],
    enode: "enode://4aeb4ab6c14b23e2c4cfdce879c04b0748a20d8e9b59e25ded2a08143e265c6c25936e74cbc8e641e3312ca288673d91f2f93f8e277de3cfa444ecdaaf982052@157.90.35.166:30303",
    id: "6b36f791352f15eb3ec4f67787074ab8ad9d487e37c4401d383f0561a0a20507",
    name: "Geth/v1.13.14-stable-2bd6bd01/linux-amd64/go1.21.7",
    network: {
      inbound: false,
      localAddress: "172.17.0.2:33666",
      remoteAddress: "157.90.35.166:30303",
      static: false,
      trusted: false
    },
    protocols: {
      eth: {
        version: 68
      },
      snap: {
        version: 1
      }
    }
}, /* ... */ {
    caps: ["eth/66", "eth/67", "eth/68", "snap/1"],
    enode: "enode://404786d90feafd54abcbcb7a7c791b6197304e58f7f582715312372af7297f194baf4abb6ce1cc5c55050e8111194d500590d0e08fcd75ce575f8fdd2e090af0@34.241.148.206:30303",
    id: "8a2a75da0f099ee3d1dcfad4a4825c81b5ab1cb3c18207e7626abae87ce589b1",
    name: "Geth/v1.11.6-stable-ea9e62ca/linux-amd64/go1.20.3",
    network: {
      inbound: false,
      localAddress: "172.17.0.2:59938",
      remoteAddress: "34.241.148.206:30303",
      static: false,
      trusted: false
    },
    protocols: {
      eth: {
        version: 68
      },
      snap: {
        version: 1
      }
    }
}]
```

## admin_removePeer {#admin-removepeer}

Disconnects from a remote node if the connection exists. It returns a boolean indicating validations succeeded. Note a `true` value doesn't necessarily mean that there was a connection which was disconnected.

| Client  | Method invocation                                    |
| :------ | ---------------------------------------------------- |
| Console | `admin.removePeer(url)`                              |
| RPC     | `{"method": "admin_removePeer", "params": [string]}` |

## admin_removeTrustedPeer {#admin-removetrustedpeer}

Removes a remote node from the trusted peer set, but it does not disconnect it automatically. It returns a boolean indicating validations succeeded.

| Client  | Method invocation                                           |
| :------ | ----------------------------------------------------------- |
| Console | `admin.removeTrustedPeer(url)`                              |
| RPC     | `{"method": "admin_removeTrustedPeer", "params": [string]}` |

## admin_startHTTP {#admin-starthttp}

The `startHTTP` administrative method starts an HTTP based JSON-RPC [API](/docs/interacting-with-geth/rpc) webserver to handle client requests. All the parameters are optional:

- `host`: network interface to open the listener socket on (defaults to `"localhost"`)
- `port`: network port to open the listener socket on (defaults to `8545`)
- `cors`: [cross-origin resource sharing](https://en.wikipedia.org/wiki/Cross-origin_resource_sharing) header to use (defaults to `""`)
- `apis`: API modules to offer over this interface (defaults to `"eth,net,web3"`)

The method returns a boolean flag specifying whether the HTTP RPC listener was opened or not. Please note, only one HTTP endpoint is allowed to be active at any time.

| Client  | Method invocation                                                                              |
| :------ | ---------------------------------------------------------------------------------------------- |
| Go      | `admin.StartHTTP(host *string, port *rpc.HexNumber, cors *string, apis *string) (bool, error)` |
| Console | `admin.startHTTP(host, port, cors, apis)`                                                      |
| RPC     | `{"method": "admin_startHTTP", "params": [host, port, cors, apis]}`                            |

**Example:**

```js
> admin.startHTTP("127.0.0.1", 8545)
true
```

## admin_startWS {#admin-startws}

The `startWS` administrative method starts an WebSocket based [JSON RPC](https://www.jsonrpc.org/specification) API webserver to handle client requests. All the parameters are optional:

- `host`: network interface to open the listener socket on (defaults to `"localhost"`)
- `port`: network port to open the listener socket on (defaults to `8546`)
- `cors`: [cross-origin resource sharing](https://en.wikipedia.org/wiki/Cross-origin_resource_sharing) header to use (defaults to `""`)
- `apis`: API modules to offer over this interface (defaults to `"eth,net,web3"`)

The method returns a boolean flag specifying whether the WebSocket RPC listener was opened or not. Please note, only one WebSocket endpoint is allowed to be active at any time.

| Client  | Method invocation                                                                            |
| :------ | -------------------------------------------------------------------------------------------- |
| Go      | `admin.StartWS(host *string, port *rpc.HexNumber, cors *string, apis *string) (bool, error)` |
| Console | `admin.startWS(host, port, cors, apis)`                                                      |
| RPC     | `{"method": "admin_startWS", "params": [host, port, cors, apis]}`                            |

**Example:**

```js
> admin.startWS("127.0.0.1", 8546)
true
```

## admin_stopHTTP {#admin-stophttp}

The `stopHTTP` administrative method closes the currently open HTTP RPC endpoint. As the node can only have a single HTTP endpoint running, this method takes no parameters, returning a boolean whether the endpoint was closed or not.

| Client  | Method invocation                |
| :------ | -------------------------------- |
| Go      | `admin.StopHTTP() (bool, error`) |
| Console | `admin.stopHTTP()`               |
| RPC     | `{"method": "admin_stopHTTP"`    |

**Example:**

```js
> admin.stopHTTP()
true
```

## admin_stopWS {#admin-stopws}

The `stopWS` administrative method closes the currently open WebSocket RPC endpoint. As the node can only have a single WebSocket endpoint running, this method takes no parameters, returning a boolean whether the endpoint was closed or not.

| Client  | Method invocation              |
| :------ | ------------------------------ |
| Go      | `admin.StopWS() (bool, error`) |
| Console | `admin.stopWS()`               |
| RPC     | `{"method": "admin_stopWS"`    |

**Example:**

```js
> admin.stopWS()
true
```
