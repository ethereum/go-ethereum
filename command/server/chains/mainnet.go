package chains

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
)

var BorMainnetGenesisHash = common.HexToHash("0xa9c28ce2141b56c474f1dc504bee9b01eb1bd7d1a507580d5519d4437a97de1b")

var BorMainnetChainConfig = &params.ChainConfig{
	ChainID:             big.NewInt(137),
	HomesteadBlock:      big.NewInt(0),
	DAOForkBlock:        nil,
	DAOForkSupport:      true,
	EIP150Hash:          common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
	EIP150Block:         big.NewInt(0),
	EIP155Block:         big.NewInt(0),
	EIP158Block:         big.NewInt(0),
	ByzantiumBlock:      big.NewInt(0),
	ConstantinopleBlock: big.NewInt(0),
	PetersburgBlock:     big.NewInt(0),
	IstanbulBlock:       big.NewInt(3395000),
	MuirGlacierBlock:    big.NewInt(3395000),
	BerlinBlock:         big.NewInt(14750000),
	Bor: &params.BorConfig{
		Period:                2,
		ProducerDelay:         6,
		Sprint:                64,
		BackupMultiplier:      2,
		ValidatorContract:     "0x0000000000000000000000000000000000001000",
		StateReceiverContract: "0x0000000000000000000000000000000000001001",
		OverrideStateSyncRecords: map[string]int{
			"14949120": 8,
			"14949184": 0,
			"14953472": 0,
			"14953536": 5,
			"14953600": 0,
			"14953664": 0,
			"14953728": 0,
			"14953792": 0,
			"14953856": 0,
		},
	},
}

//DefaultBorMainnet returns the Bor Mainnet network gensis block.
func DefaultBorMainnetGenesisBlock() *core.Genesis {
	return &core.Genesis{
		Config:     BorMainnetChainConfig,
		Nonce:      0,
		Timestamp:  1590824836,
		GasLimit:   10000000,
		Difficulty: big.NewInt(1),
		Mixhash:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		Coinbase:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
		Alloc:      readPrealloc("allocs/bor_mainnet.json"),
	}
}
