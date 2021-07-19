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


## Settings in the Configuration File

### Eth

This package is responsible for running the Ethereum protocol. 
[The configuration object is documented here](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/eth/ethconfig#Config)

#### Network Settings

| Setting           | Type         | Meaning                                                                                              |
| ----------------- | ------------ | ---------------------------------------------------------------------------------------------------- |
| NetworkId         | uint(64)     | The Chain ID for the network. [Here is a list of possible values](https://chainlist.org/)            |
| SyncMode          | string       | How to synchronize the client with the rest of the network. There are several values, [documented here](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/eth/downloader#SyncMode)                         |
| EthDiscoveryURLs  | string array | URLs to query for the list of nodes to access                          |
| SnapDiscoveryURLs | string array | URLs to query for the list of nodes to access for snap synchronization |


#### Database Settings

These settings apply to the database that keeps the state of the Ethereum chain (in memory and on disk)

| Setting            | Type         | Meaning                                                                                              |
| ------------------ | ------------ | ---------------------------------------------------------------------------------------------------- |
| NoPruning          | boolean      | Whether to disable state pruning and write everything to disk |
| NoPrefetch         | boolean      | Whether to disable prefetching and only load state on demand |
| TxLookupLimit      | uint(64)     | The maximum number of blocks from head whose tx indices are reserved. |
| Whitelist          | map          | Whitelist of required block number -> hash values to accept, usually not specified |
| SkipBcVersionCheck | boolean      |
| DatabaseHandles    | int          |
| DatabaseCache      | int          |
| DatabaseFreezer    | string       |


#### Light Client Settings

These settings apply when running `geth` as a [light node](https://ethereum.org/en/developers/docs/nodes-and-clients/#light-node).

| Setting           | Type         | Meaning                                                                                              |
| ----------------- | ------------ | ---------------------------------------------------------------------------------------------------- |
| LightServ         | int          | Maximum percentage of time allowed for serving light server requests |
| LightIngress      | int          | Incoming bandwidth limit for light servers    |
| LightEgress       | int          | Outgoing bandwidth limit for light servers |
| LightPeers        | int          | Maximum number of [Light Ethereum Sub-protocol (LES)](https://github.com/ethereum/devp2p/blob/master/caps/les.md) client peers |
| LightNoPrune      | boolean      | Whether to disable light chain pruning |
| LightNoSyncServe  | boolean      | Whether to serve light clients before syncing |
| SyncFromCheckpoint| boolean      | Whether to sync the header chain from the configured checkpoint |


### Ultra Light Client Settings

These settings apply to [ultra light clients](https://status.im/research/ulc_in_details.html).

| Setting                | Type         | Meaning                                                                                              |
| ---------------------- | ------------ | ---------------------------------------------------------------------------------------------------- |
| UltraLightServers      | string array | List of trusted ultra light servers                       |
|	UltraLightFraction     | int          | Percentage of trusted servers to accept an announcement   |
|	UltraLightOnlyAnnounce | boolean      | Whether to only announce headers, or also serve them      |




#### Trie Settings

These settings apply to the cache used to manage [trie information](https://medium.com/@eiki1212/ethereum-state-trie-architecture-explained-a30237009d4e).

| Setting                 | Type          | Meaning                                                                                              |
| ----------------------- | ------------- | ---------------------------------------------------------------------------------------------------- |
| TrieCleanCache          | int           |
| TrieCleanCacheJournal   | string        | Disk journal directory for trie cache to survive node restarts |
| TrieCleanCacheRejournal | time.Duration | Time interval to regenerate the journal for clean cache        |
| TrieDirtyCache          | int           |
| TrieTimeout             | time.Duration |
|	SnapshotCache           | int           |
|	Preimages               | boolean       |


### Misc. Settings

| Setting                 | Type          | Meaning                                                                                              |
| ----------------------- | ------------- | ---------------------------------------------------------------------------------------------------- |
| EnablePreimageRecording | boolean       | Enables tracking of SHA3 preimages in the VM   |
| DocRoot                 | string        |
| EWASMInterpreter        | string        | Type of the EWASM interpreter ("" for default) |
| EVMInterpreter          | string        | Type of the EVM interpreter ("" for default)   |
| RPCGasCap               | uint64        | Global gas cap for eth-call variants |
| RPCTxFeeCap             | float64       | Global transaction fee(price * gaslimit) cap for send-transction variants. The unit is ether. |
| Checkpoint              | [TrustedCheckpoint](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/params#TrustedCheckpoint) | Checkpoint is a hardcoded checkpoint which can be nil |
| CheckpointOracle        | [CheckpointOracleConfig](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/params#CheckpointOracleConfig)| CheckpointOracle is the configuration for checkpoint oracle |




### Eth.Miner Settings

This package implements mining, the creation of new Ethereum blocks for profit. 
[The configuration object is documented here](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/miner#Config)

| Setting                 | Type          | Meaning                                                                                              |
| ----------------------- | ------------- | ---------------------------------------------------------------------------------------------------- |
| Etherbase               | [Address](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/common#Address) | Public address for block mining rewards (default = first account)     |
|	Notify                  | string array  | HTTP URL list to be notified of new work packages (only useful in ethash) |
|	NotifyFull              | boolean       | Notify with pending block headers instead of work packages                |           
| ExtraData               | Bytes         | Block extra data set by the miner                                         |
|	GasFloor                | uint64        | Target gas floor for mined blocks                                         |
|	GasCeil                 | uint64        | Target gas ceiling for mined blocks                                       |
|	GasPrice                | \*big.Int     | Minimum gas price for mining a transaction                                |
|	Recommit                | time.Duration | The time interval for miner to re-create mining work                      |
|	Noverify                | boolean       | Disable remote mining solution verification(only useful in ethash)        |
  

### Eth.Ethash Settings

This package implements the PoW (proof of work) protocol. 
[The configuration object is documented here](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/consensus/ethash#Config)

| Settings                | Type          | Meaning                                                                                              |
| ----------------------- | ------------- | ---------------------------------------------------------------------------------------------------- |
| CacheDir                | string
| CachesInMem             | int
| CachesOnDisk            | int
| CachesLockMmap          | boolean
| DatasetDir              | string
| DatasetsInMem           | int
| DatasetsOnDisk          | int
| DatasetsLockMmap        | boolean
| PowMode                 | [Mode](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/consensus/ethash#Mode)
| NotifyFull              | boolean       | When true notifications sent by the remote sealer will be block header JSON objects instead of work package arrays.



### Eth.TxPool Settings

This package handles the transaction pool from which transactions are chosen for new blocks by a miner.
[The configuration object is documented here](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/core#TxPoolConfig)

| Settings                | Type          | Meaning                                                                                              |
| ----------------------- | ------------- | ---------------------------------------------------------------------------------------------------- |
| Locals                  | [Address](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/common#Address) array | Addresses that should be treated by default as local
| NoLocals                | boolean       | Whether local transaction handling should be disabled
|Journal                  | string        | Journal of local transactions to survive node restarts
| Rejournal               | [Duration](https://pkg.go.dev/time#Duration) | Time interval to regenerate the local transaction journal
| PriceLimit              | uint64        | Minimum gas price to enforce for acceptance into the pool
| PriceBump               | uint64        | Minimum price bump percentage to replace an already existing transaction (nonce)
| AccountSlots            | uint64        | Number of executable transaction slots guaranteed per account
| lobalSlots              | uint64        | Maximum number of executable transaction slots for all accounts
| AccountQueue            | uint64        | Maximum number of non-executable transaction slots permitted per account
| GlobalQueue             | uint64        | Maximum number of non-executable transaction slots for all accounts
| Lifetime                | [Duration](https://pkg.go.dev/time#Duration) | Maximum amount of time non-executable transaction are queued



### Eth.GPO Settings

This is the Gas Price Oracle. [The configuration object is documented here](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/eth/gasprice#Config).

| Settings                | Type          | Meaning                                                                                              |
| ----------------------- | ------------- | ---------------------------------------------------------------------------------------------------- |
| Blocks                  | int
| Percentile              | int
| Default                 | [\*big.Int](https://pkg.go.dev/math/big#Int)
| MaxPrice                | [\*big.Int](https://pkg.go.dev/math/big#Int)
| IgnorePrice             | [\*big.Int](https://pkg.go.dev/math/big#Int)



### Node Settings

This package is used for the settings of the node itself and the Remote Procedure Calls (RPC) it uses to communicate. 
[The configuration object is documented here](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/node#Config)

| Settings                | Type          | Meaning                                                                                              |
| ----------------------- | ------------- | ---------------------------------------------------------------------------------------------------- |
| Name                    | string        | The instance name of the node. It must not contain the / character and is used in the devp2p node identifier. 
| UserIdent               | string        | UserIdent, if set, is used as an additional component in the devp2p node identifier.
| Version                 | string        | The version number of the program. It is used in the devp2p node identifier.
| DataDir                 | string        | DataDir is the file system folder the node should use for any data storage requirements. 
| KeyStoreDir             | string        | File system folder that contains private keys. 
| ExternalSigner          | string        | External URI for a clef-type signer
| UseLightweightKDF       | boolean       | If true, lowers the memory and CPU requirements of the key store scrypt KDF at the expense of security.
| InsecureUnlockAllowed   | boolean       | Allow user to unlock accounts in unsafe http environment.
| NoUSB                   | boolean       | Disable hardware wallet monitoring and connectivity
| USB                     | boolean       | Enable hardware wallet monitoring and connectivity.
| SmartCardDaemonPath     | string        |	Path to the smartcard daemon's socket

	// IPCPath is the requested location to place the IPC endpoint. If the path is
	// a simple file name, it is placed inside the data directory (or on the root
	// pipe path on Windows), whereas if it's a resolvable path name (absolute or
	// relative), then that specific path is enforced. An empty path disables IPC.
	IPCPath string

	// HTTPHost is the host interface on which to start the HTTP RPC server. If this
	// field is empty, no HTTP API endpoint will be started.
	HTTPHost string

	// HTTPPort is the TCP port number on which to start the HTTP RPC server. The
	// default zero value is/ valid and will pick a port number randomly (useful
	// for ephemeral nodes).
	HTTPPort int `toml:",omitempty"`

	// HTTPCors is the Cross-Origin Resource Sharing header to send to requesting
	// clients. Please be aware that CORS is a browser enforced security, it's fully
	// useless for custom HTTP clients.
	HTTPCors []string `toml:",omitempty"`

	// HTTPVirtualHosts is the list of virtual hostnames which are allowed on incoming requests.
	// This is by default {'localhost'}. Using this prevents attacks like
	// DNS rebinding, which bypasses SOP by simply masquerading as being within the same
	// origin. These attacks do not utilize CORS, since they are not cross-domain.
	// By explicitly checking the Host-header, the server will not allow requests
	// made against the server with a malicious host domain.
	// Requests using ip address directly are not affected
	HTTPVirtualHosts []string `toml:",omitempty"`

	// HTTPModules is a list of API modules to expose via the HTTP RPC interface.
	// If the module list is empty, all RPC API endpoints designated public will be
	// exposed.
	HTTPModules []string

	// HTTPTimeouts allows for customization of the timeout values used by the HTTP RPC
	// interface.
	HTTPTimeouts rpc.HTTPTimeouts

	// HTTPPathPrefix specifies a path prefix on which http-rpc is to be served.
	HTTPPathPrefix string `toml:",omitempty"`

	// WSHost is the host interface on which to start the websocket RPC server. If
	// this field is empty, no websocket API endpoint will be started.
	WSHost string

	// WSPort is the TCP port number on which to start the websocket RPC server. The
	// default zero value is/ valid and will pick a port number randomly (useful for
	// ephemeral nodes).
	WSPort int `toml:",omitempty"`

	// WSPathPrefix specifies a path prefix on which ws-rpc is to be served.
	WSPathPrefix string `toml:",omitempty"`

	// WSOrigins is the list of domain to accept websocket requests from. Please be
	// aware that the server can only act upon the HTTP request the client sends and
	// cannot verify the validity of the request header.
	WSOrigins []string `toml:",omitempty"`

	// WSModules is a list of API modules to expose via the websocket RPC interface.
	// If the module list is empty, all RPC API endpoints designated public will be
	// exposed.
	WSModules []string

	// WSExposeAll exposes all API modules via the WebSocket RPC interface rather
	// than just the public ones.
	//
	// *WARNING* Only set this if the node is running in a trusted network, exposing
	// private APIs to untrusted users is a major security risk.
	WSExposeAll bool `toml:",omitempty"`

	// GraphQLCors is the Cross-Origin Resource Sharing header to send to requesting
	// clients. Please be aware that CORS is a browser enforced security, it's fully
	// useless for custom HTTP clients.
	GraphQLCors []string `toml:",omitempty"`

	// GraphQLVirtualHosts is the list of virtual hostnames which are allowed on incoming requests.
	// This is by default {'localhost'}. Using this prevents attacks like
	// DNS rebinding, which bypasses SOP by simply masquerading as being within the same
	// origin. These attacks do not utilize CORS, since they are not cross-domain.
	// By explicitly checking the Host-header, the server will not allow requests
	// made against the server with a malicious host domain.
	// Requests using ip address directly are not affected
	GraphQLVirtualHosts []string `toml:",omitempty"`

	
	// AllowUnprotectedTxs allows non EIP-155 protected transactions to be send over RPC.
	AllowUnprotectedTxs bool `toml:",omitempty"`
  
  

### Node.P2P Settings

These are peer to peer network settings. [The configuration object is documented here](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/p2p#Config)

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

### Node.HTTPTimeouts Settings

These are the timeouts for the HTTP server in various states. [The configuration object is documented 
here](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/rpc#HTTPTimeouts)

```toml
[Node.HTTPTimeouts]
ReadTimeout = 30000000000
WriteTimeout = 30000000000
IdleTimeout = 120000000000
```

### Metrics Settings

Geth can log to an [InfluxDB](https://www.influxdata.com/products/influxdb/) for metrics.
[The configuration object is documented here](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/metrics#Config)

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
