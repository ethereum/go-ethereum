package monitor

import (
	"github.com/ethereum/go-ethereum/log"
	"math/big"
	"testing"
)

func TestNewSystemUsageMonitor(t *testing.T) {
	sum := GetSystemUsageMonitor()

	for i := 0; i < 1; i++ {
		sum.BlockStart(big.NewInt(int64(i)))
		MockTransactionRecord(sum)
		sum.BlockEnd()
	}
}

func TestNewSystemUsageMonitor_SaveBlockData(t *testing.T) {
	sum := GetSystemUsageMonitor()

	for i := 0; i < 1; i++ {
		sum.BlockStart(big.NewInt(int64(i)))
		MockTransactionRecord(sum)
		blockData := sum.BlockEnd()
		db, err := NewMongoDb(TestMongoUri)
		if err != nil {
			log.Error("Failed to create mongo db")
		}

		sum.SetDb(db)
		err = sum.SaveBlockData(*blockData)
		if err != nil {
			log.Error("Unable to save block data")
		}
	}
}

func TestNewSystemUsageMonitor_SaveTransactionData(t *testing.T) {
	sum := GetSystemUsageMonitor()

	for i := 0; i < 1; i++ {
		sum.TransactionStart(i)
		MockOperationRecord(sum)
		txData := sum.TransactionEnd()
		db, err := NewMongoDb(TestMongoUri)
		if err != nil {
			log.Error("Failed to create mongo db")
		}

		sum.SetDb(db)
		err = sum.SaveTxData(*txData)
		if err != nil {
			log.Error("Unable to save tx data")
		}
	}
}

func MockTransactionRecord(sum *SystemUsageMonitor) {
	for i := 0; i < 10; i++ {
		sum.TransactionStart(i)
		MockOperationRecord(sum)
		sum.TransactionEnd()
	}
}

func MockOperationRecord(sum *SystemUsageMonitor) {
	for i := 0; i < 1000; i++ {
		sum.OperationStart("TEST")
		sum.OperationEnd()
	}
}

func BenchmarkNewSystemUsageMonitor(b *testing.B) {
	lastSum := GetSystemUsageMonitor()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum := GetSystemUsageMonitor()
		if lastSum != sum {
			log.Error("BenchmarkNewSystemUsageMonitor wrong!")
		}
	}
}

func BenchmarkSystemUsageMonitor_GetOperationDurationUsage(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum := GetSystemUsageMonitor()
		sum.OperationStart("TEST")
		sum.OperationEnd()
	}
}

func BenchmarkSystemUsageMonitor_GetSystemCurrentUsage(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum := GetSystemUsageMonitor()
		sum.GetSystemCurrentUsage()
	}
}

func TestSystemUsageMonitor_SetDb(t *testing.T) {
	sum := GetSystemUsageMonitor()
	db, err := NewMongoDb(TestMongoUri)

	if err != nil {
		t.Fatal(err)
	}
	sum.SetDb(db)
}
