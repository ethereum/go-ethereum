---
title: Management APIs
---

Beside the official [DApp APIs](https://github.com/ethereum/wiki/wiki/JSON-RPC) interface
go-ethereum has support for additional management APIs. Similar to the DApp APIs, these
are also provided using [JSON-RPC](http://www.jsonrpc.org/specification) and follow
exactly the same conventions. Geth comes with a console client which has support for all
additional APIs described here.

## Enabling the management APIs

To offer these APIs over the Geth RPC endpoints, please specify them with the
`--${interface}api` command line argument (where `${interface}` can be `rpc` for the HTTP
endpoint, `ws` for the WebSocket endpoint and `ipc` for the unix socket (Unix) or named
pipe (Windows) endpoint).

For example: `geth --ipcapi admin,eth,miner --rpcapi eth,web3 --rpc`

* Enables the admin, official DApp and miner API over the IPC interface
* Enables the official DApp and web3 API over the HTTP interface

The HTTP RPC interface must be explicitly enabled using the `--rpc` flag.

Please note, offering an API over the HTTP (`rpc`) or WebSocket (`ws`) interfaces will
give everyone access to the APIs who can access this interface (DApps, browser tabs, etc).
Be careful which APIs you enable. By default Geth enables all APIs over the IPC (`ipc`)
interface and only the `db`, `eth`, `net` and `web3` APIs over the HTTP and WebSocket
interfaces.

To determine which APIs an interface provides, the `modules` JSON-RPC method can be
invoked. For example over an `ipc` interface on unix systems:

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
for the [JavaScript console](javascript-console) but
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
| [addPeer](#admin_addpeer)    | [backtraceAt](#debug_backtraceat)                 | [setExtra](#miner_setextra)         | [ecRecover](#personal_ecrecover)         | [content](#txpool_content) |
| [datadir](#admin_datadir)          | [blockProfile](#debug_blockprofile)               | [setGasPrice](#miner_setgasprice)   | [importRawKey](#personal_importrawkey)   | [inspect](#txpool_inspect) |
| [nodeInfo](#admin_nodeinfo)  | [cpuProfile](#debug_cpuprofile)                   | [start](#miner_start)               | [listAccounts](#personal_listaccounts)   | [status](#txpool_status)   |
| [peers](#admin_peers)        | [dumpBlock](#debug_dumpblock)                     | [stop](#miner_stop)                 | [lockAccount](#personal_lockaccount)     |                            |
| [setSolc](#admin_setsolc)    | [gcStats](#debug_gcstats)                         | [getHashrate](#miner_gethashrate)   | [newAccount](#personal_newaccount)       |                            |
| [startRPC](#admin_startrpc)  | [getBlockRlp](#debug_getblockrlp)                 | [setEtherbase](#miner_setetherbase) | [unlockAccount](#personal_unlockaccount) |                            |
| [startWS](#admin_startws)    | [goTrace](#debug_gotrace)                         |                                     | [sendTransaction](#personal_sendtransaction) |                        |
| [stopRPC](#admin_stoprpc)    | [memStats](#debug_memstats)                       |                                     | [sign](#personal_sign)                   |                            |
| [stopWS](#admin_stopws)      | [seedHash](#debug_seedhash)[sign](#personal_sign)|                                      |                                          |                            |
|                              | [setBlockProfileRate](#debug_setblockprofilerate) |                                     |                                          |                            |
|                              | [setHead](#debug_sethead)                         |                                     |                                          |                            |
|                              | [stacks](#debug_stacks)                           |                                     |                                          |                            |
|                              | [startCPUProfile](#debug_startcpuprofile)         |                                     |                                          |                            |
|                              | [startGoTrace](#debug_startgotrace)               |                                     |                                          |                            |
|                              | [stopCPUProfile](#debug_stopcpuprofile)           |                                     |                                          |                            |
|                              | [stopGoTrace](#debug_stopgotrace)                 |                                     |                                          |                            |
|                              | [traceBlock](#debug_traceblock)                   |                                     |                                          |                            |
|                              | [traceBlockByNumber](#debug_traceblockbynumber)        |                                     |                                          |                            |
|                              | [traceBlockByHash](#debug_traceblockbyhash)            |                                     |                                          |                            |
|                              | [traceBlockFromFile](#debug_traceblockfromfile)   |                                     |                                          |                            |
|                              | [traceTransaction](#debug_tracetransaction)       |                                     |                                          |                            |
|                              | [verbosity](#debug_verbosity)                     |                                     |                                          |                            |
|                              | [vmodule](#debug_vmodule)                         |                                     |                                          |                            |
|                              | [writeBlockProfile](#debug_writeblockprofile)     |                                     |                                          |                            |
|                              | [writeMemProfile](#debug_writememprofile)         |                                     |                                          |                            |
|                              | [standardTraceBlockToFile](#debug_standardtraceblocktofile)|                             |                                          |                            | 



