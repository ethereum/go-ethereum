package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigDefault(t *testing.T) {
	// the default config should work out of the box
	config := DefaultConfig()
	assert.NoError(t, config.loadChain())

	_, err := config.buildNode()
	assert.NoError(t, err)

	_, err = config.buildEth()
	assert.NoError(t, err)
}

func TestConfigMerge(t *testing.T) {
	c0 := &Config{
		Chain: stringPtr("0"),
		Debug: boolPtr(false),
		Whitelist: mapStrPtr(map[string]string{
			"a": "b",
		}),
		P2P: &P2PConfig{
			Discovery: &P2PDiscovery{
				StaticNodes: []string{
					"a",
				},
			},
		},
	}
	c1 := &Config{
		Chain: stringPtr("1"),
		Whitelist: mapStrPtr(map[string]string{
			"b": "c",
		}),
		P2P: &P2PConfig{
			Discovery: &P2PDiscovery{
				StaticNodes: []string{
					"b",
				},
			},
		},
	}
	expected := &Config{
		Chain: stringPtr("1"),
		Debug: boolPtr(false),
		Whitelist: mapStrPtr(map[string]string{
			"a": "b",
			"b": "c",
		}),
		P2P: &P2PConfig{
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
