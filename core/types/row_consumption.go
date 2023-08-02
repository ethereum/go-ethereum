package types

type RowUsage struct {
	IsOk            bool                 `json:"is_ok"`
	RowNumber       uint64               `json:"row_number"`
	RowUsageDetails []SubCircuitRowUsage `json:"row_usage_details"`
}

//go:generate gencodec -type SubCircuitRowUsage -out gen_row_consumption_json.go
type SubCircuitRowUsage struct {
	Name      string `json:"name" gencodec:"required"`
	RowNumber uint64 `json:"row_number" gencodec:"required"`
}

type RowConsumption []SubCircuitRowUsage
