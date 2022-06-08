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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

type jsonEncoder interface {
	EncodeJSON() (string, error)
}

// encodeOrError tries to encode the object into json.
// If the encoding fails the resulting error is returned.
func encodeOrError(encoder jsonEncoder) string {
	enc, err := encoder.EncodeJSON()
	if err != nil {
		return err.Error()
	}
	return enc
}

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

// String returns a printable representation of the nonce.
func (n *Nonce) String() string {
	return n.GetHex()
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

// String returns a printable representation of the bloom filter.
func (b *Bloom) String() string {
	return b.GetHex()
}

// Header represents a block header in the Ethereum blockchain.
type Header struct {
	header *types.Header
}

// NewHeaderFromRLP parses a header from an RLP data dump.
func NewHeaderFromRLP(data []byte) (*Header, error) {
	h := &Header{
		header: new(types.Header),
	}
	if err := rlp.DecodeBytes(common.CopyBytes(data), h.header); err != nil {
		return nil, err
	}
	return h, nil
}

// EncodeRLP encodes a header into an RLP data dump.
func (h *Header) EncodeRLP() ([]byte, error) {
	return rlp.EncodeToBytes(h.header)
}

// NewHeaderFromJSON parses a header from a JSON data dump.
func NewHeaderFromJSON(data string) (*Header, error) {
	h := &Header{
		header: new(types.Header),
	}
	if err := json.Unmarshal([]byte(data), h.header); err != nil {
		return nil, err
	}
	return h, nil
}

// EncodeJSON encodes a header into a JSON data dump.
func (h *Header) EncodeJSON() (string, error) {
	data, err := json.Marshal(h.header)
	return string(data), err
}

// String returns a printable representation of the header.
func (h *Header) String() string {
	return encodeOrError(h)
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
func (h *Header) GetGasLimit() int64     { return int64(h.header.GasLimit) }
func (h *Header) GetGasUsed() int64      { return int64(h.header.GasUsed) }
func (h *Header) GetTime() int64         { return int64(h.header.Time) }
func (h *Header) GetExtra() []byte       { return h.header.Extra }
func (h *Header) GetMixDigest() *Hash    { return &Hash{h.header.MixDigest} }
func (h *Header) GetNonce() *Nonce       { return &Nonce{h.header.Nonce} }
func (h *Header) GetHash() *Hash         { return &Hash{h.header.Hash()} }

// Headers represents a slice of headers.
type Headers struct{ headers []*types.Header }

// Size returns the number of headers in the slice.
func (h *Headers) Size() int {
	return len(h.headers)
}

// Get returns the header at the given index from the slice.
func (h *Headers) Get(index int) (header *Header, _ error) {
	if index < 0 || index >= len(h.headers) {
		return nil, errors.New("index out of bounds")
	}
	return &Header{h.headers[index]}, nil
}

// Block represents an entire block in the Ethereum blockchain.
type Block struct {
	block *types.Block
}

// NewBlockFromRLP parses a block from an RLP data dump.
func NewBlockFromRLP(data []byte) (*Block, error) {
	b := &Block{
		block: new(types.Block),
	}
	if err := rlp.DecodeBytes(common.CopyBytes(data), b.block); err != nil {
		return nil, err
	}
	return b, nil
}

// EncodeRLP encodes a block into an RLP data dump.
func (b *Block) EncodeRLP() ([]byte, error) {
	return rlp.EncodeToBytes(b.block)
}

// NewBlockFromJSON parses a block from a JSON data dump.
func NewBlockFromJSON(data string) (*Block, error) {
	b := &Block{
		block: new(types.Block),
	}
	if err := json.Unmarshal([]byte(data), b.block); err != nil {
		return nil, err
	}
	return b, nil
}

// EncodeJSON encodes a block into a JSON data dump.
func (b *Block) EncodeJSON() (string, error) {
	data, err := json.Marshal(b.block)
	return string(data), err
}

// String returns a printable representation of the block.
func (b *Block) String() string {
	return encodeOrError(b)
}

func (b *Block) GetParentHash() *Hash           { return &Hash{b.block.ParentHash()} }
func (b *Block) GetUncleHash() *Hash            { return &Hash{b.block.UncleHash()} }
func (b *Block) GetCoinbase() *Address          { return &Address{b.block.Coinbase()} }
func (b *Block) GetRoot() *Hash                 { return &Hash{b.block.Root()} }
func (b *Block) GetTxHash() *Hash               { return &Hash{b.block.TxHash()} }
func (b *Block) GetReceiptHash() *Hash          { return &Hash{b.block.ReceiptHash()} }
func (b *Block) GetBloom() *Bloom               { return &Bloom{b.block.Bloom()} }
func (b *Block) GetDifficulty() *BigInt         { return &BigInt{b.block.Difficulty()} }
func (b *Block) GetNumber() int64               { return b.block.Number().Int64() }
func (b *Block) GetGasLimit() int64             { return int64(b.block.GasLimit()) }
func (b *Block) GetGasUsed() int64              { return int64(b.block.GasUsed()) }
func (b *Block) GetTime() int64                 { return int64(b.block.Time()) }
func (b *Block) GetExtra() []byte               { return b.block.Extra() }
func (b *Block) GetMixDigest() *Hash            { return &Hash{b.block.MixDigest()} }
func (b *Block) GetNonce() int64                { return int64(b.block.Nonce()) }
func (b *Block) GetHash() *Hash                 { return &Hash{b.block.Hash()} }
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

// NewContractCreation creates a new transaction for deploying a new contract with
// the given properties.
func NewContractCreation(nonce int64, amount *BigInt, gasLimit int64, gasPrice *BigInt, data []byte) *Transaction {
	return &Transaction{types.NewContractCreation(uint64(nonce), amount.bigint, uint64(gasLimit), gasPrice.bigint, common.CopyBytes(data))}
}

// NewTransaction creates a new transaction with the given properties. Contracts
// can be created by transacting with a nil recipient.
func NewTransaction(nonce int64, to *Address, amount *BigInt, gasLimit int64, gasPrice *BigInt, data []byte) *Transaction {
	if to == nil {
		return &Transaction{types.NewContractCreation(uint64(nonce), amount.bigint, uint64(gasLimit), gasPrice.bigint, common.CopyBytes(data))}
	}
	return &Transaction{types.NewTransaction(uint64(nonce), to.address, amount.bigint, uint64(gasLimit), gasPrice.bigint, common.CopyBytes(data))}
}

// NewTransactionFromRLP parses a transaction from an RLP data dump.
func NewTransactionFromRLP(data []byte) (*Transaction, error) {
	tx := &Transaction{
		tx: new(types.Transaction),
	}
	if err := rlp.DecodeBytes(common.CopyBytes(data), tx.tx); err != nil {
		return nil, err
	}
	return tx, nil
}

// EncodeRLP encodes a transaction into an RLP data dump.
func (tx *Transaction) EncodeRLP() ([]byte, error) {
	return rlp.EncodeToBytes(tx.tx)
}

// NewTransactionFromJSON parses a transaction from a JSON data dump.
func NewTransactionFromJSON(data string) (*Transaction, error) {
	tx := &Transaction{
		tx: new(types.Transaction),
	}
	if err := json.Unmarshal([]byte(data), tx.tx); err != nil {
		return nil, err
	}
	return tx, nil
}

// EncodeJSON encodes a transaction into a JSON data dump.
func (tx *Transaction) EncodeJSON() (string, error) {
	data, err := json.Marshal(tx.tx)
	return string(data), err
}

// String returns a printable representation of the transaction.
func (tx *Transaction) String() string {
	return encodeOrError(tx)
}

func (tx *Transaction) GetData() []byte      { return tx.tx.Data() }
func (tx *Transaction) GetGas() int64        { return int64(tx.tx.Gas()) }
func (tx *Transaction) GetGasPrice() *BigInt { return &BigInt{tx.tx.GasPrice()} }
func (tx *Transaction) GetValue() *BigInt    { return &BigInt{tx.tx.Value()} }
func (tx *Transaction) GetNonce() int64      { return int64(tx.tx.Nonce()) }

func (tx *Transaction) GetHash() *Hash   { return &Hash{tx.tx.Hash()} }
func (tx *Transaction) GetCost() *BigInt { return &BigInt{tx.tx.Cost()} }

func (tx *Transaction) GetTo() *Address {
	if to := tx.tx.To(); to != nil {
		return &Address{*to}
	}
	return nil
}

func (tx *Transaction) WithSignature(sig []byte, chainID *BigInt) (signedTx *Transaction, _ error) {
	var signer types.Signer = types.HomesteadSigner{}
	if chainID != nil {
		signer = types.NewEIP155Signer(chainID.bigint)
	}
	rawTx, err := tx.tx.WithSignature(signer, common.CopyBytes(sig))
	return &Transaction{rawTx}, err
}

// Transactions represents a slice of transactions.
type Transactions struct{ txs types.Transactions }

// Size returns the number of transactions in the slice.
func (txs *Transactions) Size() int {
	return len(txs.txs)
}

// Get returns the transaction at the given index from the slice.
func (txs *Transactions) Get(index int) (tx *Transaction, _ error) {
	if index < 0 || index >= len(txs.txs) {
		return nil, errors.New("index out of bounds")
	}
	return &Transaction{txs.txs[index]}, nil
}

// Receipt represents the results of a transaction.
type Receipt struct {
	receipt *types.Receipt
}

// NewReceiptFromRLP parses a transaction receipt from an RLP data dump.
func NewReceiptFromRLP(data []byte) (*Receipt, error) {
	r := &Receipt{
		receipt: new(types.Receipt),
	}
	if err := rlp.DecodeBytes(common.CopyBytes(data), r.receipt); err != nil {
		return nil, err
	}
	return r, nil
}

// EncodeRLP encodes a transaction receipt into an RLP data dump.
func (r *Receipt) EncodeRLP() ([]byte, error) {
	return rlp.EncodeToBytes(r.receipt)
}

// NewReceiptFromJSON parses a transaction receipt from a JSON data dump.
func NewReceiptFromJSON(data string) (*Receipt, error) {
	r := &Receipt{
		receipt: new(types.Receipt),
	}
	if err := json.Unmarshal([]byte(data), r.receipt); err != nil {
		return nil, err
	}
	return r, nil
}

// EncodeJSON encodes a transaction receipt into a JSON data dump.
func (r *Receipt) EncodeJSON() (string, error) {
	data, err := json.Marshal(r.receipt)
	return string(data), err
}

// String returns a printable representation of the receipt.
func (r *Receipt) String() string {
	return encodeOrError(r)
}

func (r *Receipt) GetStatus() int               { return int(r.receipt.Status) }
func (r *Receipt) GetPostState() []byte         { return r.receipt.PostState }
func (r *Receipt) GetCumulativeGasUsed() int64  { return int64(r.receipt.CumulativeGasUsed) }
func (r *Receipt) GetBloom() *Bloom             { return &Bloom{r.receipt.Bloom} }
func (r *Receipt) GetLogs() *Logs               { return &Logs{r.receipt.Logs} }
func (r *Receipt) GetTxHash() *Hash             { return &Hash{r.receipt.TxHash} }
func (r *Receipt) GetContractAddress() *Address { return &Address{r.receipt.ContractAddress} }
func (r *Receipt) GetGasUsed() int64            { return int64(r.receipt.GasUsed) }
