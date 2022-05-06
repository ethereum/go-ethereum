package server

import (
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	godebug "runtime/debug"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/gasprice"
	"github.com/ethereum/go-ethereum/internal/cli/server/chains"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/params"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/imdario/mergo"
	"github.com/mitchellh/go-homedir"
	gopsutil "github.com/shirou/gopsutil/mem"
)

type Config struct {
	chain *chains.Chain

	// Chain is the chain to sync with
	Chain string `hcl:"chain,optional"`

	// Name, or identity of the node
	Name string `hcl:"name,optional"`

	// Whitelist is a list of required (block number, hash) pairs to accept
	Whitelist map[string]string `hcl:"whitelist,optional"`

	// LogLevel is the level of the logs to put out
	LogLevel string `hcl:"log-level,optional"`

	// DataDir is the directory to store the state in
	DataDir string `hcl:"data-dir,optional"`

	// SyncMode selects the sync protocol
	SyncMode string `hcl:"sync-mode,optional"`

	// GcMode selects the garbage collection mode for the trie
	GcMode string `hcl:"gc-mode,optional"`

	// XXX
	Snapshot bool `hcl:"snapshot,optional"`

	// Ethstats is the address of the ethstats server to send telemetry
	Ethstats string `hcl:"ethstats,optional"`

	// P2P has the p2p network related settings
	P2P *P2PConfig `hcl:"p2p,block"`

	// Heimdall has the heimdall connection related settings
	Heimdall *HeimdallConfig `hcl:"heimdall,block"`

	// TxPool has the transaction pool related settings
	TxPool *TxPoolConfig `hcl:"txpool,block"`

	// Sealer has the validator related settings
	Sealer *SealerConfig `hcl:"sealer,block"`

	// JsonRPC has the json-rpc related settings
	JsonRPC *JsonRPCConfig `hcl:"jsonrpc,block"`

	// Gpo has the gas price oracle related settings
	Gpo *GpoConfig `hcl:"gpo,block"`

	// Telemetry has the telemetry related settings
	Telemetry *TelemetryConfig `hcl:"telemetry,block"`

	// Cache has the cache related settings
	Cache *CacheConfig `hcl:"cache,block"`

	// Account has the validator account related settings
	Accounts *AccountsConfig `hcl:"accounts,block"`

	// GRPC has the grpc server related settings
	GRPC *GRPCConfig

	// Developer has the developer mode related settings
	Developer *DeveloperConfig
}

type P2PConfig struct {
	// MaxPeers sets the maximum number of connected peers
	MaxPeers uint64 `hcl:"max-peers,optional"`

	// MaxPendPeers sets the maximum number of pending connected peers
	MaxPendPeers uint64 `hcl:"max-pend-peers,optional"`

	// Bind is the bind address
	Bind string `hcl:"bind,optional"`

	// Port is the port number
	Port uint64 `hcl:"port,optional"`

	// NoDiscover is used to disable discovery
	NoDiscover bool `hcl:"no-discover,optional"`

	// NAT it used to set NAT options
	NAT string `hcl:"nat,optional"`

	// Discovery has the p2p discovery related settings
	Discovery *P2PDiscovery `hcl:"discovery,block"`
}

type P2PDiscovery struct {
	// V5Enabled is used to enable disc v5 discovery mode
	V5Enabled bool `hcl:"v5-enabled,optional"`

	// Bootnodes is the list of initial bootnodes
	Bootnodes []string `hcl:"bootnodes,optional"`

	// BootnodesV4 is the list of initial v4 bootnodes
	BootnodesV4 []string `hcl:"bootnodesv4,optional"`

	// BootnodesV5 is the list of initial v5 bootnodes
	BootnodesV5 []string `hcl:"bootnodesv5,optional"`

	// StaticNodes is the list of static nodes
	StaticNodes []string `hcl:"static-nodes,optional"`

	// TrustedNodes is the list of trusted nodes
	TrustedNodes []string `hcl:"trusted-nodes,optional"`

	// DNS is the list of enrtree:// URLs which will be queried for nodes to connect to
	DNS []string `hcl:"dns,optional"`
}

type HeimdallConfig struct {
	// URL is the url of the heimdall server
	URL string `hcl:"url,optional"`

	// Without is used to disable remote heimdall during testing
	Without bool `hcl:"without,optional"`
}

type TxPoolConfig struct {
	// Locals are the addresses that should be treated by default as local
	Locals []string `hcl:"locals,optional"`

	// NoLocals enables whether local transaction handling should be disabled
	NoLocals bool `hcl:"no-locals,optional"`

	// Journal is the path to store local transactions to survive node restarts
	Journal string `hcl:"journal,optional"`

	// Rejournal is the time interval to regenerate the local transaction journal
	Rejournal    time.Duration
	RejournalRaw string `hcl:"rejournal,optional"`

	// PriceLimit is the minimum gas price to enforce for acceptance into the pool
	PriceLimit uint64 `hcl:"price-limit,optional"`

	// PriceBump is the minimum price bump percentage to replace an already existing transaction (nonce)
	PriceBump uint64 `hcl:"price-bump,optional"`

	// AccountSlots is the number of executable transaction slots guaranteed per account
	AccountSlots uint64 `hcl:"account-slots,optional"`

	// GlobalSlots is the maximum number of executable transaction slots for all accounts
	GlobalSlots uint64 `hcl:"global-slots,optional"`

	// AccountQueue is the maximum number of non-executable transaction slots permitted per account
	AccountQueue uint64 `hcl:"account-queue,optional"`

	// GlobalQueueis the maximum number of non-executable transaction slots for all accounts
	GlobalQueue uint64 `hcl:"global-queue,optional"`

	// Lifetime is the maximum amount of time non-executable transaction are queued
	LifeTime    time.Duration
	LifeTimeRaw string `hcl:"lifetime,optional"`
}

type SealerConfig struct {
	// Enabled is used to enable validator mode
	Enabled bool `hcl:"enabled,optional"`

	// Etherbase is the address of the validator
	Etherbase string `hcl:"etherbase,optional"`

	// ExtraData is the block extra data set by the miner
	ExtraData string `hcl:"extra-data,optional"`

	// GasCeil is the target gas ceiling for mined blocks.
	GasCeil uint64 `hcl:"gas-ceil,optional"`

	// GasPrice is the minimum gas price for mining a transaction
	GasPrice    *big.Int
	GasPriceRaw string `hcl:"gas-price,optional"`
}

type JsonRPCConfig struct {
	// IPCDisable enables whether ipc is enabled or not
	IPCDisable bool `hcl:"ipc-disable,optional"`

	// IPCPath is the path of the ipc endpoint
	IPCPath string `hcl:"ipc-path,optional"`

	// VHost is the list of valid virtual hosts
	VHost []string `hcl:"vhost,optional"`

	// Cors is the list of Cors endpoints
	Cors []string `hcl:"cors,optional"`

	// GasCap is the global gas cap for eth-call variants.
	GasCap uint64 `hcl:"gas-cap,optional"`

	// TxFeeCap is the global transaction fee cap for send-transaction variants
	TxFeeCap float64 `hcl:"tx-fee-cap,optional"`

	// Http has the json-rpc http related settings
	Http *APIConfig `hcl:"http,block"`

	// Http has the json-rpc websocket related settings
	Ws *APIConfig `hcl:"ws,block"`

	// Http has the json-rpc graphql related settings
	Graphql *APIConfig `hcl:"graphql,block"`
}

type GRPCConfig struct {
	// Addr is the bind address for the grpc rpc server
	Addr string
}

type APIConfig struct {
	// Enabled selects whether the api is enabled
	Enabled bool `hcl:"enabled,optional"`

	// Port is the port number for this api
	Port uint64 `hcl:"port,optional"`

	// Prefix is the http prefix to expose this api
	Prefix string `hcl:"prefix,optional"`

	// Host is the address to bind the api
	Host string `hcl:"host,optional"`

	// Modules is the list of enabled api modules
	Modules []string `hcl:"modules,optional"`
}

type GpoConfig struct {
	// Blocks is the number of blocks to track to compute the price oracle
	Blocks uint64 `hcl:"blocks,optional"`

	// Percentile sets the weights to new blocks
	Percentile uint64 `hcl:"percentile,optional"`

	// MaxPrice is an upper bound gas price
	MaxPrice    *big.Int
	MaxPriceRaw string `hcl:"max-price,optional"`

	// IgnorePrice is a lower bound gas price
	IgnorePrice    *big.Int
	IgnorePriceRaw string `hcl:"ignore-price,optional"`
}

type TelemetryConfig struct {
	// Enabled enables metrics
	Enabled bool `hcl:"enabled,optional"`

	// Expensive enables expensive metrics
	Expensive bool `hcl:"expensive,optional"`

	// InfluxDB has the influxdb related settings
	InfluxDB *InfluxDBConfig `hcl:"influx,block"`

	// Prometheus Address
	PrometheusAddr string `hcl:"prometheus-addr,optional"`

	// Open collector endpoint
	OpenCollectorEndpoint string `hcl:"opencollector-endpoint,optional"`
}

type InfluxDBConfig struct {
	// V1Enabled enables influx v1 mode
	V1Enabled bool `hcl:"v1-enabled,optional"`

	// Endpoint is the url endpoint of the influxdb service
	Endpoint string `hcl:"endpoint,optional"`

	// Database is the name of the database in Influxdb to store the metrics.
	Database string `hcl:"database,optional"`

	// Enabled is the username to authorize access to Influxdb
	Username string `hcl:"username,optional"`

	// Password is the password to authorize access to Influxdb
	Password string `hcl:"password,optional"`

	// Tags are tags attaches to all generated metrics
	Tags map[string]string `hcl:"tags,optional"`

	// Enabled enables influx v2 mode
	V2Enabled bool `hcl:"v2-enabled,optional"`

	// Token is the token to authorize access to Influxdb V2.
	Token string `hcl:"token,optional"`

	// Bucket is the bucket to store metrics in Influxdb V2.
	Bucket string `hcl:"bucket,optional"`

	// Organization is the name of the organization for Influxdb V2.
	Organization string `hcl:"organization,optional"`
}

type CacheConfig struct {
	// Cache is the amount of cache of the node
	Cache uint64 `hcl:"cache,optional"`

	// PercGc is percentage of cache used for garbage collection
	PercGc uint64 `hcl:"perc-gc,optional"`

	// PercSnapshot is percentage of cache used for snapshots
	PercSnapshot uint64 `hcl:"perc-snapshot,optional"`

	// PercDatabase is percentage of cache used for the database
	PercDatabase uint64 `hcl:"perc-database,optional"`

	// PercTrie is percentage of cache used for the trie
	PercTrie uint64 `hcl:"perc-trie,optional"`

	// Journal is the disk journal directory for trie cache to survive node restarts
	Journal string `hcl:"journal,optional"`

	// Rejournal is the time interval to regenerate the journal for clean cache
	Rejournal    time.Duration
	RejournalRaw string `hcl:"rejournal,optional"`

	// NoPrefetch is used to disable prefetch of tries
	NoPrefetch bool `hcl:"no-prefetch,optional"`

	// Preimages is used to enable the track of hash preimages
	Preimages bool `hcl:"preimages,optional"`

	// TxLookupLimit sets the maximum number of blocks from head whose tx indices are reserved.
	TxLookupLimit uint64 `hcl:"tx-lookup-limit,optional"`
}

type AccountsConfig struct {
	// Unlock is the list of addresses to unlock in the node
	Unlock []string `hcl:"unlock,optional"`

	// PasswordFile is the file where the account passwords are stored
	PasswordFile string `hcl:"password-file,optional"`

	// AllowInsecureUnlock allows user to unlock accounts in unsafe http environment.
	AllowInsecureUnlock bool `hcl:"allow-insecure-unlock,optional"`

	// UseLightweightKDF enables a faster but less secure encryption of accounts
	UseLightweightKDF bool `hcl:"use-lightweight-kdf,optional"`
}

type DeveloperConfig struct {
	// Enabled enables the developer mode
	Enabled bool `hcl:"dev,optional"`

	// Period is the block period to use in developer mode
	Period uint64 `hcl:"period,optional"`
}

func DefaultConfig() *Config {
	return &Config{
		Chain:     "mainnet",
		Name:      Hostname(),
		Whitelist: map[string]string{},
		LogLevel:  "INFO",
		DataDir:   defaultDataDir(),
		P2P: &P2PConfig{
			MaxPeers:     30,
			MaxPendPeers: 50,
			Bind:         "0.0.0.0",
			Port:         30303,
			NoDiscover:   false,
			NAT:          "any",
			Discovery: &P2PDiscovery{
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
			URL:     "http://localhost:1317",
			Without: false,
		},
		SyncMode: "full",
		GcMode:   "full",
		Snapshot: true,
		TxPool: &TxPoolConfig{
			Locals:       []string{},
			NoLocals:     false,
			Journal:      "",
			Rejournal:    time.Duration(1 * time.Hour),
			PriceLimit:   1,
			PriceBump:    10,
			AccountSlots: 16,
			GlobalSlots:  4096,
			AccountQueue: 64,
			GlobalQueue:  1024,
			LifeTime:     time.Duration(3 * time.Hour),
		},
		Sealer: &SealerConfig{
			Enabled:   false,
			Etherbase: "",
			GasCeil:   8000000,
			GasPrice:  big.NewInt(params.GWei),
			ExtraData: "",
		},
		Gpo: &GpoConfig{
			Blocks:      20,
			Percentile:  60,
			MaxPrice:    gasprice.DefaultMaxPrice,
			IgnorePrice: gasprice.DefaultIgnorePrice,
		},
		JsonRPC: &JsonRPCConfig{
			IPCDisable: false,
			IPCPath:    "",
			Cors:       []string{"*"},
			VHost:      []string{"*"},
			GasCap:     ethconfig.Defaults.RPCGasCap,
			TxFeeCap:   ethconfig.Defaults.RPCTxFeeCap,
			Http: &APIConfig{
				Enabled: false,
				Port:    8545,
				Prefix:  "",
				Host:    "localhost",
				Modules: []string{"web3", "net"},
			},
			Ws: &APIConfig{
				Enabled: false,
				Port:    8546,
				Prefix:  "",
				Host:    "localhost",
				Modules: []string{"web3", "net"},
			},
			Graphql: &APIConfig{
				Enabled: false,
			},
		},
		Ethstats: "",
		Telemetry: &TelemetryConfig{
			Enabled:               false,
			Expensive:             false,
			PrometheusAddr:        "",
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
			Cache:         1024,
			PercDatabase:  50,
			PercTrie:      15,
			PercGc:        25,
			PercSnapshot:  10,
			Journal:       "triecache",
			Rejournal:     60 * time.Minute,
			NoPrefetch:    false,
			Preimages:     false,
			TxLookupLimit: 2350000,
		},
		Accounts: &AccountsConfig{
			Unlock:              []string{},
			PasswordFile:        "",
			AllowInsecureUnlock: false,
			UseLightweightKDF:   false,
		},
		GRPC: &GRPCConfig{
			Addr: ":3131",
		},
		Developer: &DeveloperConfig{
			Enabled: false,
			Period:  0,
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
		{"sealer.gasprice", &c.Sealer.GasPrice, &c.Sealer.GasPriceRaw},
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
		{"txpool.lifetime", &c.TxPool.LifeTime, &c.TxPool.LifeTimeRaw},
		{"txpool.rejournal", &c.TxPool.Rejournal, &c.TxPool.RejournalRaw},
		{"cache.rejournal", &c.Cache.Rejournal, &c.Cache.RejournalRaw},
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
		// read file and apply the legacy config
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return readLegacyConfig(data)
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
	if c.Developer.Enabled {
		return nil
	}
	chain, ok := chains.GetChain(c.Chain)
	if !ok {
		return fmt.Errorf("chain '%s' not found", c.Chain)
	}
	c.chain = chain

	// preload some default values that depend on the chain file
	if c.P2P.Discovery.DNS == nil {
		c.P2P.Discovery.DNS = c.chain.DNS
	}

	// depending on the chain we have different cache values
	if c.Chain == "mainnet" {
		c.Cache.Cache = 4096
	} else {
		c.Cache.Cache = 1024
	}
	return nil
}

func (c *Config) buildEth(stack *node.Node) (*ethconfig.Config, error) {
	dbHandles, err := makeDatabaseHandles()
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

	// gas price oracle
	{
		n.GPO.Blocks = int(c.Gpo.Blocks)
		n.GPO.Percentile = int(c.Gpo.Percentile)
		n.GPO.MaxPrice = c.Gpo.MaxPrice
		n.GPO.IgnorePrice = c.Gpo.IgnorePrice
	}

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
		n.Miner.GasPrice = c.Sealer.GasPrice
		n.Miner.GasCeil = c.Sealer.GasCeil
		n.Miner.ExtraData = []byte(c.Sealer.ExtraData)

		if etherbase := c.Sealer.Etherbase; etherbase != "" {
			if !common.IsHexAddress(etherbase) {
				return nil, fmt.Errorf("etherbase is not an address: %s", etherbase)
			}
			n.Miner.Etherbase = common.HexToAddress(etherbase)
		}
	}

	// update for developer mode
	if c.Developer.Enabled {
		// Get a keystore
		var ks *keystore.KeyStore
		if keystores := stack.AccountManager().Backends(keystore.KeyStoreType); len(keystores) > 0 {
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

		// get developer mode chain config
		c.chain = chains.GetDeveloperChain(c.Developer.Period, developer.Address)

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

	// whitelist
	{
		n.PeerRequiredBlocks = map[uint64]common.Hash{}
		for k, v := range c.Whitelist {
			number, err := strconv.ParseUint(k, 0, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid whitelist block number %s: %v", k, err)
			}
			var hash common.Hash
			if err = hash.UnmarshalText([]byte(v)); err != nil {
				return nil, fmt.Errorf("invalid whitelist hash %s: %v", v, err)
			}
			n.PeerRequiredBlocks[number] = hash
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
			allowance := uint64(mem.Total / 1024 / 1024 / 3)
			if cache > allowance {
				log.Warn("Sanitizing cache to Go's GC limits", "provided", cache, "updated", allowance)
				cache = allowance
			}
		}
		// Tune the garbage collector
		gogc := math.Max(20, math.Min(100, 100/(float64(cache)/1024)))

		log.Debug("Sanitizing Go's GC trigger", "percent", int(gogc))
		godebug.SetGCPercent(int(gogc))

		n.TrieCleanCacheJournal = c.Cache.Journal
		n.TrieCleanCacheRejournal = c.Cache.Rejournal
		n.DatabaseCache = calcPerc(c.Cache.PercDatabase)
		n.SnapshotCache = calcPerc(c.Cache.PercSnapshot)
		n.TrieCleanCache = calcPerc(c.Cache.PercTrie)
		n.TrieDirtyCache = calcPerc(c.Cache.PercGc)
		n.NoPrefetch = c.Cache.NoPrefetch
		n.Preimages = c.Cache.Preimages
		n.TxLookupLimit = c.Cache.TxLookupLimit
	}

	n.RPCGasCap = c.JsonRPC.GasCap
	if n.RPCGasCap != 0 {
		log.Info("Set global gas cap", "cap", n.RPCGasCap)
	} else {
		log.Info("Global gas cap disabled")
	}
	n.RPCTxFeeCap = c.JsonRPC.TxFeeCap

	// sync mode. It can either be "fast", "full" or "snap". We disable
	// for now the "light" mode.
	switch c.SyncMode {
	case "full":
		n.SyncMode = downloader.FullSync
	case "snap":
		n.SyncMode = downloader.SnapSync
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
	if c.Snapshot {
		if n.SyncMode == downloader.SnapSync {
			log.Info("Snap sync requested, enabling --snapshot")
		} else {
			// disable snapshot
			n.TrieCleanCache += n.SnapshotCache
			n.SnapshotCache = 0
		}
	}

	n.DatabaseHandles = dbHandles
	return &n, nil
}

var (
	clientIdentifier = "bor"
	gitCommit        = "" // Git SHA1 commit hash of the release (set via linker flags)
	gitDate          = "" // Git commit date YYYYMMDD of the release (set via linker flags)
)

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
		UseLightweightKDF:     c.Accounts.UseLightweightKDF,
		InsecureUnlockAllowed: c.Accounts.AllowInsecureUnlock,
		Version:               params.VersionWithCommit(gitCommit, gitDate),
		IPCPath:               ipcPath,
		P2P: p2p.Config{
			MaxPeers:        int(c.P2P.MaxPeers),
			MaxPendingPeers: int(c.P2P.MaxPendPeers),
			ListenAddr:      c.P2P.Bind + ":" + strconv.Itoa(int(c.P2P.Port)),
			DiscoveryV5:     c.P2P.Discovery.V5Enabled,
		},
		HTTPModules:         c.JsonRPC.Http.Modules,
		HTTPCors:            c.JsonRPC.Cors,
		HTTPVirtualHosts:    c.JsonRPC.VHost,
		HTTPPathPrefix:      c.JsonRPC.Http.Prefix,
		WSModules:           c.JsonRPC.Ws.Modules,
		WSOrigins:           c.JsonRPC.Cors,
		WSPathPrefix:        c.JsonRPC.Ws.Prefix,
		GraphQLCors:         c.JsonRPC.Cors,
		GraphQLVirtualHosts: c.JsonRPC.VHost,
	}

	// dev mode
	if c.Developer.Enabled {
		cfg.UseLightweightKDF = true

		// disable p2p networking
		c.P2P.NoDiscover = true
		cfg.P2P.ListenAddr = ""
		cfg.P2P.NoDial = true
		cfg.P2P.DiscoveryV5 = false
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
		// if no bootnodes are defined, use the ones from the chain file.
		bootnodes := c.P2P.Discovery.Bootnodes
		if len(bootnodes) == 0 {
			bootnodes = c.chain.Bootnodes
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
		if cfg.P2P.TrustedNodes, err = parseBootnodes(c.P2P.Discovery.TrustedNodes); err != nil {
			return nil, err
		}
	}

	if c.P2P.NoDiscover {
		// Disable networking, for now, we will not even allow incomming connections
		cfg.P2P.MaxPeers = 0
		cfg.P2P.NoDiscovery = true
	}
	return cfg, nil
}

func (c *Config) Merge(cc ...*Config) error {
	for _, elem := range cc {
		if err := mergo.Merge(c, elem, mergo.WithOverride, mergo.WithAppendSlice); err != nil {
			return fmt.Errorf("failed to merge configurations: %v", err)
		}
	}
	return nil
}

func makeDatabaseHandles() (int, error) {
	limit, err := fdlimit.Maximum()
	if err != nil {
		return -1, err
	}
	raised, err := fdlimit.Raise(uint64(limit))
	if err != nil {
		return -1, err
	}
	return int(raised / 2), nil
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

func defaultDataDir() string {
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
