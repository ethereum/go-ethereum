package blockstm

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
)

var randomness = rand.Intn(10) + 10

// create test data for a given txIdx and incarnation
func valueFor(txIdx, inc int) []byte {
	return []byte(fmt.Sprintf("%ver:%ver:%ver", txIdx*5, txIdx+inc, inc*5))
}

func getCommonAddress(i int) common.Address {
	return common.BigToAddress(big.NewInt(int64(i % randomness)))
}

func TestHelperFunctions(t *testing.T) {
	t.Parallel()

	ap1 := NewAddressKey(getCommonAddress(1))
	ap2 := NewAddressKey(getCommonAddress(2))

	mvh := MakeMVHashMap()

	mvh.Write(ap1, Version{0, 1}, valueFor(0, 1))
	mvh.Write(ap1, Version{0, 2}, valueFor(0, 2))
	res := mvh.Read(ap1, 0)
	require.Equal(t, -1, res.DepIdx())
	require.Equal(t, -1, res.Incarnation())
	require.Equal(t, 2, res.Status())

	mvh.Write(ap2, Version{1, 1}, valueFor(1, 1))
	mvh.Write(ap2, Version{1, 2}, valueFor(1, 2))
	res = mvh.Read(ap2, 1)
	require.Equal(t, -1, res.DepIdx())
	require.Equal(t, -1, res.Incarnation())
	require.Equal(t, 2, res.Status())

	mvh.Write(ap1, Version{2, 1}, valueFor(2, 1))
	mvh.Write(ap1, Version{2, 2}, valueFor(2, 2))
	res = mvh.Read(ap1, 2)
	require.Equal(t, 0, res.DepIdx())
	require.Equal(t, 2, res.Incarnation())
	require.Equal(t, valueFor(0, 2), res.Value().([]byte))
	require.Equal(t, 0, res.Status())
}

func TestFlushMVWrite(t *testing.T) {
	t.Parallel()

	ap1 := NewAddressKey(getCommonAddress(1))
	ap2 := NewAddressKey(getCommonAddress(2))

	mvh := MakeMVHashMap()

	var res MVReadResult

	wd := []WriteDescriptor{}

	wd = append(wd, WriteDescriptor{
		Path: ap1,
		V:    Version{0, 1},
		Val:  valueFor(0, 1),
	})
	wd = append(wd, WriteDescriptor{
		Path: ap1,
		V:    Version{0, 2},
		Val:  valueFor(0, 2),
	})
	wd = append(wd, WriteDescriptor{
		Path: ap2,
		V:    Version{1, 1},
		Val:  valueFor(1, 1),
	})
	wd = append(wd, WriteDescriptor{
		Path: ap2,
		V:    Version{1, 2},
		Val:  valueFor(1, 2),
	})
	wd = append(wd, WriteDescriptor{
		Path: ap1,
		V:    Version{2, 1},
		Val:  valueFor(2, 1),
	})
	wd = append(wd, WriteDescriptor{
		Path: ap1,
		V:    Version{2, 2},
		Val:  valueFor(2, 2),
	})

	mvh.FlushMVWriteSet(wd)

	res = mvh.Read(ap1, 0)
	require.Equal(t, -1, res.DepIdx())
	require.Equal(t, -1, res.Incarnation())
	require.Equal(t, 2, res.Status())

	res = mvh.Read(ap2, 1)
	require.Equal(t, -1, res.DepIdx())
	require.Equal(t, -1, res.Incarnation())
	require.Equal(t, 2, res.Status())

	res = mvh.Read(ap1, 2)
	require.Equal(t, 0, res.DepIdx())
	require.Equal(t, 2, res.Incarnation())
	require.Equal(t, valueFor(0, 2), res.Value().([]byte))
	require.Equal(t, 0, res.Status())
}

// TODO - handle panic
func TestLowerIncarnation(t *testing.T) {
	t.Parallel()

	ap1 := NewAddressKey(getCommonAddress(1))

	mvh := MakeMVHashMap()

	mvh.Write(ap1, Version{0, 2}, valueFor(0, 2))
	mvh.Read(ap1, 0)
	mvh.Write(ap1, Version{1, 2}, valueFor(1, 2))
	mvh.Write(ap1, Version{0, 5}, valueFor(0, 5))
	mvh.Write(ap1, Version{1, 5}, valueFor(1, 5))
}

func TestMarkEstimate(t *testing.T) {
	t.Parallel()

	ap1 := NewAddressKey(getCommonAddress(1))

	mvh := MakeMVHashMap()

	mvh.Write(ap1, Version{7, 2}, valueFor(7, 2))
	mvh.MarkEstimate(ap1, 7)
	mvh.Write(ap1, Version{7, 4}, valueFor(7, 4))
}

func TestMVHashMapBasics(t *testing.T) {
	t.Parallel()

	// memory locations
	ap1 := NewAddressKey(getCommonAddress(1))
	ap2 := NewAddressKey(getCommonAddress(2))
	ap3 := NewAddressKey(getCommonAddress(3))

	mvh := MakeMVHashMap()

	res := mvh.Read(ap1, 5)
	require.Equal(t, -1, res.depIdx)

	mvh.Write(ap1, Version{10, 1}, valueFor(10, 1))

	res = mvh.Read(ap1, 9)
	require.Equal(t, -1, res.depIdx, "reads that should go the DB return dependency -1")
	res = mvh.Read(ap1, 10)
	require.Equal(t, -1, res.depIdx, "Read returns entries from smaller txns, not txn 10")

	// Reads for a higher txn return the entry written by txn 10.
	res = mvh.Read(ap1, 15)
	require.Equal(t, 10, res.depIdx, "reads for a higher txn return the entry written by txn 10.")
	require.Equal(t, 1, res.incarnation)
	require.Equal(t, valueFor(10, 1), res.value)

	// More writes.
	mvh.Write(ap1, Version{12, 0}, valueFor(12, 0))
	mvh.Write(ap1, Version{8, 3}, valueFor(8, 3))

	// Verify reads.
	res = mvh.Read(ap1, 15)
	require.Equal(t, 12, res.depIdx)
	require.Equal(t, 0, res.incarnation)
	require.Equal(t, valueFor(12, 0), res.value)

	res = mvh.Read(ap1, 11)
	require.Equal(t, 10, res.depIdx)
	require.Equal(t, 1, res.incarnation)
	require.Equal(t, valueFor(10, 1), res.value)

	res = mvh.Read(ap1, 10)
	require.Equal(t, 8, res.depIdx)
	require.Equal(t, 3, res.incarnation)
	require.Equal(t, valueFor(8, 3), res.value)

	// Mark the entry written by 10 as an estimate.
	mvh.MarkEstimate(ap1, 10)

	res = mvh.Read(ap1, 11)
	require.Equal(t, 10, res.depIdx)
	require.Equal(t, -1, res.incarnation, "dep at tx 10 is now an estimate")

	// Delete the entry written by 10, write to a different ap.
	mvh.Delete(ap1, 10)
	mvh.Write(ap2, Version{10, 2}, valueFor(10, 2))

	// Read by txn 11 no longer observes entry from txn 10.
	res = mvh.Read(ap1, 11)
	require.Equal(t, 8, res.depIdx)
	require.Equal(t, 3, res.incarnation)
	require.Equal(t, valueFor(8, 3), res.value)

	// Reads, writes for ap2 and ap3.
	mvh.Write(ap2, Version{5, 0}, valueFor(5, 0))
	mvh.Write(ap3, Version{20, 4}, valueFor(20, 4))

	res = mvh.Read(ap2, 10)
	require.Equal(t, 5, res.depIdx)
	require.Equal(t, 0, res.incarnation)
	require.Equal(t, valueFor(5, 0), res.value)

	res = mvh.Read(ap3, 21)
	require.Equal(t, 20, res.depIdx)
	require.Equal(t, 4, res.incarnation)
	require.Equal(t, valueFor(20, 4), res.value)

	// Clear ap1 and ap3.
	mvh.Delete(ap1, 12)
	mvh.Delete(ap1, 8)
	mvh.Delete(ap3, 20)

	// Reads from ap1 and ap3 go to db.
	res = mvh.Read(ap1, 30)
	require.Equal(t, -1, res.depIdx)

	res = mvh.Read(ap3, 30)
	require.Equal(t, -1, res.depIdx)

	// No-op delete at ap2 - doesn't panic because ap2 does exist
	mvh.Delete(ap2, 11)

	// Read entry by txn 10 at ap2.
	res = mvh.Read(ap2, 15)
	require.Equal(t, 10, res.depIdx)
	require.Equal(t, 2, res.incarnation)
	require.Equal(t, valueFor(10, 2), res.value)
}

func BenchmarkWriteTimeSameLocationDifferentTxIdx(b *testing.B) {
	mvh2 := MakeMVHashMap()
	ap2 := NewAddressKey(getCommonAddress(2))

	randInts := []int{}
	for i := 0; i < b.N; i++ {
		randInts = append(randInts, rand.Intn(1000000000000000))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mvh2.Write(ap2, Version{randInts[i], 1}, valueFor(randInts[i], 1))
	}
}

func BenchmarkReadTimeSameLocationDifferentTxIdx(b *testing.B) {
	mvh2 := MakeMVHashMap()
	ap2 := NewAddressKey(getCommonAddress(2))
	txIdxSlice := []int{}

	for i := 0; i < b.N; i++ {
		txIdx := rand.Intn(1000000000000000)
		txIdxSlice = append(txIdxSlice, txIdx)
		mvh2.Write(ap2, Version{txIdx, 1}, valueFor(txIdx, 1))
	}

	b.ResetTimer()

	for _, value := range txIdxSlice {
		mvh2.Read(ap2, value)
	}
}

func TestTimeComplexity(t *testing.T) {
	t.Parallel()

	// for 1000000 read and write with no dependency at different memory location
	mvh1 := MakeMVHashMap()

	for i := 0; i < 1000000; i++ {
		ap1 := NewAddressKey(getCommonAddress(i))
		mvh1.Write(ap1, Version{i, 1}, valueFor(i, 1))
		mvh1.Read(ap1, i)
	}

	// for 1000000 read and write with dependency at same memory location
	mvh2 := MakeMVHashMap()
	ap2 := NewAddressKey(getCommonAddress(2))

	for i := 0; i < 1000000; i++ {
		mvh2.Write(ap2, Version{i, 1}, valueFor(i, 1))
		mvh2.Read(ap2, i)
	}
}

func TestWriteTimeSameLocationDifferentTxnIdx(t *testing.T) {
	t.Parallel()

	mvh1 := MakeMVHashMap()
	ap1 := NewAddressKey(getCommonAddress(1))

	for i := 0; i < 1000000; i++ {
		mvh1.Write(ap1, Version{i, 1}, valueFor(i, 1))
	}
}

func TestWriteTimeSameLocationSameTxnIdx(t *testing.T) {
	t.Parallel()

	mvh1 := MakeMVHashMap()
	ap1 := NewAddressKey(getCommonAddress(1))

	for i := 0; i < 1000000; i++ {
		mvh1.Write(ap1, Version{1, i}, valueFor(i, 1))
	}
}

func TestWriteTimeDifferentLocation(t *testing.T) {
	t.Parallel()

	mvh1 := MakeMVHashMap()

	for i := 0; i < 1000000; i++ {
		ap1 := NewAddressKey(getCommonAddress(i))
		mvh1.Write(ap1, Version{i, 1}, valueFor(i, 1))
	}
}

func TestReadTimeSameLocation(t *testing.T) {
	t.Parallel()

	mvh1 := MakeMVHashMap()
	ap1 := NewAddressKey(getCommonAddress(1))

	mvh1.Write(ap1, Version{1, 1}, valueFor(1, 1))

	for i := 0; i < 1000000; i++ {
		mvh1.Read(ap1, 2)
	}
}
