// Copyright 2016 The go-ethereum Authors
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

// Contains all the wrappers from the core/types package.

package geth

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
)

// A Nonce is a 64-bit hash which proves (combined with the mix-hash) that
// a sufficient amount of computation has been carried out on a block.
type Nonce struct {
	nonce types.BlockNonce
}

// GetBytes retrieves the byte representation of the block nonce.
func (n *Nonce) GetBytes() []byte {
	return n.nonce[:]
}

// GetHex retrieves the hex string representation of the block nonce.
func (n *Nonce) GetHex() string {
	return fmt.Sprintf("0x%x", n.nonce[:])
}

// Bloom represents a 256 bit bloom filter.
type Bloom struct {
	bloom types.Bloom
}

// GetBytes retrieves the byte representation of the bloom filter.
func (b *Bloom) GetBytes() []byte {
	return b.bloom[:]
}

// GetHex retrieves the hex string representation of the bloom filter.
func (b *Bloom) GetHex() string {
	return fmt.Sprintf("0x%x", b.bloom[:])
}

// Header represents a block header in the Ethereum blockchain.
type Header struct {
	header *types.Header
}

func (h *Header) GetParentHash() *Hash   { return &Hash{h.header.ParentHash} }
func (h *Header) GetUncleHash() *Hash    { return &Hash{h.header.UncleHash} }
func (h *Header) GetCoinbase() *Address  { return &Address{h.header.Coinbase} }
func (h *Header) GetRoot() *Hash         { return &Hash{h.header.Root} }
func (h *Header) GetTxHash() *Hash       { return &Hash{h.header.TxHash} }
func (h *Header) GetReceiptHash() *Hash  { return &Hash{h.header.ReceiptHash} }
func (h *Header) GetBloom() *Bloom       { return &Bloom{h.header.Bloom} }
func (h *Header) GetDifficulty() *BigInt { return &BigInt{h.header.Difficulty} }
func (h *Header) GetNumber() int64       { return h.header.Number.Int64() }
func (h *Header) GetGasLimit() int64     { return h.header.GasLimit.Int64() }
func (h *Header) GetGasUsed() int64      { return h.header.GasUsed.Int64() }
func (h *Header) GetTime() int64         { return h.header.Time.Int64() }
func (h *Header) GetExtra() []byte       { return h.header.Extra }
func (h *Header) GetMixDigest() *Hash    { return &Hash{h.header.MixDigest} }
func (h *Header) GetNonce() *Nonce       { return &Nonce{h.header.Nonce} }

func (h *Header) GetHash() *Hash        { return &Hash{h.header.Hash()} }
func (h *Header) GetHashNoNonce() *Hash { return &Hash{h.header.HashNoNonce()} }

// Headers represents a slice of headers.
type Headers struct{ headers []*types.Header }

// Size returns the number of headers in the slice.
func (h *Headers) Size() int {
	return len(h.headers)
}

// Get returns the header at the given index from the slice.
func (h *Headers) Get(index int) (*Header, error) {
	if index < 0 || index >= len(h.headers) {
		return nil, errors.New("index out of bounds")
	}
	return &Header{h.headers[index]}, nil
}

// Block represents an entire block in the Ethereum blockchain.
type Block struct {
	block *types.Block
}

func (b *Block) GetParentHash() *Hash   { return &Hash{b.block.ParentHash()} }
func (b *Block) GetUncleHash() *Hash    { return &Hash{b.block.UncleHash()} }
func (b *Block) GetCoinbase() *Address  { return &Address{b.block.Coinbase()} }
func (b *Block) GetRoot() *Hash         { return &Hash{b.block.Root()} }
func (b *Block) GetTxHash() *Hash       { return &Hash{b.block.TxHash()} }
func (b *Block) GetReceiptHash() *Hash  { return &Hash{b.block.ReceiptHash()} }
func (b *Block) GetBloom() *Bloom       { return &Bloom{b.block.Bloom()} }
func (b *Block) GetDifficulty() *BigInt { return &BigInt{b.block.Difficulty()} }
func (b *Block) GetNumber() int64       { return b.block.Number().Int64() }
func (b *Block) GetGasLimit() int64     { return b.block.GasLimit().Int64() }
func (b *Block) GetGasUsed() int64      { return b.block.GasUsed().Int64() }
func (b *Block) GetTime() int64         { return b.block.Time().Int64() }
func (b *Block) GetExtra() []byte       { return b.block.Extra() }
func (b *Block) GetMixDigest() *Hash    { return &Hash{b.block.MixDigest()} }
func (b *Block) GetNonce() int64        { return int64(b.block.Nonce()) }

func (b *Block) GetHash() *Hash        { return &Hash{b.block.Hash()} }
func (b *Block) GetHashNoNonce() *Hash { return &Hash{b.block.HashNoNonce()} }

func (b *Block) GetHeader() *Header             { return &Header{b.block.Header()} }
func (b *Block) GetUncles() *Headers            { return &Headers{b.block.Uncles()} }
func (b *Block) GetTransactions() *Transactions { return &Transactions{b.block.Transactions()} }
func (b *Block) GetTransaction(hash *Hash) *Transaction {
	return &Transaction{b.block.Transaction(hash.hash)}
}

// Transaction represents a single Ethereum transaction.
type Transaction struct {
	tx *types.Transaction
}

func (tx *Transaction) GetData() []byte      { return tx.tx.Data() }
func (tx *Transaction) GetGas() int64        { return tx.tx.Gas().Int64() }
func (tx *Transaction) GetGasPrice() *BigInt { return &BigInt{tx.tx.GasPrice()} }
func (tx *Transaction) GetValue() *BigInt    { return &BigInt{tx.tx.Value()} }
func (tx *Transaction) GetNonce() int64      { return int64(tx.tx.Nonce()) }

func (tx *Transaction) GetHash() *Hash    { return &Hash{tx.tx.Hash()} }
func (tx *Transaction) GetSigHash() *Hash { return &Hash{tx.tx.SigHash()} }
func (tx *Transaction) GetCost() *BigInt  { return &BigInt{tx.tx.Cost()} }

func (tx *Transaction) GetFrom() (*Address, error) {
	from, err := tx.tx.From()
	return &Address{from}, err
}

func (tx *Transaction) GetTo() *Address {
	if to := tx.tx.To(); to != nil {
		return &Address{*to}
	}
	return nil
}

func (tx *Transaction) WithSignature(sig []byte) (*Transaction, error) {
	t, err := tx.tx.WithSignature(sig)
	return &Transaction{t}, err
}

// Transactions represents a slice of transactions.
type Transactions struct{ txs types.Transactions }

// Size returns the number of transactions in the slice.
func (t *Transactions) Size() int {
	return len(t.txs)
}

// Get returns the transaction at the given index from the slice.
func (t *Transactions) Get(index int) (*Transaction, error) {
	if index < 0 || index >= len(t.txs) {
		return nil, errors.New("index out of bounds")
	}
	return &Transaction{t.txs[index]}, nil
}

// Receipt represents the results of a transaction.
type Receipt struct {
	receipt *types.Receipt
}

func (r *Receipt) GetPostState() []byte          { return r.receipt.PostState }
func (r *Receipt) GetCumulativeGasUsed() *BigInt { return &BigInt{r.receipt.CumulativeGasUsed} }
func (r *Receipt) GetBloom() *Bloom              { return &Bloom{r.receipt.Bloom} }
func (r *Receipt) GetLogs() *Logs                { return &Logs{r.receipt.Logs} }
func (r *Receipt) GetTxHash() *Hash              { return &Hash{r.receipt.TxHash} }
func (r *Receipt) GetContractAddress() *Address  { return &Address{r.receipt.ContractAddress} }
func (r *Receipt) GetGasUsed() *BigInt           { return &BigInt{r.receipt.GasUsed} }
