package rpc

import (
	"encoding/json"
	// "fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

type BlockRes struct {
	fullTx bool

	BlockNumber     *big.Int          `json:"number"`
	BlockHash       common.Hash       `json:"hash"`
	ParentHash      common.Hash       `json:"parentHash"`
	Nonce           [8]byte           `json:"nonce"`
	Sha3Uncles      common.Hash       `json:"sha3Uncles"`
	LogsBloom       types.Bloom       `json:"logsBloom"`
	TransactionRoot common.Hash       `json:"transactionRoot"`
	StateRoot       common.Hash       `json:"stateRoot"`
	Miner           common.Address    `json:"miner"`
	Difficulty      *big.Int          `json:"difficulty"`
	TotalDifficulty *big.Int          `json:"totalDifficulty"`
	Size            *big.Int          `json:"size"`
	ExtraData       []byte            `json:"extraData"`
	GasLimit        *big.Int          `json:"gasLimit"`
	MinGasPrice     int64             `json:"minGasPrice"`
	GasUsed         *big.Int          `json:"gasUsed"`
	UnixTimestamp   int64             `json:"timestamp"`
	Transactions    []*TransactionRes `json:"transactions"`
	Uncles          []common.Hash     `json:"uncles"`
}

func (b *BlockRes) MarshalJSON() ([]byte, error) {
	var ext struct {
		BlockNumber     string        `json:"number"`
		BlockHash       string        `json:"hash"`
		ParentHash      string        `json:"parentHash"`
		Nonce           string        `json:"nonce"`
		Sha3Uncles      string        `json:"sha3Uncles"`
		LogsBloom       string        `json:"logsBloom"`
		TransactionRoot string        `json:"transactionRoot"`
		StateRoot       string        `json:"stateRoot"`
		Miner           string        `json:"miner"`
		Difficulty      string        `json:"difficulty"`
		TotalDifficulty string        `json:"totalDifficulty"`
		Size            string        `json:"size"`
		ExtraData       string        `json:"extraData"`
		GasLimit        string        `json:"gasLimit"`
		MinGasPrice     string        `json:"minGasPrice"`
		GasUsed         string        `json:"gasUsed"`
		UnixTimestamp   string        `json:"timestamp"`
		Transactions    []interface{} `json:"transactions"`
		Uncles          []string      `json:"uncles"`
	}

	// convert strict types to hexified strings
	ext.BlockNumber = common.ToHex(b.BlockNumber.Bytes())
	ext.BlockHash = b.BlockHash.Hex()
	ext.ParentHash = b.ParentHash.Hex()
	ext.Nonce = common.ToHex(b.Nonce[:])
	ext.Sha3Uncles = b.Sha3Uncles.Hex()
	ext.LogsBloom = common.ToHex(b.LogsBloom[:])
	ext.TransactionRoot = b.TransactionRoot.Hex()
	ext.StateRoot = b.StateRoot.Hex()
	ext.Miner = b.Miner.Hex()
	ext.Difficulty = common.ToHex(b.Difficulty.Bytes())
	ext.TotalDifficulty = common.ToHex(b.TotalDifficulty.Bytes())
	ext.Size = common.ToHex(b.Size.Bytes())
	// ext.ExtraData = common.ToHex(b.ExtraData)
	ext.GasLimit = common.ToHex(b.GasLimit.Bytes())
	// ext.MinGasPrice = common.ToHex(big.NewInt(b.MinGasPrice).Bytes())
	ext.GasUsed = common.ToHex(b.GasUsed.Bytes())
	ext.UnixTimestamp = common.ToHex(big.NewInt(b.UnixTimestamp).Bytes())
	ext.Transactions = make([]interface{}, len(b.Transactions))
	if b.fullTx {
		for i, tx := range b.Transactions {
			ext.Transactions[i] = tx
		}
	} else {
		for i, tx := range b.Transactions {
			ext.Transactions[i] = tx.Hash.Hex()
		}
	}
	ext.Uncles = make([]string, len(b.Uncles))
	for i, v := range b.Uncles {
		ext.Uncles[i] = v.Hex()
	}

	return json.Marshal(ext)
}

func NewBlockRes(block *types.Block) *BlockRes {
	if block == nil {
		return &BlockRes{}
	}

	res := new(BlockRes)
	res.BlockNumber = block.Number()
	res.BlockHash = block.Hash()
	res.ParentHash = block.ParentHash()
	res.Nonce = block.Header().Nonce
	res.Sha3Uncles = block.Header().UncleHash
	res.LogsBloom = block.Bloom()
	res.TransactionRoot = block.Header().TxHash
	res.StateRoot = block.Root()
	res.Miner = block.Header().Coinbase
	res.Difficulty = block.Difficulty()
	res.TotalDifficulty = block.Td
	res.Size = big.NewInt(int64(block.Size()))
	// res.ExtraData =
	res.GasLimit = block.GasLimit()
	// res.MinGasPrice =
	res.GasUsed = block.GasUsed()
	res.UnixTimestamp = block.Time()
	res.Transactions = make([]*TransactionRes, len(block.Transactions()))
	for i, tx := range block.Transactions() {
		v := NewTransactionRes(tx)
		v.BlockHash = block.Hash()
		v.BlockNumber = block.Number().Int64()
		v.TxIndex = int64(i)
		res.Transactions[i] = v
	}
	res.Uncles = make([]common.Hash, len(block.Uncles()))
	for i, uncle := range block.Uncles() {
		res.Uncles[i] = uncle.Hash()
	}
	return res
}

type TransactionRes struct {
	Hash        common.Hash     `json:"hash"`
	Nonce       uint64          `json:"nonce"`
	BlockHash   common.Hash     `json:"blockHash,omitempty"`
	BlockNumber int64           `json:"blockNumber,omitempty"`
	TxIndex     int64           `json:"transactionIndex,omitempty"`
	From        common.Address  `json:"from"`
	To          *common.Address `json:"to"`
	Value       *big.Int        `json:"value"`
	Gas         *big.Int        `json:"gas"`
	GasPrice    *big.Int        `json:"gasPrice"`
	Input       []byte          `json:"input"`
}

func (t *TransactionRes) MarshalJSON() ([]byte, error) {
	var ext struct {
		Hash        string      `json:"hash"`
		Nonce       string      `json:"nonce"`
		BlockHash   string      `json:"blockHash,omitempty"`
		BlockNumber string      `json:"blockNumber,omitempty"`
		TxIndex     string      `json:"transactionIndex,omitempty"`
		From        string      `json:"from"`
		To          interface{} `json:"to"`
		Value       string      `json:"value"`
		Gas         string      `json:"gas"`
		GasPrice    string      `json:"gasPrice"`
		Input       string      `json:"input"`
	}

	ext.Hash = t.Hash.Hex()
	ext.Nonce = common.ToHex(big.NewInt(int64(t.Nonce)).Bytes())
	ext.BlockHash = t.BlockHash.Hex()
	ext.BlockNumber = common.ToHex(big.NewInt(t.BlockNumber).Bytes())
	ext.TxIndex = common.ToHex(big.NewInt(t.TxIndex).Bytes())
	ext.From = t.From.Hex()
	if t.To == nil {
		ext.To = nil
	} else {
		ext.To = t.To.Hex()
	}
	ext.Value = common.ToHex(t.Value.Bytes())
	ext.Gas = common.ToHex(t.Gas.Bytes())
	ext.GasPrice = common.ToHex(t.GasPrice.Bytes())
	ext.Input = common.ToHex(t.Input)

	return json.Marshal(ext)
}

func NewTransactionRes(tx *types.Transaction) *TransactionRes {
	var v = new(TransactionRes)
	v.Hash = tx.Hash()
	v.Nonce = tx.Nonce()
	v.From, _ = tx.From()
	v.To = tx.To()
	v.Value = tx.Value()
	v.Gas = tx.Gas()
	v.GasPrice = tx.GasPrice()
	v.Input = tx.Data()
	return v
}

type FilterLogRes struct {
	Hash             string `json:"hash"`
	Address          string `json:"address"`
	Data             string `json:"data"`
	BlockNumber      string `json:"blockNumber"`
	TransactionHash  string `json:"transactionHash"`
	BlockHash        string `json:"blockHash"`
	TransactionIndex string `json:"transactionIndex"`
	LogIndex         string `json:"logIndex"`
}

type FilterWhisperRes struct {
	Hash       string `json:"hash"`
	From       string `json:"from"`
	To         string `json:"to"`
	Expiry     string `json:"expiry"`
	Sent       string `json:"sent"`
	Ttl        string `json:"ttl"`
	Topics     string `json:"topics"`
	Payload    string `json:"payload"`
	WorkProved string `json:"workProved"`
}

type LogRes struct {
	Address common.Address `json:"address"`
	Topics  []common.Hash  `json:"topics"`
	Data    []byte         `json:"data"`
	Number  uint64         `json:"number"`
}

func NewLogRes(log state.Log) LogRes {
	var l LogRes
	l.Topics = make([]common.Hash, len(log.Topics()))
	l.Address = log.Address()
	l.Data = log.Data()
	l.Number = log.Number()
	for j, topic := range log.Topics() {
		l.Topics[j] = topic
	}
	return l
}

func (l *LogRes) MarshalJSON() ([]byte, error) {
	var ext struct {
		Address string   `json:"address"`
		Topics  []string `json:"topics"`
		Data    string   `json:"data"`
		Number  string   `json:"number"`
	}

	ext.Address = l.Address.Hex()
	ext.Data = common.Bytes2Hex(l.Data)
	ext.Number = common.Bytes2Hex(big.NewInt(int64(l.Number)).Bytes())
	ext.Topics = make([]string, len(l.Topics))
	for i, v := range l.Topics {
		ext.Topics[i] = v.Hex()
	}

	return json.Marshal(ext)
}

func NewLogsRes(logs state.Logs) (ls []LogRes) {
	ls = make([]LogRes, len(logs))

	for i, log := range logs {
		ls[i] = NewLogRes(log)
	}

	return
}
