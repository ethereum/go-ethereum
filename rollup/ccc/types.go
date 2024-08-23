package ccc

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

type RowUsage struct {
	IsOk            bool                       `json:"is_ok"`
	RowNumber       uint64                     `json:"row_number"`
	RowUsageDetails []types.SubCircuitRowUsage `json:"row_usage_details"`
}

type WrappedRowUsage struct {
	AccRowUsage *RowUsage `json:"acc_row_usage,omitempty"`
	Error       string    `json:"error,omitempty"`
}

type WrappedTxNum struct {
	TxNum uint64 `json:"tx_num"`
	Error string `json:"error,omitempty"`
}
