package miner

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
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

func (b *Builder) FillPending() error {
	if err := b.wrk.commitPendingTxs(b.env); err != nil {
		return err
	}
	return nil
}

func (b *Builder) BuildBlock() (*types.Block, error) {
	work := b.env

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
