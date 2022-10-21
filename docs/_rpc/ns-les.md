---
title: les Namespace
sort_key: C
---

The `les` API allows you to manage LES server settings, including client parameters and payment settings for prioritized clients. It also provides functions to query checkpoint information in both server and client mode.

* TOC
{:toc}

### les_serverInfo

Get information about currently connected and total/individual allowed connection capacity.

| Client  | Method invocation                                           |
|:--------|-------------------------------------------------------------|
| Go      | `les.ServerInfo() map[string]interface{}`                   |
| Console | `les.serverInfo()`                                          |
| RPC     | `{"method": "les_serverInfo", "params": []}`                |

#### Example

```javascript
> les.serverInfo
{
  freeClientCapacity: 16000,
  maximumCapacity: 1600000,
  minimumCapacity: 16000,
  priorityConnectedCapacity: 180000,
  totalCapacity: 1600000,
  totalConnectedCapacity: 180000
}
```

### les_clientInfo

Get individual client information (connection, balance, pricing) on the specified list of clients or for all connected clients if the ID list is empty.

| Client  | Method invocation                                                         |
|:--------|---------------------------------------------------------------------------|
| Go      | `les.ClientInfo(ids []enode.ID) map[enode.ID]map[string]interface{}`      |
| Console | `les.clientInfo([id, ...])`                                               |
| RPC     | `{"method": "les_clientInfo", "params": [[id, ...]]}`                     |

#### Example

```javascript
> les.clientInfo([])
{
  37078bf8ea160a2b3d129bb4f3a930ce002356f83b820f467a07c1fe291531ea: {
    capacity: 16000,
    connectionTime: 11225.335901136,
    isConnected: true,
    pricing/balance: 998266395881,
    pricing/balanceMeta: "",
    pricing/negBalance: 501657912857,
    priority: true
  },
  6a47fe7bb23fd335df52ef1690f37ab44265a537b1d18eb616a3e77f898d9e77: {
    capacity: 100000,
    connectionTime: 9874.839293082,
    isConnected: true,
    pricing/balance: 2908840710198,
    pricing/balanceMeta: "qwerty",
    pricing/negBalance: 206242704507,
    priority: true
  },
  740c78f7d914e5c763731bc751b513fc2388ffa0b47db080ded3e8b305e68c75: {
    capacity: 16000,
    connectionTime: 3089.286712188,
    isConnected: true,
    pricing/balance: 998266400174,
    pricing/balanceMeta: "",
    pricing/negBalance: 55135348863,
    priority: true
  },
  9985ade55b515f79f64274bf2ae440ca8c433cfb0f283fb6010bf46f796b2a3b: {
    capacity: 16000,
    connectionTime: 11479.335479545,
    isConnected: true,
    pricing/balance: 998266452203,
    pricing/balanceMeta: "",
    pricing/negBalance: 564116425655,
    priority: true
  },
  ce65ada2c3e17d6da00cec0b3cc4c8ed5e74428b60f42fa287eaaec8cca62544: {
    capacity: 16000,
    connectionTime: 7095.794385419,
    isConnected: true,
    pricing/balance: 998266448492,
    pricing/balanceMeta: "",
    pricing/negBalance: 214617753229,
    priority: true
  },
  e1495ceb6db842f3ee66428d4bb7f4a124b2b17111dae35d141c3d568b869ef1: {
    capacity: 16000,
    connectionTime: 8614.018237937,
    isConnected: true,
    pricing/balance: 998266391796,
    pricing/balanceMeta: "",
    pricing/negBalance: 185964891797,
    priority: true
  }
}
```

### les_priorityClientInfo

Get individual client information on clients with a positive balance in the specified ID range, `start` included, `stop` excluded. If `stop` is zero then results are returned until the last existing balance entry. `maxCount` limits the number of returned results. If the count limit is reached but there are more IDs in the range then the first missing ID is included in the result with an empty value assigned to it.

| Client  | Method invocation                                                                                  |
|:--------|----------------------------------------------------------------------------------------------------|
| Go      | `les.PriorityClientInfo(start, stop enode.ID, maxCount int) map[enode.ID]map[string]interface{}`   |
| Console | `les.priorityClientInfo(id, id, number)`                                                           |
| RPC     | `{"method": "les_priorityClientInfo", "params": [id, id, number]}`                                 |

#### Example

```javascript
> les.priorityClientInfo("0x0000000000000000000000000000000000000000000000000000000000000000", "0x0000000000000000000000000000000000000000000000000000000000000000", 100)
{
  37078bf8ea160a2b3d129bb4f3a930ce002356f83b820f467a07c1fe291531ea: {
    capacity: 16000,
    connectionTime: 11128.247204027,
    isConnected: true,
    pricing/balance: 999819815030,
    pricing/balanceMeta: "",
    pricing/negBalance: 501657912857,
    priority: true
  },
  6a47fe7bb23fd335df52ef1690f37ab44265a537b1d18eb616a3e77f898d9e77: {
    capacity: 100000,
    connectionTime: 9777.750592047,
    isConnected: true,
    pricing/balance: 2918549830576,
    pricing/balanceMeta: "qwerty",
    pricing/negBalance: 206242704507,
    priority: true
  },
  740c78f7d914e5c763731bc751b513fc2388ffa0b47db080ded3e8b305e68c75: {
    capacity: 16000,
    connectionTime: 2992.198001116,
    isConnected: true,
    pricing/balance: 999819845102,
    pricing/balanceMeta: "",
    pricing/negBalance: 55135348863,
    priority: true
  },
  9985ade55b515f79f64274bf2ae440ca8c433cfb0f283fb6010bf46f796b2a3b: {
    capacity: 16000,
    connectionTime: 11382.246766963,
    isConnected: true,
    pricing/balance: 999819871598,
    pricing/balanceMeta: "",
    pricing/negBalance: 564116425655,
    priority: true
  },
  ce65ada2c3e17d6da00cec0b3cc4c8ed5e74428b60f42fa287eaaec8cca62544: {
    capacity: 16000,
    connectionTime: 6998.705683407,
    isConnected: true,
    pricing/balance: 999819882177,
    pricing/balanceMeta: "",
    pricing/negBalance: 214617753229,
    priority: true
  },
  e1495ceb6db842f3ee66428d4bb7f4a124b2b17111dae35d141c3d568b869ef1: {
    capacity: 16000,
    connectionTime: 8516.929533901,
    isConnected: true,
    pricing/balance: 999819891640,
    pricing/balanceMeta: "",
    pricing/negBalance: 185964891797,
    priority: true
  }
}

> les.priorityClientInfo("0x4000000000000000000000000000000000000000000000000000000000000000", "0xe000000000000000000000000000000000000000000000000000000000000000", 2)
{
  6a47fe7bb23fd335df52ef1690f37ab44265a537b1d18eb616a3e77f898d9e77: {
    capacity: 100000,
    connectionTime: 9842.11178361,
    isConnected: true,
    pricing/balance: 2912113588853,
    pricing/balanceMeta: "qwerty",
    pricing/negBalance: 206242704507,
    priority: true
  },
  740c78f7d914e5c763731bc751b513fc2388ffa0b47db080ded3e8b305e68c75: {
    capacity: 16000,
    connectionTime: 3056.559199029,
    isConnected: true,
    pricing/balance: 998790060237,
    pricing/balanceMeta: "",
    pricing/negBalance: 55135348863,
    priority: true
  },
  9985ade55b515f79f64274bf2ae440ca8c433cfb0f283fb6010bf46f796b2a3b: {}
}
```

### les_addBalance

Add signed value to the token balance of the specified client and update its `meta` tag. The balance cannot go below zero or over `2^^63-1`. The balance values before and after the update are returned. The `meta` tag can be used to store a sequence number or reference to the last processed incoming payment, token expiration info, balance in other currencies or any application-specific additional information.

| Client  | Method invocation                                                                 |
|:--------|-----------------------------------------------------------------------------------|
| Go      | `les.AddBalance(id enode.ID, value int64, meta string) ([2]uint64, error)}`       |
| Console | `les.addBalance(id, number, string)`                                              |
| RPC     | `{"method": "les_addBalance", "params": [id, number, string]}`                    |

#### Example

```javascript
> les.addBalance("0x6a47fe7bb23fd335df52ef1690f37ab44265a537b1d18eb616a3e77f898d9e77", 1000000000, "qwerty")
[968379616, 1968379616]
```

### les_setClientParams

Set capacity and pricing factors for the specified list of connected clients or for all connected clients if the ID list is empty.

| Client  | Method invocation                                                                 |
|:--------|-----------------------------------------------------------------------------------|
| Go      | `les.SetClientParams(ids []enode.ID, params map[string]interface{}) error`        |
| Console | `les.setClientParams([id, ...], {string: value, ...})`                            |
| RPC     | `{"method": "les_setClientParams", "params": [[id, ...], {string: value, ...}]}`  |

#### Example

```javascript
> les.setClientParams(["0x6a47fe7bb23fd335df52ef1690f37ab44265a537b1d18eb616a3e77f898d9e77"], {
	"capacity": 100000,
	"pricing/timeFactor": 0,
	"pricing/capacityFactor": 1000000000,
	"pricing/requestCostFactor": 1000000000,
	"pricing/negative/timeFactor": 0,
	"pricing/negative/capacityFactor": 1000000000,
	"pricing/negative/requestCostFactor": 1000000000,
})
null
```

### les_setDefaultParams

Set default pricing factors for subsequently connected clients.

| Client  | Method invocation                                                                 |
|:--------|-----------------------------------------------------------------------------------|
| Go      | `les.SetDefaultParams(params map[string]interface{}) error`                       |
| Console | `les.setDefaultParams({string: value, ...})`                                      |
| RPC     | `{"method": "les_setDefaultParams", "params": [{string: value, ...}]}`            |

#### Example

```javascript
> les.setDefaultParams({
	"pricing/timeFactor": 0,
	"pricing/capacityFactor": 1000000000,
	"pricing/requestCostFactor": 1000000000,
	"pricing/negative/timeFactor": 0,
	"pricing/negative/capacityFactor": 1000000000,
	"pricing/negative/requestCostFactor": 1000000000,
})
null
```

### les_latestCheckpoint

Get the index and hashes of the latest known checkpoint.

| Client  | Method invocation                                           |
|:--------|-------------------------------------------------------------|
| Go      | `les.LatestCheckpoint() ([4]string, error)`                 |
| Console | `les.latestCheckpoint()`                                    |
| RPC     | `{"method": "les_latestCheckpoint", "params": []}`          |

#### Example

```javascript
> les.latestCheckpoint
["0x110", "0x6eedf8142d06730b391bfcbd32e9bbc369ab0b46ae226287ed5b29505a376164", "0x191bb2265a69c30201a616ae0d65a4ceb5937c2f0c94b125ff55343d707463e5", "0xf58409088a5cb2425350a59d854d546d37b1e7bef8bbf6afee7fd15f943d626a"]
```

### les_getCheckpoint

Get checkpoint hashes by index.

| Client  | Method invocation                                           |
|:--------|-------------------------------------------------------------|
| Go      | `les.GetCheckpoint(index uint64) ([3]string, error)`        |
| Console | `les.getCheckpoint(number)`                                 |
| RPC     | `{"method": "les_getCheckpoint", "params": [number]}`       |

#### Example

```javascript
> les.getCheckpoint(256)
["0x93eb4af0b224b1097e09181c2e51536fe0a3bf3bb4d93e9a69cab9eb3e28c75f", "0x0eb055e384cf58bc72ca20ca5e2b37d8d4115dce80ab4a19b72b776502c4dd5b", "0xda6c02f7c51f9ecc3eca71331a7eaad724e5a0f4f906ce9251a2f59e3115dd6a"]
```

### les_getCheckpointContractAddress

Get the address of the checkpoint oracle contract.

| Client  | Method invocation                                                 |
|:--------|-------------------------------------------------------------------|
| Go      | `les.GetCheckpointContractAddress() (string, error)`              |
| Console | `les.checkpointContractAddress()`                                 |
| RPC     | `{"method": "les_getCheckpointContractAddress", "params": []}`    |

#### Example

```javascript
> les.checkpointContractAddress
"0x9a9070028361F7AAbeB3f2F2Dc07F82C4a98A02a"
```

