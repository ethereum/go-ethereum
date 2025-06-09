package bor

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/bor/clerk"
	"github.com/ethereum/go-ethereum/consensus/bor/statefull"
	"github.com/ethereum/go-ethereum/core/types"
)

//go:generate mockgen -destination=./genesis_contract_mock.go -package=bor . GenesisContract
type GenesisContract interface {
	CommitState(event *clerk.EventRecordWithTime, state vm.StateDB, header *types.Header, chCtx statefull.ChainContext, tracer *tracing.Hooks) (uint64, error)
	LastStateId(state vm.StateDB, number uint64, hash common.Hash) (*big.Int, error)
}
