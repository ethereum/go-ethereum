package gasprice

import (
	"context"
	"math/big"
	"sort"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
)

func (oracle *Oracle) calculateSuggestPriorityFee(ctx context.Context, header *types.Header) (*big.Int, bool) {
	headHash := header.Hash()
	// If the latest gasprice is still available, return it.
	oracle.cacheLock.RLock()
	lastHead, lastPrice, lastIsCongested := oracle.lastHead, oracle.lastPrice, oracle.lastIsCongested
	oracle.cacheLock.RUnlock()
	if headHash == lastHead {
		return new(big.Int).Set(lastPrice), lastIsCongested
	}
	oracle.fetchLock.Lock()
	defer oracle.fetchLock.Unlock()

	// Try checking the cache again, maybe the last fetch fetched what we need
	oracle.cacheLock.RLock()
	lastHead, lastPrice, lastIsCongested = oracle.lastHead, oracle.lastPrice, oracle.lastIsCongested
	oracle.cacheLock.RUnlock()
	if headHash == lastHead {
		return new(big.Int).Set(lastPrice), lastIsCongested
	}

	var isCongested bool
	// Before Curie (EIP-1559), we need to return the total suggested gas price. After Curie we return defaultGasTipCap wei as the tip cap,
	// as the base fee is set separately or added manually for legacy transactions.
	suggestion := oracle.defaultGasTipCap
	if !oracle.backend.ChainConfig().IsCurie(header.Number) {
		suggestion = oracle.defaultBasePrice
	}

	// find the maximum gas used by any of the transactions in the block to use as the gas limit
	// capacity margin
	receipts, err := oracle.backend.GetReceipts(ctx, header.Hash())
	if receipts == nil || err != nil {
		log.Debug("failed to get block receipts during calculating suggest priority fee", "block number", header.Number, "err", err)
		// If the lastIsCongested is true on the cache, return the lastPrice.
		// We believe it's better to err on the side of returning a higher-than-needed suggestion than a lower-than-needed one.
		if lastIsCongested {
			return lastPrice, lastIsCongested
		}
		return suggestion, isCongested
	}
	var maxTxGasUsed uint64

	for i := range receipts {
		gu := receipts[i].GasUsed
		if gu > maxTxGasUsed {
			maxTxGasUsed = gu
		}
	}

	// find the maximum transaction size by any of the transactions in the block to use as the block
	// size limit capacity margin
	var (
		maxTxSizeUsed   common.StorageSize
		totalTxSizeUsed common.StorageSize
	)
	block, err := oracle.backend.BlockByNumber(ctx, rpc.BlockNumber(header.Number.Int64()))
	if block == nil || err != nil {
		log.Error("failed to get last block", "err", err)
		return suggestion, isCongested
	}
	txs := block.Transactions()

	for i := range txs {
		su := txs[i].Size()
		if su > maxTxSizeUsed {
			maxTxSizeUsed = su
		}
		totalTxSizeUsed = totalTxSizeUsed + su
	}

	// sanity check the max gas used and transaction size value
	if maxTxGasUsed > header.GasLimit {
		log.Error("found tx consuming more gas than the block limit", "gas", maxTxGasUsed)
		return suggestion, isCongested
	}
	if !oracle.backend.ChainConfig().Scroll.IsValidBlockSize(maxTxSizeUsed) {
		log.Error("found tx consuming more size than the block size limit", "size", maxTxSizeUsed)
		return suggestion, isCongested
	}

	if header.GasUsed+maxTxGasUsed > header.GasLimit ||
		!oracle.backend.ChainConfig().Scroll.IsValidBlockSizeForMining(totalTxSizeUsed+maxTxSizeUsed) {
		// There are two cases that represent a block is "at capacity":
		//   1. When building the block, there is a pending transaction in the txpool that could not be
		//      included because adding it would exceed the block's gas limit.
		//   2. Or, there is a pending transaction that could not be included because adding it would
		//      exceed the block's transaction payload size (block size limit).
		//
		// Since we don't have access to the txpool, we instead adopt the following heuristic:
		// consider a block as at capacity if either:
		//   - the total gas consumed by its transactions is within max-tx-gas-used of the block gas
		//     limit, where max-tx-gas-used is the most gas used by any one transaction within the block, or
		//   - the total transaction payload size is within max-tx-size-used of the block size limit,
		//     where max-tx-size-used is the largest transaction size in the block.
		//
		// This heuristic is almost perfectly accurate when transactions always consume the same amount
		// of gas and have similar sizes, but becomes less accurate as gas usage or payload size varies
		// between transactions. The typical error is that we assume a block is at capacity when it was
		// not, because max-tx-gas-used or max-tx-size-used will in most cases over-estimate the
		// "capacity margin". But it's better to err on the side of returning a higher-than-needed
		// suggestion than a lower-than-needed one, in order to satisfy our desire for high chance of
		// inclusion and rising fees under high demand.
		baseFee := block.BaseFee()
		if len(txs) == 0 {
			log.Error("block was at capacity but doesn't have transactions")
			return suggestion, isCongested
		}
		tips := bigIntArray(make([]*big.Int, len(txs)))
		for i := range txs {
			tips[i] = txs[i].EffectiveGasTipValue(baseFee)
		}
		sort.Sort(tips)
		median := tips[len(tips)/2]
		newSuggestion := new(big.Int).Add(median, new(big.Int).Div(median, big.NewInt(10)))
		isCongested = true
		// use the new suggestion only if it's bigger than the minimum
		if newSuggestion.Cmp(suggestion) > 0 {
			suggestion = newSuggestion
		}
	}

	// the suggestion should be capped by oracle.maxPrice
	if suggestion.Cmp(oracle.maxPrice) > 0 {
		suggestion.Set(oracle.maxPrice)
	}

	// update the cache only if it's latest block header
	latestHeader, _ := oracle.backend.HeaderByNumber(ctx, rpc.LatestBlockNumber)
	if header.Hash() == latestHeader.Hash() {
		oracle.cacheLock.Lock()
		oracle.lastHead = header.Hash()
		oracle.lastPrice = suggestion
		oracle.lastIsCongested = isCongested
		oracle.cacheLock.Unlock()
	}

	return suggestion, isCongested
}

// SuggestScrollPriorityFee returns a max priority fee value that can be used such that newly
// created transactions have a very high chance to be included in the following blocks, using a
// simplified and more predictable algorithm appropriate for chains like Scroll with a single
// known block builder.
//
// In the typical case, which results whenever the last block had room for more transactions, this
// function returns a minimum suggested priority fee value. Otherwise it returns the higher of this
// minimum suggestion or 10% over the median effective priority fee from the last block.
//
// Rationale: For a chain such as Scroll where there is a single block builder whose behavior is
// known, we know priority fee (as long as it is non-zero) has no impact on the probability for tx
// inclusion as long as there is capacity for it in the block. In this case then, there's no reason
// to return any value higher than some fixed minimum. Blocks typically reach capacity only under
// extreme events such as airdrops, meaning predicting whether the next block is going to be at
// capacity is difficult *except* in the case where we're already experiencing the increased demand
// from such an event. We therefore expect whether the last known block is at capacity to be one of
// the best predictors of whether the next block is likely to be at capacity. (An even better
// predictor is to look at the state of the transaction pool, but we want an algorithm that works
// even if the txpool is private or unavailable.)
//
// In the event the next block may be at capacity, the algorithm should allow for average fees to
// rise in order to reach a market price that appropriately reflects demand. We accomplish this by
// returning a suggestion that is a significant amount (10%) higher than the median effective
// priority fee from the previous block.
func (oracle *Oracle) SuggestScrollPriorityFee(ctx context.Context, header *types.Header) *big.Int {
	suggestion, _ := oracle.calculateSuggestPriorityFee(ctx, header)

	return new(big.Int).Set(suggestion)
}
