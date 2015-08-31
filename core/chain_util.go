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
	headKey = []byte("LastBlock")

	headerHashPre = []byte("header-hash-")
	bodyHashPre   = []byte("body-hash-")
	blockNumPre   = []byte("block-num-")
	ExpDiffPeriod = big.NewInt(100000)

	blockHashPre = []byte("block-hash-") // [deprecated by eth/63]
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

// CalcTD computes the total difficulty of block.
func CalcTD(block, parent *types.Block) *big.Int {
	if parent == nil {
		return block.Difficulty()
	}
	d := block.Difficulty()
	d.Add(d, parent.Td)
	return d
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

// storageBody is the block body encoding used for the database.
type storageBody struct {
	Transactions []*types.Transaction
	Uncles       []*types.Header
}

// GetHashByNumber retrieves a hash assigned to a canonical block number.
func GetHashByNumber(db common.Database, number uint64) common.Hash {
	data, _ := db.Get(append(blockNumPre, big.NewInt(int64(number)).Bytes()...))
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// GetHeadHash retrieves the hash of the current canonical head block.
func GetHeadHash(db common.Database) common.Hash {
	data, _ := db.Get(headKey)
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// GetHeaderRLPByHash retrieves a block header in its raw RLP database encoding,
// or nil if the header's not found.
func GetHeaderRLPByHash(db common.Database, hash common.Hash) []byte {
	data, _ := db.Get(append(headerHashPre, hash[:]...))
	return data
}

// GetHeaderByHash retrieves the block header corresponding to the hash, nil if
// none found.
func GetHeaderByHash(db common.Database, hash common.Hash) *types.Header {
	data := GetHeaderRLPByHash(db, hash)
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

// GetBodyRLPByHash retrieves the block body (transactions and uncles) in RLP
// encoding, and the associated total difficulty.
func GetBodyRLPByHash(db common.Database, hash common.Hash) ([]byte, *big.Int) {
	combo, _ := db.Get(append(bodyHashPre, hash[:]...))
	if len(combo) == 0 {
		return nil, nil
	}
	buffer := bytes.NewBuffer(combo)

	td := new(big.Int)
	if err := rlp.Decode(buffer, td); err != nil {
		glog.V(logger.Error).Infof("invalid block td RLP for hash %x: %v", hash, err)
		return nil, nil
	}
	return buffer.Bytes(), td
}

// GetBodyByHash retrieves the block body (transactons, uncles, total difficulty)
// corresponding to the hash, nils if none found.
func GetBodyByHash(db common.Database, hash common.Hash) ([]*types.Transaction, []*types.Header, *big.Int) {
	data, td := GetBodyRLPByHash(db, hash)
	if len(data) == 0 || td == nil {
		return nil, nil, nil
	}
	body := new(storageBody)
	if err := rlp.Decode(bytes.NewReader(data), body); err != nil {
		glog.V(logger.Error).Infof("invalid block body RLP for hash %x: %v", hash, err)
		return nil, nil, nil
	}
	return body.Transactions, body.Uncles, td
}

// GetBlockByHash retrieves an entire block corresponding to the hash, assembling
// it back from the stored header and body.
func GetBlockByHash(db common.Database, hash common.Hash) *types.Block {
	// Retrieve the block header and body contents
	header := GetHeaderByHash(db, hash)
	if header == nil {
		return nil
	}
	transactions, uncles, td := GetBodyByHash(db, hash)
	if td == nil {
		return nil
	}
	// Reassemble the block and return
	block := types.NewBlockWithHeader(header).WithBody(transactions, uncles)
	block.Td = td

	return block
}

// GetBlockByNumber returns the canonical block by number or nil if not found.
func GetBlockByNumber(db common.Database, number uint64) *types.Block {
	key, _ := db.Get(append(blockNumPre, big.NewInt(int64(number)).Bytes()...))
	if len(key) == 0 {
		return nil
	}
	return GetBlockByHash(db, common.BytesToHash(key))
}

// WriteCanonNumber stores the canonical hash for the given block number.
func WriteCanonNumber(db common.Database, hash common.Hash, number uint64) error {
	key := append(blockNumPre, big.NewInt(int64(number)).Bytes()...)
	if err := db.Put(key, hash.Bytes()); err != nil {
		glog.Fatalf("failed to store number to hash mapping into database: %v", err)
		return err
	}
	return nil
}

// WriteHead updates the head block of the chain database.
func WriteHead(db common.Database, block *types.Block) error {
	if err := WriteCanonNumber(db, block.Hash(), block.NumberU64()); err != nil {
		glog.Fatalf("failed to store canonical number into database: %v", err)
		return err
	}
	if err := db.Put(headKey, block.Hash().Bytes()); err != nil {
		glog.Fatalf("failed to store last block into database: %v", err)
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
	key := append(headerHashPre, header.Hash().Bytes()...)
	if err := db.Put(key, data); err != nil {
		glog.Fatalf("failed to store header into database: %v", err)
		return err
	}
	glog.V(logger.Debug).Infof("stored header #%v [%x…]", header.Number, header.Hash().Bytes()[:4])
	return nil
}

// WriteBody serializes the body of a block into the database.
func WriteBody(db common.Database, block *types.Block) error {
	body, err := rlp.EncodeToBytes(&storageBody{block.Transactions(), block.Uncles()})
	if err != nil {
		return err
	}
	td, err := rlp.EncodeToBytes(block.Td)
	if err != nil {
		return err
	}
	key := append(bodyHashPre, block.Hash().Bytes()...)
	if err := db.Put(key, append(td, body...)); err != nil {
		glog.Fatalf("failed to store block body into database: %v", err)
		return err
	}
	glog.V(logger.Debug).Infof("stored block body #%v [%x…]", block.Number, block.Hash().Bytes()[:4])
	return nil
}

// WriteBlock serializes a block into the database, header and body separately.
func WriteBlock(db common.Database, block *types.Block) error {
	// Store the body first to retain database consistency
	if err := WriteBody(db, block); err != nil {
		return err
	}
	// Store the header too, signaling full block ownership
	if err := WriteHeader(db, block.Header()); err != nil {
		return err
	}
	return nil
}

// DeleteHeader removes all block header data associated with a hash.
func DeleteHeader(db common.Database, hash common.Hash) {
	db.Delete(append(headerHashPre, hash.Bytes()...))
}

// DeleteBody removes all block body data associated with a hash.
func DeleteBody(db common.Database, hash common.Hash) {
	db.Delete(append(bodyHashPre, hash.Bytes()...))
}

// DeleteBlock removes all block data associated with a hash.
func DeleteBlock(db common.Database, hash common.Hash) {
	DeleteHeader(db, hash)
	DeleteBody(db, hash)
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
