# Server

The ```bor server``` command runs the Bor client.

## Options

- ```chain```: Name of the chain to sync

- ```identity```: Name/Identity of the node

- ```log-level```: Set log level for the server

- ```datadir```: Path of the data directory to store information

- ```keystore```: Path of the directory to store keystores

- ```config```: File for the config file

- ```syncmode```: Blockchain sync mode ("fast", "full", or "snap")

- ```gcmode```: Blockchain garbage collection mode ("full", "archive")

- ```requiredblocks```: Comma separated block number-to-hash mappings to enforce (<number>=<hash>)

- ```snapshot```: Disables/Enables the snapshot-database mode (default = true)

- ```bor.heimdall```: URL of Heimdall service

- ```bor.withoutheimdall```: Run without Heimdall service (for testing purpose)

- ```ethstats```: Reporting URL of a ethstats service (nodename:secret@host:port)

- ```gpo.blocks```: Number of recent blocks to check for gas prices

- ```gpo.percentile```: Suggested gas price is the given percentile of a set of recent transaction gas prices

- ```gpo.maxprice```: Maximum gas price will be recommended by gpo

- ```gpo.ignoreprice```: Gas price below which gpo will ignore transactions

- ```disable-bor-wallet```: Disable the personal wallet endpoints

- ```grpc.addr```: Address and port to bind the GRPC server

- ```dev```: Enable developer mode with ephemeral proof-of-authority network and a pre-funded developer account, mining enabled

- ```dev.period```: Block period to use in developer mode (0 = mine only if transaction pending)

### Account Management Options

- ```unlock```: Comma separated list of accounts to unlock

- ```password```: Password file to use for non-interactive password input

- ```allow-insecure-unlock```: Allow insecure account unlocking when account-related RPCs are exposed by http

- ```lightkdf```: Reduce key-derivation RAM & CPU usage at some expense of KDF strength

### Cache Options

- ```cache```: Megabytes of memory allocated to internal caching (default = 4096 mainnet full node)

- ```cache.database```: Percentage of cache memory allowance to use for database io

- ```cache.trie```: Percentage of cache memory allowance to use for trie caching (default = 15% full mode, 30% archive mode)

- ```cache.trie.journal```: Disk journal directory for trie cache to survive node restarts

- ```cache.trie.rejournal```: Time interval to regenerate the trie cache journal

- ```cache.gc```: Percentage of cache memory allowance to use for trie pruning (default = 25% full mode, 0% archive mode)

- ```cache.snapshot```: Percentage of cache memory allowance to use for snapshot caching (default = 10% full mode, 20% archive mode)

- ```cache.noprefetch```: Disable heuristic state prefetch during block import (less CPU and disk IO, more time waiting for data)

- ```cache.preimages```: Enable recording the SHA3/keccak preimages of trie keys

- ```txlookuplimit```: Number of recent blocks to maintain transactions index for (default = about 56 days, 0 = entire chain)

### JsonRPC Options

- ```rpc.gascap```: Sets a cap on gas that can be used in eth_call/estimateGas (0=infinite)

- ```rpc.txfeecap```: Sets a cap on transaction fee (in ether) that can be sent via the RPC APIs (0 = no cap)

- ```ipcdisable```: Disable the IPC-RPC server

- ```ipcpath```: Filename for IPC socket/pipe within the datadir (explicit paths escape it)

- ```http.corsdomain```: Comma separated list of domains from which to accept cross origin requests (browser enforced)

- ```http.vhosts```: Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard.

- ```ws.corsdomain```: Comma separated list of domains from which to accept cross origin requests (browser enforced)

- ```ws.vhosts```: Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard.

- ```graphql.corsdomain```: Comma separated list of domains from which to accept cross origin requests (browser enforced)

- ```graphql.vhosts```: Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard.

- ```http```: Enable the HTTP-RPC server

- ```http.addr```: HTTP-RPC server listening interface

- ```http.port```: HTTP-RPC server listening port

- ```http.rpcprefix```: HTTP path path prefix on which JSON-RPC is served. Use '/' to serve on all paths.

- ```http.api```: API's offered over the HTTP-RPC interface

- ```ws```: Enable the WS-RPC server

- ```ws.addr```: WS-RPC server listening interface

- ```ws.port```: WS-RPC server listening port

- ```ws.rpcprefix```: HTTP path prefix on which JSON-RPC is served. Use '/' to serve on all paths.

- ```ws.api```: API's offered over the WS-RPC interface

- ```graphql```: Enable GraphQL on the HTTP-RPC server. Note that GraphQL can only be started if an HTTP server is started as well.

### P2P Options

- ```bind```: Network binding address

- ```port```: Network listening port

- ```bootnodes```: Comma separated enode URLs for P2P discovery bootstrap

- ```maxpeers```: Maximum number of network peers (network disabled if set to 0)

- ```maxpendpeers```: Maximum number of pending connection attempts (defaults used if set to 0)

- ```nat```: NAT port mapping mechanism (any|none|upnp|pmp|extip:<IP>)

- ```nodiscover```: Disables the peer discovery mechanism (manual peer addition)

- ```v5disc```: Enables the experimental RLPx V5 (Topic Discovery) mechanism

### Sealer Options

- ```mine```: Enable mining

- ```miner.etherbase```: Public address for block mining rewards (default = first account)

- ```miner.extradata```: Block extra data set by the miner (default = client version)

- ```miner.gaslimit```: Target gas ceiling for mined blocks

- ```miner.gasprice```: Minimum gas price for mining a transaction

### Telemetry Options

- ```metrics```: Enable metrics collection and reporting

- ```metrics.expensive```: Enable expensive metrics collection and reporting

- ```metrics.influxdb```: Enable metrics export/push to an external InfluxDB database (v1)

- ```metrics.influxdb.endpoint```: InfluxDB API endpoint to report metrics to

- ```metrics.influxdb.database```: InfluxDB database name to push reported metrics to

- ```metrics.influxdb.username```: Username to authorize access to the database

- ```metrics.influxdb.password```: Password to authorize access to the database

- ```metrics.influxdb.tags```: Comma-separated InfluxDB tags (key/values) attached to all measurements

- ```metrics.prometheus-addr```: Address for Prometheus Server

- ```metrics.opencollector-endpoint```: OpenCollector Endpoint (host:port)

- ```metrics.influxdbv2```: Enable metrics export/push to an external InfluxDB v2 database

- ```metrics.influxdb.token```: Token to authorize access to the database (v2 only)

- ```metrics.influxdb.bucket```: InfluxDB bucket name to push reported metrics to (v2 only)

- ```metrics.influxdb.organization```: InfluxDB organization name (v2 only)

### Transaction Pool Options

- ```txpool.locals```: Comma separated accounts to treat as locals (no flush, priority inclusion)

- ```txpool.nolocals```: Disables price exemptions for locally submitted transactions

- ```txpool.journal```: Disk journal for local transaction to survive node restarts

- ```txpool.rejournal```: Time interval to regenerate the local transaction journal

- ```txpool.pricelimit```: Minimum gas price limit to enforce for acceptance into the pool

- ```txpool.pricebump```: Price bump percentage to replace an already existing transaction

- ```txpool.accountslots```: Minimum number of executable transaction slots guaranteed per account

- ```txpool.globalslots```: Maximum number of executable transaction slots for all accounts

- ```txpool.accountqueue```: Maximum number of non-executable transaction slots permitted per account

- ```txpool.globalqueue```: Maximum number of non-executable transaction slots for all accounts

- ```txpool.lifetime```: Maximum amount of time non-executable transaction are queued