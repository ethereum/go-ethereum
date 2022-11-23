package bor

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/span"
	"github.com/ethereum/go-ethereum/consensus/bor/valset"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

//go:generate mockgen -destination=./span_mock.go -package=bor . Spanner
type Spanner interface {
	GetCurrentSpan(headerHash common.Hash) (*span.Span, error)
	GetCurrentValidators(headerHash common.Hash, blockNumber uint64) ([]*valset.Validator, error)
	CommitSpan(heimdallSpan span.HeimdallSpan, state *state.StateDB, header *types.Header, chainContext core.ChainContext) error
}
