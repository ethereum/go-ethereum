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

// Package light implements on-demand retrieval capable state and chain objects
// for the Ethereum Light Client.
package les

import (
	"bytes"
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

type LesOdrRequest interface {
	GetCost(*peer) uint64
	Request(uint64, *peer) error
	Valid(ethdb.Database, *Msg) bool // if true, keeps the retrieved object
}

func LesRequest(req light.OdrRequest) LesOdrRequest {
	switch r := req.(type) {
	case *light.BlockRequest:
		return (*BlockRequest)(r)
	case *light.ReceiptsRequest:
		return (*ReceiptsRequest)(r)
	case *light.TrieRequest:
		return (*TrieRequest)(r)
	case *light.CodeRequest:
		return (*CodeRequest)(r)
	case *light.ChtRequest:
		return (*ChtRequest)(r)
	default:
		return nil
	}
}

// BlockRequest is the ODR request type for block bodies
type BlockRequest light.BlockRequest

// GetCost returns the cost of the given ODR request according to the serving
// peer's cost table (implementation of LesOdrRequest)
func (self *BlockRequest) GetCost(peer *peer) uint64 {
	return peer.GetRequestCost(GetBlockBodiesMsg, 1)
}

// Request sends an ODR request to the LES network (implementation of LesOdrRequest)
func (self *BlockRequest) Request(reqID uint64, peer *peer) error {
	glog.V(logger.Debug).Infof("ODR: requesting body of block %08x from peer %v", self.Hash[:4], peer.id)
	return peer.RequestBodies(reqID, self.GetCost(peer), []common.Hash{self.Hash})
}

// Valid processes an ODR request reply message from the LES network
// returns true and stores results in memory if the message was a valid reply
// to the request (implementation of LesOdrRequest)
func (self *BlockRequest) Valid(db ethdb.Database, msg *Msg) bool {
	glog.V(logger.Debug).Infof("ODR: validating body of block %08x", self.Hash[:4])
	if msg.MsgType != MsgBlockBodies {
		glog.V(logger.Debug).Infof("ODR: invalid message type")
		return false
	}
	bodies := msg.Obj.([]*types.Body)
	if len(bodies) != 1 {
		glog.V(logger.Debug).Infof("ODR: invalid number of entries: %d", len(bodies))
		return false
	}
	body := bodies[0]
	header := core.GetHeader(db, self.Hash, self.Number)
	if header == nil {
		glog.V(logger.Debug).Infof("ODR: header not found for block %08x", self.Hash[:4])
		return false
	}
	txHash := types.DeriveSha(types.Transactions(body.Transactions))
	if header.TxHash != txHash {
		glog.V(logger.Debug).Infof("ODR: header.TxHash %08x does not match received txHash %08x", header.TxHash[:4], txHash[:4])
		return false
	}
	uncleHash := types.CalcUncleHash(body.Uncles)
	if header.UncleHash != uncleHash {
		glog.V(logger.Debug).Infof("ODR: header.UncleHash %08x does not match received uncleHash %08x", header.UncleHash[:4], uncleHash[:4])
		return false
	}
	data, err := rlp.EncodeToBytes(body)
	if err != nil {
		glog.V(logger.Debug).Infof("ODR: body RLP encode error: %v", err)
		return false
	}
	self.Rlp = data
	glog.V(logger.Debug).Infof("ODR: validation successful")
	return true
}

// ReceiptsRequest is the ODR request type for block receipts by block hash
type ReceiptsRequest light.ReceiptsRequest

// GetCost returns the cost of the given ODR request according to the serving
// peer's cost table (implementation of LesOdrRequest)
func (self *ReceiptsRequest) GetCost(peer *peer) uint64 {
	return peer.GetRequestCost(GetReceiptsMsg, 1)
}

// Request sends an ODR request to the LES network (implementation of LesOdrRequest)
func (self *ReceiptsRequest) Request(reqID uint64, peer *peer) error {
	glog.V(logger.Debug).Infof("ODR: requesting receipts for block %08x from peer %v", self.Hash[:4], peer.id)
	return peer.RequestReceipts(reqID, self.GetCost(peer), []common.Hash{self.Hash})
}

// Valid processes an ODR request reply message from the LES network
// returns true and stores results in memory if the message was a valid reply
// to the request (implementation of LesOdrRequest)
func (self *ReceiptsRequest) Valid(db ethdb.Database, msg *Msg) bool {
	glog.V(logger.Debug).Infof("ODR: validating receipts for block %08x", self.Hash[:4])
	if msg.MsgType != MsgReceipts {
		glog.V(logger.Debug).Infof("ODR: invalid message type")
		return false
	}
	receipts := msg.Obj.([]types.Receipts)
	if len(receipts) != 1 {
		glog.V(logger.Debug).Infof("ODR: invalid number of entries: %d", len(receipts))
		return false
	}
	hash := types.DeriveSha(receipts[0])
	header := core.GetHeader(db, self.Hash, self.Number)
	if header == nil {
		glog.V(logger.Debug).Infof("ODR: header not found for block %08x", self.Hash[:4])
		return false
	}
	if !bytes.Equal(header.ReceiptHash[:], hash[:]) {
		glog.V(logger.Debug).Infof("ODR: header receipts hash %08x does not match calculated RLP hash %08x", header.ReceiptHash[:4], hash[:4])
		return false
	}
	self.Receipts = receipts[0]
	glog.V(logger.Debug).Infof("ODR: validation successful")
	return true
}

type ProofReq struct {
	BHash       common.Hash
	AccKey, Key []byte
	FromLevel   uint
}

// ODR request type for state/storage trie entries, see LesOdrRequest interface
type TrieRequest light.TrieRequest

// GetCost returns the cost of the given ODR request according to the serving
// peer's cost table (implementation of LesOdrRequest)
func (self *TrieRequest) GetCost(peer *peer) uint64 {
	return peer.GetRequestCost(GetProofsMsg, 1)
}

// Request sends an ODR request to the LES network (implementation of LesOdrRequest)
func (self *TrieRequest) Request(reqID uint64, peer *peer) error {
	glog.V(logger.Debug).Infof("ODR: requesting trie root %08x key %08x from peer %v", self.Id.Root[:4], self.Key[:4], peer.id)
	req := &ProofReq{
		BHash:  self.Id.BlockHash,
		AccKey: self.Id.AccKey,
		Key:    self.Key,
	}
	return peer.RequestProofs(reqID, self.GetCost(peer), []*ProofReq{req})
}

// Valid processes an ODR request reply message from the LES network
// returns true and stores results in memory if the message was a valid reply
// to the request (implementation of LesOdrRequest)
func (self *TrieRequest) Valid(db ethdb.Database, msg *Msg) bool {
	glog.V(logger.Debug).Infof("ODR: validating trie root %08x key %08x", self.Id.Root[:4], self.Key[:4])

	if msg.MsgType != MsgProofs {
		glog.V(logger.Debug).Infof("ODR: invalid message type")
		return false
	}
	proofs := msg.Obj.([][]rlp.RawValue)
	if len(proofs) != 1 {
		glog.V(logger.Debug).Infof("ODR: invalid number of entries: %d", len(proofs))
		return false
	}
	_, err := trie.VerifyProof(self.Id.Root, self.Key, proofs[0])
	if err != nil {
		glog.V(logger.Debug).Infof("ODR: merkle proof verification error: %v", err)
		return false
	}
	self.Proof = proofs[0]
	glog.V(logger.Debug).Infof("ODR: validation successful")
	return true
}

type CodeReq struct {
	BHash  common.Hash
	AccKey []byte
}

// ODR request type for node data (used for retrieving contract code), see LesOdrRequest interface
type CodeRequest light.CodeRequest

// GetCost returns the cost of the given ODR request according to the serving
// peer's cost table (implementation of LesOdrRequest)
func (self *CodeRequest) GetCost(peer *peer) uint64 {
	return peer.GetRequestCost(GetCodeMsg, 1)
}

// Request sends an ODR request to the LES network (implementation of LesOdrRequest)
func (self *CodeRequest) Request(reqID uint64, peer *peer) error {
	glog.V(logger.Debug).Infof("ODR: requesting node data for hash %08x from peer %v", self.Hash[:4], peer.id)
	req := &CodeReq{
		BHash:  self.Id.BlockHash,
		AccKey: self.Id.AccKey,
	}
	return peer.RequestCode(reqID, self.GetCost(peer), []*CodeReq{req})
}

// Valid processes an ODR request reply message from the LES network
// returns true and stores results in memory if the message was a valid reply
// to the request (implementation of LesOdrRequest)
func (self *CodeRequest) Valid(db ethdb.Database, msg *Msg) bool {
	glog.V(logger.Debug).Infof("ODR: validating node data for hash %08x", self.Hash[:4])
	if msg.MsgType != MsgCode {
		glog.V(logger.Debug).Infof("ODR: invalid message type")
		return false
	}
	reply := msg.Obj.([][]byte)
	if len(reply) != 1 {
		glog.V(logger.Debug).Infof("ODR: invalid number of entries: %d", len(reply))
		return false
	}
	data := reply[0]
	hash := crypto.Sha3Hash(data)
	if !bytes.Equal(self.Hash[:], hash[:]) {
		glog.V(logger.Debug).Infof("ODR: requested hash %08x does not match received data hash %08x", self.Hash[:4], hash[:4])
		return false
	}
	self.Data = data
	glog.V(logger.Debug).Infof("ODR: validation successful")
	return true
}

type ChtReq struct {
	ChtNum, BlockNum, FromLevel uint64
}

type ChtResp struct {
	Header *types.Header
	Proof  []rlp.RawValue
}

// ODR request type for requesting headers by Canonical Hash Trie, see LesOdrRequest interface
type ChtRequest light.ChtRequest

// GetCost returns the cost of the given ODR request according to the serving
// peer's cost table (implementation of LesOdrRequest)
func (self *ChtRequest) GetCost(peer *peer) uint64 {
	return peer.GetRequestCost(GetHeaderProofsMsg, 1)
}

// Request sends an ODR request to the LES network (implementation of LesOdrRequest)
func (self *ChtRequest) Request(reqID uint64, peer *peer) error {
	glog.V(logger.Debug).Infof("ODR: requesting CHT #%d block #%d from peer %v", self.ChtNum, self.BlockNum, peer.id)
	req := &ChtReq{
		ChtNum:   self.ChtNum,
		BlockNum: self.BlockNum,
	}
	return peer.RequestHeaderProofs(reqID, self.GetCost(peer), []*ChtReq{req})
}

// Valid processes an ODR request reply message from the LES network
// returns true and stores results in memory if the message was a valid reply
// to the request (implementation of LesOdrRequest)
func (self *ChtRequest) Valid(db ethdb.Database, msg *Msg) bool {
	glog.V(logger.Debug).Infof("ODR: validating CHT #%d block #%d", self.ChtNum, self.BlockNum)

	if msg.MsgType != MsgHeaderProofs {
		glog.V(logger.Debug).Infof("ODR: invalid message type")
		return false
	}
	proofs := msg.Obj.([]ChtResp)
	if len(proofs) != 1 {
		glog.V(logger.Debug).Infof("ODR: invalid number of entries: %d", len(proofs))
		return false
	}
	proof := proofs[0]
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], self.BlockNum)
	value, err := trie.VerifyProof(self.ChtRoot, encNumber[:], proof.Proof)
	if err != nil {
		glog.V(logger.Debug).Infof("ODR: CHT merkle proof verification error: %v", err)
		return false
	}
	var node light.ChtNode
	if err := rlp.DecodeBytes(value, &node); err != nil {
		glog.V(logger.Debug).Infof("ODR: error decoding CHT node: %v", err)
		return false
	}
	if node.Hash != proof.Header.Hash() {
		glog.V(logger.Debug).Infof("ODR: CHT header hash does not match")
		return false
	}

	self.Proof = proof.Proof
	self.Header = proof.Header
	self.Td = node.Td
	glog.V(logger.Debug).Infof("ODR: validation successful")
	return true
}
