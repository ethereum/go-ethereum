package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	godebug "runtime/debug"

	"github.com/ethereum/go-ethereum/command/server/chains"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/gasprice"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/params"
	"github.com/imdario/mergo"
	"github.com/mitchellh/go-homedir"
	gopsutil "github.com/shirou/gopsutil/mem"
)

type Config struct {
	chain *chains.Chain

	Chain     string            `json:"chain"`
	Debug     bool              `json:""`
	Whitelist map[string]string `json:""`
	LogLevel  string            `json:"log-level"`
	DataDir   string            `json:"data-dir"`
	P2P       *P2PConfig        `json:""`
	SyncMode  string            `json:"sync-mode"`
	GcMode    string            `json:"gc-mode"`
	Snapshot  bool              `json:"snapshot"`
	Heimdall  *HeimdallConfig   `json:"heimdall"`
	TxPool    *TxPoolConfig     `json:"txpool"`
	Sealer    *SealerConfig     `json:"sealer"`
	JsonRPC   *JsonRPCConfig    `json:"jsonrpc"`
	Gpo       *GpoConfig        `json:"gpo"`
	Ethstats  string            `json:"ethstats"`
	Metrics   *MetricsConfig    `json:""`
	Cache     *CacheConfig      `json:""`
	Accounts  *AccountsConfig   `json:""`
}

type P2PConfig struct {
	MaxPeers     uint64        `json:"max_peers"`
	MaxPendPeers uint64        `json:"max_pend_peers"`
	Bind         string        `json:""`
	Port         uint64        `json:""`
	NoDiscover   bool          `json:""`
	NAT          string        `json:""`
	Discovery    *P2PDiscovery `json:""`
}

type P2PDiscovery struct {
	V5Enabled    bool     `json:""`
	Bootnodes    []string `json:""`
	BootnodesV4  []string `json:""`
	BootnodesV5  []string `json:""`
	StaticNodes  []string `json:""`
	TrustedNodes []string `json:""`
	DNS          []string `json:""`
}

type HeimdallConfig struct {
	URL     string `json:""`
	Without bool   `json:""`
}

type TxPoolConfig struct {
	Locals       []string      `json:""`
	NoLocals     bool          `json:""`
	Journal      string        `json:""`
	Rejournal    time.Duration `json:""`
	PriceLimit   uint64        `json:""`
	PriceBump    uint64        `json:""`
	AccountSlots uint64        `json:""`
	GlobalSlots  uint64        `json:""`
	AccountQueue uint64        `json:""`
	GlobalQueue  uint64        `json:""`
	LifeTime     time.Duration `json:""`
}

type SealerConfig struct {
	Enabled   bool     `json:""`
	Etherbase string   `json:""`
	ExtraData string   `json:""`
	GasCeil   uint64   `json:""`
	GasPrice  *big.Int `json:""`
}

type JsonRPCConfig struct {
	IPCDisable bool   `json:""`
	IPCPath    string `json:""`

	Modules []string `json:""`
	VHost   []string `json:""`
	Cors    []string `json:""`

	GasCap   uint64  `json:""`
	TxFeeCap float64 `json:""`

	Http    *APIConfig `json:""`
	Ws      *APIConfig `json:""`
	Graphql *APIConfig `json:""`
}

type APIConfig struct {
	Enabled bool   `json:""`
	Port    uint64 `json:""`
	Prefix  string `json:""`
	Host    string `json:""`
}

type GpoConfig struct {
	Blocks      uint64   `json:""`
	Percentile  uint64   `json:""`
	MaxPrice    *big.Int `json:""`
	IgnorePrice *big.Int `json:""`
}

type MetricsConfig struct {
	Enabled   bool            `json:""`
	Expensive bool            `json:""`
	InfluxDB  *InfluxDBConfig `json:""`
}

type InfluxDBConfig struct {
	V1Enabled    bool              `json:""`
	Endpoint     string            `json:""`
	Database     string            `json:""`
	Username     string            `json:""`
	Password     string            `json:""`
	Tags         map[string]string `json:""`
	V2Enabled    bool              `json:""`
	Token        string            `json:""`
	Bucket       string            `json:""`
	Organization string            `json:""`
}

type CacheConfig struct {
	Cache         uint64        `json:""`
	PercGc        uint64        `json:""`
	PercSnapshot  uint64        `json:""`
	PercDatabase  uint64        `json:""`
	PercTrie      uint64        `json:""`
	Journal       string        `json:""`
	Rejournal     time.Duration `json:""`
	NoPrefetch    bool          `json:""`
	Preimages     bool          `json:""`
	TxLookupLimit uint64        `json:""`
}

type AccountsConfig struct {
	Unlock              []string `json:""`
	PasswordFile        string   `json:""`
	AllowInsecureUnlock bool     `json:""`
	UseLightweightKDF   bool     `json:""`
}

func DefaultConfig() *Config {
	return &Config{
		Chain:     "mainnet",
		Debug:     false,
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
		SyncMode: "fast",
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
			Modules:    []string{"web3", "net"},
			Cors:       []string{"*"},
			VHost:      []string{"*"},
			GasCap:     ethconfig.Defaults.RPCGasCap,
			TxFeeCap:   ethconfig.Defaults.RPCTxFeeCap,
			Http: &APIConfig{
				Enabled: false,
				Port:    8545,
				Prefix:  "",
				Host:    "localhost",
			},
			Ws: &APIConfig{
				Enabled: false,
				Port:    8546,
				Prefix:  "",
				Host:    "localhost",
			},
			Graphql: &APIConfig{
				Enabled: false,
			},
		},
		Ethstats: "",
		Metrics: &MetricsConfig{
			Enabled:   false,
			Expensive: false,
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
	}
}

func readConfigFile(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// TODO: Use hcl as config format
	ext := filepath.Ext(path)
	if ext == ".toml" {
		return readLegacyConfig(data)
	}

	// read configuration file
	if ext != ".json" {
		return nil, fmt.Errorf("suffix of %s is not json", path)
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *Config) loadChain() error {
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
	if c.Chain != "mainnet" {
		c.Cache.Cache = 4096
	} else {
		c.Cache.Cache = 1024
	}
	return nil
}

func (c *Config) buildEth() (*ethconfig.Config, error) {
	dbHandles, err := makeDatabaseHandles()
	if err != nil {
		return nil, err
	}
	n := ethconfig.Defaults
	n.NetworkId = c.chain.NetworkId
	n.Genesis = c.chain.Genesis
	n.HeimdallURL = c.Heimdall.URL
	n.WithoutHeimdall = c.Heimdall.Without

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

	// discovery (this params should be in node.Config)
	{
		n.EthDiscoveryURLs = c.P2P.Discovery.DNS
		n.SnapDiscoveryURLs = c.P2P.Discovery.DNS
	}

	// whitelist
	{
		n.Whitelist = map[uint64]common.Hash{}
		for k, v := range c.Whitelist {
			number, err := strconv.ParseUint(k, 0, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid whitelist block number %s: %v", k, err)
			}
			var hash common.Hash
			if err = hash.UnmarshalText([]byte(v)); err != nil {
				return nil, fmt.Errorf("invalid whitelist hash %s: %v", v, err)
			}
			n.Whitelist[number] = hash
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
	case "fast":
		n.SyncMode = downloader.FastSync
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
		HTTPModules:         c.JsonRPC.Modules,
		HTTPCors:            c.JsonRPC.Cors,
		HTTPVirtualHosts:    c.JsonRPC.VHost,
		HTTPPathPrefix:      c.JsonRPC.Http.Prefix,
		WSModules:           c.JsonRPC.Modules,
		WSOrigins:           c.JsonRPC.Cors,
		WSPathPrefix:        c.JsonRPC.Ws.Prefix,
		GraphQLCors:         c.JsonRPC.Cors,
		GraphQLVirtualHosts: c.JsonRPC.VHost,
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
