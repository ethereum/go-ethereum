---
title: Geth logs
sort key: N
---

A Geth node continually reports messages to the console allowing users to monitor Geth's current 
status in real-time. The logs indicate when Geth is running normally and indicates when some 
attention is required. However, reading these logs can be difficult for new users. This page 
will help to interpret the log messages to better understand what Geth is doing.

Note that there are a large number of log messages covering a wide range of possible scenarios for 
a Geth node. This page will only address a subset of commonly seen messages. For more, see the 
[Geth Github](https://github.com/ethereum/go-ethereum), [Discord](https://discord.gg/WHNkYDsAKU) 
or search on [ethereum.stackexchange](https://ethereum.stackexchange.com/). Log messages are 
usually sufficiently self-describing that they do not require additional explanation.


## Configuring log messages

Log messages are displayed to the console by default. The messages can be tuned to be more or less 
detailed by passing `--verbosity` and a value between 0 and 6 to Geth at startup:

```sh
0 = silent (no log messages)
1 = error (error messages only)
2 = warn (error messages and warnings only)
3 = info (error messages, warnings and normal activity logs)
4 = debug (all info plus additional messages for debugging)
5 = detail (all info plus detailed debugging messages)
```
The default is `--verbosity 3`.

Log messages can also be redirected so they are saved to a text file instead of being displayed in 
the console. In Linux the syntax `>> <path> 2>&1` redirects both `stdout` and `stderr` messages 
to `<path>`. For example:

```sh
# saves detailed logs to path/eth.log
geth --verbosity 5 >> /path/eth.log 2>&1
```

### Startup

When Geth starts up it immediately reports a fairly long page of configuration details and status 
reports that allow the user to confirm Geth is on the right network and operating in its intended 
modes. The basic structure of a log message is as follows:

```
MESSAGE_TYPE [MONTH-DAY][TIME] MESSAGE VALUE
```
Where `MESSAGE_TYPE` can be `INFO`, `WARN`, `ERROR` or `DEBUG`. These tags categorize log messages 
according to their purpose. `INFO` messages inform the user about Geth's current configuration and 
status. `WARN` messages are for alerting the user to details that affect the way Geth is running. 
`ERROR` messages are for alerting the user to problems. `DEBUG` is for messages that are relevant 
to troubleshooting or for developers working on Geth.

The messages displayed on startup break down as follows:

```
INFO [10-04|10:20:52.028] Starting Geth on Ethereum mainnet... 
INFO [10-04|10:20:52.028] Bumping default cache on mainnet         provided=1024 updated=4096
INFO [10-04|10:20:52.030] Maximum peer count                       ETH=50 LES=0 total=50
INFO [10-04|10:20:52.031] Smartcard socket not found, disabling    err="stat /run/pcscd/pcscd.comm: no such file or directory"
INFO [10-04|10:20:52.034] Set global gas cap                       cap=50,000,000
INFO [10-04|10:20:52.035] Allocated trie memory caches             clean=614.00MiB dirty=1024.00MiB
INFO [10-04|10:20:52.035] Allocated cache and file handles         database=/home/go-ethereum/devnet/geth/chaindata cache=2.00GiB handles=524,288
INFO [10-04|10:20:52.128] Opened ancient database                  database=/home/go-ethereum/devnet/geth/chaindata/ancient/chain readonly=false
INFO [10-04|10:20:52.129] Disk storage enabled for ethash caches   dir=/home/go-ethereum/devnet/geth/ethash count=3
INFO [10-04|10:20:52.129] Disk storage enabled for ethash DAGs     dir=/home/.ethash                        count=2
INFO [10-04|10:20:52.129] Initialising Ethereum protocol           network=1 dbversion=<nil>
INFO [10-04|10:20:52.129] Writing default main-net genesis block 
INFO [10-04|10:20:52.372] Persisted trie from memory database      nodes=12356 size=1.78MiB time=21.535537ms gcnodes=0 gcsize=0.00B gctime=0s livenodes=1 livesize=0.00B
```

The logs above show the user that the node is connecting to Ethereum Mainnet and some low level 
configuration details. The cache size is bumped to the Mainnet default (4096). The maximum peer 
count is the highest number of peers this node is allowed to connect to and can be used to 
control the bandwidth requirements of the node. Logs relating to `ethash` are out of date since 
Ethereum moved to proof-of-stake based consensus and can safely be ignored. 


```
--------------------------------------------------------------------------------------------------------------------------------------------------------- 
INFO [10-04|10:20:52.386] Chain ID:  1 (mainnet) 
INFO [10-04|10:20:52.386] Consensus: Beacon (proof-of-stake), merged from Ethash (proof-of-work) 
INFO [10-04|10:20:52.386]  
INFO [10-04|10:20:52.386] Pre-Merge hard forks: 
INFO [10-04|10:20:52.386]  - Homestead:                   1150000  (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/homestead.md) 
INFO [10-04|10:20:52.386]  - DAO Fork:                    1920000  (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/dao-fork.md) 
INFO [10-04|10:20:52.386]  - Tangerine Whistle (EIP 150): 2463000  (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/tangerine-whistle.md) 
INFO [10-04|10:20:52.386]  - Spurious Dragon/1 (EIP 155): 2675000  (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/spurious-dragon.md) 
INFO [10-04|10:20:52.386]  - Spurious Dragon/2 (EIP 158): 2675000  (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/spurious-dragon.md) 
INFO [10-04|10:20:52.386]  - Byzantium:                   4370000  (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/byzantium.md) 
INFO [10-04|10:20:52.386]  - Constantinople:              7280000  (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/constantinople.md) 
INFO [10-04|10:20:52.386]  - Petersburg:                  7280000  (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/petersburg.md) 
INFO [10-04|10:20:52.386]  - Istanbul:                    9069000  (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/istanbul.md) 
INFO [10-04|10:20:52.387]  - Muir Glacier:                9200000  (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/muir-glacier.md) 
INFO [10-04|10:20:52.387]  - Berlin:                      12244000 (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/berlin.md) 
INFO [10-04|10:20:52.387]  - London:                      12965000 (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/london.md) 
INFO [10-04|10:20:52.387]  - Arrow Glacier:               13773000 (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/arrow-glacier.md) 
INFO [10-04|10:20:52.387]  - Gray Glacier:                15050000 (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/gray-glacier.md)
```

The above block of messages are related to past Ethereum hard forks. The names are the names 
of the hard forks and the numbers are the blocks at which the hard fork occurs. This means 
that blocks with numbers that exceed these values have the configuration required by that 
hard fork. The specification of each hard fork is available at the provided links (and more 
information is available on [ethereum.org](https://ethereum.org/en/history/)). The message 
`Consensus: Beacon (proof-of-stake), merged from Ethash (proof-of-work)` indicates that the 
node requires a Beacon node to follow the canonical chain - Geth cannot participate in 
consensus on its own.

```
INFO [10-04|10:20:52.387] Merge configured: 
INFO [10-04|10:20:52.387]  - Hard-fork specification:    https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/paris.md 
INFO [10-04|10:20:52.387]  - Network known to be merged: true 
INFO [10-04|10:20:52.387]  - Total terminal difficulty:  58750000000000000000000 
INFO [10-04|10:20:52.387]  - Merge netsplit block:       <nil> 
INFO [10-04|10:20:52.387] 
INFO [10-04|10:20:52.388] Chain post-merge, sync via beacon client 
WARN [10-04|10:20:52.388] Engine API enabled                       protocol=eth
--------------------------------------------------------------------------------------------------------------------------------------------------------- 
```

The messages above relate to [The Merge](https://ethereum.org/en/upgrades/merge/). The Merge was 
Ethereum's transition from proof-of-work to proof-of-stake based consensus. In Geth, The Merge 
came in the form of the Paris hard fork which was triggered at a [terminal total difficulty](https://ethereum.org/en/glossary/#terminal-total-difficulty) 
of 58750000000000000000000 instead of a preconfigured block number like previous hard forks. 
The hard fork specification is linked in the log message. The message `network known to be merged: true` 
indicates that the node is following a chain that has passed the terminal total difficulty and undergone 
the Paris hard fork. Since September 15 2022 this will always be true for nodes on Ethereum Mainnet 
(and the merged testnets Sepolia and Goerli). The warning `Engine API enabled` informs the user that 
Geth is exposing the set of API methods required for communication with a consensus client.

```
INFO [10-04|10:20:52.389] Starting peer-to-peer node               instance=Geth/v1.11.0-unstable-e004e7d2-20220926/linux-amd64/go1.19.1
INFO [10-04|10:20:52.409] New local node record                    seq=1,664,875,252,408 id=9aa0e5b14ccd75ec ip=127.0.0.1 udp=30303 tcp=30303
INFO [10-04|10:20:52.409] Started P2P networking                   self=enode://1ef45ab610c2893b70483bf1791b550e5a93763058b0abf7c6d9e6201e07212d61c4896d64de07342c9df734650e3b40812c2dc01f894b6c385acd180ed30fc8@127.0.0.1:30303
INFO [10-04|10:20:52.410] IPC endpoint opened                      url=/home/go-ethereum/devnet/geth.ipc
INFO [10-04|10:20:52.410] Generated JWT secret                     path=/home/go-ethereum/devnet/geth/jwtsecret
INFO [10-04|10:20:52.411] HTTP server started                      endpoint=127.0.0.1:8545 auth=false prefix= cors= vhosts=localhost
INFO [10-04|10:20:52.411] WebSocket enabled                        url=ws://127.0.0.1:8551
INFO [10-04|10:20:52.411] HTTP server started                      endpoint=127.0.0.1:8551 auth=true  prefix= cors=localhost vhosts=localhost
INFO [10-04|10:20:54.785] New local node record                    seq=1,664,875,252,409 id=9aa0e5b14ccd75ec ip=82.11.59.221 udp=30303 tcp=30303
INFO [10-04|10:20:55.267] Mapped network port                      proto=udp extport=30303 intport=30303 interface="UPNP IGDv1-IP1"
INFO [10-04|10:20:55.833] Mapped network port                      proto=tcp extport=30303 intport=30303 interface="UPNP IGDv1-IP1"
INFO [10-04|10:21:03.100] Looking for peers                        peercount=0 tried=20 static=0
```

The logs above relate to Geth starting up its peer-to-peer components and seeking other nodes to 
connect to. The long address reported to `Started P2P networking` is the nodes own enode address. 
The `IPC Endpoint` is the location of the node's IPC file that can be used to connect a Javascript 
console. There is a log message confirming that a JWT secret was generated and reporting its path. 
This is required to authenticate communication between Geth and the consensus client. There are 
also messages here reporting on the HTTP server that can be used to send requests to Geth. There 
should be two HTTP servers - one for interacting with Geth (defaults to `localhost:8545`) and one 
for communication with the consensus client (defaults to `localhost:8551`).


### Syncing

The default for Geth is to sync in snap mode. This requires a block header to be provided to Geth by 
the consensus client. The header is then used as a target to sync to. Geth requests block headers 
from its peers that are parents of the target until there is a continuous chain of sequential headers 
of sufficient length. Then, Geth requests block bodies and receipts for each header and simultaneously 
starts downloading state data. This state data is stored in the form of a [Patricia Merkle Trie](https://ethereum.org/en/developers/docs/data-structures-and-encoding/patricia-merkle-trie/). Only the leaves of the trie are downloaded, 
the full trie structure is then locally regenerated from the leaves up. Meanwhile, the blockchain 
continues to progress and the target header is updated. This means some of the regenerated state 
data needs to be updated. This is known as *healing*.

Assuming Geth has a synced consensus client and some peers it will start importing headers, block 
bodies and receipts. The log messages for data downloading look as follows:

```
INFO [07-28|10:29:49.681] Block synchronisation started
INFO [07-28|10:29:50.427] Imported new block headers               count=1    elapsed=253.434ms number=12,914,945 hash=ee1a08..9ce38a
INFO [07-28|10:30:00.224] Imported new block receipts              count=64   elapsed=13.703s   number=12,914,881 hash=fef964..d789fc age=18m5s     size=7.69MiB
INFO [07-28|10:30:18.658] Imported new block headers               count=1    elapsed=46.715ms  number=12,914,946 hash=7b24c8..2d8006
```

For state sync, Geth reports when the state heal is in progress. This can take a long time. 
The log message includes values for the number of `accounts`, `slots`, `codes` and `nodes` that were 
downloaded in the current healing phase, and the pending field is the number of state entires waiting 
to be downloaded. The `pending` value is not necessarily the number of state entries remaining until 
the healing is finished. As the blockchain progresses the state trie is updated and therefore the data
that needs to be downloaded to heal the trie can increase as well as decrease over time. Ultimately, 
the state should heal faster than the blockchain progresses so the node can get in sync. When the state 
healing is finished there is a post-sync snapshot generation phase. The node is not in sync until the 
state healing phase is over. If the node is still regularly reporting `State heal in progress` it is not 
yet in sync - the state healing is still ongoing.

```
INFO [07-28|10:30:21.965] State heal in progress                   accounts=169,633@7.48MiB  slots=57314@4.17MiB    codes=4895@38.14MiB nodes=43,293,196@11.70GiB pending=112,626
INFO [09-06|01:31:59.885] Rebuilding state snapshot
INFO [09-06|01:31:59.910] Resuming state snapshot generation root=bc64d4..fc1edd accounts=0 slots=0 storage=0.00B dangling=0 elapsed=18.838ms
```

The sync can be confirmed using [`eth.syncing`](https://ethereum.org/en/developers/docs/apis/json-rpc/#eth_syncing)
- it will return `false` if the node is in sync. If `eth.syncing` returns anything other than `false` it has not 
finished syncing. Generally, if syncing is still ongoing, `eth.syncing` will return block info that looks as follows:

```js
> eth.sycing

{
  currentBlock: 15285946,
  healedBytecodeBytes: 991164713,
  healedBytecodes: 130880,
  healedTrienodeBytes: 489298493475,
  healedTrienodes: 1752917331,
  healingBytecode: 0,
  healingTrienodes: 1745,
  highestBlock: 16345003,
  startingBlock: 12218525,
  syncedAccountBytes: 391561544809,
  syncedAccounts: 136498212,
  syncedBytecodeBytes: 2414143936,
  syncedBytecodes: 420599,
  syncedStorage: 496503178,
  syncedStorageBytes: 103368240246
}
```

There are other log messages that are commonly seen during syncing. For example:

```sh
WARN [09-28|11:06:01.363] Snapshot extension registration failed 
```

This warning is nothing to worry about - it is reporting a configuration mismatch between the node and a 
peer. It does not mean syncing is stalling or failing, it simply results in the peer being dropped and 
replaced.

```sh
# consensus client has identified a new head to use as a sync target - account for this in state sync
INFO [10-03|15:34:01.336] Forkchoice requested sync to new head    number=15,670,037 hash=84d4ec..4c0e2b
```

The message above indicates that the fork choice algorithm, which is run by the consensus client, has 
identified a new target Geth should sync up to. This redirects the sync to prevent syncing to an outdated 
target and is a natural part of syncing a live blockchain.


## Transaction logs

Transactions submitted over local IPC, Websockets or HTTP connections are reported in the console logs. 
For example, a simple ETH transaction appears in the console logs as follows:

```sh
INFO [09-06|01:31:59.910] Submitted transaction             hash=0x2893b70483bf1791b550e5a93763058b0abf7c6d9e6201e07212dbc64d4764532 from: 0xFB48587362536C606d6e89f717Fsd229673246e6 nonce: 43 recipient: 0x7C60662d63536e89f717F9673sd22246F6eB4858 value: 100,000,000,000,000,000
```

Other user actions have similar log messages that are displayed to the console.

## Common warnings

There are many warnings that can be emitted by Geth as part of its normal operation. However, some are 
asked about especially frequently on the [Geth Github](https://github.com/ethereum/go-ethereum) and 
[Discord](https://discord.gg/WHNkYDsAKU) channel.  

```sh
WARN [10-03|18:00:40.413] Unexpected trienode heal packet          peer=9f0e8fbf         reqid=6,915,308,639,612,522,441
```

The above is often seen and misinterpreted as a problem with snap sync. In reality, it indicates a request 
timeout that may be because I/O speed is low. It is usually not an issue, but if this message is seen very 
often over prolonged periods of time it might be rooted in a local connectivity or hardware issue.

```sh
WARN [10-03|13:10:26.441] Post-merge network, but no beacon client seen. Please launch one to follow the chain!
```

The above message is emitted when Geth is run without a consensus client on a post-merge proof-of-stake network. 
Since Ethereum moved to proof-of-stake Geth alone is not enough to follow the chain because the consensus logic 
is now implemented by a separate piece of software called a consensus client. This log message is displayed 
when the consensus client is missing. Read more about this on our 
[consensus clients](/docs/interface/consensus-clients.md) page.

```sh
WARN [10-03 |13:10:26.499] Beacon client online, but never received consensus updates. Please ensure your beacon client is operational to follow the chain!
```

The message above indicates that a consensus client is present but not working correctly. The most likely 
reason for this is that the client is not yet in sync. Waiting for the consensus client to sync should 
solve the issue.

```sh
WARN [10-03 | 13:15:56.543] Dropping unsynced node during sync    id = e2fdc0d92d70953 conn = ...
```
This message indicates that a peer is being dropped because it is not fully synced. This is normal - the necessary data will be requested from an alternative peer instead.

## Summary

There are a wide range of log messages that are emitted while Geth is running. The level of detail in the 
logs can be configured using the `verbosity` flag at startup. This page has outlined some of the common 
messages users can expect to see when Geth is run with default verbosity, without attempting to be comprehensive. 
For more, please see the [Geth Github](https://github.com/ethereum/go-ethereum) and [Discord](https://discord.gg/WHNkYDsAKU).
