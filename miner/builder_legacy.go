package miner

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// SUAVE

func (miner *Miner) rawCommitTransactions(env *environment, txs types.Transactions) error {
	gasLimit := env.header.GasLimit
	if env.gasPool == nil {
		env.gasPool = new(core.GasPool).AddGas(gasLimit)
	}

	// TODO: logs should be a part of the env and returned to whoever requested the block
	// var coalescedLogs []*types.Log

	for _, tx := range txs {
		// If we don't have enough gas for any further transactions then we're done.
		if env.gasPool.Gas() < params.TxGas {
			log.Trace("Not enough gas for further transactions", "have", env.gasPool, "want", params.TxGas)
			break
		}
		// Error may be ignored here. The error has already been checked
		// during transaction acceptance is the transaction pool.
		from, _ := types.Sender(env.signer, tx)

		// Check whether the tx is replay protected. If we're not in the EIP155 hf
		// phase, start ignoring the sender until we do.
		if tx.Protected() && !miner.chainConfig.IsEIP155(env.header.Number) {
			log.Trace("Ignoring reply protected transaction", "hash", tx.Hash(), "eip155", miner.chainConfig.EIP155Block)

			return fmt.Errorf("invalid reply protected tx %s", tx.Hash())
		}

		// Start executing the transaction
		env.state.SetTxContext(tx.Hash(), env.tcount)

		// logs, err := w.commitTransaction(env, tx)
		err := miner.commitTransaction(env, tx)

		switch {
		case errors.Is(err, core.ErrNonceTooLow):
			log.Debug("Skipping transaction with low nonce", "hash", tx.Hash(), "sender", from, "nonce", tx.Nonce())
			return err
		case errors.Is(err, nil):
			// coalescedLogs = append(coalescedLogs, logs...)
			env.tcount++
		default:
			// Transaction is regarded as invalid, drop all consecutive transactions from
			// the same sender because of `nonce-too-high` clause.
			log.Debug("Transaction failed, account skipped", "hash", tx.Hash(), "err", err)
			return err
		}
	}
	return nil
}

func (miner *Miner) commitPendingTxs(work *environment) error {
	if err := miner.fillTransactions(nil, work); err != nil {
		return err
	}
	return nil
}

func (miner *Miner) buildBlockFromTxs(ctx context.Context, args *types.BuildBlockArgs, txs types.Transactions) (*types.Block, *big.Int, error) {
	params := &generateParams{
		timestamp:   args.Timestamp,
		forceTime:   true,
		parentHash:  args.Parent,
		coinbase:    args.FeeRecipient,
		random:      args.Random,
		extra:       args.Extra,
		withdrawals: args.Withdrawals,
		beaconRoot:  &args.BeaconRoot,
		// noUncle:     true,
		noTxs: false,
	}

	work, err := miner.prepareWork(params)
	if err != nil {
		return nil, nil, err
	}

	profitPre := work.state.GetBalance(args.FeeRecipient)

	if err := miner.rawCommitTransactions(work, txs); err != nil {
		return nil, nil, err
	}
	if args.FillPending {
		if err := miner.commitPendingTxs(work); err != nil {
			return nil, nil, err
		}
	}

	profitPost := work.state.GetBalance(args.FeeRecipient)
	// TODO : Is it okay to set Uncle List to nil?
	body := types.Body{Transactions: work.txs, Withdrawals: params.withdrawals}
	block, err := miner.engine.FinalizeAndAssemble(miner.chain, work.header, work.state, &body, work.receipts)
	if err != nil {
		return nil, nil, err
	}
	blockProfit := new(big.Int).Sub(profitPost.ToBig(), profitPre.ToBig())
	return block, blockProfit, nil
}

func (miner *Miner) buildBlockFromBundles(ctx context.Context, args *types.BuildBlockArgs, bundles []types.SBundle) (*types.Block, *big.Int, error) {
	// create ephemeral addr and private key for payment txn
	ephemeralPrivKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, nil, err
	}
	ephemeralAddr := crypto.PubkeyToAddress(ephemeralPrivKey.PublicKey)

	params := &generateParams{
		timestamp:   args.Timestamp,
		forceTime:   true,
		parentHash:  args.Parent,
		coinbase:    ephemeralAddr, // NOTE : overriding BuildBlockArgs.FeeRecipient TODO : make customizable
		random:      args.Random,
		extra:       args.Extra,
		withdrawals: args.Withdrawals,
		beaconRoot:  &args.BeaconRoot,
		// noUncle:     true,
		noTxs: false,
	}

	work, err := miner.prepareWork(params)
	if err != nil {
		return nil, nil, err
	}

	// Assume static 28000 gas transfers for both mev-share and proposer payments
	refundTransferCost := new(big.Int).Mul(big.NewInt(28000), work.header.BaseFee)

	profitPre := work.state.GetBalance(params.coinbase)

	for _, bundle := range bundles {
		// NOTE: failing bundles will cause the block to not be built!

		// apply bundle
		profitPreBundle := work.state.GetBalance(params.coinbase)
		if err := miner.rawCommitTransactions(work, bundle.Txs); err != nil {
			return nil, nil, err
		}
		profitPostBundle := work.state.GetBalance(params.coinbase)

		// calc & refund user if bundle has multiple txns and wants refund
		if len(bundle.Txs) > 1 && bundle.RefundPercent != nil {
			// Note: PoC logic, this could be gamed by not sending any eth to coinbase
			refundPrct := *bundle.RefundPercent
			if refundPrct == 0 {
				// default refund
				refundPrct = 10
			}
			bundleProfit := new(big.Int).Sub(profitPostBundle.ToBig(), profitPreBundle.ToBig())
			refundAmt := new(big.Int).Div(bundleProfit, big.NewInt(int64(refundPrct)))
			// subtract payment txn transfer costs
			refundAmt = new(big.Int).Sub(refundAmt, refundTransferCost)

			currNonce := work.state.GetNonce(ephemeralAddr)
			// HACK to include payment txn
			// multi refund block untested
			userTx := bundle.Txs[0] // NOTE : assumes first txn is refund recipient
			refundAddr, err := types.Sender(types.LatestSignerForChainID(userTx.ChainId()), userTx)
			if err != nil {
				return nil, nil, err
			}
			paymentTx, err := types.SignTx(types.NewTx(&types.LegacyTx{
				Nonce:    currNonce,
				To:       &refundAddr,
				Value:    refundAmt,
				Gas:      28000,
				GasPrice: work.header.BaseFee,
			}), work.signer, ephemeralPrivKey)

			if err != nil {
				return nil, nil, err
			}

			// commit payment txn
			if err := miner.rawCommitTransactions(work, types.Transactions{paymentTx}); err != nil {
				return nil, nil, err
			}
		}
	}
	if args.FillPending {
		if err := miner.commitPendingTxs(work); err != nil {
			return nil, nil, err
		}
	}

	profitPost := work.state.GetBalance(params.coinbase)
	proposerProfit := new(big.Int).Set(profitPost.ToBig()) // = post-pre-transfer_cost
	proposerProfit = proposerProfit.Sub(profitPost.ToBig(), profitPre.ToBig())
	proposerProfit = proposerProfit.Sub(proposerProfit, refundTransferCost)

	currNonce := work.state.GetNonce(ephemeralAddr)
	paymentTx, err := types.SignTx(types.NewTx(&types.LegacyTx{
		Nonce:    currNonce,
		To:       &args.FeeRecipient,
		Value:    proposerProfit,
		Gas:      28000,
		GasPrice: work.header.BaseFee,
	}), work.signer, ephemeralPrivKey)
	if err != nil {
		return nil, nil, fmt.Errorf("could not sign proposer payment: %w", err)
	}

	// commit payment txn
	if err := miner.rawCommitTransactions(work, types.Transactions{paymentTx}); err != nil {
		return nil, nil, fmt.Errorf("could not sign proposer payment: %w", err)
	}

	log.Info("buildBlockFromBundles", "num_bundles", len(bundles), "num_txns", len(work.txs), "profit", proposerProfit)
	// TODO : Is it okay to set Uncle List to nil?
	body := types.Body{Transactions: work.txs, Withdrawals: params.withdrawals}
	block, err := miner.engine.FinalizeAndAssemble(miner.chain, work.header, work.state, &body, work.receipts)
	if err != nil {
		return nil, nil, err
	}
	return block, proposerProfit, nil
}
