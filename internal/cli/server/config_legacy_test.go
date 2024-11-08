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

		testConfig.Identity = ""
		testConfig.DataDir = "./data"
		testConfig.KeyStoreDir = "./keystore"
		testConfig.Verbosity = 3
		testConfig.RPCBatchLimit = 0
		testConfig.Snapshot = true
		testConfig.BorLogs = false
		testConfig.RequiredBlocks = map[string]string{
			"31000000": "0x2087b9e2b353209c2c21e370c82daa12278efd0fe5f0febe6c29035352cf050e",
			"32000000": "0x875500011e5eecc0c554f95d07b31cf59df4ca2505f4dbbfffa7d4e4da917c68",
		}
		testConfig.Sealer.GasPrice = big.NewInt(25000000000)
		testConfig.Sealer.Recommit = 20 * time.Second
		testConfig.JsonRPC.RPCEVMTimeout = 5 * time.Second
		testConfig.JsonRPC.TxFeeCap = 6.0
		testConfig.JsonRPC.Http.API = []string{"eth", "bor"}
		testConfig.JsonRPC.Ws.API = []string{""}
		testConfig.Gpo.MaxPrice = big.NewInt(5000000000000)

		assert.Equal(t, expectedConfig, testConfig)
	}

	// read file in hcl format
	t.Run("toml", func(t *testing.T) {
		readFile("./testdata/test.toml")
	})
}

func TestDefaultConfigLegacy(t *testing.T) {
	readFile := func(path string) {
		expectedConfig, err := readLegacyConfig(path)
		assert.NoError(t, err)

		testConfig := DefaultConfig()

		testConfig.Identity = "Polygon-Devs"
		testConfig.DataDir = "/var/lib/bor"

		assert.Equal(t, expectedConfig, testConfig)
	}

	// read file in hcl format
	t.Run("toml", func(t *testing.T) {
		readFile("./testdata/default.toml")
	})
}
