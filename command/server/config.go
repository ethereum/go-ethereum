package server

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/command/server/chains"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"github.com/imdario/mergo"
)

func stringPtr(s string) *string {
	return &s
}

func uint64Ptr(i uint64) *uint64 {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

func durPtr(d time.Duration) *time.Duration {
	return &d
}

type Config struct {
	chain *chains.Chain

	Chain    *string
	Debug    *bool
	LogLevel *string
	DataDir  *string
	P2P      *P2PConfig
	SyncMode *string
	EthStats *string
	TxPool   *TxPoolConfig
	Sealer   *SealerConfig
}

type P2PConfig struct {
	MaxPeers   *uint64
	Bind       *string
	Port       *uint64
	NoDiscover *bool
	V5Disc     *bool
	Discovery  *P2PDiscovery
}

type P2PDiscovery struct {
	Bootnodes    []string
	BootnodesV4  []string
	BootnodesV5  []string
	StaticNodes  []string
	TrustedNodes []string
	DNS          []string
}

type TxPoolConfig struct {
	Locals       []string
	NoLocals     *bool
	Journal      *string
	Rejournal    *time.Duration
	PriceLimit   *uint64
	PriceBump    *uint64
	AccountSlots *uint64
	GlobalSlots  *uint64
	AccountQueue *uint64
	GlobalQueue  *uint64
	LifeTime     *time.Duration
}

type SealerConfig struct {
	Enabled   *bool
	Etherbase *string
	ExtraData *string
	GasCeil   *uint64
	GasPrice  *big.Int
}

func DefaultConfig() *Config {
	return &Config{
		Chain:    stringPtr("mainnet"),
		Debug:    boolPtr(false),
		LogLevel: stringPtr("INFO"),
		DataDir:  stringPtr(""),
		P2P: &P2PConfig{
			MaxPeers:   uint64Ptr(30),
			Bind:       stringPtr("0.0.0.0"),
			Port:       uint64Ptr(30303),
			NoDiscover: boolPtr(false),
			V5Disc:     boolPtr(false),
			Discovery: &P2PDiscovery{
				Bootnodes:    []string{},
				BootnodesV4:  []string{},
				BootnodesV5:  []string{},
				StaticNodes:  []string{},
				TrustedNodes: []string{},
				DNS:          []string{},
			},
		},
		SyncMode: stringPtr("fast"),
		EthStats: stringPtr(""),
		TxPool: &TxPoolConfig{
			Locals:       []string{},
			NoLocals:     boolPtr(false),
			Journal:      stringPtr(""),
			Rejournal:    durPtr(1 * time.Hour),
			PriceLimit:   uint64Ptr(1),
			PriceBump:    uint64Ptr(10),
			AccountSlots: uint64Ptr(16),
			GlobalSlots:  uint64Ptr(4096),
			AccountQueue: uint64Ptr(64),
			GlobalQueue:  uint64Ptr(1024),
			LifeTime:     durPtr(3 * time.Hour),
		},
		Sealer: &SealerConfig{
			Enabled:  boolPtr(false),
			GasCeil:  uint64Ptr(8000000),
			GasPrice: big.NewInt(params.GWei),
		},
	}
}

func readConfigFile(path string) (*Config, error) {
	return nil, nil
}

func (c *Config) loadChain() error {
	chain, ok := chains.GetChain(*c.Chain)
	if !ok {
		return fmt.Errorf("chain '%s' not found", *c.Chain)
	}
	c.chain = chain

	// preload some default values that are on the chain file
	if c.P2P.Discovery.Bootnodes == nil {
		c.P2P.Discovery.Bootnodes = c.chain.Bootnodes
	}
	if c.P2P.Discovery.DNS == nil {
		c.P2P.Discovery.DNS = c.chain.DNS
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

	// txpool options
	{
		cfg := n.TxPool
		cfg.NoLocals = *c.TxPool.NoLocals
		cfg.Journal = *c.TxPool.Journal
		cfg.Rejournal = *c.TxPool.Rejournal
		cfg.PriceLimit = *c.TxPool.PriceLimit
		cfg.PriceBump = *c.TxPool.PriceBump
		cfg.AccountSlots = *c.TxPool.AccountSlots
		cfg.GlobalSlots = *c.TxPool.GlobalSlots
		cfg.AccountQueue = *c.TxPool.AccountQueue
		cfg.GlobalQueue = *c.TxPool.GlobalQueue
		cfg.Lifetime = *c.TxPool.LifeTime
	}

	// miner options
	{
		cfg := n.Miner
		cfg.Etherbase = common.HexToAddress(*c.Sealer.Etherbase)
		cfg.GasPrice = c.Sealer.GasPrice
		cfg.GasCeil = *c.Sealer.GasCeil
	}

	// discovery (this params should be in node.Config)
	{
		n.EthDiscoveryURLs = c.P2P.Discovery.DNS
		n.SnapDiscoveryURLs = c.P2P.Discovery.DNS
	}

	var syncMode downloader.SyncMode
	switch *c.SyncMode {
	case "fast":
		syncMode = downloader.FastSync
	default:
		return nil, fmt.Errorf("sync mode '%s' not found", syncMode)
	}
	n.SyncMode = syncMode
	n.DatabaseHandles = dbHandles

	return &n, nil
}

var (
	clientIdentifier = "bor"
	gitCommit        = "" // Git SHA1 commit hash of the release (set via linker flags)
	gitDate          = "" // Git commit date YYYYMMDD of the release (set via linker flags)
)

func (c *Config) buildNode() (*node.Config, error) {
	cfg := &node.Config{
		Name:    clientIdentifier,
		DataDir: *c.DataDir,
		Version: params.VersionWithCommit(gitCommit, gitDate),
		P2P: p2p.Config{
			MaxPeers:   int(*c.P2P.MaxPeers),
			ListenAddr: *c.P2P.Bind + ":" + strconv.Itoa(int(*c.P2P.Port)),
		},
		/*
			HTTPHost:         *c.BindAddr,
			HTTPPort:         int(*c.Ports.HTTP),
			HTTPVirtualHosts: []string{"*"},
			WSHost:           *c.BindAddr,
			WSPort:           int(*c.Ports.Websocket),
		*/
	}

	// setup private key for DevP2P if not found
	devP2PPrivKey, err := readDevP2PKey(*c.DataDir)
	if err != nil {
		return nil, err
	}
	cfg.P2P.PrivateKey = devP2PPrivKey

	// Discovery
	if cfg.P2P.BootstrapNodes, err = parseBootnodes(c.P2P.Discovery.Bootnodes); err != nil {
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

	if *c.P2P.NoDiscover {
		// Disable networking, for now, we will not even allow incomming connections
		cfg.P2P.MaxPeers = 0
		cfg.P2P.NoDiscovery = true
	}
	return cfg, nil
}

func (c *Config) Merge(cc ...*Config) error {
	for _, elem := range cc {
		if err := mergo.Merge(&c, elem); err != nil {
			return err
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

const devP2PKeyPath = "devp2p.key"

func readDevP2PKey(dataDir string) (*ecdsa.PrivateKey, error) {
	path := filepath.Join(dataDir, devP2PKeyPath)
	_, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to stat (%s): %v", path, err)
	}

	if os.IsNotExist(err) {
		priv, err := crypto.GenerateKey()
		if err != nil {
			return nil, err
		}
		if err := crypto.SaveECDSA(path, priv); err != nil {
			return nil, err
		}
		return priv, nil
	}

	// exists
	priv, err := crypto.LoadECDSA(path)
	if err != nil {
		return nil, err
	}
	return priv, nil
}
