package miner

import (
	"errors"
	"fmt"
	"math/big"

	denebBuilder "github.com/attestantio/go-builder-client/api/deneb"
	builderV1 "github.com/attestantio/go-builder-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/flashbots/go-boost-utils/ssz"
	"github.com/holiman/uint256"
)

type BuilderConfig struct {
	ChainConfig *params.ChainConfig
	Engine      consensus.Engine
	EthBackend  Backend
	Chain       *core.BlockChain
	GasCeil     uint64
}

type BuilderArgs struct {
	ParentHash     common.Hash
	FeeRecipient   common.Address
	ProposerPubkey []byte
	Extra          []byte
	Slot           uint64
}

type Builder struct {
	env   *environment
	wrk   *worker
	args  *BuilderArgs
	block *types.Block
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
	// If the context is not set, the logs will not be recorded
	b.env.state.SetTxContext(txn.Hash(), b.env.tcount)

	logs, err := b.wrk.commitTransaction(b.env, txn)
	if err != nil {
		return &types.SimulateTransactionResult{
			Error:   err.Error(),
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
	b.block = block
	return block, nil
}

func (b *Builder) Bid(builderPubKey phase0.BLSPubKey) (*SubmitBlockRequest, error) {
	work := b.env

	if b.block == nil {
		return nil, fmt.Errorf("block not built")
	}

	envelope := engine.BlockToExecutableData(b.block, totalFees(b.block, work.receipts), work.sidecars)
	payload, err := executableDataToDenebExecutionPayload(envelope.ExecutionPayload)
	if err != nil {
		return nil, err
	}

	value, overflow := uint256.FromBig(envelope.BlockValue)
	if overflow {
		return nil, fmt.Errorf("block value %v overflows", *envelope.BlockValue)
	}
	var proposerPubkey [48]byte
	copy(proposerPubkey[:], b.args.ProposerPubkey)

	blockBidMsg := builderV1.BidTrace{
		Slot:                 b.args.Slot,
		ParentHash:           phase0.Hash32(payload.ParentHash),
		BlockHash:            phase0.Hash32(payload.BlockHash),
		BuilderPubkey:        builderPubKey,
		ProposerPubkey:       phase0.BLSPubKey(proposerPubkey),
		ProposerFeeRecipient: bellatrix.ExecutionAddress(b.args.FeeRecipient),
		GasLimit:             envelope.ExecutionPayload.GasLimit,
		GasUsed:              envelope.ExecutionPayload.GasUsed,
		Value:                value,
	}

	genesisForkVersion := phase0.Version{0x00, 0x00, 0x10, 0x20}
	builderSigningDomain := ssz.ComputeDomain(ssz.DomainTypeAppBuilder, genesisForkVersion, phase0.Root{})

	root, err := ssz.ComputeSigningRoot(&blockBidMsg, builderSigningDomain)
	if err != nil {
		return nil, err
	}

	bidRequest := SubmitBlockRequest{
		Root: phase0.Root(root),
		SubmitBlockRequest: denebBuilder.SubmitBlockRequest{
			Message:          &blockBidMsg,
			ExecutionPayload: payload,
			Signature:        phase0.BLSSignature{},
			BlobsBundle:      &denebBuilder.BlobsBundle{},
		},
	}
	return &bidRequest, nil
}

// SubmitBlockRequest is an extension of the builder.SubmitBlockRequest with the root
// of the bid that needs to be signed
type SubmitBlockRequest struct {
	denebBuilder.SubmitBlockRequest
	Root phase0.Root
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

func executableDataToDenebExecutionPayload(data *engine.ExecutableData) (*deneb.ExecutionPayload, error) {
	transactionData := make([]bellatrix.Transaction, len(data.Transactions))
	for i, tx := range data.Transactions {
		transactionData[i] = bellatrix.Transaction(tx)
	}

	withdrawalData := make([]*capella.Withdrawal, len(data.Withdrawals))
	for i, wd := range data.Withdrawals {
		withdrawalData[i] = &capella.Withdrawal{
			Index:          capella.WithdrawalIndex(wd.Index),
			ValidatorIndex: phase0.ValidatorIndex(wd.Validator),
			Address:        bellatrix.ExecutionAddress(wd.Address),
			Amount:         phase0.Gwei(wd.Amount),
		}
	}

	baseFeePerGas := new(uint256.Int)
	if baseFeePerGas.SetFromBig(data.BaseFeePerGas) {
		return nil, errors.New("base fee per gas: overflow")
	}

	return &deneb.ExecutionPayload{
		ParentHash:    [32]byte(data.ParentHash),
		FeeRecipient:  [20]byte(data.FeeRecipient),
		StateRoot:     [32]byte(data.StateRoot),
		ReceiptsRoot:  [32]byte(data.ReceiptsRoot),
		LogsBloom:     types.BytesToBloom(data.LogsBloom),
		PrevRandao:    [32]byte(data.Random),
		BlockNumber:   data.Number,
		GasLimit:      data.GasLimit,
		GasUsed:       data.GasUsed,
		Timestamp:     data.Timestamp,
		ExtraData:     data.ExtraData,
		BaseFeePerGas: baseFeePerGas,
		BlockHash:     [32]byte(data.BlockHash),
		Transactions:  transactionData,
		Withdrawals:   withdrawalData,
	}, nil
}
