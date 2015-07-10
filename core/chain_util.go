// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"bytes"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	blockHashPre = []byte("block-hash-")
	blockNumPre  = []byte("block-num-")
)

// CalcDifficulty is the difficulty adjustment algorithm. It returns
// the difficulty that a new block b should have when created at time
// given the parent block's time and difficulty.
func CalcDifficulty(time, parentTime uint64, parentDiff *big.Int) *big.Int {
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
		return params.MinimumDifficulty
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
func CalcGasLimit(parent *types.Block) *big.Int {
	decay := new(big.Int).Div(parent.GasLimit(), params.GasLimitBoundDivisor)
	contrib := new(big.Int).Mul(parent.GasUsed(), big.NewInt(3))
	contrib = contrib.Div(contrib, big.NewInt(2))
	contrib = contrib.Div(contrib, params.GasLimitBoundDivisor)

	gl := new(big.Int).Sub(parent.GasLimit(), decay)
	gl = gl.Add(gl, contrib)
	gl = gl.Add(gl, big.NewInt(1))
	gl.Set(common.BigMax(gl, params.MinGasLimit))

	if gl.Cmp(params.GenesisGasLimit) < 0 {
		gl.Add(parent.GasLimit(), decay)
		gl.Set(common.BigMin(gl, params.GenesisGasLimit))
	}
	return gl
}

// GetBlockByHash returns the block corresponding to the hash or nil if not found
func GetBlockByHash(db common.Database, hash common.Hash) *types.Block {
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

// GetBlockByHash returns the canonical block by number or nil if not found
func GetBlockByNumber(db common.Database, number uint64) *types.Block {
	key, _ := db.Get(append(blockNumPre, big.NewInt(int64(number)).Bytes()...))
	if len(key) == 0 {
		return nil
	}

	return GetBlockByHash(db, common.BytesToHash(key))
}

// WriteHead force writes the current head
func WriteHead(db common.Database, block *types.Block) error {
	key := append(blockNumPre, block.Number().Bytes()...)
	err := db.Put(key, block.Hash().Bytes())
	if err != nil {
		return err
	}
	err = db.Put([]byte("LastBlock"), block.Hash().Bytes())
	if err != nil {
		return err
	}
	return nil
}

// WriteBlock writes a block to the database
func WriteBlock(db common.Database, block *types.Block) error {
	tstart := time.Now()

	enc, _ := rlp.EncodeToBytes((*types.StorageBlock)(block))
	key := append(blockHashPre, block.Hash().Bytes()...)
	err := db.Put(key, enc)
	if err != nil {
		glog.Fatal("db write fail:", err)
		return err
	}

	if glog.V(logger.Debug) {
		glog.Infof("wrote block #%v %s. Took %v\n", block.Number(), common.PP(block.Hash().Bytes()), time.Since(tstart))
	}

	return nil
}
