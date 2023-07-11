package rawdb

import (
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

func TestReadBlockRowConsumption(t *testing.T) {
	l2BlockHash := common.BigToHash(big.NewInt(10))
	rc := types.RowConsumption{
		Rows: 11,
	}
	db := NewMemoryDatabase()
	WriteBlockRowConsumption(db, l2BlockHash, rc)
	got := ReadBlockRowConsumption(db, l2BlockHash)
	if got == nil || got.Rows != rc.Rows {
		t.Fatal("RowConsumption mismatch", "expected", rc, "got", got)
	}
}
