// Copyright 2026 The quarkchain Authors
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

package posw

import (
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// PoSWSimulate retains PoSW rules while skipping Ethash seal verification.
type PoSWSimulate struct {
	*PoSW
}

var _ consensus.Engine = (*PoSWSimulate)(nil)
var _ consensus.SealVerifier = (*PoSWSimulate)(nil)

// NewSimulate creates a simulation engine with the same configuration checks as New.
func NewSimulate(config *params.QuarkChainConfig, genesis *types.Header) (*PoSWSimulate, error) {
	engine, err := New(config, genesis)
	if err != nil {
		return nil, err
	}
	return &PoSWSimulate{PoSW: engine}, nil
}

// VerifySeal computes parent-state adjusted difficulty without performing Ethash work.
// As with PoSW, the header must already be validated and its parent state available.
func (p *PoSWSimulate) VerifySeal(chain consensus.ChainHeaderReader, header *types.Header) error {
	reader, _ := chain.(consensus.StakingReader)
	difficulty, err := GetAdjustedDifficulty(p.config.PoSW, reader, header)
	if err != nil {
		return err
	}
	if difficulty.Sign() <= 0 {
		return errInvalidDifficulty
	}
	return nil
}
