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

// Contains the block header type and related methods.

package types

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
)

// A BlockNonce is a 64-bit hash which proves (combined with the
// mix-hash) that a suffcient amount of computation has been carried
// out on a block.
type BlockNonce [8]byte

func EncodeNonce(i uint64) BlockNonce {
	var n BlockNonce
	binary.BigEndian.PutUint64(n[:], i)
	return n
}

func (n BlockNonce) Uint64() uint64 {
	return binary.BigEndian.Uint64(n[:])
}

// RawHeader is the base consensus header used by the Ethereum system, which
// contains only the bare essential fields defined by the consensus protocol.
//
// This structure is mutable, and can be used to create an immutable header
// used throughout the codebase. Do not pass this structure around, only use
// to aid in new header construction or data query from an existing header.
type RawHeader struct {
	ParentHash  common.Hash    // Hash of the previous block in the chain
	UncleHash   common.Hash    // Hash of the uncles contained in this block
	Coinbase    common.Address // Owner address of the block (miner)
	Root        common.Hash    // Merkle-Patricia state trie root hash
	TxHash      common.Hash    // Hash of the transactions contained in this block
	ReceiptHash common.Hash    // Hash of the transactions receipts contained in this block
	Bloom       Bloom          // Bloom filter for the event logs in the block
	Difficulty  *big.Int       // Proof-of-work difficulty of this block
	Number      *big.Int       // Index number of the block within the chain
	GasLimit    *big.Int       // Maximum gas allowance for this block
	GasUsed     *big.Int       // Amount of gas actually used in this block
	Time        *big.Int       // Creation timestamp of this block
	Extra       []byte         // Extra data inserted into the block by the miner
	MixDigest   common.Hash    // Miner mix digest for quick difficulty verification
	Nonce       BlockNonce     // Proof-of-work nonce of the block
}

// hash calculates the nonce-inclusive hash of the header. As the raw header is
// mutable, this hasher cannot cache its expensive operation, hence it's private
// and users are required to use the immutable Header's Hash method.
func (h *RawHeader) hash() common.Hash {
	return rlpHash(h)
}

// hashNoNonce calculates the miner hash of the header (no mix digest or nonce).
// Similarly to the hash method, as the raw header is mutable, this hasher also
// cannot cache its expensive operation, hence it's private and users are asked
// to use the immutable Header's HashNoNonce method.
func (h *RawHeader) hashNoNonce() common.Hash {
	return rlpHash([]interface{}{
		h.ParentHash,
		h.UncleHash,
		h.Coinbase,
		h.Root,
		h.TxHash,
		h.ReceiptHash,
		h.Bloom,
		h.Difficulty,
		h.Number,
		h.GasLimit,
		h.GasUsed,
		h.Time,
		h.Extra,
	})
}

// Copy creates a deep copy of a raw header.
func (h *RawHeader) Copy() *RawHeader {
	cpy := *h
	if cpy.Difficulty = new(big.Int); h.Difficulty != nil {
		cpy.Difficulty.Set(h.Difficulty)
	}
	if cpy.Number = new(big.Int); h.Number != nil {
		cpy.Number.Set(h.Number)
	}
	if cpy.GasLimit = new(big.Int); h.GasLimit != nil {
		cpy.GasLimit.Set(h.GasLimit)
	}
	if cpy.GasUsed = new(big.Int); h.GasUsed != nil {
		cpy.GasUsed.Set(h.GasUsed)
	}
	if cpy.Time = new(big.Int); h.Time != nil {
		cpy.Time.Set(h.Time)
	}
	if len(h.Extra) > 0 {
		cpy.Extra = common.CopyBytes(h.Extra)
	}
	return &cpy
}

// Header is the immutable Ethereum block header, containing all of the metadata
// related to a chain block, and some additional helper fields, caches.
type Header struct {
	rawHeader RawHeader // Base header fields part of the consensus protocol

	hashNoNonce   atomic.Value // Cached hash of the header without the nonce
	hashWithNonce atomic.Value // Cached hash of the header with the nonce
	rlpSize       atomic.Value // Size of the header in RLP encoded format
}

// NewHeader creates an immutable header from a mutable raw header.
func NewHeader(raw *RawHeader) *Header {
	return &Header{
		rawHeader: *raw.Copy(),
	}
}

// Raw creates and returns a mutable deep copy of the raw consensus header.
func (h *Header) Raw() *RawHeader {
	return h.rawHeader.Copy()
}

// ParentHash retrieves the hash of this block's parent in the blockchain.
func (h *Header) ParentHash() common.Hash { return h.rawHeader.ParentHash }

// UncleHash retrieves the hash of the uncle blocks contained within this block.
func (h *Header) UncleHash() common.Hash { return h.rawHeader.UncleHash }

// Coinbase retrieves the account owning this block (i.e. miner).
func (h *Header) Coinbase() common.Address { return h.rawHeader.Coinbase }

// Root retrieves the root hash of the Merkle-Patricia state trie formed.
func (h *Header) Root() common.Hash { return h.rawHeader.Root }

// TxHash retrieves the hash of the transactions contained within this block.
func (h *Header) TxHash() common.Hash { return h.rawHeader.TxHash }

// ReceiptHash retrieves the hash of the transaction receipts contained within.
func (h *Header) ReceiptHash() common.Hash { return h.rawHeader.ReceiptHash }

// Bloom retrieves the bloom filter of the event logs contained within.
func (h *Header) Bloom() Bloom { return h.rawHeader.Bloom }

// Difficulty retrieves the proof-of-work difficulty of this block.
func (h *Header) Difficulty() *big.Int { return new(big.Int).Set(h.rawHeader.Difficulty) }

// Number retrieves the index/number of the block within the chain.
func (h *Header) Number() *big.Int { return new(big.Int).Set(h.rawHeader.Number) }

// GasLimit retrieves the maximum gas allowance for this block.
func (h *Header) GasLimit() *big.Int { return new(big.Int).Set(h.rawHeader.GasLimit) }

// GasUsed retrieves the amount of gas actually used in this block.
func (h *Header) GasUsed() *big.Int { return new(big.Int).Set(h.rawHeader.GasUsed) }

// Time retrieves the creation timestamp of this block.
func (h *Header) Time() *big.Int { return new(big.Int).Set(h.rawHeader.Time) }

// Extra retrieves the extra data inserted into the block by the miner.
func (h *Header) Extra() []byte {
	return common.CopyBytes(h.rawHeader.Extra)
}

// MixDigest retrieves the miner mix digest for quick difficulty verification.
func (h *Header) MixDigest() common.Hash { return h.rawHeader.MixDigest }

// Nonce retrieves the proof-of-work nonce of the block.
func (h *Header) Nonce() BlockNonce { return h.rawHeader.Nonce }

// RlpSize retrieves the RLP encoded size of the header, calculating and caching
// if it unknown.
func (h *Header) RlpSize() common.StorageSize {
	if size := h.rlpSize.Load(); size != nil && size.(common.StorageSize).Int64() != 0 {
		return size.(common.StorageSize)
	}
	counter := writeCounter(0)
	rlp.Encode(&counter, h.rawHeader)

	h.rlpSize.Store(common.StorageSize(counter))
	return common.StorageSize(counter)
}

// Hash retrieves the nonce-inclusive hash of the header, calculating and caching
// it if unknown.
func (h *Header) Hash() common.Hash {
	if hash := h.hashWithNonce.Load(); hash != nil && hash.(common.Hash) != (common.Hash{}) {
		return hash.(common.Hash)
	}
	hash := h.rawHeader.hash()
	h.hashWithNonce.Store(hash)
	return hash
}

// HashNoNonce retrieves the miner hash of the header (no mix digest or nonce),
// calculating and caching it if unknown.
func (h *Header) HashNoNonce() common.Hash {
	if hash := h.hashNoNonce.Load(); hash != nil && hash.(common.Hash) != (common.Hash{}) {
		return hash.(common.Hash)
	}
	hash := h.rawHeader.hashNoNonce()
	h.hashNoNonce.Store(hash)
	return hash
}

// DecodeRLP implements the rlp.Decoder interface, deserializing a raw header
// from an RLP Data stream.
func (h *Header) DecodeRLP(s *rlp.Stream) error {
	// Reset any previously cached fields
	h.hashNoNonce.Store(common.Hash{})
	h.hashWithNonce.Store(common.Hash{})
	h.rlpSize.Store(common.StorageSize(0))

	// Fill in the real RLP encoded size of the header
	_, size, _ := s.Kind()
	h.rlpSize.Store(common.StorageSize(rlp.ListSize(size)))

	// Decode the RLP encoded raw header
	return s.Decode(&h.rawHeader)
}

// EncodeRLP implements the rlp.Encoder interface, serializing a raw header into
// and RLP data stream.
func (h *Header) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, h.rawHeader)
}

func (h *Header) UnmarshalJSON(data []byte) error {
	var ext struct {
		ParentHash string
		Coinbase   string
		Difficulty string
		GasLimit   string
		Time       *big.Int
		Extra      string
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&ext); err != nil {
		return err
	}
	h.rawHeader.ParentHash = common.HexToHash(ext.ParentHash)
	h.rawHeader.Coinbase = common.HexToAddress(ext.Coinbase)
	h.rawHeader.Difficulty = common.String2Big(ext.Difficulty)
	h.rawHeader.Time = ext.Time
	h.rawHeader.Extra = []byte(ext.Extra)

	return nil
}

// Copy creates a deep copy of a header.
func (h *Header) Copy() *Header {
	cpy := new(Header)
	cpy.rawHeader = *h.rawHeader.Copy()
	return cpy
}

// String implements the fmt.Stringer interface, formatting a header into a string.
func (h *Header) String() string {
	return fmt.Sprintf(`Header(%x):
[
	ParentHash:	    %x
	UncleHash:	    %x
	Coinbase:	    %x
	Root:		    %x
	TxSha:		    %x
	ReceiptSha:	    %x
	Bloom:		    %x
	Difficulty:	    %v
	Number:		    %v
	GasLimit:	    %v
	GasUsed:	    %v
	Time:		    %v
	Extra:		    %s
	MixDigest:	    %x
	Nonce:		    %x
]`, h.Hash(), h.ParentHash(), h.UncleHash(), h.Coinbase(), h.Root(), h.TxHash(), h.ReceiptHash(), h.Bloom(), h.Difficulty(), h.Number(), h.GasLimit(), h.GasUsed(), h.Time(), h.Extra(), h.MixDigest(), h.Nonce())
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}
