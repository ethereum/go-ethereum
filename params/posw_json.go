// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package params

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/math"
)

// MarshalJSON uses lowerCamelCase field names and emits integers and ratios as
// unquoted decimal numbers. Ratios must have a finite decimal representation.
func (c PoSWConfig) MarshalJSON() ([]byte, error) {
	type configJSON struct {
		AdjustmentCutoff        uint64       `json:"adjustmentCutoff"`              // Output example: 7, without quotes.
		AdjustmentFactor        uint64       `json:"adjustmentFactor"`              // Output example: 512, without quotes.
		MinimumDifficulty       *big.Int     `json:"minimumDifficulty"`             // Output example: 1000000, without quotes; null if nil.
		CoinbaseAmount          *big.Int     `json:"coinbaseAmount"`                // Output example: 6500000000000000000, without quotes; null if nil.
		EpochInterval           uint64       `json:"epochInterval"`                 // Output example: 3153600, without quotes.
		DecayFactor             *json.Number `json:"decayFactor"`                   // Output example: 0.88, without quotes; null if nil.
		OriginalBlockNumber     uint64       `json:"originalBlockNumber,omitempty"` // Output example: 12345678, without quotes; omitted when zero.
		StakingEnabled          bool         `json:"stakingEnabled"`                // Output example: true or false, without quotes.
		WindowSize              uint64       `json:"windowSize"`                    // Output example: 512, without quotes.
		DiffDivider             uint64       `json:"diffDivider"`                   // Output example: 10000, without quotes.
		StakePerBlock           *big.Int     `json:"stakePerBlock"`                 // Output example: 1000000000000000000000000, without quotes; null if nil.
		PoSWStakingDecayEnabled bool         `json:"poswStakingDecayEnabled"`       // Output example: true or false, without quotes.
	}
	decay, err := ratioToJSON(c.DecayFactor)
	if err != nil {
		return nil, fmt.Errorf("posw: decayFactor: %w", err)
	}
	enc := configJSON{
		AdjustmentCutoff:        c.AdjustmentCutoff,
		AdjustmentFactor:        c.AdjustmentFactor,
		MinimumDifficulty:       c.MinimumDifficulty,
		CoinbaseAmount:          c.CoinbaseAmount,
		EpochInterval:           c.EpochInterval,
		DecayFactor:             decay,
		OriginalBlockNumber:     c.OriginalBlockNumber,
		StakingEnabled:          c.StakingEnabled,
		WindowSize:              c.WindowSize,
		DiffDivider:             c.DiffDivider,
		StakePerBlock:           c.StakePerBlock,
		PoSWStakingDecayEnabled: c.PoSWStakingDecayEnabled,
	}
	return json.Marshal(&enc)
}

// UnmarshalJSON accepts decimal numbers, decimal strings and hex strings for
// integers, as core.Genesis does. Ratios accept numbers (0.88) or numeric strings
// ("0.88"). All numeric values are parsed exactly without using float64.
// Omitted or null fields leave the existing configuration values unchanged.
// Call Check separately to validate the consensus rules.
func (c *PoSWConfig) UnmarshalJSON(input []byte) error {
	type configJSON struct {
		AdjustmentCutoff        *math.HexOrDecimal64  `json:"adjustmentCutoff"`              // Supported input: 7, "7" or "0x07".
		AdjustmentFactor        *math.HexOrDecimal64  `json:"adjustmentFactor"`              // Supported input: 512, "512" or "0x200".
		MinimumDifficulty       *math.HexOrDecimal256 `json:"minimumDifficulty"`             // Supported input: 1000000, "1000000" or "0xf4240".
		CoinbaseAmount          *math.HexOrDecimal256 `json:"coinbaseAmount"`                // Supported input: decimal integer or decimal/hex string, e.g. 100, "100" or "0x64".
		EpochInterval           *math.HexOrDecimal64  `json:"epochInterval"`                 // Supported input: 3153600, "3153600" or "0x301ec0".
		DecayFactor             *json.Number          `json:"decayFactor"`                   // Supported input: 0.88 or "0.88"; fraction strings such as "22/25" are rejected.
		OriginalBlockNumber     *math.HexOrDecimal64  `json:"originalBlockNumber,omitempty"` // Supported input: 12345678, "12345678" or "0xbc614e".
		StakingEnabled          *bool                 `json:"stakingEnabled"`                // Supported input: true or false, without quotes.
		WindowSize              *math.HexOrDecimal64  `json:"windowSize"`                    // Supported input: 512, "512" or "0x200".
		DiffDivider             *math.HexOrDecimal64  `json:"diffDivider"`                   // Supported input: 10000, "10000" or "0x2710".
		StakePerBlock           *math.HexOrDecimal256 `json:"stakePerBlock"`                 // Supported input: decimal integer or decimal/hex string, e.g. 100, "100" or "0x64".
		PoSWStakingDecayEnabled *bool                 `json:"poswStakingDecayEnabled"`       // Supported input: true or false, without quotes.
	}
	var dec configJSON
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	decay, err := ratioFromJSON(dec.DecayFactor)
	if err != nil {
		return fmt.Errorf("posw: decayFactor: %w", err)
	}
	if dec.AdjustmentCutoff != nil {
		c.AdjustmentCutoff = uint64(*dec.AdjustmentCutoff)
	}
	if dec.AdjustmentFactor != nil {
		c.AdjustmentFactor = uint64(*dec.AdjustmentFactor)
	}
	if dec.MinimumDifficulty != nil {
		c.MinimumDifficulty = (*big.Int)(dec.MinimumDifficulty)
	}
	if dec.CoinbaseAmount != nil {
		c.CoinbaseAmount = (*big.Int)(dec.CoinbaseAmount)
	}
	if dec.EpochInterval != nil {
		c.EpochInterval = uint64(*dec.EpochInterval)
	}
	if dec.DecayFactor != nil {
		c.DecayFactor = decay
	}
	if dec.OriginalBlockNumber != nil {
		c.OriginalBlockNumber = uint64(*dec.OriginalBlockNumber)
	}
	if dec.StakingEnabled != nil {
		c.StakingEnabled = *dec.StakingEnabled
	}
	if dec.WindowSize != nil {
		c.WindowSize = uint64(*dec.WindowSize)
	}
	if dec.DiffDivider != nil {
		c.DiffDivider = uint64(*dec.DiffDivider)
	}
	if dec.StakePerBlock != nil {
		c.StakePerBlock = (*big.Int)(dec.StakePerBlock)
	}
	if dec.PoSWStakingDecayEnabled != nil {
		c.PoSWStakingDecayEnabled = *dec.PoSWStakingDecayEnabled
	}
	return nil
}

func ratioFromJSON(value *json.Number) (*big.Rat, error) {
	if value == nil {
		return nil, nil
	}
	ratio, ok := new(big.Rat).SetString(value.String())
	if !ok {
		return nil, fmt.Errorf("invalid ratio %q", value.String())
	}
	return ratio, nil
}

// ratioToJSON emits the exact decimal value, without rounding or fraction strings.
func ratioToJSON(value *big.Rat) (*json.Number, error) {
	if value == nil {
		return nil, nil
	}
	precision, exact := value.FloatPrec()
	if !exact {
		return nil, fmt.Errorf("ratio %s has no finite decimal representation", value)
	}
	number := json.Number(value.FloatString(precision))
	return &number, nil
}
