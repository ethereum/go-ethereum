package monitor

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"math/big"
	"sync"
	"time"
)

var systemUsageMonitor SystemUsageMonitor
var monitorOnce sync.Once

func GetCurrentTime() time.Time {
	return time.Now()
}

type ISystemUsageMonitor interface {
	GetSystemUsage() map[string]float64
}

type SystemUsageMonitor struct {
	tool                      Tool
	db                        Idb
	currentBlockId            *big.Int
	currentTransactionIndex   int
	currentOperationName      string
	currentBlockData          BlockData
	currentTransactionData    TransactionData
	currentOperationData      OperationData
	operationStartSystemUsage SystemUsage
	operationEndSystemUsage   SystemUsage
}

func GetSystemUsageMonitor() *SystemUsageMonitor {

	monitorOnce.Do(func() {
		systemUsageMonitor = SystemUsageMonitor{
			tool: *NewTool(),
		}
		db, err := NewMongoDb("mongodb://geth:123456@localhost:27017/admin")
		if err != nil {
			log.Error("Failed to create mongo db")
		}
		systemUsageMonitor.SetDb(db)
	})
	return &systemUsageMonitor
}

func Init() {
	systemUsageMonitor := GetSystemUsageMonitor()
	db, err := NewMongoDb("mongodb://geth:123456@localhost:27017/admin")
	if err != nil {
		log.Error("Failed to create mongo db")
	}
	systemUsageMonitor.SetDb(db)
}

func (sum *SystemUsageMonitor) SetDb(db Idb) {
	sum.db = db
}

func (sum *SystemUsageMonitor) SaveBlockData(blockData BlockData) error {
	if &sum.db == nil {
		return fmt.Errorf("db is not set")
	}
	if len(blockData.TransactionDataList) > 0 {
		return sum.db.SaveBlockData(blockData)
	} else {
		return nil
	}

}

func (sum *SystemUsageMonitor) IsInBlock() bool {
	if sum.currentBlockId != nil && sum.currentBlockId.Int64() >= 0 {
		return true
	} else {
		return false
	}
}

func (sum *SystemUsageMonitor) SaveTxData(txData TransactionData) error {
	if &sum.db == nil {
		return fmt.Errorf("db is not set")
	}
	return sum.db.SaveTxData(txData)
}

func (sum *SystemUsageMonitor) GetSystemCurrentUsage() *SystemUsage {
	return &SystemUsage{
		GetCurrentTime(),
		sum.tool.GetCpuData(),
		sum.tool.GetMemData(),
		sum.tool.GetIOData(),
	}
}

func getMapDataDiff(d1 map[string]float64, d2 map[string]float64) *map[string]float64 {
	diff := map[string]float64{}
	for k, v := range d1 {
		diff[k] = d2[k] - v
	}
	return &diff
}

func (sum *SystemUsageMonitor) BlockStart(blockId *big.Int) {
	sum.currentBlockId = blockId
	sum.currentBlockData = BlockData{
		BlockId:             blockId,
		TransactionDataList: []TransactionData{},
	}
}

func (sum *SystemUsageMonitor) BlockEnd() (blockData *BlockData) {
	sum.currentBlockId = big.NewInt(-1)
	ret := sum.currentBlockData
	sum.currentBlockData = BlockData{
		BlockId:             big.NewInt(-1),
		TransactionDataList: []TransactionData{},
	}
	return &ret
}

func (sum *SystemUsageMonitor) TransactionStart(txIndex int) {
	sum.currentTransactionData = TransactionData{
		TransactionIndex:  txIndex,
		OperationDataList: []OperationData{},
	}
}

func (sum *SystemUsageMonitor) TransactionEnd() (transactionData *TransactionData) {
	ret := sum.currentTransactionData
	sum.currentTransactionData = TransactionData{
		TransactionIndex:  -1,
		OperationDataList: []OperationData{},
	}
	sum.currentBlockData.TransactionDataList = append(sum.currentBlockData.TransactionDataList, ret)
	return &ret
}

func (sum *SystemUsageMonitor) OperationStart(op string) {
	sum.currentOperationData = OperationData{
		Op:            op,
		DurationUsage: SystemDurationUsage{},
	}
	sum.operationStartSystemUsage = *sum.GetSystemCurrentUsage()
}

func (sum *SystemUsageMonitor) OperationEnd(gas uint64) (operationData *OperationData) {
	sum.operationEndSystemUsage = *sum.GetSystemCurrentUsage()
	sum.currentOperationData.DurationUsage = *sum.GetOperationDurationUsage()
	sum.currentOperationData.UsedGas = gas
	ret := sum.currentOperationData
	sum.currentOperationData = OperationData{
		Op:            "",
		DurationUsage: SystemDurationUsage{},
		UsedGas:       gas,
	}
	sum.currentTransactionData.OperationDataList = append(sum.currentTransactionData.OperationDataList, ret)
	return &ret
}

func (sum *SystemUsageMonitor) GetOperationDurationUsage() *SystemDurationUsage {
	return &SystemDurationUsage{
		DurationTime: sum.operationEndSystemUsage.CurrentTime.Sub(sum.operationStartSystemUsage.CurrentTime),
		CpuData:      *getMapDataDiff(sum.operationStartSystemUsage.CpuData, sum.operationEndSystemUsage.CpuData),
		MemData:      *getMapDataDiff(sum.operationStartSystemUsage.MemData, sum.operationEndSystemUsage.MemData),
		IOData:       *getMapDataDiff(sum.operationStartSystemUsage.IOData, sum.operationEndSystemUsage.IOData),
	}
}

func (sum *SystemUsageMonitor) GetUsedTimeByTxHash(hash string) (*time.Duration, error) {
	transactionData, err := sum.GetTransactionDataByTxHash(hash)
	if err != nil {
		return nil, err
	}
	var allTime time.Duration
	for _, opData := range transactionData.OperationDataList {
		allTime += opData.DurationUsage.DurationTime
	}
	return &allTime, err
}

func (sum *SystemUsageMonitor) GetUsedGasByTxHash(hash string) (uint64, error) {
	transactionData, err := sum.GetTransactionDataByTxHash(hash)
	if err != nil {
		return uint64(0), err
	}

	return transactionData.UsedGas, err
}

func (sum *SystemUsageMonitor) GetTransactionDataByTxHash(hash string) (*TransactionData, error) {
	transactionData, err := sum.db.GetTransactionDataByTxHash(hash)
	if err != nil {
		return nil, errors.New("Can not find the tx by hash ")
	}
	return transactionData, nil
}

func (sdu *SystemDurationUsage) ToString() string {
	_json, err := json.MarshalIndent(sdu, "", "\t")
	if err != nil {
		log.Error(err.Error())
		return "Can not convert systemDurationUsage to string"
	}
	return string(_json)
}

func (scu *SystemUsage) ToString() string {
	_json, err := json.MarshalIndent(scu, "", "\t")
	if err != nil {
		log.Error(err.Error())
		return "Can not convert systemUsage to string"
	}
	return string(_json)
}
