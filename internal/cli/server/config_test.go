package server

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigDefault(t *testing.T) {
	// the default config should work out of the box
	config := DefaultConfig()
	assert.NoError(t, config.loadChain())

	_, err := config.buildNode()
	assert.NoError(t, err)

	_, err = config.buildEth(nil)
	assert.NoError(t, err)
}

func TestConfigMerge(t *testing.T) {
	c0 := &Config{
		Chain:    "0",
		Snapshot: true,
		Whitelist: map[string]string{
			"a": "b",
		},
		TxPool: &TxPoolConfig{
			LifeTime: 5 * time.Second,
		},
		P2P: &P2PConfig{
			Discovery: &P2PDiscovery{
				StaticNodes: []string{
					"a",
				},
			},
		},
	}
	c1 := &Config{
		Chain: "1",
		Whitelist: map[string]string{
			"b": "c",
		},
		P2P: &P2PConfig{
			MaxPeers: 10,
			Discovery: &P2PDiscovery{
				StaticNodes: []string{
					"b",
				},
			},
		},
	}
	expected := &Config{
		Chain:    "1",
		Snapshot: true,
		Whitelist: map[string]string{
			"a": "b",
			"b": "c",
		},
		TxPool: &TxPoolConfig{
			LifeTime: 5 * time.Second,
		},
		P2P: &P2PConfig{
			MaxPeers: 10,
			Discovery: &P2PDiscovery{
				StaticNodes: []string{
					"a",
					"b",
				},
			},
		},
	}
	assert.NoError(t, c0.Merge(c1))
	assert.Equal(t, c0, expected)
}

func TestConfigLoadFile(t *testing.T) {
	readFile := func(path string) {
		config, err := readConfigFile(path)
		assert.NoError(t, err)
		assert.Equal(t, config, &Config{
			DataDir: "./data",
			Whitelist: map[string]string{
				"a": "b",
			},
			P2P: &P2PConfig{
				MaxPeers: 30,
			},
			TxPool: &TxPoolConfig{
				LifeTime: time.Duration(1 * time.Second),
			},
			Gpo: &GpoConfig{
				MaxPrice: big.NewInt(100),
			},
			Sealer: &SealerConfig{},
			Cache:  &CacheConfig{},
		})
	}

	// read file in hcl format
	t.Run("hcl", func(t *testing.T) {
		readFile("./testdata/simple.hcl")
	})
	// read file in json format
	t.Run("json", func(t *testing.T) {
		readFile("./testdata/simple.json")
	})
}

var dummyEnodeAddr = "enode://0cb82b395094ee4a2915e9714894627de9ed8498fb881cec6db7c65e8b9a5bd7f2f25cc84e71e89d0947e51c76e85d0847de848c7782b13c0255247a6758178c@44.232.55.71:30303"

func TestConfigBootnodesDefault(t *testing.T) {
	t.Run("EmptyBootnodes", func(t *testing.T) {
		// if no bootnodes are specific, we use the ones from the genesis chain
		config := DefaultConfig()
		assert.NoError(t, config.loadChain())

		cfg, err := config.buildNode()
		assert.NoError(t, err)
		assert.NotEmpty(t, cfg.P2P.BootstrapNodes)
	})
	t.Run("NotEmptyBootnodes", func(t *testing.T) {
		// if bootnodes specific, DO NOT load the genesis bootnodes
		config := DefaultConfig()
		config.P2P.Discovery.Bootnodes = []string{dummyEnodeAddr}

		cfg, err := config.buildNode()
		assert.NoError(t, err)
		assert.Len(t, cfg.P2P.BootstrapNodes, 1)
	})
}
