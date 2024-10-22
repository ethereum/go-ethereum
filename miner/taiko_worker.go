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

	if currentHead == nil {
		return nil, fmt.Errorf("failed to find current head")
	}

	// Check if tx pool is empty at first.
	if len(w.txpool.Pending(txpool.PendingFilter{MinTip: uint256.NewInt(minTip), BaseFee: uint256.MustFromBig(baseFee), OnlyPlainTxs: true})) == 0 {
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

	commitTxs := func(firstTransaction *types.Transaction) (*types.Transaction, *PreBuiltTxList, error) {
		env.tcount = 0
		env.txs = []*types.Transaction{}
		env.gasPool = new(core.GasPool).AddGas(blockMaxGasLimit)
		env.header.GasLimit = blockMaxGasLimit

		lastTransaction := w.commitL2Transactions(
			env,
			firstTransaction,
			newTransactionsByPriceAndNonce(signer, maps.Clone(localTxs), baseFee),
			newTransactionsByPriceAndNonce(signer, maps.Clone(remoteTxs), baseFee),
			maxBytesPerTxList,
			minTip,
		)

		b, err := encodeAndCompressTxList(env.txs)
		if err != nil {
			return nil, nil, err
		}

		return lastTransaction, &PreBuiltTxList{
			TxList:           env.txs,
			EstimatedGasUsed: env.header.GasLimit - env.gasPool.Gas(),
			BytesLength:      uint64(len(b)),
		}, nil
	}

	var (
		lastTx *types.Transaction
		res    *PreBuiltTxList
	)
	for i := 0; i < int(maxTransactionsLists); i++ {
		if lastTx, res, err = commitTxs(lastTx); err != nil {
			return nil, err
		}

		if len(res.TxList) == 0 {
			break
		}

		txsLists = append(txsLists, res)
	}

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
		// A L2 block needs to have have at least one `TaikoL2.anchor` / `TaikoL2.anchorV2`.
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
	firstTransaction *types.Transaction,
	txsLocal *transactionsByPriceAndNonce,
	txsRemote *transactionsByPriceAndNonce,
	maxBytesPerTxList uint64,
	minTip uint64,
) *types.Transaction {
	var (
		txs             = txsLocal
		isLocal         = true
		lastTransaction *types.Transaction
	)

	if firstTransaction != nil {
		env.txs = append(env.txs, firstTransaction)
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

			data, err := rlp.EncodeToBytes(env.txs)
			if err != nil {
				log.Trace("Failed to rlp encode the pending transaction %s: %w", tx.Hash(), err)
				txs.Pop()
				continue
			}
			if len(data) >= int(maxBytesPerTxList) {
				// Encode and compress the txList, if the byte length is > maxBytesPerTxList, remove the latest tx and break.
				b, err := compress(data)
				if err != nil {
					log.Trace("Failed to rlp encode and compress the pending transaction %s: %w", tx.Hash(), err)
					txs.Pop()
					continue
				}
				if len(b) > int(maxBytesPerTxList) {
					lastTransaction = env.txs[env.tcount-1]
					env.txs = env.txs[0 : env.tcount-1]
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

	return lastTransaction
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

	if err := w.Flush(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
