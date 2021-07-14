package monitor

import (
	"fmt"
	"math/big"
	"strconv"
	"testing"
)

var TestMongoUri = "mongodb://geth:123456@localhost:27017/admin"

func BenchmarkNewMongoDb(b *testing.B) {
	lastMongoDb, err := NewMongoDb(TestMongoUri)
	if err != nil {
		b.Fatal("initialize mongodb failed")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mongoDb, err := NewMongoDb(TestMongoUri)
		if err != nil {
			b.Fatal("initialize mongodb failed")
		}
		if mongoDb != lastMongoDb {
			b.Fatal("class mongodb is not singleton")
		}
	}
}

func TestMongoDb_GetBlockData(t *testing.T) {

	blockName := big.NewInt(1)
	db, err := NewMongoDb(TestMongoUri)
	if err != nil {
		t.Error("Failed to create mongodb")
		return
	}
	err = db.SaveBlockData(*GenerateBlockMockData(blockName))
	if err != nil {
		t.Error("Failed to save block data")
		return
	}

	blockData, err1 := db.GetBlockData(blockName)
	if err1 != nil {
		t.Error("Could not get block data.")
		return
	} else {
		fmt.Print("block's name: ", blockData.BlockId, "\n")
		fmt.Print("Len of transactions: ", strconv.Itoa(len(blockData.TransactionDataList)), "\n")
	}

}
