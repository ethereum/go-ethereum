package api

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc/shared"
)

type hexdata struct {
	data  []byte
	isNil bool
}

func (d *hexdata) String() string {
	return "0x" + common.Bytes2Hex(d.data)
}

func (d *hexdata) MarshalJSON() ([]byte, error) {
	if d.isNil {
		return json.Marshal(nil)
	}
	return json.Marshal(d.String())
}

func newHexData(input interface{}) *hexdata {
	d := new(hexdata)

	if input == nil {
		d.isNil = true
		return d
	}
	switch input := input.(type) {
	case []byte:
		d.data = input
	case common.Hash:
		d.data = input.Bytes()
	case *common.Hash:
		if input == nil {
			d.isNil = true
		} else {
			d.data = input.Bytes()
		}
	case common.Address:
		d.data = input.Bytes()
	case *common.Address:
		if input == nil {
			d.isNil = true
		} else {
			d.data = input.Bytes()
		}
	case types.Bloom:
		d.data = input.Bytes()
	case *types.Bloom:
		if input == nil {
			d.isNil = true
		} else {
			d.data = input.Bytes()
		}
	case *big.Int:
		if input == nil {
			d.isNil = true
		} else {
			d.data = input.Bytes()
		}
	case int64:
		d.data = big.NewInt(input).Bytes()
	case uint64:
		buff := make([]byte, 8)
		binary.BigEndian.PutUint64(buff, input)
		d.data = buff
	case int:
		d.data = big.NewInt(int64(input)).Bytes()
	case uint:
		d.data = big.NewInt(int64(input)).Bytes()
	case int8:
		d.data = big.NewInt(int64(input)).Bytes()
	case uint8:
		d.data = big.NewInt(int64(input)).Bytes()
	case int16:
		d.data = big.NewInt(int64(input)).Bytes()
	case uint16:
		buff := make([]byte, 2)
		binary.BigEndian.PutUint16(buff, input)
		d.data = buff
	case int32:
		d.data = big.NewInt(int64(input)).Bytes()
	case uint32:
		buff := make([]byte, 4)
		binary.BigEndian.PutUint32(buff, input)
		d.data = buff
	case string: // hexstring
		// aaargh ffs TODO: avoid back-and-forth hex encodings where unneeded
		bytes, err := hex.DecodeString(strings.TrimPrefix(input, "0x"))
		if err != nil {
			d.isNil = true
		} else {
			d.data = bytes
		}
	default:
		d.isNil = true
	}

	return d
}

type hexnum struct {
	data  []byte
	isNil bool
}

func (d *hexnum) String() string {
	// Get hex string from bytes
	out := common.Bytes2Hex(d.data)
	// Trim leading 0s
	out = strings.TrimLeft(out, "0")
	// Output "0x0" when value is 0
	if len(out) == 0 {
		out = "0"
	}
	return "0x" + out
}

func (d *hexnum) MarshalJSON() ([]byte, error) {
	if d.isNil {
		return json.Marshal(nil)
	}
	return json.Marshal(d.String())
}

func newHexNum(input interface{}) *hexnum {
	d := new(hexnum)

	d.data = newHexData(input).data

	return d
}

type BlockRes struct {
	fullTx bool

	BlockNumber     *hexnum           `json:"number"`
	BlockHash       *hexdata          `json:"hash"`
	ParentHash      *hexdata          `json:"parentHash"`
	Nonce           *hexdata          `json:"nonce"`
	Sha3Uncles      *hexdata          `json:"sha3Uncles"`
	LogsBloom       *hexdata          `json:"logsBloom"`
	TransactionRoot *hexdata          `json:"transactionsRoot"`
	StateRoot       *hexdata          `json:"stateRoot"`
	Miner           *hexdata          `json:"miner"`
	Difficulty      *hexnum           `json:"difficulty"`
	TotalDifficulty *hexnum           `json:"totalDifficulty"`
	Size            *hexnum           `json:"size"`
	ExtraData       *hexdata          `json:"extraData"`
	GasLimit        *hexnum           `json:"gasLimit"`
	GasUsed         *hexnum           `json:"gasUsed"`
	UnixTimestamp   *hexnum           `json:"timestamp"`
	Transactions    []*TransactionRes `json:"transactions"`
	Uncles          []*UncleRes       `json:"uncles"`
}

func (b *BlockRes) MarshalJSON() ([]byte, error) {
	if b.fullTx {
		var ext struct {
			BlockNumber     *hexnum           `json:"number"`
			BlockHash       *hexdata          `json:"hash"`
			ParentHash      *hexdata          `json:"parentHash"`
			Nonce           *hexdata          `json:"nonce"`
			Sha3Uncles      *hexdata          `json:"sha3Uncles"`
			LogsBloom       *hexdata          `json:"logsBloom"`
			TransactionRoot *hexdata          `json:"transactionsRoot"`
			StateRoot       *hexdata          `json:"stateRoot"`
			Miner           *hexdata          `json:"miner"`
			Difficulty      *hexnum           `json:"difficulty"`
			TotalDifficulty *hexnum           `json:"totalDifficulty"`
			Size            *hexnum           `json:"size"`
			ExtraData       *hexdata          `json:"extraData"`
			GasLimit        *hexnum           `json:"gasLimit"`
			GasUsed         *hexnum           `json:"gasUsed"`
			UnixTimestamp   *hexnum           `json:"timestamp"`
			Transactions    []*TransactionRes `json:"transactions"`
			Uncles          []*hexdata        `json:"uncles"`
		}

		ext.BlockNumber = b.BlockNumber
		ext.BlockHash = b.BlockHash
		ext.ParentHash = b.ParentHash
		ext.Nonce = b.Nonce
		ext.Sha3Uncles = b.Sha3Uncles
		ext.LogsBloom = b.LogsBloom
		ext.TransactionRoot = b.TransactionRoot
		ext.StateRoot = b.StateRoot
		ext.Miner = b.Miner
		ext.Difficulty = b.Difficulty
		ext.TotalDifficulty = b.TotalDifficulty
		ext.Size = b.Size
		ext.ExtraData = b.ExtraData
		ext.GasLimit = b.GasLimit
		ext.GasUsed = b.GasUsed
		ext.UnixTimestamp = b.UnixTimestamp
		ext.Transactions = b.Transactions
		ext.Uncles = make([]*hexdata, len(b.Uncles))
		for i, u := range b.Uncles {
			ext.Uncles[i] = u.BlockHash
		}
		return json.Marshal(ext)
	} else {
		var ext struct {
			BlockNumber     *hexnum    `json:"number"`
			BlockHash       *hexdata   `json:"hash"`
			ParentHash      *hexdata   `json:"parentHash"`
			Nonce           *hexdata   `json:"nonce"`
			Sha3Uncles      *hexdata   `json:"sha3Uncles"`
			LogsBloom       *hexdata   `json:"logsBloom"`
			TransactionRoot *hexdata   `json:"transactionsRoot"`
			StateRoot       *hexdata   `json:"stateRoot"`
			Miner           *hexdata   `json:"miner"`
			Difficulty      *hexnum    `json:"difficulty"`
			TotalDifficulty *hexnum    `json:"totalDifficulty"`
			Size            *hexnum    `json:"size"`
			ExtraData       *hexdata   `json:"extraData"`
			GasLimit        *hexnum    `json:"gasLimit"`
			GasUsed         *hexnum    `json:"gasUsed"`
			UnixTimestamp   *hexnum    `json:"timestamp"`
			Transactions    []*hexdata `json:"transactions"`
			Uncles          []*hexdata `json:"uncles"`
		}

		ext.BlockNumber = b.BlockNumber
		ext.BlockHash = b.BlockHash
		ext.ParentHash = b.ParentHash
		ext.Nonce = b.Nonce
		ext.Sha3Uncles = b.Sha3Uncles
		ext.LogsBloom = b.LogsBloom
		ext.TransactionRoot = b.TransactionRoot
		ext.StateRoot = b.StateRoot
		ext.Miner = b.Miner
		ext.Difficulty = b.Difficulty
		ext.TotalDifficulty = b.TotalDifficulty
		ext.Size = b.Size
		ext.ExtraData = b.ExtraData
		ext.GasLimit = b.GasLimit
		ext.GasUsed = b.GasUsed
		ext.UnixTimestamp = b.UnixTimestamp
		ext.Transactions = make([]*hexdata, len(b.Transactions))
		for i, tx := range b.Transactions {
			ext.Transactions[i] = tx.Hash
		}
		ext.Uncles = make([]*hexdata, len(b.Uncles))
		for i, u := range b.Uncles {
			ext.Uncles[i] = u.BlockHash
		}
		return json.Marshal(ext)
	}
}

func NewBlockRes(block *types.Block, fullTx bool) *BlockRes {
	if block == nil {
		return nil
	}

	res := new(BlockRes)
	res.fullTx = fullTx
	res.BlockNumber = newHexNum(block.Number())
	res.BlockHash = newHexData(block.Hash())
	res.ParentHash = newHexData(block.ParentHash())
	res.Nonce = newHexData(block.Nonce())
	res.Sha3Uncles = newHexData(block.UncleHash())
	res.LogsBloom = newHexData(block.Bloom())
	res.TransactionRoot = newHexData(block.TxHash())
	res.StateRoot = newHexData(block.Root())
	res.Miner = newHexData(block.Coinbase())
	res.Difficulty = newHexNum(block.Difficulty())
	res.TotalDifficulty = newHexNum(block.Td)
	res.Size = newHexNum(block.Size().Int64())
	res.ExtraData = newHexData(block.Extra())
	res.GasLimit = newHexNum(block.GasLimit())
	res.GasUsed = newHexNum(block.GasUsed())
	res.UnixTimestamp = newHexNum(block.Time())

	txs := block.Transactions()
	res.Transactions = make([]*TransactionRes, len(txs))
	for i, tx := range txs {
		res.Transactions[i] = NewTransactionRes(tx)
		res.Transactions[i].BlockHash = res.BlockHash
		res.Transactions[i].BlockNumber = res.BlockNumber
		res.Transactions[i].TxIndex = newHexNum(i)
	}

	uncles := block.Uncles()
	res.Uncles = make([]*UncleRes, len(uncles))
	for i, uncle := range uncles {
		res.Uncles[i] = NewUncleRes(uncle)
	}

	return res
}

type TransactionRes struct {
	Hash        *hexdata `json:"hash"`
	Nonce       *hexnum  `json:"nonce"`
	BlockHash   *hexdata `json:"blockHash"`
	BlockNumber *hexnum  `json:"blockNumber"`
	TxIndex     *hexnum  `json:"transactionIndex"`
	From        *hexdata `json:"from"`
	To          *hexdata `json:"to"`
	Value       *hexnum  `json:"value"`
	Gas         *hexnum  `json:"gas"`
	GasPrice    *hexnum  `json:"gasPrice"`
	Input       *hexdata `json:"input"`
}

func NewTransactionRes(tx *types.Transaction) *TransactionRes {
	if tx == nil {
		return nil
	}

	var v = new(TransactionRes)
	v.Hash = newHexData(tx.Hash())
	v.Nonce = newHexNum(tx.Nonce())
	// v.BlockHash =
	// v.BlockNumber =
	// v.TxIndex =
	from, _ := tx.From()
	v.From = newHexData(from)
	v.To = newHexData(tx.To())
	v.Value = newHexNum(tx.Value())
	v.Gas = newHexNum(tx.Gas())
	v.GasPrice = newHexNum(tx.GasPrice())
	v.Input = newHexData(tx.Data())
	return v
}

type UncleRes struct {
	BlockNumber     *hexnum  `json:"number"`
	BlockHash       *hexdata `json:"hash"`
	ParentHash      *hexdata `json:"parentHash"`
	Nonce           *hexdata `json:"nonce"`
	Sha3Uncles      *hexdata `json:"sha3Uncles"`
	ReceiptHash     *hexdata `json:"receiptHash"`
	LogsBloom       *hexdata `json:"logsBloom"`
	TransactionRoot *hexdata `json:"transactionsRoot"`
	StateRoot       *hexdata `json:"stateRoot"`
	Miner           *hexdata `json:"miner"`
	Difficulty      *hexnum  `json:"difficulty"`
	ExtraData       *hexdata `json:"extraData"`
	GasLimit        *hexnum  `json:"gasLimit"`
	GasUsed         *hexnum  `json:"gasUsed"`
	UnixTimestamp   *hexnum  `json:"timestamp"`
}

func NewUncleRes(h *types.Header) *UncleRes {
	if h == nil {
		return nil
	}

	var v = new(UncleRes)
	v.BlockNumber = newHexNum(h.Number)
	v.BlockHash = newHexData(h.Hash())
	v.ParentHash = newHexData(h.ParentHash)
	v.Sha3Uncles = newHexData(h.UncleHash)
	v.Nonce = newHexData(h.Nonce[:])
	v.LogsBloom = newHexData(h.Bloom)
	v.TransactionRoot = newHexData(h.TxHash)
	v.StateRoot = newHexData(h.Root)
	v.Miner = newHexData(h.Coinbase)
	v.Difficulty = newHexNum(h.Difficulty)
	v.ExtraData = newHexData(h.Extra)
	v.GasLimit = newHexNum(h.GasLimit)
	v.GasUsed = newHexNum(h.GasUsed)
	v.UnixTimestamp = newHexNum(h.Time)
	v.ReceiptHash = newHexData(h.ReceiptHash)

	return v
}

// type FilterLogRes struct {
// 	Hash             string `json:"hash"`
// 	Address          string `json:"address"`
// 	Data             string `json:"data"`
// 	BlockNumber      string `json:"blockNumber"`
// 	TransactionHash  string `json:"transactionHash"`
// 	BlockHash        string `json:"blockHash"`
// 	TransactionIndex string `json:"transactionIndex"`
// 	LogIndex         string `json:"logIndex"`
// }

// type FilterWhisperRes struct {
// 	Hash       string `json:"hash"`
// 	From       string `json:"from"`
// 	To         string `json:"to"`
// 	Expiry     string `json:"expiry"`
// 	Sent       string `json:"sent"`
// 	Ttl        string `json:"ttl"`
// 	Topics     string `json:"topics"`
// 	Payload    string `json:"payload"`
// 	WorkProved string `json:"workProved"`
// }

type ReceiptRes struct {
	TransactionHash   *hexdata       `json:transactionHash`
	TransactionIndex  *hexnum        `json:transactionIndex`
	BlockNumber       *hexnum        `json:blockNumber`
	BlockHash         *hexdata       `json:blockHash`
	CumulativeGasUsed *hexnum        `json:cumulativeGasUsed`
	GasUsed           *hexnum        `json:gasUsed`
	ContractAddress   *hexdata       `json:contractAddress`
	Logs              *[]interface{} `json:logs`
}

func NewReceiptRes(rec *types.Receipt) *ReceiptRes {
	if rec == nil {
		return nil
	}

	var v = new(ReceiptRes)
	v.TransactionHash = newHexData(rec.TxHash)
	// v.TransactionIndex = newHexNum(input)
	// v.BlockNumber = newHexNum(input)
	// v.BlockHash = newHexData(input)
	v.CumulativeGasUsed = newHexNum(rec.CumulativeGasUsed)
	// v.GasUsed = newHexNum(input)
	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if bytes.Compare(rec.ContractAddress.Bytes(), bytes.Repeat([]byte{0}, 20)) != 0 {
		v.ContractAddress = newHexData(rec.ContractAddress)
	}
	// v.Logs = rec.Logs()

	return v
}

func numString(raw interface{}) (*big.Int, error) {
	var number *big.Int
	// Parse as integer
	num, ok := raw.(float64)
	if ok {
		number = big.NewInt(int64(num))
		return number, nil
	}

	// Parse as string/hexstring
	str, ok := raw.(string)
	if ok {
		number = common.String2Big(str)
		return number, nil
	}

	return nil, shared.NewInvalidTypeError("", "not a number or string")
}

func blockHeight(raw interface{}, number *int64) error {
	// Parse as integer
	num, ok := raw.(float64)
	if ok {
		*number = int64(num)
		return nil
	}

	// Parse as string/hexstring
	str, ok := raw.(string)
	if !ok {
		return shared.NewInvalidTypeError("", "not a number or string")
	}

	switch str {
	case "earliest":
		*number = 0
	case "latest":
		*number = -1
	case "pending":
		*number = -2
	default:
		if common.HasHexPrefix(str) {
			*number = common.String2Big(str).Int64()
		} else {
			return shared.NewInvalidTypeError("blockNumber", "is not a valid string")
		}
	}

	return nil
}

func blockHeightFromJson(msg json.RawMessage, number *int64) error {
	var raw interface{}
	if err := json.Unmarshal(msg, &raw); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}
	return blockHeight(raw, number)
}
