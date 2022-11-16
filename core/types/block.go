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
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	EmptyRootHash  = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	EmptyUncleHash = rlpHash([]*Header(nil))
)

// A BlockNonce is a 64-bit hash which proves (combined with the
// mix-hash) that a sufficient amount of computation has been carried
// out on a block.
type BlockNonce [8]byte

// EncodeNonce converts the given integer to a block nonce.
func EncodeNonce(i uint64) BlockNonce {
	var n BlockNonce
	binary.BigEndian.PutUint64(n[:], i)
	return n
}

// Uint64 returns the integer value of a block nonce.
func (n BlockNonce) Uint64() uint64 {
	return binary.BigEndian.Uint64(n[:])
}

// MarshalText encodes n as a hex string with 0x prefix.
func (n BlockNonce) MarshalText() ([]byte, error) {
	return hexutil.Bytes(n[:]).MarshalText()
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (n *BlockNonce) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("BlockNonce", input, n[:])
}

//go:generate go run github.com/fjl/gencodec -type Header -field-override headerMarshaling -out gen_header_json.go
//go:generate go run ../../rlp/rlpgen -type Header -out gen_header_rlp.go

// Header represents a block header in the Ethereum blockchain.
type Header struct {
	ParentHash  common.Hash    `json:"parentHash"       gencodec:"required"`
	UncleHash   common.Hash    `json:"sha3Uncles"       gencodec:"required"`
	Coinbase    common.Address `json:"miner"`
	Root        common.Hash    `json:"stateRoot"        gencodec:"required"`
	TxHash      common.Hash    `json:"transactionsRoot" gencodec:"required"`
	ReceiptHash common.Hash    `json:"receiptsRoot"     gencodec:"required"`
	Bloom       Bloom          `json:"logsBloom"        gencodec:"required"`
	Difficulty  *big.Int       `json:"difficulty"       gencodec:"required"`
	Number      *big.Int       `json:"number"           gencodec:"required"`
	GasLimit    uint64         `json:"gasLimit"         gencodec:"required"`
	GasUsed     uint64         `json:"gasUsed"          gencodec:"required"`
	Time        uint64         `json:"timestamp"        gencodec:"required"`
	Extra       []byte         `json:"extraData"        gencodec:"required"`
	MixDigest   common.Hash    `json:"mixHash"`
	Nonce       BlockNonce     `json:"nonce"`

	// BaseFee was added by EIP-1559 and is ignored in legacy headers.
	BaseFee *big.Int `json:"baseFeePerGas" rlp:"optional"`

	// WithdrawalsHash was added by EIP-4895 and is ignored in legacy headers.
	WithdrawalsHash *common.Hash `json:"withdrawalsRoot" rlp:"optional"`

	// ExcessDataGas was added by EIP-4844 and is ignored in legacy headers.
	ExcessDataGas *big.Int `json:"excessDataGas" rlp:"optional"`

	/*
		TODO (MariusVanDerWijden) Add this field once needed
		// Random was added during the merge and contains the BeaconState randomness
		Random common.Hash `json:"random" rlp:"optional"`
	*/
}

// field type overrides for gencodec
type headerMarshaling struct {
	Difficulty    *hexutil.Big
	Number        *hexutil.Big
	GasLimit      hexutil.Uint64
	GasUsed       hexutil.Uint64
	Time          hexutil.Uint64
	Extra         hexutil.Bytes
	BaseFee       *hexutil.Big
	ExcessDataGas *hexutil.Big
	Hash          common.Hash `json:"hash"` // adds call to Hash() in MarshalJSON
}

// SetExcessDataGas sets the excess_data_gas field in the header
func (h *Header) SetExcessDataGas(v *big.Int) {
	h.ExcessDataGas = new(big.Int)
	if v != nil {
		h.ExcessDataGas.Set(v)
	}
	if h.WithdrawalsHash == nil {
		// leaving this nil would result in a buggy encoding
		h.WithdrawalsHash = &EmptyRootHash
	}
}

// Hash returns the block hash of the header, which is simply the keccak256 hash of its
// RLP encoding.
func (h *Header) Hash() common.Hash {
	return rlpHash(h)
}

var headerSize = common.StorageSize(reflect.TypeOf(Header{}).Size())

// Size returns the approximate memory used by all internal contents. It is used
// to approximate and limit the memory consumption of various caches.
func (h *Header) Size() common.StorageSize {
	var baseFeeBits int
	if h.BaseFee != nil {
		baseFeeBits = h.BaseFee.BitLen()
	}
	return headerSize + common.StorageSize(len(h.Extra)+(h.Difficulty.BitLen()+h.Number.BitLen()+baseFeeBits)/8)
}

// SanityCheck checks a few basic things -- these checks are way beyond what
// any 'sane' production values should hold, and can mainly be used to prevent
// that the unbounded fields are stuffed with junk data to add processing
// overhead
func (h *Header) SanityCheck() error {
	if h.Number != nil && !h.Number.IsUint64() {
		return fmt.Errorf("too large block number: bitlen %d", h.Number.BitLen())
	}
	if h.Difficulty != nil {
		if diffLen := h.Difficulty.BitLen(); diffLen > 80 {
			return fmt.Errorf("too large block difficulty: bitlen %d", diffLen)
		}
	}
	if eLen := len(h.Extra); eLen > 100*1024 {
		return fmt.Errorf("too large block extradata: size %d", eLen)
	}
	if h.BaseFee != nil {
		if bfLen := h.BaseFee.BitLen(); bfLen > 256 {
			return fmt.Errorf("too large base fee: bitlen %d", bfLen)
		}
	}
	return nil
}

// EmptyBody returns true if there is no additional 'body' to complete the header
// that is: no transactions and no uncles.
func (h *Header) EmptyBody() bool {
	return h.TxHash == EmptyRootHash && h.UncleHash == EmptyUncleHash
}

// EmptyReceipts returns true if there are no receipts for this header/block.
func (h *Header) EmptyReceipts() bool {
	return h.ReceiptHash == EmptyRootHash
}

// Body is a simple (mutable, non-safe) data container for storing and moving
// a block's data contents (transactions and uncles) together.
type Body struct {
	Transactions []*Transaction
	Uncles       []*Header
}

// Block represents an entire block in the Ethereum blockchain.
type Block struct {
	header       *Header
	uncles       []*Header
	transactions Transactions
	withdrawals  []*Withdrawal

	// caches
	hash atomic.Value
	size atomic.Value

	// These fields are used by package eth to track
	// inter-peer block relay.
	ReceivedAt   time.Time
	ReceivedFrom interface{}
}

// a view over a transaction to RLP encode/decode it the minimal way
type minimalTx Transaction

func (tx *minimalTx) DecodeRLP(s *rlp.Stream) error {
	kind, size, err := s.Kind()
	switch {
	case err != nil:
		return err
	case kind == rlp.List:
		// It's a legacy transaction.
		var inner LegacyTx
		err := s.Decode(&inner)
		if err == nil {
			(*Transaction)(tx).setDecoded(&inner, rlp.ListSize(size))
		}
		return err
	case kind == rlp.String:
		// It's an EIP-2718 typed TX envelope.
		var b []byte
		if b, err = s.Bytes(); err != nil {
			return err
		}
		inner, err := (*Transaction)(tx).decodeTypedMinimal(b)
		if err == nil {
			(*Transaction)(tx).setDecoded(inner, uint64(len(b)))
		}
		return err
	default:
		return rlp.ErrExpectedList
	}
}

func (tx *minimalTx) EncodeRLP(w io.Writer) error {
	if (*Transaction)(tx).Type() == LegacyTxType {
		return rlp.Encode(w, tx.inner)
	}
	// It's an EIP-2718 typed TX envelope.
	buf := encodeBufferPool.Get().(*bytes.Buffer)
	defer encodeBufferPool.Put(buf)
	buf.Reset()
	if err := (*Transaction)(tx).encodeTypedMinimal(buf); err != nil {
		return err
	}
	return rlp.Encode(w, buf.Bytes())
}

// a view over a regular transactions slice, to RLP decode/encode the transactions all the minimal way
type extBlockTxs []*Transaction

func (txs *extBlockTxs) DecodeRLP(s *rlp.Stream) error {
	// we need generics to do this nicely...
	var out []*minimalTx
	for i, tx := range *txs {
		out[i] = (*minimalTx)(tx)
	}
	if err := s.Decode(&out); err != nil {
		return fmt.Errorf("failed to decode list of minimal txs: %v", err)
	}
	rawtxs := make([]*Transaction, len(out))
	for i, tx := range out {
		rawtxs[i] = (*Transaction)(tx)
	}
	*txs = rawtxs
	return nil
}

func (txs *extBlockTxs) EncodeRLP(w io.Writer) error {
	out := make([]*minimalTx, len(*txs))
	for i, tx := range *txs {
		out[i] = (*minimalTx)(tx)
	}
	return rlp.Encode(w, &out)
}

// "external" block encoding. used for eth protocol, etc.
type extblock struct {
	Header      *Header
	Txs         *extBlockTxs
	Uncles      []*Header
	Withdrawals []*Withdrawal `rlp:"optional"`
}

// NewBlock creates a new block. The input data is copied,
// changes to header and to the field values will not affect the
// block.
//
// The values of TxHash, UncleHash, ReceiptHash and Bloom in header
// are ignored and set to values derived from the given txs, uncles
// and receipts.
func NewBlock(header *Header, txs []*Transaction, uncles []*Header, receipts []*Receipt, hasher TrieHasher) *Block {
	b := &Block{header: CopyHeader(header)}

	// TODO: panic if len(txs) != len(receipts)
	if len(txs) == 0 {
		b.header.TxHash = EmptyRootHash
	} else {
		b.header.TxHash = DeriveSha(Transactions(txs), hasher)
		b.transactions = make(Transactions, len(txs))
		copy(b.transactions, txs)
	}

	if len(receipts) == 0 {
		b.header.ReceiptHash = EmptyRootHash
	} else {
		b.header.ReceiptHash = DeriveSha(Receipts(receipts), hasher)
		b.header.Bloom = CreateBloom(receipts)
	}

	if len(uncles) == 0 {
		b.header.UncleHash = EmptyUncleHash
	} else {
		b.header.UncleHash = CalcUncleHash(uncles)
		b.uncles = make([]*Header, len(uncles))
		for i := range uncles {
			b.uncles[i] = CopyHeader(uncles[i])
		}
	}

	return b
}

// NewBlock2 creates a new block with withdrawals. The input data
// is copied, changes to header and to the field values will not
// affect the block.
//
// The values of TxHash, UncleHash, ReceiptHash and Bloom in header
// are ignored and set to values derived from the given txs, uncles
// and receipts.
func NewBlock2(header *Header, txs []*Transaction, uncles []*Header, receipts []*Receipt, withdrawals []*Withdrawal, hasher TrieHasher) *Block {
	b := NewBlock(header, txs, uncles, receipts, hasher)
	if withdrawals == nil {
		return b
	}

	b.withdrawals = make([]*Withdrawal, len(withdrawals))
	copy(b.withdrawals, withdrawals)

	if len(withdrawals) == 0 {
		b.header.WithdrawalsHash = &EmptyRootHash
	} else {
		h := DeriveSha(Withdrawals(withdrawals), hasher)
		b.header.WithdrawalsHash = &h
	}
	return b
}

// NewBlockWithHeader creates a block with the given header data. The
// header data is copied, changes to header and to the field values
// will not affect the block.
func NewBlockWithHeader(header *Header) *Block {
	return &Block{header: CopyHeader(header)}
}

// CopyHeader creates a deep copy of a block header to prevent side effects from
// modifying a header variable.
func CopyHeader(h *Header) *Header {
	cpy := *h
	if cpy.Difficulty = new(big.Int); h.Difficulty != nil {
		cpy.Difficulty.Set(h.Difficulty)
	}
	if cpy.Number = new(big.Int); h.Number != nil {
		cpy.Number.Set(h.Number)
	}
	if h.BaseFee != nil {
		cpy.BaseFee = new(big.Int).Set(h.BaseFee)
	}
	if h.ExcessDataGas != nil {
		cpy.SetExcessDataGas(h.ExcessDataGas)
	}
	if len(h.Extra) > 0 {
		cpy.Extra = make([]byte, len(h.Extra))
		copy(cpy.Extra, h.Extra)
	}
	if h.WithdrawalsHash != nil {
		cpy.WithdrawalsHash = new(common.Hash)
		cpy.WithdrawalsHash.SetBytes(h.WithdrawalsHash.Bytes())
	}
	return &cpy
}

// DecodeRLP decodes the Ethereum
func (b *Block) DecodeRLP(s *rlp.Stream) error {
	eb := extblock{Txs: new(extBlockTxs)}
	_, size, _ := s.Kind()
	if err := s.Decode(&eb); err != nil {
		return err
	}
	for i, tx := range *eb.Txs {
		if tx.wrapData != nil {
			return fmt.Errorf("transactions in blocks must not contain wrap-data, tx %d is bad", i)
		}
	}
	b.header, b.uncles, b.transactions, b.withdrawals = eb.Header, eb.Uncles, []*Transaction(*eb.Txs), eb.Withdrawals
	b.size.Store(rlp.ListSize(size))
	return nil
}

// EncodeRLP serializes b into the Ethereum RLP block format.
func (b *Block) EncodeRLP(w io.Writer) error {
	if b.header.ExcessDataGas != nil && b.header.WithdrawalsHash == nil {
		// This situation should not arise, but if it does (due to a bug) you'd silently produce an
		// encoding that would fail to decode. ref:
		// https://github.com/ethereum/go-ethereum/pull/26077
		return errors.New("nil WithdrawalsHash in header with non-nil ExcessDataGas")
	}
	return rlp.Encode(w, extblock{
		Header:      b.header,
		Txs:         (*extBlockTxs)(&b.transactions),
		Uncles:      b.uncles,
		Withdrawals: b.withdrawals,
	})
}

// TODO: copies

func (b *Block) Uncles() []*Header          { return b.uncles }
func (b *Block) Transactions() Transactions { return b.transactions }

func (b *Block) Transaction(hash common.Hash) *Transaction {
	for _, transaction := range b.transactions {
		if transaction.Hash() == hash {
			return transaction
		}
	}
	return nil
}

func (b *Block) Number() *big.Int     { return new(big.Int).Set(b.header.Number) }
func (b *Block) GasLimit() uint64     { return b.header.GasLimit }
func (b *Block) GasUsed() uint64      { return b.header.GasUsed }
func (b *Block) Difficulty() *big.Int { return new(big.Int).Set(b.header.Difficulty) }
func (b *Block) Time() uint64         { return b.header.Time }

func (b *Block) NumberU64() uint64        { return b.header.Number.Uint64() }
func (b *Block) MixDigest() common.Hash   { return b.header.MixDigest }
func (b *Block) Nonce() uint64            { return binary.BigEndian.Uint64(b.header.Nonce[:]) }
func (b *Block) Bloom() Bloom             { return b.header.Bloom }
func (b *Block) Coinbase() common.Address { return b.header.Coinbase }
func (b *Block) Root() common.Hash        { return b.header.Root }
func (b *Block) ParentHash() common.Hash  { return b.header.ParentHash }
func (b *Block) TxHash() common.Hash      { return b.header.TxHash }
func (b *Block) ReceiptHash() common.Hash { return b.header.ReceiptHash }
func (b *Block) UncleHash() common.Hash   { return b.header.UncleHash }
func (b *Block) Extra() []byte            { return common.CopyBytes(b.header.Extra) }

func (b *Block) BaseFee() *big.Int {
	if b.header.BaseFee == nil {
		return nil
	}
	return new(big.Int).Set(b.header.BaseFee)
}

func (b *Block) WithdrawalsHash() *common.Hash {
	if b.header.WithdrawalsHash == nil {
		return nil
	}
	var h common.Hash
	h.SetBytes(b.header.WithdrawalsHash.Bytes())
	return &h
}

func (b *Block) Withdrawals() Withdrawals {
	return b.withdrawals
}

func (b *Block) ExcessDataGas() *big.Int {
	if b.header.ExcessDataGas == nil {
		return nil
	}
	return new(big.Int).Set(b.header.ExcessDataGas)
}

func (b *Block) Header() *Header { return CopyHeader(b.header) }

// Body returns the non-header content of the block.
func (b *Block) Body() *Body { return &Body{b.transactions, b.uncles} }

// Size returns the true RLP encoded storage size of the block, either by encoding
// and returning it, or returning a previously cached value.
func (b *Block) Size() uint64 {
	if size := b.size.Load(); size != nil {
		return size.(uint64)
	}
	c := writeCounter(0)
	rlp.Encode(&c, b)
	b.size.Store(uint64(c))
	return uint64(c)
}

// SanityCheck can be used to prevent that unbounded fields are
// stuffed with junk data to add processing overhead
func (b *Block) SanityCheck() error {
	return b.header.SanityCheck()
}

type writeCounter uint64

func (c *writeCounter) Write(b []byte) (int, error) {
	*c += writeCounter(len(b))
	return len(b), nil
}

func CalcUncleHash(uncles []*Header) common.Hash {
	if len(uncles) == 0 {
		return EmptyUncleHash
	}
	return rlpHash(uncles)
}

// WithSeal returns a new block with the data from b but the header replaced with
// the sealed one.
func (b *Block) WithSeal(header *Header) *Block {
	cpy := *header

	return &Block{
		header:       &cpy,
		transactions: b.transactions,
		uncles:       b.uncles,
		withdrawals:  b.withdrawals,
	}
}

// WithBody returns a new block with the given transaction and uncle contents.
func (b *Block) WithBody(transactions []*Transaction, uncles []*Header) *Block {
	block := &Block{
		header:       CopyHeader(b.header),
		transactions: make([]*Transaction, len(transactions)),
		uncles:       make([]*Header, len(uncles)),
	}
	copy(block.transactions, transactions)
	for i := range uncles {
		block.uncles[i] = CopyHeader(uncles[i])
	}
	return block
}

// WithBody2 returns a new block with the given transaction, uncle, and
// withdrawal contents.
func (b *Block) WithBody2(transactions []*Transaction, uncles []*Header, withdrawals []*Withdrawal) *Block {
	block := b.WithBody(transactions, uncles)
	if withdrawals == nil {
		return block
	}
	block.withdrawals = make([]*Withdrawal, len(withdrawals))
	copy(block.withdrawals, withdrawals)
	return block
}

// Hash returns the keccak256 hash of b's header.
// The hash is computed on the first call and cached thereafter.
func (b *Block) Hash() common.Hash {
	if hash := b.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := b.header.Hash()
	b.hash.Store(v)
	return v
}

type Blocks []*Block

// HeaderParentHashFromRLP returns the parentHash of an RLP-encoded
// header. If 'header' is invalid, the zero hash is returned.
func HeaderParentHashFromRLP(header []byte) common.Hash {
	// parentHash is the first list element.
	listContent, _, err := rlp.SplitList(header)
	if err != nil {
		return common.Hash{}
	}
	parentHash, _, err := rlp.SplitString(listContent)
	if err != nil {
		return common.Hash{}
	}
	if len(parentHash) != 32 {
		return common.Hash{}
	}
	return common.BytesToHash(parentHash)
}
