package beacon

import (
	"errors"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/configs"
)

const MainnetType = iota

type BaseConfig struct {
	APIPort           uint64
	API               string
	DefaultCheckpoint common.Root
	Chain             ChainConfig
	Spec              *common.Spec
	MaxCheckpointAge  uint64
}

func Mainnet() *BaseConfig {
	return &BaseConfig{
		APIPort:           8545,
		API:               "https://www.lightclientdata.org",
		DefaultCheckpoint: common.Root(hexutil.MustDecode("0x766647f3c4e1fc91c0db9a9374032ae038778411fbff222974e11f2e3ce7dadf")),
		Chain: ChainConfig{
			ChainID:     1,
			GenesisTime: 1606824023,
			GenesisRoot: common.Root(hexutil.MustDecode("0x4b363db94e286120d76eb905340fdd4e54bfe9f06bf33ff6cf5ad27f511bfe95")),
		},
		Spec:             configs.Mainnet,
		MaxCheckpointAge: 1_209_600,
	}
}

func ToBaseConfig(networkType int) (*BaseConfig, error) {
	switch networkType {
	case MainnetType:
		return Mainnet(), nil
	default:
		return nil, errors.New("unknown network type")
	}
}
