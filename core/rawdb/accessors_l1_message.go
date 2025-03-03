package rawdb

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"time"
	"unsafe"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/rlp"
)

var (
	// L1 message iterator metrics
	iteratorNextCalledCounter      = metrics.NewRegisteredCounter("rawdb/l1_message/iterator/next_called", nil)
	iteratorInnerNextCalledCounter = metrics.NewRegisteredCounter("rawdb/l1_message/iterator/inner_next_called", nil)
	iteratorLengthMismatchCounter  = metrics.NewRegisteredCounter("rawdb/l1_message/iterator/length_mismatch", nil)
	iteratorNextDurationTimer      = metrics.NewRegisteredTimer("rawdb/l1_message/iterator/next_time", nil)
	iteratorL1MessageSizeGauge     = metrics.NewRegisteredGauge("rawdb/l1_message/size", nil)
)

// WriteSyncedL1BlockNumber writes the highest synced L1 block number to the database.
func WriteSyncedL1BlockNumber(db ethdb.KeyValueWriter, L1BlockNumber uint64) {
	value := big.NewInt(0).SetUint64(L1BlockNumber).Bytes()

	if err := db.Put(syncedL1BlockNumberKey, value); err != nil {
		log.Crit("Failed to update synced L1 block number", "err", err)
	}
}

// ReadSyncedL1BlockNumber retrieves the highest synced L1 block number.
func ReadSyncedL1BlockNumber(db ethdb.Reader) *uint64 {
	data, err := db.Get(syncedL1BlockNumberKey)
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("Failed to read synced L1 block number from database", "err", err)
	}
	if len(data) == 0 {
		return nil
	}

	number := new(big.Int).SetBytes(data)
	if !number.IsUint64() {
		log.Crit("Unexpected synced L1 block number in database", "number", number)
	}

	value := number.Uint64()
	return &value
}

// WriteHighestSyncedQueueIndex writes the highest synced L1 message queue index to the database.
func WriteHighestSyncedQueueIndex(db ethdb.KeyValueWriter, queueIndex uint64) {
	value := big.NewInt(0).SetUint64(queueIndex).Bytes()

	if err := db.Put(highestSyncedQueueIndexKey, value); err != nil {
		log.Crit("Failed to update highest synced L1 message queue index", "err", err)
	}
}

// ReadHighestSyncedQueueIndex retrieves the highest synced L1 message queue index.
func ReadHighestSyncedQueueIndex(db ethdb.Reader) uint64 {
	data, err := db.Get(highestSyncedQueueIndexKey)
	if err != nil && isNotFoundErr(err) {
		return 0
	}
	if err != nil {
		log.Crit("Failed to read highest synced L1 message queue index from database", "err", err)
	}
	if len(data) == 0 {
		return 0
	}

	number := new(big.Int).SetBytes(data)
	if !number.IsUint64() {
		log.Crit("Unexpected highest synced L1 block number in database", "number", number)
	}

	return number.Uint64()
}

// WriteL1Message writes an L1 message to the database.
// We assume that L1 messages are written to DB following their queue index order.
func WriteL1Message(db ethdb.KeyValueWriter, l1Msg types.L1MessageTx) {
	bytes, err := rlp.EncodeToBytes(l1Msg)
	if err != nil {
		log.Crit("Failed to RLP encode L1 message", "err", err)
	}
	if err := db.Put(L1MessageKey(l1Msg.QueueIndex), bytes); err != nil {
		log.Crit("Failed to store L1 message", "err", err)
	}

	WriteHighestSyncedQueueIndex(db, l1Msg.QueueIndex)
}

// WriteL1Messages writes an array of L1 messages to the database.
// Note: pass a db of type `ethdb.Batcher` to batch writes in memory.
func WriteL1Messages(db ethdb.KeyValueWriter, l1Msgs []types.L1MessageTx) {
	for _, msg := range l1Msgs {
		WriteL1Message(db, msg)
	}
}

// ReadL1MessageRLP retrieves an L1 message in its raw RLP database encoding.
func ReadL1MessageRLP(db ethdb.Reader, queueIndex uint64) rlp.RawValue {
	data, err := db.Get(L1MessageKey(queueIndex))
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("Failed to load L1 message", "queueIndex", queueIndex, "err", err)
	}
	return data
}

// ReadL1Message retrieves the L1 message corresponding to the enqueue index.
func ReadL1Message(db ethdb.Reader, queueIndex uint64) *types.L1MessageTx {
	data := ReadL1MessageRLP(db, queueIndex)
	if len(data) == 0 {
		return nil
	}
	l1Msg := new(types.L1MessageTx)
	if err := rlp.Decode(bytes.NewReader(data), l1Msg); err != nil {
		log.Crit("Invalid L1 message RLP", "queueIndex", queueIndex, "data", data, "err", err)
	}
	return l1Msg
}

// L1MessageIterator is a wrapper around ethdb.Iterator that
// allows us to iterate over L1 messages in the database. It
// implements an interface similar to ethdb.Iterator.
type L1MessageIterator struct {
	inner         ethdb.Iterator
	keyLength     int
	maxQueueIndex uint64
}

// iterateL1MessagesFrom creates an L1MessageIterator that iterates over
// all L1 message in the database starting at the provided enqueue index.
func iterateL1MessagesFrom(db ethdb.Database, fromQueueIndex uint64) L1MessageIterator {
	start := encodeBigEndian(fromQueueIndex)
	it := db.NewIterator(l1MessagePrefix, start)
	keyLength := len(l1MessagePrefix) + 8
	maxQueueIndex := ReadHighestSyncedQueueIndex(db)

	return L1MessageIterator{
		inner:         it,
		keyLength:     keyLength,
		maxQueueIndex: maxQueueIndex,
	}
}

// Next moves the iterator to the next key/value pair.
// It returns false when the iterator is exhausted.
// TODO: Consider reading items in batches.
func (it *L1MessageIterator) Next() bool {
	iteratorNextCalledCounter.Inc(1)

	defer func(t0 time.Time) {
		iteratorNextDurationTimer.Update(time.Since(t0))
	}(time.Now())

	for it.inner.Next() {
		iteratorInnerNextCalledCounter.Inc(1)

		key := it.inner.Key()
		if len(key) == it.keyLength {
			return true
		} else {
			iteratorLengthMismatchCounter.Inc(1)
		}
	}
	return false
}

// QueueIndex returns the enqueue index of the current L1 message.
func (it *L1MessageIterator) QueueIndex() uint64 {
	key := it.inner.Key()
	raw := key[len(l1MessagePrefix) : len(l1MessagePrefix)+8]
	queueIndex := binary.BigEndian.Uint64(raw)
	return queueIndex
}

// L1Message returns the current L1 message.
func (it *L1MessageIterator) L1Message() types.L1MessageTx {
	data := it.inner.Value()
	l1Msg := types.L1MessageTx{}
	if err := rlp.DecodeBytes(data, &l1Msg); err != nil {
		log.Crit("Invalid L1 message RLP", "data", data, "err", err)
	}
	return l1Msg
}

// Release releases the associated resources.
func (it *L1MessageIterator) Release() {
	it.inner.Release()
}

// Error returns any accumulated error.
// Exhausting all the key/value pairs is not considered to be an error.
func (it *L1MessageIterator) Error() error {
	return it.inner.Error()
}

// L1MessageV1Iterator is a wrapper around L1MessageIterator that allows us to iterate over L1 messages V1.
type L1MessageV1Iterator struct {
	db           ethdb.Database
	v2StartIndex *uint64
	L1MessageIterator
}

// IterateL1MessagesV1From yields a L1MessageV1Iterator with following behavior:
// - If fromQueueIndex >= L1MessageV2StartIndex: yield 0 messages.
// - Otherwise, simply yield all messages (guaranteed to be V1) starting from `fromQueueIndex` until `L1MessageV2StartIndex`.
func IterateL1MessagesV1From(db ethdb.Database, fromQueueIndex uint64) L1MessageV1Iterator {
	return L1MessageV1Iterator{
		db:                db,
		v2StartIndex:      ReadL1MessageV2StartIndex(db),
		L1MessageIterator: iterateL1MessagesFrom(db, fromQueueIndex),
	}
}

func (it *L1MessageV1Iterator) Next() bool {
	for it.L1MessageIterator.Next() {
		// L1MessageV2StartIndex is the first queue index of L1 messages that are from L1MessageQueueV2.
		// Therefore, we stop reading L1 messages V1 when we reach this index.
		// We need to check in every iteration if not yet set as the start index can be set in the meantime when we are reading L1 messages.
		if it.v2StartIndex == nil {
			it.v2StartIndex = ReadL1MessageV2StartIndex(it.db)
		}

		if it.v2StartIndex != nil && it.QueueIndex() >= *it.v2StartIndex {
			return false
		}
		return true
	}
	return false
}

// L1MessageV2Iterator is a wrapper around L1MessageIterator that allows us to iterate over L1 messages V2.
type L1MessageV2Iterator struct {
	v2StartIndex *uint64
	L1MessageIterator
}

// IterateL1MessagesV2From yields a L1MessageV2Iterator with following behavior:
// - If fromQueueIndex < v2StartIndex: yield 0 messages.
// - Otherwise, simply yield all messages (guaranteed to be v2) starting from `fromQueueIndex`.
func IterateL1MessagesV2From(db ethdb.Database, fromQueueIndex uint64) L1MessageV2Iterator {
	v2StartIndex := ReadL1MessageV2StartIndex(db)

	return L1MessageV2Iterator{
		v2StartIndex:      v2StartIndex,
		L1MessageIterator: iterateL1MessagesFrom(db, fromQueueIndex),
	}
}

func (it *L1MessageV2Iterator) Next() bool {
	if it.v2StartIndex == nil {
		return false
	}

	return it.L1MessageIterator.Next() && it.QueueIndex() >= *it.v2StartIndex
}

// ReadL1MessagesV1From retrieves up to `maxCount` L1 messages V1 starting at `startIndex`.
// If startIndex is >= L1MessageV2StartIndex, this function returns an empty slice.
func ReadL1MessagesV1From(db ethdb.Database, startIndex, maxCount uint64) []types.L1MessageTx {
	msgs := make([]types.L1MessageTx, 0, maxCount)
	it := IterateL1MessagesV1From(db, startIndex)
	defer it.Release()

	index := startIndex
	count := maxCount

	for count > 0 && it.Next() {
		msg := it.L1Message()

		// sanity check
		if msg.QueueIndex != index {
			log.Crit(
				"Unexpected QueueIndex in ReadL1MessagesV1From",
				"expected", index,
				"got", msg.QueueIndex,
				"startIndex", startIndex,
				"maxCount", maxCount,
			)
		}

		msgs = append(msgs, msg)
		index += 1
		count -= 1

		iteratorL1MessageSizeGauge.Update(int64(unsafe.Sizeof(msg) + uintptr(cap(msg.Data))))

		if msg.QueueIndex == it.maxQueueIndex {
			break
		}
	}

	if err := it.Error(); err != nil {
		log.Crit("Failed to read L1 messages", "err", err)
	}

	return msgs
}

// ReadL1MessagesV2From retrieves up to `maxCount` L1 messages V2 starting at `startIndex`.
// If startIndex is smaller than L1MessageV2StartIndex, this function returns an empty slice.
func ReadL1MessagesV2From(db ethdb.Database, startIndex, maxCount uint64) []types.L1MessageTx {
	msgs := make([]types.L1MessageTx, 0, maxCount)

	it := IterateL1MessagesV2From(db, startIndex)
	defer it.Release()

	index := startIndex
	count := maxCount

	for count > 0 && it.Next() {
		msg := it.L1Message()

		// sanity check
		if msg.QueueIndex != index {
			log.Crit(
				"Unexpected QueueIndex in ReadL1MessagesV2From",
				"expected", index,
				"got", msg.QueueIndex,
				"startIndex", startIndex,
				"maxCount", maxCount,
			)
		}

		msgs = append(msgs, msg)
		index += 1
		count -= 1

		iteratorL1MessageSizeGauge.Update(int64(unsafe.Sizeof(msg) + uintptr(cap(msg.Data))))

		if msg.QueueIndex == it.maxQueueIndex {
			break
		}
	}

	if err := it.Error(); err != nil {
		log.Crit("Failed to read L1 messages", "err", err)
	}

	return msgs
}

// WriteFirstQueueIndexNotInL2Block writes the queue index of the first message
// that is NOT included in the ledger up to and including the provided L2 block.
// The L2 block is identified by its block hash. If the L2 block contains zero
// L1 messages, this value MUST equal its parent's value.
func WriteFirstQueueIndexNotInL2Block(db ethdb.KeyValueWriter, l2BlockHash common.Hash, queueIndex uint64) {
	if err := db.Put(FirstQueueIndexNotInL2BlockKey(l2BlockHash), encodeBigEndian(queueIndex)); err != nil {
		log.Crit("Failed to store first L1 message not in L2 block", "l2BlockHash", l2BlockHash, "err", err)
	}
}

// ReadFirstQueueIndexNotInL2Block retrieves the queue index of the first message
// that is NOT included in the ledger up to and including the provided L2 block.
func ReadFirstQueueIndexNotInL2Block(db ethdb.Reader, l2BlockHash common.Hash) *uint64 {
	data, err := db.Get(FirstQueueIndexNotInL2BlockKey(l2BlockHash))
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("Failed to read first L1 message not in L2 block from database", "l2BlockHash", l2BlockHash, "err", err)
	}
	if len(data) == 0 {
		return nil
	}
	queueIndex := binary.BigEndian.Uint64(data)
	return &queueIndex
}

// WriteL1MessageV2StartIndex writes the start index of L1 messages that are from L1MessageQueueV2.
func WriteL1MessageV2StartIndex(db ethdb.KeyValueWriter, queueIndex uint64) {
	// Write with binary.BigEndian.PutUint64 to ensure that 0 values are written as 8 bytes.
	// big.NewInt(0).SetUint64(l1BlockNumber).Bytes() would write 0 as empty slice which leads to problems when reading
	// the value as non-existent and 0 are not distinguishable.
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], queueIndex)

	if err := db.Put(l1MessageV2StartIndexKey, buf[:]); err != nil {
		log.Crit("Failed to update L1MessageV2 start index", "err", err)
	}
}

// ReadL1MessageV2StartIndex retrieves the start index of L1 messages that are from L1MessageQueueV2.
func ReadL1MessageV2StartIndex(db ethdb.Reader) *uint64 {
	data, err := db.Get(l1MessageV2StartIndexKey)
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("Failed to read L1MessageV2 start index from database", "err", err)
	}
	if len(data) == 0 {
		return nil
	}
	if len(data) != 8 {
		return nil
	}
	res := binary.BigEndian.Uint64(data)

	return &res
}

// WriteL1MessageV2FirstL1BlockNumber writes the first synced L1 block number for L1MessageV2.
func WriteL1MessageV2FirstL1BlockNumber(db ethdb.KeyValueWriter, l1BlockNumber uint64) {
	value := big.NewInt(0).SetUint64(l1BlockNumber).Bytes()

	if err := db.Put(l1MessageV2FirstL1BlockNumberKey, value); err != nil {
		log.Crit("Failed to update L1MessageV2 start index", "err", err)
	}
}

// ReadL1MessageV2FirstL1BlockNumber retrieves the first synced L1 block number for L1MessageV2.
func ReadL1MessageV2FirstL1BlockNumber(db ethdb.Reader) *uint64 {
	data, err := db.Get(l1MessageV2FirstL1BlockNumberKey)
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("Failed to read L1MessageV2 first L1 block number from database", "err", err)
	}
	if len(data) == 0 {
		return nil
	}

	number := new(big.Int).SetBytes(data)
	if !number.IsUint64() {
		log.Crit("Unexpected number for L1MessageV2 first L1 block number", "number", number)
	}

	res := number.Uint64()
	return &res
}
