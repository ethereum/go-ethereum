package rawdb

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math/big"

	leveldb "github.com/syndtr/goleveldb/leveldb/errors"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rlp"
)

func isNotFoundErr(err error) bool {
	return errors.Is(err, leveldb.ErrNotFound) || errors.Is(err, memorydb.ErrMemorydbNotFound)
}

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

// WriteL1Message writes an L1 message to the database.
func WriteL1Message(db ethdb.KeyValueWriter, l1Msg types.L1MessageTx) {
	bytes, err := rlp.EncodeToBytes(l1Msg)
	if err != nil {
		log.Crit("Failed to RLP encode L1 message", "err", err)
	}
	if err := db.Put(L1MessageKey(l1Msg.QueueIndex), bytes); err != nil {
		log.Crit("Failed to store L1 message", "err", err)
	}
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
	inner     ethdb.Iterator
	keyLength int
}

// IterateL1MessagesFrom creates an L1MessageIterator that iterates over
// all L1 message in the database starting at the provided enqueue index.
func IterateL1MessagesFrom(db ethdb.Iteratee, fromQueueIndex uint64) L1MessageIterator {
	start := encodeQueueIndex(fromQueueIndex)
	it := db.NewIterator(l1MessagePrefix, start)
	keyLength := len(l1MessagePrefix) + 8

	return L1MessageIterator{
		inner:     it,
		keyLength: keyLength,
	}
}

// Next moves the iterator to the next key/value pair.
// It returns false when the iterator is exhausted.
// TODO: Consider reading items in batches.
func (it *L1MessageIterator) Next() bool {
	for it.inner.Next() {
		key := it.inner.Key()
		if len(key) == it.keyLength {
			return true
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

// ReadL1MessagesFrom retrieves up to `maxCount` L1 messages starting at `startIndex`.
func ReadL1MessagesFrom(db ethdb.Iteratee, startIndex, maxCount uint64) []types.L1MessageTx {
	msgs := make([]types.L1MessageTx, 0, maxCount)
	it := IterateL1MessagesFrom(db, startIndex)
	defer it.Release()

	index := startIndex
	count := maxCount

	for count > 0 && it.Next() {
		msg := it.L1Message()

		// sanity check
		if msg.QueueIndex != index {
			log.Crit(
				"Unexpected QueueIndex in ReadL1MessagesFrom",
				"expected", index,
				"got", msg.QueueIndex,
				"startIndex", startIndex,
				"maxCount", maxCount,
			)
		}

		msgs = append(msgs, msg)
		index += 1
		count -= 1
	}

	return msgs
}

// WriteFirstQueueIndexNotInL2Block writes the queue index of the first message
// that is NOT included in the ledger up to and including the provided L2 block.
// The L2 block is identified by its block hash. If the L2 block contains zero
// L1 messages, this value MUST equal its parent's value.
func WriteFirstQueueIndexNotInL2Block(db ethdb.KeyValueWriter, l2BlockHash common.Hash, queueIndex uint64) {
	if err := db.Put(FirstQueueIndexNotInL2BlockKey(l2BlockHash), encodeQueueIndex(queueIndex)); err != nil {
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
