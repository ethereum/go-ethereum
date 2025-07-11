// Copyright 2023 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"errors"

	"github.com/ethereum/go-ethereum/params"
)

// LookupInstructionSet returns the instruction set for the fork configured by
// the rules.
func LookupInstructionSet(rules params.Rules) (JumpTable, error) {
	switch {
	case rules.IsVerkle:
		return newCancunInstructionSet(), errors.New("verkle-fork not defined yet")
	case rules.IsOsaka:
		return newOsakaInstructionSet(), nil
	case rules.IsPrague:
		return newPragueInstructionSet(), nil
	case rules.IsCancun:
		return newCancunInstructionSet(), nil
	case rules.IsShanghai:
		return newShanghaiInstructionSet(), nil
	case rules.IsMerge:
		return newMergeInstructionSet(), nil
	case rules.IsLondon:
		return newLondonInstructionSet(), nil
	case rules.IsBerlin:
		return newBerlinInstructionSet(), nil
	case rules.IsIstanbul:
		return newIstanbulInstructionSet(), nil
	case rules.IsConstantinople:
		return newConstantinopleInstructionSet(), nil
	case rules.IsByzantium:
		return newByzantiumInstructionSet(), nil
	case rules.IsEIP158:
		return newSpuriousDragonInstructionSet(), nil
	case rules.IsEIP150:
		return newTangerineWhistleInstructionSet(), nil
	case rules.IsHomestead:
		return newHomesteadInstructionSet(), nil
	}
	return newFrontierInstructionSet(), nil
}

// Stack returns the minimum and maximum stack requirements.
func (op *operation) Stack() (int, int) {
	return op.minStack, op.maxStack
}
