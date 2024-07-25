package rawdb

import (
	"math/big"
	"sync"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

func TestReadWriteNumSkippedTransactions(t *testing.T) {
	blockNumbers := []uint64{
		1,
		1 << 2,
		1 << 8,
		1 << 16,
		1 << 32,
	}

	db := NewMemoryDatabase()
	for _, num := range blockNumbers {
		writeNumSkippedTransactions(db, num)
		got := ReadNumSkippedTransactions(db)

		if got != num {
			t.Fatal("Num skipped transactions mismatch", "expected", num, "got", got)
		}
	}
}

func newTestTransaction(queueIndex uint64) *types.Transaction {
	l1msg := types.L1MessageTx{
		QueueIndex: queueIndex,
		Gas:        0,
		To:         &common.Address{},
		Value:      big.NewInt(0),
		Data:       nil,
		Sender:     common.Address{},
	}
	return types.NewTx(&l1msg)
}

func TestReadWriteSkippedTransactionNoIndex(t *testing.T) {
	tx := newTestTransaction(123)
	db := NewMemoryDatabase()
	writeSkippedTransaction(db, tx, nil, "random reason", 1, &common.Hash{1})
	got := ReadSkippedTransaction(db, tx.Hash())
	if got == nil || got.Tx.Hash() != tx.Hash() || got.Reason != "random reason" || got.BlockNumber != 1 || got.BlockHash == nil || *got.BlockHash != (common.Hash{1}) {
		t.Fatal("Skipped transaction mismatch", "got", got)
	}
}

func TestReadSkippedTransactionV1AsV2(t *testing.T) {
	tx := newTestTransaction(123)
	db := NewMemoryDatabase()
	writeSkippedTransactionV1(db, tx, "random reason", 1, &common.Hash{1})
	got := ReadSkippedTransaction(db, tx.Hash())
	if got == nil || got.Tx.Hash() != tx.Hash() || got.Reason != "random reason" || got.BlockNumber != 1 || got.BlockHash == nil || *got.BlockHash != (common.Hash{1}) {
		t.Fatal("Skipped transaction mismatch", "got", got)
	}
}

func TestReadWriteSkippedTransaction(t *testing.T) {
	tx := newTestTransaction(123)
	db := NewMemoryDatabase()
	WriteSkippedTransaction(db, tx, nil, "random reason", 1, &common.Hash{1})
	got := ReadSkippedTransaction(db, tx.Hash())
	if got == nil || got.Tx.Hash() != tx.Hash() || got.Reason != "random reason" || got.BlockNumber != 1 || got.BlockHash == nil || *got.BlockHash != (common.Hash{1}) {
		t.Fatal("Skipped transaction mismatch", "got", got)
	}
	count := ReadNumSkippedTransactions(db)
	if count != 1 {
		t.Fatal("Skipped transaction count mismatch", "expected", 1, "got", count)
	}
	hash := ReadSkippedTransactionHash(db, 0)
	if hash == nil || *hash != tx.Hash() {
		t.Fatal("Skipped L1 message hash mismatch", "expected", tx.Hash(), "got", hash)
	}
}

func TestSkippedTransactionConcurrentUpdate(t *testing.T) {
	count := 20
	tx := newTestTransaction(123)
	db := NewMemoryDatabase()
	var wg sync.WaitGroup
	for ii := 0; ii < count; ii++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			WriteSkippedTransaction(db, tx, nil, "random reason", 1, &common.Hash{1})
		}()
	}
	wg.Wait()
	got := ReadNumSkippedTransactions(db)
	if got != uint64(count) {
		t.Fatal("Skipped transaction count mismatch", "expected", count, "got", got)
	}
}

func TestIterateSkippedTransactions(t *testing.T) {
	db := NewMemoryDatabase()

	txs := []*types.Transaction{
		newTestTransaction(1),
		newTestTransaction(2),
		newTestTransaction(3),
		newTestTransaction(4),
		newTestTransaction(5),
	}

	for _, tx := range txs {
		WriteSkippedTransaction(db, tx, nil, "random reason", 1, &common.Hash{1})
	}

	// simulate skipped L2 tx that's not included in the index
	l2tx := newTestTransaction(6)
	writeSkippedTransaction(db, l2tx, nil, "random reason", 1, &common.Hash{1})

	it := IterateSkippedTransactionsFrom(db, 2)
	defer it.Release()

	for ii := 2; ii < len(txs); ii++ {
		finished := !it.Next()
		if finished {
			t.Fatal("Iterator terminated early", "ii", ii)
		}

		index := it.Index()
		if index != uint64(ii) {
			t.Fatal("Invalid skipped transaction index", "expected", ii, "got", index)
		}

		hash := it.TransactionHash()
		if hash != txs[ii].Hash() {
			t.Fatal("Invalid skipped transaction hash", "expected", txs[ii].Hash(), "got", hash)
		}
	}

	finished := !it.Next()
	if !finished {
		t.Fatal("Iterator did not terminate")
	}
}
