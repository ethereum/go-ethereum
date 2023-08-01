package circuitcapacitychecker

import (
	"errors"

	"github.com/scroll-tech/go-ethereum/core/types"
)

var (
	ErrUnknown                     = errors.New("unknown circuit capacity checker error")
	ErrTxRowConsumptionOverflow    = errors.New("tx row consumption oveflow")
	ErrBlockRowConsumptionOverflow = errors.New("block row consumption oveflow")
)

type WrappedRowUsage struct {
	AccRowUsage *types.RowUsage `json:"acc_row_usage,omitempty"`
	TxRowUsage  *types.RowUsage `json:"tx_row_usage,omitempty"`
	Error       string          `json:"error,omitempty"`
}
