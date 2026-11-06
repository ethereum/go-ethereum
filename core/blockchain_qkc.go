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
	"math/big"

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

// GetCoinbaseAmount derives the miner's after-tax reward and fees from an imported shard
// block and its receipts. Hash-based reads also support side chains and retries;
// no state is needed, so pruning the parent state does not affect reporting.
// Ordinary transfers, contract transfers and deposit values are excluded, even
// when their recipient is the block's coinbase.
func (bc *BlockChain) GetCoinbaseAmount(shardBlockHash common.Hash) (*uint256.Int, error) {
	if bc.chainConfig.QuarkChain == nil {
		return nil, errors.New("coinbase amount is only available on a shard chain")
	}
	block := bc.GetBlockByHash(shardBlockHash)
	if block == nil {
		return nil, fmt.Errorf("shard block %s not found", shardBlockHash.Hex())
	}
	receipts := bc.GetReceiptsByHash(shardBlockHash)
	if receipts == nil || len(receipts) != len(block.Transactions()) {
		return nil, fmt.Errorf("missing or incomplete receipts for shard block %s", shardBlockHash.Hex())
	}
	amount := bc.chainConfig.QuarkChain.PoSW.BlockRewardAt(block.Number())
	amount = bc.chainConfig.QuarkChain.AfterTax(amount)
	for i, tx := range block.Transactions() {
		tip, err := tx.EffectiveGasTip(block.BaseFee())
		if err != nil {
			return nil, fmt.Errorf("shard block %s transaction %d fee: %w", shardBlockHash.Hex(), i, err)
		}
		amount.Add(amount, CalcTxFeeAfterTax(bc.chainConfig, receipts[i].GasUsed, tip))
	}
	result, overflow := uint256.FromBig(amount)
	if overflow {
		return nil, fmt.Errorf("coinbase amount for shard block %s exceeds uint256", shardBlockHash.Hex())
	}
	return result, nil
}

// CalcTxFeeAfterTax computes one transaction's miner fee after shard tax and rounding.
// gasUsed is net of refunds. effectiveTip excludes the base fee for ordinary
// transactions and is the full gas price for deposits.
func CalcTxFeeAfterTax(config *params.ChainConfig, gasUsed uint64, effectiveTip *big.Int) *big.Int {
	fee := new(big.Int).Mul(new(big.Int).SetUint64(gasUsed), effectiveTip)
	if config.QuarkChain != nil {
		return config.QuarkChain.AfterTax(fee)
	}
	return fee
}

// GetShardLockedStakes executes the shard staking getter at header's state.
// Shard consensus supplies the candidate's parent header. Legacy
// shard PoSW used spendable account balances; explicit contract locking replaces
// that rule here. getLockedStakes returns zero as soon as an account is unlocked.
func (bc *BlockChain) GetShardLockedStakes(header *types.Header, staker common.Address) (*uint256.Int, error) {
	stakes, _, err := bc.getLockedStakes(header, params.ShardStakingContractAddress, staker)
	return stakes, err
}

// GetRootChainStakes reads root-chain stakes and the mining signer from the root coinbase chain.
// The caller selects the shard block confirmed by the candidate root's parent;
// header is that shard block's header, not a root-block header.
func (bc *BlockChain) GetRootChainStakes(header *types.Header, staker common.Address) (*uint256.Int, common.Address, error) {
	if !bc.chainConfig.IsRootCoinbaseChain() {
		return nil, common.Address{}, errors.New("root-chain stakes are only available on the root coinbase chain")
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
