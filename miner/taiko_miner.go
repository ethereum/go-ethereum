package miner

import (
	"math/big"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// SealBlockWith mines and seals a block without changing the canonical chain.
func (miner *Miner) SealBlockWith(
	parent common.Hash,
	timestamp uint64,
	blkMeta *engine.BlockMetadata,
	baseFeePerGas *big.Int,
	withdrawals types.Withdrawals,
) (*types.Block, error) {
	return miner.worker.sealBlockWith(parent, timestamp, blkMeta, baseFeePerGas, withdrawals)
}

// BuildTransactionsLists builds multiple transactions lists which satisfy all the given limits.
func (miner *Miner) BuildTransactionsLists(
	beneficiary common.Address,
	baseFee *big.Int,
	maxTransactionsPerBlock uint64,
	blockMaxGasLimit uint64,
	maxBytesPerTxList uint64,
	locals []string,
	maxTransactionsLists uint64,
) ([]types.Transactions, error) {
	return miner.worker.BuildTransactionsLists(
		beneficiary,
		baseFee,
		maxTransactionsPerBlock,
		blockMaxGasLimit,
		maxBytesPerTxList,
		locals,
		maxTransactionsLists,
	)
}
