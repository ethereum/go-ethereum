package server

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigLegacy(t *testing.T) {

	readFile := func(path string) {
		expectedConfig, err := readLegacyConfig(path)
		assert.NoError(t, err)

		testConfig := DefaultConfig()

		testConfig.DataDir = "./data"
		testConfig.Snapshot = false
		testConfig.RequiredBlocks = map[string]string{
			"31000000": "0x2087b9e2b353209c2c21e370c82daa12278efd0fe5f0febe6c29035352cf050e",
			"32000000": "0x875500011e5eecc0c554f95d07b31cf59df4ca2505f4dbbfffa7d4e4da917c68",
		}
		testConfig.P2P.MaxPeers = 30
		testConfig.TxPool.Locals = []string{}
		testConfig.TxPool.LifeTime = time.Second
		testConfig.Sealer.Enabled = true
		testConfig.Sealer.GasCeil = 30000000
		testConfig.Sealer.GasPrice = big.NewInt(1000000000)
		testConfig.Gpo.IgnorePrice = big.NewInt(4)
		testConfig.Cache.Cache = 1024
		testConfig.Cache.Rejournal = time.Second

		assert.Equal(t, expectedConfig, testConfig)
	}

	// read file in hcl format
	t.Run("toml", func(t *testing.T) {
		readFile("./testdata/test.toml")
	})
}
