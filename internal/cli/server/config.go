package server

import (
	"crypto/ecdsa"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	godebug "runtime/debug"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/imdario/mergo"
	"github.com/mitchellh/go-homedir"
	gopsutil "github.com/shirou/gopsutil/mem"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/gasprice"
	"github.com/ethereum/go-ethereum/internal/cli/server/chains"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/p2p/netutil"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

type Config struct {
	chain *chains.Chain

	// Chain is the chain to sync with
	Chain string `hcl:"chain,optional" toml:"chain,optional"`

	// Identity of the node
	Identity string `hcl:"identity,optional" toml:"identity,optional"`

	// RequiredBlocks is a list of required (block number, hash) pairs to accept
	RequiredBlocks map[string]string `hcl:"eth.requiredblocks,optional" toml:"eth.requiredblocks,optional"`

	// Verbosity is the level of the logs to put out
	Verbosity int `hcl:"verbosity,optional" toml:"verbosity,optional"`

	// LogLevel is the level of the logs to put out
	LogLevel string `hcl:"log-level,optional" toml:"log-level,optional"`

	// Record information useful for VM and contract debugging
	EnablePreimageRecording bool `hcl:"vmdebug,optional" toml:"vmdebug,optional"`

	// DataDir is the directory to store the state in
	DataDir string `hcl:"datadir,optional" toml:"datadir,optional"`

	// Ancient is the directory to store the state in
	Ancient string `hcl:"ancient,optional" toml:"ancient,optional"`

	// DBEngine is used to select leveldb or pebble as database
	DBEngine string `hcl:"db.engine,optional" toml:"db.engine,optional"`

	// KeyStoreDir is the directory to store keystores
	KeyStoreDir string `hcl:"keystore,optional" toml:"keystore,optional"`

	// Maximum number of messages in a batch (default=100, use 0 for no limits)
	RPCBatchLimit uint64 `hcl:"rpc.batchlimit,optional" toml:"rpc.batchlimit,optional"`

	// Maximum size (in bytes) a result of an rpc request could have (default=100000, use 0 for no limits)
	RPCReturnDataLimit uint64 `hcl:"rpc.returndatalimit,optional" toml:"rpc.returndatalimit,optional"`

	// SyncMode selects the sync protocol
	SyncMode string `hcl:"syncmode,optional" toml:"syncmode,optional"`

	// GcMode selects the garbage collection mode for the trie
	GcMode string `hcl:"gcmode,optional" toml:"gcmode,optional"`

	// Snapshot enables the snapshot database mode
	Snapshot bool `hcl:"snapshot,optional" toml:"snapshot,optional"`

	// BorLogs enables bor log retrieval
	BorLogs bool `hcl:"bor.logs,optional" toml:"bor.logs,optional"`

	// Ethstats is the address of the ethstats server to send telemetry
	Ethstats string `hcl:"ethstats,optional" toml:"ethstats,optional"`

	// Logging has the logging related settings
	Logging *LoggingConfig `hcl:"log,block" toml:"log,block"`

	// P2P has the p2p network related settings
	P2P *P2PConfig `hcl:"p2p,block" toml:"p2p,block"`

	// Heimdall has the heimdall connection related settings
	Heimdall *HeimdallConfig `hcl:"heimdall,block" toml:"heimdall,block"`

	// TxPool has the transaction pool related settings
	TxPool *TxPoolConfig `hcl:"txpool,block" toml:"txpool,block"`

	// Sealer has the validator related settings
	Sealer *SealerConfig `hcl:"miner,block" toml:"miner,block"`

	// JsonRPC has the json-rpc related settings
	JsonRPC *JsonRPCConfig `hcl:"jsonrpc,block" toml:"jsonrpc,block"`

	// Gpo has the gas price oracle related settings
	Gpo *GpoConfig `hcl:"gpo,block" toml:"gpo,block"`

	// Telemetry has the telemetry related settings
	Telemetry *TelemetryConfig `hcl:"telemetry,block" toml:"telemetry,block"`

	// Cache has the cache related settings
	Cache *CacheConfig `hcl:"cache,block" toml:"cache,block"`

	ExtraDB *ExtraDBConfig `hcl:"leveldb,block" toml:"leveldb,block"`

	// Account has the validator account related settings
	Accounts *AccountsConfig `hcl:"accounts,block" toml:"accounts,block"`

	// GRPC has the grpc server related settings
	GRPC *GRPCConfig `hcl:"grpc,block" toml:"grpc,block"`

	// Developer has the developer mode related settings
	Developer *DeveloperConfig `hcl:"developer,block" toml:"developer,block"`

	// ParallelEVM has the parallel evm related settings
	ParallelEVM *ParallelEVMConfig `hcl:"parallelevm,block" toml:"parallelevm,block"`

	// Develop Fake Author mode to produce blocks without authorisation
	DevFakeAuthor bool `hcl:"devfakeauthor,optional" toml:"devfakeauthor,optional"`

	// Pprof has the pprof related settings
	Pprof *PprofConfig `hcl:"pprof,block" toml:"pprof,block"`
}

type LoggingConfig struct {
	// Per-module verbosity: comma-separated list of <pattern>=<level> (e.g. eth/*=5,p2p=4)
	Vmodule string `hcl:"vmodule,optional" toml:"vmodule,optional"`

	// Format logs with JSON
	Json bool `hcl:"json,optional" toml:"json,optional"`

	// Request a stack trace at a specific logging statement (e.g. "block.go:271")
	Backtrace string `hcl:"backtrace,optional" toml:"backtrace,optional"`

	// Prepends log messages with call-site location (file and line number)
	Debug bool `hcl:"debug,optional" toml:"debug,optional"`

	// EnableBlockTracking allows logging of information collected while tracking block lifecycle
	EnableBlockTracking bool `hcl:"enable-block-tracking,optional" toml:"enable-block-tracking,optional"`

	// TODO - implement this
	// // Write execution trace to the given file
	// Trace string `hcl:"trace,optional" toml:"trace,optional"`
}

type PprofConfig struct {
	// Enableed enable the pprof HTTP server
	Enabled bool `hcl:"pprof,optional" toml:"pprof,optional"`

	// pprof HTTP server listening port
	Port int `hcl:"port,optional" toml:"port,optional"`

	// pprof HTTP server listening interface
	Addr string `hcl:"addr,optional" toml:"addr,optional"`

	// Turn on memory profiling with the given rate
	MemProfileRate int `hcl:"memprofilerate,optional" toml:"memprofilerate,optional"`

	// Turn on block profiling with the given rate
	BlockProfileRate int `hcl:"blockprofilerate,optional" toml:"blockprofilerate,optional"`

	// // Write CPU profile to the given file
	// CPUProfile string `hcl:"cpuprofile,optional" toml:"cpuprofile,optional"`
}

type P2PConfig struct {
	// MaxPeers sets the maximum number of connected peers
	MaxPeers uint64 `hcl:"maxpeers,optional" toml:"maxpeers,optional"`

	// MaxPendPeers sets the maximum number of pending connected peers
	MaxPendPeers uint64 `hcl:"maxpendpeers,optional" toml:"maxpendpeers,optional"`

	// Bind is the bind address
	Bind string `hcl:"bind,optional" toml:"bind,optional"`

	// Port is the port number
	Port uint64 `hcl:"port,optional" toml:"port,optional"`

	// NoDiscover is used to disable discovery
	NoDiscover bool `hcl:"nodiscover,optional" toml:"nodiscover,optional"`

	// NAT it used to set NAT options
	NAT string `hcl:"nat,optional" toml:"nat,optional"`

	// Connectivity can be restricted to certain IP networks.
	// If this option is set to a non-nil value, only hosts which match one of the
	// IP networks contained in the list are considered.
	NetRestrict string `hcl:"netrestrict,optional" toml:"netrestrict,optional"`

	// P2P node key file
	NodeKey string `hcl:"nodekey,optional" toml:"nodekey,optional"`

	// P2P node key as hex
	NodeKeyHex string `hcl:"nodekeyhex,optional" toml:"nodekeyhex,optional"`

	// Discovery has the p2p discovery related settings
	Discovery *P2PDiscovery `hcl:"discovery,block" toml:"discovery,block"`

	// TxArrivalWait sets the maximum duration the transaction fetcher will wait for
	// an announced transaction to arrive before explicitly requesting it
	TxArrivalWait    time.Duration `hcl:"-,optional" toml:"-"`
	TxArrivalWaitRaw string        `hcl:"txarrivalwait,optional" toml:"txarrivalwait,optional"`
}

type P2PDiscovery struct {
	// DiscoveryV4 specifies whether V4 discovery should be started.
	DiscoveryV4 bool `hcl:"v4disc,optional" toml:"v4disc,optional"`

	// V5Enabled is used to enable disc v5 discovery mode
	V5Enabled bool `hcl:"v5disc,optional" toml:"v5disc,optional"`

	// Bootnodes is the list of initial bootnodes
	Bootnodes []string `hcl:"bootnodes,optional" toml:"bootnodes,optional"`

	// BootnodesV4 is the list of initial v4 bootnodes
	BootnodesV4 []string `hcl:"bootnodesv4,optional" toml:"bootnodesv4,optional"`

	// BootnodesV5 is the list of initial v5 bootnodes
	BootnodesV5 []string `hcl:"bootnodesv5,optional" toml:"bootnodesv5,optional"`

	// StaticNodes is the list of static nodes
	StaticNodes []string `hcl:"static-nodes,optional" toml:"static-nodes,optional"`

	// TrustedNodes is the list of trusted nodes
	TrustedNodes []string `hcl:"trusted-nodes,optional" toml:"trusted-nodes,optional"`

	// DNS is the list of enrtree:// URLs which will be queried for nodes to connect to
	DNS []string `hcl:"dns,optional" toml:"dns,optional"`
}

type HeimdallConfig struct {
	// URL is the url of the heimdall server
	URL string `hcl:"url,optional" toml:"url,optional"`

	// Without is used to disable remote heimdall during testing
	Without bool `hcl:"bor.without,optional" toml:"bor.without,optional"`

	// GRPCAddress is the address of the heimdall grpc server
	GRPCAddress string `hcl:"grpc-address,optional" toml:"grpc-address,optional"`

	// RunHeimdall is used to run heimdall as a child process
	RunHeimdall bool `hcl:"bor.runheimdall,optional" toml:"bor.runheimdall,optional"`

	// RunHeimdal args are the arguments to run heimdall with
	RunHeimdallArgs string `hcl:"bor.runheimdallargs,optional" toml:"bor.runheimdallargs,optional"`

	// UseHeimdallApp is used to fetch data from heimdall app when running heimdall as a child process
	UseHeimdallApp bool `hcl:"bor.useheimdallapp,optional" toml:"bor.useheimdallapp,optional"`
}

type TxPoolConfig struct {
	// Locals are the addresses that should be treated by default as local
	Locals []string `hcl:"locals,optional" toml:"locals,optional"`

	// NoLocals enables whether local transaction handling should be disabled
	NoLocals bool `hcl:"nolocals,optional" toml:"nolocals,optional"`

	// Journal is the path to store local transactions to survive node restarts
	Journal string `hcl:"journal,optional" toml:"journal,optional"`

	// Rejournal is the time interval to regenerate the local transaction journal
	Rejournal    time.Duration `hcl:"-,optional" toml:"-"`
	RejournalRaw string        `hcl:"rejournal,optional" toml:"rejournal,optional"`

	// PriceLimit is the minimum gas price to enforce for acceptance into the pool
	PriceLimit uint64 `hcl:"pricelimit,optional" toml:"pricelimit,optional"`

	// PriceBump is the minimum price bump percentage to replace an already existing transaction (nonce)
	PriceBump uint64 `hcl:"pricebump,optional" toml:"pricebump,optional"`

	// AccountSlots is the number of executable transaction slots guaranteed per account
	AccountSlots uint64 `hcl:"accountslots,optional" toml:"accountslots,optional"`

	// GlobalSlots is the maximum number of executable transaction slots for all accounts
	GlobalSlots uint64 `hcl:"globalslots,optional" toml:"globalslots,optional"`

	// AccountQueue is the maximum number of non-executable transaction slots permitted per account
	AccountQueue uint64 `hcl:"accountqueue,optional" toml:"accountqueue,optional"`

	// GlobalQueueis the maximum number of non-executable transaction slots for all accounts
	GlobalQueue uint64 `hcl:"globalqueue,optional" toml:"globalqueue,optional"`

	// lifetime is the maximum amount of time non-executable transaction are queued
	LifeTime    time.Duration `hcl:"-,optional" toml:"-"`
	LifeTimeRaw string        `hcl:"lifetime,optional" toml:"lifetime,optional"`
}

type SealerConfig struct {
	// Enabled is used to enable validator mode
	Enabled bool `hcl:"mine,optional" toml:"mine,optional"`

	// Etherbase is the address of the validator
	Etherbase string `hcl:"etherbase,optional" toml:"etherbase,optional"`

	// ExtraData is the block extra data set by the miner
	ExtraData string `hcl:"extradata,optional" toml:"extradata,optional"`

	// GasCeil is the target gas ceiling for mined blocks.
	GasCeil uint64 `hcl:"gaslimit,optional" toml:"gaslimit,optional"`

	// GasPrice is the minimum gas price for mining a transaction
	GasPrice    *big.Int `hcl:"-,optional" toml:"-"`
	GasPriceRaw string   `hcl:"gasprice,optional" toml:"gasprice,optional"`

	// The time interval for miner to re-create mining work.
	Recommit    time.Duration `hcl:"-,optional" toml:"-"`
	RecommitRaw string        `hcl:"recommit,optional" toml:"recommit,optional"`

	CommitInterruptFlag bool `hcl:"commitinterrupt,optional" toml:"commitinterrupt,optional"`
}

type JsonRPCConfig struct {
	// IPCDisable enables whether ipc is enabled or not
	IPCDisable bool `hcl:"ipcdisable,optional" toml:"ipcdisable,optional"`

	// IPCPath is the path of the ipc endpoint
	IPCPath string `hcl:"ipcpath,optional" toml:"ipcpath,optional"`

	// GasCap is the global gas cap for eth-call variants.
	GasCap uint64 `hcl:"gascap,optional" toml:"gascap,optional"`

	// Sets a timeout used for eth_call (0=infinite)
	RPCEVMTimeout    time.Duration `hcl:"-,optional" toml:"-"`
	RPCEVMTimeoutRaw string        `hcl:"evmtimeout,optional" toml:"evmtimeout,optional"`

	// TxFeeCap is the global transaction fee cap for send-transaction variants
	TxFeeCap float64 `hcl:"txfeecap,optional" toml:"txfeecap,optional"`

	// Http has the json-rpc http related settings
	Http *APIConfig `hcl:"http,block" toml:"http,block"`

	// Ws has the json-rpc websocket related settings
	Ws *APIConfig `hcl:"ws,block" toml:"ws,block"`

	// Graphql has the json-rpc graphql related settings
	Graphql *APIConfig `hcl:"graphql,block" toml:"graphql,block"`

	// AUTH RPC related settings
	Auth *AUTHConfig `hcl:"auth,block" toml:"auth,block"`

	HttpTimeout *HttpTimeouts `hcl:"timeouts,block" toml:"timeouts,block"`

	AllowUnprotectedTxs bool `hcl:"allow-unprotected-txs,optional" toml:"allow-unprotected-txs,optional"`

	// EnablePersonal enables the deprecated personal namespace.
	EnablePersonal bool `hcl:"enabledeprecatedpersonal,optional" toml:"enabledeprecatedpersonal,optional"`
}

type AUTHConfig struct {
	// JWTSecret is the hex-encoded jwt secret.
	JWTSecret string `hcl:"jwtsecret,optional" toml:"jwtsecret,optional"`

	// Addr is the listening address on which authenticated APIs are provided.
	Addr string `hcl:"addr,optional" toml:"addr,optional"`

	// Port is the port number on which authenticated APIs are provided.
	Port uint64 `hcl:"port,optional" toml:"port,optional"`

	// VHosts is the list of virtual hostnames which are allowed on incoming requests
	// for the authenticated api. This is by default {'localhost'}.
	VHosts []string `hcl:"vhosts,optional" toml:"vhosts,optional"`
}

type GRPCConfig struct {
	// Addr is the bind address for the grpc rpc server
	Addr string `hcl:"addr,optional" toml:"addr,optional"`
}

type APIConfig struct {
	// Enabled selects whether the api is enabled
	Enabled bool `hcl:"enabled,optional" toml:"enabled,optional"`

	// Port is the port number for this api
	Port uint64 `hcl:"port,optional" toml:"port,optional"`

	// Prefix is the http prefix to expose this api
	Prefix string `hcl:"prefix,optional" toml:"prefix,optional"`

	// Host is the address to bind the api
	Host string `hcl:"host,optional" toml:"host,optional"`

	// API is the list of enabled api modules
	API []string `hcl:"api,optional" toml:"api,optional"`

	// VHost is the list of valid virtual hosts
	VHost []string `hcl:"vhosts,optional" toml:"vhosts,optional"`

	// Cors is the list of Cors endpoints
	Cors []string `hcl:"corsdomain,optional" toml:"corsdomain,optional"`

	// Origins is the list of endpoints to accept requests from (only consumed for websockets)
	Origins []string `hcl:"origins,optional" toml:"origins,optional"`

	// ExecutionPoolSize is max size of workers to be used for rpc execution
	ExecutionPoolSize uint64 `hcl:"ep-size,optional" toml:"ep-size,optional"`

	// ExecutionPoolRequestTimeout is timeout used by execution pool for rpc execution
	ExecutionPoolRequestTimeout    time.Duration `hcl:"-,optional" toml:"-"`
	ExecutionPoolRequestTimeoutRaw string        `hcl:"ep-requesttimeout,optional" toml:"ep-requesttimeout,optional"`
}

// Used from rpc.HTTPTimeouts
type HttpTimeouts struct {
	// ReadTimeout is the maximum duration for reading the entire
	// request, including the body.
	//
	// Because ReadTimeout does not let Handlers make per-request
	// decisions on each request body's acceptable deadline or
	// upload rate, most users will prefer to use
	// ReadHeaderTimeout. It is valid to use them both.
	ReadTimeout    time.Duration `hcl:"-,optional" toml:"-"`
	ReadTimeoutRaw string        `hcl:"read,optional" toml:"read,optional"`

	// WriteTimeout is the maximum duration before timing out
	// writes of the response. It is reset whenever a new
	// request's header is read. Like ReadTimeout, it does not
	// let Handlers make decisions on a per-request basis.
	WriteTimeout    time.Duration `hcl:"-,optional" toml:"-"`
	WriteTimeoutRaw string        `hcl:"write,optional" toml:"write,optional"`

	// IdleTimeout is the maximum amount of time to wait for the
	// next request when keep-alives are enabled. If IdleTimeout
	// is zero, the value of ReadTimeout is used. If both are
	// zero, ReadHeaderTimeout is used.
	IdleTimeout    time.Duration `hcl:"-,optional" toml:"-"`
	IdleTimeoutRaw string        `hcl:"idle,optional" toml:"idle,optional"`
}

type GpoConfig struct {
	// Blocks is the number of blocks to track to compute the price oracle
	Blocks uint64 `hcl:"blocks,optional" toml:"blocks,optional"`

	// Percentile sets the weights to new blocks
	Percentile uint64 `hcl:"percentile,optional" toml:"percentile,optional"`

	// Maximum header history of gasprice oracle
	MaxHeaderHistory int `hcl:"maxheaderhistory,optional" toml:"maxheaderhistory,optional"`

	// Maximum block history of gasprice oracle
	MaxBlockHistory int `hcl:"maxblockhistory,optional" toml:"maxblockhistory,optional"`

	// MaxPrice is an upper bound gas price
	MaxPrice    *big.Int `hcl:"-,optional" toml:"-"`
	MaxPriceRaw string   `hcl:"maxprice,optional" toml:"maxprice,optional"`

	// IgnorePrice is a lower bound gas price
	IgnorePrice    *big.Int `hcl:"-,optional" toml:"-"`
	IgnorePriceRaw string   `hcl:"ignoreprice,optional" toml:"ignoreprice,optional"`
}

type TelemetryConfig struct {
	// Enabled enables metrics
	Enabled bool `hcl:"metrics,optional" toml:"metrics,optional"`

	// Expensive enables expensive metrics
	Expensive bool `hcl:"expensive,optional" toml:"expensive,optional"`

	// InfluxDB has the influxdb related settings
	InfluxDB *InfluxDBConfig `hcl:"influx,block" toml:"influx,block"`

	// Prometheus Address
	PrometheusAddr string `hcl:"prometheus-addr,optional" toml:"prometheus-addr,optional"`

	// Open collector endpoint
	OpenCollectorEndpoint string `hcl:"opencollector-endpoint,optional" toml:"opencollector-endpoint,optional"`
}

type InfluxDBConfig struct {
	// V1Enabled enables influx v1 mode
	V1Enabled bool `hcl:"influxdb,optional" toml:"influxdb,optional"`

	// Endpoint is the url endpoint of the influxdb service
	Endpoint string `hcl:"endpoint,optional" toml:"endpoint,optional"`

	// Database is the name of the database in Influxdb to store the metrics.
	Database string `hcl:"database,optional" toml:"database,optional"`

	// Enabled is the username to authorize access to Influxdb
	Username string `hcl:"username,optional" toml:"username,optional"`

	// Password is the password to authorize access to Influxdb
	Password string `hcl:"password,optional" toml:"password,optional"`

	// Tags are tags attaches to all generated metrics
	Tags map[string]string `hcl:"tags,optional" toml:"tags,optional"`

	// Enabled enables influx v2 mode
	V2Enabled bool `hcl:"influxdbv2,optional" toml:"influxdbv2,optional"`

	// Token is the token to authorize access to Influxdb V2.
	Token string `hcl:"token,optional" toml:"token,optional"`

	// Bucket is the bucket to store metrics in Influxdb V2.
	Bucket string `hcl:"bucket,optional" toml:"bucket,optional"`

	// Organization is the name of the organization for Influxdb V2.
	Organization string `hcl:"organization,optional" toml:"organization,optional"`
}

type CacheConfig struct {
	// Cache is the amount of cache of the node
	Cache uint64 `hcl:"cache,optional" toml:"cache,optional"`

	// PercGc is percentage of cache used for garbage collection
	PercGc uint64 `hcl:"gc,optional" toml:"gc,optional"`

	// PercSnapshot is percentage of cache used for snapshots
	PercSnapshot uint64 `hcl:"snapshot,optional" toml:"snapshot,optional"`

	// PercDatabase is percentage of cache used for the database
	PercDatabase uint64 `hcl:"database,optional" toml:"database,optional"`

	// PercTrie is percentage of cache used for the trie
	PercTrie uint64 `hcl:"trie,optional" toml:"trie,optional"`

	// NoPrefetch is used to disable prefetch of tries
	NoPrefetch bool `hcl:"noprefetch,optional" toml:"noprefetch,optional"`

	// Preimages is used to enable the track of hash preimages
	Preimages bool `hcl:"preimages,optional" toml:"preimages,optional"`

	// TxLookupLimit sets the maximum number of blocks from head whose tx indices are reserved.
	TxLookupLimit uint64 `hcl:"txlookuplimit,optional" toml:"txlookuplimit,optional"`

	// Number of block states to keep in memory (default = 128)
	TriesInMemory uint64 `hcl:"triesinmemory,optional" toml:"triesinmemory,optional"`

	// This is the number of blocks for which logs will be cached in the filter system.
	FilterLogCacheSize int `hcl:"blocklogs,optional" toml:"blocklogs,optional"`

	// Time after which the Merkle Patricia Trie is stored to disc from memory
	TrieTimeout    time.Duration `hcl:"-,optional" toml:"-"`
	TrieTimeoutRaw string        `hcl:"timeout,optional" toml:"timeout,optional"`

	// Raise the open file descriptor resource limit (default = system fd limit)
	FDLimit int `hcl:"fdlimit,optional" toml:"fdlimit,optional"`
}

type ExtraDBConfig struct {
	LevelDbCompactionTableSize           uint64  `hcl:"compactiontablesize,optional" toml:"compactiontablesize,optional"`
	LevelDbCompactionTableSizeMultiplier float64 `hcl:"compactiontablesizemultiplier,optional" toml:"compactiontablesizemultiplier,optional"`
	LevelDbCompactionTotalSize           uint64  `hcl:"compactiontotalsize,optional" toml:"compactiontotalsize,optional"`
	LevelDbCompactionTotalSizeMultiplier float64 `hcl:"compactiontotalsizemultiplier,optional" toml:"compactiontotalsizemultiplier,optional"`
}

type AccountsConfig struct {
	// Unlock is the list of addresses to unlock in the node
	Unlock []string `hcl:"unlock,optional" toml:"unlock,optional"`

	// PasswordFile is the file where the account passwords are stored
	PasswordFile string `hcl:"password,optional" toml:"password,optional"`

	// AllowInsecureUnlock allows user to unlock accounts in unsafe http environment.
	AllowInsecureUnlock bool `hcl:"allow-insecure-unlock,optional" toml:"allow-insecure-unlock,optional"`

	// UseLightweightKDF enables a faster but less secure encryption of accounts
	UseLightweightKDF bool `hcl:"lightkdf,optional" toml:"lightkdf,optional"`

	// DisableBorWallet disables the personal wallet endpoints
	DisableBorWallet bool `hcl:"disable-bor-wallet,optional" toml:"disable-bor-wallet,optional"`
}

type DeveloperConfig struct {
	// Enabled enables the developer mode
	Enabled bool `hcl:"dev,optional" toml:"dev,optional"`

	// Period is the block period to use in developer mode
	Period uint64 `hcl:"period,optional" toml:"period,optional"`

	// Initial block gas limit
	GasLimit uint64 `hcl:"gaslimit,optional" toml:"gaslimit,optional"`
}

type ParallelEVMConfig struct {
	Enable bool `hcl:"enable,optional" toml:"enable,optional"`

	SpeculativeProcesses int `hcl:"procs,optional" toml:"procs,optional"`
}

func DefaultConfig() *Config {
	return &Config{
		Chain:                   "mainnet",
		Identity:                Hostname(),
		RequiredBlocks:          map[string]string{},
		Verbosity:               3,
		LogLevel:                "",
		EnablePreimageRecording: false,
		DataDir:                 DefaultDataDir(),
		Ancient:                 "",
		DBEngine:                "leveldb",
		KeyStoreDir:             "",
		Logging: &LoggingConfig{
			Vmodule:             "",
			Json:                false,
			Backtrace:           "",
			Debug:               false,
			EnableBlockTracking: false,
		},
		RPCBatchLimit:      100,
		RPCReturnDataLimit: 100000,
		P2P: &P2PConfig{
			MaxPeers:      50,
			MaxPendPeers:  50,
			Bind:          "0.0.0.0",
			Port:          30303,
			NoDiscover:    false,
			NAT:           "any",
			NetRestrict:   "",
			TxArrivalWait: 500 * time.Millisecond,
			Discovery: &P2PDiscovery{
				DiscoveryV4:  true,
				V5Enabled:    false,
				Bootnodes:    []string{},
				BootnodesV4:  []string{},
				BootnodesV5:  []string{},
				StaticNodes:  []string{},
				TrustedNodes: []string{},
				DNS:          []string{},
			},
		},
		Heimdall: &HeimdallConfig{
			URL:         "http://localhost:1317",
			Without:     false,
			GRPCAddress: "",
		},
		SyncMode: "full",
		GcMode:   "full",
		Snapshot: true,
		BorLogs:  false,
		TxPool: &TxPoolConfig{
			Locals:       []string{},
			NoLocals:     false,
			Journal:      "transactions.rlp",
			Rejournal:    1 * time.Hour,
			PriceLimit:   1, // geth's default
			PriceBump:    10,
			AccountSlots: 16,
			GlobalSlots:  32768,
			AccountQueue: 16,
			GlobalQueue:  32768,
			LifeTime:     3 * time.Hour,
		},
		Sealer: &SealerConfig{
			Enabled:             false,
			Etherbase:           "",
			GasCeil:             30_000_000,                  // geth's default
			GasPrice:            big.NewInt(1 * params.GWei), // geth's default
			ExtraData:           "",
			Recommit:            125 * time.Second,
			CommitInterruptFlag: true,
		},
		Gpo: &GpoConfig{
			Blocks:           20,
			Percentile:       60,
			MaxHeaderHistory: 1024,
			MaxBlockHistory:  1024,
			MaxPrice:         gasprice.DefaultMaxPrice,
			IgnorePrice:      gasprice.DefaultIgnorePrice,
		},
		JsonRPC: &JsonRPCConfig{
			IPCDisable:          false,
			IPCPath:             "",
			GasCap:              ethconfig.Defaults.RPCGasCap,
			TxFeeCap:            ethconfig.Defaults.RPCTxFeeCap,
			RPCEVMTimeout:       ethconfig.Defaults.RPCEVMTimeout,
			AllowUnprotectedTxs: false,
			EnablePersonal:      false,
			Http: &APIConfig{
				Enabled:                     false,
				Port:                        8545,
				Prefix:                      "",
				Host:                        "localhost",
				API:                         []string{"eth", "net", "web3", "txpool", "bor"},
				Cors:                        []string{"localhost"},
				VHost:                       []string{"localhost"},
				ExecutionPoolSize:           40,
				ExecutionPoolRequestTimeout: 0,
			},
			Ws: &APIConfig{
				Enabled:                     false,
				Port:                        8546,
				Prefix:                      "",
				Host:                        "localhost",
				API:                         []string{"net", "web3"},
				Origins:                     []string{"localhost"},
				ExecutionPoolSize:           40,
				ExecutionPoolRequestTimeout: 0,
			},
			Graphql: &APIConfig{
				Enabled: false,
				Cors:    []string{"localhost"},
				VHost:   []string{"localhost"},
			},
			HttpTimeout: &HttpTimeouts{
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  120 * time.Second,
			},
			Auth: &AUTHConfig{
				JWTSecret: "",
				Port:      node.DefaultAuthPort,
				Addr:      node.DefaultAuthHost,
				VHosts:    node.DefaultAuthVhosts,
			},
		},
		Ethstats: "",
		Telemetry: &TelemetryConfig{
			Enabled:               false,
			Expensive:             false,
			PrometheusAddr:        "127.0.0.1:7071",
			OpenCollectorEndpoint: "",
			InfluxDB: &InfluxDBConfig{
				V1Enabled:    false,
				Endpoint:     "",
				Database:     "",
				Username:     "",
				Password:     "",
				Tags:         map[string]string{},
				V2Enabled:    false,
				Token:        "",
				Bucket:       "",
				Organization: "",
			},
		},
		Cache: &CacheConfig{
			Cache:              1024, // geth's default (suitable for mumbai)
			PercDatabase:       50,
			PercTrie:           15,
			PercGc:             25,
			PercSnapshot:       10,
			NoPrefetch:         false,
			Preimages:          false,
			TxLookupLimit:      2350000,
			TriesInMemory:      128,
			FilterLogCacheSize: ethconfig.Defaults.FilterLogCacheSize,
			TrieTimeout:        60 * time.Minute,
			FDLimit:            0,
		},
		ExtraDB: &ExtraDBConfig{
			// These are LevelDB defaults, specifying here for clarity in code and in logging.
			// See: https://github.com/syndtr/goleveldb/blob/126854af5e6d8295ef8e8bee3040dd8380ae72e8/leveldb/opt/options.go
			LevelDbCompactionTableSize:           2, // MiB
			LevelDbCompactionTableSizeMultiplier: 1,
			LevelDbCompactionTotalSize:           10, // MiB
			LevelDbCompactionTotalSizeMultiplier: 10,
		},
		Accounts: &AccountsConfig{
			Unlock:              []string{},
			PasswordFile:        "",
			AllowInsecureUnlock: false,
			UseLightweightKDF:   false,
			DisableBorWallet:    true,
		},
		GRPC: &GRPCConfig{
			Addr: ":3131",
		},
		Developer: &DeveloperConfig{
			Enabled:  false,
			Period:   0,
			GasLimit: 11500000,
		},
		DevFakeAuthor: false,
		Pprof: &PprofConfig{
			Enabled:          false,
			Port:             6060,
			Addr:             "127.0.0.1",
			MemProfileRate:   512 * 1024,
			BlockProfileRate: 0,
			// CPUProfile:       "",
		},
		ParallelEVM: &ParallelEVMConfig{
			Enable:               true,
			SpeculativeProcesses: 8,
		},
	}
}

func (c *Config) fillBigInt() error {
	tds := []struct {
		path string
		td   **big.Int
		str  *string
	}{
		{"gpo.maxprice", &c.Gpo.MaxPrice, &c.Gpo.MaxPriceRaw},
		{"gpo.ignoreprice", &c.Gpo.IgnorePrice, &c.Gpo.IgnorePriceRaw},
		{"miner.gasprice", &c.Sealer.GasPrice, &c.Sealer.GasPriceRaw},
	}

	for _, x := range tds {
		if *x.str != "" {
			b := new(big.Int)

			var ok bool

			if strings.HasPrefix(*x.str, "0x") {
				b, ok = b.SetString((*x.str)[2:], 16)
			} else {
				b, ok = b.SetString(*x.str, 10)
			}

			if !ok {
				return fmt.Errorf("%s can't parse big int %s", x.path, *x.str)
			}

			*x.str = ""
			*x.td = b
		}
	}

	return nil
}

func (c *Config) fillTimeDurations() error {
	tds := []struct {
		path string
		td   *time.Duration
		str  *string
	}{
		{"jsonrpc.evmtimeout", &c.JsonRPC.RPCEVMTimeout, &c.JsonRPC.RPCEVMTimeoutRaw},
		{"miner.recommit", &c.Sealer.Recommit, &c.Sealer.RecommitRaw},
		{"jsonrpc.timeouts.read", &c.JsonRPC.HttpTimeout.ReadTimeout, &c.JsonRPC.HttpTimeout.ReadTimeoutRaw},
		{"jsonrpc.timeouts.write", &c.JsonRPC.HttpTimeout.WriteTimeout, &c.JsonRPC.HttpTimeout.WriteTimeoutRaw},
		{"jsonrpc.timeouts.idle", &c.JsonRPC.HttpTimeout.IdleTimeout, &c.JsonRPC.HttpTimeout.IdleTimeoutRaw},
		{"jsonrpc.ws.ep-requesttimeout", &c.JsonRPC.Ws.ExecutionPoolRequestTimeout, &c.JsonRPC.Ws.ExecutionPoolRequestTimeoutRaw},
		{"jsonrpc.http.ep-requesttimeout", &c.JsonRPC.Http.ExecutionPoolRequestTimeout, &c.JsonRPC.Http.ExecutionPoolRequestTimeoutRaw},
		{"txpool.lifetime", &c.TxPool.LifeTime, &c.TxPool.LifeTimeRaw},
		{"txpool.rejournal", &c.TxPool.Rejournal, &c.TxPool.RejournalRaw},
		{"cache.timeout", &c.Cache.TrieTimeout, &c.Cache.TrieTimeoutRaw},
		{"p2p.txarrivalwait", &c.P2P.TxArrivalWait, &c.P2P.TxArrivalWaitRaw},
	}

	for _, x := range tds {
		if x.td != nil && x.str != nil && *x.str != "" {
			d, err := time.ParseDuration(*x.str)
			if err != nil {
				return fmt.Errorf("%s can't parse time duration %s", x.path, *x.str)
			}

			*x.str = ""
			*x.td = d
		}
	}

	return nil
}

func readConfigFile(path string) (*Config, error) {
	ext := filepath.Ext(path)
	if ext == ".toml" {
		return readLegacyConfig(path)
	}

	config := &Config{
		TxPool: &TxPoolConfig{},
		Cache:  &CacheConfig{},
		Sealer: &SealerConfig{},
	}

	if err := hclsimple.DecodeFile(path, nil, config); err != nil {
		return nil, fmt.Errorf("failed to decode config file '%s': %v", path, err)
	}

	if err := config.fillBigInt(); err != nil {
		return nil, err
	}

	if err := config.fillTimeDurations(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) loadChain() error {
	chain, err := chains.GetChain(c.Chain)
	if err != nil {
		return err
	}

	c.chain = chain

	// preload some default values that depend on the chain file
	if c.P2P.Discovery.DNS == nil {
		c.P2P.Discovery.DNS = c.chain.DNS
	}

	return nil
}

//nolint:gocognit
func (c *Config) buildEth(stack *node.Node, accountManager *accounts.Manager) (*ethconfig.Config, error) {
	dbHandles, err := MakeDatabaseHandles(c.Cache.FDLimit)
	if err != nil {
		return nil, err
	}

	n := ethconfig.Defaults

	// only update for non-developer mode as we don't yet
	// have the chain object for it.
	if !c.Developer.Enabled {
		n.NetworkId = c.chain.NetworkId
		n.Genesis = c.chain.Genesis
	}

	n.HeimdallURL = c.Heimdall.URL
	n.WithoutHeimdall = c.Heimdall.Without
	n.HeimdallgRPCAddress = c.Heimdall.GRPCAddress
	n.RunHeimdall = c.Heimdall.RunHeimdall
	n.RunHeimdallArgs = c.Heimdall.RunHeimdallArgs
	n.UseHeimdallApp = c.Heimdall.UseHeimdallApp

	// Developer Fake Author for producing blocks without authorisation on bor consensus
	n.DevFakeAuthor = c.DevFakeAuthor

	// Developer Fake Author for producing blocks without authorisation on bor consensus
	n.DevFakeAuthor = c.DevFakeAuthor

	// gas price oracle
	{
		n.GPO.Blocks = int(c.Gpo.Blocks)
		n.GPO.Percentile = int(c.Gpo.Percentile)
		n.GPO.MaxHeaderHistory = uint64(c.Gpo.MaxHeaderHistory)
		n.GPO.MaxBlockHistory = uint64(c.Gpo.MaxBlockHistory)
		n.GPO.MaxPrice = c.Gpo.MaxPrice
		n.GPO.IgnorePrice = c.Gpo.IgnorePrice
	}

	n.EnablePreimageRecording = c.EnablePreimageRecording

	// txpool options
	{
		n.TxPool.NoLocals = c.TxPool.NoLocals
		n.TxPool.Journal = c.TxPool.Journal
		n.TxPool.Rejournal = c.TxPool.Rejournal
		n.TxPool.PriceLimit = c.TxPool.PriceLimit
		n.TxPool.PriceBump = c.TxPool.PriceBump
		n.TxPool.AccountSlots = c.TxPool.AccountSlots
		n.TxPool.GlobalSlots = c.TxPool.GlobalSlots
		n.TxPool.AccountQueue = c.TxPool.AccountQueue
		n.TxPool.GlobalQueue = c.TxPool.GlobalQueue
		n.TxPool.Lifetime = c.TxPool.LifeTime
	}

	// miner options
	{
		n.Miner.Recommit = c.Sealer.Recommit
		n.Miner.GasPrice = c.Sealer.GasPrice
		n.Miner.GasCeil = c.Sealer.GasCeil
		n.Miner.ExtraData = []byte(c.Sealer.ExtraData)
		n.Miner.CommitInterruptFlag = c.Sealer.CommitInterruptFlag

		if etherbase := c.Sealer.Etherbase; etherbase != "" {
			if !common.IsHexAddress(etherbase) {
				return nil, fmt.Errorf("etherbase is not an address: %s", etherbase)
			}

			n.Miner.Etherbase = common.HexToAddress(etherbase)
		}
	}

	// unlock accounts
	if len(c.Accounts.Unlock) > 0 {
		if !stack.Config().InsecureUnlockAllowed && stack.Config().ExtRPCEnabled() {
			return nil, fmt.Errorf("account unlock with HTTP access is forbidden")
		}

		ks := accountManager.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)

		passwords, err := MakePasswordListFromFile(c.Accounts.PasswordFile)
		if err != nil {
			return nil, err
		}

		if len(passwords) < len(c.Accounts.Unlock) {
			return nil, fmt.Errorf("number of passwords provided (%v) is less than number of accounts (%v) to unlock",
				len(passwords), len(c.Accounts.Unlock))
		}

		for i, account := range c.Accounts.Unlock {
			unlockAccount(ks, account, i, passwords)
		}
	}

	// update for developer mode
	if c.Developer.Enabled {
		// Get a keystore
		var ks *keystore.KeyStore
		if keystores := accountManager.Backends(keystore.KeyStoreType); len(keystores) > 0 {
			ks = keystores[0].(*keystore.KeyStore)
		}

		// Create new developer account or reuse existing one
		var (
			developer  accounts.Account
			passphrase string
			err        error
		)

		// etherbase has been set above, configuring the miner address from command line flags.
		if n.Miner.Etherbase != (common.Address{}) {
			developer = accounts.Account{Address: n.Miner.Etherbase}
		} else if accs := ks.Accounts(); len(accs) > 0 {
			developer = ks.Accounts()[0]
		} else {
			developer, err = ks.NewAccount(passphrase)
			if err != nil {
				return nil, fmt.Errorf("failed to create developer account: %v", err)
			}
		}
		if err := ks.Unlock(developer, passphrase); err != nil {
			return nil, fmt.Errorf("failed to unlock developer account: %v", err)
		}

		log.Info("Using developer account", "address", developer.Address)

		// Set the Etherbase
		c.Sealer.Etherbase = developer.Address.Hex()
		n.Miner.Etherbase = developer.Address

		// get developer mode chain config
		c.chain = chains.GetDeveloperChain(c.Developer.Period, c.Developer.GasLimit, developer.Address)

		// update the parameters
		n.NetworkId = c.chain.NetworkId
		n.Genesis = c.chain.Genesis

		// Update cache
		c.Cache.Cache = 1024

		// Update sync mode
		c.SyncMode = "full"

		// update miner gas price
		if n.Miner.GasPrice == nil {
			n.Miner.GasPrice = big.NewInt(1)
		}
	}

	// discovery (this params should be in node.Config)
	{
		n.EthDiscoveryURLs = c.P2P.Discovery.DNS
		n.SnapDiscoveryURLs = c.P2P.Discovery.DNS
	}

	// RequiredBlocks
	{
		n.RequiredBlocks = map[uint64]common.Hash{}

		for k, v := range c.RequiredBlocks {
			number, err := strconv.ParseUint(k, 0, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid required block number %s: %v", k, err)
			}

			var hash common.Hash
			if err = hash.UnmarshalText([]byte(v)); err != nil {
				return nil, fmt.Errorf("invalid required block hash %s: %v", v, err)
			}

			n.RequiredBlocks[number] = hash
		}
	}

	// cache
	{
		cache := c.Cache.Cache
		calcPerc := func(val uint64) int {
			return int(cache * (val) / 100)
		}

		// Cap the cache allowance
		mem, err := gopsutil.VirtualMemory()
		if err == nil {
			if 32<<(^uintptr(0)>>63) == 32 && mem.Total > 2*1024*1024*1024 {
				log.Warn("Lowering memory allowance on 32bit arch", "available", mem.Total/1024/1024, "addressable", 2*1024)
				mem.Total = 2 * 1024 * 1024 * 1024
			}

			allowance := mem.Total / 1024 / 1024 / 3
			if cache > allowance {
				log.Warn("Sanitizing cache to Go's GC limits", "provided", cache, "updated", allowance)
				cache = allowance
			}
		}
		// Tune the garbage collector
		gogc := math.Max(20, math.Min(100, 100/(float64(cache)/1024)))

		log.Debug("Sanitizing Go's GC trigger", "percent", int(gogc))
		godebug.SetGCPercent(int(gogc))

		n.DatabaseCache = calcPerc(c.Cache.PercDatabase)
		n.SnapshotCache = calcPerc(c.Cache.PercSnapshot)
		n.TrieCleanCache = calcPerc(c.Cache.PercTrie)
		n.TrieDirtyCache = calcPerc(c.Cache.PercGc)
		n.NoPrefetch = c.Cache.NoPrefetch
		n.Preimages = c.Cache.Preimages
		n.TxLookupLimit = c.Cache.TxLookupLimit
		n.TrieTimeout = c.Cache.TrieTimeout
		n.TriesInMemory = c.Cache.TriesInMemory
		n.FilterLogCacheSize = c.Cache.FilterLogCacheSize
	}

	// LevelDB
	{
		n.LevelDbCompactionTableSize = c.ExtraDB.LevelDbCompactionTableSize
		n.LevelDbCompactionTableSizeMultiplier = c.ExtraDB.LevelDbCompactionTableSizeMultiplier
		n.LevelDbCompactionTotalSize = c.ExtraDB.LevelDbCompactionTotalSize
		n.LevelDbCompactionTotalSizeMultiplier = c.ExtraDB.LevelDbCompactionTotalSizeMultiplier
	}

	n.RPCGasCap = c.JsonRPC.GasCap
	if n.RPCGasCap != 0 {
		log.Info("Set global gas cap", "cap", n.RPCGasCap)
	} else {
		log.Info("Global gas cap disabled")
	}

	n.RPCEVMTimeout = c.JsonRPC.RPCEVMTimeout

	n.RPCTxFeeCap = c.JsonRPC.TxFeeCap

	// sync mode. It can either be "fast", "full" or "snap". We disable
	// for now the "light" mode.
	switch c.SyncMode {
	case "full":
		n.SyncMode = downloader.FullSync
	case "snap":
		// n.SyncMode = downloader.SnapSync // TODO(snap): Uncomment when we have snap sync working
		n.SyncMode = downloader.FullSync

		log.Warn("Bor doesn't support Snap Sync yet, switching to Full Sync mode")
	default:
		return nil, fmt.Errorf("sync mode '%s' not found", c.SyncMode)
	}

	// archive mode. It can either be "archive" or "full".
	switch c.GcMode {
	case "full":
		n.NoPruning = false
	case "archive":
		n.NoPruning = true
		if !n.Preimages {
			n.Preimages = true

			log.Info("Enabling recording of key preimages since archive mode is used")
		}
	default:
		return nil, fmt.Errorf("gcmode '%s' not found", c.GcMode)
	}

	// snapshot disable check
	if !c.Snapshot {
		if n.SyncMode == downloader.SnapSync {
			log.Info("Snap sync requested, enabling --snapshot")
		} else {
			// disable snapshot
			n.TrieCleanCache += n.SnapshotCache
			n.SnapshotCache = 0
		}
	}

	n.BorLogs = c.BorLogs
	n.DatabaseHandles = dbHandles

	n.ParallelEVM.Enable = c.ParallelEVM.Enable
	n.ParallelEVM.SpeculativeProcesses = c.ParallelEVM.SpeculativeProcesses
	n.RPCReturnDataLimit = c.RPCReturnDataLimit

	if c.Ancient != "" {
		n.DatabaseFreezer = c.Ancient
	}

	n.EnableBlockTracking = c.Logging.EnableBlockTracking

	return &n, nil
}

var (
	clientIdentifier = "bor"
	gitCommit        = "" // Git SHA1 commit hash of the release (set via linker flags)
	gitDate          = "" // Git commit date YYYYMMDD of the release (set via linker flags)
)

// tries unlocking the specified account a few times.
func unlockAccount(ks *keystore.KeyStore, address string, i int, passwords []string) (accounts.Account, string) {
	account, err := utils.MakeAddress(ks, address)

	if err != nil {
		utils.Fatalf("Could not list accounts: %v", err)
	}

	for trials := 0; trials < 3; trials++ {
		prompt := fmt.Sprintf("Unlocking account %s | Attempt %d/%d", address, trials+1, 3)
		password := utils.GetPassPhraseWithList(prompt, false, i, passwords)
		err = ks.Unlock(account, password)

		if err == nil {
			log.Info("Unlocked account", "address", account.Address.Hex())
			return account, password
		}

		if err, ok := err.(*keystore.AmbiguousAddrError); ok {
			log.Info("Unlocked account", "address", account.Address.Hex())
			return ambiguousAddrRecovery(ks, err, password), password
		}

		if err != keystore.ErrDecrypt {
			// No need to prompt again if the error is not decryption-related.
			break
		}
	}
	// All trials expended to unlock account, bail out
	utils.Fatalf("Failed to unlock account %s (%v)", address, err)

	return accounts.Account{}, ""
}

func ambiguousAddrRecovery(ks *keystore.KeyStore, err *keystore.AmbiguousAddrError, auth string) accounts.Account {
	log.Warn("Multiple key files exist for", "address", err.Addr)

	for _, a := range err.Matches {
		log.Info("Multiple keys", "file", a.URL.String())
	}

	log.Info("Testing your password against all of them...")

	var match *accounts.Account

	for _, a := range err.Matches {
		if err := ks.Unlock(a, auth); err == nil {
			// nolint: gosec, exportloopref
			match = &a
			break
		}
	}

	if match == nil {
		utils.Fatalf("None of the listed files could be unlocked.")
	}

	log.Info("Your password unlocked", "key", match.URL.String())
	log.Warn("In order to avoid this warning, you need to remove the following duplicate key files:")

	for _, a := range err.Matches {
		if a != *match {
			log.Warn("Duplicate", "key", a.URL.String())
		}
	}

	return *match
}

// setNodeKey creates a node key from set command line flags, either loading it
// from a file or as a specified hex value. If neither flags were provided, this
// method returns nil and an emphemeral key is to be generated.
func getNodeKey(hex string, file string) *ecdsa.PrivateKey {
	var (
		key *ecdsa.PrivateKey
		err error
	)

	switch {
	case file != "" && hex != "":
		utils.Fatalf("Options %q and %q are mutually exclusive", file, hex)
	case file != "":
		if key, err = crypto.LoadECDSA(file); err != nil {
			utils.Fatalf("Option %q: %v", file, err)
		}

		return key
	case hex != "":
		if key, err = crypto.HexToECDSA(hex); err != nil {
			utils.Fatalf("Option %q: %v", hex, err)
		}

		return key
	}

	return nil
}

func (c *Config) buildNode() (*node.Config, error) {
	ipcPath := ""
	if !c.JsonRPC.IPCDisable {
		ipcPath = clientIdentifier + ".ipc"
		if c.JsonRPC.IPCPath != "" {
			ipcPath = c.JsonRPC.IPCPath
		}
	}

	cfg := &node.Config{
		Name:                  clientIdentifier,
		DataDir:               c.DataDir,
		DBEngine:              c.DBEngine,
		KeyStoreDir:           c.KeyStoreDir,
		UseLightweightKDF:     c.Accounts.UseLightweightKDF,
		InsecureUnlockAllowed: c.Accounts.AllowInsecureUnlock,
		Version:               params.VersionWithCommit(gitCommit, gitDate),
		IPCPath:               ipcPath,
		AllowUnprotectedTxs:   c.JsonRPC.AllowUnprotectedTxs,
		EnablePersonal:        c.JsonRPC.EnablePersonal,
		P2P: p2p.Config{
			MaxPeers:        int(c.P2P.MaxPeers),
			MaxPendingPeers: int(c.P2P.MaxPendPeers),
			ListenAddr:      c.P2P.Bind + ":" + strconv.Itoa(int(c.P2P.Port)),
			DiscoveryV4:     c.P2P.Discovery.DiscoveryV4,
			DiscoveryV5:     c.P2P.Discovery.V5Enabled,
			TxArrivalWait:   c.P2P.TxArrivalWait,
		},
		HTTPModules:         c.JsonRPC.Http.API,
		HTTPCors:            c.JsonRPC.Http.Cors,
		HTTPVirtualHosts:    c.JsonRPC.Http.VHost,
		HTTPPathPrefix:      c.JsonRPC.Http.Prefix,
		WSModules:           c.JsonRPC.Ws.API,
		WSOrigins:           c.JsonRPC.Ws.Origins,
		WSPathPrefix:        c.JsonRPC.Ws.Prefix,
		GraphQLCors:         c.JsonRPC.Graphql.Cors,
		GraphQLVirtualHosts: c.JsonRPC.Graphql.VHost,
		HTTPTimeouts: rpc.HTTPTimeouts{
			ReadTimeout:  c.JsonRPC.HttpTimeout.ReadTimeout,
			WriteTimeout: c.JsonRPC.HttpTimeout.WriteTimeout,
			IdleTimeout:  c.JsonRPC.HttpTimeout.IdleTimeout,
		},
		JWTSecret:                              c.JsonRPC.Auth.JWTSecret,
		AuthPort:                               int(c.JsonRPC.Auth.Port),
		AuthAddr:                               c.JsonRPC.Auth.Addr,
		AuthVirtualHosts:                       c.JsonRPC.Auth.VHosts,
		RPCBatchLimit:                          c.RPCBatchLimit,
		WSJsonRPCExecutionPoolSize:             c.JsonRPC.Ws.ExecutionPoolSize,
		WSJsonRPCExecutionPoolRequestTimeout:   c.JsonRPC.Ws.ExecutionPoolRequestTimeout,
		HTTPJsonRPCExecutionPoolSize:           c.JsonRPC.Http.ExecutionPoolSize,
		HTTPJsonRPCExecutionPoolRequestTimeout: c.JsonRPC.Http.ExecutionPoolRequestTimeout,
	}

	if c.P2P.NetRestrict != "" {
		list, err := netutil.ParseNetlist(c.P2P.NetRestrict)
		if err != nil {
			utils.Fatalf("Option %q: %v", c.P2P.NetRestrict, err)
		}

		cfg.P2P.NetRestrict = list
	}

	key := getNodeKey(c.P2P.NodeKeyHex, c.P2P.NodeKey)
	if key != nil {
		cfg.P2P.PrivateKey = key
	}

	// dev mode
	if c.Developer.Enabled {
		cfg.UseLightweightKDF = true

		// disable p2p networking
		c.P2P.NoDiscover = true
		cfg.P2P.ListenAddr = ""
		cfg.P2P.NoDial = true
		cfg.P2P.DiscoveryV5 = false

		// enable JsonRPC HTTP API
		c.JsonRPC.Http.Enabled = true
		cfg.HTTPModules = []string{"admin", "debug", "eth", "miner", "net", "personal", "txpool", "web3", "bor"}
	}

	// enable jsonrpc endpoints
	{
		if c.JsonRPC.Http.Enabled {
			cfg.HTTPHost = c.JsonRPC.Http.Host
			cfg.HTTPPort = int(c.JsonRPC.Http.Port)
		}

		if c.JsonRPC.Ws.Enabled {
			cfg.WSHost = c.JsonRPC.Ws.Host
			cfg.WSPort = int(c.JsonRPC.Ws.Port)
		}
	}

	natif, err := nat.Parse(c.P2P.NAT)
	if err != nil {
		return nil, fmt.Errorf("wrong 'nat' flag: %v", err)
	}

	cfg.P2P.NAT = natif

	// only check for non-developer modes
	if !c.Developer.Enabled {
		// Discovery
		// Append the bootnodes defined with those hardcoded in the config file
		bootnodes := c.P2P.Discovery.Bootnodes
		if c.chain != nil {
			bootnodes = append(bootnodes, c.chain.Bootnodes...)
		}

		if cfg.P2P.BootstrapNodes, err = parseBootnodes(bootnodes); err != nil {
			return nil, err
		}

		if cfg.P2P.BootstrapNodesV5, err = parseBootnodes(c.P2P.Discovery.BootnodesV5); err != nil {
			return nil, err
		}

		if cfg.P2P.StaticNodes, err = parseBootnodes(c.P2P.Discovery.StaticNodes); err != nil {
			return nil, err
		}

		if len(cfg.P2P.StaticNodes) == 0 {
			cfg.P2P.StaticNodes = cfg.StaticNodes()
		}

		if cfg.P2P.TrustedNodes, err = parseBootnodes(c.P2P.Discovery.TrustedNodes); err != nil {
			return nil, err
		}

		if len(cfg.P2P.TrustedNodes) == 0 {
			cfg.P2P.TrustedNodes = cfg.TrustedNodes()
		}
	}

	if c.P2P.NoDiscover {
		// Disable peer discovery
		cfg.P2P.NoDiscovery = true
	}

	return cfg, nil
}

func (c *Config) Merge(cc ...*Config) error {
	for _, elem := range cc {
		if err := mergo.Merge(c, elem, mergo.WithOverwriteWithEmptyValue); err != nil {
			return fmt.Errorf("failed to merge configurations: %v", err)
		}
	}

	return nil
}

func MakeDatabaseHandles(max int) (int, error) {
	limit, err := fdlimit.Maximum()
	if err != nil {
		return -1, err
	}

	switch {
	case max == 0:
		// User didn't specify a meaningful value, use system limits
	case max < 128:
		// User specified something unhealthy, just use system defaults
		log.Error("File descriptor limit invalid (<128)", "had", max, "updated", limit)
	case max > limit:
		// User requested more than the OS allows, notify that we can't allocate it
		log.Warn("Requested file descriptors denied by OS", "req", max, "limit", limit)
	default:
		// User limit is meaningful and within allowed range, use that
		limit = max
	}

	raised, err := fdlimit.Raise(uint64(limit))
	if err != nil {
		return -1, err
	}

	return int(raised / 2), nil // Leave half for networking and other stuff
}

func parseBootnodes(urls []string) ([]*enode.Node, error) {
	dst := []*enode.Node{}

	for _, url := range urls {
		if url != "" {
			node, err := enode.Parse(enode.ValidSchemes, url)
			if err != nil {
				return nil, fmt.Errorf("invalid bootstrap url '%s': %v", url, err)
			}

			dst = append(dst, node)
		}
	}

	return dst, nil
}

func DefaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home, _ := homedir.Dir()
	if home == "" {
		// we cannot guess a stable location
		return ""
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Bor")
	case "windows":
		appdata := os.Getenv("LOCALAPPDATA")
		if appdata == "" {
			// Windows XP and below don't have LocalAppData.
			panic("environment variable LocalAppData is undefined")
		}

		return filepath.Join(appdata, "Bor")
	default:
		return filepath.Join(home, ".bor")
	}
}

func Hostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "bor"
	}

	return hostname
}

func MakePasswordListFromFile(path string) ([]string, error) {
	text, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read password file: %v", err)
	}

	lines := strings.Split(string(text), "\n")

	// Sanitise DOS line endings.
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}

	return lines, nil
}
