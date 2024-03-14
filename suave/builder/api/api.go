package api

import (
	"context"
	"math/big"

	denebBuilder "github.com/attestantio/go-builder-client/api/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// TODO: Can we aggregate all the gencodec generation into a single file?
//go:generate go run github.com/fjl/gencodec -type BuildBlockArgs -field-override buildBlockArgsMarshaling -out gen_buildblockargs_json.go
//go:generate go run github.com/fjl/gencodec -type SimulateTransactionResult -field-override simulateTransactionResultMarshaling -out gen_simulatetxnresult_json.go
//go:generate go run github.com/fjl/gencodec -type SimulatedLog -field-override simulateLogMarshaling -out gen_simulateLog_json.go

type Bundle struct {
	BlockNumber     *big.Int           `json:"blockNumber,omitempty"` // if BlockNumber is set it must match DecryptionCondition!
	MaxBlock        *big.Int           `json:"maxBlock,omitempty"`
	Txs             types.Transactions `json:"txs"`
	RevertingHashes []common.Hash      `json:"revertingHashes,omitempty"`
	RefundPercent   *int               `json:"percent,omitempty"`
}

type BuildBlockArgs struct {
	Slot           uint64              `json:"slot"`
	ProposerPubkey []byte              `json:"proposerPubkey"`
	Parent         common.Hash         `json:"parent"`
	Timestamp      uint64              `json:"timestamp"`
	FeeRecipient   common.Address      `json:"feeRecipient"`
	GasLimit       uint64              `json:"gasLimit"`
	Random         common.Hash         `json:"random"`
	Withdrawals    []*types.Withdrawal `json:"withdrawals"`
	Extra          []byte              `json:"extra"`
}

// field type overrides for gencodec
type buildBlockArgsMarshaling struct {
	Slot           hexutil.Uint64
	ProposerPubkey hexutil.Bytes
	Timestamp      hexutil.Uint64
	GasLimit       hexutil.Uint64
	Extra          hexutil.Bytes
}

type SimulateTransactionResult struct {
	Egp     uint64          `json:"egp"`
	Logs    []*SimulatedLog `json:"logs"`
	Success bool            `json:"success"`
	Error   string          `json:"error"`
}

// field type overrides for gencodec
type simulateTransactionResultMarshaling struct {
	Egp hexutil.Uint64
}

type SimulatedLog struct {
	Data   []byte         `json:"data"`
	Addr   common.Address `json:"addr"`
	Topics []common.Hash  `json:"topics"`
}

type simulateLogMarshaling struct {
	Data hexutil.Bytes
}

// SubmitBlockRequest is an extension of the builder.SubmitBlockRequest with the root
// of the bid that needs to be signed
type SubmitBlockRequest struct {
	denebBuilder.SubmitBlockRequest
	Root phase0.Root
}

type API interface {
	NewSession(ctx context.Context, args *BuildBlockArgs) (string, error)
	AddTransaction(ctx context.Context, sessionId string, tx *types.Transaction) (*SimulateTransactionResult, error)
	BuildBlock(ctx context.Context, sessionId string) error
	Bid(ctx context.Context, sessioId string, blsPubKey phase0.BLSPubKey) (*SubmitBlockRequest, error)
	GetBalance(ctx context.Context, sessionId string, addr common.Address) (*big.Int, error)
}
