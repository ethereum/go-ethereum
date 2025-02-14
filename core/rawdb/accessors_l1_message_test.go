package rawdb

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

func TestReadWriteSyncedL1BlockNumber(t *testing.T) {
	blockNumbers := []uint64{
		1,
		1 << 2,
		1 << 8,
		1 << 16,
		1 << 32,
	}

	db := NewMemoryDatabase()
	for _, num := range blockNumbers {
		WriteSyncedL1BlockNumber(db, num)
		got := ReadSyncedL1BlockNumber(db)

		if got == nil || *got != num {
			t.Fatal("Block number mismatch", "expected", num, "got", got)
		}
	}
}

func newL1MessageTx(queueIndex uint64) types.L1MessageTx {
	return types.L1MessageTx{
		QueueIndex: queueIndex,
		Gas:        0,
		To:         &common.Address{},
		Value:      big.NewInt(0),
		Data:       nil,
		Sender:     common.Address{},
	}
}

func TestReadWriteL1Message(t *testing.T) {
	queueIndex := uint64(123)
	msg := newL1MessageTx(queueIndex)
	db := NewMemoryDatabase()
	WriteL1Messages(db, []types.L1MessageTx{msg})
	got := ReadL1Message(db, queueIndex)
	if got == nil || got.QueueIndex != queueIndex {
		t.Fatal("L1 message mismatch", "expected", queueIndex, "got", got)
	}

	max := ReadHighestSyncedQueueIndex(db)
	if max != 123 {
		t.Fatal("max index mismatch", "expected", 123, "got", max)
	}
}

func TestIterateL1Message(t *testing.T) {
	msgs := []types.L1MessageTx{
		newL1MessageTx(100),
		newL1MessageTx(101),
		newL1MessageTx(103),
		newL1MessageTx(200),
		newL1MessageTx(1000),
	}

	db := NewMemoryDatabase()
	WriteL1Messages(db, msgs)

	max := ReadHighestSyncedQueueIndex(db)
	if max != 1000 {
		t.Fatal("max index mismatch", "expected", 1000, "got", max)
	}

	it := iterateL1MessagesFrom(db, 103)
	defer it.Release()

	for ii := 2; ii < len(msgs); ii++ {
		finished := !it.Next()
		if finished {
			t.Fatal("Iterator terminated early", "ii", ii)
		}

		got := it.L1Message()
		if got.QueueIndex != msgs[ii].QueueIndex {
			t.Fatal("Invalid result", "expected", msgs[ii].QueueIndex, "got", got.QueueIndex)
		}
	}

	finished := !it.Next()
	if !finished {
		t.Fatal("Iterator did not terminate")
	}
}

func TestReadL1MessageTxRange(t *testing.T) {
	msgs := []types.L1MessageTx{
		newL1MessageTx(100),
		newL1MessageTx(101),
		newL1MessageTx(102),
		newL1MessageTx(103),
	}

	db := NewMemoryDatabase()
	WriteL1Messages(db, msgs)

	got := ReadL1MessagesV1From(db, 101, 3)

	if len(got) != 3 {
		t.Fatal("Invalid length", "expected", 3, "got", len(got))
	}

	if got[0].QueueIndex != 101 || got[1].QueueIndex != 102 || got[2].QueueIndex != 103 {
		t.Fatal("Invalid result", "got", got)
	}
}

func TestReadWriteLastL1MessageInL2Block(t *testing.T) {
	inputs := []uint64{
		1,
		1 << 2,
		1 << 8,
		1 << 16,
		1 << 32,
	}

	db := NewMemoryDatabase()
	for _, num := range inputs {
		l2BlockHash := common.Hash{byte(num)}
		WriteFirstQueueIndexNotInL2Block(db, l2BlockHash, num)
		got := ReadFirstQueueIndexNotInL2Block(db, l2BlockHash)

		if got == nil || *got != num {
			t.Fatal("Enqueue index mismatch", "expected", num, "got", got)
		}
	}
}

func TestIterationStopsAtMaxQueueIndex(t *testing.T) {
	msgs := []types.L1MessageTx{
		newL1MessageTx(100),
		newL1MessageTx(101),
		newL1MessageTx(102),
		newL1MessageTx(103),
	}

	db := NewMemoryDatabase()
	WriteL1Messages(db, msgs)

	// artificially change max index from 103 to 102
	WriteHighestSyncedQueueIndex(db, 102)

	// iteration should terminate at 102 and not read 103
	got := ReadL1MessagesV1From(db, 100, 10)

	if len(got) != 3 {
		t.Fatal("Invalid length", "expected", 3, "got", len(got))
	}
}

func TestIterateL1MessagesV1From(t *testing.T) {
	msgs := []types.L1MessageTx{
		newL1MessageTx(100),
		newL1MessageTx(101),
		newL1MessageTx(102),
		newL1MessageTx(103),
		newL1MessageTx(104),
	}

	db := NewMemoryDatabase()
	WriteL1Messages(db, msgs)
	WriteL1MessageV2StartIndex(db, 103)

	it := IterateL1MessagesV1From(db, 100)
	defer it.Release()

	for _, msg := range msgs {
		if msg.QueueIndex < 103 {
			require.True(t, it.Next(), "Iterator terminated early")
			require.Equal(t, msg.QueueIndex, it.L1Message().QueueIndex, "Invalid result")
		} else {
			require.Falsef(t, it.Next(), "Iterator did not terminate, queueIndex %d", msg.QueueIndex)
		}
	}

	finished := !it.Next()
	if !finished {
		t.Fatal("Iterator did not terminate")
	}
}

func TestIterateL1MessagesV2From(t *testing.T) {
	msgs := []types.L1MessageTx{
		newL1MessageTx(100),
		newL1MessageTx(101),
		newL1MessageTx(102),
		newL1MessageTx(103),
		newL1MessageTx(104),
	}

	db := NewMemoryDatabase()
	WriteL1Messages(db, msgs)

	// no L1MessageV2 in the database
	{
		it := IterateL1MessagesV2From(db, 100)
		require.Falsef(t, it.Next(), "Iterator did not terminate")
		it.Release()
	}

	WriteL1MessageV2StartIndex(db, 103)
	// L1MessageV2 in the database starting from 103 -> no iteration
	{
		it := IterateL1MessagesV2From(db, 100)
		require.Falsef(t, it.Next(), "Iterator did not terminate")
		it.Release()
	}

	it := IterateL1MessagesV2From(db, 103)
	defer it.Release()

	for _, msg := range msgs {
		if msg.QueueIndex >= 103 {
			require.True(t, it.Next(), "Iterator terminated early")
			require.Equal(t, msg.QueueIndex, it.L1Message().QueueIndex, "Invalid result")
		}
	}

	finished := !it.Next()
	if !finished {
		t.Fatal("Iterator did not terminate")
	}
}
