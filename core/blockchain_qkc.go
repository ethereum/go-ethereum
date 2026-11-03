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

// getLockedStakesSelector is bytes4(keccak256("getLockedStakes(address)")).
// Obtain with: cast sig "getLockedStakes(address)"
const getLockedStakesSelector = "0xfd8c4646"

var _ consensus.StakingReader = (*BlockChain)(nil)

// GetShardLockedStakes executes the shard staking getter at header's state.
// Shard consensus supplies the candidate's parent header. Legacy
// shard PoSW used spendable account balances; explicit contract locking replaces
// that rule here. getLockedStakes returns zero as soon as an account is unlocked.
func (bc *BlockChain) GetShardLockedStakes(header *types.Header, staker common.Address) (*uint256.Int, error) {
	stakes, _, err := bc.getLockedStakes(header, params.ShardStakingContractAddress, staker)
	return stakes, err
}

// GetRootChainStakes reads root-chain stakes and the mining signer from shard 0.
// The caller selects the shard block confirmed by the candidate root's parent;
// header is that shard block's header, not a root-block header.
func (bc *BlockChain) GetRootChainStakes(header *types.Header, staker common.Address) (*uint256.Int, common.Address, error) {
	if bc.chainConfig.QuarkChain == nil || bc.chainConfig.QuarkChain.ChainID != 0 {
		return nil, common.Address{}, errors.New("root-chain stakes are only available on shard 0")
	}
	return bc.getLockedStakes(header, params.RootStakingContractAddress, staker)
}

// getLockedStakes calls the selected staking contract at the specified shard state.
func (bc *BlockChain) getLockedStakes(header *types.Header, address, staker common.Address) (*uint256.Int, common.Address, error) {
	if !bc.HasState(header.Root) {
		return nil, common.Address{}, consensus.ErrPrunedAncestor
	}
	statedb, err := bc.StateAt(header.Root, header.Number, header.Time)
	if err != nil {
		return nil, common.Address{}, err
	}
	codeHash := statedb.GetCodeHash(address)
	if err := statedb.Error(); err != nil {
		return nil, common.Address{}, err
	}
	if codeHash != params.PoSWStakingRuntimeCodeHash {
		return nil, common.Address{}, errors.New("invalid staking contract runtime")
	}
	evm := vm.NewEVM(NewEVMBlockContext(header, bc, nil), statedb, bc.chainConfig, vm.Config{})
	defer evm.Release()
	evm.SetTxContext(vm.TxContext{})
	rules := evm.GetRules()
	statedb.Prepare(rules, common.Address{}, common.Address{}, &address, vm.ActivePrecompiles(rules), nil)
	// ABI calldata: a 4-byte selector followed by a 32-byte address word.
	// The 20 address bytes start after 12 padding bytes, at offset 4 + 12 = 16.
	input := make([]byte, 36)
	copy(input, common.FromHex(getLockedStakesSelector))
	copy(input[16:], staker[:])
	output, _, err := evm.StaticCall(common.Address{}, address, input, vm.NewGasBudget(1_000_000, 0))
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("getLockedStakes: %w", err)
	}
	if err := statedb.Error(); err != nil {
		return nil, common.Address{}, err
	}
	// ABI result: two 32-byte words (amount, signer); signer starts at 32 + 12 = 44.
	if len(output) != 64 {
		return nil, common.Address{}, fmt.Errorf("invalid getLockedStakes response length %d", len(output))
	}
	return new(uint256.Int).SetBytes(output[:32]), common.BytesToAddress(output[44:]), nil
}
