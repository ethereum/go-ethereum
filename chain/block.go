package chain

import (
	"bytes"
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethstate"
	"github.com/ethereum/go-ethereum/ethtrie"
	"github.com/ethereum/go-ethereum/ethutil"
)

type BlockInfo struct {
	Number uint64
	Hash   []byte
	Parent []byte
	TD     *big.Int
}

func (bi *BlockInfo) RlpDecode(data []byte) {
	decoder := ethutil.NewValueFromBytes(data)

	bi.Number = decoder.Get(0).Uint()
	bi.Hash = decoder.Get(1).Bytes()
	bi.Parent = decoder.Get(2).Bytes()
	bi.TD = decoder.Get(3).BigInt()
}

func (bi *BlockInfo) RlpEncode() []byte {
	return ethutil.Encode([]interface{}{bi.Number, bi.Hash, bi.Parent, bi.TD})
}

type Blocks []*Block

func (self Blocks) AsSet() ethutil.UniqueSet {
	set := make(ethutil.UniqueSet)
	for _, block := range self {
		set.Insert(block.Hash())
	}

	return set
}

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

func Number(b1, b2 *Block) bool { return b1.Number.Cmp(b2.Number) < 0 }

type Block struct {
	// Hash to the previous block
	PrevHash ethutil.Bytes
	// Uncles of this block
	Uncles   Blocks
	UncleSha []byte
	// The coin base address
	Coinbase []byte
	// Block Trie state
	//state *ethutil.Trie
	state *ethstate.State
	// Difficulty for the current block
	Difficulty *big.Int
	// Creation time
	Time int64
	// The block number
	Number *big.Int
	// Minimum Gas Price
	MinGasPrice *big.Int
	// Gas limit
	GasLimit *big.Int
	// Gas used
	GasUsed *big.Int
	// Extra data
	Extra string
	// Block Nonce for verification
	Nonce ethutil.Bytes
	// List of transactions and/or contracts
	transactions      Transactions
	receipts          Receipts
	TxSha, ReceiptSha []byte
	LogsBloom         []byte
}

func NewBlockFromBytes(raw []byte) *Block {
	block := &Block{}
	block.RlpDecode(raw)

	return block
}

// New block takes a raw encoded string
func NewBlockFromRlpValue(rlpValue *ethutil.Value) *Block {
	block := &Block{}
	block.RlpValueDecode(rlpValue)

	return block
}

func CreateBlock(root interface{},
	prevHash []byte,
	base []byte,
	Difficulty *big.Int,
	Nonce []byte,
	extra string) *Block {

	block := &Block{
		PrevHash:    prevHash,
		Coinbase:    base,
		Difficulty:  Difficulty,
		Nonce:       Nonce,
		Time:        time.Now().Unix(),
		Extra:       extra,
		UncleSha:    nil,
		GasUsed:     new(big.Int),
		MinGasPrice: new(big.Int),
		GasLimit:    new(big.Int),
	}
	block.SetUncles([]*Block{})

	block.state = ethstate.New(ethtrie.New(ethutil.Config.Db, root))

	return block
}

// Returns a hash of the block
func (block *Block) Hash() ethutil.Bytes {
	return crypto.Sha3(ethutil.NewValue(block.header()).Encode())
	//return crypto.Sha3(block.Value().Encode())
}

func (block *Block) HashNoNonce() []byte {
	return crypto.Sha3(ethutil.Encode(block.miningHeader()))
}

func (block *Block) State() *ethstate.State {
	return block.state
}

func (block *Block) Transactions() []*Transaction {
	return block.transactions
}

func (block *Block) CalcGasLimit(parent *Block) *big.Int {
	if block.Number.Cmp(big.NewInt(0)) == 0 {
		return ethutil.BigPow(10, 6)
	}

	// ((1024-1) * parent.gasLimit + (gasUsed * 6 / 5)) / 1024

	previous := new(big.Int).Mul(big.NewInt(1024-1), parent.GasLimit)
	current := new(big.Rat).Mul(new(big.Rat).SetInt(parent.GasUsed), big.NewRat(6, 5))
	curInt := new(big.Int).Div(current.Num(), current.Denom())

	result := new(big.Int).Add(previous, curInt)
	result.Div(result, big.NewInt(1024))

	min := big.NewInt(125000)

	return ethutil.BigMax(min, result)
}

func (block *Block) BlockInfo() BlockInfo {
	bi := BlockInfo{}
	data, _ := ethutil.Config.Db.Get(append(block.Hash(), []byte("Info")...))
	bi.RlpDecode(data)

	return bi
}

func (self *Block) GetTransaction(hash []byte) *Transaction {
	for _, tx := range self.transactions {
		if bytes.Compare(tx.Hash(), hash) == 0 {
			return tx
		}
	}

	return nil
}

// Sync the block's state and contract respectively
func (block *Block) Sync() {
	block.state.Sync()
}

func (block *Block) Undo() {
	// Sync the block state itself
	block.state.Reset()
}

/////// Block Encoding
func (block *Block) rlpReceipts() interface{} {
	// Marshal the transactions of this block
	encR := make([]interface{}, len(block.receipts))
	for i, r := range block.receipts {
		// Cast it to a string (safe)
		encR[i] = r.RlpData()
	}

	return encR
}

func (block *Block) rlpUncles() interface{} {
	// Marshal the transactions of this block
	uncles := make([]interface{}, len(block.Uncles))
	for i, uncle := range block.Uncles {
		// Cast it to a string (safe)
		uncles[i] = uncle.header()
	}

	return uncles
}

func (block *Block) SetUncles(uncles []*Block) {
	block.Uncles = uncles
	block.UncleSha = crypto.Sha3(ethutil.Encode(block.rlpUncles()))
}

func (self *Block) SetReceipts(receipts Receipts) {
	self.receipts = receipts
	self.ReceiptSha = DeriveSha(receipts)
	self.LogsBloom = CreateBloom(self)
}

func (self *Block) SetTransactions(txs Transactions) {
	self.transactions = txs
	self.TxSha = DeriveSha(txs)
}

func (block *Block) Value() *ethutil.Value {
	return ethutil.NewValue([]interface{}{block.header(), block.transactions, block.rlpUncles()})
}

func (block *Block) RlpEncode() []byte {
	// Encode a slice interface which contains the header and the list of
	// transactions.
	return block.Value().Encode()
}

func (block *Block) RlpDecode(data []byte) {
	rlpValue := ethutil.NewValueFromBytes(data)
	block.RlpValueDecode(rlpValue)
}

func (block *Block) RlpValueDecode(decoder *ethutil.Value) {
	block.setHeader(decoder.Get(0))

	// Tx list might be empty if this is an uncle. Uncles only have their
	// header set.
	if decoder.Get(1).IsNil() == false { // Yes explicitness
		//receipts := decoder.Get(1)
		//block.receipts = make([]*Receipt, receipts.Len())
		txs := decoder.Get(1)
		block.transactions = make(Transactions, txs.Len())
		for i := 0; i < txs.Len(); i++ {
			block.transactions[i] = NewTransactionFromValue(txs.Get(i))
			//receipt := NewRecieptFromValue(receipts.Get(i))
			//block.transactions[i] = receipt.Tx
			//block.receipts[i] = receipt
		}

	}

	if decoder.Get(2).IsNil() == false { // Yes explicitness
		uncles := decoder.Get(2)
		block.Uncles = make([]*Block, uncles.Len())
		for i := 0; i < uncles.Len(); i++ {
			block.Uncles[i] = NewUncleBlockFromValue(uncles.Get(i))
		}
	}

}

func (self *Block) setHeader(header *ethutil.Value) {
	self.PrevHash = header.Get(0).Bytes()
	self.UncleSha = header.Get(1).Bytes()
	self.Coinbase = header.Get(2).Bytes()
	self.state = ethstate.New(ethtrie.New(ethutil.Config.Db, header.Get(3).Val))
	self.TxSha = header.Get(4).Bytes()
	self.ReceiptSha = header.Get(5).Bytes()
	self.LogsBloom = header.Get(6).Bytes()
	self.Difficulty = header.Get(7).BigInt()
	self.Number = header.Get(8).BigInt()
	self.MinGasPrice = header.Get(9).BigInt()
	self.GasLimit = header.Get(10).BigInt()
	self.GasUsed = header.Get(11).BigInt()
	self.Time = int64(header.Get(12).BigInt().Uint64())
	self.Extra = header.Get(13).Str()
	self.Nonce = header.Get(14).Bytes()
}

func NewUncleBlockFromValue(header *ethutil.Value) *Block {
	block := &Block{}
	block.setHeader(header)

	return block
}

func (block *Block) Trie() *ethtrie.Trie {
	return block.state.Trie
}

func (block *Block) GetRoot() interface{} {
	return block.state.Trie.Root
}

func (block *Block) Diff() *big.Int {
	return block.Difficulty
}

func (self *Block) Receipts() []*Receipt {
	return self.receipts
}

func (block *Block) miningHeader() []interface{} {
	return []interface{}{
		// Sha of the previous block
		block.PrevHash,
		// Sha of uncles
		block.UncleSha,
		// Coinbase address
		block.Coinbase,
		// root state
		block.state.Trie.Root,
		// tx root
		block.TxSha,
		// Sha of tx
		block.ReceiptSha,
		// Bloom
		block.LogsBloom,
		// Current block Difficulty
		block.Difficulty,
		// The block number
		block.Number,
		// Block minimum gas price
		block.MinGasPrice,
		// Block upper gas bound
		block.GasLimit,
		// Block gas used
		block.GasUsed,
		// Time the block was found?
		block.Time,
		// Extra data
		block.Extra,
	}
}

func (block *Block) header() []interface{} {
	return append(block.miningHeader(), block.Nonce)
}

func (block *Block) String() string {
	return fmt.Sprintf(`
	BLOCK(%x): Size: %v
	PrevHash:   %x
	UncleSha:   %x
	Coinbase:   %x
	Root:       %x
	TxSha       %x
	ReceiptSha: %x
	Bloom:      %x
	Difficulty: %v
	Number:     %v
	MinGas:     %v
	MaxLimit:   %v
	GasUsed:    %v
	Time:       %v
	Extra:      %v
	Nonce:      %x
	NumTx:      %v
`,
		block.Hash(),
		block.Size(),
		block.PrevHash,
		block.UncleSha,
		block.Coinbase,
		block.state.Trie.Root,
		block.TxSha,
		block.ReceiptSha,
		block.LogsBloom,
		block.Difficulty,
		block.Number,
		block.MinGasPrice,
		block.GasLimit,
		block.GasUsed,
		block.Time,
		block.Extra,
		block.Nonce,
		len(block.transactions),
	)
}

func (self *Block) Size() ethutil.StorageSize {
	return ethutil.StorageSize(len(self.RlpEncode()))
}

// Implement RlpEncodable
func (self *Block) RlpData() interface{} {
	return self.Value().Val
}
