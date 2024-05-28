package chains

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
)

var amoyTestnet = &Chain{
	NetworkId: 80002,
	Genesis: &core.Genesis{
		Config: &params.ChainConfig{
			ChainID:             big.NewInt(80002),
			HomesteadBlock:      big.NewInt(0),
			DAOForkBlock:        nil,
			DAOForkSupport:      true,
			EIP150Block:         big.NewInt(0),
			EIP155Block:         big.NewInt(0),
			EIP158Block:         big.NewInt(0),
			ByzantiumBlock:      big.NewInt(0),
			ConstantinopleBlock: big.NewInt(0),
			PetersburgBlock:     big.NewInt(0),
			IstanbulBlock:       big.NewInt(0),
			MuirGlacierBlock:    big.NewInt(0),
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(73100),
			ShanghaiBlock:       big.NewInt(73100),
			CancunBlock:         big.NewInt(5423600),
			Bor: &params.BorConfig{
				JaipurBlock: big.NewInt(73100),
				DelhiBlock:  big.NewInt(73100),
				IndoreBlock: big.NewInt(73100),
				StateSyncConfirmationDelay: map[string]uint64{
					"0": 128,
				},
				Period: map[string]uint64{
					"0": 2,
				},
				ProducerDelay: map[string]uint64{
					"0": 4,
				},
				Sprint: map[string]uint64{
					"0": 16,
				},
				BackupMultiplier: map[string]uint64{
					"0": 2,
				},
				ValidatorContract:     "0x0000000000000000000000000000000000001000",
				StateReceiverContract: "0x0000000000000000000000000000000000001001",
				BurntContract: map[string]string{
					"0":     "0x000000000000000000000000000000000000dead",
					"73100": "0xeCDD77cE6f146cCf5dab707941d318Bd50eeD2C9",
				},
			},
		},
		Nonce:      0,
		Timestamp:  1700225065,
		GasLimit:   10000000,
		Difficulty: big.NewInt(1),
		Mixhash:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		Coinbase:   common.HexToAddress("0x0000000000000000000000000000000000000000"),
		Alloc:      readPrealloc("allocs/amoy.json"),
	},
	Bootnodes: []string{
		"enode://bce861be777e91b0a5a49d58a51e14f32f201b4c6c2d1fbea6c7a1f14756cbb3f931f3188d6b65de8b07b53ff28d03b6e366d09e56360d2124a9fc5a15a0913d@54.217.171.196:30303",
		"enode://4a3dc0081a346d26a73d79dd88216a9402d2292318e2db9947dbc97ea9c4afb2498dc519c0af04420dc13a238c279062da0320181e7c1461216ce4513bfd40bf@13.251.184.185:30303",
	},
}
