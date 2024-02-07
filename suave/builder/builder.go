package builder

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/suave/builder/api"
)

type builder struct {
	config   *builderConfig
	txns     []*types.Transaction
	receipts []*types.Receipt
	state    *state.StateDB
	gasPool  *core.GasPool
	gasUsed  *uint64
	signer   types.Signer
}

type builderConfig struct {
	preState *state.StateDB
	header   *types.Header
	config   *params.ChainConfig
	context  core.ChainContext
}

func newBuilder(config *builderConfig) *builder {
	gp := core.GasPool(config.header.GasLimit)
	var gasUsed uint64

	return &builder{
		config:  config,
		state:   config.preState.Copy(),
		gasPool: &gp,
		gasUsed: &gasUsed,
		signer:  types.MakeSigner(config.config, config.header.Number, config.header.Time),
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
