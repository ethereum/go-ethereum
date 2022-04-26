package server

import (
	"github.com/ethereum/go-ethereum/internal/cli/flagset"
)

func (c *Command) Flags() *flagset.Flagset {
	c.cliConfig = DefaultConfig()

	f := flagset.NewFlagSet("server")

	f.StringFlag(&flagset.StringFlag{
		Name:  "chain",
		Usage: "Name of the chain to sync",
		Value: &c.cliConfig.Chain,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "name",
		Usage: "Name/Identity of the node",
		Value: &c.cliConfig.Name,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "log-level",
		Usage: "Set log level for the server",
		Value: &c.cliConfig.LogLevel,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "datadir",
		Usage: "Path of the data directory to store information",
		Value: &c.cliConfig.DataDir,
	})
	f.SliceStringFlag(&flagset.SliceStringFlag{
		Name:  "config",
		Usage: "File for the config file",
		Value: &c.configFile,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "syncmode",
		Usage: `Blockchain sync mode ("fast", "full", "snap" or "light")`,
		Value: &c.cliConfig.SyncMode,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "gcmode",
		Usage: `Blockchain garbage collection mode ("full", "archive")`,
		Value: &c.cliConfig.GcMode,
	})
	f.MapStringFlag(&flagset.MapStringFlag{
		Name:  "whitelist",
		Usage: "Comma separated block number-to-hash mappings to enforce (<number>=<hash>)",
		Value: &c.cliConfig.Whitelist,
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "snapshot",
		Usage: `Enables snapshot-database mode (default = enable)`,
		Value: &c.cliConfig.Snapshot,
	})

	// heimdall
	f.StringFlag(&flagset.StringFlag{
		Name:  "bor.heimdall",
		Usage: "URL of Heimdall service",
		Value: &c.cliConfig.Heimdall.URL,
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "bor.withoutheimdall",
		Usage: "Run without Heimdall service (for testing purpose)",
		Value: &c.cliConfig.Heimdall.Without,
	})

	// txpool options
	f.SliceStringFlag(&flagset.SliceStringFlag{
		Name:  "txpool.locals",
		Usage: "Comma separated accounts to treat as locals (no flush, priority inclusion)",
		Value: &c.cliConfig.TxPool.Locals,
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "txpool.nolocals",
		Usage: "Disables price exemptions for locally submitted transactions",
		Value: &c.cliConfig.TxPool.NoLocals,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "txpool.journal",
		Usage: "Disk journal for local transaction to survive node restarts",
		Value: &c.cliConfig.TxPool.Journal,
	})
	f.DurationFlag(&flagset.DurationFlag{
		Name:  "txpool.rejournal",
		Usage: "Time interval to regenerate the local transaction journal",
		Value: &c.cliConfig.TxPool.Rejournal,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "txpool.pricelimit",
		Usage: "Minimum gas price limit to enforce for acceptance into the pool",
		Value: &c.cliConfig.TxPool.PriceLimit,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "txpool.pricebump",
		Usage: "Price bump percentage to replace an already existing transaction",
		Value: &c.cliConfig.TxPool.PriceBump,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "txpool.accountslots",
		Usage: "Minimum number of executable transaction slots guaranteed per account",
		Value: &c.cliConfig.TxPool.AccountSlots,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "txpool.globalslots",
		Usage: "Maximum number of executable transaction slots for all accounts",
		Value: &c.cliConfig.TxPool.GlobalSlots,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "txpool.accountqueue",
		Usage: "Maximum number of non-executable transaction slots permitted per account",
		Value: &c.cliConfig.TxPool.AccountQueue,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "txpool.globalqueue",
		Usage: "Maximum number of non-executable transaction slots for all accounts",
		Value: &c.cliConfig.TxPool.GlobalQueue,
	})
	f.DurationFlag(&flagset.DurationFlag{
		Name:  "txpool.lifetime",
		Usage: "Maximum amount of time non-executable transaction are queued",
		Value: &c.cliConfig.TxPool.LifeTime,
	})

	// sealer options
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "mine",
		Usage: "Enable mining",
		Value: &c.cliConfig.Sealer.Enabled,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "miner.etherbase",
		Usage: "Public address for block mining rewards (default = first account)",
		Value: &c.cliConfig.Sealer.Etherbase,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "miner.extradata",
		Usage: "Block extra data set by the miner (default = client version)",
		Value: &c.cliConfig.Sealer.ExtraData,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "miner.gaslimit",
		Usage: "Target gas ceiling for mined blocks",
		Value: &c.cliConfig.Sealer.GasCeil,
	})
	f.BigIntFlag(&flagset.BigIntFlag{
		Name:  "miner.gasprice",
		Usage: "Minimum gas price for mining a transaction",
		Value: c.cliConfig.Sealer.GasPrice,
	})

	// ethstats
	f.StringFlag(&flagset.StringFlag{
		Name:  "ethstats",
		Usage: "Reporting URL of a ethstats service (nodename:secret@host:port)",
		Value: &c.cliConfig.Ethstats,
	})

	// gas price oracle
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "gpo.blocks",
		Usage: "Number of recent blocks to check for gas prices",
		Value: &c.cliConfig.Gpo.Blocks,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "gpo.percentile",
		Usage: "Suggested gas price is the given percentile of a set of recent transaction gas prices",
		Value: &c.cliConfig.Gpo.Percentile,
	})
	f.BigIntFlag(&flagset.BigIntFlag{
		Name:  "gpo.maxprice",
		Usage: "Maximum gas price will be recommended by gpo",
		Value: c.cliConfig.Gpo.MaxPrice,
	})
	f.BigIntFlag(&flagset.BigIntFlag{
		Name:  "gpo.ignoreprice",
		Usage: "Gas price below which gpo will ignore transactions",
		Value: c.cliConfig.Gpo.IgnorePrice,
	})

	// cache options
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "cache",
		Usage: "Megabytes of memory allocated to internal caching (default = 4096 mainnet full node)",
		Value: &c.cliConfig.Cache.Cache,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "cache.database",
		Usage: "Percentage of cache memory allowance to use for database io",
		Value: &c.cliConfig.Cache.PercDatabase,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "cache.trie",
		Usage: "Percentage of cache memory allowance to use for trie caching (default = 15% full mode, 30% archive mode)",
		Value: &c.cliConfig.Cache.PercTrie,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "cache.trie.journal",
		Usage: "Disk journal directory for trie cache to survive node restarts",
		Value: &c.cliConfig.Cache.Journal,
	})
	f.DurationFlag(&flagset.DurationFlag{
		Name:  "cache.trie.rejournal",
		Usage: "Time interval to regenerate the trie cache journal",
		Value: &c.cliConfig.Cache.Rejournal,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "cache.gc",
		Usage: "Percentage of cache memory allowance to use for trie pruning (default = 25% full mode, 0% archive mode)",
		Value: &c.cliConfig.Cache.PercGc,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "cache.snapshot",
		Usage: "Percentage of cache memory allowance to use for snapshot caching (default = 10% full mode, 20% archive mode)",
		Value: &c.cliConfig.Cache.PercSnapshot,
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "cache.noprefetch",
		Usage: "Disable heuristic state prefetch during block import (less CPU and disk IO, more time waiting for data)",
		Value: &c.cliConfig.Cache.NoPrefetch,
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "cache.preimages",
		Usage: "Enable recording the SHA3/keccak preimages of trie keys",
		Value: &c.cliConfig.Cache.Preimages,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "txlookuplimit",
		Usage: "Number of recent blocks to maintain transactions index for (default = about 56 days, 0 = entire chain)",
		Value: &c.cliConfig.Cache.TxLookupLimit,
	})

	// rpc options
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "rpc.gascap",
		Usage: "Sets a cap on gas that can be used in eth_call/estimateGas (0=infinite)",
		Value: &c.cliConfig.JsonRPC.GasCap,
	})
	f.Float64Flag(&flagset.Float64Flag{
		Name:  "rpc.txfeecap",
		Usage: "Sets a cap on transaction fee (in ether) that can be sent via the RPC APIs (0 = no cap)",
		Value: &c.cliConfig.JsonRPC.TxFeeCap,
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "ipcdisable",
		Usage: "Disable the IPC-RPC server",
		Value: &c.cliConfig.JsonRPC.IPCDisable,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "ipcpath",
		Usage: "Filename for IPC socket/pipe within the datadir (explicit paths escape it)",
		Value: &c.cliConfig.JsonRPC.IPCPath,
	})
	f.SliceStringFlag(&flagset.SliceStringFlag{
		Name:  "jsonrpc.corsdomain",
		Usage: "Comma separated list of domains from which to accept cross origin requests (browser enforced)",
		Value: &c.cliConfig.JsonRPC.Cors,
	})
	f.SliceStringFlag(&flagset.SliceStringFlag{
		Name:  "jsonrpc.vhosts",
		Usage: "Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard.",
		Value: &c.cliConfig.JsonRPC.VHost,
	})

	// http options
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "http",
		Usage: "Enable the HTTP-RPC server",
		Value: &c.cliConfig.JsonRPC.Http.Enabled,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "http.addr",
		Usage: "HTTP-RPC server listening interface",
		Value: &c.cliConfig.JsonRPC.Http.Host,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "http.port",
		Usage: "HTTP-RPC server listening port",
		Value: &c.cliConfig.JsonRPC.Http.Port,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "http.rpcprefix",
		Usage: "HTTP path path prefix on which JSON-RPC is served. Use '/' to serve on all paths.",
		Value: &c.cliConfig.JsonRPC.Http.Prefix,
	})
	f.SliceStringFlag(&flagset.SliceStringFlag{
		Name:  "http.modules",
		Usage: "API's offered over the HTTP-RPC interface",
		Value: &c.cliConfig.JsonRPC.Http.Modules,
	})

	// ws options
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "ws",
		Usage: "Enable the WS-RPC server",
		Value: &c.cliConfig.JsonRPC.Ws.Enabled,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "ws.addr",
		Usage: "WS-RPC server listening interface",
		Value: &c.cliConfig.JsonRPC.Ws.Host,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "ws.port",
		Usage: "WS-RPC server listening port",
		Value: &c.cliConfig.JsonRPC.Ws.Port,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "ws.rpcprefix",
		Usage: "HTTP path prefix on which JSON-RPC is served. Use '/' to serve on all paths.",
		Value: &c.cliConfig.JsonRPC.Ws.Prefix,
	})
	f.SliceStringFlag(&flagset.SliceStringFlag{
		Name:  "ws.modules",
		Usage: "API's offered over the WS-RPC interface",
		Value: &c.cliConfig.JsonRPC.Ws.Modules,
	})

	// graphql options
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "graphql",
		Usage: "Enable GraphQL on the HTTP-RPC server. Note that GraphQL can only be started if an HTTP server is started as well.",
		Value: &c.cliConfig.JsonRPC.Graphql.Enabled,
	})

	// p2p options
	f.StringFlag(&flagset.StringFlag{
		Name:  "bind",
		Usage: "Network binding address",
		Value: &c.cliConfig.P2P.Bind,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "port",
		Usage: "Network listening port",
		Value: &c.cliConfig.P2P.Port,
	})
	f.SliceStringFlag(&flagset.SliceStringFlag{
		Name:  "bootnodes",
		Usage: "Comma separated enode URLs for P2P discovery bootstrap",
		Value: &c.cliConfig.P2P.Discovery.Bootnodes,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "maxpeers",
		Usage: "Maximum number of network peers (network disabled if set to 0)",
		Value: &c.cliConfig.P2P.MaxPeers,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "maxpendpeers",
		Usage: "Maximum number of pending connection attempts (defaults used if set to 0)",
		Value: &c.cliConfig.P2P.MaxPendPeers,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "nat",
		Usage: "NAT port mapping mechanism (any|none|upnp|pmp|extip:<IP>)",
		Value: &c.cliConfig.P2P.NAT,
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "nodiscover",
		Usage: "Disables the peer discovery mechanism (manual peer addition)",
		Value: &c.cliConfig.P2P.NoDiscover,
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "v5disc",
		Usage: "Enables the experimental RLPx V5 (Topic Discovery) mechanism",
		Value: &c.cliConfig.P2P.Discovery.V5Enabled,
	})

	// metrics
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "metrics",
		Usage: "Enable metrics collection and reporting",
		Value: &c.cliConfig.Telemetry.Enabled,
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "metrics.expensive",
		Usage: "Enable expensive metrics collection and reporting",
		Value: &c.cliConfig.Telemetry.Expensive,
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "metrics.influxdb",
		Usage: "Enable metrics export/push to an external InfluxDB database (v1)",
		Value: &c.cliConfig.Telemetry.InfluxDB.V1Enabled,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "metrics.influxdb.endpoint",
		Usage: "InfluxDB API endpoint to report metrics to",
		Value: &c.cliConfig.Telemetry.InfluxDB.Endpoint,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "metrics.influxdb.database",
		Usage: "InfluxDB database name to push reported metrics to",
		Value: &c.cliConfig.Telemetry.InfluxDB.Database,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "metrics.influxdb.username",
		Usage: "Username to authorize access to the database",
		Value: &c.cliConfig.Telemetry.InfluxDB.Username,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "metrics.influxdb.password",
		Usage: "Password to authorize access to the database",
		Value: &c.cliConfig.Telemetry.InfluxDB.Password,
	})
	f.MapStringFlag(&flagset.MapStringFlag{
		Name:  "metrics.influxdb.tags",
		Usage: "Comma-separated InfluxDB tags (key/values) attached to all measurements",
		Value: &c.cliConfig.Telemetry.InfluxDB.Tags,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "metrics.prometheus-addr",
		Usage: "Address for Prometheus Server",
		Value: &c.cliConfig.Telemetry.PrometheusAddr,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "metrics.opencollector-endpoint",
		Usage: "OpenCollector Endpoint (host:port)",
		Value: &c.cliConfig.Telemetry.OpenCollectorEndpoint,
	})
	// influx db v2
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "metrics.influxdbv2",
		Usage: "Enable metrics export/push to an external InfluxDB v2 database",
		Value: &c.cliConfig.Telemetry.InfluxDB.V2Enabled,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "metrics.influxdb.token",
		Usage: "Token to authorize access to the database (v2 only)",
		Value: &c.cliConfig.Telemetry.InfluxDB.Token,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "metrics.influxdb.bucket",
		Usage: "InfluxDB bucket name to push reported metrics to (v2 only)",
		Value: &c.cliConfig.Telemetry.InfluxDB.Bucket,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "metrics.influxdb.organization",
		Usage: "InfluxDB organization name (v2 only)",
		Value: &c.cliConfig.Telemetry.InfluxDB.Organization,
	})

	// account
	f.SliceStringFlag(&flagset.SliceStringFlag{
		Name:  "unlock",
		Usage: "Comma separated list of accounts to unlock",
		Value: &c.cliConfig.Accounts.Unlock,
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "password",
		Usage: "Password file to use for non-interactive password input",
		Value: &c.cliConfig.Accounts.PasswordFile,
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "allow-insecure-unlock",
		Usage: "Allow insecure account unlocking when account-related RPCs are exposed by http",
		Value: &c.cliConfig.Accounts.AllowInsecureUnlock,
	})
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "lightkdf",
		Usage: "Reduce key-derivation RAM & CPU usage at some expense of KDF strength",
		Value: &c.cliConfig.Accounts.UseLightweightKDF,
	})

	// grpc
	f.StringFlag(&flagset.StringFlag{
		Name:  "grpc.addr",
		Usage: "Address and port to bind the GRPC server",
		Value: &c.cliConfig.GRPC.Addr,
	})

	// developer
	f.BoolFlag(&flagset.BoolFlag{
		Name:  "dev",
		Usage: "Enable developer mode with ephemeral proof-of-authority network and a pre-funded developer account, mining enabled",
		Value: &c.cliConfig.Developer.Enabled,
	})
	f.Uint64Flag(&flagset.Uint64Flag{
		Name:  "dev.period",
		Usage: "Block period to use in developer mode (0 = mine only if transaction pending)",
		Value: &c.cliConfig.Developer.Period,
	})
	return f
}
