package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// StateChanges includes all changes after a StateSnapshot has been plattened and includes the state objects, refunds and logs.
type StateChanges struct {
	StateObjects map[common.Address]*StateObject
	Refund       *big.Int
	Logs         []*vm.Log
}
