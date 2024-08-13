# Server

The ```bor server``` command runs the Bor client.

## Options

- ```bor.devfakeauthor```: Run miner without validator set authorization [dev mode] : Use with '--bor.withoutheimdall' (default: false)

- ```bor.heimdall```: URL of Heimdall service (default: http://localhost:1317)

- ```bor.heimdallgRPC```: Address of Heimdall gRPC service

- ```bor.logs```: Enables bor log retrieval (default: false)

- ```bor.runheimdall```: Run Heimdall service as a child process (default: false)

- ```bor.runheimdallargs```: Arguments to pass to Heimdall service

- ```bor.useheimdallapp```: Use child heimdall process to fetch data, Only works when bor.runheimdall is true (default: false)

- ```bor.withoutheimdall```: Run without Heimdall service (for testing purpose) (default: false)

- ```chain```: Name of the chain to sync ('amoy', 'mumbai', 'mainnet') or path to a genesis file (default: mainnet)

- ```config```: Path to the TOML configuration file

- ```datadir```: Path of the data directory to store information

- ```datadir.ancient```: Data directory for ancient chain segments (default = inside chaindata)

- ```db.engine```: Backing database implementation to use ('leveldb' or 'pebble') (default: leveldb)

- ```dev```: Enable developer mode with ephemeral proof-of-authority network and a pre-funded developer account, mining enabled (default: false)

- ```dev.gaslimit```: Initial block gas limit (default: 11500000)

- ```dev.period```: Block period to use in developer mode (0 = mine only if transaction pending) (default: 0)

- ```disable-bor-wallet```: Disable the personal wallet endpoints (default: true)

- ```eth.requiredblocks```: Comma separated block number-to-hash mappings to require for peering (<number>=<hash>)

- ```ethstats```: Reporting URL of a ethstats service (nodename:secret@host:port)

- ```gcmode```: Blockchain garbage collection mode ("full", "archive") (default: full)

- ```gpo.blocks```: Number of recent blocks to check for gas prices (default: 20)

- ```gpo.ignoreprice```: Gas price below which gpo will ignore transactions (default: 25000000000)

- ```gpo.maxblockhistory```: Maximum block history of gasprice oracle (default: 1024)

- ```gpo.maxheaderhistory```: Maximum header history of gasprice oracle (default: 1024)

- ```gpo.maxprice```: Maximum gas price will be recommended by gpo (default: 500000000000)

- ```gpo.percentile```: Suggested gas price is the given percentile of a set of recent transaction gas prices (default: 60)

- ```grpc.addr```: Address and port to bind the GRPC server (default: :3131)

- ```identity```: Name/Identity of the node

- ```keystore```: Path of the directory where keystores are located

- ```log-level```: Log level for the server (trace|debug|info|warn|error|crit), will be deprecated soon. Use verbosity instead

- ```parallelevm.enable```: Enable Block STM (default: true)

- ```parallelevm.procs```: Number of speculative processes (cores) in Block STM (default: 8)

- ```pprof```: Enable the pprof HTTP server (default: false)

- ```pprof.addr```: pprof HTTP server listening interface (default: 127.0.0.1)

- ```pprof.blockprofilerate```: Turn on block profiling with the given rate (default: 0)

- ```pprof.memprofilerate```: Turn on memory profiling with the given rate (default: 524288)

- ```pprof.port```: pprof HTTP server listening port (default: 6060)

- ```rpc.batchlimit```: Maximum number of messages in a batch (use 0 for no limits) (default: 100)

- ```rpc.returndatalimit```: Maximum size (in bytes) a result of an rpc request could have (use 0 for no limits) (default: 100000)

- ```snapshot```: Enables the snapshot-database mode (default: true)

- ```state.scheme```: Scheme to use for storing ethereum state ('hash' or 'path') (default: hash)

- ```syncmode```: Blockchain sync mode (only "full" sync supported) (default: full)

- ```verbosity```: Logging verbosity for the server (5=trace|4=debug|3=info|2=warn|1=error|0=crit) (default: 3)

- ```vmdebug```: Record information useful for VM and contract debugging (default: false)

### Account Management Options

- ```allow-insecure-unlock```: Allow insecure account unlocking when account-related RPCs are exposed by http (default: false)

- ```lightkdf```: Reduce key-derivation RAM & CPU usage at some expense of KDF strength (default: false)

- ```password```: Password file to use for non-interactive password input

- ```unlock```: Comma separated list of accounts to unlock

### Cache Options

- ```cache```: Megabytes of memory allocated to internal caching (default: 1024)

- ```cache.blocklogs```: Size (in number of blocks) of the log cache for filtering (default: 32)

- ```cache.database```: Percentage of cache memory allowance to use for database io (default: 50)

- ```cache.gc```: Percentage of cache memory allowance to use for trie pruning (default: 25)

- ```cache.noprefetch```: Disable heuristic state prefetch during block import (less CPU and disk IO, more time waiting for data) (default: false)

- ```cache.preimages```: Enable recording the SHA3/keccak preimages of trie keys (default: false)

- ```cache.snapshot```: Percentage of cache memory allowance to use for snapshot caching (default: 10)

- ```cache.trie```: Percentage of cache memory allowance to use for trie caching (default: 15)

- ```cache.triesinmemory```: Number of block states (tries) to keep in memory (default: 128)

- ```fdlimit```: Raise the open file descriptor resource limit (default = system fd limit) (default: 0)

- ```txlookuplimit```: Number of recent blocks to maintain transactions index for (default: 2350000)

### ExtraDB Options

- ```leveldb.compaction.table.size```: LevelDB SSTable/file size in mebibytes (default: 2)

- ```leveldb.compaction.table.size.multiplier```: Multiplier on LevelDB SSTable/file size. Size for a level is determined by: `leveldb.compaction.table.size * (leveldb.compaction.table.size.multiplier ^ Level)` (default: 1)

- ```leveldb.compaction.total.size```: Total size in mebibytes of SSTables in a given LevelDB level. Size for a level is determined by: `leveldb.compaction.total.size * (leveldb.compaction.total.size.multiplier ^ Level)` (default: 10)

- ```leveldb.compaction.total.size.multiplier```: Multiplier on level size on LevelDB levels. Size for a level is determined by: `leveldb.compaction.total.size * (leveldb.compaction.total.size.multiplier ^ Level)` (default: 10)

### JsonRPC Options

- ```authrpc.addr```: Listening address for authenticated APIs (default: localhost)

- ```authrpc.jwtsecret```: Path to a JWT secret to use for authenticated RPC endpoints

- ```authrpc.port```: Listening port for authenticated APIs (default: 8551)

- ```authrpc.vhosts```: Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard. (default: localhost)

- ```graphql```: Enable GraphQL on the HTTP-RPC server. Note that GraphQL can only be started if an HTTP server is started as well. (default: false)

- ```graphql.corsdomain```: Comma separated list of domains from which to accept cross origin requests (browser enforced) (default: localhost)

- ```graphql.vhosts```: Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard. (default: localhost)

- ```http```: Enable the HTTP-RPC server (default: false)

- ```http.addr```: HTTP-RPC server listening interface (default: localhost)

- ```http.api```: API's offered over the HTTP-RPC interface (default: eth,net,web3,txpool,bor)

- ```http.corsdomain```: Comma separated list of domains from which to accept cross origin requests (browser enforced) (default: localhost)

- ```http.ep-requesttimeout```: Request Timeout for rpc execution pool for HTTP requests (default: 0s)

- ```http.ep-size```: Maximum size of workers to run in rpc execution pool for HTTP requests (default: 40)

- ```http.port```: HTTP-RPC server listening port (default: 8545)

- ```http.rpcprefix```: HTTP path path prefix on which JSON-RPC is served. Use '/' to serve on all paths.

- ```http.vhosts```: Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard. (default: localhost)

- ```ipcdisable```: Disable the IPC-RPC server (default: false)

- ```ipcpath```: Filename for IPC socket/pipe within the datadir (explicit paths escape it)

- ```rpc.allow-unprotected-txs```: Allow for unprotected (non EIP155 signed) transactions to be submitted via RPC (default: false)

- ```rpc.enabledeprecatedpersonal```: Enables the (deprecated) personal namespace (default: false)

- ```rpc.evmtimeout```: Sets a timeout used for eth_call (0=infinite) (default: 5s)

- ```rpc.gascap```: Sets a cap on gas that can be used in eth_call/estimateGas (0=infinite) (default: 50000000)

- ```rpc.txfeecap```: Sets a cap on transaction fee (in ether) that can be sent via the RPC APIs (0 = no cap) (default: 1)

- ```ws```: Enable the WS-RPC server (default: false)

- ```ws.addr```: WS-RPC server listening interface (default: localhost)

- ```ws.api```: API's offered over the WS-RPC interface (default: net,web3)

- ```ws.ep-requesttimeout```: Request Timeout for rpc execution pool for WS requests (default: 0s)

- ```ws.ep-size```: Maximum size of workers to run in rpc execution pool for WS requests (default: 40)

- ```ws.origins```: Origins from which to accept websockets requests (default: localhost)

- ```ws.port```: WS-RPC server listening port (default: 8546)

- ```ws.rpcprefix```: HTTP path prefix on which JSON-RPC is served. Use '/' to serve on all paths.

### Logging Options

- ```log.backtrace```: Request a stack trace at a specific logging statement (e.g. 'block.go:271')

- ```log.debug```: Prepends log messages with call-site location (file and line number) (default: false)

- ```log.enable-block-tracking```: Enables additional logging of information collected while tracking block lifecycle (default: false)

- ```log.json```: Format logs with JSON (default: false)

- ```vmodule```: Per-module verbosity: comma-separated list of <pattern>=<level> (e.g. eth/*=5,p2p=4)

### P2P Options

- ```bind```: Network binding address (default: 0.0.0.0)

- ```bootnodes```: Comma separated enode URLs for P2P discovery bootstrap

- ```maxpeers```: Maximum number of network peers (network disabled if set to 0) (default: 50)

- ```maxpendpeers```: Maximum number of pending connection attempts (default: 50)

- ```nat```: NAT port mapping mechanism (any|none|upnp|pmp|extip:<IP>) (default: any)

- ```netrestrict```: Restricts network communication to the given IP networks (CIDR masks)

- ```nodekey```:  P2P node key file

- ```nodekeyhex```: P2P node key as hex

- ```nodiscover```: Disables the peer discovery mechanism (manual peer addition) (default: false)

- ```port```: Network listening port (default: 30303)

- ```txarrivalwait```: Maximum duration to wait for a transaction before explicitly requesting it (default: 500ms)

- ```v4disc```: Enables the V4 discovery mechanism (default: true)

- ```v5disc```: Enables the experimental RLPx V5 (Topic Discovery) mechanism (default: false)

### Sealer Options

- ```mine```: Enable mining (default: false)

- ```miner.etherbase```: Public address for block mining rewards

- ```miner.extradata```: Block extra data set by the miner (default = client version)

- ```miner.gaslimit```: Target gas ceiling (gas limit) for mined blocks (default: 30000000)

- ```miner.gasprice```: Minimum gas price for mining a transaction (default: 25000000000)

- ```miner.interruptcommit```: Interrupt block commit when block creation time is passed (default: true)

- ```miner.recommit```: The time interval for miner to re-create mining work (default: 2m5s)

### Telemetry Options

- ```metrics```: Enable metrics collection and reporting (default: false)

- ```metrics.expensive```: Enable expensive metrics collection and reporting (default: false)

- ```metrics.influxdb```: Enable metrics export/push to an external InfluxDB database (v1) (default: false)

- ```metrics.influxdb.bucket```: InfluxDB bucket name to push reported metrics to (v2 only)

- ```metrics.influxdb.database```: InfluxDB database name to push reported metrics to

- ```metrics.influxdb.endpoint```: InfluxDB API endpoint to report metrics to

- ```metrics.influxdb.organization```: InfluxDB organization name (v2 only)

- ```metrics.influxdb.password```: Password to authorize access to the database

- ```metrics.influxdb.tags```: Comma-separated InfluxDB tags (key/values) attached to all measurements

- ```metrics.influxdb.token```: Token to authorize access to the database (v2 only)

- ```metrics.influxdb.username```: Username to authorize access to the database

- ```metrics.influxdbv2```: Enable metrics export/push to an external InfluxDB v2 database (default: false)

- ```metrics.opencollector-endpoint```: OpenCollector Endpoint (host:port)

- ```metrics.prometheus-addr```: Address for Prometheus Server (default: 127.0.0.1:7071)

### Transaction Pool Options

- ```txpool.accountqueue```: Maximum number of non-executable transaction slots permitted per account (default: 16)

- ```txpool.accountslots```: Minimum number of executable transaction slots guaranteed per account (default: 16)

- ```txpool.globalqueue```: Maximum number of non-executable transaction slots for all accounts (default: 32768)

- ```txpool.globalslots```: Maximum number of executable transaction slots for all accounts (default: 32768)

- ```txpool.journal```: Disk journal for local transaction to survive node restarts (default: transactions.rlp)

- ```txpool.lifetime```: Maximum amount of time non-executable transaction are queued (default: 3h0m0s)

- ```txpool.locals```: Comma separated accounts to treat as locals (no flush, priority inclusion)

- ```txpool.nolocals```: Disables price exemptions for locally submitted transactions (default: false)

- ```txpool.pricebump```: Price bump percentage to replace an already existing transaction (default: 10)

- ```txpool.pricelimit```: Minimum gas price limit to enforce for acceptance into the pool (default: 25000000000)

- ```txpool.rejournal```: Time interval to regenerate the local transaction journal (default: 1h0m0s)