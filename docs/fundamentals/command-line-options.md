---
title: Command-line Options
description: A list of commands for Geth
---

Geth is primarily controlled using the command line. Geth is started using the `geth`
command. It is stopped by pressing `ctrl-c`.

You can configure Geth using command-line options (a.k.a. flags). Geth also has
sub-commands, which can be used to invoke functionality such as the console or blockchain
import/export.

The command-line help listing is reproduced below for your convenience. The same
information can be obtained at any time from your own Geth instance by running:

```sh
geth --help
```

## Commands {#commands}

```sh
NAME:
   geth - the go-ethereum command line interface

USAGE:
   geth [global options] command [command options] [arguments...]

VERSION:
   1.13.1-stable-3f40e65c

COMMANDS:
   account                Manage accounts
   attach                 Start an interactive JavaScript environment (connect to node)
   console                Start an interactive JavaScript environment
   db                     Low level database operations
   dump                   Dump a specific block from storage
   dumpconfig             Export configuration values in a TOML format
   dumpgenesis            Dumps genesis block JSON configuration to stdout
   export                 Export blockchain into file
   export-preimages       Export the preimage database into an RLP stream
   import                 Import a blockchain file
   import-preimages       Import the preimage database from an RLP stream
   init                   Bootstrap and initialize a new genesis block
   js                     (DEPRECATED) Execute the specified JavaScript files
   license                Display license information
   removedb               Remove blockchain and state databases
   show-deprecated-flags  Show flags that have been deprecated
   snapshot               A set of commands based on the snapshot
   verkle                 A set of experimental verkle tree management commands
   version                Print version numbers
   version-check          Checks (online) for known Geth security vulnerabilities
   wallet                 Manage Ethereum presale wallets
   help, h                Shows a list of commands or help for one command

GLOBAL OPTIONS:

    --log.rotate                        (default: false)                   ($GETH_LOG_ROTATE)
          Enables log file rotation

   ACCOUNT


    --allow-insecure-unlock             (default: false)                   ($GETH_ALLOW_INSECURE_UNLOCK)
          Allow insecure account unlocking when account-related RPCs are exposed by http

    --keystore value                                                       ($GETH_KEYSTORE)
          Directory for the keystore (default = inside the datadir)

    --lightkdf                          (default: false)                   ($GETH_LIGHTKDF)
          Reduce key-derivation RAM & CPU usage at some expense of KDF strength

    --password value                                                       ($GETH_PASSWORD)
          Password file to use for non-interactive password input

    --pcscdpath value                   (default: "/run/pcscd/pcscd.comm") ($GETH_PCSCDPATH)
          Path to the smartcard daemon (pcscd) socket file

    --signer value                                                         ($GETH_SIGNER)
          External signer (url or path to ipc file)

    --unlock value                                                         ($GETH_UNLOCK)
          Comma separated list of accounts to unlock

    --usb                               (default: false)                   ($GETH_USB)
          Enable monitoring and management of USB hardware wallets

   ALIASED (deprecated)


    --cache.trie.journal value                                             ($GETH_CACHE_TRIE_JOURNAL)
          Disk journal directory for trie cache to survive node restarts

    --cache.trie.rejournal value        (default: 0s)                      ($GETH_CACHE_TRIE_REJOURNAL)
          Time interval to regenerate the trie cache journal

    --nousb                             (default: false)                   ($GETH_NOUSB)
          Disables monitoring for and managing USB hardware wallets (deprecated)

    --txlookuplimit value               (default: 2350000)                 ($GETH_TXLOOKUPLIMIT)
          Number of recent blocks to maintain transactions index for (default = about one
          year, 0 = entire chain) (deprecated, use history.transactions instead)

    --v5disc                            (default: false)                   ($GETH_V5DISC)
          Enables the experimental RLPx V5 (Topic Discovery) mechanism (deprecated, use
          --discv5 instead)

    --whitelist value                                                      ($GETH_WHITELIST)
          Comma separated block number-to-hash mappings to enforce (<number>=<hash>)
          (deprecated in favor of --eth.requiredblocks)

   API AND CONSOLE


    --authrpc.addr value                (default: "localhost")             ($GETH_AUTHRPC_ADDR)
          Listening address for authenticated APIs

    --authrpc.jwtsecret value                                              ($GETH_AUTHRPC_JWTSECRET)
          Path to a JWT secret to use for authenticated RPC endpoints

    --authrpc.port value                (default: 8551)                    ($GETH_AUTHRPC_PORT)
          Listening port for authenticated APIs

    --authrpc.vhosts value              (default: "localhost")             ($GETH_AUTHRPC_VHOSTS)
          Comma separated list of virtual hostnames from which to accept requests (server
          enforced). Accepts '*' wildcard.

    --exec value                                                           ($GETH_EXEC)
          Execute JavaScript statement

    --graphql                           (default: false)                   ($GETH_GRAPHQL)
          Enable GraphQL on the HTTP-RPC server. Note that GraphQL can only be started if
          an HTTP server is started as well.

    --graphql.corsdomain value                                             ($GETH_GRAPHQL_CORSDOMAIN)
          Comma separated list of domains from which to accept cross origin requests
          (browser enforced)

    --graphql.vhosts value              (default: "localhost")             ($GETH_GRAPHQL_VHOSTS)
          Comma separated list of virtual hostnames from which to accept requests (server
          enforced). Accepts '*' wildcard.

    --header value, -H value
          Pass custom headers to the RPC server when using --remotedb or the geth attach
          console. This flag can be given multiple times.

    --http                              (default: false)                   ($GETH_HTTP)
          Enable the HTTP-RPC server

    --http.addr value                   (default: "localhost")             ($GETH_HTTP_ADDR)
          HTTP-RPC server listening interface

    --http.api value                                                       ($GETH_HTTP_API)
          API's offered over the HTTP-RPC interface

    --http.corsdomain value                                                ($GETH_HTTP_CORSDOMAIN)
          Comma separated list of domains from which to accept cross origin requests
          (browser enforced)

    --http.port value                   (default: 8545)                    ($GETH_HTTP_PORT)
          HTTP-RPC server listening port

    --http.rpcprefix value                                                 ($GETH_HTTP_RPCPREFIX)
          HTTP path path prefix on which JSON-RPC is served. Use '/' to serve on all
          paths.

    --http.vhosts value                 (default: "localhost")             ($GETH_HTTP_VHOSTS)
          Comma separated list of virtual hostnames from which to accept requests (server
          enforced). Accepts '*' wildcard.

    --ipcdisable                        (default: false)                   ($GETH_IPCDISABLE)
          Disable the IPC-RPC server

    --ipcpath value                                                        ($GETH_IPCPATH)
          Filename for IPC socket/pipe within the datadir (explicit paths escape it)

    --jspath value                      (default: .)                       ($GETH_JSPATH)
          JavaScript root path for `loadScript`

    --preload value                                                        ($GETH_PRELOAD)
          Comma separated list of JavaScript files to preload into the console

    --rpc.allow-unprotected-txs         (default: false)                   ($GETH_RPC_ALLOW_UNPROTECTED_TXS)
          Allow for unprotected (non EIP155 signed) transactions to be submitted via RPC

    --rpc.batch-request-limit value     (default: 1000)                    ($GETH_RPC_BATCH_REQUEST_LIMIT)
          Maximum number of requests in a batch

    --rpc.batch-response-max-size value (default: 25000000)                ($GETH_RPC_BATCH_RESPONSE_MAX_SIZE)
          Maximum number of bytes returned from a batched call

    --rpc.enabledeprecatedpersonal      (default: false)                   ($GETH_RPC_ENABLEDEPRECATEDPERSONAL)
          Enables the (deprecated) personal namespace

    --rpc.evmtimeout value              (default: 5s)                      ($GETH_RPC_EVMTIMEOUT)
          Sets a timeout used for eth_call (0=infinite)

    --rpc.gascap value                  (default: 50000000)                ($GETH_RPC_GASCAP)
          Sets a cap on gas that can be used in eth_call/estimateGas (0=infinite)

    --rpc.txfeecap value                (default: 1)
          Sets a cap on transaction fee (in ether) that can be sent via the RPC APIs (0 =
          no cap)

    --ws                                (default: false)                   ($GETH_WS)
          Enable the WS-RPC server

    --ws.addr value                     (default: "localhost")             ($GETH_WS_ADDR)
          WS-RPC server listening interface

    --ws.api value                                                         ($GETH_WS_API)
          API's offered over the WS-RPC interface

    --ws.origins value                                                     ($GETH_WS_ORIGINS)
          Origins from which to accept websockets requests

    --ws.port value                     (default: 8546)                    ($GETH_WS_PORT)
          WS-RPC server listening port

    --ws.rpcprefix value                                                   ($GETH_WS_RPCPREFIX)
          HTTP path prefix on which JSON-RPC is served. Use '/' to serve on all paths.

   DEVELOPER CHAIN


    --dev                               (default: false)                   ($GETH_DEV)
          Ephemeral proof-of-authority network with a pre-funded developer account, mining
          enabled

    --dev.gaslimit value                (default: 11500000)                ($GETH_DEV_GASLIMIT)
          Initial block gas limit

    --dev.period value                  (default: 0)                       ($GETH_DEV_PERIOD)
          Block period to use in developer mode (0 = mine only if transaction pending)

   ETHEREUM


    --bloomfilter.size value            (default: 2048)                    ($GETH_BLOOMFILTER_SIZE)
          Megabytes of memory allocated to bloom-filter for pruning

    --config value                                                         ($GETH_CONFIG)
          TOML configuration file

    --datadir value                     (default: /root/.ethereum)         ($GETH_DATADIR)
          Data directory for the databases and keystore

    --datadir.ancient value                                                ($GETH_DATADIR_ANCIENT)
          Root directory for ancient data (default = inside chaindata)

    --datadir.minfreedisk value                                            ($GETH_DATADIR_MINFREEDISK)
          Minimum free disk space in MB, once reached triggers auto shut down (default =
          --cache.gc converted to MB, 0 = disabled)

    --db.engine value                                                      ($GETH_DB_ENGINE)
          Backing database implementation to use ('pebble' or 'leveldb')

    --eth.requiredblocks value                                             ($GETH_ETH_REQUIREDBLOCKS)
          Comma separated block number-to-hash mappings to require for peering
          (<number>=<hash>)

    --exitwhensynced                    (default: false)                   ($GETH_EXITWHENSYNCED)
          Exits after block synchronisation completes

    --goerli                            (default: false)                   ($GETH_GOERLI)
          Görli network: pre-configured proof-of-authority test network

    --holesky                           (default: false)                   ($GETH_HOLESKY)
          Holesky network: pre-configured proof-of-stake test network

    --mainnet                           (default: false)                   ($GETH_MAINNET)
          Ethereum mainnet

    --networkid value                   (default: 1)                       ($GETH_NETWORKID)
          Explicitly set network id (integer)(For testnets: use --goerli, --sepolia,
          --holesky instead)

    --override.cancun value             (default: 0)                       ($GETH_OVERRIDE_CANCUN)
          Manually specify the Cancun fork timestamp, overriding the bundled setting

    --override.verkle value             (default: 0)                       ($GETH_OVERRIDE_VERKLE)
          Manually specify the Verkle fork timestamp, overriding the bundled setting

    --sepolia                           (default: false)                   ($GETH_SEPOLIA)
          Sepolia network: pre-configured proof-of-work test network

    --snapshot                          (default: true)                    ($GETH_SNAPSHOT)
          Enables snapshot-database mode (default = enable)

   GAS PRICE ORACLE


    --gpo.blocks value                  (default: 20)                      ($GETH_GPO_BLOCKS)
          Number of recent blocks to check for gas prices

    --gpo.ignoreprice value             (default: 2)
          Gas price below which gpo will ignore transactions

    --gpo.maxprice value                (default: 500000000000)
          Maximum transaction priority fee (or gasprice before London fork) to be
          recommended by gpo

    --gpo.percentile value              (default: 60)                      ($GETH_GPO_PERCENTILE)
          Suggested gas price is the given percentile of a set of recent transaction gas
          prices

   LIGHT CLIENT


    --light.egress value                (default: 0)                       ($GETH_LIGHT_EGRESS)
          Outgoing bandwidth limit for serving light clients (kilobytes/sec, 0 =
          unlimited)

    --light.ingress value               (default: 0)                       ($GETH_LIGHT_INGRESS)
          Incoming bandwidth limit for serving light clients (kilobytes/sec, 0 =
          unlimited)

    --light.maxpeers value              (default: 100)                     ($GETH_LIGHT_MAXPEERS)
          Maximum number of light clients to serve, or light servers to attach to

    --light.nopruning                   (default: false)                   ($GETH_LIGHT_NOPRUNING)
          Disable ancient light chain data pruning

    --light.nosyncserve                 (default: false)                   ($GETH_LIGHT_NOSYNCSERVE)
          Enables serving light clients before syncing

    --light.serve value                 (default: 0)                       ($GETH_LIGHT_SERVE)
          Maximum percentage of time allowed for serving LES requests (multi-threaded
          processing allows values over 100)

   LOGGING AND DEBUGGING


    --log.backtrace value                                                  ($GETH_LOG_BACKTRACE)
          Request a stack trace at a specific logging statement (e.g. "block.go:271")

    --log.compress                      (default: false)                   ($GETH_LOG_COMPRESS)
          Compress the log files

    --log.debug                         (default: false)                   ($GETH_LOG_DEBUG)
          Prepends log messages with call-site location (file and line number)

    --log.file value                                                       ($GETH_LOG_FILE)
          Write logs to a file

    --log.format value                                                     ($GETH_LOG_FORMAT)
          Log format to use (json|logfmt|terminal)

    --log.maxage value                  (default: 30)                      ($GETH_LOG_MAXAGE)
          Maximum number of days to retain a log file

    --log.maxbackups value              (default: 10)                      ($GETH_LOG_MAXBACKUPS)
          Maximum number of log files to retain

    --log.maxsize value                 (default: 100)                     ($GETH_LOG_MAXSIZE)
          Maximum size in MBs of a single log file

    --log.vmodule value                                                    ($GETH_LOG_VMODULE)
          Per-module verbosity: comma-separated list of <pattern>=<level> (e.g.
          eth/*=5,p2p=4)

    --nocompaction                      (default: false)                   ($GETH_NOCOMPACTION)
          Disables db compaction after import

    --pprof                             (default: false)                   ($GETH_PPROF)
          Enable the pprof HTTP server

    --pprof.addr value                  (default: "127.0.0.1")             ($GETH_PPROF_ADDR)
          pprof HTTP server listening interface

    --pprof.blockprofilerate value      (default: 0)                       ($GETH_PPROF_BLOCKPROFILERATE)
          Turn on block profiling with the given rate

    --pprof.cpuprofile value                                               ($GETH_PPROF_CPUPROFILE)
          Write CPU profile to the given file

    --pprof.memprofilerate value        (default: 524288)                  ($GETH_PPROF_MEMPROFILERATE)
          Turn on memory profiling with the given rate

    --pprof.port value                  (default: 6060)                    ($GETH_PPROF_PORT)
          pprof HTTP server listening port

    --remotedb value                                                       ($GETH_REMOTEDB)
          URL for remote database

    --trace value                                                          ($GETH_TRACE)
          Write execution trace to the given file

    --verbosity value                   (default: 3)                       ($GETH_VERBOSITY)
          Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail

   METRICS AND STATS


    --ethstats value                                                       ($GETH_ETHSTATS)
          Reporting URL of a ethstats service (nodename:secret@host:port)

    --metrics                           (default: false)                   ($GETH_METRICS)
          Enable metrics collection and reporting

    --metrics.addr value                                                   ($GETH_METRICS_ADDR)
          Enable stand-alone metrics HTTP server listening interface.

    --metrics.expensive                 (default: false)                   ($GETH_METRICS_EXPENSIVE)
          Enable expensive metrics collection and reporting

    --metrics.influxdb                  (default: false)                   ($GETH_METRICS_INFLUXDB)
          Enable metrics export/push to an external InfluxDB database

    --metrics.influxdb.bucket value     (default: "geth")                  ($GETH_METRICS_INFLUXDB_BUCKET)
          InfluxDB bucket name to push reported metrics to (v2 only)

    --metrics.influxdb.database value   (default: "geth")                  ($GETH_METRICS_INFLUXDB_DATABASE)
          InfluxDB database name to push reported metrics to

    --metrics.influxdb.endpoint value   (default: "http://localhost:8086") ($GETH_METRICS_INFLUXDB_ENDPOINT)
          InfluxDB API endpoint to report metrics to

    --metrics.influxdb.organization value (default: "geth")                  ($GETH_METRICS_INFLUXDB_ORGANIZATION)
          InfluxDB organization name (v2 only)

    --metrics.influxdb.password value   (default: "test")                  ($GETH_METRICS_INFLUXDB_PASSWORD)
          Password to authorize access to the database

    --metrics.influxdb.tags value       (default: "host=localhost")        ($GETH_METRICS_INFLUXDB_TAGS)
          Comma-separated InfluxDB tags (key/values) attached to all measurements

    --metrics.influxdb.token value      (default: "test")                  ($GETH_METRICS_INFLUXDB_TOKEN)
          Token to authorize access to the database (v2 only)

    --metrics.influxdb.username value   (default: "test")                  ($GETH_METRICS_INFLUXDB_USERNAME)
          Username to authorize access to the database

    --metrics.influxdbv2                (default: false)                   ($GETH_METRICS_INFLUXDBV2)
          Enable metrics export/push to an external InfluxDB v2 database

    --metrics.port value                (default: 6060)                    ($GETH_METRICS_PORT)
          Metrics HTTP server listening port.
          Please note that --metrics.addr must be set
          to start the server.

   MINER


    --mine                              (default: false)                   ($GETH_MINE)
          Enable mining

    --miner.etherbase value                                                ($GETH_MINER_ETHERBASE)
          0x prefixed public address for block mining rewards

    --miner.extradata value                                                ($GETH_MINER_EXTRADATA)
          Block extra data set by the miner (default = client version)

    --miner.gaslimit value              (default: 30000000)                ($GETH_MINER_GASLIMIT)
          Target gas ceiling for mined blocks

    --miner.gasprice value              (default: 0)                       ($GETH_MINER_GASPRICE)
          Minimum gas price for mining a transaction

    --miner.newpayload-timeout value    (default: 2s)                      ($GETH_MINER_NEWPAYLOAD_TIMEOUT)
          Specify the maximum time allowance for creating a new payload

    --miner.recommit value              (default: 2s)                      ($GETH_MINER_RECOMMIT)
          Time interval to recreate the block being mined

   MISC


    --help, -h                          (default: false)
          show help

    --synctarget value                                                     ($GETH_SYNCTARGET)
          File for containing the hex-encoded block-rlp as sync target(dev feature)

    --version, -v                       (default: false)
          print the version

   NETWORKING


    --bootnodes value                                                      ($GETH_BOOTNODES)
          Comma separated enode URLs for P2P discovery bootstrap

    --discovery.dns value                                                  ($GETH_DISCOVERY_DNS)
          Sets DNS discovery entry points (use "" to disable DNS)

    --discovery.port value              (default: 30303)                   ($GETH_DISCOVERY_PORT)
          Use a custom UDP port for P2P discovery

    --discovery.v4, --discv4            (default: true)                    ($GETH_DISCOVERY_V4)
          Enables the V4 discovery mechanism

    --discovery.v5, --discv5            (default: false)                   ($GETH_DISCOVERY_V5)
          Enables the experimental RLPx V5 (Topic Discovery) mechanism

    --identity value                                                       ($GETH_IDENTITY)
          Custom node name

    --maxpeers value                    (default: 50)                      ($GETH_MAXPEERS)
          Maximum number of network peers (network disabled if set to 0)

    --maxpendpeers value                (default: 0)                       ($GETH_MAXPENDPEERS)
          Maximum number of pending connection attempts (defaults used if set to 0)

    --nat value                         (default: "any")                   ($GETH_NAT)
          NAT port mapping mechanism (any|none|upnp|pmp|pmp:<IP>|extip:<IP>)

    --netrestrict value                                                    ($GETH_NETRESTRICT)
          Restricts network communication to the given IP networks (CIDR masks)

    --nodekey value                                                        ($GETH_NODEKEY)
          P2P node key file

    --nodekeyhex value                                                     ($GETH_NODEKEYHEX)
          P2P node key as hex (for testing)

    --nodiscover                        (default: false)                   ($GETH_NODISCOVER)
          Disables the peer discovery mechanism (manual peer addition)

    --port value                        (default: 30303)                   ($GETH_PORT)
          Network listening port

   PERFORMANCE TUNING


    --cache value                       (default: 1024)                    ($GETH_CACHE)
          Megabytes of memory allocated to internal caching (default = 4096 mainnet full
          node, 128 light mode)

    --cache.blocklogs value             (default: 32)                      ($GETH_CACHE_BLOCKLOGS)
          Size (in number of blocks) of the log cache for filtering

    --cache.database value              (default: 50)                      ($GETH_CACHE_DATABASE)
          Percentage of cache memory allowance to use for database io

    --cache.gc value                    (default: 25)                      ($GETH_CACHE_GC)
          Percentage of cache memory allowance to use for trie pruning (default = 25% full
          mode, 0% archive mode)

    --cache.noprefetch                  (default: false)                   ($GETH_CACHE_NOPREFETCH)
          Disable heuristic state prefetch during block import (less CPU and disk IO, more
          time waiting for data)

    --cache.preimages                   (default: false)                   ($GETH_CACHE_PREIMAGES)
          Enable recording the SHA3/keccak preimages of trie keys

    --cache.snapshot value              (default: 10)                      ($GETH_CACHE_SNAPSHOT)
          Percentage of cache memory allowance to use for snapshot caching (default = 10%
          full mode, 20% archive mode)

    --cache.trie value                  (default: 15)                      ($GETH_CACHE_TRIE)
          Percentage of cache memory allowance to use for trie caching (default = 15% full
          mode, 30% archive mode)

    --crypto.kzg value                  (default: "gokzg")                 ($GETH_CRYPTO_KZG)
          KZG library implementation to use; gokzg (recommended) or ckzg

    --fdlimit value                     (default: 0)                       ($GETH_FDLIMIT)
          Raise the open file descriptor resource limit (default = system fd limit)

   STATE HISTORY MANAGEMENT


    --gcmode value                      (default: "full")                  ($GETH_GCMODE)
          Blockchain garbage collection mode, only relevant in state.scheme=hash ("full",
          "archive")

    --history.state value               (default: 90000)                   ($GETH_HISTORY_STATE)
          Number of recent blocks to retain state history for (default = 90,000 blocks, 0
          = entire chain)

    --history.transactions value        (default: 2350000)                 ($GETH_HISTORY_TRANSACTIONS)
          Number of recent blocks to maintain transactions index for (default = about one
          year, 0 = entire chain)

    --state.scheme value                (default: "hash")                  ($GETH_STATE_SCHEME)
          Scheme to use for storing ethereum state ('hash' or 'path')

    --syncmode value                    (default: snap)                    ($GETH_SYNCMODE)
          Blockchain sync mode ("snap", "full" or "light")

   TRANSACTION POOL (BLOB)


    --blobpool.datacap value            (default: 10737418240)             ($GETH_BLOBPOOL_DATACAP)
          Disk space to allocate for pending blob transactions (soft limit)

    --blobpool.datadir value            (default: "blobpool")              ($GETH_BLOBPOOL_DATADIR)
          Data directory to store blob transactions in

    --blobpool.pricebump value          (default: 100)                     ($GETH_BLOBPOOL_PRICEBUMP)
          Price bump percentage to replace an already existing blob transaction

   TRANSACTION POOL (EVM)


    --txpool.accountqueue value         (default: 64)                      ($GETH_TXPOOL_ACCOUNTQUEUE)
          Maximum number of non-executable transaction slots permitted per account

    --txpool.accountslots value         (default: 16)                      ($GETH_TXPOOL_ACCOUNTSLOTS)
          Minimum number of executable transaction slots guaranteed per account

    --txpool.globalqueue value          (default: 1024)                    ($GETH_TXPOOL_GLOBALQUEUE)
          Maximum number of non-executable transaction slots for all accounts

    --txpool.globalslots value          (default: 5120)                    ($GETH_TXPOOL_GLOBALSLOTS)
          Maximum number of executable transaction slots for all accounts

    --txpool.journal value              (default: "transactions.rlp")      ($GETH_TXPOOL_JOURNAL)
          Disk journal for local transaction to survive node restarts

    --txpool.lifetime value             (default: 3h0m0s)                  ($GETH_TXPOOL_LIFETIME)
          Maximum amount of time non-executable transaction are queued

    --txpool.locals value                                                  ($GETH_TXPOOL_LOCALS)
          Comma separated accounts to treat as locals (no flush, priority inclusion)

    --txpool.nolocals                   (default: false)                   ($GETH_TXPOOL_NOLOCALS)
          Disables price exemptions for locally submitted transactions

    --txpool.pricebump value            (default: 10)                      ($GETH_TXPOOL_PRICEBUMP)
          Price bump percentage to replace an already existing transaction

    --txpool.pricelimit value           (default: 1)                       ($GETH_TXPOOL_PRICELIMIT)
          Minimum gas price tip to enforce for acceptance into the pool

    --txpool.rejournal value            (default: 1h0m0s)                  ($GETH_TXPOOL_REJOURNAL)
          Time interval to regenerate the local transaction journal

   VIRTUAL MACHINE


    --vmdebug                           (default: false)                   ($GETH_VMDEBUG)
          Record information useful for VM and contract debugging


COPYRIGHT:
   Copyright 2013-2023 The go-ethereum Authors
```
