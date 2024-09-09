package multicall

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type MultiCallMetaData[T interface{}] struct {
	Address     common.Address
	Data        []byte
	Deserialize func([]byte) (T, error)
}

type DeserializedMulticall3Result struct {
	Success bool
	Value   any
}

func (md *MultiCallMetaData[T]) Raw() RawMulticall {
	return RawMulticall{
		Address: md.Address,
		Data:    md.Data,
		Deserialize: func(data []byte) (any, error) {
			res, err := md.Deserialize(data)
			return any(res), err
		},
	}
}

type RawMulticall struct {
	Address     common.Address
	Data        []byte
	Deserialize func([]byte) (any, error)
}

type MulticallClient struct {
	Contract     *bind.BoundContract
	ABI          *abi.ABI
	Context      context.Context
	MaxBatchSize *uint64
}

type Multicall3Result struct {
	Success    bool
	ReturnData []byte
}

type ParamMulticall3Call3 struct {
	Target       common.Address
	AllowFailure bool
	CallData     []byte
}

/*
 * Some RPC providers may limit the amount of calldata you can send in one eth_call. This utility
 * provides a mechanism for chunking calls by the len(CallData) used.
 *
 * This function checks whether the calldata appended exceeds maxBatchSizeBytes
 */
func chunkCalls(allCalls []ParamMulticall3Call3, maxBatchSizeBytes uint64) [][]ParamMulticall3Call3 {
	results := [][]ParamMulticall3Call3{}

	currentBatchSize := uint64(0)
	currentBatch := []ParamMulticall3Call3{}

	for _, call := range allCalls {
		if (currentBatchSize + uint64(len(call.CallData))) > maxBatchSizeBytes {
			results = append(results, currentBatch)
			currentBatchSize = 0
			currentBatch = []ParamMulticall3Call3{}
		}

		currentBatch = append(currentBatch, call)
		currentBatchSize += uint64(len(call.CallData))
	}

	if len(currentBatch) > 0 {
		results = append(results, currentBatch)
	}

	return results
}
