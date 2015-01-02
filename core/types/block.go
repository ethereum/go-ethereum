package types

import (
	"bytes"
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/ptrie"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/state"
)

type Header struct {
	// Hash to the previous block
	ParentHash ethutil.Bytes
	// Uncles of this block
	UncleHash []byte
	// The coin base address
	Coinbase []byte
	// Block Trie state
	Root []byte
	// Tx sha
	TxHash []byte
	// Receipt sha
	ReceiptHash []byte
	// Bloom
	Bloom []byte
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
	Extra string
	// Block Nonce for verification
	Nonce ethutil.Bytes
}

func (self *Header) rlpData(withNonce bool) []interface{} {
	fields := []interface{}{self.ParentHash, self.UncleHash, self.Coinbase, self.Root, self.TxHash, self.ReceiptHash, self.Bloom, self.Difficulty, self.Number, self.GasLimit, self.GasUsed, self.Time, self.Extra}
	if withNonce {
		fields = append(fields, self.Nonce)
	}

	return fields
}

func (self *Header) RlpData() interface{} {
	return self.rlpData(true)
}

func (self *Header) Hash() []byte {
	return crypto.Sha3(ethutil.Encode(self.rlpData(true)))
}

func (self *Header) HashNoNonce() []byte {
	return crypto.Sha3(ethutil.Encode(self.rlpData(false)))
}

type Block struct {
	header       *Header
	uncles       []*Header
	transactions Transactions
	Td           *big.Int

	receipts Receipts
	Reward   *big.Int
}

func NewBlock(parentHash []byte, coinbase []byte, root []byte, difficulty *big.Int, nonce []byte, extra string) *Block {
	header := &Header{
		Root:       root,
		ParentHash: parentHash,
		Coinbase:   coinbase,
		Difficulty: difficulty,
		Nonce:      nonce,
		Time:       uint64(time.Now().Unix()),
		Extra:      extra,
		GasUsed:    new(big.Int),
		GasLimit:   new(big.Int),
	}

	block := &Block{header: header, Reward: new(big.Int)}

	return block
}

func NewBlockWithHeader(header *Header) *Block {
	return &Block{header: header}
}

func (self *Block) DecodeRLP(s *rlp.Stream) error {
	if _, err := s.List(); err != nil {
		return err
	}

	var header Header
	if err := s.Decode(&header); err != nil {
		return err
	}

	var transactions []*Transaction
	if err := s.Decode(&transactions); err != nil {
		return err
	}

	var uncleHeaders []*Header
	if err := s.Decode(&uncleHeaders); err != nil {
		return err
	}

	var tdBytes []byte
	if err := s.Decode(&tdBytes); err != nil {
		// If this block comes from the network that's fine. If loaded from disk it should be there
		// Blocks don't store their Td when propagated over the network
	} else {
		self.Td = ethutil.BigD(tdBytes)
	}

	if err := s.ListEnd(); err != nil {
		return err
	}

	self.header = &header
	self.uncles = uncleHeaders
	self.transactions = transactions

	return nil
}

func (self *Block) Header() *Header {
	return self.header
}

func (self *Block) Uncles() []*Header {
	return self.uncles
}

func (self *Block) SetUncles(uncleHeaders []*Header) {
	self.uncles = uncleHeaders
	self.header.UncleHash = crypto.Sha3(ethutil.Encode(uncleHeaders))
}

func (self *Block) Transactions() Transactions {
	return self.transactions
}

func (self *Block) Transaction(hash []byte) *Transaction {
	for _, transaction := range self.transactions {
		if bytes.Equal(hash, transaction.Hash()) {
			return transaction
		}
	}
	return nil
}

func (self *Block) SetTransactions(transactions Transactions) {
	self.transactions = transactions
	self.header.TxHash = DeriveSha(transactions)
}

func (self *Block) Receipts() Receipts {
	return self.receipts
}

func (self *Block) SetReceipts(receipts Receipts) {
	self.receipts = receipts
	self.header.ReceiptHash = DeriveSha(receipts)
	self.header.Bloom = CreateBloom(receipts)
}

func (self *Block) RlpData() interface{} {
	return []interface{}{self.header, self.transactions, self.uncles}
}

func (self *Block) RlpDataForStorage() interface{} {
	return []interface{}{self.header, self.transactions, self.uncles, self.Td /* TODO receipts */}
}

// Header accessors (add as you need them)
func (self *Block) Number() *big.Int          { return self.header.Number }
func (self *Block) NumberU64() uint64         { return self.header.Number.Uint64() }
func (self *Block) ParentHash() []byte        { return self.header.ParentHash }
func (self *Block) Bloom() []byte             { return self.header.Bloom }
func (self *Block) Coinbase() []byte          { return self.header.Coinbase }
func (self *Block) Time() int64               { return int64(self.header.Time) }
func (self *Block) GasLimit() *big.Int        { return self.header.GasLimit }
func (self *Block) GasUsed() *big.Int         { return self.header.GasUsed }
func (self *Block) Hash() []byte              { return self.header.Hash() }
func (self *Block) Trie() *ptrie.Trie         { return ptrie.New(self.header.Root, ethutil.Config.Db) }
func (self *Block) State() *state.StateDB     { return state.New(self.Trie()) }
func (self *Block) Size() ethutil.StorageSize { return ethutil.StorageSize(len(ethutil.Encode(self))) }
func (self *Block) SetRoot(root []byte)       { self.header.Root = root }

// Implement block.Pow
func (self *Block) Difficulty() *big.Int { return self.header.Difficulty }
func (self *Block) N() []byte            { return self.header.Nonce }
func (self *Block) HashNoNonce() []byte {
	return crypto.Sha3(ethutil.Encode(self.header.rlpData(false)))
}

func (self *Block) String() string {
	return fmt.Sprintf(`BLOCK(%x): Size: %v TD: %v {
Header:
[
%v
]
Transactions:
%v
Uncles:
%v
}
`, self.header.Hash(), self.Size(), self.Td, self.header, self.transactions, self.uncles)
}

func (self *Header) String() string {
	return fmt.Sprintf(`
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
	Extra:		    %v
	Nonce:		    %x
`, self.ParentHash, self.UncleHash, self.Coinbase, self.Root, self.TxHash, self.ReceiptHash, self.Bloom, self.Difficulty, self.Number, self.GasLimit, self.GasUsed, self.Time, self.Extra, self.Nonce)
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
