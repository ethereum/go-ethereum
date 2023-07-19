package types

import (
	"github.com/scroll-tech/go-ethereum/common/hexutil"
)

//go:generate gencodec -type SubCircuitRowConsumption -field-override subCircuitRowConsumptionMarshaling -out gen_row_consumption_json.go
type SubCircuitRowConsumption struct {
	CircuitName string `json:"circuitName" gencodec:"required"`
	Rows        uint64 `json:"rows" gencodec:"required"`
}

type RowConsumption []SubCircuitRowConsumption

// field type overrides for gencodec
type subCircuitRowConsumptionMarshaling struct {
	Rows hexutil.Uint64
}
