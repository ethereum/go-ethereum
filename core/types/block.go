// Copyright 2014 The go-ethereum Authors
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

// Package types contains data types related to Ethereum consensus.
package types

import (
	"fmt"
	"io"
	"math/big"
	"sort"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// Header is the main Ethereum block, containing all of the block data belonging
// to the consensus protocol, as well as a few added fields for the implementation.
type Block struct {
	header       *Header
	uncles       []*Header
	transactions Transactions
	receipts     Receipts

	// caches
	size atomic.Value

	// Td is used by package core to store the total difficulty
	// of the chain up to and including the block.
	Td *big.Int

	// ReceivedAt is used by package eth to track block propagation time.
	ReceivedAt time.Time
}

// [deprecated by eth/63]
// StorageBlock defines the RLP encoding of a Block stored in the
// state database. The StorageBlock encoding contains fields that
// would otherwise need to be recomputed.
type StorageBlock Block

// "external" block encoding. used for eth protocol, etc.
type extblock struct {
	Header *Header
	Txs    []*Transaction
	Uncles []*Header
}

// [deprecated by eth/63]
// "storage" block encoding. used for database.
type storageblock struct {
	Header *Header
	Txs    []*Transaction
	Uncles []*Header
	TD     *big.Int
}

var (
	emptyRootHash  = DeriveSha(Transactions{})
	emptyUncleHash = CalcUncleHash(nil)
)

// NewBlock creates a new block. The input data is copied,
// changes to header and to the field values will not affect the
// block.
//
// The values of TxHash, UncleHash, ReceiptHash and Bloom in header
// are ignored and set to values derived from the given txs, uncles
// and receipts.
func NewBlock(header *RawHeader, txs []*Transaction, uncles []*Header, receipts []*Receipt) *Block {
	head := header.Copy()

	// TODO: panic if len(txs) != len(receipts)
	if len(txs) == 0 {
		head.TxHash = emptyRootHash
	} else {
		head.TxHash = DeriveSha(Transactions(txs))
	}
	txsCopy := make(Transactions, len(txs))
	copy(txsCopy, txs)

	if len(receipts) == 0 {
		head.ReceiptHash = emptyRootHash
	} else {
		head.ReceiptHash = DeriveSha(Receipts(receipts))
		head.Bloom = CreateBloom(receipts)
	}
	receiptsCopy := make([]*Receipt, len(receipts))
	copy(receiptsCopy, receipts)

	if len(uncles) == 0 {
		head.UncleHash = emptyUncleHash
	} else {
		head.UncleHash = CalcUncleHash(uncles)
	}
	unclesCopy := make([]*Header, len(uncles))
	for i := range uncles {
		unclesCopy[i] = uncles[i].Copy()
	}
	// Assemble and return an immutable block
	return &Block{
		header:       &Header{rawHeader: *head},
		transactions: txsCopy,
		receipts:     receiptsCopy,
		uncles:       unclesCopy,
		Td:           new(big.Int),
	}
}

// NewBlockWithRawHeader creates a block with the given header data. The header
// data is copied, changes to header and to the field values will not affect
// the block.
func NewBlockWithRawHeader(raw *RawHeader) *Block {
	return &Block{header: NewHeader(raw)}
}

// NewBlockWithHeader creates a block with the given immutable header.
func NewBlockWithHeader(header *Header) *Block {
	return &Block{header: header}
}

func (b *Block) ValidateFields() error {
	if b.header == nil {
		return fmt.Errorf("header is nil")
	}
	for i, transaction := range b.transactions {
		if transaction == nil {
			return fmt.Errorf("transaction %d is nil", i)
		}
	}
	for i, uncle := range b.uncles {
		if uncle == nil {
			return fmt.Errorf("uncle %d is nil", i)
		}
	}
	return nil
}

func (b *Block) DecodeRLP(s *rlp.Stream) error {
	var eb extblock
	_, size, _ := s.Kind()
	if err := s.Decode(&eb); err != nil {
		return err
	}
	b.header, b.uncles, b.transactions = eb.Header, eb.Uncles, eb.Txs
	b.size.Store(common.StorageSize(rlp.ListSize(size)))
	return nil
}

func (b *Block) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, extblock{
		Header: b.header,
		Txs:    b.transactions,
		Uncles: b.uncles,
	})
}

// [deprecated by eth/63]
func (b *StorageBlock) DecodeRLP(s *rlp.Stream) error {
	var sb storageblock
	if err := s.Decode(&sb); err != nil {
		return err
	}
	b.header, b.uncles, b.transactions, b.Td = sb.Header, sb.Uncles, sb.Txs, sb.TD
	return nil
}

// [deprecated by eth/63]
func (b *StorageBlock) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, storageblock{
		Header: b.header,
		Txs:    b.transactions,
		Uncles: b.uncles,
		TD:     b.Td,
	})
}

// TODO: copies
func (b *Block) Uncles() []*Header          { return b.uncles }
func (b *Block) Transactions() Transactions { return b.transactions }
func (b *Block) Receipts() Receipts         { return b.receipts }

func (b *Block) Transaction(hash common.Hash) *Transaction {
	for _, transaction := range b.transactions {
		if transaction.Hash() == hash {
			return transaction
		}
	}
	return nil
}

func (b *Block) Number() *big.Int     { return b.header.Number() }
func (b *Block) GasLimit() *big.Int   { return b.header.GasLimit() }
func (b *Block) GasUsed() *big.Int    { return b.header.GasUsed() }
func (b *Block) Difficulty() *big.Int { return b.header.Difficulty() }
func (b *Block) Time() *big.Int       { return b.header.Time() }

func (b *Block) NumberU64() uint64        { return b.header.Number().Uint64() }
func (b *Block) MixDigest() common.Hash   { return b.header.MixDigest() }
func (b *Block) Nonce() uint64            { return b.header.Nonce().Uint64() }
func (b *Block) Bloom() Bloom             { return b.header.Bloom() }
func (b *Block) Coinbase() common.Address { return b.header.Coinbase() }
func (b *Block) Root() common.Hash        { return b.header.Root() }
func (b *Block) ParentHash() common.Hash  { return b.header.ParentHash() }
func (b *Block) TxHash() common.Hash      { return b.header.TxHash() }
func (b *Block) ReceiptHash() common.Hash { return b.header.ReceiptHash() }
func (b *Block) UncleHash() common.Hash   { return b.header.UncleHash() }
func (b *Block) Extra() []byte            { return b.header.Extra() }

func (b *Block) Header() *Header { return b.header }

func (b *Block) HashNoNonce() common.Hash { return b.header.HashNoNonce() }
func (b *Block) Hash() common.Hash        { return b.header.Hash() }

func (b *Block) Size() common.StorageSize {
	if size := b.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	c := writeCounter(0)
	rlp.Encode(&c, b)
	b.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

type writeCounter common.StorageSize

func (c *writeCounter) Write(b []byte) (int, error) {
	*c += writeCounter(len(b))
	return len(b), nil
}

func CalcUncleHash(uncles []*Header) common.Hash {
	return rlpHash(uncles)
}

// WithMiningResult returns a new block with the data from b
// where nonce and mix digest are set to the provided values.
func (b *Block) WithMiningResult(nonce uint64, mixDigest common.Hash) *Block {
	header := b.header.Raw()
	header.Nonce = EncodeNonce(nonce)
	header.MixDigest = mixDigest
	return &Block{
		header:       &Header{rawHeader: *header},
		transactions: b.transactions,
		receipts:     b.receipts,
		uncles:       b.uncles,
		Td:           b.Td,
	}
}

// WithBody returns a new block with the given transaction and uncle contents.
func (b *Block) WithBody(transactions []*Transaction, uncles []*Header) *Block {
	block := &Block{
		header:       b.header.Copy(),
		transactions: make([]*Transaction, len(transactions)),
		uncles:       make([]*Header, len(uncles)),
	}
	copy(block.transactions, transactions)
	for i := range uncles {
		block.uncles[i] = uncles[i].Copy()
	}
	return block
}

// Implement pow.Block

func (b *Block) String() string {
	str := fmt.Sprintf(`Block(#%v): Size: %v TD: %v {
MinerHash: %x
%v
Transactions:
%v
Uncles:
%v
}
`, b.Number(), b.Size(), b.Td, b.header.HashNoNonce(), b.header, b.transactions, b.uncles)
	return str
}

type Blocks []*Block

type BlockBy func(b1, b2 *Block) bool

func (self BlockBy) Sort(blocks Blocks) {
	bs := blockSorter{
		blocks: blocks,
		by:     self,
	}
	sort.Sort(bs)
}

type blockSorter struct {
	blocks Blocks
	by     func(b1, b2 *Block) bool
}

func (self blockSorter) Len() int { return len(self.blocks) }
func (self blockSorter) Swap(i, j int) {
	self.blocks[i], self.blocks[j] = self.blocks[j], self.blocks[i]
}
func (self blockSorter) Less(i, j int) bool { return self.by(self.blocks[i], self.blocks[j]) }

func Number(b1, b2 *Block) bool { return b1.header.Number().Cmp(b2.header.Number()) < 0 }
