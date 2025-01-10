package params

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

func u64(val uint64) *uint64 { return &val }

// Network IDs
var (
	TaikoMainnetNetworkID     = big.NewInt(167000)
	TaikoInternalL2ANetworkID = big.NewInt(167001)
	TaikoInternalL2BNetworkID = big.NewInt(167002)
	SnaefellsjokullNetworkID  = big.NewInt(167003)
	AskjaNetworkID            = big.NewInt(167004)
	GrimsvotnNetworkID        = big.NewInt(167005)
	EldfellNetworkID          = big.NewInt(167006)
	JolnirNetworkID           = big.NewInt(167007)
	KatlaNetworkID            = big.NewInt(167008)
	HeklaNetworkID            = big.NewInt(167009)
)

var networkIDToChainConfig = map[*big.Int]*ChainConfig{
	TaikoMainnetNetworkID:      TaikoChainConfig,
	TaikoInternalL2ANetworkID:  TaikoChainConfig,
	TaikoInternalL2BNetworkID:  TaikoChainConfig,
	SnaefellsjokullNetworkID:   TaikoChainConfig,
	AskjaNetworkID:             TaikoChainConfig,
	GrimsvotnNetworkID:         TaikoChainConfig,
	EldfellNetworkID:           TaikoChainConfig,
	JolnirNetworkID:            TaikoChainConfig,
	KatlaNetworkID:             TaikoChainConfig,
	HeklaNetworkID:             TaikoChainConfig,
	MainnetChainConfig.ChainID: MainnetChainConfig,
	SepoliaChainConfig.ChainID: SepoliaChainConfig,
	TestChainConfig.ChainID:    TestChainConfig,
	NonActivatedConfig.ChainID: NonActivatedConfig,
}

func NetworkIDToChainConfigOrDefault(networkID *big.Int) *ChainConfig {
	if config, ok := networkIDToChainConfig[networkID]; ok {
		return config
	}

	return AllEthashProtocolChanges
}

var TaikoChainConfig = &ChainConfig{
	ChainID:                       TaikoInternalL2ANetworkID, // Use Internal Devnet network ID by default.
	HomesteadBlock:                common.Big0,
	EIP150Block:                   common.Big0,
	EIP155Block:                   common.Big0,
	EIP158Block:                   common.Big0,
	ByzantiumBlock:                common.Big0,
	ConstantinopleBlock:           common.Big0,
	PetersburgBlock:               common.Big0,
	IstanbulBlock:                 common.Big0,
	BerlinBlock:                   common.Big0,
	LondonBlock:                   common.Big0,
	ShanghaiTime:                  u64(0),
	MergeNetsplitBlock:            nil,
	TerminalTotalDifficulty:       common.Big0,
	TerminalTotalDifficultyPassed: true,
	Taiko:                         true,
}
