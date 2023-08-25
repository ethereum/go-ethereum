package circuitcapacitychecker

import (
	"errors"

	"github.com/scroll-tech/go-ethereum/core/types"
)

var (
	ErrUnknown                     = errors.New("unknown circuit capacity checker error")
	ErrBlockRowConsumptionOverflow = errors.New("block row consumption overflow")
)

type WrappedRowUsage struct {
	AccRowUsage *types.RowUsage `json:"acc_row_usage,omitempty"`
	Error       string          `json:"error,omitempty"`
}
