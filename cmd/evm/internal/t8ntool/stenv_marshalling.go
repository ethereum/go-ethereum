package t8ntool

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
)

// UnmarshalJSON unmarshals from JSON.
func (s *stEnv) UnmarshalJSON(input []byte) error {
	type stEnv struct {
		Coinbase    *common.UnprefixedAddress           `json:"currentCoinbase"`
		Difficulty  *math.HexOrDecimal256               `json:"currentDifficulty"`
		GasLimit    *math.HexOrDecimal64                `json:"currentGasLimit"`
		GasTarget   *math.HexOrDecimal64                `json:"currentGasTarget"`
		BaseFee     *math.HexOrDecimal256               `json:"currentBaseFee,omitempty"`
		Number      *math.HexOrDecimal64                `json:"currentNumber"`
		Timestamp   *math.HexOrDecimal64                `json:"currentTimestamp"`
		BlockHashes map[math.HexOrDecimal64]common.Hash `json:"blockHashes,omitempty"`
		Ommers      []ommer                             `json:"ommers,omitempty"`
	}
	var dec stEnv
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Coinbase == nil {
		return errors.New("missing required field 'currentCoinbase' for stEnv")
	}
	s.Coinbase = common.Address(*dec.Coinbase)
	if dec.Difficulty == nil {
		return errors.New("missing required field 'currentDifficulty' for stEnv")
	}
	s.Difficulty = (*big.Int)(dec.Difficulty)
	switch {
	case dec.GasLimit == nil && dec.GasTarget == nil:
		return errors.New("at least one of 'currentGasLimit' or 'currentGasTarget' must be defined")
	case dec.GasLimit != nil && dec.GasTarget != nil:
		return errors.New("only one of 'currentGasLimit' or 'currentGasTarget' may be defined")
	case dec.GasLimit != nil && dec.GasTarget == nil:
		s.GasLimit = uint64(*dec.GasLimit)
	case dec.GasLimit == nil && dec.GasTarget != nil:
		s.GasLimit = uint64(*dec.GasTarget)
	}
	if dec.BaseFee != nil {
		s.BaseFee = (*big.Int)(dec.BaseFee)
	}
	if dec.Number == nil {
		return errors.New("missing required field 'currentNumber' for stEnv")
	}
	s.Number = uint64(*dec.Number)
	if dec.Timestamp == nil {
		return errors.New("missing required field 'currentTimestamp' for stEnv")
	}
	s.Timestamp = uint64(*dec.Timestamp)
	if dec.BlockHashes != nil {
		s.BlockHashes = dec.BlockHashes
	}
	if dec.Ommers != nil {
		s.Ommers = dec.Ommers
	}
	return nil
}
