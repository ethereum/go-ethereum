---
title: The Configuration File
sort_key: B
---

Geth can use a configuration file to specify various parameters, instead of using command line arguments. To get a configuration file
with the defaults, run:

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
| NetworkId         | uint(64)     | The network ID for the network, which in most cases should be identical to the chain id. [Here is a list of possible values](https://chainlist.org/)            |
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
| UltraLightFraction     | int          | Percentage of trusted servers to accept an announcement   |
| UltraLightOnlyAnnounce | boolean      | Whether to only announce headers, or also serve them      |




#### Trie Settings

These settings apply to the cache used to manage [trie information](https://medium.com/@eiki1212/ethereum-state-trie-architecture-explained-a30237009d4e).

| Setting                 | Type          | Meaning                                                                                              |
| ----------------------- | ------------- | ---------------------------------------------------------------------------------------------------- |
| TrieCleanCache          | int           |
| TrieCleanCacheJournal   | string        | Disk journal directory for trie cache to survive node restarts |
| TrieCleanCacheRejournal | time.Duration | Time interval to regenerate the journal for clean cache        |
| TrieDirtyCache          | int           |
| TrieTimeout             | time.Duration |
| SnapshotCache           | int           |
| Preimages               | boolean       |


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
| Notify                  | string array  | HTTP URL list to be notified of new work packages (only useful in ethash) |
| NotifyFull              | boolean       | Notify with pending block headers instead of work packages                |           
| ExtraData               | Bytes         | Block extra data set by the miner                                         |
| GasFloor                | uint64        | Target gas floor for mined blocks                                         |
| GasCeil                 | uint64        | Target gas ceiling for mined blocks                                       |
| GasPrice                | \*big.Int     | Minimum gas price for mining a transaction                                |
| Recommit                | time.Duration | The time interval for miner to re-create mining work                      |
| Noverify                | boolean       | Disable remote mining solution verification(only useful in ethash)        |
  

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
| GlobalSlots              | uint64        | Maximum number of executable transaction slots for all accounts
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
| SmartCardDaemonPath     | string        | Path to the smartcard daemon's socket
| IPCPath                 | string        | Requested location to place the IPC endpoint.
| HTTPHost                | string        | Host interface on which to start the HTTP RPC server
| HTTPPort                | int           | TCP port for HTTP RPC server. Zero means to pick a port number randomly
| HTTPCors                | string array  | Cross-Origin Resource Sharing header to send to requesting clients (which may or may not obey it)
| HTTPVirtualHosts        | string array  | List of virtual hostnames which are allowed on incoming requests	
| HTTPModules             | string array  | List of API modules to expose via the HTTP RPC interface
| HTTPPathPrefix          | string        | Path prefix on which http-rpc is to be served.
| WSHost                  | string        | Host interface on which to start the websocket RPC server
| WSPort                  | int           | TCP port for websocker RPC server. Zero means to pick a port number randomly
| WSPathPrefix            | string        | Path prefix on which ws-rpc is to be served.
| WSOrigins               | string array  | List of domain to accept websocket requests from
| WSModules               | string array  | List of API modules to expose via the websocket RPC interface.
| WSExpose                | boolean       | Expose all API modules via the WebSocket RPC interface rather than just the public ones.  **Only for trusted networks**
| GraphQLCors             | string array  | Cross-Origin Resource Sharing header to send to requesting clients (which may or may not obey it)
| GraphQLVirtualHosts     | string array  | List of virtual hostnames which are allowed on incoming requests.
| AllowUnprotectedTxs     | boolean       | Allow non [EIP-155 protected transactions](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md) to be sent over RPC
	
  

### Node.P2P Settings

These are peer to peer network settings. [The configuration object is documented here](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/p2p#Config)


| Settings                | Type          | Meaning                                                                                              |
| ----------------------- | ------------- | ---------------------------------------------------------------------------------------------------- |
| PrivateKey              | [PrivateKey](https://pkg.go.dev/crypto/ecdsa#PrivateKey) | The private key for the node |
| MaxPeers                | int           | Maximum number of peers that can be connected
| MaxPendingPeers         | int           | Maximum number of peers that can be pending in the handshake phase, counted separately for inbound and outbound connections.
| DialRatio               | int           | Ratio of inbound to dialed connections. Defaults to 3 (one third of connections can be dialed)
| NoDiscovery             | boolean       | Disable the peer discovery mechanism, useful for protocol debugging (manual topology).
| DiscoveryV5             | boolean       | Whether the new topic-discovery based V5 discovery protocol should be started or not
| Name                    | string        | Node name of this server.
| BootstrapNodes          | [Node array](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/p2p/enode#Node) | Known nodes, used to establish connectivity with the rest of the network.
| BootstrapNodesV5        | [Node array](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/p2p/enode#Node) | Known nodes, used to establish connectivity with the rest of the network using the V5 discovery protocol
| StaticNodes             | [Node array](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/p2p/enode#Node) | Pre-configured connections which are always  maintained and re-connected on disconnects.
| TrustedNodes            | [Node array](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/p2p/enode#Node) | Pre-configured connections which are always
 allowed to connect, even above the peer limit
| NetRestrict             | [Netlist](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/p2p/netutil#Netlist) | If set to a non-nil value, only hosts that match one of the IP networks are allowed to connect
| NodeDatabase            | string        | Path to the database containing the previously seen live nodes in the network
| Protocols               | [Protocol array](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/p2p#Protocol) | Protocols supported by the server
| ListenAddr              | string        | If non-nil address, the server will listen for incoming connections. If port is zero, the OS picks a port
| NAT                     | [Interface](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/p2p/nat#Interface) |  If set to a non-nil value, the given NAT port mapper is used to make the listening port available to the Internet
| Dialer                  | [NodeDialer](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/p2p#NodeDialer) | If Dialer is set to a non-nil value, the given Dialer is used to dial outbound peer connections.
| NoDial                  | boolean      | If NoDial is true, the server will not dial any peers.
| EnableMsgEvents         | boolean      | If EnableMsgEvents is set then the server will emit PeerEvents whenever a message is sent to or received from a peer
| Logger                  | [Logger](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/log#Logger) | Custom logger to use with the p2p.Server.



### Node.HTTPTimeouts Settings

These are the timeouts for the HTTP server in various states. [The configuration object is documented 
here](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/rpc#HTTPTimeouts)


| Settings                | Type          | Meaning                                                                                              |
| ----------------------- | ------------- | ---------------------------------------------------------------------------------------------------- |
| ReadTimeout             | [Duration](https://pkg.go.dev/time#Duration) | Maximum duration for reading the entire request, including the body.
| WriteTimeout            | [Duration](https://pkg.go.dev/time#Duration) | Maximum duration before timing out
| IdleTimeout             | [Duration](https://pkg.go.dev/time#Duration) | Maximum amount of time to wait for the next request when keep-alives are enabled.



### Metrics Settings

Geth can log to an [InfluxDB](https://www.influxdata.com/products/influxdb/) for metrics.
[The configuration object is documented here](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.4/metrics#Config)


| Settings                | Type          | Meaning                                                                                              |
| ----------------------- | ------------- | ---------------------------------------------------------------------------------------------------- |
| Enabled                 | boolean 
| EnabledExpensive        | boolean 
| HTTP                    | string 
| Port                    | int    
| EnableInfluxDB          | boolean 
| InfluxDBEndpoint        | string 
| InfluxDBDatabase        | string 
| InfluxDBUsername        | string 
| InfluxDBPassword        | string 
| InfluxDBTags            | string 
