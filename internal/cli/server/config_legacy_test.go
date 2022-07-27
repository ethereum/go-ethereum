package server

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigLegacy(t *testing.T) {

	readFile := func(path string) {
		config, err := readLegacyConfig(path)
		assert.NoError(t, err)

		assert.Equal(t, config, &Config{
			DataDir: "./data",
			RequiredBlocks: map[string]string{
				"a": "b",
			},
			P2P: &P2PConfig{
				MaxPeers: 30,
			},
			TxPool: &TxPoolConfig{
				Locals:    []string{},
				Rejournal: 1 * time.Hour,
				LifeTime:  1 * time.Second,
			},
			Gpo: &GpoConfig{
				MaxPrice:    big.NewInt(100),
				IgnorePrice: big.NewInt(2),
			},
			Sealer: &SealerConfig{
				Enabled:  false,
				GasCeil:  20000000,
				GasPrice: big.NewInt(30000000000),
			},
			Cache: &CacheConfig{
				Cache:     1024,
				Rejournal: 1 * time.Hour,
			},
		})
	}

	// read file in hcl format
	t.Run("toml", func(t *testing.T) {
		readFile("./testdata/test.toml")
	})
}
