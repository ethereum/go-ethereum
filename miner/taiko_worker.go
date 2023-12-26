package miner

import (
	"errors"
	"fmt"
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
)

// BuildTransactionsLists builds multiple transactions lists which satisfy all the given limits.
func (w *worker) BuildTransactionsLists(
	beneficiary common.Address,
	baseFee *big.Int,
	maxTransactionsPerBlock uint64,
	blockMaxGasLimit uint64,
	maxBytesPerTxList uint64,
	localAccounts []string,
	maxTransactionsLists uint64,
) ([]types.Transactions, error) {
	var (
		txsLists    []types.Transactions
		currentHead = w.chain.CurrentBlock()
	)

	if currentHead == nil {
		return nil, fmt.Errorf("failed to find current head")
	}

	// Check if tx pool is empty at first.
	if len(w.eth.TxPool().Pending(false)) == 0 {
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

	env, err := w.prepareWork(params)
	if err != nil {
		return nil, err
	}
	defer env.discard()

	var (
		signer = types.MakeSigner(w.chainConfig, new(big.Int).Add(currentHead.Number, common.Big1), currentHead.Time)
	)

	commitTxs := func() (types.Transactions, error) {
		env.tcount = 0
		env.txs = []*types.Transaction{}
		env.gasPool = new(core.GasPool).AddGas(blockMaxGasLimit)
		env.header.GasLimit = blockMaxGasLimit

		// Split the pending transactions into locals and remotes, then
		// fill the block with all available pending transactions.
		localTxs, remoteTxs := w.getPendingTxs(localAccounts)

		w.commitL2Transactions(
			env,
			newTransactionsByPriceAndNonce(signer, localTxs, baseFee),
			newTransactionsByPriceAndNonce(signer, remoteTxs, baseFee),
			maxTransactionsPerBlock,
			maxBytesPerTxList,
		)

		return env.txs, nil
	}

	for i := 0; i < int(maxTransactionsLists); i++ {
		txs, err := commitTxs()
		if err != nil {
			return nil, err
		}

		if len(txs) == 0 {
			break
		}

		txsLists = append(txsLists, txs)
	}

	return txsLists, nil
}

// sealBlockWith mines and seals a block with the given block metadata.
func (w *worker) sealBlockWith(
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
		// A L2 block needs to have have at least one `V1TaikoL2.anchor` or
		// `V1TaikoL2.invalidateBlock` transaction.
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
	w.extra = blkMeta.ExtraData

	env, err := w.prepareWork(params)
	if err != nil {
		return nil, err
	}
	defer env.discard()

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
		sender, err := types.LatestSignerForChainID(tx.ChainId()).Sender(tx)
		if err != nil {
			log.Info("Skip an invalid proposed transaction", "hash", tx.Hash(), "reason", err)
			continue
		}

		env.state.Prepare(rules, sender, blkMeta.Beneficiary, tx.To(), vm.ActivePrecompiles(rules), tx.AccessList())
		env.state.SetTxContext(tx.Hash(), env.tcount)
		if _, err := w.commitTransaction(env, tx); err != nil {
			log.Info("Skip an invalid proposed transaction", "hash", tx.Hash(), "reason", err)
			continue
		}
		env.tcount++
	}

	block, err := w.engine.FinalizeAndAssemble(w.chain, env.header, env.state, env.txs, nil, env.receipts, withdrawals)
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
func (w *worker) getPendingTxs(localAccounts []string) (
	map[common.Address][]*txpool.LazyTransaction,
	map[common.Address][]*txpool.LazyTransaction,
) {
	pending := w.eth.TxPool().Pending(false)
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
func (w *worker) commitL2Transactions(
	env *environment,
	txsLocal *transactionsByPriceAndNonce,
	txsRemote *transactionsByPriceAndNonce,
	maxTransactionsPerBlock uint64,
	maxBytesPerTxList uint64,
) {
	var (
		txs            = txsLocal
		isLocal        = true
		accTxListBytes int
	)

loop:
	for {
		// If we don't have enough gas for any further transactions then we're done.
		if env.gasPool.Gas() < params.TxGas {
			log.Trace("Not enough gas for further transactions", "have", env.gasPool, "want", params.TxGas)
			break
		}

		// Retrieve the next transaction and abort if all done.
		ltx := txs.Peek()
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
			log.Warn("Ignoring evicted transaction")

			txs.Pop()
			continue
		}
		// Error may be ignored here. The error has already been checked
		// during transaction acceptance is the transaction pool.
		from, _ := types.Sender(env.signer, tx)

		b, err := rlp.EncodeToBytes(tx)
		if err != nil {
			log.Trace("Failed to rlp encode the pending transaction %s: %w", tx.Hash(), err)
			txs.Pop()
			continue
		}
		if accTxListBytes+len(b) >= int(maxBytesPerTxList) {
			break
		}

		// Check whether the tx is replay protected. If we're not in the EIP155 hf
		// phase, start ignoring the sender until we do.
		if tx.Protected() && !w.chainConfig.IsEIP155(env.header.Number) {
			log.Trace("Ignoring reply protected transaction", "hash", tx.Hash(), "eip155", w.chainConfig.EIP155Block)

			txs.Pop()
			continue
		}
		// Start executing the transaction
		env.state.SetTxContext(tx.Hash(), env.tcount)

		_, err = w.commitTransaction(env, tx)
		switch {
		case errors.Is(err, core.ErrGasLimitReached):
			// Pop the current out-of-gas transaction without shifting in the next from the account
			log.Trace("Gas limit exceeded for current block", "sender", from)
			txs.Pop()

		case errors.Is(err, core.ErrNonceTooLow):
			// New head notification data race between the transaction pool and miner, shift
			log.Trace("Skipping transaction with low nonce", "sender", from, "nonce", tx.Nonce())
			txs.Shift()

		case errors.Is(err, core.ErrNonceTooHigh):
			// Reorg notification data race between the transaction pool and miner, skip account =
			log.Trace("Skipping account with hight nonce", "sender", from, "nonce", tx.Nonce())
			txs.Pop()

		case errors.Is(err, nil):
			// Everything ok, shift in the next transaction from the same account
			env.tcount++
			txs.Shift()

			if env.tcount >= int(maxTransactionsPerBlock) {
				break loop
			}
			accTxListBytes += len(b)

		case errors.Is(err, types.ErrTxTypeNotSupported):
			// Pop the unsupported transaction without shifting in the next from the account
			log.Trace("Skipping unsupported transaction type", "sender", from, "type", tx.Type())
			txs.Pop()

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Debug("Transaction failed, account skipped", "hash", tx.Hash(), "err", err)
			txs.Shift()
		}
	}
}
