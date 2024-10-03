package params

import (
	"math/big"
	"testing"
)

func TestNetworkIDToChainConfigOrDefault(t *testing.T) {
	tests := []struct {
		name            string
		networkID       *big.Int
		wantChainConfig *ChainConfig
	}{
		{
			"taikoMainnetNetworkID",
			TaikoMainnetNetworkID,
			TaikoChainConfig,
		},
		{
			"taikoInternalL2ANetworkId",
			TaikoInternalL2ANetworkID,
			TaikoChainConfig,
		},
		{
			"taikoInternalL2BNetworkId",
			TaikoInternalL2BNetworkID,
			TaikoChainConfig,
		},
		{
			"snaefoll",
			SnaefellsjokullNetworkID,
			TaikoChainConfig,
		},
		{
			"askja",
			AskjaNetworkID,
			TaikoChainConfig,
		},
		{
			"grimsvotn",
			GrimsvotnNetworkID,
			TaikoChainConfig,
		},
		{
			"eldfellNetworkID",
			EldfellNetworkID,
			TaikoChainConfig,
		},
		{
			"jolnirNetworkID",
			JolnirNetworkID,
			TaikoChainConfig,
		},
		{
			"katlaNetworkID",
			KatlaNetworkID,
			TaikoChainConfig,
		},
		{
			"heklaNetworkID",
			HeklaNetworkID,
			TaikoChainConfig,
		},
		{
			"mainnet",
			MainnetChainConfig.ChainID,
			MainnetChainConfig,
		},
		{
			"sepolia",
			SepoliaChainConfig.ChainID,
			SepoliaChainConfig,
		},
		{
			"doesntExist",
			big.NewInt(89390218390),
			AllEthashProtocolChanges,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if config := NetworkIDToChainConfigOrDefault(tt.networkID); config != tt.wantChainConfig {
				t.Fatalf("expected %v, got %v", config, tt.wantChainConfig)
			}
		})
	}
}
