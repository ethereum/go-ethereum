// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

// Package config separates QuarkChain network definitions from local settings.
package config

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/naoina/toml"
)

// NetworkConfig is shared by every node in a network. NetworkID identifies the
// P2P network, independently of the EVM ChainID in each shard's chain config.
type NetworkConfig struct {
	NetworkID uint32
	Root      RootConfig
	Shards    []params.ChainConfig
}

// RootConfig contains root-chain consensus settings. The root is not an EVM
// chain and does not inherit a shard's chain ID or PoSW configuration.
type RootConfig struct {
	ConsensusType   params.ConsensusType
	PoSW            *params.PoSWConfig
	CoinbaseChainID uint32 // Intended root reward and staking chain; runtime integration follows separately.
}

const (
	RoleMaster = "master"
	RoleSlave  = "slave"
)

// ClusterConfig describes a deployment. Every process reads the same network
// definition and selects either Master or one entry from Slaves.
type ClusterConfig struct {
	NetworkFile string
	Master      NodeConfig
	Slaves      []NodeConfig
}

// NodeConfig contains the effective settings of one process. Master has no
// ChainID, PrivateHost, PrivatePort or Eth settings. A slave's PrivateHost and
// PrivatePort define its internal TCP endpoint, used both for listening and for
// dialing by Master and the other slaves. The TOML PublicHost and PublicPort
// keys configure Node.HTTPHost and Node.HTTPPort without duplicating their state.
type NodeConfig struct {
	ChainID         *uint32
	PrivateHost     string
	PrivatePort     int
	CoinbaseAddress common.Address // Local mining reward recipient; not a network rule.
	Node            node.Config
	Eth             *ethconfig.Config

	Role           string            `toml:"-"`
	NetworkFile    string            `toml:"-"`
	ShardEndpoints map[uint32]string `toml:"-"` // Derived for Master; never configured separately.
}

// TOML uses geth's case-sensitive Go field names and rejects unknown fields.
var tomlSettings = toml.Config{
	NormFieldName: func(_ reflect.Type, key string) string { return key },
	FieldToKey:    func(_ reflect.Type, field string) string { return field },
}

// DefaultNodeConfig returns per-process geth defaults. A nil chainID selects
// Master; a non-nil ID selects the slave for that EVM chain ID.
func DefaultNodeConfig(chainID *uint32) NodeConfig {
	cfg := NodeConfig{Role: RoleMaster, Node: node.DefaultConfig}
	name := "master"
	if chainID != nil {
		id := *chainID
		cfg.Role, cfg.ChainID = RoleSlave, &id
		name = fmt.Sprintf("shard-%d", id)
		eth := ethconfig.Defaults
		// TOML text decoding mutates big integers in place. Keep defaults and
		// other shard entries independent when overriding these values.
		eth.Miner.GasPrice = new(big.Int).Set(eth.Miner.GasPrice)
		eth.GPO.MaxPrice = new(big.Int).Set(eth.GPO.MaxPrice)
		eth.GPO.IgnorePrice = new(big.Int).Set(eth.GPO.IgnorePrice)
		cfg.Eth = &eth
		cfg.Node.P2P.ListenAddr, cfg.Node.P2P.DiscAddr = "", ""
		cfg.Node.P2P.MaxPeers = 0
		cfg.Node.P2P.NoDiscovery, cfg.Node.P2P.NoDial = true, true
		cfg.Node.P2P.DiscoveryV4, cfg.Node.P2P.DiscoveryV5 = false, false
		cfg.Node.HTTPModules = append(slices.Clone(cfg.Node.HTTPModules), "eth")
		cfg.Node.WSModules = append(slices.Clone(cfg.Node.WSModules), "eth")
	}
	cfg.Node.Name = "qkc-" + name
	if cfg.Node.DataDir != "" {
		cfg.Node.DataDir = filepath.Join(cfg.Node.DataDir, "qkc", name)
	}
	cfg.Node.IPCPath = cfg.Node.Name + ".ipc"
	return cfg
}

// UnmarshalTOML reads identity first, then overlays explicit settings on that
// process's defaults. Both passes use the same strict schema; zero and false
// values remain explicit overrides, including in array-of-table entries.
func (c *NodeConfig) UnmarshalTOML(decode func(interface{}) error) error {
	// Decode execution fields directly: ethconfig.Config's generated decoder
	// replaces nested tables instead of overlaying their individual fields.
	type executionConfig ethconfig.Config
	type plain struct {
		ChainID         *uint32
		PrivateHost     string
		PrivatePort     int
		PublicHost      *string
		PublicPort      *int
		CoinbaseAddress common.Address
		Node            node.Config
		Eth             *executionConfig
	}
	var identity plain
	if err := decode(&identity); err != nil {
		return err
	}
	cfg := DefaultNodeConfig(identity.ChainID)
	overlay := plain{
		ChainID: cfg.ChainID, PrivateHost: cfg.PrivateHost, PrivatePort: cfg.PrivatePort,
		CoinbaseAddress: cfg.CoinbaseAddress, Node: cfg.Node, Eth: (*executionConfig)(cfg.Eth),
	}
	if err := decode(&overlay); err != nil {
		return err
	}
	// Public keys take precedence over the underlying geth HTTP settings. Pointers
	// preserve explicit empty hosts and zero ports as overrides.
	if overlay.PublicHost != nil {
		overlay.Node.HTTPHost = *overlay.PublicHost
	}
	if overlay.PublicPort != nil {
		overlay.Node.HTTPPort = *overlay.PublicPort
	}
	cfg.ChainID, cfg.PrivateHost, cfg.PrivatePort = overlay.ChainID, overlay.PrivateHost, overlay.PrivatePort
	cfg.Node, cfg.Eth = overlay.Node, (*ethconfig.Config)(overlay.Eth)
	cfg.CoinbaseAddress = overlay.CoinbaseAddress
	*c = cfg
	return nil
}

// LoadCluster loads a deployment, resolving NetworkFile relative to it.
// Validation follows process selection and explicit CLI overrides.
func LoadCluster(file string) (*ClusterConfig, error) {
	if file == "" {
		return nil, errors.New("deployment file is required")
	}
	cfg := ClusterConfig{Master: DefaultNodeConfig(nil)}
	if err := load(file, &cfg); err != nil {
		return nil, err
	}
	if cfg.NetworkFile != "" && !filepath.IsAbs(cfg.NetworkFile) {
		cfg.NetworkFile = filepath.Join(filepath.Dir(file), cfg.NetworkFile)
	}
	return &cfg, nil
}

// SelectNode selects one process. Only Master's effective configuration includes
// the shard endpoint map, derived from deployment entries without duplication.
func (c *ClusterConfig) SelectNode(role string, chainID *uint32) (*NodeConfig, error) {
	var selected *NodeConfig
	switch role {
	case RoleMaster:
		if chainID != nil {
			return nil, errors.New("master cannot select a chain ID")
		}
		selected = &c.Master
		selected.ShardEndpoints = make(map[uint32]string, len(c.Slaves))
		for _, shard := range c.Slaves {
			if shard.ChainID != nil {
				selected.ShardEndpoints[*shard.ChainID] = shard.PrivateEndpoint()
			}
		}
	case RoleSlave:
		if chainID == nil {
			return nil, errors.New("slave process requires a chain ID")
		}
		for i := range c.Slaves {
			if c.Slaves[i].ChainID != nil && *c.Slaves[i].ChainID == *chainID {
				selected = &c.Slaves[i]
				break
			}
		}
		if selected == nil {
			return nil, fmt.Errorf("chain ID %d is not in the deployment", *chainID)
		}
	default:
		return nil, errors.New("set --qkc.role to master or slave")
	}
	selected.NetworkFile = c.NetworkFile
	return selected, nil
}

// PrivateEndpoint returns the shard's internal TCP address. An empty host means
// no cluster listener, as for Master. IPv6 hosts are joined with brackets.
func (c *NodeConfig) PrivateEndpoint() string {
	if c.PrivateHost == "" {
		return ""
	}
	return net.JoinHostPort(c.PrivateHost, strconv.Itoa(c.PrivatePort))
}

// Validate checks the complete deployment after overrides. Every network shard
// has one process, with distinct TCP listeners, IPC endpoints and configured data
// directories.
func (c *ClusterConfig) Validate(network *NetworkConfig) error {
	if err := network.Validate(); err != nil {
		return err
	}
	if c.Master.Role != RoleMaster || c.Master.ChainID != nil {
		return errors.New("Master must not specify ChainID")
	}
	if err := c.Master.Validate(network); err != nil {
		return fmt.Errorf("Master: %w", err)
	}
	ids, dirs, ipcEndpoints := make(map[uint32]bool), make(map[string]bool), make(map[string]bool)
	tcpEndpoints := make(map[uint64][]*net.TCPAddr)
	for i, cfg := range append([]NodeConfig{c.Master}, c.Slaves...) {
		if i > 0 {
			if cfg.ChainID == nil {
				return fmt.Errorf("Slaves[%d]: ChainID is required", i-1)
			}
			if ids[*cfg.ChainID] {
				return fmt.Errorf("duplicate deployed chain ID %d", *cfg.ChainID)
			}
			ids[*cfg.ChainID] = true
			if err := cfg.Validate(network); err != nil {
				return fmt.Errorf("Slaves[%d]: %w", i-1, err)
			}
		}
		endpoints := []string{cfg.PrivateEndpoint(), cfg.Node.P2P.ListenAddr, cfg.Node.WSEndpoint()}
		if cfg.Node.WSHost == "" || cfg.Node.HTTPPort != cfg.Node.WSPort {
			// Geth shares the listener and uses WSHost when HTTP and WS ports match.
			endpoints = append(endpoints, cfg.Node.HTTPEndpoint())
		}
		for _, address := range endpoints {
			if address == "" {
				continue // Disabled listener.
			}
			// NodeConfig.Validate has already checked the address and port.
			host, port, _ := net.SplitHostPort(address)
			value, _ := strconv.ParseUint(port, 10, 16)
			if value == 0 {
				continue // Dynamically assigned ports cannot conflict here.
			}
			// Resolve aliases using the same address selection as net.Listen.
			endpoint, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(canonicalHost(host), port))
			if err != nil {
				return fmt.Errorf("resolve TCP listener %q: %w", address, err)
			}
			ip := endpoint.AddrPort().Addr().Unmap()
			for _, other := range tcpEndpoints[value] {
				otherIP := other.AddrPort().Addr().Unmap()
				if ip == otherIP || !ip.IsValid() || !otherIP.IsValid() || ip.IsUnspecified() || otherIP.IsUnspecified() {
					return fmt.Errorf("TCP listeners conflict: %s and %s", address, other)
				}
			}
			tcpEndpoints[value] = append(tcpEndpoints[value], endpoint)
		}
		if cfg.Node.DataDir != "" {
			dir, err := filepath.Abs(cfg.Node.DataDir)
			if err != nil {
				return err
			}
			if dirs[dir] {
				return fmt.Errorf("process data directory is shared: %s", cfg.Node.DataDir)
			}
			dirs[dir] = true
		}
		if endpoint := cfg.Node.IPCEndpoint(); endpoint != "" {
			path, err := filepath.Abs(endpoint)
			if err != nil {
				return err
			}
			if runtime.GOOS == "windows" {
				path = strings.ToLower(path) // Windows named pipes are case-insensitive.
			}
			if ipcEndpoints[path] {
				return fmt.Errorf("process IPC endpoint is shared: %s", endpoint)
			}
			ipcEndpoints[path] = true
		}
	}
	if len(ids) != len(network.Shards) {
		return errors.New("deployment must contain every network shard")
	}
	return nil
}

// canonicalHost normalizes IP literals and hostname case without resolving DNS.
func canonicalHost(host string) string {
	if ip, err := netip.ParseAddr(host); err == nil {
		return ip.Unmap().String()
	}
	return strings.ToLower(host)
}

// LoadNetwork loads and validates a network definition. Each shard's routing
// identity and topology are derived from the EVM chain IDs, sorted numerically.
// Root and shard PoSW configurations are required, including when staking is disabled.
func LoadNetwork(file string) (*NetworkConfig, error) {
	var cfg NetworkConfig
	if file == "" {
		return nil, errors.New("NetworkFile is required")
	}
	if err := load(file, &cfg); err != nil {
		return nil, err
	}
	ids := make([]uint32, len(cfg.Shards))
	for i := range cfg.Shards {
		id, err := routingChainID(&cfg.Shards[i])
		if err != nil {
			return nil, fmt.Errorf("%s: Shards[%d]: %w", file, i, err)
		}
		ids[i] = id
	}
	slices.Sort(ids)
	for _, shard := range cfg.Shards {
		if shard.QuarkChain == nil {
			continue
		}
		shard.QuarkChain.ChainID = uint32(shard.ChainID.Uint64())
		shard.QuarkChain.ChainIDs = slices.Clone(ids)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return &cfg, nil
}

// routingChainID checks the EVM identifier before narrowing it to the wire format.
func routingChainID(chain *params.ChainConfig) (uint32, error) {
	if chain.ChainID == nil || chain.ChainID.Sign() <= 0 || chain.ChainID.BitLen() > 32 {
		return 0, errors.New("ChainID must be a positive uint32 for EVM execution and routing")
	}
	id := uint32(chain.ChainID.Uint64())
	if id == math.MaxUint32 {
		return 0, fmt.Errorf("chain ID %d is reserved for the root chain", id)
	}
	return id, nil
}

func load(file string, cfg interface{}) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := tomlSettings.NewDecoder(f).Decode(cfg); err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}
	return nil
}

// Validate checks network IDs, shard topology, algorithms, fork order and each chain's PoSW.
func (c *NetworkConfig) Validate() error {
	if c.NetworkID == 0 {
		return errors.New("NetworkID must be positive; it is not inferred from an EVM chain ID")
	}
	if err := c.Root.ConsensusType.Check(); err != nil {
		return fmt.Errorf("Root: %w", err)
	}
	if err := c.Root.PoSW.Check(); err != nil {
		return fmt.Errorf("Root: %w", err)
	}
	if len(c.Shards) == 0 {
		return errors.New("network must define at least one shard")
	}
	ids := make(map[uint32]bool, len(c.Shards))
	for i, shard := range c.Shards {
		if shard.QuarkChain == nil {
			return fmt.Errorf("Shards[%d]: QuarkChain configuration is required", i)
		}
		id, err := routingChainID(&shard)
		if err != nil {
			return fmt.Errorf("Shards[%d]: %w", i, err)
		}
		if err := shard.QuarkChain.ConsensusType.Check(); err != nil {
			return fmt.Errorf("chain %d: %w", id, err)
		}
		if ids[id] {
			return fmt.Errorf("duplicate ChainID %d", id)
		}
		ids[id] = true
		if shard.QuarkChain.ChainID != id {
			return fmt.Errorf("chain %d: QuarkChain.ChainID must match the EVM ChainID", id)
		}
		if err := shard.CheckConfigForkOrder(); err != nil {
			return fmt.Errorf("chain %d: %w", id, err)
		}
	}
	if !ids[c.Root.CoinbaseChainID] {
		return fmt.Errorf("Root.CoinbaseChainID %d must identify a network chain for root rewards and staking", c.Root.CoinbaseChainID)
	}
	for _, shard := range c.Shards {
		qkc := shard.QuarkChain
		seen := make(map[uint32]bool, len(qkc.ChainIDs))
		for _, id := range qkc.ChainIDs {
			if !ids[id] || seen[id] {
				return fmt.Errorf("chain %d: routing chain IDs contain unknown or duplicate chain %d", qkc.ChainID, id)
			}
			seen[id] = true
		}
		if len(seen) != len(ids) {
			return fmt.Errorf("chain %d: routing chain IDs must contain every network chain", qkc.ChainID)
		}
	}
	return nil
}

// Validate checks effective process settings. Master uses the network definition
// directly; only shard execution settings bind Eth.NetworkId to NetworkID.
func (c *NodeConfig) Validate(network *NetworkConfig) error {
	if err := network.Validate(); err != nil {
		return err
	}
	switch c.Role {
	case RoleMaster:
		if c.Eth != nil || c.ChainID != nil || c.PrivateHost != "" || c.PrivatePort != 0 {
			return errors.New("master cannot configure Eth, ChainID, PrivateHost or PrivatePort")
		}
	case RoleSlave:
		if c.ChainID == nil || c.Eth == nil {
			return errors.New("slave requires ChainID and execution settings")
		}
		found := false
		for _, shard := range network.Shards {
			found = found || shard.ChainID.Uint64() == uint64(*c.ChainID)
		}
		if !found {
			return fmt.Errorf("chain ID %d is not in the network", *c.ChainID)
		}
		ip, err := netip.ParseAddr(c.PrivateHost)
		if c.PrivateHost == "" || strings.ContainsAny(c.PrivateHost, " \t\r\n/[]") ||
			(strings.Contains(c.PrivateHost, ":") && err != nil) || (err == nil && ip.Unmap().IsUnspecified()) {
			return fmt.Errorf("PrivateHost must be a concrete host without brackets or a port: %q", c.PrivateHost)
		}
		if c.PrivatePort < 1 || c.PrivatePort > 65535 {
			return errors.New("PrivatePort must be between 1 and 65535")
		}
		p := c.Node.P2P
		if p.ListenAddr != "" || p.DiscAddr != "" || p.MaxPeers != 0 || !p.NoDiscovery || !p.NoDial ||
			p.DiscoveryV4 || p.DiscoveryV5 || len(p.BootstrapNodes)+len(p.BootstrapNodesV5)+len(p.StaticNodes)+len(p.TrustedNodes) != 0 {
			return errors.New("slave external P2P must be disabled; configure PrivateHost and PrivatePort instead")
		}
	default:
		return fmt.Errorf("invalid process role %q", c.Role)
	}
	if c.Eth != nil {
		if c.Eth.NetworkId != 0 && c.Eth.NetworkId != uint64(network.NetworkID) {
			return fmt.Errorf("Eth.NetworkId %d conflicts with NetworkID %d", c.Eth.NetworkId, network.NetworkID)
		}
		if c.Eth.Genesis != nil {
			return errors.New("Eth.Genesis is a network definition and cannot be set locally")
		}
		if c.Eth.OverrideOsaka != nil || c.Eth.OverrideAmsterdam != nil || c.Eth.OverrideBPO1 != nil || c.Eth.OverrideBPO2 != nil || c.Eth.OverrideUBT != nil {
			return errors.New("local fork overrides are not allowed; configure forks in the network file")
		}
		if c.Eth.StateScheme != "" && c.Eth.StateScheme != "hash" && c.Eth.StateScheme != "path" {
			return fmt.Errorf("invalid Eth.StateScheme %q", c.Eth.StateScheme)
		}
		if !c.Eth.SyncMode.IsValid() {
			return fmt.Errorf("invalid Eth.SyncMode %d", c.Eth.SyncMode)
		}
		for name, value := range map[string]int{"Eth.DatabaseCache": c.Eth.DatabaseCache, "Eth.TrieCleanCache": c.Eth.TrieCleanCache, "Eth.TrieDirtyCache": c.Eth.TrieDirtyCache, "Eth.SnapshotCache": c.Eth.SnapshotCache} {
			if value < 0 {
				return fmt.Errorf("%s must not be negative", name)
			}
		}
		c.Eth.NetworkId = uint64(network.NetworkID)
	}
	if c.Node.DBEngine != "" && c.Node.DBEngine != "pebble" && c.Node.DBEngine != "leveldb" {
		return fmt.Errorf("invalid Node.DBEngine %q", c.Node.DBEngine)
	}
	for name, value := range map[string]int{
		"Node.P2P.MaxPeers": c.Node.P2P.MaxPeers, "Node.P2P.MaxPendingPeers": c.Node.P2P.MaxPendingPeers,
	} {
		if value < 0 {
			return fmt.Errorf("%s must not be negative", name)
		}
	}
	for name, port := range map[string]int{"PublicPort": c.Node.HTTPPort, "Node.WSPort": c.Node.WSPort, "Node.AuthPort": c.Node.AuthPort} {
		if port < 0 || port > math.MaxUint16 {
			return fmt.Errorf("%s must be between 0 and 65535", name)
		}
	}
	for name, addr := range map[string]string{"Node.P2P.ListenAddr": c.Node.P2P.ListenAddr, "Node.P2P.DiscAddr": c.Node.P2P.DiscAddr} {
		if addr == "" {
			continue
		}
		_, port, err := net.SplitHostPort(addr)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return fmt.Errorf("%s: invalid port %q", name, port)
		}
	}
	return nil
}
