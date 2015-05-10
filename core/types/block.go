package types

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
)

type Header struct {
	// Hash to the previous block
	ParentHash common.Hash
	// Uncles of this block
	UncleHash common.Hash
	// The coin base address
	Coinbase common.Address
	// Block Trie state
	Root common.Hash
	// Tx sha
	TxHash common.Hash
	// Receipt sha
	ReceiptHash common.Hash
	// Bloom
	Bloom Bloom
	// Difficulty for the current block
	Difficulty *big.Int
	// The block number
	Number *big.Int
	// Gas limit
	GasLimit *big.Int
	// Gas used
	GasUsed *big.Int
	// Creation time
	Time uint64
	// Extra data
	Extra []byte
	// Mix digest for quick checking to prevent DOS
	MixDigest common.Hash
	// Nonce
	Nonce [8]byte
}

func (self *Header) Hash() common.Hash {
	return rlpHash(self.rlpData(true))
}

func (self *Header) HashNoNonce() common.Hash {
	return rlpHash(self.rlpData(false))
}

func (self *Header) rlpData(withNonce bool) []interface{} {
	fields := []interface{}{
		self.ParentHash,
		self.UncleHash,
		self.Coinbase,
		self.Root,
		self.TxHash,
		self.ReceiptHash,
		self.Bloom,
		self.Difficulty,
		self.Number,
		self.GasLimit,
		self.GasUsed,
		self.Time,
		self.Extra,
	}
	if withNonce {
		fields = append(fields, self.MixDigest, self.Nonce)
	}
	return fields
}

func (self *Header) RlpData() interface{} {
	return self.rlpData(true)
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

type Block struct {
	// Preset Hash for mock (Tests)
	HeaderHash       common.Hash
	ParentHeaderHash common.Hash
	// ^^^^ ignore ^^^^

	header       *Header
	uncles       []*Header
	transactions Transactions
	Td           *big.Int
	queued       bool // flag for blockpool to skip TD check

	ReceivedAt time.Time

	receipts Receipts
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

func NewBlock(parentHash common.Hash, coinbase common.Address, root common.Hash, difficulty *big.Int, nonce uint64, extra []byte) *Block {
	header := &Header{
		Root:       root,
		ParentHash: parentHash,
		Coinbase:   coinbase,
		Difficulty: difficulty,
		Time:       uint64(time.Now().Unix()),
		Extra:      extra,
		GasUsed:    new(big.Int),
		GasLimit:   new(big.Int),
		Number:     new(big.Int),
	}
	header.SetNonce(nonce)
	block := &Block{header: header}
	block.Td = new(big.Int)

	return block
}

func (self *Header) SetNonce(nonce uint64) {
	binary.BigEndian.PutUint64(self.Nonce[:], nonce)
}

func NewBlockWithHeader(header *Header) *Block {
	return &Block{header: header}
}

func (self *Block) ValidateFields() error {
	if self.header == nil {
		return fmt.Errorf("header is nil")
	}
	for i, transaction := range self.transactions {
		if transaction == nil {
			return fmt.Errorf("transaction %d is nil", i)
		}
	}
	for i, uncle := range self.uncles {
		if uncle == nil {
			return fmt.Errorf("uncle %d is nil", i)
		}
	}
	return nil
}

func (self *Block) DecodeRLP(s *rlp.Stream) error {
	var eb extblock
	if err := s.Decode(&eb); err != nil {
		return err
	}
	self.header, self.uncles, self.transactions = eb.Header, eb.Uncles, eb.Txs
	return nil
}

func (self Block) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, extblock{
		Header: self.header,
		Txs:    self.transactions,
		Uncles: self.uncles,
	})
}

func (self *StorageBlock) DecodeRLP(s *rlp.Stream) error {
	var sb storageblock
	if err := s.Decode(&sb); err != nil {
		return err
	}
	self.header, self.uncles, self.transactions, self.Td = sb.Header, sb.Uncles, sb.Txs, sb.TD
	return nil
}

func (self StorageBlock) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, storageblock{
		Header: self.header,
		Txs:    self.transactions,
		Uncles: self.uncles,
		TD:     self.Td,
	})
}

func (self *Block) Header() *Header {
	return self.header
}

func (self *Block) Uncles() []*Header {
	return self.uncles
}

func (self *Block) CalculateUnclesHash() common.Hash {
	return rlpHash(self.uncles)
}

func (self *Block) SetUncles(uncleHeaders []*Header) {
	self.uncles = uncleHeaders
	self.header.UncleHash = rlpHash(uncleHeaders)
}

func (self *Block) Transactions() Transactions {
	return self.transactions
}

func (self *Block) Transaction(hash common.Hash) *Transaction {
	for _, transaction := range self.transactions {
		if transaction.Hash() == hash {
			return transaction
		}
	}
	return nil
}

func (self *Block) SetTransactions(transactions Transactions) {
	self.transactions = transactions
	self.header.TxHash = DeriveSha(transactions)
}
func (self *Block) AddTransaction(transaction *Transaction) {
	self.transactions = append(self.transactions, transaction)
	self.SetTransactions(self.transactions)
}

func (self *Block) Receipts() Receipts {
	return self.receipts
}

func (self *Block) SetReceipts(receipts Receipts) {
	self.receipts = receipts
	self.header.ReceiptHash = DeriveSha(receipts)
	self.header.Bloom = CreateBloom(receipts)
}
func (self *Block) AddReceipt(receipt *Receipt) {
	self.receipts = append(self.receipts, receipt)
	self.SetReceipts(self.receipts)
}

func (self *Block) RlpData() interface{} {
	return []interface{}{self.header, self.transactions, self.uncles}
}

func (self *Block) RlpDataForStorage() interface{} {
	return []interface{}{self.header, self.transactions, self.uncles, self.Td /* TODO receipts */}
}

// Header accessors (add as you need them)
func (self *Block) Number() *big.Int       { return self.header.Number }
func (self *Block) NumberU64() uint64      { return self.header.Number.Uint64() }
func (self *Block) MixDigest() common.Hash { return self.header.MixDigest }
func (self *Block) Nonce() uint64 {
	return binary.BigEndian.Uint64(self.header.Nonce[:])
}
func (self *Block) SetNonce(nonce uint64) {
	self.header.SetNonce(nonce)
}

func (self *Block) Queued() bool     { return self.queued }
func (self *Block) SetQueued(q bool) { self.queued = q }

func (self *Block) Bloom() Bloom             { return self.header.Bloom }
func (self *Block) Coinbase() common.Address { return self.header.Coinbase }
func (self *Block) Time() int64              { return int64(self.header.Time) }
func (self *Block) GasLimit() *big.Int       { return self.header.GasLimit }
func (self *Block) GasUsed() *big.Int        { return self.header.GasUsed }
func (self *Block) Root() common.Hash        { return self.header.Root }
func (self *Block) SetRoot(root common.Hash) { self.header.Root = root }
func (self *Block) GetTransaction(i int) *Transaction {
	if len(self.transactions) > i {
		return self.transactions[i]
	}
	return nil
}
func (self *Block) GetUncle(i int) *Header {
	if len(self.uncles) > i {
		return self.uncles[i]
	}
	return nil
}

func (self *Block) Size() common.StorageSize {
	c := writeCounter(0)
	rlp.Encode(&c, self)
	return common.StorageSize(c)
}

type writeCounter common.StorageSize

func (c *writeCounter) Write(b []byte) (int, error) {
	*c += writeCounter(len(b))
	return len(b), nil
}

// Implement pow.Block
func (self *Block) Difficulty() *big.Int     { return self.header.Difficulty }
func (self *Block) HashNoNonce() common.Hash { return self.header.HashNoNonce() }

func (self *Block) Hash() common.Hash {
	if (self.HeaderHash != common.Hash{}) {
		return self.HeaderHash
	} else {
		return self.header.Hash()
	}
}

func (self *Block) ParentHash() common.Hash {
	if (self.ParentHeaderHash != common.Hash{}) {
		return self.ParentHeaderHash
	} else {
		return self.header.ParentHash
	}
}

func (self *Block) Copy() *Block {
	block := NewBlock(self.header.ParentHash, self.Coinbase(), self.Root(), new(big.Int), self.Nonce(), self.header.Extra)
	block.header.Bloom = self.header.Bloom
	block.header.TxHash = self.header.TxHash
	block.transactions = self.transactions
	block.header.UncleHash = self.header.UncleHash
	block.uncles = self.uncles
	block.header.GasLimit.Set(self.header.GasLimit)
	block.header.GasUsed.Set(self.header.GasUsed)
	block.header.ReceiptHash = self.header.ReceiptHash
	block.header.Difficulty.Set(self.header.Difficulty)
	block.header.Number.Set(self.header.Number)
	block.header.Time = self.header.Time
	block.header.MixDigest = self.header.MixDigest
	if self.Td != nil {
		block.Td.Set(self.Td)
	}

	return block
}

func (self *Block) String() string {
	str := fmt.Sprintf(`Block(#%v): Size: %v TD: %v {
MinerHash: %x
%v
Transactions:
%v
Uncles:
%v
}
`, self.Number(), self.Size(), self.Td, self.header.HashNoNonce(), self.header, self.transactions, self.uncles)

	if (self.HeaderHash != common.Hash{}) {
		str += fmt.Sprintf("\nFake hash = %x", self.HeaderHash)
	}

	if (self.ParentHeaderHash != common.Hash{}) {
		str += fmt.Sprintf("\nFake parent hash = %x", self.ParentHeaderHash)
	}

	return str
}

func (self *Header) String() string {
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
	MixDigest:          %x
	Nonce:		    %x
]`, self.Hash(), self.ParentHash, self.UncleHash, self.Coinbase, self.Root, self.TxHash, self.ReceiptHash, self.Bloom, self.Difficulty, self.Number, self.GasLimit, self.GasUsed, self.Time, self.Extra, self.MixDigest, self.Nonce)
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

func Number(b1, b2 *Block) bool { return b1.Header().Number.Cmp(b2.Header().Number) < 0 }
