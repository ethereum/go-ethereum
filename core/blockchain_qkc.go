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

package core

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var _ consensus.StakingReader = (*BlockChain)(nil)

// GetLockedStakes executes the shard staking getter at the requested parent
// state, never the current canonical state or the candidate's post-state. Legacy
// shard PoSW used spendable account balances; explicit contract locking replaces
// that rule here. getLockedStakes returns zero as soon as an account is unlocked.
func (bc *BlockChain) GetLockedStakes(parent *types.Header, staker common.Address) (*uint256.Int, error) {
	statedb, err := bc.StateAt(parent.Root, parent.Number, parent.Time)
	if err != nil {
		return nil, err
	}
	address := params.ShardStakingContractAddress
	if statedb.GetCodeHash(address) != params.PoSWStakingRuntimeCodeHash {
		return nil, errors.New("invalid shard staking contract runtime")
	}
	rules := bc.chainConfig.Rules(parent.Number, true, parent.Time)
	statedb.Prepare(rules, common.Address{}, common.Address{}, &address, vm.ActivePrecompiles(rules), nil)
	input := make([]byte, 36)
	// getLockedStakes(address); the second return value (root signer) is unused.
	copy(input, common.FromHex("0xfd8c4646"))
	copy(input[16:], staker[:])
	evm := vm.NewEVM(NewEVMBlockContext(parent, bc, nil), statedb, bc.chainConfig, vm.Config{})
	output, _, err := evm.StaticCall(common.Address{}, address, input, vm.NewGasBudget(100_000, 100_000))
	if err != nil {
		return nil, fmt.Errorf("getLockedStakes: %w", err)
	}
	if err := statedb.Error(); err != nil {
		return nil, err
	}
	if len(output) != 64 {
		return nil, fmt.Errorf("invalid getLockedStakes response length %d", len(output))
	}
	return new(uint256.Int).SetBytes(output[:32]), nil
}
