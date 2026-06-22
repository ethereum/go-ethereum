// Ported verbatim from github.com/QuarkChain/goquarkchain/cluster/config (byte-compatible).

package config

import (
	"encoding/json"
	"math/big"

	ethcom "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/qkc/account"
)

type ChainConfig struct {
	ChainID           uint32 `json:"CHAIN_ID"`
	ShardSize         uint32 `json:"SHARD_SIZE"`
	DefaultChainToken string `json:"DEFAULT_CHAIN_TOKEN"`
	ConsensusType     string `json:"CONSENSUS_TYPE"`

	// Only set when CONSENSUS_TYPE is not NONE
	ConsensusConfig *POWConfig    `json:"CONSENSUS_CONFIG"`
	Genesis         *ShardGenesis `json:"GENESIS"`

	CoinbaseAddress account.Address `json:"-"`
	CoinbaseAmount  *big.Int        `json:"COINBASE_AMOUNT"`
	EpochInterval   uint64          `json:"EPOCH_INTERVAL"`

	DifficultyAdjustmentCutoffTime uint32      `json:"DIFFICULTY_ADJUSTMENT_CUTOFF_TIME"`
	DifficultyAdjustmentFactor     uint32      `json:"DIFFICULTY_ADJUSTMENT_FACTOR"`
	ExtraShardBlocksInRootBlock    uint32      `json:"EXTRA_SHARD_BLOCKS_IN_ROOT_BLOCK"`
	PoswConfig                     *POSWConfig `json:"POSW_CONFIG"`
}

func NewChainConfig() *ChainConfig {
	return &ChainConfig{
		ChainID:                        0,
		ShardSize:                      2,
		DefaultChainToken:              DefaultToken,
		ConsensusType:                  PoWNone,
		ConsensusConfig:                nil,
		Genesis:                        NewShardGenesis(),
		CoinbaseAmount:                 new(big.Int).Mul(big.NewInt(5), QuarkashToJiaozi),
		DifficultyAdjustmentCutoffTime: 7,
		DifficultyAdjustmentFactor:     512,
		ExtraShardBlocksInRootBlock:    3,
		PoswConfig:                     NewPOSWConfig(),
		EpochInterval:                  uint64(210000 * 60),
	}
}

type ChainConfigAlias ChainConfig

func (c *ChainConfig) MarshalJSON() ([]byte, error) {
	addr := c.CoinbaseAddress.ToHex()
	jsonConfig := struct {
		ChainConfigAlias
		CoinbaseAddress string `json:"COINBASE_ADDRESS"`
	}{ChainConfigAlias: ChainConfigAlias(*c), CoinbaseAddress: addr}
	return json.Marshal(jsonConfig)
}

func (c *ChainConfig) UnmarshalJSON(input []byte) error {
	var jsonConfig struct {
		ChainConfigAlias
		CoinbaseAddress string `json:"COINBASE_ADDRESS"`
	}
	if err := json.Unmarshal(input, &jsonConfig); err != nil {
		return err
	}
	*c = ChainConfig(jsonConfig.ChainConfigAlias)
	address, err := account.CreatAddressFromBytes(ethcom.FromHex(jsonConfig.CoinbaseAddress))
	if err != nil {
		return err
	}
	c.CoinbaseAddress = address
	return nil
}
