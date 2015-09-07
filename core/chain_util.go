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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	headHeaderKey = []byte("LastHeader")
	headBlockKey  = []byte("LastBlock")

	blockPrefix    = []byte("block-")
	blockNumPrefix = []byte("block-num-")

	headerSuffix = []byte("-header")
	bodySuffix   = []byte("-body")
	tdSuffix     = []byte("-td")

	ExpDiffPeriod = big.NewInt(100000)
	blockHashPre  = []byte("block-hash-") // [deprecated by eth/63]
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
func GetCanonicalHash(db common.Database, number uint64) common.Hash {
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
// fast synchronization mechanism.
func GetHeadHeaderHash(db common.Database) common.Hash {
	data, _ := db.Get(headHeaderKey)
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// GetHeadBlockHash retrieves the hash of the current canonical head block.
func GetHeadBlockHash(db common.Database) common.Hash {
	data, _ := db.Get(headBlockKey)
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// GetHeaderRLP retrieves a block header in its raw RLP database encoding, or nil
// if the header's not found.
func GetHeaderRLP(db common.Database, hash common.Hash) rlp.RawValue {
	data, _ := db.Get(append(append(blockPrefix, hash[:]...), headerSuffix...))
	return data
}

// GetHeader retrieves the block header corresponding to the hash, nil if none
// found.
func GetHeader(db common.Database, hash common.Hash) *types.Header {
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
func GetBodyRLP(db common.Database, hash common.Hash) rlp.RawValue {
	data, _ := db.Get(append(append(blockPrefix, hash[:]...), bodySuffix...))
	return data
}

// GetBody retrieves the block body (transactons, uncles) corresponding to the
// hash, nil if none found.
func GetBody(db common.Database, hash common.Hash) *types.Body {
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
func GetTd(db common.Database, hash common.Hash) *big.Int {
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
func GetBlock(db common.Database, hash common.Hash) *types.Block {
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

// WriteCanonicalHash stores the canonical hash for the given block number.
func WriteCanonicalHash(db common.Database, hash common.Hash, number uint64) error {
	key := append(blockNumPrefix, big.NewInt(int64(number)).Bytes()...)
	if err := db.Put(key, hash.Bytes()); err != nil {
		glog.Fatalf("failed to store number to hash mapping into database: %v", err)
		return err
	}
	return nil
}

// WriteHeadHeaderHash stores the head header's hash.
func WriteHeadHeaderHash(db common.Database, hash common.Hash) error {
	if err := db.Put(headHeaderKey, hash.Bytes()); err != nil {
		glog.Fatalf("failed to store last header's hash into database: %v", err)
		return err
	}
	return nil
}

// WriteHeadBlockHash stores the head block's hash.
func WriteHeadBlockHash(db common.Database, hash common.Hash) error {
	if err := db.Put(headBlockKey, hash.Bytes()); err != nil {
		glog.Fatalf("failed to store last block's hash into database: %v", err)
		return err
	}
	return nil
}

// WriteHeader serializes a block header into the database.
func WriteHeader(db common.Database, header *types.Header) error {
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
func WriteBody(db common.Database, hash common.Hash, body *types.Body) error {
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
func WriteTd(db common.Database, hash common.Hash, td *big.Int) error {
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
func WriteBlock(db common.Database, block *types.Block) error {
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

// DeleteCanonicalHash removes the number to hash canonical mapping.
func DeleteCanonicalHash(db common.Database, number uint64) {
	db.Delete(append(blockNumPrefix, big.NewInt(int64(number)).Bytes()...))
}

// DeleteHeader removes all block header data associated with a hash.
func DeleteHeader(db common.Database, hash common.Hash) {
	db.Delete(append(append(blockPrefix, hash.Bytes()...), headerSuffix...))
}

// DeleteBody removes all block body data associated with a hash.
func DeleteBody(db common.Database, hash common.Hash) {
	db.Delete(append(append(blockPrefix, hash.Bytes()...), bodySuffix...))
}

// DeleteTd removes all block total difficulty data associated with a hash.
func DeleteTd(db common.Database, hash common.Hash) {
	db.Delete(append(append(blockPrefix, hash.Bytes()...), tdSuffix...))
}

// DeleteBlock removes all block data associated with a hash.
func DeleteBlock(db common.Database, hash common.Hash) {
	DeleteHeader(db, hash)
	DeleteBody(db, hash)
	DeleteTd(db, hash)
}

// [deprecated by eth/63]
// GetBlockByHashOld returns the old combined block corresponding to the hash
// or nil if not found. This method is only used by the upgrade mechanism to
// access the old combined block representation. It will be dropped after the
// network transitions to eth/63.
func GetBlockByHashOld(db common.Database, hash common.Hash) *types.Block {
	data, _ := db.Get(append(blockHashPre, hash[:]...))
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
