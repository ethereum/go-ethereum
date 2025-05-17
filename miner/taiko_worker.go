package miner

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"maps"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/holiman/uint256"
)

const (
	TxListCompressionCheckInterval = 100
	TxListCompressionPruneStep     = 10
)

// BuildTransactionsLists builds multiple transactions lists which satisfy all the given conditions
// 1. All transactions should all be able to pay the given base fee.
// 2. The total gas used should not exceed the given blockMaxGasLimit
// 3. The total bytes used should not exceed the given maxBytesPerTxList
// 4. The total number of transactions lists should not exceed the given maxTransactionsLists
func (w *Miner) buildTransactionsLists(
	beneficiary common.Address,
	baseFee *big.Int,
	blockMaxGasLimit uint64,
	maxBytesPerTxList uint64,
	localAccounts []string,
	maxTransactionsLists uint64,
	minTip uint64,
) ([]*PreBuiltTxList, error) {
	var (
		txsLists    []*PreBuiltTxList
		currentHead = w.chain.CurrentBlock()
	)

	log.Info("buildTransactionsLists",
		"blockMaxGasLimit", blockMaxGasLimit,
		"maxBytesPerTxList", maxBytesPerTxList,
		"maxTransactionsLists", maxTransactionsLists,
		"localAccounts", localAccounts,
		"minTip", minTip,
	)

	if currentHead == nil {
		log.Error("buildTransactionsLists failed to find current head")
		return nil, fmt.Errorf("failed to find current head")
	}

	// Check if tx pool is empty at first.
	if len(w.txpool.Pending(
		txpool.PendingFilter{
			MinTip:       uint256.NewInt(minTip),
			BaseFee:      uint256.MustFromBig(baseFee),
			OnlyPlainTxs: true,
		},
	)) == 0 {
		log.Warn("buildTransactionsLists: tx pool is empty",
			"minTip", minTip,
			"onlyPlainTxs", true,
		)
		return txsLists, nil
	}

	params := &generateParams{
		timestamp:     uint64(time.Now().Unix()),
		forceTime:     true,
		parentHash:    currentHead.Hash(),
		coinbase:      beneficiary,
		random:        currentHead.MixDigest,
		noTxs:         false,
		baseFeePerGas: baseFee,
	}

	log.Info("buildTransactionsLists: prepare work",
		"timestamp", params.timestamp,
		"forceTime", params.forceTime,
		"parentHash", params.parentHash,
		"coinbase", params.coinbase,
		"random", params.random,
		"noTxs", params.noTxs,
	)

	env, err := w.prepareWork(params, false)
	if err != nil {
		return nil, err
	}

	var (
		signer = types.MakeSigner(w.chainConfig, new(big.Int).Add(currentHead.Number, common.Big1), currentHead.Time)
		// Split the pending transactions into locals and remotes, then
		// fill the block with all available pending transactions.
		localTxs, remoteTxs = w.getPendingTxs(localAccounts, baseFee)
	)

	commitTxs := func(pruningResult *txsPruningResult) (*txsPruningResult, *PreBuiltTxList, error) {
		env.tcount = 0
		env.txs = []*types.Transaction{}
		env.gasPool = new(core.GasPool).AddGas(blockMaxGasLimit - accumulateGasUsed(pruningResult.ReceiptsPruned))
		env.header.GasLimit = blockMaxGasLimit

		result, err := w.commitL2Transactions(
			env,
			pruningResult.TxsPruned,
			pruningResult.ReceiptsPruned,
			newTransactionsByPriceAndNonce(signer, maps.Clone(localTxs), baseFee),
			newTransactionsByPriceAndNonce(signer, maps.Clone(remoteTxs), baseFee),
			maxBytesPerTxList,
			minTip,
		)
		if err != nil {
			log.Error("buildTransactionsLists: commit transactions failed", "error", err)

			return nil, nil, err
		}

		log.Info("buildTransactionsLists: commit transactions",
			"localTxs", len(localTxs),
			"remoteTxs", len(remoteTxs),
			"txsPruned", len(pruningResult.TxsPruned),
			"receiptsPruned", len(pruningResult.ReceiptsPruned),
			"txsRemaining", len(result.TxsRemaining),
			"receiptsRemaining", len(result.ReceiptsRemaining),
			"size", result.Size,
			"gasUsed", accumulateGasUsed(result.ReceiptsRemaining),
			"bytesLength", uint64(result.Size),
		)

		return result, &PreBuiltTxList{
			TxList:           result.TxsRemaining,
			EstimatedGasUsed: accumulateGasUsed(result.ReceiptsRemaining),
			BytesLength:      uint64(result.Size),
		}, nil
	}

	var (
		pruningResult  = new(txsPruningResult)
		preBuiltTxList *PreBuiltTxList
	)
	for range int(maxTransactionsLists) {
		if pruningResult, preBuiltTxList, err = commitTxs(pruningResult); err != nil {
			return nil, err
		}

		if len(preBuiltTxList.TxList) == 0 {
			break
		}

		txsLists = append(txsLists, preBuiltTxList)
	}

	log.Info("buildTransactionsLists: commit transactions finished",
		"txLists", len(txsLists),
	)

	return txsLists, nil
}

// sealBlockWith mines and seals a block with the given block metadata.
func (w *Miner) sealBlockWith(
	parent common.Hash,
	timestamp uint64,
	blkMeta *engine.BlockMetadata,
	baseFeePerGas *big.Int,
	withdrawals types.Withdrawals,
) (*types.Block, error) {
	// Decode transactions bytes.
	var txs types.Transactions
	if err := rlp.DecodeBytes(blkMeta.TxList, &txs); err != nil {
		return nil, fmt.Errorf("failed to decode txList: %w", err)
	}

	if len(txs) == 0 {
		// A L2 block needs to have have at least one `TaikoL2.anchor` / `TaikoL2.anchorV2` / `TaikoL2.anchorV3`.
		return nil, fmt.Errorf("too less transactions in the block")
	}

	params := &generateParams{
		timestamp:     timestamp,
		forceTime:     true,
		parentHash:    parent,
		coinbase:      blkMeta.Beneficiary,
		random:        blkMeta.MixHash,
		withdrawals:   withdrawals,
		noTxs:         false,
		baseFeePerGas: baseFeePerGas,
	}

	// Set extraData
	w.SetExtra(blkMeta.ExtraData)

	env, err := w.prepareWork(params, false)
	if err != nil {
		return nil, err
	}

	env.header.GasLimit = blkMeta.GasLimit

	// Commit transactions.
	gasLimit := env.header.GasLimit
	rules := w.chain.Config().Rules(env.header.Number, true, timestamp)

	env.gasPool = new(core.GasPool).AddGas(gasLimit)

	for i, tx := range txs {
		if i == 0 {
			if err := tx.MarkAsAnchor(); err != nil {
				return nil, err
			}
		}
		// Skip blob transactions
		if tx.Type() == types.BlobTxType {
			log.Debug("Skip a blob transaction", "hash", tx.Hash())
			continue
		}
		sender, err := types.LatestSignerForChainID(w.chainConfig.ChainID).Sender(tx)
		if err != nil {
			log.Debug("Skip an invalid proposed transaction", "hash", tx.Hash(), "reason", err)
			continue
		}

		env.state.Prepare(rules, sender, blkMeta.Beneficiary, tx.To(), vm.ActivePrecompiles(rules), tx.AccessList())
		env.state.SetTxContext(tx.Hash(), env.tcount)
		if err := w.commitTransaction(env, tx); err != nil {
			log.Debug("Skip an invalid proposed transaction", "hash", tx.Hash(), "reason", err)
			continue
		}
		env.tcount++
	}

	block, err := w.engine.FinalizeAndAssemble(
		w.chain,
		env.header,
		env.state,
		&types.Body{Transactions: env.txs, Withdrawals: withdrawals},
		env.receipts,
	)
	if err != nil {
		return nil, err
	}

	results := make(chan *types.Block, 1)
	if err := w.engine.Seal(w.chain, block, results, nil); err != nil {
		return nil, err
	}
	block = <-results

	return block, nil
}

// getPendingTxs fetches the pending transactions from tx pool.
func (w *Miner) getPendingTxs(localAccounts []string, baseFee *big.Int) (
	map[common.Address][]*txpool.LazyTransaction,
	map[common.Address][]*txpool.LazyTransaction,
) {
	pending := w.txpool.Pending(txpool.PendingFilter{OnlyPlainTxs: true, BaseFee: uint256.MustFromBig(baseFee)})
	localTxs, remoteTxs := make(map[common.Address][]*txpool.LazyTransaction), pending

	for _, local := range localAccounts {
		account := common.HexToAddress(local)
		if txs := remoteTxs[account]; len(txs) > 0 {
			delete(remoteTxs, account)
			localTxs[account] = txs
		}
	}

	return localTxs, remoteTxs
}

// commitL2Transactions tries to commit the transactions into the given state.
func (w *Miner) commitL2Transactions(
	env *environment,
	presetTxs []*types.Transaction,
	presetReceipts []*types.Receipt,
	txsLocal *transactionsByPriceAndNonce,
	txsRemote *transactionsByPriceAndNonce,
	maxBytesPerTxList uint64,
	minTip uint64,
) (*txsPruningResult, error) {
	var (
		txs           = txsLocal
		isLocal       = true
		pruningResult *txsPruningResult
		err           error
	)

	if presetTxs != nil {
		env.tcount = len(presetTxs)
		env.txs = append(env.txs, presetTxs...)
		env.receipts = append(env.receipts, presetReceipts...)
	}

loop:
	for {
		// If we don't have enough gas for any further transactions then we're done.
		if env.gasPool.Gas() < params.TxGas {
			log.Trace("Not enough gas for further transactions", "have", env.gasPool, "want", params.TxGas)
			break
		}

		// Retrieve the next transaction and abort if all done.
		ltx, _ := txs.Peek()
		if ltx == nil {
			if isLocal {
				txs = txsRemote
				isLocal = false
				continue
			}
			break
		}
		tx := ltx.Resolve()
		if tx == nil {
			log.Trace("Ignoring evicted transaction")

			txs.Pop()
			continue
		}

		if tx.GasTipCapIntCmp(new(big.Int).SetUint64(minTip)) < 0 {
			log.Trace("Ignoring transaction with low tip", "hash", tx.Hash(), "tip", tx.GasTipCap(), "minTip", minTip)
			txs.Pop()
			continue
		}

		// Error may be ignored here. The error has already been checked
		// during transaction acceptance is the transaction pool.
		from, _ := types.Sender(env.signer, tx)

		// Check whether the tx is replay protected. If we're not in the EIP155 hf
		// phase, start ignoring the sender until we do.
		if tx.Protected() && !w.chainConfig.IsEIP155(env.header.Number) {
			log.Trace("Ignoring reply protected transaction", "hash", tx.Hash(), "eip155", w.chainConfig.EIP155Block)

			txs.Pop()
			continue
		}
		// Start executing the transaction
		env.state.SetTxContext(tx.Hash(), env.tcount)

		err := w.commitTransaction(env, tx)
		switch {
		case errors.Is(err, core.ErrNonceTooLow):
			// New head notification data race between the transaction pool and miner, shift
			log.Trace("Skipping transaction with low nonce", "hash", ltx.Hash, "sender", from, "nonce", tx.Nonce())
			txs.Shift()

		case errors.Is(err, nil):
			// Everything ok, collect the logs and shift in the next transaction from the same account
			txs.Shift()

			// Check the size of the compressed txList, if it exceeds the maxBytesPerTxList, break the loop.
			if env.tcount%TxListCompressionCheckInterval == 0 {
				if pruningResult, err = pruneTransactions(env.txs, env.receipts, maxBytesPerTxList); err != nil {
					return nil, err
				}
				// If there are pruned transactions, break the loop.
				if len(pruningResult.TxsPruned) > 0 {
					break loop
				}
			}

		default:
			// Transaction is regarded as invalid, drop all consecutive transactions from
			// the same sender because of `nonce-too-high` clause.
			log.Trace("Transaction failed, account skipped", "hash", ltx.Hash, "err", err)
			txs.Pop()
		}
	}

	if pruningResult, err = pruneTransactions(env.txs, env.receipts, maxBytesPerTxList); err != nil {
		return nil, err
	}

	return pruningResult, nil
}

// encodeAndCompressTxList encodes and compresses the given transactions list.
func encodeAndCompressTxList(txs types.Transactions) ([]byte, error) {
	b, err := rlp.EncodeToBytes(txs)
	if err != nil {
		return nil, err
	}

	return compress(b)
}

// compress compresses the given txList bytes using zlib.
func compress(txListBytes []byte) ([]byte, error) {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	defer w.Close()

	if _, err := w.Write(txListBytes); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// txsPruningResult represents the result of a transactions list pruning.
type txsPruningResult struct {
	TxsPruned         []*types.Transaction
	ReceiptsPruned    []*types.Receipt
	TxsRemaining      []*types.Transaction
	ReceiptsRemaining []*types.Receipt
	Size              int
}

// pruneTransactions prunes the transactions from the given environment to fit the size limit.
func pruneTransactions(
	txs []*types.Transaction,
	receipts []*types.Receipt,
	sizeLimit uint64,
) (*txsPruningResult, error) {
	var (
		prunedTxs      []*types.Transaction
		prunedReceipts []*types.Receipt
		step           = TxListCompressionPruneStep
	)

	for len(txs) > 0 {
		if len(txs) <= step {
			step = 1
		}

		b, err := encodeAndCompressTxList(txs)
		if err != nil {
			return nil, err
		}
		if len(b) <= int(sizeLimit) {
			return &txsPruningResult{
				TxsPruned:      prunedTxs,
				ReceiptsPruned: prunedReceipts,
				TxsRemaining:   txs,
				Size:           len(b),
			}, nil
		}

		prunedTxs = append(txs[len(txs)-step:], prunedTxs...)
		prunedReceipts = append(receipts[len(receipts)-step:], prunedReceipts...)
		txs = txs[:len(txs)-step]
		receipts = receipts[:len(receipts)-step]
	}

	// All transactions are pruned.
	return &txsPruningResult{
		TxsPruned:         prunedTxs,
		ReceiptsPruned:    prunedReceipts,
		TxsRemaining:      txs,
		ReceiptsRemaining: receipts,
		Size:              0,
	}, nil
}

// accumulateGasUsed sums up the gas used from the receipts.
func accumulateGasUsed(receipts []*types.Receipt) uint64 {
	var gasUsed uint64
	for _, receipt := range receipts {
		gasUsed += receipt.GasUsed
	}
	return gasUsed
}
