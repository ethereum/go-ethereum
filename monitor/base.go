package monitor

import (
	"math/big"
	"time"
)

type SystemUsage struct {
	CurrentTime time.Time          `bson:"current_time"`
	CpuData     map[string]float64 `bson:"cpu_data"`
	MemData     map[string]float64 `bson:"mem_data"`
	IOData      map[string]float64 `bson:"io_data"`
}

type SystemDurationUsage struct {
	DurationTime time.Duration      `bson:"duration_time"`
	CpuData      map[string]float64 `bson:"cpu_data"`
	MemData      map[string]float64 `bson:"mem_data"`
	IOData       map[string]float64 `bson:"io_data"`
}

type OperationData struct {
	Op            string
	DurationUsage SystemDurationUsage `bson:"duration_usage"`
}

type TransactionData struct {
	TransactionIndex  int
	OperationDataList []OperationData `bson:"operations"`
}

type BlockData struct {
	TransactionDataList []TransactionData
	BlockId             *big.Int
}
