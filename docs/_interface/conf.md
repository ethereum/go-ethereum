---
title: The Configuration File
sort_key: B
---

Geth can use a configuration file to specify various parameters, instead of using command line arguments. To get a configuration file
with the default, run:

```sh
geth dumpconfig > configFile
```

To use a configuration file, run:

```sh
geth --config configFile
```

The configuration file uses the [TOML syntax](https://en.wikipedia.org/wiki/TOML). The section names mostly correspond to [the names
of packages in the Geth source code](https://pkg.go.dev/github.com/ethereum/go-ethereum#section-directories).


## Sample Configuration File

### Eth

This package is responsible for running the Ethereum protocol. 
[The configuration object is documented here](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/eth/ethconfig)

```toml
[Eth]
NetworkId = 1
SyncMode = "fast"
EthDiscoveryURLs = ["enrtree://AKA3AM6LPBYEUDMVNU3BSVQJ5AD45Y7YPOHJLEF6W26QOE4VTUDPE@all.mainnet.ethdisco.net"]
SnapDiscoveryURLs = ["enrtree://AKA3AM6LPBYEUDMVNU3BSVQJ5AD45Y7YPOHJLEF6W26QOE4VTUDPE@all.mainnet.ethdisco.net"]
NoPruning = false
NoPrefetch = false
TxLookupLimit = 2350000
LightPeers = 100
UltraLightFraction = 75
DatabaseCache = 512
DatabaseFreezer = ""
TrieCleanCache = 154
TrieCleanCacheJournal = "triecache"
TrieCleanCacheRejournal = 3600000000000
TrieDirtyCache = 256
TrieTimeout = 3600000000000
SnapshotCache = 102
Preimages = false
EnablePreimageRecording = false
EWASMInterpreter = ""
EVMInterpreter = ""
RPCGasCap = 25000000
RPCTxFeeCap = 1e+00
```

### Eth.Miner

This package implements mining, the creation of new Ethereum blocks for profit. 
[The configuration object is documented here](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/miner#Config)

```toml
[Eth.Miner]
GasFloor = 8000000
GasCeil = 8000000
GasPrice = 1000000000
Recommit = 3000000000
Noverify = false
```

### Eth.Ethash

This package implements the PoW (proof of work) protocol. 
[The configuration object is documented here](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/consensus/ethash#Config)

```toml
[Eth.Ethash]
CacheDir = "ethash"
CachesInMem = 2
CachesOnDisk = 3
CachesLockMmap = false
DatasetDir = "/home/qbzzt1/.ethash"
DatasetsInMem = 1
DatasetsOnDisk = 2
DatasetsLockMmap = false
PowMode = 0
NotifyFull = false
```

### Eth.TxPool

This package handles the transaction pool from which transactions are chosen for new blocks. 
[The configuration object is documented here](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/core#DefaultTxPoolConfig)


```toml
[Eth.TxPool]
Locals = []
NoLocals = false
Journal = "transactions.rlp"
Rejournal = 3600000000000
PriceLimit = 1
PriceBump = 10
AccountSlots = 16
GlobalSlots = 4096
AccountQueue = 64
GlobalQueue = 1024
Lifetime = 10800000000000
```

### Eth.GPO

This is the Gas Price Oracle. [The configuration object](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/eth/ethconfig#FullNodeGPO).

```toml
[Eth.GPO]
Blocks = 20
Percentile = 60
MaxPrice = 500000000000
```

### Node

This package is used for the Remote Procedure Calls (RPCs) used to communicate between notes. [Config](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/node#Config)

```toml
[Node]
DataDir = "/home/qbzzt1/.ethereum"
IPCPath = "geth.ipc"
HTTPHost = ""
HTTPPort = 8545
HTTPVirtualHosts = ["localhost"]
HTTPModules = ["net", "web3", "eth"]
WSHost = ""
WSPort = 8546
WSModules = ["net", "web3", "eth"]
GraphQLVirtualHosts = ["localhost"]
```

# Node.P2P

[Peer to peer networking](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/p2p#Config)

```toml
[Node.P2P]
MaxPeers = 50
NoDiscovery = false
BootstrapNodes = ["enode://d860a01f9722d78051619d1e2351aba3f43f943f6f00718d1b9baa4101932a1f5011f16bb2b1bb35db20d6fe28fa0bf09636d26a87d31de9ec6203eeedb1f666@18.138.108.67:30303", "enode://22a8232c3abc76a16ae9d6c3b164f98775fe226f0917b0ca871128a74a8e9630b458460865bab457221f1d448dd9791d24c4e5d88786180ac185df813a68d4de@3.209.45.79:30303", "enode://ca6de62fce278f96aea6ec5a2daadb877e51651247cb96ee310a318def462913b653963c155a0ef6c7d50048bba6e6cea881130857413d9f50a621546b590758@34.255.23.113:30303", "enode://279944d8dcd428dffaa7436f25ca0ca43ae19e7bcf94a8fb7d1641651f92d121e972ac2e8f381414b80cc8e5555811c2ec6e1a99bb009b3f53c4c69923e11bd8@35.158.244.151:30303", "enode://8499da03c47d637b20eee24eec3c356c9a2e6148d6fe25ca195c7949ab8ec2c03e3556126b0d7ed644675e78c4318b08691b7b57de10e5f0d40d05b09238fa0a@52.187.207.27:30303", "enode://103858bdb88756c71f15e9b5e09b56dc1be52f0a5021d46301dbbfb7e130029cc9d0d6f73f693bc29b665770fff7da4d34f3c6379fe12721b5d7a0bcb5ca1fc1@191.234.162.198:30303", "enode://715171f50508aba88aecd1250af392a45a330af91d7b90701c436b618c86aaa1589c9184561907bebbb56439b8f8787bc01f49a7c77276c58c1b09822d75e8e8@52.231.165.108:30303", "enode://5d6d7cd20d6da4bb83a1d28cadb5d409b64edf314c0335df658c1a54e32c7c4a7ab7823d57c39b6a757556e68ff1df17c748b698544a55cb488b52479a92b60f@104.42.217.25:30303"]
BootstrapNodesV5 = ["enr:-KG4QOtcP9X1FbIMOe17QNMKqDxCpm14jcX5tiOE4_TyMrFqbmhPZHK_ZPG2Gxb1GE2xdtodOfx9-cgvNtxnRyHEmC0ghGV0aDKQ9aX9QgAAAAD__________4JpZIJ2NIJpcIQDE8KdiXNlY3AyNTZrMaEDhpehBDbZjM_L9ek699Y7vhUJ-eAdMyQW_Fil522Y0fODdGNwgiMog3VkcIIjKA", "enr:-KG4QDyytgmE4f7AnvW-ZaUOIi9i79qX4JwjRAiXBZCU65wOfBu-3Nb5I7b_Rmg3KCOcZM_C3y5pg7EBU5XGrcLTduQEhGV0aDKQ9aX9QgAAAAD__________4JpZIJ2NIJpcIQ2_DUbiXNlY3AyNTZrMaEDKnz_-ps3UUOfHWVYaskI5kWYO_vtYMGYCQRAR3gHDouDdGNwgiMog3VkcIIjKA", "enr:-Ku4QImhMc1z8yCiNJ1TyUxdcfNucje3BGwEHzodEZUan8PherEo4sF7pPHPSIB1NNuSg5fZy7qFsjmUKs2ea1Whi0EBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQOVphkDqal4QzPMksc5wnpuC3gvSC8AfbFOnZY_On34wIN1ZHCCIyg", "enr:-Ku4QP2xDnEtUXIjzJ_DhlCRN9SN99RYQPJL92TMlSv7U5C1YnYLjwOQHgZIUXw6c-BvRg2Yc2QsZxxoS_pPRVe0yK8Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQMeFF5GrS7UZpAH2Ly84aLK-TyvH-dRo0JM1i8yygH50YN1ZHCCJxA", "enr:-Ku4QPp9z1W4tAO8Ber_NQierYaOStqhDqQdOPY3bB3jDgkjcbk6YrEnVYIiCBbTxuar3CzS528d2iE7TdJsrL-dEKoBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQMw5fqqkw2hHC4F5HZZDPsNmPdB1Gi8JPQK7pRc9XHh-oN1ZHCCKvg", "enr:-IS4QLkKqDMy_ExrpOEWa59NiClemOnor-krjp4qoeZwIw2QduPC-q7Kz4u1IOWf3DDbdxqQIgC4fejavBOuUPy-HE4BgmlkgnY0gmlwhCLzAHqJc2VjcDI1NmsxoQLQSJfEAHZApkm5edTCZ_4qps_1k_ub2CxHFxi-gr2JMIN1ZHCCIyg", "enr:-IS4QDAyibHCzYZmIYZCjXwU9BqpotWmv2BsFlIq1V31BwDDMJPFEbox1ijT5c2Ou3kvieOKejxuaCqIcjxBjJ_3j_cBgmlkgnY0gmlwhAMaHiCJc2VjcDI1NmsxoQJIdpj_foZ02MXz4It8xKD7yUHTBx7lVFn3oeRP21KRV4N1ZHCCIyg", "enr:-Ku4QHqVeJ8PPICcWk1vSn_XcSkjOkNiTg6Fmii5j6vUQgvzMc9L1goFnLKgXqBJspJjIsB91LTOleFmyWWrFVATGngBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC1MD8qAAAAAP__________gmlkgnY0gmlwhAMRHkWJc2VjcDI1NmsxoQKLVXFOhp2uX6jeT0DvvDpPcU8FWMjQdR4wMuORMhpX24N1ZHCCIyg", "enr:-Ku4QG-2_Md3sZIAUebGYT6g0SMskIml77l6yR-M_JXc-UdNHCmHQeOiMLbylPejyJsdAPsTHJyjJB2sYGDLe0dn8uYBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC1MD8qAAAAAP__________gmlkgnY0gmlwhBLY-NyJc2VjcDI1NmsxoQORcM6e19T1T9gi7jxEZjk_sjVLGFscUNqAY9obgZaxbIN1ZHCCIyg", "enr:-Ku4QPn5eVhcoF1opaFEvg1b6JNFD2rqVkHQ8HApOKK61OIcIXD127bKWgAtbwI7pnxx6cDyk_nI88TrZKQaGMZj0q0Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC1MD8qAAAAAP__________gmlkgnY0gmlwhDayLMaJc2VjcDI1NmsxoQK2sBOLGcUb4AwuYzFuAVCaNHA-dy24UuEKkeFNgCVCsIN1ZHCCIyg", "enr:-Ku4QEWzdnVtXc2Q0ZVigfCGggOVB2Vc1ZCPEc6j21NIFLODSJbvNaef1g4PxhPwl_3kax86YPheFUSLXPRs98vvYsoBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC1MD8qAAAAAP__________gmlkgnY0gmlwhDZBrP2Jc2VjcDI1NmsxoQM6jr8Rb1ktLEsVcKAPa08wCsKUmvoQ8khiOl_SLozf9IN1ZHCCIyg"]
StaticNodes = []
TrustedNodes = []
ListenAddr = ":30303"
EnableMsgEvents = false
```

### Node.HTTPTimeouts

[Timeouts for the HTTP serve](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/rpc#HTTPTimeouts)

```toml
[Node.HTTPTimeouts]
ReadTimeout = 30000000000
WriteTimeout = 30000000000
IdleTimeout = 120000000000
```

### Metrics

[Metrics](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/metrics#Config)

```toml
[Metrics]
HTTP = "127.0.0.1"
Port = 6060
InfluxDBEndpoint = "http://localhost:8086"
InfluxDBDatabase = "geth"
InfluxDBUsername = "test"
InfluxDBPassword = "test"
InfluxDBTags = "host=localhost"
```
