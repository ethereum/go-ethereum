package ethchain

import (
	"bytes"
	"fmt"
	"math/big"
	_ "strconv"
	"time"

	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethtrie"
	"github.com/ethereum/eth-go/ethutil"
)

type BlockInfo struct {
	Number uint64
	Hash   []byte
	Parent []byte
}

func (bi *BlockInfo) RlpDecode(data []byte) {
	decoder := ethutil.NewValueFromBytes(data)

	bi.Number = decoder.Get(0).Uint()
	bi.Hash = decoder.Get(1).Bytes()
	bi.Parent = decoder.Get(2).Bytes()
}

func (bi *BlockInfo) RlpEncode() []byte {
	return ethutil.Encode([]interface{}{bi.Number, bi.Hash, bi.Parent})
}

type Blocks []*Block

func (self Blocks) AsSet() ethutil.UniqueSet {
	set := make(ethutil.UniqueSet)
	for _, block := range self {
		set.Insert(block.Hash())
	}

	return set
}

type Block struct {
	// Hash to the previous block
	PrevHash []byte
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
	Nonce []byte
	// List of transactions and/or contracts
	transactions []*Transaction
	receipts     []*Receipt
	TxSha        []byte
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
		UncleSha:    EmptyShaList,
		GasUsed:     new(big.Int),
		MinGasPrice: new(big.Int),
		GasLimit:    new(big.Int),
	}
	block.SetUncles([]*Block{})

	block.state = ethstate.New(ethtrie.New(ethutil.Config.Db, root))

	return block
}

// Returns a hash of the block
func (block *Block) Hash() []byte {
	return ethcrypto.Sha3Bin(block.Value().Encode())
}

func (block *Block) HashNoNonce() []byte {
	return ethcrypto.Sha3Bin(ethutil.Encode([]interface{}{block.PrevHash,
		block.UncleSha, block.Coinbase, block.state.Trie.Root,
		block.TxSha, block.Difficulty, block.Number, block.MinGasPrice,
		block.GasLimit, block.GasUsed, block.Time, block.Extra}))
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
	for _, receipt := range self.receipts {
		if bytes.Compare(receipt.Tx.Hash(), hash) == 0 {
			return receipt.Tx
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

	// Sha of the concatenated uncles
	block.UncleSha = ethcrypto.Sha3Bin(ethutil.Encode(block.rlpUncles()))
}

func (self *Block) SetReceipts(receipts []*Receipt, txs []*Transaction) {
	self.receipts = receipts
	self.setTransactions(txs)
}

func (block *Block) setTransactions(txs []*Transaction) {
	block.transactions = txs
}

func CreateTxSha(receipts Receipts) (sha []byte) {
	trie := ethtrie.New(ethutil.Config.Db, "")
	for i, receipt := range receipts {
		trie.Update(string(ethutil.NewValue(i).Encode()), string(ethutil.NewValue(receipt.RlpData()).Encode()))
	}

	switch trie.Root.(type) {
	case string:
		sha = []byte(trie.Root.(string))
	case []byte:
		sha = trie.Root.([]byte)
	default:
		panic(fmt.Sprintf("invalid root type %T", trie.Root))
	}

	return sha
}

func (self *Block) SetTxHash(receipts Receipts) {
	self.TxSha = CreateTxSha(receipts)
}

func (block *Block) Value() *ethutil.Value {
	return ethutil.NewValue([]interface{}{block.header(), block.rlpReceipts(), block.rlpUncles()})
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
	header := decoder.Get(0)

	block.PrevHash = header.Get(0).Bytes()
	block.UncleSha = header.Get(1).Bytes()
	block.Coinbase = header.Get(2).Bytes()
	block.state = ethstate.New(ethtrie.New(ethutil.Config.Db, header.Get(3).Val))
	block.TxSha = header.Get(4).Bytes()
	block.Difficulty = header.Get(5).BigInt()
	block.Number = header.Get(6).BigInt()
	//fmt.Printf("#%v : %x\n", block.Number, block.Coinbase)
	block.MinGasPrice = header.Get(7).BigInt()
	block.GasLimit = header.Get(8).BigInt()
	block.GasUsed = header.Get(9).BigInt()
	block.Time = int64(header.Get(10).BigInt().Uint64())
	block.Extra = header.Get(11).Str()
	block.Nonce = header.Get(12).Bytes()

	// Tx list might be empty if this is an uncle. Uncles only have their
	// header set.
	if decoder.Get(1).IsNil() == false { // Yes explicitness
		receipts := decoder.Get(1)
		block.transactions = make([]*Transaction, receipts.Len())
		block.receipts = make([]*Receipt, receipts.Len())
		for i := 0; i < receipts.Len(); i++ {
			receipt := NewRecieptFromValue(receipts.Get(i))
			block.transactions[i] = receipt.Tx
			block.receipts[i] = receipt
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

func NewUncleBlockFromValue(header *ethutil.Value) *Block {
	block := &Block{}

	block.PrevHash = header.Get(0).Bytes()
	block.UncleSha = header.Get(1).Bytes()
	block.Coinbase = header.Get(2).Bytes()
	block.state = ethstate.New(ethtrie.New(ethutil.Config.Db, header.Get(3).Val))
	block.TxSha = header.Get(4).Bytes()
	block.Difficulty = header.Get(5).BigInt()
	block.Number = header.Get(6).BigInt()
	block.MinGasPrice = header.Get(7).BigInt()
	block.GasLimit = header.Get(8).BigInt()
	block.GasUsed = header.Get(9).BigInt()
	block.Time = int64(header.Get(10).BigInt().Uint64())
	block.Extra = header.Get(11).Str()
	block.Nonce = header.Get(12).Bytes()

	return block
}

func (block *Block) GetRoot() interface{} {
	return block.state.Trie.Root
}

func (self *Block) Receipts() []*Receipt {
	return self.receipts
}

func (block *Block) header() []interface{} {
	return []interface{}{
		// Sha of the previous block
		block.PrevHash,
		// Sha of uncles
		block.UncleSha,
		// Coinbase address
		block.Coinbase,
		// root state
		block.state.Trie.Root,
		// Sha of tx
		block.TxSha,
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
		// Block's Nonce for validation
		block.Nonce,
	}
}

func (block *Block) String() string {
	return fmt.Sprintf(`
	BLOCK(%x): Size: %v
	PrevHash:   %x
	UncleSha:   %x
	Coinbase:   %x
	Root:       %x
	TxSha:      %x
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
