package monitor

import (
	"math/big"
	"math/rand"
	"strconv"
)

func GenerateBlockMockData(blockId *big.Int) *BlockData {
	blockData := &BlockData{}
	blockData.BlockId = blockId
	blockData.TransactionDataList = []TransactionData{}
	for i := 0; i < rand.Intn(100)+50; i++ {
		blockData.TransactionDataList = append(blockData.TransactionDataList, *GenerateTransactionMockData(i))
	}
	return blockData
}

func GenerateTransactionMockData(transactionIndex int) *TransactionData {
	transactionData := &TransactionData{}
	transactionData.TransactionIndex = transactionIndex
	transactionData.OperationDataList = []OperationData{}
	for i := 0; i < rand.Intn(10000)+5000; i++ {
		transactionData.OperationDataList = append(transactionData.OperationDataList,
			*GenerateOperationMockData("OP" + strconv.Itoa(rand.Intn(100))))
	}
	return transactionData
}

func GenerateOperationMockData(op string) *OperationData {
	operationData := OperationData{}
	operationData.Op = op
	operationData.DurationUsage = SystemDurationUsage{
		CpuData: map[string]float64{"percent": float64(rand.Intn(10000) / 10000)},
		MemData: map[string]float64{"percent": float64(rand.Intn(10000) / 10000)},
		IOData:  map[string]float64{"percent": float64(rand.Intn(10000) / 10000)},
	}
	return &operationData
}
