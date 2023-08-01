package types

import (
	"github.com/scroll-tech/go-ethereum/common/hexutil"
)

type RowUsage struct {
	IsOk            bool                 `json:"is_ok"`
	RowNumber       uint64               `json:"row_number"`
	RowUsageDetails []SubCircuitRowUsage `json:"row_usage_details"`
}

//go:generate gencodec -type SubCircuitRowUsage -field-override subCircuitRowUsageMarshaling -out gen_row_consumption_json.go
type SubCircuitRowUsage struct {
	Name      string `json:"name" gencodec:"required"`
	RowNumber uint64 `json:"row_number" gencodec:"required"`
}

type RowConsumption []SubCircuitRowUsage

// field type overrides for gencodec
type subCircuitRowUsageMarshaling struct {
	RowNumber hexutil.Uint64
}
