---
title: Command-line Options
sort_key: A
---

```
$ geth --help
NAME:
   geth - the go-ethereum command line interface

   Copyright 2013-2019 The go-ethereum Authors

USAGE:
   geth [options] command [command options] [arguments...]

VERSION:
   1.9.6-stable

COMMANDS:
   account                            Manage accounts
   attach                             Start an interactive JavaScript environment (connect to node)
   console                            Start an interactive JavaScript environment
   copydb                             Create a local chain from a target chaindata folder
   dump                               Dump a specific block from storage
   dumpconfig                         Show configuration values
   export                             Export blockchain into file
   export-preimages                   Export the preimage database into an RLP stream
   import                             Import a blockchain file
   import-preimages                   Import the preimage database from an RLP stream
   init                               Bootstrap and initialize a new genesis block
   inspect                            Inspect the storage size for each type of data in the database
   js                                 Execute the specified JavaScript files
   license                            Display license information
   makecache                          Generate ethash verification cache (for testing)
   makedag                            Generate ethash mining DAG (for testing)
   removedb                           Remove blockchain and state databases
   retesteth                          Launches geth in retesteth mode
   version                            Print version numbers
   wallet                             Manage Ethereum presale wallets
   help, h                            Shows a list of commands or help for one command

ETHEREUM OPTIONS:
  --config value                      TOML configuration file
  --datadir value                     Data directory for the databases and keystore (default: "~/Library/Ethereum")
  --datadir.ancient value             Data directory for ancient chain segments (default = inside chaindata)
  --keystore value                    Directory for the keystore (default = inside the datadir)
  --nousb                             Disables monitoring for and managing USB hardware wallets
  --pcscdpath value                   Path to the smartcard daemon (pcscd) socket file
  --networkid value                   Network identifier (integer, 1=Frontier, 2=Morden (disused), 3=Ropsten, 4=Rinkeby) (default: 1)
  --testnet                           Ropsten network: pre-configured proof-of-work test network
  --rinkeby                           Rinkeby network: pre-configured proof-of-authority test network
  --goerli                            Görli network: pre-configured proof-of-authority test network
  --syncmode value                    Blockchain sync mode ("fast", "full", or "light") (default: fast)
  --exitwhensynced                    Exits after block synchronisation completes
  --gcmode value                      Blockchain garbage collection mode ("full", "archive") (default: "full")
  --ethstats value                    Reporting URL of a ethstats service (nodename:secret@host:port)
  --identity value                    Custom node name
  --lightkdf                          Reduce key-derivation RAM & CPU usage at some expense of KDF strength
  --whitelist value                   Comma separated block number-to-hash mappings to enforce (<number>=<hash>)

LIGHT CLIENT OPTIONS:
  --light.serve value                 Maximum percentage of time allowed for serving LES requests (multi-threaded processing allows values over 100) (default: 0)
  --light.ingress value               Incoming bandwidth limit for serving light clients (kilobytes/sec, 0 = unlimited) (default: 0)
  --light.egress value                Outgoing bandwidth limit for serving light clients (kilobytes/sec, 0 = unlimited) (default: 0)
  --light.maxpeers value              Maximum number of light clients to serve, or light servers to attach to (default: 100)
  --ulc.servers value                 List of trusted ultra-light servers
  --ulc.fraction value                Minimum % of trusted ultra-light servers required to announce a new head (default: 75)
  --ulc.onlyannounce                  Ultra light server sends announcements only

DEVELOPER CHAIN OPTIONS:
  --dev                               Ephemeral proof-of-authority network with a pre-funded developer account, mining enabled
  --dev.period value                  Block period to use in developer mode (0 = mine only if transaction pending) (default: 0)

ETHASH OPTIONS:
  --ethash.cachedir value             Directory to store the ethash verification caches (default = inside the datadir)
  --ethash.cachesinmem value          Number of recent ethash caches to keep in memory (16MB each) (default: 2)
  --ethash.cachesondisk value         Number of recent ethash caches to keep on disk (16MB each) (default: 3)
  --ethash.dagdir value               Directory to store the ethash mining DAGs (default: "~/Library/Ethash")
  --ethash.dagsinmem value            Number of recent ethash mining DAGs to keep in memory (1+GB each) (default: 1)
  --ethash.dagsondisk value           Number of recent ethash mining DAGs to keep on disk (1+GB each) (default: 2)

TRANSACTION POOL OPTIONS:
  --txpool.locals value               Comma separated accounts to treat as locals (no flush, priority inclusion)
  --txpool.nolocals                   Disables price exemptions for locally submitted transactions
  --txpool.journal value              Disk journal for local transaction to survive node restarts (default: "transactions.rlp")
  --txpool.rejournal value            Time interval to regenerate the local transaction journal (default: 1h0m0s)
  --txpool.pricelimit value           Minimum gas price limit to enforce for acceptance into the pool (default: 1)
  --txpool.pricebump value            Price bump percentage to replace an already existing transaction (default: 10)
  --txpool.accountslots value         Minimum number of executable transaction slots guaranteed per account (default: 16)
  --txpool.globalslots value          Maximum number of executable transaction slots for all accounts (default: 4096)
  --txpool.accountqueue value         Maximum number of non-executable transaction slots permitted per account (default: 64)
  --txpool.globalqueue value          Maximum number of non-executable transaction slots for all accounts (default: 1024)
  --txpool.lifetime value             Maximum amount of time non-executable transaction are queued (default: 3h0m0s)

PERFORMANCE TUNING OPTIONS:
  --cache value                       Megabytes of memory allocated to internal caching (default = 4096 mainnet full node, 128 light mode) (default: 1024)
  --cache.database value              Percentage of cache memory allowance to use for database io (default: 50)
  --cache.trie value                  Percentage of cache memory allowance to use for trie caching (default = 25% full mode, 50% archive mode) (default: 25)
  --cache.gc value                    Percentage of cache memory allowance to use for trie pruning (default = 25% full mode, 0% archive mode) (default: 25)
  --cache.noprefetch                  Disable heuristic state prefetch during block import (less CPU and disk IO, more time waiting for data)

ACCOUNT OPTIONS:
  --unlock value                      Comma separated list of accounts to unlock
  --password value                    Password file to use for non-interactive password input
  --signer value                      External signer (url or path to ipc file)
  --allow-insecure-unlock             Allow insecure account unlocking when account-related RPCs are exposed by http

API AND CONSOLE OPTIONS:
  --ipcdisable                        Disable the IPC-RPC server
  --ipcpath value                     Filename for IPC socket/pipe within the datadir (explicit paths escape it)
  --rpc                               Enable the HTTP-RPC server
  --rpcaddr value                     HTTP-RPC server listening interface (default: "localhost")
  --rpcport value                     HTTP-RPC server listening port (default: 8545)
  --rpcapi value                      API's offered over the HTTP-RPC interface
  --rpc.gascap value                  Sets a cap on gas that can be used in eth_call/estimateGas (default: 0)
  --rpccorsdomain value               Comma separated list of domains from which to accept cross origin requests (browser enforced)
  --rpcvhosts value                   Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard. (default: "localhost")
  --ws                                Enable the WS-RPC server
  --wsaddr value                      WS-RPC server listening interface (default: "localhost")
  --wsport value                      WS-RPC server listening port (default: 8546)
  --wsapi value                       API's offered over the WS-RPC interface
  --wsorigins value                   Origins from which to accept websockets requests
  --graphql                           Enable the GraphQL server
  --graphql.addr value                GraphQL server listening interface (default: "localhost")
  --graphql.port value                GraphQL server listening port (default: 8547)
  --graphql.corsdomain value          Comma separated list of domains from which to accept cross origin requests (browser enforced)
  --graphql.vhosts value              Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard. (default: "localhost")
  --jspath loadScript                 JavaScript root path for loadScript (default: ".")
  --exec value                        Execute JavaScript statement
  --preload value                     Comma separated list of JavaScript files to preload into the console

NETWORKING OPTIONS:
  --bootnodes value                   Comma separated enode URLs for P2P discovery bootstrap (set v4+v5 instead for light servers)
  --bootnodesv4 value                 Comma separated enode URLs for P2P v4 discovery bootstrap (light server, full nodes)
  --bootnodesv5 value                 Comma separated enode URLs for P2P v5 discovery bootstrap (light server, light nodes)
  --port value                        Network listening port (default: 30303)
  --maxpeers value                    Maximum number of network peers (network disabled if set to 0) (default: 50)
  --maxpendpeers value                Maximum number of pending connection attempts (defaults used if set to 0) (default: 0)
  --nat value                         NAT port mapping mechanism (any|none|upnp|pmp|extip:<IP>) (default: "any")
  --nodiscover                        Disables the peer discovery mechanism (manual peer addition)
  --v5disc                            Enables the experimental RLPx V5 (Topic Discovery) mechanism
  --netrestrict value                 Restricts network communication to the given IP networks (CIDR masks)
  --nodekey value                     P2P node key file
  --nodekeyhex value                  P2P node key as hex (for testing)

MINER OPTIONS:
  --mine                              Enable mining
  --miner.threads value               Number of CPU threads to use for mining (default: 0)
  --miner.notify value                Comma separated HTTP URL list to notify of new work packages
  --miner.gasprice value              Minimum gas price for mining a transaction (default: 1000000000)
  --miner.gastarget value             Target gas floor for mined blocks (default: 8000000)
  --miner.gaslimit value              Target gas ceiling for mined blocks (default: 8000000)
  --miner.etherbase value             Public address for block mining rewards (default = first account) (default: "0")
  --miner.extradata value             Block extra data set by the miner (default = client version)
  --miner.recommit value              Time interval to recreate the block being mined (default: 3s)
  --miner.noverify                    Disable remote sealing verification

GAS PRICE ORACLE OPTIONS:
  --gpoblocks value                   Number of recent blocks to check for gas prices (default: 20)
  --gpopercentile value               Suggested gas price is the given percentile of a set of recent transaction gas prices (default: 60)

VIRTUAL MACHINE OPTIONS:
  --vmdebug                           Record information useful for VM and contract debugging
  --vm.evm value                      External EVM configuration (default = built-in interpreter)
  --vm.ewasm value                    External ewasm configuration (default = built-in interpreter)

LOGGING AND DEBUGGING OPTIONS:
  --fakepow                           Disables proof-of-work verification
  --nocompaction                      Disables db compaction after import
  --verbosity value                   Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail (default: 3)
  --vmodule value                     Per-module verbosity: comma-separated list of <pattern>=<level> (e.g. eth/*=5,p2p=4)
  --backtrace value                   Request a stack trace at a specific logging statement (e.g. "block.go:271")
  --debug                             Prepends log messages with call-site location (file and line number)
  --pprof                             Enable the pprof HTTP server
  --pprofaddr value                   pprof HTTP server listening interface (default: "127.0.0.1")
  --pprofport value                   pprof HTTP server listening port (default: 6060)
  --memprofilerate value              Turn on memory profiling with the given rate (default: 524288)
  --blockprofilerate value            Turn on block profiling with the given rate (default: 0)
  --cpuprofile value                  Write CPU profile to the given file
  --trace value                       Write execution trace to the given file

METRICS AND STATS OPTIONS:
  --metrics                           Enable metrics collection and reporting
  --metrics.expensive                 Enable expensive metrics collection and reporting
  --metrics.influxdb                  Enable metrics export/push to an external InfluxDB database
  --metrics.influxdb.endpoint value   InfluxDB API endpoint to report metrics to (default: "http://localhost:8086")
  --metrics.influxdb.database value   InfluxDB database name to push reported metrics to (default: "geth")
  --metrics.influxdb.username value   Username to authorize access to the database (default: "test")
  --metrics.influxdb.password value   Password to authorize access to the database (default: "test")
  --metrics.influxdb.tags value       Comma-separated InfluxDB tags (key/values) attached to all measurements (default: "host=localhost")

WHISPER (EXPERIMENTAL) OPTIONS:
  --shh                               Enable Whisper
  --shh.maxmessagesize value          Max message size accepted (default: 1048576)
  --shh.pow value                     Minimum POW accepted (default: 0.2)
  --shh.restrict-light                Restrict connection between two whisper light clients

DEPRECATED OPTIONS:
  --lightserv value                   Maximum percentage of time allowed for serving LES requests (deprecated, use --light.serve) (default: 0)
  --lightpeers value                  Maximum number of light clients to serve, or light servers to attach to  (deprecated, use --light.maxpeers) (default: 100)
  --minerthreads value                Number of CPU threads to use for mining (deprecated, use --miner.threads) (default: 0)
  --targetgaslimit value              Target gas floor for mined blocks (deprecated, use --miner.gastarget) (default: 8000000)
  --gasprice value                    Minimum gas price for mining a transaction (deprecated, use --miner.gasprice) (default: 1000000000)
  --etherbase value                   Public address for block mining rewards (default = first account, deprecated, use --miner.etherbase) (default: "0")
  --extradata value                   Block extra data set by the miner (default = client version, deprecated, use --miner.extradata)

MISC OPTIONS:
  --override.istanbul value           Manually specify Istanbul fork-block, overriding the bundled setting (default: 0)
  --help, -h                          show help


COPYRIGHT:
   Copyright 2013-2019 The go-ethereum Authors
```
