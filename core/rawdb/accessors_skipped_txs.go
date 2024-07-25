package rawdb

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math/big"
	"sync"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rlp"
)

// mutex used to avoid concurrent updates of NumSkippedTransactions
var mu sync.Mutex

// writeNumSkippedTransactions writes the number of skipped transactions to the database.
func writeNumSkippedTransactions(db ethdb.KeyValueWriter, numSkipped uint64) {
	value := big.NewInt(0).SetUint64(numSkipped).Bytes()

	if err := db.Put(numSkippedTransactionsKey, value); err != nil {
		log.Crit("Failed to update the number of skipped transactions", "err", err)
	}
}

// ReadNumSkippedTransactions retrieves the number of skipped transactions.
func ReadNumSkippedTransactions(db ethdb.Reader) uint64 {
	data, err := db.Get(numSkippedTransactionsKey)
	if err != nil && isNotFoundErr(err) {
		return 0
	}
	if err != nil {
		log.Crit("Failed to read number of skipped transactions from database", "err", err)
	}
	if len(data) == 0 {
		return 0
	}

	number := new(big.Int).SetBytes(data)
	if !number.IsUint64() {
		log.Crit("Unexpected number of skipped transactions in database", "number", number)
	}
	return number.Uint64()
}

// SkippedTransaction stores the transaction object, along with the skip reason and block context.
type SkippedTransaction struct {
	// Tx is the skipped transaction.
	// We store the tx itself because otherwise geth will discard it after skipping.
	Tx *types.Transaction

	// Reason is the skip reason.
	Reason string

	// BlockNumber is the number of the block in which this transaction was skipped.
	BlockNumber uint64

	// BlockHash is the hash of the block in which this transaction was skipped or nil.
	BlockHash *common.Hash
}

// SkippedTransactionV2 stores the SkippedTransaction object along with serialized traces.
type SkippedTransactionV2 struct {
	// Tx is the skipped transaction.
	// We store the tx itself otherwise geth will discard it after skipping.
	Tx *types.Transaction

	// Traces is the serialized wrapped traces of the skipped transaction.
	// We only store it when `MinerStoreSkippedTxTracesFlag` is enabled, so it might be empty.
	// Note that we do not directly utilize `*types.BlockTrace` due to the fact that
	// types.BlockTrace.StorageTrace.Proofs is of type `map[string][]hexutil.Bytes`, which is not RLP-serializable.
	TracesBytes []byte

	// Reason is the skip reason.
	Reason string

	// BlockNumber is the number of the block in which this transaction was skipped.
	BlockNumber uint64

	// BlockHash is the hash of the block in which this transaction was skipped or nil.
	BlockHash *common.Hash
}

// writeSkippedTransaction writes a skipped transaction to the database.
func writeSkippedTransaction(db ethdb.KeyValueWriter, tx *types.Transaction, traces *types.BlockTrace, reason string, blockNumber uint64, blockHash *common.Hash) {
	var err error
	// workaround: RLP decoding fails if this is nil
	if blockHash == nil {
		blockHash = &common.Hash{}
	}
	stx := SkippedTransactionV2{Tx: tx, Reason: reason, BlockNumber: blockNumber, BlockHash: blockHash}
	if traces != nil {
		if stx.TracesBytes, err = json.Marshal(traces); err != nil {
			log.Crit("Failed to json marshal skipped transaction", "hash", tx.Hash().String(), "err", err)
		}
	}
	bytes, err := rlp.EncodeToBytes(stx)
	if err != nil {
		log.Crit("Failed to RLP encode skipped transaction", "hash", tx.Hash().String(), "err", err)
	}
	if err := db.Put(SkippedTransactionKey(tx.Hash()), bytes); err != nil {
		log.Crit("Failed to store skipped transaction", "hash", tx.Hash().String(), "err", err)
	}
}

// writeSkippedTransactionV1 is the old version of writeSkippedTransaction, we keep it for testing compatibility purpose.
func writeSkippedTransactionV1(db ethdb.KeyValueWriter, tx *types.Transaction, reason string, blockNumber uint64, blockHash *common.Hash) {
	// workaround: RLP decoding fails if this is nil
	if blockHash == nil {
		blockHash = &common.Hash{}
	}
	stx := SkippedTransaction{Tx: tx, Reason: reason, BlockNumber: blockNumber, BlockHash: blockHash}
	bytes, err := rlp.EncodeToBytes(stx)
	if err != nil {
		log.Crit("Failed to RLP encode skipped transaction", "hash", tx.Hash().String(), "err", err)
	}
	if err := db.Put(SkippedTransactionKey(tx.Hash()), bytes); err != nil {
		log.Crit("Failed to store skipped transaction", "hash", tx.Hash().String(), "err", err)
	}
}

// readSkippedTransactionRLP retrieves a skipped transaction in its raw RLP database encoding.
func readSkippedTransactionRLP(db ethdb.Reader, txHash common.Hash) rlp.RawValue {
	data, err := db.Get(SkippedTransactionKey(txHash))
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("Failed to load skipped transaction", "hash", txHash.String(), "err", err)
	}
	return data
}

// ReadSkippedTransaction retrieves a skipped transaction by its hash, along with its skipped reason.
func ReadSkippedTransaction(db ethdb.Reader, txHash common.Hash) *SkippedTransactionV2 {
	data := readSkippedTransactionRLP(db, txHash)
	if len(data) == 0 {
		return nil
	}
	var stxV2 SkippedTransactionV2
	var stx SkippedTransaction
	if err := rlp.Decode(bytes.NewReader(data), &stxV2); err != nil {
		if err := rlp.Decode(bytes.NewReader(data), &stx); err != nil {
			log.Crit("Invalid skipped transaction RLP", "hash", txHash.String(), "data", data, "err", err)
		}
		stxV2.Tx = stx.Tx
		stxV2.Reason = stx.Reason
		stxV2.BlockNumber = stx.BlockNumber
		stxV2.BlockHash = stx.BlockHash
	}

	if stxV2.BlockHash != nil && *stxV2.BlockHash == (common.Hash{}) {
		stxV2.BlockHash = nil
	}
	return &stxV2
}

// writeSkippedTransactionHash writes the hash of a skipped transaction to the database.
func writeSkippedTransactionHash(db ethdb.KeyValueWriter, index uint64, txHash common.Hash) {
	if err := db.Put(SkippedTransactionHashKey(index), txHash[:]); err != nil {
		log.Crit("Failed to store skipped transaction hash", "index", index, "hash", txHash.String(), "err", err)
	}
}

// ReadSkippedTransactionHash retrieves the hash of a skipped transaction by its index.
func ReadSkippedTransactionHash(db ethdb.Reader, index uint64) *common.Hash {
	data, err := db.Get(SkippedTransactionHashKey(index))
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("Failed to load skipped transaction hash", "index", index, "err", err)
	}
	hash := common.BytesToHash(data)
	return &hash
}

// WriteSkippedTransaction writes a skipped transaction to the database and also updates the count and lookup index.
// Note: The lookup index and count will include duplicates if there are chain reorgs.
func WriteSkippedTransaction(db ethdb.Database, tx *types.Transaction, traces *types.BlockTrace, reason string, blockNumber uint64, blockHash *common.Hash) {
	// this method is not accessed concurrently, but just to be sure...
	mu.Lock()
	defer mu.Unlock()

	index := ReadNumSkippedTransactions(db)

	// update in a batch
	batch := db.NewBatch()
	writeSkippedTransaction(batch, tx, traces, reason, blockNumber, blockHash)
	writeSkippedTransactionHash(batch, index, tx.Hash())
	writeNumSkippedTransactions(batch, index+1)

	// write to DB
	if err := batch.Write(); err != nil {
		log.Crit("Failed to store skipped transaction", "hash", tx.Hash().String(), "err", err)
	}
}

// SkippedTransactionIterator is a wrapper around ethdb.Iterator that
// allows us to iterate over skipped transaction hashes in the database.
// It implements an interface similar to ethdb.Iterator.
type SkippedTransactionIterator struct {
	inner     ethdb.Iterator
	db        ethdb.Reader
	keyLength int
}

// IterateSkippedTransactionsFrom creates a SkippedTransactionIterator that iterates
// over all skipped transaction hashes in the database starting at the provided index.
func IterateSkippedTransactionsFrom(db ethdb.Database, index uint64) SkippedTransactionIterator {
	start := encodeBigEndian(index)
	it := db.NewIterator(skippedTransactionHashPrefix, start)
	keyLength := len(skippedTransactionHashPrefix) + 8

	return SkippedTransactionIterator{
		inner:     it,
		db:        db,
		keyLength: keyLength,
	}
}

// Next moves the iterator to the next key/value pair.
// It returns false when the iterator is exhausted.
// TODO: Consider reading items in batches.
func (it *SkippedTransactionIterator) Next() bool {
	for it.inner.Next() {
		key := it.inner.Key()
		if len(key) == it.keyLength {
			return true
		}
	}
	return false
}

// Index returns the index of the current skipped transaction hash.
func (it *SkippedTransactionIterator) Index() uint64 {
	key := it.inner.Key()
	raw := key[len(skippedTransactionHashPrefix) : len(skippedTransactionHashPrefix)+8]
	index := binary.BigEndian.Uint64(raw)
	return index
}

// TransactionHash returns the current skipped transaction hash.
func (it *SkippedTransactionIterator) TransactionHash() common.Hash {
	data := it.inner.Value()
	return common.BytesToHash(data)
}

// Release releases the associated resources.
func (it *SkippedTransactionIterator) Release() {
	it.inner.Release()
}
