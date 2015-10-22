// Copyright 2015 The go-ethereum Authors
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

package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	headHeaderKey = []byte("LastHeader")
	headBlockKey  = []byte("LastBlock")
	headFastKey   = []byte("LastFast")

	blockPrefix    = []byte("block-")
	blockNumPrefix = []byte("block-num-")

	headerSuffix = []byte("-header")
	bodySuffix   = []byte("-body")
	tdSuffix     = []byte("-td")

	txMetaSuffix        = []byte{0x01}
	receiptsPrefix      = []byte("receipts-")
	blockReceiptsPrefix = []byte("receipts-block-")

	mipmapPre    = []byte("mipmap-log-bloom-")
	MIPMapLevels = []uint64{1000000, 500000, 100000, 50000, 1000}

	ExpDiffPeriod   = big.NewInt(100000)
	blockHashPrefix = []byte("block-hash-") // [deprecated by the header/block split, remove eventually]
)

// CalcDifficulty is the difficulty adjustment algorithm. It returns
// the difficulty that a new block b should have when created at time
// given the parent block's time and difficulty.
func CalcDifficulty(time, parentTime uint64, parentNumber, parentDiff *big.Int) *big.Int {
	diff := new(big.Int)
	adjust := new(big.Int).Div(parentDiff, params.DifficultyBoundDivisor)
	bigTime := new(big.Int)
	bigParentTime := new(big.Int)

	bigTime.SetUint64(time)
	bigParentTime.SetUint64(parentTime)

	if bigTime.Sub(bigTime, bigParentTime).Cmp(params.DurationLimit) < 0 {
		diff.Add(parentDiff, adjust)
	} else {
		diff.Sub(parentDiff, adjust)
	}
	if diff.Cmp(params.MinimumDifficulty) < 0 {
		diff = params.MinimumDifficulty
	}

	periodCount := new(big.Int).Add(parentNumber, common.Big1)
	periodCount.Div(periodCount, ExpDiffPeriod)
	if periodCount.Cmp(common.Big1) > 0 {
		// diff = diff + 2^(periodCount - 2)
		expDiff := periodCount.Sub(periodCount, common.Big2)
		expDiff.Exp(common.Big2, expDiff, nil)
		diff.Add(diff, expDiff)
		diff = common.BigMax(diff, params.MinimumDifficulty)
	}

	return diff
}

// CalcGasLimit computes the gas limit of the next block after parent.
// The result may be modified by the caller.
// This is miner strategy, not consensus protocol.
func CalcGasLimit(parent *types.Block) *big.Int {
	// contrib = (parentGasUsed * 3 / 2) / 1024
	contrib := new(big.Int).Mul(parent.GasUsed(), big.NewInt(3))
	contrib = contrib.Div(contrib, big.NewInt(2))
	contrib = contrib.Div(contrib, params.GasLimitBoundDivisor)

	// decay = parentGasLimit / 1024 -1
	decay := new(big.Int).Div(parent.GasLimit(), params.GasLimitBoundDivisor)
	decay.Sub(decay, big.NewInt(1))

	/*
		strategy: gasLimit of block-to-mine is set based on parent's
		gasUsed value.  if parentGasUsed > parentGasLimit * (2/3) then we
		increase it, otherwise lower it (or leave it unchanged if it's right
		at that usage) the amount increased/decreased depends on how far away
		from parentGasLimit * (2/3) parentGasUsed is.
	*/
	gl := new(big.Int).Sub(parent.GasLimit(), decay)
	gl = gl.Add(gl, contrib)
	gl.Set(common.BigMax(gl, params.MinGasLimit))

	// however, if we're now below the target (GenesisGasLimit) we increase the
	// limit as much as we can (parentGasLimit / 1024 -1)
	if gl.Cmp(params.GenesisGasLimit) < 0 {
		gl.Add(parent.GasLimit(), decay)
		gl.Set(common.BigMin(gl, params.GenesisGasLimit))
	}
	return gl
}

// GetCanonicalHash retrieves a hash assigned to a canonical block number.
func GetCanonicalHash(db ethdb.Database, number uint64) common.Hash {
	data, _ := db.Get(append(blockNumPrefix, big.NewInt(int64(number)).Bytes()...))
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// GetHeadHeaderHash retrieves the hash of the current canonical head block's
// header. The difference between this and GetHeadBlockHash is that whereas the
// last block hash is only updated upon a full block import, the last header
// hash is updated already at header import, allowing head tracking for the
// light synchronization mechanism.
func GetHeadHeaderHash(db ethdb.Database) common.Hash {
	data, _ := db.Get(headHeaderKey)
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// GetHeadBlockHash retrieves the hash of the current canonical head block.
func GetHeadBlockHash(db ethdb.Database) common.Hash {
	data, _ := db.Get(headBlockKey)
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// GetHeadFastBlockHash retrieves the hash of the current canonical head block during
// fast synchronization. The difference between this and GetHeadBlockHash is that
// whereas the last block hash is only updated upon a full block import, the last
// fast hash is updated when importing pre-processed blocks.
func GetHeadFastBlockHash(db ethdb.Database) common.Hash {
	data, _ := db.Get(headFastKey)
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// GetHeaderRLP retrieves a block header in its raw RLP database encoding, or nil
// if the header's not found.
func GetHeaderRLP(db ethdb.Database, hash common.Hash) rlp.RawValue {
	data, _ := db.Get(append(append(blockPrefix, hash[:]...), headerSuffix...))
	return data
}

// GetHeader retrieves the block header corresponding to the hash, nil if none
// found.
func GetHeader(db ethdb.Database, hash common.Hash) *types.Header {
	data := GetHeaderRLP(db, hash)
	if len(data) == 0 {
		return nil
	}
	header := new(types.Header)
	if err := rlp.Decode(bytes.NewReader(data), header); err != nil {
		glog.V(logger.Error).Infof("invalid block header RLP for hash %x: %v", hash, err)
		return nil
	}
	return header
}

// GetBodyRLP retrieves the block body (transactions and uncles) in RLP encoding.
func GetBodyRLP(db ethdb.Database, hash common.Hash) rlp.RawValue {
	data, _ := db.Get(append(append(blockPrefix, hash[:]...), bodySuffix...))
	return data
}

// GetBody retrieves the block body (transactons, uncles) corresponding to the
// hash, nil if none found.
func GetBody(db ethdb.Database, hash common.Hash) *types.Body {
	data := GetBodyRLP(db, hash)
	if len(data) == 0 {
		return nil
	}
	body := new(types.Body)
	if err := rlp.Decode(bytes.NewReader(data), body); err != nil {
		glog.V(logger.Error).Infof("invalid block body RLP for hash %x: %v", hash, err)
		return nil
	}
	return body
}

// GetTd retrieves a block's total difficulty corresponding to the hash, nil if
// none found.
func GetTd(db ethdb.Database, hash common.Hash) *big.Int {
	data, _ := db.Get(append(append(blockPrefix, hash.Bytes()...), tdSuffix...))
	if len(data) == 0 {
		return nil
	}
	td := new(big.Int)
	if err := rlp.Decode(bytes.NewReader(data), td); err != nil {
		glog.V(logger.Error).Infof("invalid block total difficulty RLP for hash %x: %v", hash, err)
		return nil
	}
	return td
}

// GetBlock retrieves an entire block corresponding to the hash, assembling it
// back from the stored header and body.
func GetBlock(db ethdb.Database, hash common.Hash) *types.Block {
	// Retrieve the block header and body contents
	header := GetHeader(db, hash)
	if header == nil {
		return nil
	}
	body := GetBody(db, hash)
	if body == nil {
		return nil
	}
	// Reassemble the block and return
	return types.NewBlockWithHeader(header).WithBody(body.Transactions, body.Uncles)
}

// GetBlockReceipts retrieves the receipts generated by the transactions included
// in a block given by its hash.
func GetBlockReceipts(db ethdb.Database, hash common.Hash) types.Receipts {
	data, _ := db.Get(append(blockReceiptsPrefix, hash[:]...))
	if len(data) == 0 {
		return nil
	}
	storageReceipts := []*types.ReceiptForStorage{}
	if err := rlp.DecodeBytes(data, &storageReceipts); err != nil {
		glog.V(logger.Error).Infof("invalid receipt array RLP for hash %x: %v", hash, err)
		return nil
	}
	receipts := make(types.Receipts, len(storageReceipts))
	for i, receipt := range storageReceipts {
		receipts[i] = (*types.Receipt)(receipt)
	}
	return receipts
}

// GetTransaction retrieves a specific transaction from the database, along with
// its added positional metadata.
func GetTransaction(db ethdb.Database, hash common.Hash) (*types.Transaction, common.Hash, uint64, uint64) {
	// Retrieve the transaction itself from the database
	data, _ := db.Get(hash.Bytes())
	if len(data) == 0 {
		return nil, common.Hash{}, 0, 0
	}
	var tx types.Transaction
	if err := rlp.DecodeBytes(data, &tx); err != nil {
		return nil, common.Hash{}, 0, 0
	}
	// Retrieve the blockchain positional metadata
	data, _ = db.Get(append(hash.Bytes(), txMetaSuffix...))
	if len(data) == 0 {
		return nil, common.Hash{}, 0, 0
	}
	var meta struct {
		BlockHash  common.Hash
		BlockIndex uint64
		Index      uint64
	}
	if err := rlp.DecodeBytes(data, &meta); err != nil {
		return nil, common.Hash{}, 0, 0
	}
	return &tx, meta.BlockHash, meta.BlockIndex, meta.Index
}

// GetReceipt returns a receipt by hash
func GetReceipt(db ethdb.Database, txHash common.Hash) *types.Receipt {
	data, _ := db.Get(append(receiptsPrefix, txHash[:]...))
	if len(data) == 0 {
		return nil
	}
	var receipt types.ReceiptForStorage
	err := rlp.DecodeBytes(data, &receipt)
	if err != nil {
		glog.V(logger.Core).Infoln("GetReceipt err:", err)
	}
	return (*types.Receipt)(&receipt)
}

// WriteCanonicalHash stores the canonical hash for the given block number.
func WriteCanonicalHash(db ethdb.Database, hash common.Hash, number uint64) error {
	key := append(blockNumPrefix, big.NewInt(int64(number)).Bytes()...)
	if err := db.Put(key, hash.Bytes()); err != nil {
		glog.Fatalf("failed to store number to hash mapping into database: %v", err)
		return err
	}
	return nil
}

// WriteHeadHeaderHash stores the head header's hash.
func WriteHeadHeaderHash(db ethdb.Database, hash common.Hash) error {
	if err := db.Put(headHeaderKey, hash.Bytes()); err != nil {
		glog.Fatalf("failed to store last header's hash into database: %v", err)
		return err
	}
	return nil
}

// WriteHeadBlockHash stores the head block's hash.
func WriteHeadBlockHash(db ethdb.Database, hash common.Hash) error {
	if err := db.Put(headBlockKey, hash.Bytes()); err != nil {
		glog.Fatalf("failed to store last block's hash into database: %v", err)
		return err
	}
	return nil
}

// WriteHeadFastBlockHash stores the fast head block's hash.
func WriteHeadFastBlockHash(db ethdb.Database, hash common.Hash) error {
	if err := db.Put(headFastKey, hash.Bytes()); err != nil {
		glog.Fatalf("failed to store last fast block's hash into database: %v", err)
		return err
	}
	return nil
}

// WriteHeader serializes a block header into the database.
func WriteHeader(db ethdb.Database, header *types.Header) error {
	data, err := rlp.EncodeToBytes(header)
	if err != nil {
		return err
	}
	key := append(append(blockPrefix, header.Hash().Bytes()...), headerSuffix...)
	if err := db.Put(key, data); err != nil {
		glog.Fatalf("failed to store header into database: %v", err)
		return err
	}
	glog.V(logger.Debug).Infof("stored header #%v [%x…]", header.Number, header.Hash().Bytes()[:4])
	return nil
}

// WriteBody serializes the body of a block into the database.
func WriteBody(db ethdb.Database, hash common.Hash, body *types.Body) error {
	data, err := rlp.EncodeToBytes(body)
	if err != nil {
		return err
	}
	key := append(append(blockPrefix, hash.Bytes()...), bodySuffix...)
	if err := db.Put(key, data); err != nil {
		glog.Fatalf("failed to store block body into database: %v", err)
		return err
	}
	glog.V(logger.Debug).Infof("stored block body [%x…]", hash.Bytes()[:4])
	return nil
}

// WriteTd serializes the total difficulty of a block into the database.
func WriteTd(db ethdb.Database, hash common.Hash, td *big.Int) error {
	data, err := rlp.EncodeToBytes(td)
	if err != nil {
		return err
	}
	key := append(append(blockPrefix, hash.Bytes()...), tdSuffix...)
	if err := db.Put(key, data); err != nil {
		glog.Fatalf("failed to store block total difficulty into database: %v", err)
		return err
	}
	glog.V(logger.Debug).Infof("stored block total difficulty [%x…]: %v", hash.Bytes()[:4], td)
	return nil
}

// WriteBlock serializes a block into the database, header and body separately.
func WriteBlock(db ethdb.Database, block *types.Block) error {
	// Store the body first to retain database consistency
	if err := WriteBody(db, block.Hash(), &types.Body{block.Transactions(), block.Uncles()}); err != nil {
		return err
	}
	// Store the header too, signaling full block ownership
	if err := WriteHeader(db, block.Header()); err != nil {
		return err
	}
	return nil
}

// WriteBlockReceipts stores all the transaction receipts belonging to a block
// as a single receipt slice. This is used during chain reorganisations for
// rescheduling dropped transactions.
func WriteBlockReceipts(db ethdb.Database, hash common.Hash, receipts types.Receipts) error {
	// Convert the receipts into their storage form and serialize them
	storageReceipts := make([]*types.ReceiptForStorage, len(receipts))
	for i, receipt := range receipts {
		storageReceipts[i] = (*types.ReceiptForStorage)(receipt)
	}
	bytes, err := rlp.EncodeToBytes(storageReceipts)
	if err != nil {
		return err
	}
	// Store the flattened receipt slice
	if err := db.Put(append(blockReceiptsPrefix, hash.Bytes()...), bytes); err != nil {
		glog.Fatalf("failed to store block receipts into database: %v", err)
		return err
	}
	glog.V(logger.Debug).Infof("stored block receipts [%x…]", hash.Bytes()[:4])
	return nil
}

// WriteTransactions stores the transactions associated with a specific block
// into the given database. Beside writing the transaction, the function also
// stores a metadata entry along with the transaction, detailing the position
// of this within the blockchain.
func WriteTransactions(db ethdb.Database, block *types.Block) error {
	batch := db.NewBatch()

	// Iterate over each transaction and encode it with its metadata
	for i, tx := range block.Transactions() {
		// Encode and queue up the transaction for storage
		data, err := rlp.EncodeToBytes(tx)
		if err != nil {
			return err
		}
		if err := batch.Put(tx.Hash().Bytes(), data); err != nil {
			return err
		}
		// Encode and queue up the transaction metadata for storage
		meta := struct {
			BlockHash  common.Hash
			BlockIndex uint64
			Index      uint64
		}{
			BlockHash:  block.Hash(),
			BlockIndex: block.NumberU64(),
			Index:      uint64(i),
		}
		data, err = rlp.EncodeToBytes(meta)
		if err != nil {
			return err
		}
		if err := batch.Put(append(tx.Hash().Bytes(), txMetaSuffix...), data); err != nil {
			return err
		}
	}
	// Write the scheduled data into the database
	if err := batch.Write(); err != nil {
		glog.Fatalf("failed to store transactions into database: %v", err)
		return err
	}
	return nil
}

// WriteReceipts stores a batch of transaction receipts into the database.
func WriteReceipts(db ethdb.Database, receipts types.Receipts) error {
	batch := db.NewBatch()

	// Iterate over all the receipts and queue them for database injection
	for _, receipt := range receipts {
		storageReceipt := (*types.ReceiptForStorage)(receipt)
		data, err := rlp.EncodeToBytes(storageReceipt)
		if err != nil {
			return err
		}
		if err := batch.Put(append(receiptsPrefix, receipt.TxHash.Bytes()...), data); err != nil {
			return err
		}
	}
	// Write the scheduled data into the database
	if err := batch.Write(); err != nil {
		glog.Fatalf("failed to store receipts into database: %v", err)
		return err
	}
	return nil
}

// DeleteCanonicalHash removes the number to hash canonical mapping.
func DeleteCanonicalHash(db ethdb.Database, number uint64) {
	db.Delete(append(blockNumPrefix, big.NewInt(int64(number)).Bytes()...))
}

// DeleteHeader removes all block header data associated with a hash.
func DeleteHeader(db ethdb.Database, hash common.Hash) {
	db.Delete(append(append(blockPrefix, hash.Bytes()...), headerSuffix...))
}

// DeleteBody removes all block body data associated with a hash.
func DeleteBody(db ethdb.Database, hash common.Hash) {
	db.Delete(append(append(blockPrefix, hash.Bytes()...), bodySuffix...))
}

// DeleteTd removes all block total difficulty data associated with a hash.
func DeleteTd(db ethdb.Database, hash common.Hash) {
	db.Delete(append(append(blockPrefix, hash.Bytes()...), tdSuffix...))
}

// DeleteBlock removes all block data associated with a hash.
func DeleteBlock(db ethdb.Database, hash common.Hash) {
	DeleteBlockReceipts(db, hash)
	DeleteHeader(db, hash)
	DeleteBody(db, hash)
	DeleteTd(db, hash)
}

// DeleteBlockReceipts removes all receipt data associated with a block hash.
func DeleteBlockReceipts(db ethdb.Database, hash common.Hash) {
	db.Delete(append(blockReceiptsPrefix, hash.Bytes()...))
}

// DeleteTransaction removes all transaction data associated with a hash.
func DeleteTransaction(db ethdb.Database, hash common.Hash) {
	db.Delete(hash.Bytes())
	db.Delete(append(hash.Bytes(), txMetaSuffix...))
}

// DeleteReceipt removes all receipt data associated with a transaction hash.
func DeleteReceipt(db ethdb.Database, hash common.Hash) {
	db.Delete(append(receiptsPrefix, hash.Bytes()...))
}

// [deprecated by the header/block split, remove eventually]
// GetBlockByHashOld returns the old combined block corresponding to the hash
// or nil if not found. This method is only used by the upgrade mechanism to
// access the old combined block representation. It will be dropped after the
// network transitions to eth/63.
func GetBlockByHashOld(db ethdb.Database, hash common.Hash) *types.Block {
	data, _ := db.Get(append(blockHashPrefix, hash[:]...))
	if len(data) == 0 {
		return nil
	}
	var block types.StorageBlock
	if err := rlp.Decode(bytes.NewReader(data), &block); err != nil {
		glog.V(logger.Error).Infof("invalid block RLP for hash %x: %v", hash, err)
		return nil
	}
	return (*types.Block)(&block)
}

// returns a formatted MIP mapped key by adding prefix, canonical number and level
//
// ex. fn(98, 1000) = (prefix || 1000 || 0)
func mipmapKey(num, level uint64) []byte {
	lkey := make([]byte, 8)
	binary.BigEndian.PutUint64(lkey, level)
	key := new(big.Int).SetUint64(num / level * level)

	return append(mipmapPre, append(lkey, key.Bytes()...)...)
}

// WriteMapmapBloom writes each address included in the receipts' logs to the
// MIP bloom bin.
func WriteMipmapBloom(db ethdb.Database, number uint64, receipts types.Receipts) error {
	batch := db.NewBatch()
	for _, level := range MIPMapLevels {
		key := mipmapKey(number, level)
		bloomDat, _ := db.Get(key)
		bloom := types.BytesToBloom(bloomDat)
		for _, receipt := range receipts {
			for _, log := range receipt.Logs {
				bloom.Add(log.Address.Big())
			}
		}
		batch.Put(key, bloom.Bytes())
	}
	if err := batch.Write(); err != nil {
		return fmt.Errorf("mipmap write fail for: %d: %v", number, err)
	}
	return nil
}

// GetMipmapBloom returns a bloom filter using the number and level as input
// parameters. For available levels see MIPMapLevels.
func GetMipmapBloom(db ethdb.Database, number, level uint64) types.Bloom {
	bloomDat, _ := db.Get(mipmapKey(number, level))
	return types.BytesToBloom(bloomDat)
}
