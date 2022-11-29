package params

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Network IDs
var (
	TaikoMainnetNetworkID = big.NewInt(167)
	TaikoAlpha1NetworkID  = big.NewInt(167001)
	TaikoAlpha2NetworkID  = big.NewInt(167002)
)

var TaikoChainConfig = &ChainConfig{
	ChainID:                       TaikoMainnetNetworkID, // Use mainnet network ID by default.
	HomesteadBlock:                common.Big0,
	EIP150Block:                   common.Big0,
	EIP155Block:                   common.Big0,
	EIP158Block:                   common.Big0,
	ByzantiumBlock:                common.Big0,
	ConstantinopleBlock:           common.Big0,
	PetersburgBlock:               common.Big0,
	IstanbulBlock:                 common.Big0,
	BerlinBlock:                   common.Big0,
	LondonBlock:                   nil,
	MergeNetsplitBlock:            nil,
	TerminalTotalDifficulty:       common.Big0,
	TerminalTotalDifficultyPassed: true,
	Taiko:                         true,
}
