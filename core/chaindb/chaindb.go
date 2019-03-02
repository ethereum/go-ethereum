// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.
package chaindb

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
)

// ChainDB is the blockchain data storage layer. It provides the ability
// to retrieve, store, and delete the various constructs (i.e. headers,
// receipts) and metadata (i.e. header hash mappings, transaction locations).
type ChainDB struct {
	writer

	db ethdb.Database
}

// Batch consolidates writes and deletions to the ChainDB it originated
// from. The changes will be committed when Write is called.
type Batch struct {
	writer

	batch ethdb.Batch
}

// Consolidates shared writing and deleting code used by both ChainDB and Batch.
// Note that this considers deleting to be a form of writing unlike ethdb.
type writer struct {
	db ethdbPutterAndDeleter
}

// Composite type to enable both putting and deleting.
type ethdbPutterAndDeleter interface {
	ethdb.Putter
	ethdb.Deleter
}

// Wrap creates a new ChainDB that uses the specified database as its
// underlying data store.
func Wrap(db ethdb.Database) *ChainDB {
	return &ChainDB{writer{db}, db}
}

// Returns a Batch associated with the ChainDB.
func (c *ChainDB) NewBatch() *Batch {
	return newBatch(c.db.NewBatch())
}

func newBatch(batch ethdb.Batch) *Batch {
	return &Batch{writer{batch}, batch}
}

// Write commits the changes within the batch to its associated ChainDB.
func (b *Batch) Write() error {
	return b.batch.Write()
}

// Reset drops all pending changes contained in the batch.
func (b *Batch) Reset() {
	b.batch.Reset()
}

// Value size returns the number of changes pending in the batch.
func (b *Batch) ValueSize() int {
	return b.batch.ValueSize()
}

// ReadHeadHeaderHash retrieves the current canonical head header hash.
func (c *ChainDB) ReadHeadHeaderHash() common.Hash {
	return rawdb.ReadHeadHeaderHash(c.db)
}

// WriteHeadHeaderHash stores the current canonical head header hash.
func (w *writer) WriteHeadHeaderHash(hash common.Hash) {
	rawdb.WriteHeadHeaderHash(w.db, hash)
}

// ReadHeadBlockHash retrieves the current canonical head block hash.
//
// Note that this is different from the canonical head header hash as
// this corresponds to all block data and not just the header.
func (c *ChainDB) ReadHeadBlockHash() common.Hash {
	return rawdb.ReadHeadBlockHash(c.db)
}

// WriteHeadBlockHash stores the current fast-sync head block hash.
func (w *writer) WriteHeadBlockHash(hash common.Hash) {
	rawdb.WriteHeadBlockHash(w.db, hash)
}

// ReadHeadFastBlockHash retrieves the current fast-sync head block hash.
func (c *ChainDB) ReadHeadFastBlockHash() common.Hash {
	return rawdb.ReadHeadFastBlockHash(c.db)
}

// WriteHeadFastBlockHash stores the current head block hash.
//
// Note that this is different from the canonical head header hash as
// this corresponds to all block data and not just the header.
func (w *writer) WriteHeadFastBlockHash(hash common.Hash) {
	rawdb.WriteHeadFastBlockHash(w.db, hash)
}

// ReadCanonicalHash retrieves the canonical hash for a block number,
// returning a common.Hash{} if it does not exist.
func (c *ChainDB) ReadCanonicalHash(number uint64) common.Hash {
	return rawdb.ReadCanonicalHash(c.db, number)
}

// WriteCanonicalHash stores the canoical hash assigned to a block number.
func (w *writer) WriteCanonicalHash(number uint64, hash common.Hash) {
	rawdb.WriteCanonicalHash(w.db, hash, number)
}

// DeleteCanonicalHash removes the number to canonical hash mapping.
func (w *writer) DeleteCanonicalHash(number uint64) {
	rawdb.DeleteCanonicalHash(w.db, number)
}

// ReadHeaderNumber returns the header number assigned to a hash.
func (c *ChainDB) ReadHeaderNumber(hash common.Hash) *uint64 {
	return rawdb.ReadHeaderNumber(c.db, hash)
}

// writeHeaderNumber associates the block number with the hash.
func (w *writer) writeHeaderNumber(hash common.Hash, number uint64) {
	rawdb.WriteHeaderNumber(w.db, hash, number)
}

// HasHeader returns whether the block header associated with the
// specified hash and number pair is currently stored.
func (c *ChainDB) HasHeader(hash common.Hash, number uint64) bool {
	return rawdb.HasHeader(c.db, hash, number)
}

// ReadHeader returns the block header associated with the specified
// hash and number pair.
func (c *ChainDB) ReadHeader(hash common.Hash, number uint64) *types.Header {
	return rawdb.ReadHeader(c.db, hash, number)
}

// WriteHeader stores the block header, including its associated
// hash-to-number mapping.
func (w *writer) WriteHeader(header *types.Header) {
	rawdb.WriteHeader(w.db, header)
}

// DeleteHeader removes the block header and its associated
// hash-to-number mapping.
func (w *writer) DeleteHeader(hash common.Hash, number uint64) {
	rawdb.DeleteHeader(w.db, hash, number)
}

// HasBody returns whether the block body associated with the
// specified hash and number pair is currently stored.
func (c *ChainDB) HasBody(hash common.Hash, number uint64) bool {
	return rawdb.HasBody(c.db, hash, number)
}

// ReadBody returns the block body associated with the specified
// hash and number pair.
func (c *ChainDB) ReadBody(hash common.Hash, number uint64) *types.Body {
	return rawdb.ReadBody(c.db, hash, number)
}

// WriteBody stores the block body associated with hash and number.
func (w *writer) WriteBody(hash common.Hash, number uint64, body *types.Body) {
	rawdb.WriteBody(w.db, hash, number, body)
}

// DeleteBody removes the block body associated with the specified hash
// and number pair.
func (w *writer) DeleteBody(hash common.Hash, number uint64) {
	rawdb.DeleteBody(w.db, hash, number)
}

// ReadBodyRLP returns the RLP-encoded block body associated with the specified
// hash and number pair.
func (c *ChainDB) ReadBodyRLP(hash common.Hash, number uint64) rlp.RawValue {
	return rawdb.ReadBodyRLP(c.db, hash, number)
}

// ReadBlock returns the block associated with the specified hash and
// number pair.
func (c *ChainDB) ReadBlock(hash common.Hash, number uint64) *types.Block {
	return rawdb.ReadBlock(c.db, hash, number)
}

// WriteBlock stores the block (header and body), including its associated
// hash-to-number mapping.
func (w *writer) WriteBlock(block *types.Block) {
	rawdb.WriteBlock(w.db, block)
}

// DeleteBlock removes the block, receipts and other metadata associated
// with the specified hash and number pair.
func (w *writer) DeleteBlock(hash common.Hash, number uint64) {
	rawdb.DeleteBody(w.db, hash, number)
}

// HasHeader returns whether the transaction receipts associated with
// the specified hash and number pair is currently stored.
func (c *ChainDB) HasReceipts(hash common.Hash, number uint64) bool {
	return rawdb.HasReceipts(c.db, hash, number)
}

// ReadReceipts returns the transaction receipts in the block with the
// specified hash and number pair.
func (c *ChainDB) ReadReceipts(hash common.Hash, number uint64) types.Receipts {
	return rawdb.ReadReceipts(c.db, hash, number)
}

// WriteReceipts stores transaction receipts associated with the hash and
// number pair.
func (w *writer) WriteReceipts(hash common.Hash, number uint64, receipts types.Receipts) {
	rawdb.WriteReceipts(w.db, hash, number, receipts)
}

// DeleteReceipts removes transaction receipts associated with the hash and
// number pair.
func (w *writer) DeleteReceipts(hash common.Hash, number uint64) {
	rawdb.DeleteReceipts(w.db, hash, number)
}

// ReadTD returns the total difficulty associated with the specified
// specified block header hash and number pair.
func (c *ChainDB) ReadTD(hash common.Hash, number uint64) *big.Int {
	return rawdb.ReadTd(c.db, hash, number)
}

// WriteTd stores the total difficulty associated with the specified
// hash and number pair.
func (w *writer) WriteTD(hash common.Hash, number uint64, td *big.Int) {
	rawdb.WriteTd(w.db, hash, number, td)
}

// DeleteTd removes the total difficulty data associated with the specified
// hash and number pair.
func (w *writer) DeleteTD(hash common.Hash, number uint64) {
	rawdb.DeleteTd(w.db, hash, number)
}

// ReadTxLookupEntry retrieves the block hash the transaction is located in.
func (c *ChainDB) ReadTxLookupEntry(hash common.Hash) common.Hash {
	return rawdb.ReadTxLookupEntry(c.db, hash)
}

// WriteTxLookupEntries stores the block hash each of the transactions
// within the block are stored in, enabling hash based transaction and receipt lookups.

// Note that the block hash-to-number mapping and block body is assumed to have been stored
// already by  WriteBlock.
func (w *writer) WriteTxLookupEntries(block *types.Block) {
	rawdb.WriteTxLookupEntries(w.db, block)
}

// DeleteTxLookupEntry removes the lookup metadat for transaction hash.
func (w *writer) DeleteTxLookupEntry(hash common.Hash) {
	rawdb.DeleteTxLookupEntry(w.db, hash)
}

func (c *ChainDB) ReadTransaction(hash common.Hash) (*types.Transaction, common.Hash, uint64, uint64) {
	return rawdb.ReadTransaction(c.db, hash)
}

// ReadPreimage retrieves a the preimage of the hash.
func (c *ChainDB) ReadPreimage(hash common.Hash) []byte {
	return rawdb.ReadPreimage(c.db, hash)
}

// WritePreimages stores the provided preimage mappings.
func (w *writer) WritePreimages(preimages map[common.Hash][]byte) {
	rawdb.WritePreimages(w.db, preimages)
}
