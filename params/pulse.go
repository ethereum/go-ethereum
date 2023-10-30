package params

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common/math"
)

// Optional treasury for launching PulseChain testnets
type Treasury struct {
	Addr    string                `json:"addr"`
	Balance *math.HexOrDecimal256 `json:"balance"`
}

// A trivially small amount of work to add to the Ethereum Mainnet TTD
// to allow for un-merging and merging with the PulseChain beacon chain
var PulseChainTTDOffset = big.NewInt(131_072)

// This value is defined as LAST_ACTUAL_MAINNET_TTD + PulseChainTTDOffset
// where LAST_ACTUAL_MAINNET_TTD = 58_750_003_716_598_352_816_469
var PulseChainTerminalTotalDifficulty, _ = new(big.Int).SetString("58_750_003_716_598_352_947_541", 0)

var (
	PulseChainConfig = &ChainConfig{
		ChainID:                       big.NewInt(369),
		HomesteadBlock:                big.NewInt(1_150_000),
		DAOForkBlock:                  big.NewInt(1_920_000),
		DAOForkSupport:                true,
		EIP150Block:                   big.NewInt(2_463_000),
		EIP155Block:                   big.NewInt(2_675_000),
		EIP158Block:                   big.NewInt(2_675_000),
		ByzantiumBlock:                big.NewInt(4_370_000),
		ConstantinopleBlock:           big.NewInt(7_280_000),
		PetersburgBlock:               big.NewInt(7_280_000),
		IstanbulBlock:                 big.NewInt(9_069_000),
		MuirGlacierBlock:              big.NewInt(9_200_000),
		BerlinBlock:                   big.NewInt(12_244_000),
		LondonBlock:                   big.NewInt(12_965_000),
		ArrowGlacierBlock:             big.NewInt(13_773_000),
		GrayGlacierBlock:              big.NewInt(15_050_000),
		TerminalTotalDifficulty:       PulseChainTerminalTotalDifficulty,
		TerminalTotalDifficultyPassed: true,
		Ethash:                        new(EthashConfig),
		PrimordialPulseBlock:          big.NewInt(17_233_000),
		ShanghaiTime:                  newUint64(1683786515),
	}

	PulseChainTestnetV4Config = &ChainConfig{
		ChainID:                       big.NewInt(943),
		HomesteadBlock:                big.NewInt(1_150_000),
		DAOForkBlock:                  big.NewInt(1_920_000),
		DAOForkSupport:                true,
		EIP150Block:                   big.NewInt(2_463_000),
		EIP155Block:                   big.NewInt(2_675_000),
		EIP158Block:                   big.NewInt(2_675_000),
		ByzantiumBlock:                big.NewInt(4_370_000),
		ConstantinopleBlock:           big.NewInt(7_280_000),
		PetersburgBlock:               big.NewInt(7_280_000),
		IstanbulBlock:                 big.NewInt(9_069_000),
		MuirGlacierBlock:              big.NewInt(9_200_000),
		BerlinBlock:                   big.NewInt(12_244_000),
		LondonBlock:                   big.NewInt(12_965_000),
		ArrowGlacierBlock:             big.NewInt(13_773_000),
		GrayGlacierBlock:              big.NewInt(15_050_000),
		TerminalTotalDifficulty:       PulseChainTerminalTotalDifficulty,
		TerminalTotalDifficultyPassed: true,
		Ethash:                        new(EthashConfig),
		PrimordialPulseBlock:          big.NewInt(16_492_700),
		Treasury:                      testnetTreasury(),
		ShanghaiTime:                  newUint64(1682700369),
	}
)

func testnetTreasury() *Treasury {
	var pulseChainTestnetTreasuryBalance math.HexOrDecimal256
	pulseChainTestnetTreasuryBalance.UnmarshalText([]byte("0x314DC6448D9338C15B0A00000000"))

	return &Treasury{
		Addr:    "0xA592ED65885bcbCeb30442F4902a0D1Cf3AcB8fC",
		Balance: &pulseChainTestnetTreasuryBalance,
	}
}
