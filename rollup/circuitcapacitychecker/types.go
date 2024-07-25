package circuitcapacitychecker

import (
	"errors"

	"github.com/scroll-tech/go-ethereum/core/types"
)

var (
	ErrUnknown                     = errors.New("unknown circuit capacity checker error")
	ErrBlockRowConsumptionOverflow = errors.New("block row consumption overflow")
)

type WrappedCommonResult struct {
	Error string `json:"error,omitempty"`
}

type WrappedRowUsage struct {
	AccRowUsage *types.RowUsage `json:"acc_row_usage,omitempty"`
	Error       string          `json:"error,omitempty"`
}

type WrappedTxNum struct {
	TxNum uint64 `json:"tx_num"`
	Error string `json:"error,omitempty"`
}
