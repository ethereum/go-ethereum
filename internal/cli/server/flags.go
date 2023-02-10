package server

import (
	"github.com/ethereum/go-ethereum/internal/cli/flagset"
)

func (c *Command) Flags() *flagset.Flagset {
	c.cliConfig = DefaultConfig()

	f := flagset.NewFlagSet("server")

	f.StringFlag(&flagset.StringFlag{
		Name:    "chain",
		Usage:   "Name of the chain to sync ('mumbai', 'mainnet') or path to a genesis file",
		Value:   &c.cliConfig.Chain,
		Default: c.cliConfig.Chain,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:               "identity",
		Usage:              "Name/Identity of the node",
		Value:              &c.cliConfig.Identity,
		Default:            c.cliConfig.Identity,
		HideDefaultFromDoc: true,
	})
	f.IntFlag(&flagset.IntFlag{
		Name:    "verbosity",
		Usage:   "Logging verbosity for the server (5=trace|4=debug|3=info|2=warn|1=error|0=crit), default = 3",
		Value:   &c.cliConfig.Verbosity,
		Default: c.cliConfig.Verbosity,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "log-level",
		Usage:   "Log level for the server (trace|debug|info|warn|error|crit), will be deprecated soon. Use verbosity instead",
		Value:   &c.cliConfig.LogLevel,
		Default: c.cliConfig.LogLevel,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:               "datadir",
		Usage:              "Path of the data directory to store information",
		Value:              &c.cliConfig.DataDir,
		Default:            c.cliConfig.DataDir,
		HideDefaultFromDoc: true,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "datadir.ancient",
		Usage:   "Data directory for ancient chain segments (default = inside chaindata)",
		Value:   &c.cliConfig.Ancient,
		Default: c.cliConfig.Ancient,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "keystore",
		Usage: "Path of the directory where keystores are located",
		Value: &c.cliConfig.KeyStoreDir,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "config",
		Usage: "File for the config file",
		Value: &c.configFile,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "syncmode",
		Usage:   `Blockchain sync mode (only "full" sync supported)`,
		Value:   &c.cliConfig.SyncMode,
		Default: c.cliConfig.SyncMode,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "gcmode",
		Usage:   `Blockchain garbage collection mode ("full", "archive")`,
		Value:   &c.cliConfig.GcMode,
		Default: c.cliConfig.GcMode,
	})
	f.MapStringFlag(&flagset.MapStringFlag{
		Name:    "eth.requiredblocks",
		Usage:   "Comma separated block number-to-hash mappings to require for peering (<number>=<hash>)",
		Value:   &c.cliConfig.RequiredBlocks,
		Default: c.cliConfig.RequiredBlocks,
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "snapshot",
		Usage:   `Enables the snapshot-database mode`,
		Value:   &c.cliConfig.Snapshot,
		Default: c.cliConfig.Snapshot,
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "bor.logs",
		Usage:   `Enables bor log retrieval`,
		Value:   &c.cliConfig.BorLogs,
		Default: c.cliConfig.BorLogs,
	})

	// heimdall
	f.StringFlag(&flagset.StringFlag{
		Name:    "bor.heimdall",
		Usage:   "URL of Heimdall service",
		Value:   &c.cliConfig.Heimdall.URL,
		Default: c.cliConfig.Heimdall.URL,
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "bor.withoutheimdall",
		Usage:   "Run without Heimdall service (for testing purpose)",
		Value:   &c.cliConfig.Heimdall.Without,
		Default: c.cliConfig.Heimdall.Without,
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "bor.devfakeauthor",
		Usage:   "Run miner without validator set authorization [dev mode] : Use with '--bor.withoutheimdall'",
		Value:   &c.cliConfig.DevFakeAuthor,
		Default: c.cliConfig.DevFakeAuthor,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "bor.heimdallgRPC",
		Usage:   "Address of Heimdall gRPC service",
		Value:   &c.cliConfig.Heimdall.GRPCAddress,
		Default: c.cliConfig.Heimdall.GRPCAddress,
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "bor.runheimdall",
		Usage:   "Run Heimdall service as a child process",
		Value:   &c.cliConfig.Heimdall.RunHeimdall,
		Default: c.cliConfig.Heimdall.RunHeimdall,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "bor.runheimdallargs",
		Usage:   "Arguments to pass to Heimdall service",
		Value:   &c.cliConfig.Heimdall.RunHeimdallArgs,
		Default: c.cliConfig.Heimdall.RunHeimdallArgs,
	})

	// txpool options
	f.SliceStringFlag(&flagset.SliceStringFlag{
		Name:    "txpool.locals",
		Usage:   "Comma separated accounts to treat as locals (no flush, priority inclusion)",
		Value:   &c.cliConfig.TxPool.Locals,
		Default: c.cliConfig.TxPool.Locals,
		Group:   "Transaction Pool",
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "txpool.nolocals",
		Usage:   "Disables price exemptions for locally submitted transactions",
		Value:   &c.cliConfig.TxPool.NoLocals,
		Default: c.cliConfig.TxPool.NoLocals,
		Group:   "Transaction Pool",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "txpool.journal",
		Usage:   "Disk journal for local transaction to survive node restarts",
		Value:   &c.cliConfig.TxPool.Journal,
		Default: c.cliConfig.TxPool.Journal,
		Group:   "Transaction Pool",
	})
	f.DurationFlag(&flagset.DurationFlag{
		Name:    "txpool.rejournal",
		Usage:   "Time interval to regenerate the local transaction journal",
		Value:   &c.cliConfig.TxPool.Rejournal,
		Default: c.cliConfig.TxPool.Rejournal,
		Group:   "Transaction Pool",
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "txpool.pricelimit",
		Usage:   "Minimum gas price limit to enforce for acceptance into the pool",
		Value:   &c.cliConfig.TxPool.PriceLimit,
		Default: c.cliConfig.TxPool.PriceLimit,
		Group:   "Transaction Pool",
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "txpool.pricebump",
		Usage:   "Price bump percentage to replace an already existing transaction",
		Value:   &c.cliConfig.TxPool.PriceBump,
		Default: c.cliConfig.TxPool.PriceBump,
		Group:   "Transaction Pool",
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "txpool.accountslots",
		Usage:   "Minimum number of executable transaction slots guaranteed per account",
		Value:   &c.cliConfig.TxPool.AccountSlots,
		Default: c.cliConfig.TxPool.AccountSlots,
		Group:   "Transaction Pool",
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "txpool.globalslots",
		Usage:   "Maximum number of executable transaction slots for all accounts",
		Value:   &c.cliConfig.TxPool.GlobalSlots,
		Default: c.cliConfig.TxPool.GlobalSlots,
		Group:   "Transaction Pool",
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "txpool.accountqueue",
		Usage:   "Maximum number of non-executable transaction slots permitted per account",
		Value:   &c.cliConfig.TxPool.AccountQueue,
		Default: c.cliConfig.TxPool.AccountQueue,
		Group:   "Transaction Pool",
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "txpool.globalqueue",
		Usage:   "Maximum number of non-executable transaction slots for all accounts",
		Value:   &c.cliConfig.TxPool.GlobalQueue,
		Default: c.cliConfig.TxPool.GlobalQueue,
		Group:   "Transaction Pool",
	})
	f.DurationFlag(&flagset.DurationFlag{
		Name:    "txpool.lifetime",
		Usage:   "Maximum amount of time non-executable transaction are queued",
		Value:   &c.cliConfig.TxPool.LifeTime,
		Default: c.cliConfig.TxPool.LifeTime,
		Group:   "Transaction Pool",
	})

	// sealer options
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "mine",
		Usage:   "Enable mining",
		Value:   &c.cliConfig.Sealer.Enabled,
		Default: c.cliConfig.Sealer.Enabled,
		Group:   "Sealer",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "miner.etherbase",
		Usage:   "Public address for block mining rewards",
		Value:   &c.cliConfig.Sealer.Etherbase,
		Default: c.cliConfig.Sealer.Etherbase,
		Group:   "Sealer",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "miner.extradata",
		Usage:   "Block extra data set by the miner (default = client version)",
		Value:   &c.cliConfig.Sealer.ExtraData,
		Default: c.cliConfig.Sealer.ExtraData,
		Group:   "Sealer",
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "miner.gaslimit",
		Usage:   "Target gas ceiling (gas limit) for mined blocks",
		Value:   &c.cliConfig.Sealer.GasCeil,
		Default: c.cliConfig.Sealer.GasCeil,
		Group:   "Sealer",
	})
	f.BigIntFlag(&flagset.BigIntFlag{
		Name:    "miner.gasprice",
		Usage:   "Minimum gas price for mining a transaction",
		Value:   c.cliConfig.Sealer.GasPrice,
		Group:   "Sealer",
		Default: c.cliConfig.Sealer.GasPrice,
	})

	// ethstats
	f.StringFlag(&flagset.StringFlag{
		Name:    "ethstats",
		Usage:   "Reporting URL of a ethstats service (nodename:secret@host:port)",
		Value:   &c.cliConfig.Ethstats,
		Default: c.cliConfig.Ethstats,
	})

	// gas price oracle
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "gpo.blocks",
		Usage:   "Number of recent blocks to check for gas prices",
		Value:   &c.cliConfig.Gpo.Blocks,
		Default: c.cliConfig.Gpo.Blocks,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "gpo.percentile",
		Usage:   "Suggested gas price is the given percentile of a set of recent transaction gas prices",
		Value:   &c.cliConfig.Gpo.Percentile,
		Default: c.cliConfig.Gpo.Percentile,
	})
	f.BigIntFlag(&flagset.BigIntFlag{
		Name:    "gpo.maxprice",
		Usage:   "Maximum gas price will be recommended by gpo",
		Value:   c.cliConfig.Gpo.MaxPrice,
		Default: c.cliConfig.Gpo.MaxPrice,
	})
	f.BigIntFlag(&flagset.BigIntFlag{
		Name:    "gpo.ignoreprice",
		Usage:   "Gas price below which gpo will ignore transactions",
		Value:   c.cliConfig.Gpo.IgnorePrice,
		Default: c.cliConfig.Gpo.IgnorePrice,
	})

	// cache options
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "cache",
		Usage:   "Megabytes of memory allocated to internal caching",
		Value:   &c.cliConfig.Cache.Cache,
		Default: c.cliConfig.Cache.Cache,
		Group:   "Cache",
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "cache.database",
		Usage:   "Percentage of cache memory allowance to use for database io",
		Value:   &c.cliConfig.Cache.PercDatabase,
		Default: c.cliConfig.Cache.PercDatabase,
		Group:   "Cache",
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "cache.trie",
		Usage:   "Percentage of cache memory allowance to use for trie caching",
		Value:   &c.cliConfig.Cache.PercTrie,
		Default: c.cliConfig.Cache.PercTrie,
		Group:   "Cache",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "cache.trie.journal",
		Usage:   "Disk journal directory for trie cache to survive node restarts",
		Value:   &c.cliConfig.Cache.Journal,
		Default: c.cliConfig.Cache.Journal,
		Group:   "Cache",
	})
	f.DurationFlag(&flagset.DurationFlag{
		Name:    "cache.trie.rejournal",
		Usage:   "Time interval to regenerate the trie cache journal",
		Value:   &c.cliConfig.Cache.Rejournal,
		Default: c.cliConfig.Cache.Rejournal,
		Group:   "Cache",
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "cache.gc",
		Usage:   "Percentage of cache memory allowance to use for trie pruning",
		Value:   &c.cliConfig.Cache.PercGc,
		Default: c.cliConfig.Cache.PercGc,
		Group:   "Cache",
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "cache.snapshot",
		Usage:   "Percentage of cache memory allowance to use for snapshot caching",
		Value:   &c.cliConfig.Cache.PercSnapshot,
		Default: c.cliConfig.Cache.PercSnapshot,
		Group:   "Cache",
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "cache.noprefetch",
		Usage:   "Disable heuristic state prefetch during block import (less CPU and disk IO, more time waiting for data)",
		Value:   &c.cliConfig.Cache.NoPrefetch,
		Default: c.cliConfig.Cache.NoPrefetch,
		Group:   "Cache",
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "cache.preimages",
		Usage:   "Enable recording the SHA3/keccak preimages of trie keys",
		Value:   &c.cliConfig.Cache.Preimages,
		Default: c.cliConfig.Cache.Preimages,
		Group:   "Cache",
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "cache.triesinmemory",
		Usage:   "Number of block states (tries) to keep in memory (default = 128)",
		Value:   &c.cliConfig.Cache.TriesInMemory,
		Default: c.cliConfig.Cache.TriesInMemory,
		Group:   "Cache",
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "txlookuplimit",
		Usage:   "Number of recent blocks to maintain transactions index for",
		Value:   &c.cliConfig.Cache.TxLookupLimit,
		Default: c.cliConfig.Cache.TxLookupLimit,
		Group:   "Cache",
	})

	// rpc options
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "rpc.gascap",
		Usage:   "Sets a cap on gas that can be used in eth_call/estimateGas (0=infinite)",
		Value:   &c.cliConfig.JsonRPC.GasCap,
		Default: c.cliConfig.JsonRPC.GasCap,
		Group:   "JsonRPC",
	})
	f.Float64Flag(&flagset.Float64Flag{
		Name:    "rpc.txfeecap",
		Usage:   "Sets a cap on transaction fee (in ether) that can be sent via the RPC APIs (0 = no cap)",
		Value:   &c.cliConfig.JsonRPC.TxFeeCap,
		Default: c.cliConfig.JsonRPC.TxFeeCap,
		Group:   "JsonRPC",
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "ipcdisable",
		Usage:   "Disable the IPC-RPC server",
		Value:   &c.cliConfig.JsonRPC.IPCDisable,
		Default: c.cliConfig.JsonRPC.IPCDisable,
		Group:   "JsonRPC",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "ipcpath",
		Usage:   "Filename for IPC socket/pipe within the datadir (explicit paths escape it)",
		Value:   &c.cliConfig.JsonRPC.IPCPath,
		Default: c.cliConfig.JsonRPC.IPCPath,
		Group:   "JsonRPC",
	})
	f.SliceStringFlag(&flagset.SliceStringFlag{
		Name:    "http.corsdomain",
		Usage:   "Comma separated list of domains from which to accept cross origin requests (browser enforced)",
		Value:   &c.cliConfig.JsonRPC.Http.Cors,
		Default: c.cliConfig.JsonRPC.Http.Cors,
		Group:   "JsonRPC",
	})
	f.SliceStringFlag(&flagset.SliceStringFlag{
		Name:    "http.vhosts",
		Usage:   "Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard.",
		Value:   &c.cliConfig.JsonRPC.Http.VHost,
		Default: c.cliConfig.JsonRPC.Http.VHost,
		Group:   "JsonRPC",
	})
	f.SliceStringFlag(&flagset.SliceStringFlag{
		Name:    "ws.origins",
		Usage:   "Origins from which to accept websockets requests",
		Value:   &c.cliConfig.JsonRPC.Ws.Origins,
		Default: c.cliConfig.JsonRPC.Ws.Origins,
		Group:   "JsonRPC",
	})
	f.SliceStringFlag(&flagset.SliceStringFlag{
		Name:    "graphql.corsdomain",
		Usage:   "Comma separated list of domains from which to accept cross origin requests (browser enforced)",
		Value:   &c.cliConfig.JsonRPC.Graphql.Cors,
		Default: c.cliConfig.JsonRPC.Graphql.Cors,
		Group:   "JsonRPC",
	})
	f.SliceStringFlag(&flagset.SliceStringFlag{
		Name:    "graphql.vhosts",
		Usage:   "Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard.",
		Value:   &c.cliConfig.JsonRPC.Graphql.VHost,
		Default: c.cliConfig.JsonRPC.Graphql.VHost,
		Group:   "JsonRPC",
	})

	// http options
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "http",
		Usage:   "Enable the HTTP-RPC server",
		Value:   &c.cliConfig.JsonRPC.Http.Enabled,
		Default: c.cliConfig.JsonRPC.Http.Enabled,
		Group:   "JsonRPC",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "http.addr",
		Usage:   "HTTP-RPC server listening interface",
		Value:   &c.cliConfig.JsonRPC.Http.Host,
		Default: c.cliConfig.JsonRPC.Http.Host,
		Group:   "JsonRPC",
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "http.port",
		Usage:   "HTTP-RPC server listening port",
		Value:   &c.cliConfig.JsonRPC.Http.Port,
		Default: c.cliConfig.JsonRPC.Http.Port,
		Group:   "JsonRPC",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "http.rpcprefix",
		Usage:   "HTTP path path prefix on which JSON-RPC is served. Use '/' to serve on all paths.",
		Value:   &c.cliConfig.JsonRPC.Http.Prefix,
		Default: c.cliConfig.JsonRPC.Http.Prefix,
		Group:   "JsonRPC",
	})
	f.SliceStringFlag(&flagset.SliceStringFlag{
		Name:    "http.api",
		Usage:   "API's offered over the HTTP-RPC interface",
		Value:   &c.cliConfig.JsonRPC.Http.API,
		Default: c.cliConfig.JsonRPC.Http.API,
		Group:   "JsonRPC",
	})

	// ws options
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "ws",
		Usage:   "Enable the WS-RPC server",
		Value:   &c.cliConfig.JsonRPC.Ws.Enabled,
		Default: c.cliConfig.JsonRPC.Ws.Enabled,
		Group:   "JsonRPC",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "ws.addr",
		Usage:   "WS-RPC server listening interface",
		Value:   &c.cliConfig.JsonRPC.Ws.Host,
		Default: c.cliConfig.JsonRPC.Ws.Host,
		Group:   "JsonRPC",
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "ws.port",
		Usage:   "WS-RPC server listening port",
		Value:   &c.cliConfig.JsonRPC.Ws.Port,
		Default: c.cliConfig.JsonRPC.Ws.Port,
		Group:   "JsonRPC",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "ws.rpcprefix",
		Usage:   "HTTP path prefix on which JSON-RPC is served. Use '/' to serve on all paths.",
		Value:   &c.cliConfig.JsonRPC.Ws.Prefix,
		Default: c.cliConfig.JsonRPC.Ws.Prefix,
		Group:   "JsonRPC",
	})
	f.SliceStringFlag(&flagset.SliceStringFlag{
		Name:    "ws.api",
		Usage:   "API's offered over the WS-RPC interface",
		Value:   &c.cliConfig.JsonRPC.Ws.API,
		Default: c.cliConfig.JsonRPC.Ws.API,
		Group:   "JsonRPC",
	})

	// graphql options
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "graphql",
		Usage:   "Enable GraphQL on the HTTP-RPC server. Note that GraphQL can only be started if an HTTP server is started as well.",
		Value:   &c.cliConfig.JsonRPC.Graphql.Enabled,
		Default: c.cliConfig.JsonRPC.Graphql.Enabled,
		Group:   "JsonRPC",
	})

	// p2p options
	f.StringFlag(&flagset.StringFlag{
		Name:    "bind",
		Usage:   "Network binding address",
		Value:   &c.cliConfig.P2P.Bind,
		Default: c.cliConfig.P2P.Bind,
		Group:   "P2P",
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "port",
		Usage:   "Network listening port",
		Value:   &c.cliConfig.P2P.Port,
		Default: c.cliConfig.P2P.Port,
		Group:   "P2P",
	})
	f.SliceStringFlag(&flagset.SliceStringFlag{
		Name:    "bootnodes",
		Usage:   "Comma separated enode URLs for P2P discovery bootstrap",
		Value:   &c.cliConfig.P2P.Discovery.Bootnodes,
		Default: c.cliConfig.P2P.Discovery.Bootnodes,
		Group:   "P2P",
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "maxpeers",
		Usage:   "Maximum number of network peers (network disabled if set to 0)",
		Value:   &c.cliConfig.P2P.MaxPeers,
		Default: c.cliConfig.P2P.MaxPeers,
		Group:   "P2P",
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "maxpendpeers",
		Usage:   "Maximum number of pending connection attempts",
		Value:   &c.cliConfig.P2P.MaxPendPeers,
		Default: c.cliConfig.P2P.MaxPendPeers,
		Group:   "P2P",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "nat",
		Usage:   "NAT port mapping mechanism (any|none|upnp|pmp|extip:<IP>)",
		Value:   &c.cliConfig.P2P.NAT,
		Default: c.cliConfig.P2P.NAT,
		Group:   "P2P",
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "nodiscover",
		Usage:   "Disables the peer discovery mechanism (manual peer addition)",
		Value:   &c.cliConfig.P2P.NoDiscover,
		Default: c.cliConfig.P2P.NoDiscover,
		Group:   "P2P",
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "v5disc",
		Usage:   "Enables the experimental RLPx V5 (Topic Discovery) mechanism",
		Value:   &c.cliConfig.P2P.Discovery.V5Enabled,
		Default: c.cliConfig.P2P.Discovery.V5Enabled,
		Group:   "P2P",
	})

	// metrics
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "metrics",
		Usage:   "Enable metrics collection and reporting",
		Value:   &c.cliConfig.Telemetry.Enabled,
		Default: c.cliConfig.Telemetry.Enabled,
		Group:   "Telemetry",
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "metrics.expensive",
		Usage:   "Enable expensive metrics collection and reporting",
		Value:   &c.cliConfig.Telemetry.Expensive,
		Default: c.cliConfig.Telemetry.Expensive,
		Group:   "Telemetry",
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "metrics.influxdb",
		Usage:   "Enable metrics export/push to an external InfluxDB database (v1)",
		Value:   &c.cliConfig.Telemetry.InfluxDB.V1Enabled,
		Default: c.cliConfig.Telemetry.InfluxDB.V1Enabled,
		Group:   "Telemetry",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "metrics.influxdb.endpoint",
		Usage:   "InfluxDB API endpoint to report metrics to",
		Value:   &c.cliConfig.Telemetry.InfluxDB.Endpoint,
		Default: c.cliConfig.Telemetry.InfluxDB.Endpoint,
		Group:   "Telemetry",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "metrics.influxdb.database",
		Usage:   "InfluxDB database name to push reported metrics to",
		Value:   &c.cliConfig.Telemetry.InfluxDB.Database,
		Default: c.cliConfig.Telemetry.InfluxDB.Database,
		Group:   "Telemetry",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "metrics.influxdb.username",
		Usage:   "Username to authorize access to the database",
		Value:   &c.cliConfig.Telemetry.InfluxDB.Username,
		Default: c.cliConfig.Telemetry.InfluxDB.Username,
		Group:   "Telemetry",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "metrics.influxdb.password",
		Usage:   "Password to authorize access to the database",
		Value:   &c.cliConfig.Telemetry.InfluxDB.Password,
		Default: c.cliConfig.Telemetry.InfluxDB.Password,
		Group:   "Telemetry",
	})
	f.MapStringFlag(&flagset.MapStringFlag{
		Name:    "metrics.influxdb.tags",
		Usage:   "Comma-separated InfluxDB tags (key/values) attached to all measurements",
		Value:   &c.cliConfig.Telemetry.InfluxDB.Tags,
		Group:   "Telemetry",
		Default: c.cliConfig.Telemetry.InfluxDB.Tags,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "metrics.prometheus-addr",
		Usage:   "Address for Prometheus Server",
		Value:   &c.cliConfig.Telemetry.PrometheusAddr,
		Default: c.cliConfig.Telemetry.PrometheusAddr,
		Group:   "Telemetry",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "metrics.opencollector-endpoint",
		Usage:   "OpenCollector Endpoint (host:port)",
		Value:   &c.cliConfig.Telemetry.OpenCollectorEndpoint,
		Default: c.cliConfig.Telemetry.OpenCollectorEndpoint,
		Group:   "Telemetry",
	})
	// influx db v2
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "metrics.influxdbv2",
		Usage:   "Enable metrics export/push to an external InfluxDB v2 database",
		Value:   &c.cliConfig.Telemetry.InfluxDB.V2Enabled,
		Default: c.cliConfig.Telemetry.InfluxDB.V2Enabled,
		Group:   "Telemetry",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "metrics.influxdb.token",
		Usage:   "Token to authorize access to the database (v2 only)",
		Value:   &c.cliConfig.Telemetry.InfluxDB.Token,
		Default: c.cliConfig.Telemetry.InfluxDB.Token,
		Group:   "Telemetry",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "metrics.influxdb.bucket",
		Usage:   "InfluxDB bucket name to push reported metrics to (v2 only)",
		Value:   &c.cliConfig.Telemetry.InfluxDB.Bucket,
		Default: c.cliConfig.Telemetry.InfluxDB.Bucket,
		Group:   "Telemetry",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "metrics.influxdb.organization",
		Usage:   "InfluxDB organization name (v2 only)",
		Value:   &c.cliConfig.Telemetry.InfluxDB.Organization,
		Default: c.cliConfig.Telemetry.InfluxDB.Organization,
		Group:   "Telemetry",
	})

	// account
	f.SliceStringFlag(&flagset.SliceStringFlag{
		Name:    "unlock",
		Usage:   "Comma separated list of accounts to unlock",
		Value:   &c.cliConfig.Accounts.Unlock,
		Default: c.cliConfig.Accounts.Unlock,
		Group:   "Account Management",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:    "password",
		Usage:   "Password file to use for non-interactive password input",
		Value:   &c.cliConfig.Accounts.PasswordFile,
		Default: c.cliConfig.Accounts.PasswordFile,
		Group:   "Account Management",
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "allow-insecure-unlock",
		Usage:   "Allow insecure account unlocking when account-related RPCs are exposed by http",
		Value:   &c.cliConfig.Accounts.AllowInsecureUnlock,
		Default: c.cliConfig.Accounts.AllowInsecureUnlock,
		Group:   "Account Management",
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "lightkdf",
		Usage:   "Reduce key-derivation RAM & CPU usage at some expense of KDF strength",
		Value:   &c.cliConfig.Accounts.UseLightweightKDF,
		Default: c.cliConfig.Accounts.UseLightweightKDF,
		Group:   "Account Management",
	})
	f.BoolFlag((&flagset.BoolFlag{
		Name:    "disable-bor-wallet",
		Usage:   "Disable the personal wallet endpoints",
		Value:   &c.cliConfig.Accounts.DisableBorWallet,
		Default: c.cliConfig.Accounts.DisableBorWallet,
	}))

	// grpc
	f.StringFlag(&flagset.StringFlag{
		Name:    "grpc.addr",
		Usage:   "Address and port to bind the GRPC server",
		Value:   &c.cliConfig.GRPC.Addr,
		Default: c.cliConfig.GRPC.Addr,
	})

	// developer
	f.BoolFlag(&flagset.BoolFlag{
		Name:    "dev",
		Usage:   "Enable developer mode with ephemeral proof-of-authority network and a pre-funded developer account, mining enabled",
		Value:   &c.cliConfig.Developer.Enabled,
		Default: c.cliConfig.Developer.Enabled,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:    "dev.period",
		Usage:   "Block period to use in developer mode (0 = mine only if transaction pending)",
		Value:   &c.cliConfig.Developer.Period,
		Default: c.cliConfig.Developer.Period,
	})
	return f
}
