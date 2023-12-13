package storage

import (
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

const nodeDataDir = "./"

func clear() {
	os.Remove(fmt.Sprintf("%s%s", nodeDataDir, sqliteName))
}

func genBytes(length int) []byte {
	res := make([]byte, length)
	for i := 0; i < length; i++ {
		res[i] = byte(i)
	}
	return res
}

func TestBasicStorage(t *testing.T) {
	zeroNodeId := uint256.NewInt(0).Bytes32()
	storage, err := NewContentStorage(math.MaxUint32, enode.ID(zeroNodeId), nodeDataDir)
	assert.NoError(t, err)
	defer clear()
	defer storage.Close()

	contentId := []byte("test")
	content := []byte("value")

	_, err = storage.Get(contentId)
	assert.Equal(t, ErrContentNotFound, err)

	pt := storage.Put(contentId, content)
	assert.NoError(t, pt.Err())

	val, err := storage.Get(contentId)
	assert.NoError(t, err)
	assert.Equal(t, content, val)

	count, err := storage.ContentCount()
	assert.NoError(t, err)
	assert.Equal(t, count, uint64(1))

	size, err := storage.Size()
	assert.NoError(t, err)
	assert.True(t, size > 0)

	unusedSize, err := storage.UnusedSize()
	assert.NoError(t, err)

	usedSize, err := storage.UsedSize()
	assert.NoError(t, err)
	assert.True(t, usedSize == size-unusedSize)
}

func TestDBSize(t *testing.T) {
	zeroNodeId := uint256.NewInt(0).Bytes32()
	storage, err := NewContentStorage(math.MaxUint32, enode.ID(zeroNodeId), nodeDataDir)
	assert.NoError(t, err)
	defer clear()
	defer storage.Close()

	numBytes := 10000

	size1, err := storage.Size()
	assert.NoError(t, err)
	putResult := storage.Put(uint256.NewInt(1).Bytes(), genBytes(numBytes))
	assert.Nil(t, putResult.Err())

	size2, err := storage.Size()
	assert.NoError(t, err)
	putResult = storage.Put(uint256.NewInt(2).Bytes(), genBytes(numBytes))
	assert.NoError(t, putResult.Err())

	size3, err := storage.Size()
	assert.NoError(t, err)
	putResult = storage.Put(uint256.NewInt(2).Bytes(), genBytes(numBytes))
	assert.NoError(t, putResult.Err())

	size4, err := storage.Size()
	assert.NoError(t, err)
	usedSize, err := storage.UsedSize()
	assert.NoError(t, err)

	assert.True(t, size2 > size1)
	assert.True(t, size3 > size2)
	assert.True(t, size4 == size3)
	assert.True(t, usedSize == size4)

	err = storage.del(uint256.NewInt(2).Bytes())
	assert.NoError(t, err)
	err = storage.del(uint256.NewInt(1).Bytes())
	assert.NoError(t, err)

	usedSize1, err := storage.UsedSize()
	assert.NoError(t, err)
	size5, err := storage.Size()
	assert.NoError(t, err)

	assert.True(t, size4 == size5)
	assert.True(t, usedSize1 < size5)

	err = storage.ReclaimSpace()
	assert.NoError(t, err)

	usedSize2, err := storage.UsedSize()
	assert.NoError(t, err)
	size6, err := storage.Size()
	assert.NoError(t, err)

	assert.Equal(t, size1, size6)
	assert.Equal(t, usedSize2, size6)
}

func TestDBPruning(t *testing.T) {
	storageCapacity := uint64(100_000)

	zeroNodeId := uint256.NewInt(0).Bytes32()
	storage, err := NewContentStorage(storageCapacity, enode.ID(zeroNodeId), nodeDataDir)
	assert.NoError(t, err)
	defer clear()
	defer storage.Close()

	furthestElement := uint256.NewInt(40)
	secondFurthest := uint256.NewInt(30)
	thirdFurthest := uint256.NewInt(20)

	numBytes := 10_000
	// test with private put method
	pt1 := storage.Put(uint256.NewInt(1).Bytes(), genBytes(numBytes))
	assert.NoError(t, pt1.Err())
	pt2 := storage.Put(thirdFurthest.Bytes(), genBytes(numBytes))
	assert.NoError(t, pt2.Err())
	pt3 := storage.Put(uint256.NewInt(3).Bytes(), genBytes(numBytes))
	assert.NoError(t, pt3.Err())
	pt4 := storage.Put(uint256.NewInt(10).Bytes(), genBytes(numBytes))
	assert.NoError(t, pt4.Err())
	pt5 := storage.Put(uint256.NewInt(5).Bytes(), genBytes(numBytes))
	assert.NoError(t, pt5.Err())
	pt6 := storage.Put(uint256.NewInt(11).Bytes(), genBytes(numBytes))
	assert.NoError(t, pt6.Err())
	pt7 := storage.Put(furthestElement.Bytes(), genBytes(4000))
	assert.NoError(t, pt7.Err())
	pt8 := storage.Put(secondFurthest.Bytes(), genBytes(3000))
	assert.NoError(t, pt8.Err())
	pt9 := storage.Put(uint256.NewInt(2).Bytes(), genBytes(numBytes))
	assert.NoError(t, pt9.Err())

	res, _ := storage.GetLargestDistance()

	assert.Equal(t, res, uint256.NewInt(40))
	pt10 := storage.Put(uint256.NewInt(4).Bytes(), genBytes(12000))
	assert.NoError(t, pt10.Err())

	assert.False(t, pt1.Pruned())
	assert.False(t, pt2.Pruned())
	assert.False(t, pt3.Pruned())
	assert.False(t, pt4.Pruned())
	assert.False(t, pt5.Pruned())
	assert.False(t, pt6.Pruned())
	assert.False(t, pt7.Pruned())
	assert.False(t, pt8.Pruned())
	assert.False(t, pt9.Pruned())
	assert.True(t, pt10.Pruned())

	assert.Equal(t, pt10.PrunedCount(), 2)
	usedSize, err := storage.UsedSize()
	assert.NoError(t, err)
	assert.True(t, usedSize < storage.storageCapacityInBytes)

	_, err = storage.Get(furthestElement.Bytes())
	assert.Equal(t, ErrContentNotFound, err)

	_, err = storage.Get(secondFurthest.Bytes())
	assert.Equal(t, ErrContentNotFound, err)

	val, err := storage.Get(thirdFurthest.Bytes())
	assert.NoError(t, err)
	assert.NotNil(t, val)
}

func TestGetLargestDistance(t *testing.T) {
	storageCapacity := uint64(100_000)

	zeroNodeId := uint256.NewInt(0).Bytes32()
	storage, err := NewContentStorage(storageCapacity, enode.ID(zeroNodeId), nodeDataDir)
	assert.NoError(t, err)
	defer clear()
	defer storage.Close()

	furthestElement := uint256.NewInt(40)
	secondFurthest := uint256.NewInt(30)

	pt7 := storage.Put(furthestElement.Bytes(), genBytes(2000))
	assert.NoError(t, pt7.Err())

	val, err := storage.Get(furthestElement.Bytes())
	assert.NoError(t, err)
	assert.NotNil(t, val)
	pt8 := storage.Put(secondFurthest.Bytes(), genBytes(2000))
	assert.NoError(t, pt8.Err())
	res, err := storage.GetLargestDistance()
	assert.NoError(t, err)
	assert.Equal(t, furthestElement, res)
}

func TestSimpleForcePruning(t *testing.T) {
	storageCapacity := uint64(100_000)

	zeroNodeId := uint256.NewInt(0).Bytes32()
	storage, err := NewContentStorage(storageCapacity, enode.ID(zeroNodeId), nodeDataDir)
	assert.NoError(t, err)
	defer clear()
	defer storage.Close()

	furthestElement := uint256.NewInt(40)
	secondFurthest := uint256.NewInt(30)
	third := uint256.NewInt(10)

	pt1 := storage.Put(furthestElement.Bytes(), genBytes(2000))
	assert.NoError(t, pt1.Err())

	pt2 := storage.Put(secondFurthest.Bytes(), genBytes(2000))
	assert.NoError(t, pt2.Err())

	pt3 := storage.Put(third.Bytes(), genBytes(2000))
	assert.NoError(t, pt3.Err())
	res, err := storage.GetLargestDistance()
	assert.NoError(t, err)
	assert.Equal(t, furthestElement, res)

	err = storage.ForcePrune(uint256.NewInt(20))
	assert.NoError(t, err)

	_, err = storage.Get(furthestElement.Bytes())
	assert.Equal(t, ErrContentNotFound, err)

	_, err = storage.Get(secondFurthest.Bytes())
	assert.Equal(t, ErrContentNotFound, err)

	_, err = storage.Get(third.Bytes())
	assert.NoError(t, err)
}

func TestForcePruning(t *testing.T) {
	const startCap = uint64(14_159_872)
	const endCapacity = uint64(5000_000)
	const amountOfItems = 10_000

	maxUint256 := uint256.MustFromHex("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

	nodeId := uint256.MustFromHex("0x30994892f3e4889d99deb5340050510d1842778acc7a7948adffa475fed51d6e").Bytes()
	content := genBytes(1000)

	storage, err := NewContentStorage(startCap, enode.ID(nodeId), nodeDataDir)
	assert.NoError(t, err)
	defer clear()
	defer storage.Close()

	increment := uint256.NewInt(0).Div(maxUint256, uint256.NewInt(amountOfItems))
	remainder := uint256.NewInt(0).Mod(maxUint256, uint256.NewInt(amountOfItems))

	id := uint256.NewInt(0)
	putCount := 0
	// id < maxUint256 - remainder
	for id.Cmp(uint256.NewInt(0).Sub(maxUint256, remainder)) == -1 {
		res := storage.Put(id.Bytes(), content)
		assert.NoError(t, res.Err())
		id = id.Add(id, increment)
		putCount++
	}

	storage.storageCapacityInBytes = endCapacity

	oldDistance, err := storage.GetLargestDistance()
	assert.NoError(t, err)
	newDistance, err := storage.EstimateNewRadius(oldDistance)
	assert.NoError(t, err)
	assert.NotEqual(t, oldDistance.Cmp(newDistance), -1)
	err = storage.ForcePrune(newDistance)
	assert.NoError(t, err)

	var total int64
	err = storage.sqliteDB.QueryRow("SELECT count(*) FROM kvstore where greater(xor(key, (?1)), (?2)) = 1", storage.nodeId[:], newDistance.Bytes()).Scan(&total)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
}
