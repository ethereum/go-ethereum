Beside the official [DApp APIs](https://github.com/ethereum/wiki/wiki/JSON-RPC) interface go-ethereum
has support for additional management APIs. Similar to the DApp APIs, these are also provided using
[JSON-RPC](http://www.jsonrpc.org/specification) and follow exactly the same conventions. Geth comes
with a console client which has support for all additional APIs described here.

## Enabling the management APIs

To offer these APIs over the Geth RPC endpoints, please specify them with the `--${interface}api`
command line argument (where `${interface}` can be `rpc` for the HTTP endpoint, `ws` for the WebSocket
endpoint and `ipc` for the unix socket (Unix) or named pipe (Windows) endpoint).

For example: `geth --ipcapi admin,eth,miner --rpcapi eth,web3 --rpc`

* Enables the admin, official DApp and miner API over the IPC interface
* Enables the official DApp and web3 API over the HTTP interface

The HTTP RPC interface must be explicitly enabled using the `--rpc` flag.

Please note, offering an API over the HTTP (`rpc`) or WebSocket (`ws`) interfaces will give everyone
access to the APIs who can access this interface (DApps, browser tabs, etc). Be careful which APIs
you enable. By default Geth enables all APIs over the IPC (`ipc`) interface and only the `db`, `eth`,
`net` and `web3` APIs over the HTTP and WebSocket interfaces.

To determine which APIs an interface provides, the `modules` JSON-RPC method can be invoked. For
example over an `ipc` interface on unix systems:

```
echo '{"jsonrpc":"2.0","method":"rpc_modules","params":[],"id":1}' | nc -U $datadir/geth.ipc
```

will give all enabled modules including the version number:

```
{  
   "id":1,
   "jsonrpc":"2.0",
   "result":{  
      "admin":"1.0",
      "db":"1.0",
      "debug":"1.0",
      "eth":"1.0",
      "miner":"1.0",
      "net":"1.0",
      "personal":"1.0",
      "shh":"1.0",
      "txpool":"1.0",
      "web3":"1.0"
   }
}
```

## Consuming the management APIs

These additional APIs follow the same conventions as the official DApp APIs. Web3 can be
[extended](https://github.com/ethereum/web3.js/pull/229) and used to consume these additional APIs. 

The different functions are split into multiple smaller logically grouped APIs. Examples are given
for the [JavaScript console](https://github.com/ethereum/go-ethereum/wiki/JavaScript-Console) but
can easily be converted to an RPC request.

**2 examples:**

* Console: `miner.start()`

* IPC: `echo '{"jsonrpc":"2.0","method":"miner_start","params":[],"id":1}' | nc -U $datadir/geth.ipc`

* HTTP: `curl -X POST --data '{"jsonrpc":"2.0","method":"miner_start","params":[],"id":74}' localhost:8545`

With the number of THREADS as an arguments:

* Console: `miner.start(4)`

* IPC: `echo '{"jsonrpc":"2.0","method":"miner_start","params":[4],"id":1}' | nc -U $datadir/geth.ipc`

* HTTP: `curl -X POST --data '{"jsonrpc":"2.0","method":"miner_start","params":[4],"id":74}' localhost:8545`

## List of management APIs

Beside the officially exposed DApp API namespaces (`eth`, `shh`, `web3`), Geth provides the following
extra management API namespaces:

* `admin`: Geth node management
* `debug`: Geth node debugging
* `miner`: Miner and [DAG](https://github.com/ethereum/wiki/wiki/Ethash-DAG) management
* `personal`: Account management
* `txpool`: Transaction pool inspection

| [admin](#admin)              | [debug](#debug)                                   | [miner](#miner)                     | [personal](#personal)                    | [txpool](#txpool)          |
| :--------------------------- | :-----------------------------------------------  | :---------------------------------- | :--------------------------------------- | :------------------------- |
| [addPeer](#admin_addpeer)    | [backtraceAt](#debug_backtraceAt)                 | [setExtra](#miner_setextra)         | [ecRecover](#personal_ecrecover)         | [content](#txpool_content) |
| [datadir](#datadir)          | [blockProfile](#debug_blockProfile)               | [setGasPrice](#miner_setgasprice)   | [importRawKey](#personal_importrawkey)   | [inspect](#txpool_inspect) |
| [nodeInfo](#admin_nodeinfo)  | [cpuProfile](#debug_cpuProfile)                   | [start](#miner_start)               | [listAccounts](#personal_listaccounts)   | [status](#txpool_status)   |
| [peers](#admin_peers)        | [dumpBlock](#debug_dumpblock)                     | [stop](#miner_stop)                 | [lockAccount](#personal_lockaccount)     |                            |
| [setSolc](#admin_setcolc)    | [gcStats](#debug_gcStats)                         | [getHashrate](#miner_gethashrate)   | [newAccount](#personal_newaccount)       |                            |
| [startRPC](#admin_startrpc)  | [getBlockRlp](#debug_getblockrlp)                 | [setEtherbase](#miner_setetherbase) | [unlockAccount](#personal_unlockaccount) |                            |
| [startWS](#admin_startws)    | [goTrace](#debug_goTrace)                         |                                     | [sendTransaction](#personal_sendtransaction) |                        |
| [stopRPC](#admin_stoprpc)    | [memStats](#debug_memStats)                       |                                     | [sign](#personal_sign)                   |                            |
| [stopWS](#admin_stopws)      | [seedHash](#debug_seedhash)[sign](#personal_sign)|                                      |                                          |                            |
|                              | [setBlockProfileRate](#debug_setBlockProfileRate) |                                     |                                          |                            |
|                              | [setHead](#debug_sethead)                         |                                     |                                          |                            |
|                              | [stacks](#debug_stacks)                           |                                     |                                          |                            |
|                              | [startCPUProfile](#debug_startCPUProfile)         |                                     |                                          |                            |
|                              | [startGoTrace](#debug_startGoTrace)               |                                     |                                          |                            |
|                              | [stopCPUProfile](#debug_stopCPUProfile)           |                                     |                                          |                            |
|                              | [stopGoTrace](#debug_stopGoTrace)                 |                                     |                                          |                            |
|                              | [traceBlock](#debug_traceblock)                   |                                     |                                          |                            |
|                              | [traceBlockByNumber](#debug_blockbynumber)        |                                     |                                          |                            |
|                              | [traceBlockByHash](#debug_blockbyhash)            |                                     |                                          |                            |
|                              | [traceBlockFromFile](#debug_traceblockfromfile)   |                                     |                                          |                            |
|                              | [traceTransaction](#debug_tracetransaction)       |                                     |                                          |                            |
|                              | [verbosity](#debug_verbosity)                     |                                     |                                          |                            |
|                              | [vmodule](#debug_vmodule)                         |                                     |                                          |                            |
|                              | [writeBlockProfile](#debug_writeBlockProfile)     |                                     |                                          |                            |
|                              | [writeMemProfile](#debug_writeMemProfile)         |                                     |                                          |                            |

## Admin

The `admin` API gives you access to several non-standard RPC methods, which will allow you to have
a fine grained control over your Geth instance, including but not limited to network peer and RPC
endpoint management.

### admin_addPeer

The `addPeer` administrative method requests adding a new remote node to the list of tracked static
nodes. The node will try to maintain connectivity to these nodes at all times, reconnecting every
once in a while if the remote connection goes down.

The method accepts a single argument, the [`enode`](https://github.com/ethereum/wiki/wiki/enode-url-format)
URL of the remote peer to start tracking and returns a `BOOL` indicating whether the peer was accepted
for tracking or some error occurred.

| Client  | Method invocation                              |
|:-------:|------------------------------------------------|
| Go      | `admin.AddPeer(url string) (bool, error)`      |
| Console | `admin.addPeer(url)`                           |
| RPC     | `{"method": "admin_addPeer", "params": [url]}` |

#### Example

```javascript
> admin.addPeer("enode://a979fb575495b8d6db44f750317d0f4622bf4c2aa3365d6af7c284339968eef29b69ad0dce72a4d8db5ebb4968de0e3bec910127f134779fbcb0cb6d3331163c@52.16.188.185:30303")
true
```

### admin_datadir

The `datadir` administrative property can be queried for the absolute path the running Geth node
currently uses to store all its databases.

| Client  | Method invocation                 |
|:-------:|-----------------------------------|
| Go      | `admin.Datadir() (string, error`) |
| Console | `admin.datadir`                   |
| RPC     | `{"method": "admin_datadir"}`     |

#### Example

```javascript
> admin.datadir
"/home/karalabe/.ethereum"
```

### admin_nodeInfo

The `nodeInfo` administrative property can be queried for all the information known about the running
Geth node at the networking granularity. These include general information about the node itself as a
participant of the [ÐΞVp2p](https://github.com/ethereum/wiki/wiki/%C3%90%CE%9EVp2p-Wire-Protocol) P2P
overlay protocol, as well as specialized information added by each of the running application protocols
(e.g. `eth`, `les`, `shh`, `bzz`).

| Client  | Method invocation                         |
|:-------:|-------------------------------------------|
| Go      | `admin.NodeInfo() (*p2p.NodeInfo, error`) |
| Console | `admin.nodeInfo`                          |
| RPC     | `{"method": "admin_nodeInfo"}`            |

#### Example

```javascript
> admin.nodeInfo
{
  enode: "enode://44826a5d6a55f88a18298bca4773fca5749cdc3a5c9f308aa7d810e9b31123f3e7c5fba0b1d70aac5308426f47df2a128a6747040a3815cc7dd7167d03be320d@[::]:30303",
  id: "44826a5d6a55f88a18298bca4773fca5749cdc3a5c9f308aa7d810e9b31123f3e7c5fba0b1d70aac5308426f47df2a128a6747040a3815cc7dd7167d03be320d",
  ip: "::",
  listenAddr: "[::]:30303",
  name: "Geth/v1.5.0-unstable/linux/go1.6",
  ports: {
    discovery: 30303,
    listener: 30303
  },
  protocols: {
    eth: {
      difficulty: 17334254859343145000,
      genesis: "0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3",
      head: "0xb83f73fbe6220c111136aefd27b160bf4a34085c65ba89f24246b3162257c36a",
      network: 1
    }
  }
}
```

### admin_peers

The `peers` administrative property can be queried for all the information known about the connected
remote nodes at the networking granularity. These include general information about the nodes themselves
as participants of the [ÐΞVp2p](https://github.com/ethereum/wiki/wiki/%C3%90%CE%9EVp2p-Wire-Protocol)
P2P overlay protocol, as well as specialized information added by each of the running application
protocols (e.g. `eth`, `les`, `shh`, `bzz`).

| Client  | Method invocation                        |
|:-------:|------------------------------------------|
| Go      | `admin.Peers() ([]*p2p.PeerInfo, error`) |
| Console | `admin.peers`                            |
| RPC     | `{"method": "admin_peers"}`              |

#### Example

```javascript
> admin.peers
[{
    caps: ["eth/61", "eth/62", "eth/63"],
    id: "08a6b39263470c78d3e4f58e3c997cd2e7af623afce64656cfc56480babcea7a9138f3d09d7b9879344c2d2e457679e3655d4b56eaff5fd4fd7f147bdb045124",
    name: "Geth/v1.5.0-unstable/linux/go1.5.1",
    network: {
      localAddress: "192.168.0.104:51068",
      remoteAddress: "71.62.31.72:30303"
    },
    protocols: {
      eth: {
        difficulty: 17334052235346465000,
        head: "5794b768dae6c6ee5366e6ca7662bdff2882576e09609bf778633e470e0e7852",
        version: 63
      }
    }
}, /* ... */ {
    caps: ["eth/61", "eth/62", "eth/63"],
    id: "fcad9f6d3faf89a0908a11ddae9d4be3a1039108263b06c96171eb3b0f3ba85a7095a03bb65198c35a04829032d198759edfca9b63a8b69dc47a205d94fce7cc",
    name: "Geth/v1.3.5-506c9277/linux/go1.4.2",
    network: {
      localAddress: "192.168.0.104:55968",
      remoteAddress: "121.196.232.205:30303"
    },
    protocols: {
      eth: {
        difficulty: 17335165914080772000,
        head: "5794b768dae6c6ee5366e6ca7662bdff2882576e09609bf778633e470e0e7852",
        version: 63
      }
    }
}]
```

### admin_setSolc

The `setSolc` administrative method sets the Solidity compiler path to be used by the node when
invoking the `eth_compileSolidity` RPC method. The Solidity compiler path defaults to `/usr/bin/solc`
if not set, so you only need to change it for using a non-standard compiler location.

The method accepts an absolute path to the Solidity compiler to use (specifying a relative path
would depend on the current – to the user unknown – working directory of Geth), and returns the
version string reported by `solc --version`.

| Client  | Method invocation                               |
|:-------:|-------------------------------------------------|
| Go      | `admin.SetSolc(path string) (string, error`)    |
| Console | `admin.setSolc(path)`                           |
| RPC     | `{"method": "admin_setSolc", "params": [path]}` |

#### Example

```javascript
> admin.setSolc("/usr/bin/solc")
"solc, the solidity compiler commandline interface\nVersion: 0.3.2-0/Release-Linux/g++/Interpreter\n\npath: /usr/bin/solc"
```

### admin_startRPC

The `startRPC` administrative method starts an HTTP based [JSON RPC](http://www.jsonrpc.org/specification)
API webserver to handle client requests. All the parameters are optional:

* `host`: network interface to open the listener socket on (defaults to `"localhost"`)
* `port`: network port to open the listener socket on (defaults to `8545`)
* `cors`: [cross-origin resource sharing](https://en.wikipedia.org/wiki/Cross-origin_resource_sharing) header to use (defaults to `""`)
* `apis`: API modules to offer over this interface (defaults to `"eth,net,web3"`)

The method returns a boolean flag specifying whether the HTTP RPC listener was opened or not. Please note, only one HTTP endpoint is allowed to be active at any time.

| Client  | Method invocation                                                                             |
|:-------:|-----------------------------------------------------------------------------------------------|
| Go      | `admin.StartRPC(host *string, port *rpc.HexNumber, cors *string, apis *string) (bool, error)` |
| Console | `admin.startRPC(host, port, cors, apis)`                                                      |
| RPC     | `{"method": "admin_startRPC", "params": [host, port, cors, apis]}`                            |

#### Example

```javascript
> admin.startRPC("127.0.0.1", 8545)
true
```

### admin_startWS

The `startWS` administrative method starts an WebSocket based [JSON RPC](http://www.jsonrpc.org/specification)
API webserver to handle client requests. All the parameters are optional:

* `host`: network interface to open the listener socket on (defaults to `"localhost"`)
* `port`: network port to open the listener socket on (defaults to `8546`)
* `cors`: [cross-origin resource sharing](https://en.wikipedia.org/wiki/Cross-origin_resource_sharing) header to use (defaults to `""`)
* `apis`: API modules to offer over this interface (defaults to `"eth,net,web3"`)

The method returns a boolean flag specifying whether the WebSocket RPC listener was opened or not. Please note, only one WebSocket endpoint is allowed to be active at any time.

| Client  | Method invocation                                                                             |
|:-------:|-----------------------------------------------------------------------------------------------|
| Go      | `admin.StartWS(host *string, port *rpc.HexNumber, cors *string, apis *string) (bool, error)` |
| Console | `admin.startWS(host, port, cors, apis)`                                                      |
| RPC     | `{"method": "admin_startWS", "params": [host, port, cors, apis]}`                            |

#### Example

```javascript
> admin.startWS("127.0.0.1", 8546)
true
```

### admin_stopRPC

The `stopRPC` administrative method closes the currently open HTTP RPC endpoint. As the node can only have a single HTTP endpoint running, this method takes no parameters, returning a boolean whether the endpoint was closed or not.

| Client  | Method invocation               |
|:-------:|---------------------------------|
| Go      | `admin.StopRPC() (bool, error`) |
| Console | `admin.stopRPC()`               |
| RPC     | `{"method": "admin_stopRPC"`    |

#### Example

```javascript
> admin.stopRPC()
true
```

### admin_stopWS

The `stopWS` administrative method closes the currently open WebSocket RPC endpoint. As the node can only have a single WebSocket endpoint running, this method takes no parameters, returning a boolean whether the endpoint was closed or not.

| Client  | Method invocation              |
|:-------:|--------------------------------|
| Go      | `admin.StopWS() (bool, error`) |
| Console | `admin.stopWS()`               |
| RPC     | `{"method": "admin_stopWS"`    |

#### Example

```javascript
> admin.stopWS()
true
```

## Debug

The `debug` API gives you access to several non-standard RPC methods, which will allow you to inspect,
debug and set certain debugging flags during runtime.


### debug_backtraceAt

Sets the logging backtrace location. When a backtrace location
is set and a log message is emitted at that location, the stack
of the goroutine executing the log statement will be printed to stderr.

The location is specified as `<filename>:<line>`.

| Client  | Method invocation                                     |
|:-------:|-------------------------------------------------------|
| Console | `debug.backtraceAt(string)`                           |
| RPC     | `{"method": "debug_backtraceAt", "params": [string]}` |

Example:

``` javascript
> debug.backtraceAt("server.go:443")
```

### debug_blockProfile

Turns on block profiling for the given duration and writes
profile data to disk. It uses a profile rate of 1 for most
accurate information. If a different rate is desired, set
the rate and write the profile manually using
`debug_writeBlockProfile`.

| Client  | Method invocation                                              |
|:-------:|----------------------------------------------------------------|
| Console | `debug.blockProfile(file, seconds)`                            |
| RPC     | `{"method": "debug_blockProfile", "params": [string, number]}` |

### debug_cpuProfile

Turns on CPU profiling for the given duration and writes
profile data to disk.

| Client  | Method invocation                                            |
|:-------:|--------------------------------------------------------------|
| Console | `debug.cpuProfile(file, seconds)`                            |
| RPC     | `{"method": "debug_cpuProfile", "params": [string, number]}` |

### debug_dumpBlock

Retrieves the state that corresponds to the block number and returns a list of accounts (including
storage and code).

| Client  | Method invocation                                     |
|:-------:|-------------------------------------------------------|
| Go      | `debug.DumpBlock(number uint64) (state.World, error)` |
| Console | `debug.traceBlockByHash(number, [options])`           |
| RPC     | `{"method": "debug_dumpBlock", "params": [number]}`   |

#### Example

```javascript
> debug.dumpBlock(10)
{
    fff7ac99c8e4feb60c9750054bdc14ce1857f181: {
      balance: "49358640978154672",
      code: "",
      codeHash: "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
      nonce: 2,
      root: "56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
      storage: {}
    },
    fffbca3a38c3c5fcb3adbb8e63c04c3e629aafce: {
      balance: "3460945928",
      code: "",
      codeHash: "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
      nonce: 657,
      root: "56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
      storage: {}
    }
  },
  root: "19f4ed94e188dd9c7eb04226bd240fa6b449401a6c656d6d2816a87ccaf206f1"
}
```

### debug_gcStats

Returns GC statistics.

See https://golang.org/pkg/runtime/debug/#GCStats for information about
the fields of the returned object.

| Client  | Method invocation                                 |
|:-------:|---------------------------------------------------|
| Console | `debug.gcStats()`                                 |
| RPC     | `{"method": "debug_gcStats", "params": []}`       |

### debug_getBlockRlp

Retrieves and returns the RLP encoded block by number.

| Client  | Method invocation                                     |
|:-------:|-------------------------------------------------------|
| Go      | `debug.GetBlockRlp(number uint64) (string, error)`    |
| Console | `debug.getBlockRlp(number, [options])`                |
| RPC     | `{"method": "debug_getBlockRlp", "params": [number]}` |

References: [RLP](https://github.com/ethereum/wiki/wiki/RLP)

### debug_goTrace

Turns on Go runtime tracing for the given duration and writes
trace data to disk.

| Client  | Method invocation                                         |
|:-------:|-----------------------------------------------------------|
| Console | `debug.goTrace(file, seconds)`                            |
| RPC     | `{"method": "debug_goTrace", "params": [string, number]}` |

### debug_memStats

Returns detailed runtime memory statistics.

See https://golang.org/pkg/runtime/#MemStats for information about
the fields of the returned object.

| Client  | Method invocation                                 |
|:-------:|---------------------------------------------------|
| Console | `debug.memStats()`                                |
| RPC     | `{"method": "debug_memStats", "params": []}`      |

### debug_seedHash

Fetches and retrieves the seed hash of the block by number

| Client  | Method invocation                                  |
|:-------:|----------------------------------------------------|
| Go      | `debug.SeedHash(number uint64) (string, error)`    |
| Console | `debug.seedHash(number, [options])`                |
| RPC     | `{"method": "debug_seedHash", "params": [number]}` |

### debug_setHead

Sets the current head of the local chain by block number. **Note**, this is a
destructive action and may severely damage your chain. Use with *extreme* caution.

| Client  | Method invocation                                 |
|:-------:|---------------------------------------------------|
| Go      | `debug.SetHead(number uint64)`                    |
| Console | `debug.setHead(number)`                           |
| RPC     | `{"method": "debug_setHead", "params": [number]}` |

References:
[Ethash](https://github.com/ethereum/wiki/wiki/Mining#the-algorithm)

### debug_setBlockProfileRate

Sets the rate (in samples/sec) of goroutine block profile
data collection. A non-zero rate enables block profiling,
setting it to zero stops the profile. Collected profile data
can be written using `debug_writeBlockProfile`.

| Client  | Method invocation                                             |
|:-------:|---------------------------------------------------------------|
| Console | `debug.setBlockProfileRate(rate)`                             |
| RPC     | `{"method": "debug_setBlockProfileRate", "params": [number]}` |

### debug_stacks

Returns a printed representation of the stacks of all goroutines.
Note that the web3 wrapper for this method takes care of the printing
and does not return the string.

| Client  | Method invocation                                 |
|:-------:|---------------------------------------------------|
| Console | `debug.stacks()`                                  |
| RPC     | `{"method": "debug_stacks", "params": []}`        |

### debug_startCPUProfile

Turns on CPU profiling indefinitely, writing to the given file.

| Client  | Method invocation                                         |
|:-------:|-----------------------------------------------------------|
| Console | `debug.startCPUProfile(file)`                             |
| RPC     | `{"method": "debug_startCPUProfile", "params": [string]}` |

### debug_startGoTrace

Starts writing a Go runtime trace to the given file.

| Client  | Method invocation                                      |
|:-------:|--------------------------------------------------------|
| Console | `debug.startGoTrace(file)`                             |
| RPC     | `{"method": "debug_startGoTrace", "params": [string]}` |

### debug_stopCPUProfile

Stops an ongoing CPU profile.

| Client  | Method invocation                                  |
|:-------:|----------------------------------------------------|
| Console | `debug.stopCPUProfile()`                           |
| RPC     | `{"method": "debug_stopCPUProfile", "params": []}` |

### debug_stopGoTrace

Stops writing the Go runtime trace.

| Client  | Method invocation                                 |
|:-------:|---------------------------------------------------|
| Console | `debug.startGoTrace(file)`                        |
| RPC     | `{"method": "debug_stopGoTrace", "params": []}`   |

### debug_traceBlock

The `traceBlock` method will return a full stack trace of all invoked opcodes of all transaction
that were included included in this block. **Note**, the parent of this block must be present or
it will fail.

| Client  | Method invocation                                                        |
|:-------:|--------------------------------------------------------------------------|
| Go      | `debug.TraceBlock(blockRlp []byte, config. *vm.Config) BlockTraceResult` |
| Console | `debug.traceBlock(tblockRlp, [options])`                                 |
| RPC     | `{"method": "debug_traceBlock", "params": [blockRlp, {}]}`               |

References:
[RLP](https://github.com/ethereum/wiki/wiki/RLP)

#### Example

```javascript
> debug.traceBlock("0xblock_rlp")
{
  gas: 85301,
  returnValue: "",
  structLogs: [{
      depth: 1,
      error: "",
      gas: 162106,
      gasCost: 3,
      memory: null,
      op: "PUSH1",
      pc: 0,
      stack: [],
      storage: {}
  },
    /* snip */
  {
      depth: 1,
      error: "",
      gas: 100000,
      gasCost: 0,
      memory: ["0000000000000000000000000000000000000000000000000000000000000006", "0000000000000000000000000000000000000000000000000000000000000000", "0000000000000000000000000000000000000000000000000000000000000060"],
      op: "STOP",
      pc: 120,
      stack: ["00000000000000000000000000000000000000000000000000000000d67cbec9"],
      storage: {
        0000000000000000000000000000000000000000000000000000000000000004: "8241fa522772837f0d05511f20caa6da1d5a3209000000000000000400000001",
        0000000000000000000000000000000000000000000000000000000000000006: "0000000000000000000000000000000000000000000000000000000000000001",
        f652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f: "00000000000000000000000002e816afc1b5c0f39852131959d946eb3b07b5ad"
      }
  }]
```

### debug_traceBlockByNumber

Similar to [debug_traceBlock](#debug_traceBlock), `traceBlockByNumber` accepts a block number and will replay the
block that is already present in the database.

| Client  | Method invocation                                                              |
|:-------:|--------------------------------------------------------------------------------|
| Go      | `debug.TraceBlockByNumber(number uint64, config. *vm.Config) BlockTraceResult` |
| Console | `debug.traceBlockByNumber(number, [options])`                                  |
| RPC     | `{"method": "debug_traceBlockByNumber", "params": [number, {}]}`               |

References:
[RLP](https://github.com/ethereum/wiki/wiki/RLP)

### debug_traceBlockByHash

Similar to [debug_traceBlock](#debug_traceBlock), `traceBlockByHash` accepts a block hash and will replay the
block that is already present in the database.

| Client  | Method invocation                                                               |
|:-------:|---------------------------------------------------------------------------------|
| Go      | `debug.TraceBlockByHash(hash common.Hash, config. *vm.Config) BlockTraceResult` |
| Console | `debug.traceBlockByHash(hash, [options])`                                       |
| RPC     | `{"method": "debug_traceBlockByHash", "params": [hash {}]}`                     |

References:
[RLP](https://github.com/ethereum/wiki/wiki/RLP)

### debug_traceBlockFromFile

Similar to [debug_traceBlock](#debug_traceBlock), `traceBlockFromFile` accepts a file containing the RLP of the block.

| Client  | Method invocation                                                                |
|:-------:|----------------------------------------------------------------------------------|
| Go      | `debug.TraceBlockFromFile(fileName string, config. *vm.Config) BlockTraceResult` |
| Console | `debug.traceBlockFromFile(fileName, [options])`                                  |
| RPC     | `{"method": "debug_traceBlockFromFile", "params": [fileName, {}]}`               |

References:
[RLP](https://github.com/ethereum/wiki/wiki/RLP)

### debug_traceTransaction

The `traceTransaction` debugging method will attempt to run the transaction in the exact same manner
as it was executed on the network. It will replay any transaction that may have been executed prior
to this one before it will finally attempt to execute the transaction that corresponds to the given
hash.

In addition to the hash of the transaction you may give it a secondary *optional* argument, which
specifies the options for this specific call. The possible options are:

* `disableStorage`: `BOOL`. Setting this to true will disable storage capture (default = false).
* `disableMemory`: `BOOL`. Setting this to true will disable memory capture (default = false).
* `disableStack`: `BOOL`. Setting this to true will disable stack capture (default = false).
* `tracer`: `STRING`. Setting this will enable JavaScript-based transaction tracing, described below. If set, the previous four arguments will be ignored.
* `timeout`: `STRING`. Overrides the default timeout of 5 seconds for JavaScript-based tracing calls. Valid values are described [here](https://golang.org/pkg/time/#ParseDuration).

| Client  | Method invocation                                                                            |
|:-------:|----------------------------------------------------------------------------------------------|
| Go      | `debug.TraceTransaction(txHash common.Hash, logger *vm.LogConfig) (*ExecutionResurt, error)` |
| Console | `debug.traceTransaction(txHash, [options])`                                                  |
| RPC     | `{"method": "debug_traceTransaction", "params": [txHash, {}]}`                               |

#### Example

```javascript
> debug.traceTransaction("0x2059dd53ecac9827faad14d364f9e04b1d5fe5b506e3acc886eff7a6f88a696a")
{
  gas: 85301,
  returnValue: "",
  structLogs: [{
      depth: 1,
      error: "",
      gas: 162106,
      gasCost: 3,
      memory: null,
      op: "PUSH1",
      pc: 0,
      stack: [],
      storage: {}
  },
    /* snip */
  {
      depth: 1,
      error: "",
      gas: 100000,
      gasCost: 0,
      memory: ["0000000000000000000000000000000000000000000000000000000000000006", "0000000000000000000000000000000000000000000000000000000000000000", "0000000000000000000000000000000000000000000000000000000000000060"],
      op: "STOP",
      pc: 120,
      stack: ["00000000000000000000000000000000000000000000000000000000d67cbec9"],
      storage: {
        0000000000000000000000000000000000000000000000000000000000000004: "8241fa522772837f0d05511f20caa6da1d5a3209000000000000000400000001",
        0000000000000000000000000000000000000000000000000000000000000006: "0000000000000000000000000000000000000000000000000000000000000001",
        f652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f: "00000000000000000000000002e816afc1b5c0f39852131959d946eb3b07b5ad"
      }
  }]
```

#### JavaScript-based tracing
Specifying the `tracer` option in the second argument enables JavaScript-based tracing. In this mode, `tracer` is interpreted as a JavaScript expression that is expected to evaluate to an object with (at least) two methods, named `step` and `result`.

`step`is a function that takes two arguments, log and db, and is called for each step of the EVM, or when an error occurs, as the specified transaction is traced.

`log` has the following fields:

 - `pc`: Number, the current program counter
 - `op`: Object, an OpCode object representing the current opcode
 - `gas`: Number, the amount of gas remaining
 - `gasPrice`: Number, the cost in wei of each unit of gas
 - `memory`: Object, a structure representing the contract's memory space
 - `stack`: array[big.Int], the EVM execution stack
 - `depth`: The execution depth
 - `account`: The address of the account executing the current operation
 - `err`: If an error occured, information about the error

If `err` is non-null, all other fields should be ignored.

For efficiency, the same `log` object is reused on each execution step, updated with current values; make sure to copy values you want to preserve beyond the current call. For instance, this step function will not work:

    function(log) {
      this.logs.append(log);
    }

But this step function will:

    function(log) {
      this.logs.append({gas: log.gas, pc: log.pc, ...});
    }

`log.op` has the following methods:

 - `isPush()` - returns true iff the opcode is a PUSHn
 - `toString()` - returns the string representation of the opcode
 - `toNumber()` - returns the opcode's number

`log.memory` has the following methods:

 - `slice(start, stop)` - returns the specified segment of memory as a byte slice
 - `length()` - returns the length of the memory

`log.stack` has the following methods:

 - `peek(idx)` - returns the idx-th element from the top of the stack (0 is the topmost element) as a big.Int
 - `length()` - returns the number of elements in the stack

`db` has the following methods:

 - `getBalance(address)` - returns a `big.Int` with the specified account's balance
 - `getNonce(address)` - returns a Number with the specified account's nonce
 - `getCode(address)` - returns a byte slice with the code for the specified account
 - `getState(address, hash)` - returns the state value for the specified account and the specified hash
 - `exists(address)` - returns true if the specified address exists

The second function, 'result', takes no arguments, and is expected to return a JSON-serializable value to return to the RPC caller.

If the step function throws an exception or executes an illegal operation at any point, it will not be called on any further VM steps, and the error will be returned to the caller.

Note that several values are Golang big.Int objects, not JavaScript numbers or JS bigints. As such, they have the same interface as described in the godocs. Their default serialization to JSON is as a Javascript number; to serialize large numbers accurately call `.String()` on them. For convenience, `big.NewInt(x)` is provided, and will convert a uint to a Go BigInt.

Usage example, returns the top element of the stack at each CALL opcode only:

    debug.traceTransaction(txhash, {tracer: '{data: [], step: function(log) { if(log.op.toString() == "CALL") this.data.push(log.stack.peek(0)); }, result: function() { return this.data; }}'});

### debug_verbosity

Sets the logging verbosity ceiling. Log messages with level 
up to and including the given level will be printed.

The verbosity of individual packages and source files
can be raised using `debug_vmodule`.

| Client  | Method invocation                                 |
|:-------:|---------------------------------------------------|
| Console | `debug.verbosity(level)`                          |
| RPC     | `{"method": "debug_vmodule", "params": [number]}` |

### debug_vmodule

Sets the logging verbosity pattern.

| Client  | Method invocation                                 |
|:-------:|---------------------------------------------------|
| Console | `debug.vmodule(string)`                           |
| RPC     | `{"method": "debug_vmodule", "params": [string]}` |


#### Examples

If you want to see messages from a particular Go package (directory)
and all subdirectories, use:

``` javascript
> debug.vmodule("eth/*=6")
```

If you want to restrict messages to a particular package (e.g. p2p)
but exclude subdirectories, use:

``` javascript
> debug.vmodule("p2p=6")
```

If you want to see log messages from a particular source file, use

``` javascript
> debug.vmodule("server.go=6")
```

You can compose these basic patterns. If you want to see all
output from peer.go in a package below eth (eth/peer.go,
eth/downloader/peer.go) as well as output from package p2p
at level <= 5, use:

``` javascript
debug.vmodule("eth/*/peer.go=6,p2p=5")
```

### debug_writeBlockProfile

Writes a goroutine blocking profile to the given file.

| Client  | Method invocation                                           |
|:-------:|-------------------------------------------------------------|
| Console | `debug.writeBlockProfile(file)`                             |
| RPC     | `{"method": "debug_writeBlockProfile", "params": [string]}` |

### debug_writeMemProfile

Writes an allocation profile to the given file.
Note that the profiling rate cannot be set through the API,
it must be set on the command line using the `--memprofilerate`
flag.

| Client  | Method invocation                                           |
|:-------:|-------------------------------------------------------------|
| Console | `debug.writeMemProfile(file string)`                        |
| RPC     | `{"method": "debug_writeBlockProfile", "params": [string]}` |

## Miner

The `miner` API allows you to remote control the node's mining operation and set various
mining specific settings.

### miner_setExtra

Sets the extra data a miner can include when miner blocks. This is capped at
32 bytes.

| Client  | Method invocation                                  |
|:-------:|----------------------------------------------------|
| Go      | `miner.setExtra(extra string) (bool, error)`       |
| Console | `miner.setExtra(string)`                           |
| RPC     | `{"method": "miner_setExtra", "params": [string]}` |


### miner_setGasPrice

Sets the minimal accepted gas price when mining transactions. Any transactions that are
below this limit are excluded from the mining process.

| Client  | Method invocation                                     |
|:-------:|-------------------------------------------------------|
| Go      | `miner.setGasPrice(number *rpc.HexNumber) bool`       |
| Console | `miner.setGasPrice(number)`                           |
| RPC     | `{"method": "miner_setGasPrice", "params": [number]}` |

### miner_start

Start the CPU mining process with the given number of threads and generate a new DAG
if need be.

| Client  | Method invocation                                   |
|:-------:|-----------------------------------------------------|
| Go      | `miner.Start(threads *rpc.HexNumber) (bool, error)` |
| Console | `miner.start(number)`                               |
| RPC     | `{"method": "miner_start", "params": [number]}`     |


### miner_stop

Stop the CPU mining operation.

| Client  | Method invocation                            |
|:-------:|----------------------------------------------|
| Go      | `miner.Stop() bool`                          |
| Console | `miner.stop()`                               |
| RPC     | `{"method": "miner_stop", "params": []}`     |


### miner_setEtherBase

Sets the etherbase, where mining rewards will go. 

| Client  | Method invocation                                           |
|:-------:|-------------------------------------------------------------|
| Go      | `miner.SetEtherbase(common.Address) bool`                   |
| Console | `miner.setEtherbase(address)`                               |
| RPC     | `{"method": "miner_setEtherbase", "params": [address]}`     |


## Personal

The personal API manages private keys in the key store.

### personal_importRawKey

Imports the given unencrypted private key (hex string) into the key store,
encrypting it with the passphrase.

Returns the address of the new account.
 
| Client    | Method invocation                                                 |
| :-------: | ----------------------------------------------------------------- |
| Console   | `personal.importRawKey(keydata, passphrase)`                      |
| RPC       | `{"method": "personal_importRawKey", "params": [string, string]}` |

### personal_listAccounts

Returns all the Ethereum account addresses of all keys
in the key store.

| Client    | Method invocation                                   |
| :-------: | --------------------------------------------------- |
| Console   | `personal.listAccounts`                             |
| RPC       | `{"method": "personal_listAccounts", "params": []}` |

#### Example

``` javascript
> personal.listAccounts
["0x5e97870f263700f46aa00d967821199b9bc5a120", "0x3d80b31a78c30fc628f20b2c89d7ddbf6e53cedc"]
```

### personal_lockAccount

Removes the private key with given address from memory.
The account can no longer be used to send transactions.
 
| Client    | Method invocation                                        |
| :-------: | -------------------------------------------------------- |
| Console   | `personal.lockAccount(address)`                          |
| RPC       | `{"method": "personal_lockAccount", "params": [string]}` |

### personal_newAccount

Generates a new private key and stores it in the key store directory.
The key file is encrypted with the given passphrase.
Returns the address of the new account.

At the geth console, `newAccount` will prompt for a passphrase when 
it is not supplied as the argument.

| Client    | Method invocation                                       |
| :-------: | ---------------------------------------------------     |
| Console   | `personal.newAccount()`                                 |
| RPC       | `{"method": "personal_newAccount", "params": [string]}` |

#### Example
 
``` javascript
> personal.newAccount()
Passphrase: 
Repeat passphrase: 
"0x5e97870f263700f46aa00d967821199b9bc5a120"
```

The passphrase can also be supplied as a string.

``` javascript
> personal.newAccount("h4ck3r")
"0x3d80b31a78c30fc628f20b2c89d7ddbf6e53cedc"
```

### personal_unlockAccount

Decrypts the key with the given address from the key store.

Both passphrase and unlock duration are optional when using the JavaScript console.
If the passphrase is not supplied as an argument, the console will prompt for
the passphrase interactively.

The unencrypted key will be held in memory until the unlock duration expires.
If the unlock duration defaults to 300 seconds. An explicit duration
of zero seconds unlocks the key until geth exits.

The account can be used with `eth_sign` and `eth_sendTransaction` while it is unlocked.
 
| Client    | Method invocation                                                          |
| :-------: | -------------------------------------------------------------------------- |
| Console   | `personal.unlockAccount(address, passphrase, duration)`                    |
| RPC       | `{"method": "personal_unlockAccount", "params": [string, string, number]}` |

#### Examples

``` javascript
> personal.unlockAccount("0x5e97870f263700f46aa00d967821199b9bc5a120")
Unlock account 0x5e97870f263700f46aa00d967821199b9bc5a120
Passphrase: 
true
```

Supplying the passphrase and unlock duration as arguments:

``` javascript
> personal.unlockAccount("0x5e97870f263700f46aa00d967821199b9bc5a120", "foo", 30)
true
```

If you want to type in the passphrase and stil override the default unlock duration,
pass `null` as the passphrase.

```
> personal.unlockAccount("0x5e97870f263700f46aa00d967821199b9bc5a120", null, 30)
Unlock account 0x5e97870f263700f46aa00d967821199b9bc5a120
Passphrase: 
true
```

### personal_sendTransaction

Validate the given passphrase and submit transaction.

The transaction is the same argument as for `eth_sendTransaction` and contains the `from` address. If the passphrase can be used to decrypt the private key belogging to `tx.from` the transaction is verified, signed and send onto the network. The account is not unlocked globally in the node and cannot be used in other RPC calls.

| Client    | Method invocation                                                |
| :-------: | -----------------------------------------------------------------|
| Console   | `personal.sendTransaction(tx, passphrase)`                       |
| RPC       | `{"method": "personal_sendTransaction", "params": [tx, string]}` |

*Note, prior to Geth 1.5, please use `personal_signAndSendTransaction` as that was the original introductory name and only later renamed to the current final version.*

#### Examples

``` javascript
> var tx = {from: "0x391694e7e0b0cce554cb130d723a9d27458f9298", to: "0xafa3f8684e54059998bc3a7b0d2b0da075154d66", value: web3.toWei(1.23, "ether")}
undefined
> personal.sendTransaction(tx, "passphrase")
0x8474441674cdd47b35b875fd1a530b800b51a5264b9975fb21129eeb8c18582f
```

### personal_sign

The sign method calculates an Ethereum specific signature with:
`sign(keccack256("\x19Ethereum Signed Message:\n" + len(message) + message)))`.

By adding a prefix to the message makes the calculated signature recognisable as an Ethereum specific signature. This prevents misuse where a malicious DApp can sign arbitrary data (e.g. transaction) and use the signature to impersonate the victim.

See ecRecover to verify the signature.

| Client  | Method invocation                                     |
|:-------:|-------------------------------------------------------|   
| Console | `personal.sign(message, account, [password])`                |
| RPC     | `{"method": "personal_sign", "params": [message, account, password]}` |


#### Examples

``` javascript
> personal.sign("0xdeadbeaf", "0x9b2055d370f73ec7d8a03e965129118dc8f5bf83", "")
"0xa3f20717a250c2b0b729b7e5becbff67fdaef7e0699da4de7ca5895b02a170a12d887fd3b17bfdce3481f10bea41f45ba9f709d39ce8325427b57afcfc994cee1b"
```

### personal_ecRecover

`ecRecover` returns the address associated with the private key that was used to calculate the signature in `personal_sign`. 

| Client  | Method invocation                                     |
|:-------:|-------------------------------------------------------|   
| Console | `personal.ecRecover(message, signature)`                 |
| RPC     | `{"method": "personal_ecRecover", "params": [message, signature]}` |


#### Examples

``` javascript
> personal.sign("0xdeadbeaf", "0x9b2055d370f73ec7d8a03e965129118dc8f5bf83", "")
"0xa3f20717a250c2b0b729b7e5becbff67fdaef7e0699da4de7ca5895b02a170a12d887fd3b17bfdce3481f10bea41f45ba9f709d39ce8325427b57afcfc994cee1b"
> personal.ecRecover("0xdeadbeaf", "0xa3f20717a250c2b0b729b7e5becbff67fdaef7e0699da4de7ca5895b02a170a12d887fd3b17bfdce3481f10bea41f45ba9f709d39ce8325427b57afcfc994cee1b")
"0x9b2055d370f73ec7d8a03e965129118dc8f5bf83"
```


## Txpool

The `txpool` API gives you access to several non-standard RPC methods to inspect the contents of the
transaction pool containing all the currently pending transactions as well as the ones queued for
future processing.

### txpool_content

The `content` inspection property can be queried to list the exact details of all the transactions
currently pending for inclusion in the next block(s), as well as the ones that are being scheduled
for future execution only.

The result is an object with two fields `pending` and `queued`. Each of these fields are associative
arrays, in which each entry maps an origin-address to a batch of scheduled transactions. These batches
themselves are maps associating nonces with actual transactions.

Please note, there may be multiple transactions associated with the same account and nonce. This can
happen if the user broadcast mutliple ones with varying gas allowances (or even complerely different
transactions).

| Client  | Method invocation                                                       |
|:-------:|-------------------------------------------------------------------------|
| Go      | `txpool.Content() (map[string]map[string]map[string][]*RPCTransaction)` |
| Console | `txpool.content`                                                        |
| RPC     | `{"method": "txpool_content"}`                                          |

#### Example

```javascript
> txpool.content
{
  pending: {
    0x0216d5032f356960cd3749c31ab34eeff21b3395: {
      806: [{
        blockHash: "0x0000000000000000000000000000000000000000000000000000000000000000",
        blockNumber: null,
        from: "0x0216d5032f356960cd3749c31ab34eeff21b3395",
        gas: "0x5208",
        gasPrice: "0xba43b7400",
        hash: "0xaf953a2d01f55cfe080c0c94150a60105e8ac3d51153058a1f03dd239dd08586",
        input: "0x",
        nonce: "0x326",
        to: "0x7f69a91a3cf4be60020fb58b893b7cbb65376db8",
        transactionIndex: null,
        value: "0x19a99f0cf456000"
      }]
    },
    0x24d407e5a0b506e1cb2fae163100b5de01f5193c: {
      34: [{
        blockHash: "0x0000000000000000000000000000000000000000000000000000000000000000",
        blockNumber: null,
        from: "0x24d407e5a0b506e1cb2fae163100b5de01f5193c",
        gas: "0x44c72",
        gasPrice: "0x4a817c800",
        hash: "0xb5b8b853af32226755a65ba0602f7ed0e8be2211516153b75e9ed640a7d359fe",
        input: "0xb61d27f600000000000000000000000024d407e5a0b506e1cb2fae163100b5de01f5193c00000000000000000000000000000000000000000000000053444835ec580000000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
        nonce: "0x22",
        to: "0x7320785200f74861b69c49e4ab32399a71b34f1a",
        transactionIndex: null,
        value: "0x0"
      }]
    }
  },
  queued: {
    0x976a3fc5d6f7d259ebfb4cc2ae75115475e9867c: {
      3: [{
        blockHash: "0x0000000000000000000000000000000000000000000000000000000000000000",
        blockNumber: null,
        from: "0x976a3fc5d6f7d259ebfb4cc2ae75115475e9867c",
        gas: "0x15f90",
        gasPrice: "0x4a817c800",
        hash: "0x57b30c59fc39a50e1cba90e3099286dfa5aaf60294a629240b5bbec6e2e66576",
        input: "0x",
        nonce: "0x3",
        to: "0x346fb27de7e7370008f5da379f74dd49f5f2f80f",
        transactionIndex: null,
        value: "0x1f161421c8e0000"
      }]
    },
    0x9b11bf0459b0c4b2f87f8cebca4cfc26f294b63a: {
      2: [{
        blockHash: "0x0000000000000000000000000000000000000000000000000000000000000000",
        blockNumber: null,
        from: "0x9b11bf0459b0c4b2f87f8cebca4cfc26f294b63a",
        gas: "0x15f90",
        gasPrice: "0xba43b7400",
        hash: "0x3a3c0698552eec2455ed3190eac3996feccc806970a4a056106deaf6ceb1e5e3",
        input: "0x",
        nonce: "0x2",
        to: "0x24a461f25ee6a318bdef7f33de634a67bb67ac9d",
        transactionIndex: null,
        value: "0xebec21ee1da40000"
      }],
      6: [{
        blockHash: "0x0000000000000000000000000000000000000000000000000000000000000000",
        blockNumber: null,
        from: "0x9b11bf0459b0c4b2f87f8cebca4cfc26f294b63a",
        gas: "0x15f90",
        gasPrice: "0x4a817c800",
        hash: "0xbbcd1e45eae3b859203a04be7d6e1d7b03b222ec1d66dfcc8011dd39794b147e",
        input: "0x",
        nonce: "0x6",
        to: "0x6368f3f8c2b42435d6c136757382e4a59436a681",
        transactionIndex: null,
        value: "0xf9a951af55470000"
      }, {
        blockHash: "0x0000000000000000000000000000000000000000000000000000000000000000",
        blockNumber: null,
        from: "0x9b11bf0459b0c4b2f87f8cebca4cfc26f294b63a",
        gas: "0x15f90",
        gasPrice: "0x4a817c800",
        hash: "0x60803251d43f072904dc3a2d6a084701cd35b4985790baaf8a8f76696041b272",
        input: "0x",
        nonce: "0x6",
        to: "0x8db7b4e0ecb095fbd01dffa62010801296a9ac78",
        transactionIndex: null,
        value: "0xebe866f5f0a06000"
      }],
    }
  }
}
```

### txpool_inspect

The `inspect` inspection property can be queried to list a textual summary of all the transactions
currently pending for inclusion in the next block(s), as well as the ones that are being scheduled
for future execution only. This is a method specifically tailored to developers to quickly see the
transactions in the pool and find any potential issues.

The result is an object with two fields `pending` and `queued`. Each of these fields are associative
arrays, in which each entry maps an origin-address to a batch of scheduled transactions. These batches
themselves are maps associating nonces with transactions summary strings.

Please note, there may be multiple transactions associated with the same account and nonce. This can
happen if the user broadcast mutliple ones with varying gas allowances (or even complerely different
transactions).

| Client  | Method invocation                                              |
|:-------:|----------------------------------------------------------------|
| Go      | `txpool.Inspect() (map[string]map[string]map[string][]string)` |
| Console | `txpool.inspect`                                               |
| RPC     | `{"method": "txpool_inspect"}`                                 |

#### Example

```javascript
> txpool.inspect
{
  pending: {
    0x26588a9301b0428d95e6fc3a5024fce8bec12d51: {
      31813: ["0x3375ee30428b2a71c428afa5e89e427905f95f7e: 0 wei + 500000 × 20000000000 gas"]
    },
    0x2a65aca4d5fc5b5c859090a6c34d164135398226: {
      563662: ["0x958c1fa64b34db746925c6f8a3dd81128e40355e: 1051546810000000000 wei + 90000 × 20000000000 gas"],
      563663: ["0x77517b1491a0299a44d668473411676f94e97e34: 1051190740000000000 wei + 90000 × 20000000000 gas"],
      563664: ["0x3e2a7fe169c8f8eee251bb00d9fb6d304ce07d3a: 1050828950000000000 wei + 90000 × 20000000000 gas"],
      563665: ["0xaf6c4695da477f8c663ea2d8b768ad82cb6a8522: 1050544770000000000 wei + 90000 × 20000000000 gas"],
      563666: ["0x139b148094c50f4d20b01caf21b85edb711574db: 1048598530000000000 wei + 90000 × 20000000000 gas"],
      563667: ["0x48b3bd66770b0d1eecefce090dafee36257538ae: 1048367260000000000 wei + 90000 × 20000000000 gas"],
      563668: ["0x468569500925d53e06dd0993014ad166fd7dd381: 1048126690000000000 wei + 90000 × 20000000000 gas"],
      563669: ["0x3dcb4c90477a4b8ff7190b79b524773cbe3be661: 1047965690000000000 wei + 90000 × 20000000000 gas"],
      563670: ["0x6dfef5bc94b031407ffe71ae8076ca0fbf190963: 1047859050000000000 wei + 90000 × 20000000000 gas"]
    },
    0x9174e688d7de157c5c0583df424eaab2676ac162: {
      3: ["0xbb9bc244d798123fde783fcc1c72d3bb8c189413: 30000000000000000000 wei + 85000 × 21000000000 gas"]
    },
    0xb18f9d01323e150096650ab989cfecd39d757aec: {
      777: ["0xcd79c72690750f079ae6ab6ccd7e7aedc03c7720: 0 wei + 1000000 × 20000000000 gas"]
    },
    0xb2916c870cf66967b6510b76c07e9d13a5d23514: {
      2: ["0x576f25199d60982a8f31a8dff4da8acb982e6aba: 26000000000000000000 wei + 90000 × 20000000000 gas"]
    },
    0xbc0ca4f217e052753614d6b019948824d0d8688b: {
      0: ["0x2910543af39aba0cd09dbb2d50200b3e800a63d2: 1000000000000000000 wei + 50000 × 1171602790622 gas"]
    },
    0xea674fdde714fd979de3edf0f56aa9716b898ec8: {
      70148: ["0xe39c55ead9f997f7fa20ebe40fb4649943d7db66: 1000767667434026200 wei + 90000 × 20000000000 gas"]
    }
  },
  queued: {
    0x0f6000de1578619320aba5e392706b131fb1de6f: {
      6: ["0x8383534d0bcd0186d326c993031311c0ac0d9b2d: 9000000000000000000 wei + 21000 × 20000000000 gas"]
    },
    0x5b30608c678e1ac464a8994c3b33e5cdf3497112: {
      6: ["0x9773547e27f8303c87089dc42d9288aa2b9d8f06: 50000000000000000000 wei + 90000 × 50000000000 gas"]
    },
    0x976a3fc5d6f7d259ebfb4cc2ae75115475e9867c: {
      3: ["0x346fb27de7e7370008f5da379f74dd49f5f2f80f: 140000000000000000 wei + 90000 × 20000000000 gas"]
    },
    0x9b11bf0459b0c4b2f87f8cebca4cfc26f294b63a: {
      2: ["0x24a461f25ee6a318bdef7f33de634a67bb67ac9d: 17000000000000000000 wei + 90000 × 50000000000 gas"],
      6: ["0x6368f3f8c2b42435d6c136757382e4a59436a681: 17990000000000000000 wei + 90000 × 20000000000 gas", "0x8db7b4e0ecb095fbd01dffa62010801296a9ac78: 16998950000000000000 wei + 90000 × 20000000000 gas"],
      7: ["0x6368f3f8c2b42435d6c136757382e4a59436a681: 17900000000000000000 wei + 90000 × 20000000000 gas"]
    }
  }
}
```

### txpool_status

The `status` inspection property can be queried for the number of transactions currently pending for
inclusion in the next block(s), as well as the ones that are being scheduled for future execution only.

The result is an object with two fields `pending` and `queued`, each of which is a counter representing
the number of transactions in that particular state.

| Client  | Method invocation                             |
|:-------:|-----------------------------------------------|
| Go      | `txpool.Status() (map[string]*rpc.HexNumber)` |
| Console | `txpool.status`                               |
| RPC     | `{"method": "txpool_status"}`                 |

#### Example

```javascript
> txpool.status
{
  pending: 10,
  queued: 7
}
```