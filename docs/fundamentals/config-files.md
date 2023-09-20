---
title: Config files
description: How to configure geth using config files
---

There are many flags and commands that can be provided to Geth on startup to influence how your node will behave. It is often convenient to configure these options in a file rather than typing them out on the command line every time you start your node. This can be done using a simple shell script to start Geth.

There are also other configuration options that are not accessible from the command line but can be adjusted by providing Geth with a config file. This gives access to lower level configuration that influences how some of Geth's internal components behave.

## Shell scripts

The benefit of writing a shell script for starting a Geth node is that it is more easily repeatable and you don't have to remember lots of syntax for making a node behave in a certain way. This is especially useful for running multiple nodes with their own specific configurations.

To create a shell script, save the Geth startup commands in a shell file, prepended with `#!/bin/bash`. The contents of the file might look like this:

```sh
#! /bin/bash
./geth --sepolia --datadir sepoliadata --authrpc.addr localhost --authrpc.port 8551, 8545 --authrpc.vhosts localhost --authrpc.jwtsecret sepoliadata/jwtsecret --http --http.api eth,net --metrics.expensive --metric.addr 127.0.0.1 --metrics.port 6060
```

Save the file as (e.g.) `start-geth.sh`. Then make the file executable using

```sh
chmod +x start-geth.sh
```

Now you can start Geth using this shell script instead of having to create the startup configuration from scratch each time:

```sh
./start-geth.sh
```

## Config files

It is also possible to tweak the deeper configuration via a config file. The config file is more complex than a shell script but it can touch parts of the internal configuration structure of Geth that are not accessible through the command line interface.

The config file should be a `.toml` file. A convenient way to create a config file is to get Geth to create one for you and use it as a template. To do this, use the `dumpconfig` command, saving the result to a `.toml` file. Note that you also need to explicitly provide the `network_id` on the command line for the public testnets such as Sepolia or Geoerli:

```sh
./geth --sepolia dumpconfig > geth-config.toml
```

You can change the values in this file and then pass it to Geth on startup so that the node is configured exactly as you want it. To override an option specified in the configuration file, specify the same option on the command line.

To run Geth with the configuration defined in `geth-config.toml`, pass the config file path to `--config`. The `network_id` is not persisted from the config file; it has to be explicitly defined on the command line on startup, for example:

```sh
geth --sepolia --config geth-config.toml
```

### Config file example

The config file created using `dumpconfig` contains the following information (this example is for the Sepolia testnet - Mainnet and other network configurations will be slightly different):

```toml
[Eth]
NetworkId = 11155111
SyncMode = "snap"
EthDiscoveryURLs = ["enrtree://AKA3AM6LPBYEUDMVNU3BSVQJ5AD45Y7YPOHJLEF6W26QOE4VTUDPE@all.sepolia.ethdisco.net"]
SnapDiscoveryURLs = ["enrtree://AKA3AM6LPBYEUDMVNU3BSVQJ5AD45Y7YPOHJLEF6W26QOE4VTUDPE@all.sepolia.ethdisco.net"]
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
FilterLogCacheSize = 32
EnablePreimageRecording = false
RPCGasCap = 50000000
RPCEVMTimeout = 5000000000
RPCTxFeeCap = 1e+00

[Eth.Miner]
GasFloor = 0
GasCeil = 30000000
GasPrice = 1000000000
Recommit = 3000000000

[Eth.Ethash]
CacheDir = "ethash"
CachesInMem = 2
CachesOnDisk = 3
CachesLockMmap = false
DatasetDir = "/home/user/.ethash"
DatasetsInMem = 1
DatasetsOnDisk = 2
DatasetsLockMmap = false
PowMode = 0
NotifyFull = false

[Eth.TxPool]
Locals = []
NoLocals = false
Journal = "transactions.rlp"
Rejournal = 3600000000000
PriceLimit = 1
PriceBump = 10
AccountSlots = 16
GlobalSlots = 5120
AccountQueue = 64
GlobalQueue = 1024
Lifetime = 10800000000000

[Eth.GPO]
Blocks = 20
Percentile = 60
MaxHeaderHistory = 1024
MaxBlockHistory = 1024
MaxPrice = 500000000000
IgnorePrice = 2

[Node]
DataDir = "sepoliadata"
IPCPath = "geth.ipc"
HTTPHost = ""
HTTPPort = 8545
HTTPVirtualHosts = ["localhost"]
HTTPModules = ["net", "web3", "eth"]
AuthAddr = "localhost"
AuthPort = 8551
AuthVirtualHosts = ["localhost"]
WSHost = ""
WSPort = 8546
WSModules = ["net", "web3", "eth"]
GraphQLVirtualHosts = ["localhost"]

[Node.P2P]
MaxPeers = 50
NoDiscovery = false
BootstrapNodes = ["enode://9246d00bc8fd1742e5ad2428b80fc4dc45d786283e05ef6edbd9002cbc335d40998444732fbe921cb88e1d2c73d1b1de53bae6a2237996e9bfe14f871baf7066@18.168.182.86:30303", "enode://ec66ddcf1a974950bd4c782789a7e04f8aa7110a72569b6e65fcd51e937e74eed303b1ea734e4d19cfaec9fbff9b6ee65bf31dcb50ba79acce9dd63a6aca61c7@52.14.151.177:30303"]
BootstrapNodesV5 = ["enr:-KG4QOtcP9X1FbIMOe17QNMKqDxCpm14jcX5tiOE4_TyMrFqbmhPZHK_ZPG2Gxb1GE2xdtodOfx9-cgvNtxnRyHEmC0ghGV0aDKQ9aX9QgAAAAD__________4JpZIJ2NIJpcIQDE8KdiXNlY3AyNTZrMaEDhpehBDbZjM_L9ek699Y7vhUJ-eAdMyQW_Fil522Y0fODdGNwgiMog3VkcIIjKA", "enr:-KG4QDyytgmE4f7AnvW-ZaUOIi9i79qX4JwjRAiXBZCU65wOfBu-3Nb5I7b_Rmg3KCOcZM_C3y5pg7EBU5XGrcLTduQEhGV0aDKQ9aX9QgAAAAD__________4JpZIJ2NIJpcIQ2_DUbiXNlY3AyNTZrMaEDKnz_-ps3UUOfHWVYaskI5kWYO_vtYMGYCQRAR3gHDouDdGNwgiMog3VkcIIjKA", "enr:-Ku4QImhMc1z8yCiNJ1TyUxdcfNucje3BGwEHzodEZUan8PherEo4sF7pPHPSIB1NNuSg5fZy7qFsjmUKs2ea1Whi0EBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQOVphkDqal4QzPMksc5wnpuC3gvSC8AfbFOnZY_On34wIN1ZHCCIyg", "enr:-Ku4QP2xDnEtUXIjzJ_DhlCRN9SN99RYQPJL92TMlSv7U5C1YnYLjwOQHgZIUXw6c-BvRg2Yc2QsZxxoS_pPRVe0yK8Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQMeFF5GrS7UZpAH2Ly84aLK-TyvH-dRo0JM1i8yygH50YN1ZHCCJxA", "enr:-Ku4QPp9z1W4tAO8Ber_NQierYaOStqhDqQdOPY3bB3jDgkjcbk6YrEnVYIiCBbTxuar3CzS528d2iE7TdJsrL-dEKoBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQMw5fqqkw2hHC4F5HZZDPsNmPdB1Gi8JPQK7pRc9XHh-oN1ZHCCKvg", "enr:-IS4QLkKqDMy_ExrpOEWa59NiClemOnor-krjp4qoeZwIw2QduPC-q7Kz4u1IOWf3DDbdxqQIgC4fejavBOuUPy-HE4BgmlkgnY0gmlwhCLzAHqJc2VjcDI1NmsxoQLQSJfEAHZApkm5edTCZ_4qps_1k_ub2CxHFxi-gr2JMIN1ZHCCIyg", "enr:-IS4QDAyibHCzYZmIYZCjXwU9BqpotWmv2BsFlIq1V31BwDDMJPFEbox1ijT5c2Ou3kvieOKejxuaCqIcjxBjJ_3j_cBgmlkgnY0gmlwhAMaHiCJc2VjcDI1NmsxoQJIdpj_foZ02MXz4It8xKD7yUHTBx7lVFn3oeRP21KRV4N1ZHCCIyg", "enr:-Ku4QHqVeJ8PPICcWk1vSn_XcSkjOkNiTg6Fmii5j6vUQgvzMc9L1goFnLKgXqBJspJjIsB91LTOleFmyWWrFVATGngBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC1MD8qAAAAAP__________gmlkgnY0gmlwhAMRHkWJc2VjcDI1NmsxoQKLVXFOhp2uX6jeT0DvvDpPcU8FWMjQdR4wMuORMhpX24N1ZHCCIyg", "enr:-Ku4QG-2_Md3sZIAUebGYT6g0SMskIml77l6yR-M_JXc-UdNHCmHQeOiMLbylPejyJsdAPsTHJyjJB2sYGDLe0dn8uYBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC1MD8qAAAAAP__________gmlkgnY0gmlwhBLY-NyJc2VjcDI1NmsxoQORcM6e19T1T9gi7jxEZjk_sjVLGFscUNqAY9obgZaxbIN1ZHCCIyg", "enr:-Ku4QPn5eVhcoF1opaFEvg1b6JNFD2rqVkHQ8HApOKK61OIcIXD127bKWgAtbwI7pnxx6cDyk_nI88TrZKQaGMZj0q0Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC1MD8qAAAAAP__________gmlkgnY0gmlwhDayLMaJc2VjcDI1NmsxoQK2sBOLGcUb4AwuYzFuAVCaNHA-dy24UuEKkeFNgCVCsIN1ZHCCIyg", "enr:-Ku4QEWzdnVtXc2Q0ZVigfCGggOVB2Vc1ZCPEc6j21NIFLODSJbvNaef1g4PxhPwl_3kax86YPheFUSLXPRs98vvYsoBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC1MD8qAAAAAP__________gmlkgnY0gmlwhDZBrP2Jc2VjcDI1NmsxoQM6jr8Rb1ktLEsVcKAPa08wCsKUmvoQ8khiOl_SLozf9IN1ZHCCIyg"]
StaticNodes = []
TrustedNodes = []
ListenAddr = ":30303"
DiscAddr = ""
EnableMsgEvents = false

[Node.HTTPTimeouts]
ReadTimeout = 30000000000
ReadHeaderTimeout = 30000000000
WriteTimeout = 30000000000
IdleTimeout = 120000000000

[Metrics]
HTTP = "127.0.0.1"
Port = 6060
InfluxDBEndpoint = "http://localhost:8086"
InfluxDBDatabase = "geth"
InfluxDBUsername = "test"
InfluxDBPassword = "test"
InfluxDBTags = "host=localhost"
InfluxDBToken = "test"
InfluxDBBucket = "geth"
InfluxDBOrganization = "geth"
```
