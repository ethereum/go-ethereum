package builder

import (
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/suave/builder/api"
)

type builder struct {
	config             *builderConfig
	txns               []*types.Transaction
	receipts           []*types.Receipt
	state              *state.StateDB
	gasPool            *core.GasPool
	gasUsed            *uint64
	signer             types.Signer
	args               api.BuildBlockArgs
	coinbasePreBalance *big.Int
	engine             consensus.Engine
}

type builderConfig struct {
	preState    *state.StateDB
	header      *types.Header
	config      *params.ChainConfig
	context     core.ChainContext
	chainReader consensus.ChainHeaderReader

	// newpayloadTimeout is the maximum timeout allowance for creating payload.
	// The default value is 2 seconds but node operator can set it to arbitrary
	// large value. A large timeout allowance may cause Geth to fail creating
	// a non-empty payload within the specified time and eventually miss the slot
	// in case there are some computation expensive transactions in txpool.
	newpayloadTimeout time.Duration
}

func newBuilder(config *builderConfig) *builder {
	gp := core.GasPool(config.header.GasLimit)
	var gasUsed uint64

	return &builder{
		config:             config,
		state:              config.preState.Copy(),
		gasPool:            &gp,
		gasUsed:            &gasUsed,
		signer:             types.MakeSigner(config.config, config.header.Number, config.header.Time),
		coinbasePreBalance: config.preState.GetBalance(config.header.Coinbase),
	}
}

func (b *builder) takeSnapshot() func() {
	indx := len(b.txns)
	snap := b.state.Snapshot()

	return func() {
		b.txns = b.txns[:indx]
		b.receipts = b.receipts[:indx]
		b.state.RevertToSnapshot(snap)
	}
}

func (b *builder) AddBundle(bundle api.Bundle) error {
	revertFn := b.takeSnapshot()

	// create ephemeral addr and private key for payment txn
	ephemeralPrivKey, err := crypto.GenerateKey()
	if err != nil {
		return err
	}
	ephemeralAddr := crypto.PubkeyToAddress(ephemeralPrivKey.PublicKey)

	// Assume static 28000 gas transfers for both mev-share and proposer payments
	refundTransferCost := new(big.Int).Mul(big.NewInt(28000), b.config.header.BaseFee)

	// apply bundle
	profitPreBundle := b.state.GetBalance(b.config.header.Coinbase)
	if err := b.AddTransactions(bundle.Txs); err != nil {
		revertFn()
		return err
	}

	profitPostBundle := b.state.GetBalance(b.config.header.Coinbase)

	// calc & refund user if bundle has multiple txns and wants refund
	if len(bundle.Txs) > 1 && bundle.RefundPercent != nil {
		// Note: PoC logic, this could be gamed by not sending any eth to coinbase
		refundPrct := *bundle.RefundPercent
		if refundPrct == 0 {
			// default refund
			refundPrct = 10
		}
		bundleProfit := new(big.Int).Sub(profitPostBundle, profitPreBundle)
		refundAmt := new(big.Int).Div(bundleProfit, big.NewInt(int64(refundPrct)))
		// subtract payment txn transfer costs
		refundAmt = new(big.Int).Sub(refundAmt, refundTransferCost)

		currNonce := b.state.GetNonce(ephemeralAddr)

		// HACK to include payment txn
		// multi refund block untested
		userTx := bundle.Txs[0] // NOTE : assumes first txn is refund recipient
		refundAddr, err := types.Sender(types.LatestSignerForChainID(userTx.ChainId()), userTx)
		if err != nil {
			return err
		}
		paymentTx, err := types.SignTx(types.NewTx(&types.LegacyTx{
			Nonce:    currNonce,
			To:       &refundAddr,
			Value:    refundAmt,
			Gas:      28000,
			GasPrice: b.config.header.BaseFee,
		}), b.signer, ephemeralPrivKey)

		if err != nil {
			return err
		}

		// commit payment txn
		if _, err := b.AddTransaction(paymentTx); err != nil {
			revertFn()
			return err
		}
	}

	return nil
}

func (b *builder) AddTransactions(txns types.Transactions) error {
	revertFn := b.takeSnapshot()

	for _, txn := range txns {
		if _, err := b.AddTransaction(txn); err != nil {
			revertFn()
			return err
		}
	}

	return nil
}

func (b *builder) AddTransaction(txn *types.Transaction) (*types.SimulateTransactionResult, error) {
	dummyAuthor := common.Address{}

	vmConfig := vm.Config{
		NoBaseFee: true,
	}

	snap := b.state.Snapshot()

	b.state.SetTxContext(txn.Hash(), len(b.txns))
	receipt, err := core.ApplyTransaction(b.config.config, b.config.context, &dummyAuthor, b.gasPool, b.state, b.config.header, txn, b.gasUsed, vmConfig)
	if err != nil {
		b.state.RevertToSnapshot(snap)

		result := &types.SimulateTransactionResult{
			Success: false,
			Error:   err.Error(),
		}
		return result, nil
	}

	b.txns = append(b.txns, txn)
	b.receipts = append(b.receipts, receipt)

	result := &types.SimulateTransactionResult{
		Success: true,
		Logs:    []*types.SimulatedLog{},
	}
	for _, log := range receipt.Logs {
		result.Logs = append(result.Logs, &types.SimulatedLog{
			Addr:   log.Address,
			Topics: log.Topics,
			Data:   log.Data,
		})
	}

	return result, nil
}

func (b *builder) commitPendingTxs() error {
	interrupt := new(atomic.Int32)
	timer := time.AfterFunc(b.config.newpayloadTimeout, func() {
		interrupt.Store(commitInterruptTimeout)
	})
	defer timer.Stop()
	if err := b.fillTransactions(); err != nil {
		return err
	}
	return nil
}

func (b *builder) fillTransactions() error {
	// Split the pending transactions into locals and remotes
	// Fill the block with all available pending transactions.
	pending := w.eth.TxPool().Pending(true)
	localTxs, remoteTxs := make(map[common.Address]types.Transactions), pending
	for _, account := range w.eth.TxPool().Locals() {
		if txs := remoteTxs[account]; len(txs) > 0 {
			delete(remoteTxs, account)
			localTxs[account] = txs
		}
	}
	if len(localTxs) > 0 {
		txs := types.NewTransactionsByPriceAndNonce(env.signer, localTxs, env.header.BaseFee)
		if err := b.commitTransactions(env, txs, interrupt); err != nil {
			return err
		}
	}
	if len(remoteTxs) > 0 {
		txs := types.NewTransactionsByPriceAndNonce(env.signer, remoteTxs, env.header.BaseFee)
		if err := w.commitTransactions(env, txs, interrupt); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) BuildBlock() error {
	if b.args.FillPending {
		if err := b.commitPendingTxs(); err != nil {
			return err
		}
	}

	// create ephemeral addr and private key for payment txn
	ephemeralPrivKey, err := crypto.GenerateKey()
	if err != nil {
		return err
	}
	ephemeralAddr := crypto.PubkeyToAddress(ephemeralPrivKey.PublicKey)

	// Assume static 28000 gas transfers for both mev-share and proposer payments
	refundTransferCost := new(big.Int).Mul(big.NewInt(28000), b.config.header.BaseFee)

	profitPost := b.state.GetBalance(b.config.header.Coinbase)
	proposerProfit := new(big.Int).Set(profitPost) // = post-pre-transfer_cost
	proposerProfit = proposerProfit.Sub(profitPost, b.coinbasePreBalance)
	proposerProfit = proposerProfit.Sub(proposerProfit, refundTransferCost)

	currNonce := b.state.GetNonce(ephemeralAddr)
	paymentTx, err := types.SignTx(types.NewTx(&types.LegacyTx{
		Nonce:    currNonce,
		To:       &b.args.FeeRecipient,
		Value:    proposerProfit,
		Gas:      28000,
		GasPrice: b.config.header.BaseFee,
	}), b.signer, ephemeralPrivKey)
	if err != nil {
		return fmt.Errorf("could not sign proposer payment: %w", err)
	}

	// commit payment txn
	if _, err := b.AddTransaction(paymentTx); err != nil {
		return err
	}

	block, err := b.engine.FinalizeAndAssemble(b.config.chainReader, b.config.header, b.state, b.txns, []*types.Header{}, b.receipts, b.args.Withdrawals)
	if err != nil {
		return err
	}

	fmt.Println("-- block --", block)
	return nil
}
