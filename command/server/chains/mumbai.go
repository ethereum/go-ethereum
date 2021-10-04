package chains

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
)

var MumbaiGenesisHash = common.HexToHash("0x7b66506a9ebdbf30d32b43c5f15a3b1216269a1ec3a75aa3182b86176a2b1ca7")

var MumbaiChainConfig = &params.ChainConfig{
	ChainID:             big.NewInt(80001),
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
	IstanbulBlock:       big.NewInt(2722000),
	MuirGlacierBlock:    big.NewInt(2722000),
	BerlinBlock:         big.NewInt(13996000),
	Bor: &params.BorConfig{
		Period:                2,
		ProducerDelay:         6,
		Sprint:                64,
		BackupMultiplier:      2,
		ValidatorContract:     "0x0000000000000000000000000000000000001000",
		StateReceiverContract: "0x0000000000000000000000000000000000001001",
	},
}

func DefaultMumbaiGenesisBlock() *core.Genesis {
	return &core.Genesis{
		Config:     MumbaiChainConfig,
		Nonce:      0,
		Timestamp:  1558348305,
		GasLimit:   10000000,
		Difficulty: big.NewInt(1),
		Mixhash:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		Coinbase:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
		Alloc:      readPrealloc("allocs/mumbai.json"),
	}
}
