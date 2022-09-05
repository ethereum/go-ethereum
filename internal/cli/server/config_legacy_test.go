package server

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/params"
)

func TestConfigLegacy(t *testing.T) {

	readFile := func(path string) {
		expectedConfig, err := readLegacyConfig(path)
		assert.NoError(t, err)

		testConfig := &Config{
			Chain:    "mainnet",
			Identity: Hostname(),
			RequiredBlocks: map[string]string{
				"a": "b",
			},
			LogLevel: "INFO",
			DataDir:  "./data",
			P2P: &P2PConfig{
				MaxPeers:     30,
				MaxPendPeers: 50,
				Bind:         "0.0.0.0",
				Port:         30303,
				NoDiscover:   false,
				NAT:          "any",
				Discovery: &P2PDiscovery{
					V5Enabled:    false,
					Bootnodes:    []string{},
					BootnodesV4:  []string{},
					BootnodesV5:  []string{},
					StaticNodes:  []string{},
					TrustedNodes: []string{},
					DNS:          []string{},
				},
			},
			Heimdall: &HeimdallConfig{
				URL:     "http://localhost:1317",
				Without: false,
			},
			SyncMode: "full",
			GcMode:   "full",
			Snapshot: true,
			TxPool: &TxPoolConfig{
				Locals:       []string{},
				NoLocals:     false,
				Journal:      "transactions.rlp",
				Rejournal:    1 * time.Hour,
				PriceLimit:   1,
				PriceBump:    10,
				AccountSlots: 16,
				GlobalSlots:  32768,
				AccountQueue: 16,
				GlobalQueue:  32768,
				LifeTime:     1 * time.Second,
			},
			Sealer: &SealerConfig{
				Enabled:   false,
				Etherbase: "",
				GasCeil:   30000000,
				GasPrice:  big.NewInt(1 * params.GWei),
				ExtraData: "",
			},
			Gpo: &GpoConfig{
				Blocks:      20,
				Percentile:  60,
				MaxPrice:    big.NewInt(5000 * params.GWei),
				IgnorePrice: big.NewInt(4),
			},
			JsonRPC: &JsonRPCConfig{
				IPCDisable: false,
				IPCPath:    "",
				GasCap:     ethconfig.Defaults.RPCGasCap,
				TxFeeCap:   ethconfig.Defaults.RPCTxFeeCap,
				Http: &APIConfig{
					Enabled: false,
					Port:    8545,
					Prefix:  "",
					Host:    "localhost",
					API:     []string{"eth", "net", "web3", "txpool", "bor"},
					Cors:    []string{"localhost"},
					VHost:   []string{"localhost"},
				},
				Ws: &APIConfig{
					Enabled: false,
					Port:    8546,
					Prefix:  "",
					Host:    "localhost",
					API:     []string{"net", "web3"},
					Cors:    []string{"localhost"},
					VHost:   []string{"localhost"},
				},
				Graphql: &APIConfig{
					Enabled: false,
					Cors:    []string{"localhost"},
					VHost:   []string{"localhost"},
				},
			},
			Ethstats: "",
			Telemetry: &TelemetryConfig{
				Enabled:               false,
				Expensive:             false,
				PrometheusAddr:        "",
				OpenCollectorEndpoint: "",
				InfluxDB: &InfluxDBConfig{
					V1Enabled:    false,
					Endpoint:     "",
					Database:     "",
					Username:     "",
					Password:     "",
					Tags:         map[string]string{},
					V2Enabled:    false,
					Token:        "",
					Bucket:       "",
					Organization: "",
				},
			},
			Cache: &CacheConfig{
				Cache:         1024,
				PercDatabase:  50,
				PercTrie:      15,
				PercGc:        25,
				PercSnapshot:  10,
				Journal:       "triecache",
				Rejournal:     1 * time.Second,
				NoPrefetch:    false,
				Preimages:     false,
				TxLookupLimit: 2350000,
			},
			Accounts: &AccountsConfig{
				Unlock:              []string{},
				PasswordFile:        "",
				AllowInsecureUnlock: false,
				UseLightweightKDF:   false,
				DisableBorWallet:    true,
			},
			GRPC: &GRPCConfig{
				Addr: ":3131",
			},
			Developer: &DeveloperConfig{
				Enabled: false,
				Period:  0,
			},
		}

		assert.Equal(t, expectedConfig, testConfig)
	}

	// read file in hcl format
	t.Run("toml", func(t *testing.T) {
		readFile("./testdata/test.toml")
	})
}
