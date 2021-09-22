package server

import (
	"strconv"

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

type Config struct {
	Debug    *bool
	LogLevel *string
	DataDir  *string
	P2P      *P2PConfig
}

type P2PConfig struct {
	MaxPeers *uint64
	Bind     *string
	Port     *uint64
}

func DefaultConfig() *Config {
	return &Config{
		Debug:    boolPtr(false),
		LogLevel: stringPtr("INFO"),
		DataDir:  stringPtr(""),
		P2P: &P2PConfig{
			MaxPeers: uint64Ptr(30),
			Bind:     stringPtr("0.0.0.0."),
			Port:     uint64Ptr(30303),
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
	n.SyncMode = downloader.FastSync
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
