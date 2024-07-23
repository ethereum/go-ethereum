package server

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/params"
)

func TestConfigDefault(t *testing.T) {
	// the default config should work out of the box
	config := DefaultConfig()
	assert.NoError(t, config.loadChain())

	_, err := config.buildNode()
	assert.NoError(t, err)

	ethConfig, err := config.buildEth(nil, nil)
	assert.NoError(t, err)
	assertBorDefaultGasPrice(t, ethConfig)
}

// assertBorDefaultGasPrice asserts the bor default gas price is set correctly.
func assertBorDefaultGasPrice(t *testing.T, ethConfig *ethconfig.Config) {
	assert.NotNil(t, ethConfig)
	assert.Equal(t, ethConfig.Miner.GasPrice, big.NewInt(params.BorDefaultMinerGasPrice))
}

func TestConfigMerge(t *testing.T) {
	c0 := &Config{
		Chain:    "0",
		Snapshot: true,
		RequiredBlocks: map[string]string{
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
		RequiredBlocks: map[string]string{
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
		Snapshot: false,
		RequiredBlocks: map[string]string{
			"a": "b",
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

	assert.NoError(t, c0.Merge(c1))
	assert.Equal(t, c0, expected)
}

func TestDefaultDatatypeOverride(t *testing.T) {
	t.Parallel()

	// This test is specific to `maxpeers` flag (for now) to check
	// if default datatype value (0 in case of uint64) is overridden.
	c0 := &Config{
		P2P: &P2PConfig{
			MaxPeers: 30,
		},
	}

	c1 := &Config{
		P2P: &P2PConfig{
			MaxPeers: 0,
		},
	}

	expected := &Config{
		P2P: &P2PConfig{
			MaxPeers: 0,
		},
	}

	assert.NoError(t, c0.Merge(c1))
	assert.Equal(t, c0, expected)
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

func TestMakePasswordListFromFile(t *testing.T) {
	t.Parallel()

	t.Run("ReadPasswordFile", func(t *testing.T) {
		t.Parallel()

		result, _ := MakePasswordListFromFile("./testdata/password.txt")
		assert.Equal(t, []string{"test1", "test2"}, result)
	})
}
