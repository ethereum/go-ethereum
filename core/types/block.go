package types

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"sort"
	"sync/atomic"
	"time"

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

type Header struct {
	ParentHash  common.Hash    // Hash to the previous block
	UncleHash   common.Hash    // Uncles of this block
	Coinbase    common.Address // The coin base address
	Root        common.Hash    // Block Trie state
	TxHash      common.Hash    // Tx sha
	ReceiptHash common.Hash    // Receipt sha
	Bloom       Bloom          // Bloom
	Difficulty  *big.Int       // Difficulty for the current block
	Number      *big.Int       // The block number
	GasLimit    *big.Int       // Gas limit
	GasUsed     *big.Int       // Gas used
	Time        uint64         // Creation time
	Extra       []byte         // Extra data
	MixDigest   common.Hash    // for quick difficulty verification
	Nonce       BlockNonce
}

func (h *Header) Hash() common.Hash {
	return rlpHash(h)
}

func (h *Header) HashNoNonce() common.Hash {
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

func (h *Header) UnmarshalJSON(data []byte) error {
	var ext struct {
		ParentHash string
		Coinbase   string
		Difficulty string
		GasLimit   string
		Time       uint64
		Extra      string
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&ext); err != nil {
		return err
	}

	h.ParentHash = common.HexToHash(ext.ParentHash)
	h.Coinbase = common.HexToAddress(ext.Coinbase)
	h.Difficulty = common.String2Big(ext.Difficulty)
	h.Time = ext.Time
	h.Extra = []byte(ext.Extra)
	return nil
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

type Block struct {
	header       *Header
	uncles       []*Header
	transactions Transactions
	receipts     Receipts

	// caches
	hash atomic.Value
	size atomic.Value

	// Td is used by package core to store the total difficulty
	// of the chain up to and including the block.
	Td *big.Int

	// ReceivedAt is used by package eth to track block propagation time.
	ReceivedAt time.Time
}

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
func NewBlock(header *Header, txs []*Transaction, uncles []*Header, receipts []*Receipt) *Block {
	b := &Block{header: copyHeader(header), Td: new(big.Int)}

	// TODO: panic if len(txs) != len(receipts)
	if len(txs) == 0 {
		b.header.TxHash = emptyRootHash
	} else {
		b.header.TxHash = DeriveSha(Transactions(txs))
		b.transactions = make(Transactions, len(txs))
		copy(b.transactions, txs)
	}

	if len(receipts) == 0 {
		b.header.ReceiptHash = emptyRootHash
	} else {
		b.header.ReceiptHash = DeriveSha(Receipts(receipts))
		b.header.Bloom = CreateBloom(receipts)
		b.receipts = make([]*Receipt, len(receipts))
		copy(b.receipts, receipts)
	}

	if len(uncles) == 0 {
		b.header.UncleHash = emptyUncleHash
	} else {
		b.header.UncleHash = CalcUncleHash(uncles)
		b.uncles = make([]*Header, len(uncles))
		for i := range uncles {
			b.uncles[i] = copyHeader(uncles[i])
		}
	}

	return b
}

// NewBlockWithHeader creates a block with the given header data. The
// header data is copied, changes to header and to the field values
// will not affect the block.
func NewBlockWithHeader(header *Header) *Block {
	return &Block{header: copyHeader(header)}
}

func copyHeader(h *Header) *Header {
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
	if len(h.Extra) > 0 {
		cpy.Extra = make([]byte, len(h.Extra))
		copy(cpy.Extra, h.Extra)
	}
	return &cpy
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

func (b Block) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, extblock{
		Header: b.header,
		Txs:    b.transactions,
		Uncles: b.uncles,
	})
}

func (b *StorageBlock) DecodeRLP(s *rlp.Stream) error {
	var sb storageblock
	if err := s.Decode(&sb); err != nil {
		return err
	}
	b.header, b.uncles, b.transactions, b.Td = sb.Header, sb.Uncles, sb.Txs, sb.TD
	return nil
}

func (b StorageBlock) EncodeRLP(w io.Writer) error {
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

func (b *Block) Number() *big.Int     { return new(big.Int).Set(b.header.Number) }
func (b *Block) GasLimit() *big.Int   { return new(big.Int).Set(b.header.GasLimit) }
func (b *Block) GasUsed() *big.Int    { return new(big.Int).Set(b.header.GasUsed) }
func (b *Block) Difficulty() *big.Int { return new(big.Int).Set(b.header.Difficulty) }

func (b *Block) NumberU64() uint64        { return b.header.Number.Uint64() }
func (b *Block) MixDigest() common.Hash   { return b.header.MixDigest }
func (b *Block) Nonce() uint64            { return binary.BigEndian.Uint64(b.header.Nonce[:]) }
func (b *Block) Bloom() Bloom             { return b.header.Bloom }
func (b *Block) Coinbase() common.Address { return b.header.Coinbase }
func (b *Block) Time() uint64             { return b.header.Time }
func (b *Block) Root() common.Hash        { return b.header.Root }
func (b *Block) ParentHash() common.Hash  { return b.header.ParentHash }
func (b *Block) TxHash() common.Hash      { return b.header.TxHash }
func (b *Block) ReceiptHash() common.Hash { return b.header.ReceiptHash }
func (b *Block) UncleHash() common.Hash   { return b.header.UncleHash }
func (b *Block) Extra() []byte            { return common.CopyBytes(b.header.Extra) }

func (b *Block) Header() *Header { return copyHeader(b.header) }

func (b *Block) HashNoNonce() common.Hash {
	return b.header.HashNoNonce()
}

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
	cpy := *b.header
	binary.BigEndian.PutUint64(cpy.Nonce[:], nonce)
	cpy.MixDigest = mixDigest
	return &Block{
		header:       &cpy,
		transactions: b.transactions,
		receipts:     b.receipts,
		uncles:       b.uncles,
		Td:           b.Td,
	}
}

// Implement pow.Block

func (b *Block) Hash() common.Hash {
	if hash := b.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := rlpHash(b.header)
	b.hash.Store(v)
	return v
}

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

func (h *Header) String() string {
	return fmt.Sprintf(`Header(%x):
[
	ParentHash:	    %x
	UncleHash:	    %x
	Coinbase:	    %x
	Root:		    %x
	TxSha		    %x
	ReceiptSha:	    %x
	Bloom:		    %x
	Difficulty:	    %v
	Number:		    %v
	GasLimit:	    %v
	GasUsed:	    %v
	Time:		    %v
	Extra:		    %s
	MixDigest:      %x
	Nonce:		    %x
]`, h.Hash(), h.ParentHash, h.UncleHash, h.Coinbase, h.Root, h.TxHash, h.ReceiptHash, h.Bloom, h.Difficulty, h.Number, h.GasLimit, h.GasUsed, h.Time, h.Extra, h.MixDigest, h.Nonce)
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

func Number(b1, b2 *Block) bool { return b1.header.Number.Cmp(b2.header.Number) < 0 }
