package ethapi

import "github.com/ethereum/go-ethereum/metrics"

var (
	ethCallGasUsedHist = metrics.NewRegisteredHistogram("rpc/gas_used/eth_call", nil, metrics.NewExpDecaySample(1028, 0.015))
)
