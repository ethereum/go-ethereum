# Server

The ```bor server``` command runs the Bor client.

## Options

- ```chain```: Name of the chain to sync ('mumbai', 'mainnet') or path to a genesis file (default: mainnet)

- ```identity```: Name/Identity of the node

- ```verbosity```: Logging verbosity for the server (5=trace|4=debug|3=info|2=warn|1=error|0=crit), default = 3 (default: 3)

- ```log-level```: Log level for the server (trace|debug|info|warn|error|crit), will be deprecated soon. Use verbosity instead

- ```datadir```: Path of the data directory to store information

- ```vmdebug```: Record information useful for VM and contract debugging (default: false)

- ```datadir.ancient```: Data directory for ancient chain segments (default = inside chaindata)

- ```keystore```: Path of the directory where keystores are located

- ```config```: Path to the TOML configuration file

- ```syncmode```: Blockchain sync mode (only "full" sync supported) (default: full)

- ```gcmode```: Blockchain garbage collection mode ("full", "archive") (default: full)

- ```eth.requiredblocks```: Comma separated block number-to-hash mappings to require for peering (<number>=<hash>)

- ```snapshot```: Enables the snapshot-database mode (default: true)

- ```bor.logs```: Enables bor log retrieval (default: false)

- ```bor.heimdall```: URL of Heimdall service (default: http://localhost:1317)

- ```bor.withoutheimdall```: Run without Heimdall service (for testing purpose) (default: false)

- ```bor.devfakeauthor```: Run miner without validator set authorization [dev mode] : Use with '--bor.withoutheimdall' (default: false)

- ```bor.heimdallgRPC```: Address of Heimdall gRPC service

- ```bor.runheimdall```: Run Heimdall service as a child process (default: false)

- ```bor.runheimdallargs```: Arguments to pass to Heimdall service

- ```ethstats```: Reporting URL of a ethstats service (nodename:secret@host:port)

- ```gpo.blocks```: Number of recent blocks to check for gas prices (default: 20)

- ```gpo.percentile```: Suggested gas price is the given percentile of a set of recent transaction gas prices (default: 60)

- ```gpo.maxheaderhistory```: Maximum header history of gasprice oracle (default: 1024)

- ```gpo.maxblockhistory```: Maximum block history of gasprice oracle (default: 1024)

- ```gpo.maxprice```: Maximum gas price will be recommended by gpo (default: 5000000000000)

- ```gpo.ignoreprice```: Gas price below which gpo will ignore transactions (default: 2)

- ```disable-bor-wallet```: Disable the personal wallet endpoints (default: true)

- ```grpc.addr```: Address and port to bind the GRPC server (default: :3131)

- ```dev```: Enable developer mode with ephemeral proof-of-authority network and a pre-funded developer account, mining enabled (default: false)

- ```dev.period```: Block period to use in developer mode (0 = mine only if transaction pending) (default: 0)

- ```dev.gaslimit```: Initial block gas limit (default: 11500000)

- ```pprof```: Enable the pprof HTTP server (default: false)

- ```pprof.port```: pprof HTTP server listening port (default: 6060)

- ```pprof.addr```: pprof HTTP server listening interface (default: 127.0.0.1)

- ```pprof.memprofilerate```: Turn on memory profiling with the given rate (default: 524288)

- ```pprof.blockprofilerate```: Turn on block profiling with the given rate (default: 0)

### Account Management Options

- ```unlock```: Comma separated list of accounts to unlock

- ```password```: Password file to use for non-interactive password input

- ```allow-insecure-unlock```: Allow insecure account unlocking when account-related RPCs are exposed by http (default: false)

- ```lightkdf```: Reduce key-derivation RAM & CPU usage at some expense of KDF strength (default: false)

### Cache Options

- ```cache```: Megabytes of memory allocated to internal caching (default: 1024)

- ```cache.database```: Percentage of cache memory allowance to use for database io (default: 50)

- ```cache.trie```: Percentage of cache memory allowance to use for trie caching (default: 15)

- ```cache.trie.journal```: Disk journal directory for trie cache to survive node restarts (default: triecache)

- ```cache.trie.rejournal```: Time interval to regenerate the trie cache journal (default: 1h0m0s)

- ```cache.gc```: Percentage of cache memory allowance to use for trie pruning (default: 25)

- ```cache.snapshot```: Percentage of cache memory allowance to use for snapshot caching (default: 10)

- ```cache.noprefetch```: Disable heuristic state prefetch during block import (less CPU and disk IO, more time waiting for data) (default: false)

- ```cache.preimages```: Enable recording the SHA3/keccak preimages of trie keys (default: false)

- ```cache.triesinmemory```: Number of block states (tries) to keep in memory (default = 128) (default: 128)

- ```txlookuplimit```: Number of recent blocks to maintain transactions index for (default: 2350000)

- ```fdlimit```: Raise the open file descriptor resource limit (default = system fd limit) (default: 0)

### JsonRPC Options

- ```rpc.gascap```: Sets a cap on gas that can be used in eth_call/estimateGas (0=infinite) (default: 50000000)

- ```rpc.evmtimeout```: Sets a timeout used for eth_call (0=infinite) (default: 5s)

- ```rpc.txfeecap```: Sets a cap on transaction fee (in ether) that can be sent via the RPC APIs (0 = no cap) (default: 5)

- ```rpc.allow-unprotected-txs```: Allow for unprotected (non EIP155 signed) transactions to be submitted via RPC (default: false)

- ```ipcdisable```: Disable the IPC-RPC server (default: false)

- ```ipcpath```: Filename for IPC socket/pipe within the datadir (explicit paths escape it)

- ```authrpc.jwtsecret```: Path to a JWT secret to use for authenticated RPC endpoints

- ```authrpc.addr```: Listening address for authenticated APIs (default: localhost)

- ```authrpc.port```: Listening port for authenticated APIs (default: 8551)

- ```authrpc.vhosts```: Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard. (default: localhost)

- ```http.corsdomain```: Comma separated list of domains from which to accept cross origin requests (browser enforced) (default: localhost)

- ```http.vhosts```: Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard. (default: localhost)

- ```ws.origins```: Origins from which to accept websockets requests (default: localhost)

- ```graphql.corsdomain```: Comma separated list of domains from which to accept cross origin requests (browser enforced) (default: localhost)

- ```graphql.vhosts```: Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard. (default: localhost)

- ```http```: Enable the HTTP-RPC server (default: false)

- ```http.addr```: HTTP-RPC server listening interface (default: localhost)

- ```http.port```: HTTP-RPC server listening port (default: 8545)

- ```http.rpcprefix```: HTTP path path prefix on which JSON-RPC is served. Use '/' to serve on all paths.

- ```http.api```: API's offered over the HTTP-RPC interface (default: eth,net,web3,txpool,bor)

- ```ws```: Enable the WS-RPC server (default: false)

- ```ws.addr```: WS-RPC server listening interface (default: localhost)

- ```ws.port```: WS-RPC server listening port (default: 8546)

- ```ws.rpcprefix```: HTTP path prefix on which JSON-RPC is served. Use '/' to serve on all paths.

- ```ws.api```: API's offered over the WS-RPC interface (default: net,web3)

- ```graphql```: Enable GraphQL on the HTTP-RPC server. Note that GraphQL can only be started if an HTTP server is started as well. (default: false)

### Logging Options

- ```vmodule```: Per-module verbosity: comma-separated list of <pattern>=<level> (e.g. eth/*=5,p2p=4)

- ```log.json```: Format logs with JSON (default: false)

- ```log.backtrace```: Request a stack trace at a specific logging statement (e.g. 'block.go:271')

- ```log.debug```: Prepends log messages with call-site location (file and line number) (default: false)

### P2P Options

- ```bind```: Network binding address (default: 0.0.0.0)

- ```port```: Network listening port (default: 30303)

- ```bootnodes```: Comma separated enode URLs for P2P discovery bootstrap

- ```maxpeers```: Maximum number of network peers (network disabled if set to 0) (default: 50)

- ```maxpendpeers```: Maximum number of pending connection attempts (default: 50)

- ```nat```: NAT port mapping mechanism (any|none|upnp|pmp|extip:<IP>) (default: any)

- ```netrestrict```: Restricts network communication to the given IP networks (CIDR masks)

- ```nodekey```:  P2P node key file

- ```nodekeyhex```: P2P node key as hex

- ```nodiscover```: Disables the peer discovery mechanism (manual peer addition) (default: false)

- ```v5disc```: Enables the experimental RLPx V5 (Topic Discovery) mechanism (default: false)

### Sealer Options

- ```mine```: Enable mining (default: false)

- ```miner.etherbase```: Public address for block mining rewards

- ```miner.extradata```: Block extra data set by the miner (default = client version)

- ```miner.gaslimit```: Target gas ceiling (gas limit) for mined blocks (default: 30000000)

- ```miner.gasprice```: Minimum gas price for mining a transaction (default: 1000000000)

- ```miner.recommit```: The time interval for miner to re-create mining work (default: 2m5s)

### Telemetry Options

- ```metrics```: Enable metrics collection and reporting (default: false)

- ```metrics.expensive```: Enable expensive metrics collection and reporting (default: false)

- ```metrics.influxdb```: Enable metrics export/push to an external InfluxDB database (v1) (default: false)

- ```metrics.influxdb.endpoint```: InfluxDB API endpoint to report metrics to

- ```metrics.influxdb.database```: InfluxDB database name to push reported metrics to

- ```metrics.influxdb.username```: Username to authorize access to the database

- ```metrics.influxdb.password```: Password to authorize access to the database

- ```metrics.influxdb.tags```: Comma-separated InfluxDB tags (key/values) attached to all measurements

- ```metrics.prometheus-addr```: Address for Prometheus Server (default: 127.0.0.1:7071)

- ```metrics.opencollector-endpoint```: OpenCollector Endpoint (host:port) (default: 127.0.0.1:4317)

- ```metrics.influxdbv2```: Enable metrics export/push to an external InfluxDB v2 database (default: false)

- ```metrics.influxdb.token```: Token to authorize access to the database (v2 only)

- ```metrics.influxdb.bucket```: InfluxDB bucket name to push reported metrics to (v2 only)

- ```metrics.influxdb.organization```: InfluxDB organization name (v2 only)

### Transaction Pool Options

- ```txpool.locals```: Comma separated accounts to treat as locals (no flush, priority inclusion)

- ```txpool.nolocals```: Disables price exemptions for locally submitted transactions (default: false)

- ```txpool.journal```: Disk journal for local transaction to survive node restarts (default: transactions.rlp)

- ```txpool.rejournal```: Time interval to regenerate the local transaction journal (default: 1h0m0s)

- ```txpool.pricelimit```: Minimum gas price limit to enforce for acceptance into the pool (default: 1)

- ```txpool.pricebump```: Price bump percentage to replace an already existing transaction (default: 10)

- ```txpool.accountslots```: Minimum number of executable transaction slots guaranteed per account (default: 16)

- ```txpool.globalslots```: Maximum number of executable transaction slots for all accounts (default: 32768)

- ```txpool.accountqueue```: Maximum number of non-executable transaction slots permitted per account (default: 16)

- ```txpool.globalqueue```: Maximum number of non-executable transaction slots for all accounts (default: 32768)

- ```txpool.lifetime```: Maximum amount of time non-executable transaction are queued (default: 3h0m0s)