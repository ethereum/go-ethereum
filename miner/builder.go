package miner

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

type BuilderConfig struct {
	ChainConfig *params.ChainConfig
	Engine      consensus.Engine
	EthBackend  Backend
	Chain       *core.BlockChain
	GasCeil     uint64
}

type BuilderArgs struct {
	ParentHash   common.Hash
	FeeRecipient common.Address
	Extra        []byte
}

type Builder struct {
	env  *environment
	wrk  *worker
	args *BuilderArgs

	profitPre *big.Int
}

func NewBuilder(config *BuilderConfig, args *BuilderArgs) (*Builder, error) {
	b := &Builder{
		args: args,
	}

	b.wrk = &worker{
		config: &Config{
			GasCeil: config.GasCeil,
		},
		eth:         config.EthBackend,
		chainConfig: config.ChainConfig,
		engine:      config.Engine,
		chain:       config.Chain,
	}

	workerParams := &generateParams{
		parentHash: args.ParentHash,
		forceTime:  false,
		coinbase:   args.FeeRecipient,
		extra:      args.Extra,
	}
	env, err := b.wrk.prepareWork(workerParams)
	if err != nil {
		return nil, err
	}

	env.gasPool = new(core.GasPool).AddGas(env.header.GasLimit)
	b.env = env
	b.profitPre = env.state.GetBalance(env.coinbase)

	return b, nil
}

type SBundle struct {
	BlockNumber     *big.Int           `json:"blockNumber,omitempty"` // if BlockNumber is set it must match DecryptionCondition!
	MaxBlock        *big.Int           `json:"maxBlock,omitempty"`
	Txs             types.Transactions `json:"txs"`
	RevertingHashes []common.Hash      `json:"revertingHashes,omitempty"`
	RefundPercent   *int               `json:"percent,omitempty"`
}

func (b *Builder) AddTransaction(txn *types.Transaction) (*types.SimulateTransactionResult, error) {
	logs, err := b.wrk.commitTransaction(b.env, txn)
	if err != nil {
		return &types.SimulateTransactionResult{
			Success: false,
		}, nil
	}
	return receiptToSimResult(&types.Receipt{Logs: logs}), nil
}

func (b *Builder) AddBundles(bundles []*SBundle) error {
	for _, bundle := range bundles {
		if err := b.AddBundle(bundle); err != nil {
			return err
		}
	}
	return nil
}

func (b *Builder) AddBundle(bundle *SBundle) error {
	work := b.env

	// Assume static 28000 gas transfers for both mev-share and proposer payments
	refundTransferCost := new(big.Int).Mul(big.NewInt(28000), work.header.BaseFee)

	// create ephemeral addr and private key for payment txn
	ephemeralPrivKey, err := crypto.GenerateKey()
	if err != nil {
		return err
	}
	ephemeralAddr := crypto.PubkeyToAddress(ephemeralPrivKey.PublicKey)

	// apply bundle
	profitPreBundle := work.state.GetBalance(b.env.coinbase)
	if err := b.wrk.rawCommitTransactions(work, bundle.Txs); err != nil {
		return err
	}
	profitPostBundle := work.state.GetBalance(b.env.coinbase)

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

		currNonce := work.state.GetNonce(ephemeralAddr)
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
			GasPrice: work.header.BaseFee,
		}), work.signer, ephemeralPrivKey)

		if err != nil {
			return err
		}

		// commit payment txn
		if err := b.wrk.rawCommitTransactions(work, types.Transactions{paymentTx}); err != nil {
			return err
		}
	}

	return nil
}

func (b *Builder) FillPending() error {
	if err := b.wrk.commitPendingTxs(b.env); err != nil {
		return err
	}
	return nil
}

func (b *Builder) BuildBlock() (*types.Block, error) {
	work := b.env

	// Assume static 28000 gas transfers for both mev-share and proposer payments
	refundTransferCost := new(big.Int).Mul(big.NewInt(28000), work.header.BaseFee)

	// create ephemeral addr and private key for payment txn
	ephemeralPrivKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	ephemeralAddr := crypto.PubkeyToAddress(ephemeralPrivKey.PublicKey)

	profitPost := work.state.GetBalance(b.env.coinbase)
	proposerProfit := new(big.Int).Set(profitPost) // = post-pre-transfer_cost
	proposerProfit = proposerProfit.Sub(profitPost, b.profitPre)
	proposerProfit = proposerProfit.Sub(proposerProfit, refundTransferCost)

	currNonce := work.state.GetNonce(ephemeralAddr)
	paymentTx, err := types.SignTx(types.NewTx(&types.LegacyTx{
		Nonce:    currNonce,
		To:       &b.args.FeeRecipient,
		Value:    proposerProfit,
		Gas:      28000,
		GasPrice: work.header.BaseFee,
	}), work.signer, ephemeralPrivKey)
	if err != nil {
		return nil, fmt.Errorf("could not sign proposer payment: %w", err)
	}

	// commit payment txn
	if err := b.wrk.rawCommitTransactions(work, types.Transactions{paymentTx}); err != nil {
		return nil, fmt.Errorf("could not sign proposer payment: %w", err)
	}

	block, err := b.wrk.engine.FinalizeAndAssemble(b.wrk.chain, work.header, work.state, work.txs, nil, work.receipts, nil)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func receiptToSimResult(receipt *types.Receipt) *types.SimulateTransactionResult {
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
	return result
}
