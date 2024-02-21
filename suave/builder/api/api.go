package api

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

//go:generate go run github.com/fjl/gencodec -type BuildBlockArgs -field-override buildBlockArgsMarshaling -out gen_buildblockargs_json.go

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

type API interface {
	NewSession(ctx context.Context, args *BuildBlockArgs) (string, error)
	AddTransaction(ctx context.Context, sessionId string, tx *types.Transaction) (*types.SimulateTransactionResult, error)
	BuildBlock(ctx context.Context, sessionId string) error
}
