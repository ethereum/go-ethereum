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
   1.11.5-unstable-f86913bc-20230315

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
   makecache              Generate ethash verification cache (for testing)
   makedag                Generate ethash mining DAG (for testing)
   removedb               Remove blockchain and state databases
   show-deprecated-flags  Show flags that have been deprecated
   snapshot               A set of commands based on the snapshot
   verkle                 A set of experimental verkle tree management commands
   version                Print version numbers
   version-check          Checks (online) for known Geth security vulnerabilities
   wallet                 Manage Ethereum presale wallets
   help, h                Shows a list of commands or help for one command

GLOBAL OPTIONS:
   ACCOUNT

    --allow-insecure-unlock        (default: false)
          Allow insecure account unlocking when account-related RPCs are exposed by http

    --keystore value
          Directory for the keystore (default = inside the datadir)

    --lightkdf                     (default: false)
          Reduce key-derivation RAM & CPU usage at some expense of KDF strength

    --password value
          Password file to use for non-interactive password input

    --pcscdpath value
          Path to the smartcard daemon (pcscd) socket file

    --signer value
          External signer (url or path to ipc file)

    --unlock value
          Comma separated list of accounts to unlock

    --usb                          (default: false)
          Enable monitoring and management of USB hardware wallets

   ALIASED (deprecated)

    --nousb                        (default: false)
          Disables monitoring for and managing USB hardware wallets (deprecated)

    --whitelist value
          Comma separated block number-to-hash mappings to enforce (<number>=<hash>)
          (deprecated in favor of --eth.requiredblocks)

   API AND CONSOLE

    --authrpc.addr value           (default: "localhost")
          Listening address for authenticated APIs

    --authrpc.jwtsecret value
          Path to a JWT secret to use for authenticated RPC endpoints

    --authrpc.port value           (default: 8551)
          Listening port for authenticated APIs

    --authrpc.vhosts value         (default: "localhost")
          Comma separated list of virtual hostnames from which to accept requests (server
          enforced). Accepts '*' wildcard.

    --exec value
          Execute JavaScript statement

    --graphql                      (default: false)
          Enable GraphQL on the HTTP-RPC server. Note that GraphQL can only be started if
          an HTTP server is started as well.

    --graphql.corsdomain value
          Comma separated list of domains from which to accept cross origin requests
          (browser enforced)

    --graphql.vhosts value         (default: "localhost")
          Comma separated list of virtual hostnames from which to accept requests (server
          enforced). Accepts '*' wildcard.

    --header value, -H value
          Pass custom headers to the RPC server when using --remotedb or the geth attach
          console. This flag can be given multiple times.

    --http                         (default: false)
          Enable the HTTP-RPC server

    --http.addr value              (default: "localhost")
          HTTP-RPC server listening interface

    --http.api value
          API's offered over the HTTP-RPC interface

    --http.corsdomain value
          Comma separated list of domains from which to accept cross origin requests
          (browser enforced)

    --http.port value              (default: 8545)
          HTTP-RPC server listening port

    --http.rpcprefix value
          HTTP path path prefix on which JSON-RPC is served. Use '/' to serve on all
          paths.

    --http.vhosts value            (default: "localhost")
          Comma separated list of virtual hostnames from which to accept requests (server
          enforced). Accepts '*' wildcard.

    --ipcdisable                   (default: false)
          Disable the IPC-RPC server

    --ipcpath value
          Filename for IPC socket/pipe within the datadir (explicit paths escape it)

    --jspath value                 (default: .)
          JavaScript root path for `loadScript`

    --preload value
          Comma separated list of JavaScript files to preload into the console

    --rpc.allow-unprotected-txs    (default: false)
          Allow for unprotected (non EIP155 signed) transactions to be submitted via RPC

    --rpc.enabledeprecatedpersonal (default: false)
          Enables the (deprecated) personal namespace

    --rpc.evmtimeout value         (default: 5s)
          Sets a timeout used for eth_call (0=infinite)

    --rpc.gascap value             (default: 50000000)
          Sets a cap on gas that can be used in eth_call/estimateGas (0=infinite)

    --rpc.txfeecap value           (default: 1)
          Sets a cap on transaction fee (in ether) that can be sent via the RPC APIs (0 =
          no cap)

    --ws                           (default: false)
          Enable the WS-RPC server

    --ws.addr value                (default: "localhost")
          WS-RPC server listening interface

    --ws.api value
          API's offered over the WS-RPC interface

    --ws.origins value
          Origins from which to accept websockets requests

    --ws.port value                (default: 8546)
          WS-RPC server listening port

    --ws.rpcprefix value
          HTTP path prefix on which JSON-RPC is served. Use '/' to serve on all paths.

   DEVELOPER CHAIN

    --dev                          (default: false)
          Ephemeral proof-of-authority network with a pre-funded developer account, mining
          enabled

    --dev.gaslimit value           (default: 11500000)
          Initial block gas limit

    --dev.period value             (default: 0)
          Block period to use in developer mode (0 = mine only if transaction pending)

   ETHASH

    --ethash.cachedir value
          Directory to store the ethash verification caches (default = inside the datadir)

    --ethash.cachesinmem value     (default: 2)
          Number of recent ethash caches to keep in memory (16MB each)

    --ethash.cacheslockmmap        (default: false)
          Lock memory maps of recent ethash caches

    --ethash.cachesondisk value    (default: 3)
          Number of recent ethash caches to keep on disk (16MB each)

    --ethash.dagdir value          (default: /Users/fjl/Library/Ethash)
          Directory to store the ethash mining DAGs

    --ethash.dagsinmem value       (default: 1)
          Number of recent ethash mining DAGs to keep in memory (1+GB each)

    --ethash.dagslockmmap          (default: false)
          Lock memory maps for recent ethash mining DAGs

    --ethash.dagsondisk value      (default: 2)
          Number of recent ethash mining DAGs to keep on disk (1+GB each)

   ETHEREUM

    --bloomfilter.size value       (default: 2048)
          Megabytes of memory allocated to bloom-filter for pruning

    --config value
          TOML configuration file

    --datadir value                (default: /Users/fjl/Library/Ethereum)
          Data directory for the databases and keystore

    --datadir.ancient value
          Root directory for ancient data (default = inside chaindata)

    --datadir.minfreedisk value
          Minimum free disk space in MB, once reached triggers auto shut down (default =
          --cache.gc converted to MB, 0 = disabled)

    --db.engine value              (default: "leveldb")
          Backing database implementation to use ('leveldb' or 'pebble')

    --eth.requiredblocks value
          Comma separated block number-to-hash mappings to require for peering
          (<number>=<hash>)

    --exitwhensynced               (default: false)
          Exits after block synchronisation completes

    --gcmode value                 (default: "full")
          Blockchain garbage collection mode ("full", "archive")

    --goerli                       (default: false)
          Görli network: pre-configured proof-of-authority test network

    --mainnet                      (default: false)
          Ethereum mainnet

    --networkid value              (default: 1)
          Explicitly set network id (integer)(For testnets: use --rinkeby, --goerli,
          --sepolia instead)

    --override.shanghai value      (default: 0)
          Manually specify the Shanghai fork timestamp, overriding the bundled setting

    --rinkeby                      (default: false)
          Rinkeby network: pre-configured proof-of-authority test network

    --sepolia                      (default: false)
          Sepolia network: pre-configured proof-of-work test network

    --snapshot                     (default: true)
          Enables snapshot-database mode (default = enable)

    --syncmode value               (default: snap)
          Blockchain sync mode ("snap", "full" or "light")

    --txlookuplimit value          (default: 2350000)
          Number of recent blocks to maintain transactions index for (default = about one
          year, 0 = entire chain)

   GAS PRICE ORACLE

    --gpo.blocks value             (default: 20)
          Number of recent blocks to check for gas prices

    --gpo.ignoreprice value        (default: 2)
          Gas price below which gpo will ignore transactions

    --gpo.maxprice value           (default: 500000000000)
          Maximum transaction priority fee (or gasprice before London fork) to be
          recommended by gpo

    --gpo.percentile value         (default: 60)
          Suggested gas price is the given percentile of a set of recent transaction gas
          prices

   LIGHT CLIENT

    --light.egress value           (default: 0)
          Outgoing bandwidth limit for serving light clients (kilobytes/sec, 0 =
          unlimited)

    --light.ingress value          (default: 0)
          Incoming bandwidth limit for serving light clients (kilobytes/sec, 0 =
          unlimited)

    --light.maxpeers value         (default: 100)
          Maximum number of light clients to serve, or light servers to attach to

    --light.nopruning              (default: false)
          Disable ancient light chain data pruning

    --light.nosyncserve            (default: false)
          Enables serving light clients before syncing

    --light.serve value            (default: 0)
          Maximum percentage of time allowed for serving LES requests (multi-threaded
          processing allows values over 100)

    --ulc.fraction value           (default: 75)
          Minimum % of trusted ultra-light servers required to announce a new head

    --ulc.onlyannounce             (default: false)
          Ultra light server sends announcements only

    --ulc.servers value
          List of trusted ultra-light servers

   LOGGING AND DEBUGGING

    --fakepow                      (default: false)
          Disables proof-of-work verification

    --log.backtrace value
          Request a stack trace at a specific logging statement (e.g. "block.go:271")

    --log.debug                    (default: false)
          Prepends log messages with call-site location (file and line number)

    --log.file value
          Write logs to a file

    --log.json                     (default: false)
          Format logs with JSON

    --nocompaction                 (default: false)
          Disables db compaction after import

    --pprof                        (default: false)
          Enable the pprof HTTP server

    --pprof.addr value             (default: "127.0.0.1")
          pprof HTTP server listening interface

    --pprof.blockprofilerate value (default: 0)
          Turn on block profiling with the given rate

    --pprof.cpuprofile value
          Write CPU profile to the given file

    --pprof.memprofilerate value   (default: 524288)
          Turn on memory profiling with the given rate

    --pprof.port value             (default: 6060)
          pprof HTTP server listening port

    --remotedb value
          URL for remote database

    --trace value
          Write execution trace to the given file

    --verbosity value              (default: 3)
          Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail

    --vmodule value
          Per-module verbosity: comma-separated list of <pattern>=<level> (e.g.
          eth/*=5,p2p=4)

   METRICS AND STATS

    --ethstats value
          Reporting URL of a ethstats service (nodename:secret@host:port)

    --metrics                      (default: false)
          Enable metrics collection and reporting

    --metrics.addr value
          Enable stand-alone metrics HTTP server listening interface.

    --metrics.expensive            (default: false)
          Enable expensive metrics collection and reporting

    --metrics.influxdb             (default: false)
          Enable metrics export/push to an external InfluxDB database

    --metrics.influxdb.bucket value (default: "geth")
          InfluxDB bucket name to push reported metrics to (v2 only)

    --metrics.influxdb.database value (default: "geth")
          InfluxDB database name to push reported metrics to

    --metrics.influxdb.endpoint value (default: "http://localhost:8086")
          InfluxDB API endpoint to report metrics to

    --metrics.influxdb.organization value (default: "geth")
          InfluxDB organization name (v2 only)

    --metrics.influxdb.password value (default: "test")
          Password to authorize access to the database

    --metrics.influxdb.tags value  (default: "host=localhost")
          Comma-separated InfluxDB tags (key/values) attached to all measurements

    --metrics.influxdb.token value (default: "test")
          Token to authorize access to the database (v2 only)

    --metrics.influxdb.username value (default: "test")
          Username to authorize access to the database

    --metrics.influxdbv2           (default: false)
          Enable metrics export/push to an external InfluxDB v2 database

    --metrics.port value           (default: 6060)
          Metrics HTTP server listening port.
          Please note that --metrics.addr must be set
          to start the server.

   MINER

    --mine                         (default: false)
          Enable mining

    --miner.etherbase value
          0x prefixed public address for block mining rewards

    --miner.extradata value
          Block extra data set by the miner (default = client version)

    --miner.gaslimit value         (default: 30000000)
          Target gas ceiling for mined blocks

    --miner.gasprice value         (default: 0)
          Minimum gas price for mining a transaction

    --miner.newpayload-timeout value (default: 2s)
          Specify the maximum time allowance for creating a new payload

    --miner.notify value
          Comma separated HTTP URL list to notify of new work packages

    --miner.notify.full            (default: false)
          Notify with pending block headers instead of work packages

    --miner.noverify               (default: false)
          Disable remote sealing verification

    --miner.recommit value         (default: 2s)
          Time interval to recreate the block being mined

    --miner.threads value          (default: 0)
          Number of CPU threads to use for mining

   MISC

    --help, -h                     (default: false)
          show help

    --synctarget value
          File for containing the hex-encoded block-rlp as sync target(dev feature)

    --version, -v                  (default: false)
          print the version

   NETWORKING

    --bootnodes value
          Comma separated enode URLs for P2P discovery bootstrap

    --discovery.dns value
          Sets DNS discovery entry points (use "" to disable DNS)

    --discovery.port value         (default: 30303)
          Use a custom UDP port for P2P discovery

    --identity value
          Custom node name

    --maxpeers value               (default: 50)
          Maximum number of network peers (network disabled if set to 0)

    --maxpendpeers value           (default: 0)
          Maximum number of pending connection attempts (defaults used if set to 0)

    --nat value                    (default: "any")
          NAT port mapping mechanism (any|none|upnp|pmp|pmp:<IP>|extip:<IP>)

    --netrestrict value
          Restricts network communication to the given IP networks (CIDR masks)

    --nodekey value
          P2P node key file

    --nodekeyhex value
          P2P node key as hex (for testing)

    --nodiscover                   (default: false)
          Disables the peer discovery mechanism (manual peer addition)

    --port value                   (default: 30303)
          Network listening port

    --v5disc                       (default: false)
          Enables the experimental RLPx V5 (Topic Discovery) mechanism

   PERFORMANCE TUNING

    --cache value                  (default: 1024)
          Megabytes of memory allocated to internal caching (default = 4096 mainnet full
          node, 128 light mode)

    --cache.blocklogs value        (default: 32)
          Size (in number of blocks) of the log cache for filtering

    --cache.database value         (default: 50)
          Percentage of cache memory allowance to use for database io

    --cache.gc value               (default: 25)
          Percentage of cache memory allowance to use for trie pruning (default = 25% full
          mode, 0% archive mode)

    --cache.noprefetch             (default: false)
          Disable heuristic state prefetch during block import (less CPU and disk IO, more
          time waiting for data)

    --cache.preimages              (default: false)
          Enable recording the SHA3/keccak preimages of trie keys

    --cache.snapshot value         (default: 10)
          Percentage of cache memory allowance to use for snapshot caching (default = 10%
          full mode, 20% archive mode)

    --cache.trie value             (default: 15)
          Percentage of cache memory allowance to use for trie caching (default = 15% full
          mode, 30% archive mode)

    --cache.trie.journal value     (default: "triecache")
          Disk journal directory for trie cache to survive node restarts

    --cache.trie.rejournal value   (default: 1h0m0s)
          Time interval to regenerate the trie cache journal

    --fdlimit value                (default: 0)
          Raise the open file descriptor resource limit (default = system fd limit)

   TRANSACTION POOL

    --txpool.accountqueue value    (default: 64)
          Maximum number of non-executable transaction slots permitted per account

    --txpool.accountslots value    (default: 16)
          Minimum number of executable transaction slots guaranteed per account

    --txpool.globalqueue value     (default: 1024)
          Maximum number of non-executable transaction slots for all accounts

    --txpool.globalslots value     (default: 5120)
          Maximum number of executable transaction slots for all accounts

    --txpool.journal value         (default: "transactions.rlp")
          Disk journal for local transaction to survive node restarts

    --txpool.lifetime value        (default: 3h0m0s)
          Maximum amount of time non-executable transaction are queued

    --txpool.locals value
          Comma separated accounts to treat as locals (no flush, priority inclusion)

    --txpool.nolocals              (default: false)
          Disables price exemptions for locally submitted transactions

    --txpool.pricebump value       (default: 10)
          Price bump percentage to replace an already existing transaction

    --txpool.pricelimit value      (default: 1)
          Minimum gas price limit to enforce for acceptance into the pool

    --txpool.rejournal value       (default: 1h0m0s)
          Time interval to regenerate the local transaction journal

   VIRTUAL MACHINE

    --vmdebug                      (default: false)
          Record information useful for VM and contract debugging


COPYRIGHT:
   Copyright 2013-2023 The go-ethereum Authors
```
