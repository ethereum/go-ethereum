package server

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
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
	MaxPeers    *uint64
	Bind        *string
	Port        *uint64
	NoDiscover  *bool
	V5Disc      *bool
	Bootnodes   []string
	BootnodesV4 []string
	BootnodesV5 []string
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
	Seal      *bool
	Etherbase *string
	ExtraData *string
}

func DefaultConfig() *Config {
	return &Config{
		Chain:    stringPtr("mainnet"),
		Debug:    boolPtr(false),
		LogLevel: stringPtr("INFO"),
		DataDir:  stringPtr(""),
		P2P: &P2PConfig{
			MaxPeers:    uint64Ptr(30),
			Bind:        stringPtr("0.0.0.0"),
			Port:        uint64Ptr(30303),
			NoDiscover:  boolPtr(false),
			V5Disc:      boolPtr(false),
			Bootnodes:   []string{},
			BootnodesV4: []string{},
			BootnodesV5: []string{},
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
			Seal: boolPtr(false),
		},
	}
}

func readConfigFile(path string) (*Config, error) {
	return nil, nil
}

func (c *Config) buildEth() (*ethconfig.Config, error) {
	dbHandles, err := makeDatabaseHandles()
	if err != nil {
		return nil, err
	}
	n := ethconfig.Defaults
	//n.NetworkId = c.genesis.NetworkId
	//n.Genesis = c.genesis.Genesis

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
		fmt.Println(cfg)
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

func (c *Config) buildNode() (*node.Config, error) {
	/*
		bootstrap, err := parseBootnodes(c.genesis.Bootstrap)
		if err != nil {
			return nil, err
		}
		static, err := parseBootnodes(c.genesis.Static)
		if err != nil {
			return nil, err
		}
	*/

	n := &node.Config{
		Name:    "reader",
		DataDir: *c.DataDir,
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
	/*
		if *c.NoDiscovery {
			// avoid incoming connections
			n.P2P.MaxPeers = 0
			// avoid outgoing connections
			n.P2P.NoDiscovery = true
		}
	*/

	return n, nil
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
