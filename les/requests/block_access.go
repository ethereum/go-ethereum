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

package requests

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/les/access"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	blockReceiptsPre = []byte("receipts-block-")
	blockPrefix      = []byte("block-")
	bodySuffix       = []byte("-body")
)

// ODR request type for block bodies, see access.ObjectAccess interface
type BlockAccess struct {
	db        ethdb.Database
	blockHash common.Hash
	rlp       []byte
	getHeader getHeaderFn
}

type getHeaderFn func(db ethdb.Database, hash common.Hash) *types.Header

func NewBlockAccess(db ethdb.Database, blockHash common.Hash, getHeader getHeaderFn) *BlockAccess {
	return &BlockAccess{db: db, blockHash: blockHash, getHeader: getHeader}
}

func (self *BlockAccess) GetRlp() []byte {
	return self.rlp
}

func (self *BlockAccess) Request(peer *access.Peer) error {
	glog.V(access.LogLevel).Infof("ODR: requesting body of block %08x from peer %v", self.blockHash[:4], peer.Id())
	return peer.GetBlockBodies([]common.Hash{self.blockHash})
}

func (self *BlockAccess) Valid(msg *access.Msg) bool {
	glog.V(access.LogLevel).Infof("ODR: validating body of block %08x", self.blockHash[:4])
	if msg.MsgType != access.MsgBlockBodies {
		glog.V(access.LogLevel).Infof("ODR: invalid message type")
		return false
	}
	bodies := msg.Obj.([]*types.Body)
	if len(bodies) != 1 {
		glog.V(access.LogLevel).Infof("ODR: invalid number of entries: %d", len(bodies))
		return false
	}
	body := bodies[0]
	header := self.getHeader(self.db, self.blockHash)
	if header == nil {
		glog.V(access.LogLevel).Infof("ODR: header not found for block %08x", self.blockHash[:4])
		return false
	}
	txHash := types.DeriveSha(types.Transactions(body.Transactions))
	if header.TxHash != txHash {
		glog.V(access.LogLevel).Infof("ODR: header.TxHash %08x does not match received txHash %08x", header.TxHash[:4], txHash[:4])
		return false
	}
	uncleHash := types.CalcUncleHash(body.Uncles)
	if header.UncleHash != uncleHash {
		glog.V(access.LogLevel).Infof("ODR: header.UncleHash %08x does not match received uncleHash %08x", header.UncleHash[:4], uncleHash[:4])
		return false
	}
	data, err := rlp.EncodeToBytes(body)
	if err != nil {
		glog.V(access.LogLevel).Infof("ODR: body RLP encode error: %v", err)
		return false
	}
	self.rlp = data
	glog.V(access.LogLevel).Infof("ODR: validation successful")
	return true
}

func (self *BlockAccess) DbGet() bool {
	self.rlp, _ = self.db.Get(append(append(blockPrefix, self.blockHash[:]...), bodySuffix...))
	glog.V(access.LogLevel).Infof("ODR: get body %08x  len = %d", self.blockHash[:4], len(self.rlp))
	return len(self.rlp) != 0
}

func (self *BlockAccess) DbPut() {
	self.db.Put(append(append(blockPrefix, self.blockHash[:]...), bodySuffix...), self.rlp)
	glog.V(access.LogLevel).Infof("ODR: put body %08x  len = %d", self.blockHash[:4], len(self.rlp))
}

// ODR request type for block receipts by block hash, see access.ObjectAccess interface
type ReceiptsAccess struct {
	db               ethdb.Database
	blockHash        common.Hash
	receipts         types.Receipts
	getHeader        getHeaderFn
	putReceipts      putReceiptsFn
	putBlockReceipts putBlockReceiptsFn
}

type putReceiptsFn func(db ethdb.Database, receipts types.Receipts) error
type putBlockReceiptsFn func(db ethdb.Database, hash common.Hash, receipts types.Receipts) error

func NewReceiptsAccess(db ethdb.Database, blockHash common.Hash, getHeader getHeaderFn, putReceipts putReceiptsFn, putBlockReceipts putBlockReceiptsFn) *ReceiptsAccess {
	return &ReceiptsAccess{db: db, blockHash: blockHash, getHeader: getHeader, putReceipts: putReceipts, putBlockReceipts: putBlockReceipts}
}

func (self *ReceiptsAccess) GetReceipts() types.Receipts {
	return self.receipts
}

func (self *ReceiptsAccess) Request(peer *access.Peer) error {
	glog.V(access.LogLevel).Infof("ODR: requesting receipts for block %08x from peer %v", self.blockHash[:4], peer.Id())
	return peer.GetReceipts([]common.Hash{self.blockHash})
}

func (self *ReceiptsAccess) Valid(msg *access.Msg) bool {
	glog.V(access.LogLevel).Infof("ODR: validating receipts for block %08x", self.blockHash[:4])
	if msg.MsgType != access.MsgReceipts {
		glog.V(access.LogLevel).Infof("ODR: invalid message type")
		return false
	}
	receipts := msg.Obj.([]types.Receipts)
	if len(receipts) != 1 {
		glog.V(access.LogLevel).Infof("ODR: invalid number of entries: %d", len(receipts))
		return false
	}
	hash := types.DeriveSha(receipts[0])
	header := self.getHeader(self.db, self.blockHash)
	if header == nil {
		glog.V(access.LogLevel).Infof("ODR: header not found for block %08x", self.blockHash[:4])
		return false
	}
	if !bytes.Equal(header.ReceiptHash[:], hash[:]) {
		glog.V(access.LogLevel).Infof("ODR: header receipts hash %08x does not match calculated RLP hash %08x", header.ReceiptHash[:4], hash[:4])
		return false
	}
	self.receipts = receipts[0]
	glog.V(access.LogLevel).Infof("ODR: validation successful")
	return true
}

func (self *ReceiptsAccess) DbGet() bool {
	data, _ := self.db.Get(append(blockReceiptsPre, self.blockHash[:]...))
	if len(data) == 0 {
		return false
	}
	rs := []*types.ReceiptForStorage{}
	if err := rlp.DecodeBytes(data, &rs); err != nil {
		glog.V(logger.Error).Infof("invalid receipt array RLP for hash %x: %v", self.blockHash, err)
		return false
	}
	self.receipts = make(types.Receipts, len(rs))
	for i, receipt := range rs {
		self.receipts[i] = (*types.Receipt)(receipt)
	}
	return true
}

func (self *ReceiptsAccess) DbPut() {
	self.putBlockReceipts(self.db, self.blockHash, self.receipts)
	self.putReceipts(self.db, self.receipts)
}
