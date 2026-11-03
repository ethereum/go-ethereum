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

package config

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/naoina/toml"
)

func exampleNetwork(t *testing.T) *NetworkConfig {
	t.Helper()
	cfg, err := LoadNetwork("examples/network.toml")
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func writeConfig(t *testing.T, content []byte) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(file, content, 0600); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestNetworkExample(t *testing.T) {
	cfg := exampleNetwork(t)
	if cfg.NetworkID != 2 || len(cfg.Shards) != 2 {
		t.Fatalf("unexpected network: %+v", cfg)
	}
	if cfg.Root.ConsensusType != params.ConsensusEthash || cfg.Shards[0].QuarkChain.ConsensusType != params.ConsensusEthash || cfg.Shards[1].QuarkChain.ConsensusType != params.ConsensusSimulate {
		t.Fatal("root and shard consensus algorithms were not preserved")
	}
	if !cfg.Root.PoSW.StakingEnabled || cfg.Shards[0].QuarkChain.PoSW.StakingEnabled || !cfg.Shards[1].QuarkChain.PoSW.StakingEnabled {
		t.Fatal("root and shard PoSW configurations are not independent")
	}
	if cfg.Root.PoSW.StakePerBlock.String() != "1000000000000000000000000000000" {
		t.Fatal("large stake was not preserved")
	}
	if cfg.Root.PoSW.DecayFactor.Cmp(big.NewRat(22, 25)) != 0 || cfg.Shards[0].QuarkChain.PoSW.RewardTaxRate.Cmp(big.NewRat(1, 2)) != 0 {
		t.Fatal("consensus ratios were not parsed exactly")
	}
	for _, shard := range cfg.Shards {
		if shard.QuarkChain.ChainID != uint32(shard.ChainID.Uint64()) {
			t.Fatal("routing identity differs from the EVM chain ID")
		}
		if !slices.Equal(shard.QuarkChain.ChainIDs, []uint32{100001, 100002}) {
			t.Fatalf("unexpected shard topology: %v", shard.QuarkChain.ChainIDs)
		}
		if shard.QuarkChain.MaxBlocksPerRootBlock != 18 {
			t.Fatal("configured shard confirmation limit was not preserved")
		}
	}
	// Equal numeric values do not conflate the network and EVM identifiers.
	cfg.NetworkID = uint32(cfg.Shards[0].ChainID.Uint64())
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestNetworkValidation(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*NetworkConfig)
		want   string
	}{
		{"zero network", func(c *NetworkConfig) { c.NetworkID = 0 }, "NetworkID must be positive"},
		{"missing root consensus", func(c *NetworkConfig) { c.Root.ConsensusType = "" }, "Root: invalid ConsensusType"},
		{"invalid root consensus", func(c *NetworkConfig) { c.Root.ConsensusType = "NONE" }, "Root: invalid ConsensusType"},
		{"unsupported root doublesha256", func(c *NetworkConfig) { c.Root.ConsensusType = "POW_DOUBLESHA256" }, "Root: invalid ConsensusType"},
		{"unsupported root qkchash", func(c *NetworkConfig) { c.Root.ConsensusType = "POW_QKCHASH" }, "Root: invalid ConsensusType"},
		{"missing shard consensus", func(c *NetworkConfig) { c.Shards[0].QuarkChain.ConsensusType = "" }, "chain 100001: invalid ConsensusType"},
		{"invalid shard consensus", func(c *NetworkConfig) { c.Shards[1].QuarkChain.ConsensusType = "ethash" }, "chain 100002: invalid ConsensusType"},
		{"unsupported shard doublesha256", func(c *NetworkConfig) { c.Shards[0].QuarkChain.ConsensusType = "POW_DOUBLESHA256" }, "chain 100001: invalid ConsensusType"},
		{"unsupported shard qkchash", func(c *NetworkConfig) { c.Shards[1].QuarkChain.ConsensusType = "POW_QKCHASH" }, "chain 100002: invalid ConsensusType"},
		{"no shards", func(c *NetworkConfig) { c.Shards = nil }, "at least one shard"},
		{"no quarkchain", func(c *NetworkConfig) { c.Shards[0].QuarkChain = nil }, "QuarkChain configuration is required"},
		{"mismatched routing id", func(c *NetworkConfig) { c.Shards[1].QuarkChain.ChainID = 1 }, "must match the EVM ChainID"},
		{"reserved chain id", func(c *NetworkConfig) { c.Shards[1].ChainID.SetUint64(math.MaxUint32) }, "reserved for the root"},
		{"missing chain id", func(c *NetworkConfig) { c.Shards[0].ChainID = nil }, "positive uint32"},
		{"zero chain id", func(c *NetworkConfig) { c.Shards[0].ChainID = new(big.Int) }, "positive uint32"},
		{"negative chain id", func(c *NetworkConfig) { c.Shards[0].ChainID = big.NewInt(-1) }, "positive uint32"},
		{"overflow chain id", func(c *NetworkConfig) { c.Shards[0].ChainID = new(big.Int).Lsh(big.NewInt(1), 32) }, "positive uint32"},
		{"duplicate chain id", func(c *NetworkConfig) { c.Shards[1].ChainID = c.Shards[0].ChainID }, "duplicate ChainID"},
		{"missing peer", func(c *NetworkConfig) { c.Shards[0].QuarkChain.ChainIDs = []uint32{100001} }, "every network chain"},
		{"unknown peer", func(c *NetworkConfig) { c.Shards[0].QuarkChain.ChainIDs = []uint32{100001, 100003} }, "unknown or duplicate"},
		{"duplicate peer", func(c *NetworkConfig) { c.Shards[0].QuarkChain.ChainIDs = []uint32{100001, 100001} }, "unknown or duplicate"},
		{"missing root coinbase chain", func(c *NetworkConfig) { c.Root.CoinbaseChainID = 0 }, "must identify a network chain"},
		{"unknown root coinbase chain", func(c *NetworkConfig) { c.Root.CoinbaseChainID = 100003 }, "must identify a network chain"},
		{"missing root posw", func(c *NetworkConfig) { c.Root.PoSW = nil }, "Root: posw:"},
		{"missing shard posw", func(c *NetworkConfig) { c.Shards[0].QuarkChain.PoSW = nil }, "chain 100001: posw:"},
		{"invalid ordinary pow", func(c *NetworkConfig) { c.Shards[0].QuarkChain.PoSW.MinimumDifficulty = nil }, "chain 100001: posw:"},
		{"invalid root posw", func(c *NetworkConfig) { c.Root.PoSW.WindowSize = 0 }, "Root: posw:"},
		{"invalid shard posw", func(c *NetworkConfig) { c.Shards[1].QuarkChain.PoSW.DiffDivider = 0 }, "chain 100002: posw:"},
		{"negative stake", func(c *NetworkConfig) { c.Root.PoSW.StakePerBlock.SetInt64(-1) }, "Root: posw:"},
		{"overflow stake", func(c *NetworkConfig) { c.Root.PoSW.StakePerBlock.Lsh(big.NewInt(1), 256) }, "Root: posw:"},
		{"invalid fork order", func(c *NetworkConfig) { c.Shards[0].HomesteadBlock.SetInt64(1) }, "unsupported fork ordering"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := exampleNetwork(t)
			test.change(cfg)
			if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("have %v, want error containing %q", err, test.want)
			}
		})
	}
}

func TestNetworkConsensusTypes(t *testing.T) {
	for _, algorithm := range []params.ConsensusType{params.ConsensusEthash, params.ConsensusSimulate} {
		t.Run(string(algorithm), func(t *testing.T) {
			cfg := exampleNetwork(t)
			cfg.Root.ConsensusType = algorithm
			cfg.Shards[0].QuarkChain.ConsensusType = algorithm
			if err := cfg.Validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNetworkConfirmationLimit(t *testing.T) {
	input, err := os.ReadFile("examples/network.toml")
	if err != nil {
		t.Fatal(err)
	}
	for _, limit := range []uint64{0, 1, 18, math.MaxUint64} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			content := strings.Replace(string(input), "MaxBlocksPerRootBlock = 18", fmt.Sprintf("MaxBlocksPerRootBlock = %d", limit), 1)
			cfg, err := LoadNetwork(writeConfig(t, []byte(content)))
			if err != nil {
				t.Fatal(err)
			}
			if got := cfg.Shards[0].QuarkChain.MaxBlocksPerRootBlock; got != limit {
				t.Fatalf("confirmation limit = %d, want %d", got, limit)
			}
			if cfg.Shards[1].QuarkChain.MaxBlocksPerRootBlock != 18 {
				t.Fatal("confirmation limit override affected another shard")
			}
		})
	}
	content := strings.ReplaceAll(string(input), "MaxBlocksPerRootBlock = 18", "")
	cfg, err := LoadNetwork(writeConfig(t, []byte(content)))
	if err != nil {
		t.Fatal(err)
	}
	for _, shard := range cfg.Shards {
		if shard.QuarkChain.MaxBlocksPerRootBlock != 0 {
			t.Fatal("omitted confirmation limit should leave the shard uncapped")
		}
	}
	for _, value := range []string{"-1", "18446744073709551616"} {
		content := strings.Replace(string(input), "MaxBlocksPerRootBlock = 18", "MaxBlocksPerRootBlock = "+value, 1)
		if _, err := LoadNetwork(writeConfig(t, []byte(content))); err == nil || !strings.Contains(err.Error(), "MaxBlocksPerRootBlock") {
			t.Fatalf("invalid confirmation limit %s accepted: %v", value, err)
		}
	}
}

func TestNetworkDerivedChainIDs(t *testing.T) {
	input, err := os.ReadFile("examples/network.toml")
	if err != nil {
		t.Fatal(err)
	}
	// A single shard derives its own chain ID without an explicit topology list.
	single := strings.Join(strings.SplitN(string(input), "\n[[Shards]]", 3)[:2], "\n[[Shards]]")
	cfg, err := LoadNetwork(writeConfig(t, []byte(single)))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Shards) != 1 || !slices.Equal(cfg.Shards[0].QuarkChain.ChainIDs, []uint32{100001}) {
		t.Fatal("single-shard topology was not derived from the chain ID")
	}
	for _, field := range []string{"ShardID = 0", "ShardIDs = [0, 1]", "ChainID = 100001", "ChainIDs = []", "ChainIDs = [100001, 100002]", "RootCoinbaseChainID = 100001"} {
		t.Run(field, func(t *testing.T) {
			content := strings.Replace(string(input), "[Shards.QuarkChain]", "[Shards.QuarkChain]\n"+field, 1)
			file := writeConfig(t, []byte(content))
			if _, err := LoadNetwork(file); err == nil || !strings.Contains(err.Error(), file) {
				t.Fatalf("explicit routing field accepted or error lacks context: %v", err)
			}
		})
	}
	// Routing uses the actual chain IDs, including gaps, rather than array positions.
	content := strings.Replace(string(input), `ChainID = "100002"`, `ChainID = "200009"`, 1)
	content = strings.Replace(content, "CoinbaseChainID = 100001", "CoinbaseChainID = 200009", 1)
	cfg, err = LoadNetwork(writeConfig(t, []byte(content)))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Shards[1].QuarkChain.ChainID != 200009 || !slices.Equal(cfg.Shards[0].QuarkChain.ChainIDs, []uint32{100001, 200009}) {
		t.Fatal("routing did not preserve noncontiguous chain IDs")
	}
	if cfg.Root.CoinbaseChainID != 200009 {
		t.Fatal("configured root coinbase chain was replaced by the smallest chain ID")
	}
	if !cfg.Shards[0].QuarkChain.IsXShardDestination(200009) || cfg.Shards[0].QuarkChain.IsXShardDestination(1) || cfg.Shards[0].QuarkChain.IsXShardDestination(100001) {
		t.Fatal("cross-shard routing did not use the configured chain IDs")
	}
}

func TestNetworkChainIDBounds(t *testing.T) {
	input, err := os.ReadFile("examples/network.toml")
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"0", "-1", "100001", "4294967295", "4294967296", "18446744073709551616"} {
		t.Run(value, func(t *testing.T) {
			content := strings.Replace(string(input), `ChainID = "100002"`, `ChainID = "`+value+`"`, 1)
			if _, err := LoadNetwork(writeConfig(t, []byte(content))); err == nil {
				t.Fatal("invalid or duplicate chain ID was accepted")
			}
		})
	}
	content := strings.Replace(string(input), `ChainID = "100002"`, `ChainID = "4294967294"`, 1)
	cfg, err := LoadNetwork(writeConfig(t, []byte(content)))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Shards[1].QuarkChain.ChainID != math.MaxUint32-1 {
		t.Fatal("maximum routable chain ID was not preserved")
	}
}

func TestNetworkDefaults(t *testing.T) {
	const pow = `AdjustmentCutoff = 7
AdjustmentFactor = 512
MinimumDifficulty = "1"
CoinbaseAmount = "0"
RewardTaxRate = "0"
EpochInterval = 3153600
DecayFactor = "0.88"
`
	cfg, err := LoadNetwork(writeConfig(t, []byte(`NetworkID = 9
[Root]
ConsensusType = "POW_ETHASH"
CoinbaseChainID = 11
[Root.PoSW]
`+pow+`
[[Shards]]
ChainID = "11"
[Shards.QuarkChain]
ConsensusType = "POW_ETHASH"
[Shards.QuarkChain.PoSW]
`+pow+`
[[Shards]]
ChainID = "10"
[Shards.QuarkChain]
ConsensusType = "POW_ETHASH"
[Shards.QuarkChain.PoSW]
`+pow)))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Root.PoSW.StakingEnabled || cfg.Shards[0].QuarkChain.PoSW.StakingEnabled || cfg.Shards[1].QuarkChain.PoSW.StakingEnabled {
		t.Fatal("staking should be disabled by omission")
	}
	if !slices.Equal(cfg.Shards[0].QuarkChain.ChainIDs, []uint32{10, 11}) {
		t.Fatal("derived chain IDs must be sorted")
	}
	cfg.Shards[0].QuarkChain.ChainIDs[0] = 7
	if cfg.Shards[1].QuarkChain.ChainIDs[0] != 10 {
		t.Fatal("derived topology slices alias each other")
	}
}

func TestInvalidTOML(t *testing.T) {
	for _, test := range []struct {
		name    string
		network bool
		input   string
	}{
		{"wrong case", true, "networkID = 2"},
		{"network id overflow", true, "NetworkID = 4294967296"},
		{"negative network id", true, "NetworkID = -1"},
		{"duplicate key", true, "NetworkID = 2\nNetworkID = 3"},
		{"local network field", true, "[Node]\nDataDir = \"data\""},
		{"network local field", false, "[Root]\nPoSW = {}"},
		{"local typo", false, "[Master.Node]\nHttpPort = 1234"},
		{"legacy shard nodes", false, "[[ShardNodes]]\nChainID = 100001"},
		{"legacy shard id", false, "[[Slaves]]\nShardID = 0"},
		{"legacy internal address", false, "[[Slaves]]\nChainID = 100001\nAddress = \"127.0.0.1:38000\""},
		{"legacy cluster host", false, "[[Slaves]]\nChainID = 100001\nClusterHost = \"127.0.0.1\""},
		{"invalid public port type", false, "[Master]\nPublicPort = \"invalid\""},
		{"short master coinbase", false, "[Master]\nCoinbaseAddress = \"0x12\""},
		{"legacy shard coinbase", false, "[[Slaves]]\nChainID = 100001\nCoinbaseAddress = \"0x111111111111111111111111111111111111111100000001\""},
		{"invalid shard coinbase", false, "[[Slaves]]\nChainID = 100001\nCoinbaseAddress = \"0xgg11111111111111111111111111111111111111\""},
		{"nested typo", false, "[[Slaves]]\nChainID = 100001\n[Slaves.Eth.Miner]\nUnknown = 1"},
		{"invalid sync mode", false, "[[Slaves]]\nChainID = 100001\n[Slaves.Eth]\nSyncMode = \"invalid\""},
		{"derived role", false, "[Master]\nRole = \"master\""},
		{"duplicate endpoint map", false, "[Master.ShardEndpoints]\n0 = \"127.0.0.1:38000\""},
		{"chain id overflow", false, "[[Slaves]]\nChainID = 4294967296"},
		{"obsolete posw flag", true, "[Root.PoSW]\nEnabled = true"},
		{"invalid stake", true, "[Root.PoSW]\nStakePerBlock = \"invalid\""},
		{"invalid ratio", true, "[Root.PoSW]\nDecayFactor = \"invalid\""},
	} {
		t.Run(test.name, func(t *testing.T) {
			file := writeConfig(t, []byte(test.input))
			var err error
			if test.network {
				_, err = LoadNetwork(file)
			} else {
				_, err = LoadCluster(file)
			}
			if err == nil || !strings.Contains(err.Error(), file) {
				t.Fatalf("want an error identifying the file, got %v", err)
			}
			var lineError *toml.LineError
			if !errors.As(err, &lineError) || lineError.Line < 1 {
				t.Fatalf("want a TOML error identifying the line, got %v", err)
			}
		})
	}
	if _, err := LoadNetwork(""); err == nil {
		t.Fatal("missing network file accepted")
	}
	if _, err := LoadCluster(""); err == nil {
		t.Fatal("missing deployment path accepted")
	}
	if _, err := LoadCluster(filepath.Join(t.TempDir(), "missing.toml")); err == nil {
		t.Fatal("missing local file accepted")
	}
}

func exampleCluster(t *testing.T) *ClusterConfig {
	t.Helper()
	cfg, err := LoadCluster("examples/cluster.toml")
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestClusterExample(t *testing.T) {
	cfg := exampleCluster(t)
	if cfg.NetworkFile != filepath.Join("examples", "network.toml") {
		t.Fatalf("network path resolved incorrectly: %s", cfg.NetworkFile)
	}
	network := exampleNetwork(t)
	if err := cfg.Validate(network); err != nil {
		t.Fatal(err)
	}
	master, err := cfg.SelectNode(RoleMaster, nil)
	if err != nil {
		t.Fatal(err)
	}
	if master != &cfg.Master || master.Eth != nil || master.Node.P2P.MaxPeers != 25 || len(master.ShardEndpoints) != 2 {
		t.Fatalf("unexpected master configuration: %+v", master)
	}
	if master.CoinbaseAddress != common.HexToAddress("0x1111111111111111111111111111111111111111") {
		t.Fatal("master mining address was not preserved")
	}
	addresses := []common.Address{
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		common.HexToAddress("0x3333333333333333333333333333333333333333"),
	}
	for i, entry := range cfg.Slaves {
		selected, err := cfg.SelectNode(RoleSlave, entry.ChainID)
		if err != nil {
			t.Fatal(err)
		}
		if selected != &cfg.Slaves[i] || selected.NetworkFile != cfg.NetworkFile || selected.ShardEndpoints != nil {
			t.Fatal("process selection did not return the shard's own configuration")
		}
		if selected.CoinbaseAddress != addresses[i] {
			t.Fatal("selected shard has the wrong mining address")
		}
		if master.ShardEndpoints[*entry.ChainID] != fmt.Sprintf("127.0.0.1:%d", 38000+i) {
			t.Fatal("master endpoints were not derived from deployment entries")
		}
		if entry.Eth.NetworkId != uint64(network.NetworkID) || entry.Node.HTTPPort != 18545+i {
			t.Fatal("per-shard settings were not preserved")
		}
		if entry.Eth.SyncMode != ethconfig.Defaults.SyncMode || entry.Eth.RPCGasCap != ethconfig.Defaults.RPCGasCap {
			t.Fatal("omitted execution settings did not inherit geth defaults")
		}
	}
	cfg.Slaves[0].Eth.DatabaseCache = 0
	if cfg.Slaves[1].Eth.DatabaseCache != 256 {
		t.Fatal("selected shard settings changed another shard")
	}
}

func TestNodeDefaults(t *testing.T) {
	master := DefaultNodeConfig(nil)
	id := uint32(100001)
	shard := DefaultNodeConfig(&id)
	if master.Eth != nil || master.Node.P2P.MaxPeers != node.DefaultConfig.P2P.MaxPeers || shard.Eth.DatabaseCache != ethconfig.Defaults.DatabaseCache {
		t.Fatal("geth defaults were not preserved")
	}
	for _, cfg := range []NodeConfig{master, shard} {
		if cfg.Node.HTTPHost != "" || cfg.Node.WSHost != "" {
			t.Fatal("unexpected endpoint defaults")
		}
		if len(cfg.Node.P2P.BootstrapNodes) != 0 || len(cfg.Node.P2P.BootstrapNodesV5) != 0 {
			t.Fatal("an implicit network was selected")
		}
	}
	if shard.Eth.NetworkId != 0 || shard.Node.P2P.ListenAddr != "" || shard.Node.P2P.MaxPeers != 0 || !shard.Node.P2P.NoDiscovery || !shard.Node.P2P.NoDial {
		t.Fatal("slave external P2P was not disabled")
	}
	if (master.Node.DataDir != "" && master.Node.DataDir == shard.Node.DataDir) || master.Node.Name == shard.Node.Name {
		t.Fatal("process defaults share a data directory or instance name")
	}
	otherID := uint32(100002)
	for _, ephemeral := range []bool{false, true} {
		endpoints := make(map[string]bool)
		for _, cfg := range []NodeConfig{master, shard, DefaultNodeConfig(&otherID)} {
			if ephemeral {
				cfg.Node.DataDir = ""
			}
			endpoint := cfg.Node.IPCEndpoint()
			if endpoint == "" || endpoints[endpoint] {
				t.Fatalf("default IPC endpoints must be nonempty and distinct (ephemeral=%t): %q", ephemeral, endpoint)
			}
			endpoints[endpoint] = true
		}
	}
}

func TestClusterPublicEndpoint(t *testing.T) {
	for _, test := range []struct {
		name, input, host string
		port              int
		master            bool
	}{
		{"master", "[Master]\nPublicHost = \"127.0.0.1\"\nPublicPort = 18544", "127.0.0.1", 18544, true},
		{"shard", "[[Slaves]]\nChainID = 100001\nPublicHost = \"127.0.0.1\"\nPublicPort = 18545", "127.0.0.1", 18545, false},
		{"inherited public port", "[[Slaves]]\nChainID = 100001\nPublicHost = \"127.0.0.1\"", "127.0.0.1", node.DefaultHTTPPort, false},
		{"public port leaves HTTP disabled", "[[Slaves]]\nChainID = 100001\nPublicPort = 18545", "", 18545, false},
		{"native fallback", "[[Slaves]]\nChainID = 100001\n[Slaves.Node]\nHTTPHost = \"127.0.0.2\"\nHTTPPort = 18547", "127.0.0.2", 18547, false},
		{"public override", "[[Slaves]]\nChainID = 100001\nPublicHost = \"127.0.0.1\"\nPublicPort = 18545\n[Slaves.Node]\nHTTPHost = \"127.0.0.2\"\nHTTPPort = 18547", "127.0.0.1", 18545, false},
		{"explicit empty and zero", "[[Slaves]]\nChainID = 100001\nPublicHost = \"\"\nPublicPort = 0\n[Slaves.Node]\nHTTPHost = \"127.0.0.2\"\nHTTPPort = 18547", "", 0, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := LoadCluster(writeConfig(t, []byte(test.input)))
			if err != nil {
				t.Fatal(err)
			}
			selected := &cfg.Master
			if !test.master {
				selected = &cfg.Slaves[0]
			}
			if selected.Node.HTTPHost != test.host || selected.Node.HTTPPort != test.port {
				t.Fatalf("effective HTTP endpoint = %q:%d, want %q:%d", selected.Node.HTTPHost, selected.Node.HTTPPort, test.host, test.port)
			}
		})
	}
}

func TestClusterDefaultOverrides(t *testing.T) {
	cfg, err := LoadCluster(writeConfig(t, []byte(`[Master.Node.P2P]
MaxPeers = 0
DiscoveryV4 = false
[[Slaves]]
ChainID = 100001
PrivateHost = "127.0.0.1"
PrivatePort = 38000
[Slaves.Eth]
DatabaseCache = 0
[Slaves.Eth.Miner]
GasPrice = "123"
[Slaves.Eth.GPO]
MaxPrice = "456"
IgnorePrice = "7"
[[Slaves]]
ChainID = 100002
PrivateHost = "127.0.0.1"
PrivatePort = 38001
`)))
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(exampleNetwork(t)); err != nil {
		t.Fatal(err)
	}
	if cfg.Master.Node.P2P.MaxPeers != 0 || cfg.Master.Node.P2P.DiscoveryV4 || cfg.Slaves[0].Eth.DatabaseCache != 0 {
		t.Fatal("explicit zero or false was replaced by defaults")
	}
	first, second := cfg.Slaves[0].Eth, cfg.Slaves[1].Eth
	if first.Miner.GasPrice.Int64() != 123 || first.GPO.MaxPrice.Int64() != 456 || first.GPO.IgnorePrice.Int64() != 7 {
		t.Fatal("explicit big integer overrides were not applied")
	}
	if second.DatabaseCache != ethconfig.Defaults.DatabaseCache || second.Miner.GasPrice.Cmp(ethconfig.Defaults.Miner.GasPrice) != 0 || second.GPO.MaxPrice.Cmp(ethconfig.Defaults.GPO.MaxPrice) != 0 || second.GPO.IgnorePrice.Cmp(ethconfig.Defaults.GPO.IgnorePrice) != 0 {
		t.Fatal("shard overrides affected another shard or its defaults")
	}
	second.Miner.GasPrice.SetInt64(999)
	second.GPO.MaxPrice.SetInt64(999)
	second.GPO.IgnorePrice.SetInt64(999)
	if ethconfig.Defaults.Miner.GasPrice.Int64() == 999 || ethconfig.Defaults.GPO.MaxPrice.Int64() == 999 || ethconfig.Defaults.GPO.IgnorePrice.Int64() == 999 {
		t.Fatal("mutable geth defaults are shared with a shard")
	}
	// Omitting the entire Master table still supplies its defaults.
	cfg, err = LoadCluster(writeConfig(t, []byte("[[Slaves]]\nChainID = 100001\nPrivateHost = \"localhost\"\nPrivatePort = 38000\n")))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Master.Role != RoleMaster || cfg.Master.Node.P2P.MaxPeers != node.DefaultConfig.P2P.MaxPeers {
		t.Fatal("omitted master lost its defaults")
	}
}

func TestClusterNestedDefaults(t *testing.T) {
	// Use a true default to distinguish an explicit false from an omitted field.
	savedNoLocals := ethconfig.Defaults.TxPool.NoLocals
	ethconfig.Defaults.TxPool.NoLocals = true
	t.Cleanup(func() { ethconfig.Defaults.TxPool.NoLocals = savedNoLocals })
	for _, test := range []struct {
		name   string
		input  string
		change func(*ethconfig.Config)
	}{
		{"miner gas price", "[Slaves.Eth.Miner]\nGasPrice = \"123\"", func(c *ethconfig.Config) {
			c.Miner.GasPrice = big.NewInt(123)
		}},
		{"miner zeros", "[Slaves.Eth.Miner]\nGasCeil = 0\nRecommit = 0\nGasPrice = \"0\"", func(c *ethconfig.Config) {
			c.Miner.GasCeil, c.Miner.Recommit, c.Miner.GasPrice = 0, 0, new(big.Int)
		}},
		{"GPO price", "[Slaves.Eth.GPO]\nMaxPrice = \"456\"", func(c *ethconfig.Config) {
			c.GPO.MaxPrice = big.NewInt(456)
		}},
		{"GPO zeros", "[Slaves.Eth.GPO]\nBlocks = 0\nMaxPrice = \"0\"", func(c *ethconfig.Config) {
			c.GPO.Blocks, c.GPO.MaxPrice = 0, new(big.Int)
		}},
		{"txpool price bump", "[Slaves.Eth.TxPool]\nPriceBump = 25", func(c *ethconfig.Config) {
			c.TxPool.PriceBump = 25
		}},
		{"txpool zero and false", "[Slaves.Eth.TxPool]\nPriceLimit = 0\nNoLocals = false", func(c *ethconfig.Config) {
			c.TxPool.PriceLimit, c.TxPool.NoLocals = 0, false
		}},
		{"blobpool capacity", "[Slaves.Eth.BlobPool]\nDatacap = 123", func(c *ethconfig.Config) {
			c.BlobPool.Datacap = 123
		}},
		{"blobpool zero", "[Slaves.Eth.BlobPool]\nDatacap = 0", func(c *ethconfig.Config) {
			c.BlobPool.Datacap = 0
		}},
		{"empty tables", "[Slaves.Eth.Miner]\n[Slaves.Eth.GPO]\n[Slaves.Eth.TxPool]\n[Slaves.Eth.BlobPool]", func(*ethconfig.Config) {}},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := "[[Slaves]]\nChainID = 100001\n" + test.input + "\n[[Slaves]]\nChainID = 100002\n"
			cfg, err := LoadCluster(writeConfig(t, []byte(input)))
			if err != nil {
				t.Fatal(err)
			}
			want := ethconfig.Defaults
			test.change(&want)
			got := cfg.Slaves[0].Eth
			for _, prices := range [][2]*big.Int{
				{got.Miner.GasPrice, want.Miner.GasPrice},
				{got.GPO.MaxPrice, want.GPO.MaxPrice},
				{got.GPO.IgnorePrice, want.GPO.IgnorePrice},
			} {
				if prices[0] == nil || prices[0].Cmp(prices[1]) != 0 {
					t.Fatalf("gas price = %v, want %v", prices[0], prices[1])
				}
			}
			// Compare big integers numerically, independently of their representation.
			want.Miner.GasPrice, want.GPO.MaxPrice, want.GPO.IgnorePrice = got.Miner.GasPrice, got.GPO.MaxPrice, got.GPO.IgnorePrice
			if !reflect.DeepEqual(got, &want) {
				t.Fatalf("nested override lost defaults or explicit values:\nhave %+v\nwant %+v", got, &want)
			}
			if !reflect.DeepEqual(cfg.Slaves[1].Eth, &ethconfig.Defaults) {
				t.Fatal("nested overrides affected another shard")
			}
		})
	}
}

func TestProcessSelection(t *testing.T) {
	zero, unknown := uint32(0), uint32(100003)
	for _, test := range []struct {
		role string
		id   *uint32
		want string
	}{
		{"", nil, "set --qkc.role"},
		{"shard", nil, "set --qkc.role"},
		{RoleMaster, &zero, "master cannot select"},
		{RoleSlave, nil, "requires a chain ID"},
		{RoleSlave, &unknown, "not in the deployment"},
		{RoleSlave, &zero, "not in the deployment"},
	} {
		if _, err := exampleCluster(t).SelectNode(test.role, test.id); err == nil || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("have %v, want error containing %q", err, test.want)
		}
	}
}

func TestClusterValidation(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*ClusterConfig)
		want   string
	}{
		{"missing shard", func(c *ClusterConfig) { c.Slaves = c.Slaves[:1] }, "every network shard"},
		{"missing id", func(c *ClusterConfig) { c.Slaves[0].ChainID = nil }, "ChainID is required"},
		{"duplicate shard", func(c *ClusterConfig) { *c.Slaves[1].ChainID = 100001 }, "duplicate deployed chain"},
		{"unknown shard", func(c *ClusterConfig) { *c.Slaves[1].ChainID = 100003 }, "not in the network"},
		{"duplicate cluster endpoint", func(c *ClusterConfig) {
			c.Slaves[1].PrivateHost = c.Slaves[0].PrivateHost
			c.Slaves[1].PrivatePort = c.Slaves[0].PrivatePort
		}, "TCP listeners conflict"},
		{"equivalent IPv6 addresses", func(c *ClusterConfig) {
			c.Slaves[0].PrivateHost = "::1"
			c.Slaves[1].PrivateHost = "0:0:0:0:0:0:0:1"
			c.Slaves[1].PrivatePort = c.Slaves[0].PrivatePort
		}, "TCP listeners conflict"},
		{"IPv4 mapped address", func(c *ClusterConfig) {
			c.Slaves[1].PrivateHost = "::ffff:127.0.0.1"
			c.Slaves[1].PrivatePort = c.Slaves[0].PrivatePort
		}, "TCP listeners conflict"},
		{"shared datadir", func(c *ClusterConfig) { c.Slaves[1].Node.DataDir = c.Master.Node.DataDir }, "data directory is shared"},
		{"master chain id", func(c *ClusterConfig) { c.Master.ChainID = new(uint32) }, "Master must not specify ChainID"},
		{"master execution", func(c *ClusterConfig) { c.Master.Eth = &ethconfig.Config{} }, "master cannot configure Eth"},
		{"master cluster host", func(c *ClusterConfig) { c.Master.PrivateHost = "localhost" }, "master cannot configure Eth"},
		{"master cluster port", func(c *ClusterConfig) { c.Master.PrivatePort = 38002 }, "master cannot configure Eth"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := exampleCluster(t)
			test.change(cfg)
			if err := cfg.Validate(exampleNetwork(t)); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("have %v, want error containing %q", err, test.want)
			}
		})
	}
	cfg := exampleCluster(t)
	cfg.Master.Node.DataDir, cfg.Slaves[0].Node.DataDir, cfg.Slaves[1].Node.DataDir = "", "", ""
	if err := cfg.Validate(exampleNetwork(t)); err != nil {
		t.Fatalf("ephemeral data directories should be supported: %v", err)
	}
}

func TestClusterTCPEndpoints(t *testing.T) {
	for _, test := range []struct {
		name     string
		change   func(*ClusterConfig)
		conflict bool
	}{
		{"duplicate HTTP", func(c *ClusterConfig) {
			c.Slaves[0].Node.HTTPPort = c.Slaves[1].Node.HTTPPort
		}, true},
		{"master and shard HTTP", func(c *ClusterConfig) {
			c.Master.Node.HTTPHost = c.Slaves[0].Node.HTTPHost
			c.Master.Node.HTTPPort = c.Slaves[0].Node.HTTPPort
		}, true},
		{"HTTP and WS across processes", func(c *ClusterConfig) {
			c.Slaves[1].Node.WSHost = c.Slaves[0].Node.HTTPHost
			c.Slaves[1].Node.WSPort = c.Slaves[0].Node.HTTPPort
		}, true},
		{"duplicate WS", func(c *ClusterConfig) {
			c.Slaves[0].Node.WSHost, c.Slaves[1].Node.WSHost = "localhost", "localhost"
		}, true},
		{"equivalent IPv6", func(c *ClusterConfig) {
			c.Slaves[0].Node.HTTPHost = "::1"
			c.Slaves[1].Node.HTTPHost = "0:0:0:0:0:0:0:1"
			c.Slaves[0].Node.HTTPPort = c.Slaves[1].Node.HTTPPort
		}, true},
		{"hostname case", func(c *ClusterConfig) {
			c.Slaves[0].Node.HTTPHost, c.Slaves[1].Node.HTTPHost = "LOCALHOST", "localhost"
			c.Slaves[0].Node.HTTPPort = c.Slaves[1].Node.HTTPPort
		}, true},
		{"wildcard first", func(c *ClusterConfig) {
			c.Slaves[0].Node.HTTPHost = "0.0.0.0"
			c.Slaves[0].Node.HTTPPort = c.Slaves[1].Node.HTTPPort
		}, true},
		{"wildcard second", func(c *ClusterConfig) {
			c.Slaves[1].Node.HTTPHost = "::"
			c.Slaves[0].Node.HTTPPort = c.Slaves[1].Node.HTTPPort
		}, true},
		{"HTTP and own internal listener", func(c *ClusterConfig) {
			c.Slaves[0].Node.HTTPPort = 38000
		}, true},
		{"HTTP and later internal listener", func(c *ClusterConfig) {
			c.Master.Node.HTTPHost, c.Master.Node.HTTPPort = "127.0.0.1", 38001
		}, true},
		{"WS and earlier internal listener", func(c *ClusterConfig) {
			c.Slaves[1].Node.WSHost, c.Slaves[1].Node.WSPort = "127.0.0.1", 38000
		}, true},
		{"default WS host and internal listener", func(c *ClusterConfig) {
			c.Slaves[0].Node.WSHost = node.DefaultWSHost
			c.Slaves[0].Node.WSPort = c.Slaves[0].PrivatePort
		}, true},
		{"internal hostname and HTTP IP", func(c *ClusterConfig) {
			c.Slaves[0].PrivateHost = "localhost"
			c.Slaves[0].Node.HTTPPort = c.Slaves[0].PrivatePort
		}, true},
		{"HTTP hostname and later internal IP", func(c *ClusterConfig) {
			c.Master.Node.HTTPHost, c.Master.Node.HTTPPort = "LOCALHOST", 38000
		}, true},
		{"P2P hostname and HTTP IP", func(c *ClusterConfig) {
			c.Master.Node.P2P.ListenAddr = "localhost:18545"
		}, true},
		{"default WS host and mapped IPv4", func(c *ClusterConfig) {
			c.Slaves[0].PrivateHost = "::ffff:127.0.0.1"
			c.Slaves[0].Node.WSHost = node.DefaultWSHost
			c.Slaves[0].Node.WSPort = c.Slaves[0].PrivatePort
		}, true},
		{"P2P and shard HTTP", func(c *ClusterConfig) {
			c.Slaves[0].Node.HTTPPort = 30303
		}, true},
		{"P2P and master HTTP", func(c *ClusterConfig) {
			c.Master.Node.HTTPHost, c.Master.Node.HTTPPort = "127.0.0.1", 30303
		}, true},
		{"P2P and internal listener", func(c *ClusterConfig) {
			c.Master.Node.P2P.ListenAddr = "127.0.0.1:38000"
		}, true},
		{"P2P and wildcard WS", func(c *ClusterConfig) {
			c.Master.Node.P2P.ListenAddr = "127.0.0.1:30303"
			c.Slaves[0].Node.WSHost, c.Slaves[0].Node.WSPort = "0.0.0.0", 30303
		}, true},
		{"internal listener and mapped IPv4 HTTP", func(c *ClusterConfig) {
			c.Slaves[0].Node.HTTPHost, c.Slaves[0].Node.HTTPPort = "::ffff:127.0.0.1", 38000
		}, true},
		{"shared WS listener and internal listener", func(c *ClusterConfig) {
			c.Slaves[0].Node.HTTPHost = "127.0.0.2"
			c.Slaves[0].Node.WSHost = "127.0.0.1"
			c.Slaves[0].Node.HTTPPort, c.Slaves[0].Node.WSPort = 38000, 38000
		}, true},
		{"dynamic ports", func(c *ClusterConfig) {
			c.Slaves[0].Node.HTTPPort, c.Slaves[1].Node.HTTPPort = 0, 0
		}, false},
		{"disabled HTTP", func(c *ClusterConfig) {
			c.Slaves[0].Node.HTTPHost = ""
			c.Slaves[0].Node.HTTPPort = c.Slaves[1].Node.HTTPPort
		}, false},
		{"same port on different hosts", func(c *ClusterConfig) {
			c.Slaves[0].Node.HTTPHost = "127.0.0.2"
			c.Slaves[0].Node.HTTPPort = c.Slaves[1].Node.HTTPPort
		}, false},
		{"HTTP and WS in one process", func(c *ClusterConfig) {
			c.Slaves[0].Node.WSHost = c.Slaves[0].Node.HTTPHost
			c.Slaves[0].Node.WSPort = c.Slaves[0].Node.HTTPPort
		}, false},
		{"shared listener uses WS host", func(c *ClusterConfig) {
			c.Slaves[0].Node.HTTPPort = c.Slaves[1].Node.HTTPPort
			c.Slaves[0].Node.WSHost = "127.0.0.2"
			c.Slaves[0].Node.WSPort = c.Slaves[0].Node.HTTPPort
		}, false},
		{"shared WS listener avoids internal listener", func(c *ClusterConfig) {
			c.Slaves[0].Node.WSHost = "127.0.0.2"
			c.Slaves[0].Node.HTTPPort, c.Slaves[0].Node.WSPort = 38000, 38000
		}, false},
		{"internal and HTTP on different hosts", func(c *ClusterConfig) {
			c.Slaves[0].Node.HTTPHost, c.Slaves[0].Node.HTTPPort = "127.0.0.2", 38000
		}, false},
		{"default WS host and distinct loopback IP", func(c *ClusterConfig) {
			c.Slaves[0].PrivateHost = "127.0.0.2"
			c.Slaves[0].Node.WSHost = node.DefaultWSHost
			c.Slaves[0].Node.WSPort = c.Slaves[0].PrivatePort
		}, false},
		{"shared WS listener ignores HTTP hostname", func(c *ClusterConfig) {
			c.Slaves[0].Node.HTTPHost, c.Slaves[0].Node.HTTPPort = "localhost", 38000
			c.Slaves[0].Node.WSHost, c.Slaves[0].Node.WSPort = "127.0.0.2", 38000
		}, false},
		{"P2P and HTTP on different hosts", func(c *ClusterConfig) {
			c.Master.Node.P2P.ListenAddr = "127.0.0.2:30303"
			c.Slaves[0].Node.HTTPPort = 30303
		}, false},
		{"disabled P2P listener", func(c *ClusterConfig) {
			c.Master.Node.P2P.ListenAddr = ""
			c.Slaves[0].Node.HTTPPort = 30303
		}, false},
		{"dynamic P2P port", func(c *ClusterConfig) {
			c.Master.Node.P2P.ListenAddr = ":0"
		}, false},
		{"UDP discovery and internal TCP", func(c *ClusterConfig) {
			c.Master.Node.P2P.DiscAddr = ":38000"
		}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := exampleCluster(t)
			test.change(cfg)
			err := cfg.Validate(exampleNetwork(t))
			if test.conflict {
				if err == nil || !strings.Contains(err.Error(), "TCP listeners conflict") {
					t.Fatalf("want TCP listener conflict, got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestClusterIPCEndpoints(t *testing.T) {
	absolute, err := filepath.Abs("ipc/shared.ipc")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name     string
		change   func(*ClusterConfig)
		conflict bool
	}{
		{"shared explicit path", func(c *ClusterConfig) {
			c.Master.Node.IPCPath, c.Slaves[0].Node.IPCPath = "ipc/shared.ipc", "ipc/shared.ipc"
		}, true},
		{"normalized paths", func(c *ClusterConfig) {
			c.Master.Node.IPCPath, c.Slaves[0].Node.IPCPath = "ipc/shared.ipc", "ipc/./shared.ipc"
		}, true},
		{"resolved bare filename", func(c *ClusterConfig) {
			c.Slaves[0].Node.IPCPath = c.Master.Node.IPCEndpoint()
		}, true},
		{"same name with empty datadirs", func(c *ClusterConfig) {
			c.Master.Node.DataDir, c.Slaves[0].Node.DataDir = "", ""
			c.Master.Node.IPCPath, c.Slaves[0].Node.IPCPath = "shared.ipc", "shared.ipc"
		}, true},
		{"same name with distinct datadirs", func(c *ClusterConfig) {
			c.Master.Node.IPCPath, c.Slaves[0].Node.IPCPath = "shared.ipc", "shared.ipc"
		}, runtime.GOOS == "windows"},
		{"absolute and relative paths", func(c *ClusterConfig) {
			c.Master.Node.IPCPath, c.Slaves[0].Node.IPCPath = absolute, "ipc/shared.ipc"
		}, runtime.GOOS != "windows"},
		{"named pipe case", func(c *ClusterConfig) {
			c.Master.Node.IPCPath, c.Slaves[0].Node.IPCPath = "shared.ipc", "SHARED.ipc"
		}, runtime.GOOS == "windows"},
		{"disabled IPC", func(c *ClusterConfig) {
			c.Master.Node.IPCPath, c.Slaves[0].Node.IPCPath = "", ""
		}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := exampleCluster(t)
			test.change(cfg)
			err := cfg.Validate(exampleNetwork(t))
			if test.conflict {
				if err == nil || !strings.Contains(err.Error(), "IPC endpoint is shared") {
					t.Fatalf("want IPC endpoint conflict, got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNodeValidation(t *testing.T) {
	for _, test := range []struct {
		name   string
		master bool
		change func(*NodeConfig)
		want   string
	}{
		{"network mismatch", false, func(c *NodeConfig) { c.Eth.NetworkId = 100001 }, "conflicts with NetworkID"},
		{"local genesis", false, func(c *NodeConfig) { c.Eth.Genesis = &core.Genesis{} }, "Eth.Genesis"},
		{"local fork", false, func(c *NodeConfig) { c.Eth.OverrideOsaka = new(uint64) }, "local fork overrides"},
		{"database engine", true, func(c *NodeConfig) { c.Node.DBEngine = "unknown" }, "Node.DBEngine"},
		{"state scheme", false, func(c *NodeConfig) { c.Eth.StateScheme = "unknown" }, "Eth.StateScheme"},
		{"negative peers", true, func(c *NodeConfig) { c.Node.P2P.MaxPeers = -1 }, "MaxPeers"},
		{"negative cache", false, func(c *NodeConfig) { c.Eth.DatabaseCache = -1 }, "DatabaseCache"},
		{"public port", false, func(c *NodeConfig) { c.Node.HTTPPort = 65536 }, "PublicPort"},
		{"ws port", false, func(c *NodeConfig) { c.Node.WSPort = -1 }, "WSPort"},
		{"auth port", true, func(c *NodeConfig) { c.Node.AuthPort = -1 }, "AuthPort"},
		{"listen address", true, func(c *NodeConfig) { c.Node.P2P.ListenAddr = "invalid" }, "ListenAddr"},
		{"listen port", true, func(c *NodeConfig) { c.Node.P2P.ListenAddr = ":65536" }, "invalid port"},
		{"shard external peers", false, func(c *NodeConfig) { c.Node.P2P.MaxPeers = 1 }, "external P2P must be disabled"},
		{"shard external listen", false, func(c *NodeConfig) { c.Node.P2P.ListenAddr = ":30304" }, "external P2P must be disabled"},
		{"shard external discovery", false, func(c *NodeConfig) { c.Node.P2P.NoDiscovery = false }, "external P2P must be disabled"},
		{"missing cluster host", false, func(c *NodeConfig) { c.PrivateHost = "" }, "PrivateHost must be a concrete host"},
		{"wildcard host", false, func(c *NodeConfig) { c.PrivateHost = "0.0.0.0" }, "concrete host"},
		{"wildcard ipv6", false, func(c *NodeConfig) { c.PrivateHost = "0:0:0:0:0:0:0:0" }, "concrete host"},
		{"host includes port", false, func(c *NodeConfig) { c.PrivateHost = "localhost:38000" }, "without brackets or a port"},
		{"bracketed ipv6", false, func(c *NodeConfig) { c.PrivateHost = "[::1]" }, "without brackets or a port"},
		{"zero cluster port", false, func(c *NodeConfig) { c.PrivatePort = 0 }, "PrivatePort must be between"},
		{"negative cluster port", false, func(c *NodeConfig) { c.PrivatePort = -1 }, "PrivatePort must be between"},
		{"overflow cluster port", false, func(c *NodeConfig) { c.PrivatePort = 65536 }, "PrivatePort must be between"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cluster := exampleCluster(t)
			cfg := &cluster.Slaves[0]
			if test.master {
				cfg = &cluster.Master
			}
			test.change(cfg)
			if err := cfg.Validate(exampleNetwork(t)); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("have %v, want error containing %q", err, test.want)
			}
		})
	}
	cfg := DefaultNodeConfig(nil)
	cfg.Node.HTTPPort, cfg.Node.WSPort, cfg.Node.AuthPort = 0, 0, 0
	if err := cfg.Validate(exampleNetwork(t)); err != nil {
		t.Fatalf("port zero must be valid: %v", err)
	}
	cluster := exampleCluster(t)
	cluster.Slaves[0].PrivateHost = "::1"
	if err := cluster.Validate(exampleNetwork(t)); err != nil {
		t.Fatalf("IPv6 internal endpoints must be valid: %v", err)
	}
	master, err := cluster.SelectNode(RoleMaster, nil)
	if err != nil || master.ShardEndpoints[100001] != "[::1]:38000" {
		t.Fatalf("IPv6 cluster host was not joined correctly: %v", err)
	}
}
