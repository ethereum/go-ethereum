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
		BlockNumber     *hexnum       `json:"number"`
		BlockHash       *hexdata      `json:"hash"`
		ParentHash      *hexdata      `json:"parentHash"`
		Nonce           *hexnum       `json:"nonce"`
		Sha3Uncles      *hexdata      `json:"sha3Uncles"`
		LogsBloom       *hexdata      `json:"logsBloom"`
		TransactionRoot *hexdata      `json:"transactionRoot"`
		StateRoot       *hexdata      `json:"stateRoot"`
		Miner           *hexdata      `json:"miner"`
		Difficulty      *hexnum       `json:"difficulty"`
		TotalDifficulty *hexnum       `json:"totalDifficulty"`
		Size            *hexnum       `json:"size"`
		ExtraData       *hexdata      `json:"extraData"`
		GasLimit        *hexnum       `json:"gasLimit"`
		MinGasPrice     *hexnum       `json:"minGasPrice"`
		GasUsed         *hexnum       `json:"gasUsed"`
		UnixTimestamp   *hexnum       `json:"timestamp"`
		Transactions    []interface{} `json:"transactions"`
		Uncles          []*hexdata    `json:"uncles"`
	}

	// convert strict types to hexified strings
	ext.BlockNumber = newHexNum(b.BlockNumber.Bytes())
	ext.BlockHash = newHexData(b.BlockHash.Bytes())
	ext.ParentHash = newHexData(b.ParentHash.Bytes())
	ext.Nonce = newHexNum(b.Nonce[:])
	ext.Sha3Uncles = newHexData(b.Sha3Uncles.Bytes())
	ext.LogsBloom = newHexData(b.LogsBloom.Bytes())
	ext.TransactionRoot = newHexData(b.TransactionRoot.Bytes())
	ext.StateRoot = newHexData(b.StateRoot.Bytes())
	ext.Miner = newHexData(b.Miner.Bytes())
	ext.Difficulty = newHexNum(b.Difficulty.Bytes())
	ext.TotalDifficulty = newHexNum(b.TotalDifficulty.Bytes())
	ext.Size = newHexNum(b.Size.Bytes())
	ext.ExtraData = newHexData(b.ExtraData)
	ext.GasLimit = newHexNum(b.GasLimit.Bytes())
	// ext.MinGasPrice = newHexNum(big.NewInt(b.MinGasPrice).Bytes())
	ext.GasUsed = newHexNum(b.GasUsed.Bytes())
	ext.UnixTimestamp = newHexNum(big.NewInt(b.UnixTimestamp).Bytes())
	ext.Transactions = make([]interface{}, len(b.Transactions))
	if b.fullTx {
		for i, tx := range b.Transactions {
			ext.Transactions[i] = tx
		}
	} else {
		for i, tx := range b.Transactions {
			ext.Transactions[i] = newHexData(tx.Hash.Bytes())
		}
	}
	ext.Uncles = make([]*hexdata, len(b.Uncles))
	for i, v := range b.Uncles {
		ext.Uncles[i] = newHexData(v.Bytes())
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
	res.ExtraData = []byte(block.Header().Extra)
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
	BlockHash   common.Hash     `json:"blockHash"`
	BlockNumber int64           `json:"blockNumber"`
	TxIndex     int64           `json:"transactionIndex"`
	From        common.Address  `json:"from"`
	To          *common.Address `json:"to"`
	Value       *big.Int        `json:"value"`
	Gas         *big.Int        `json:"gas"`
	GasPrice    *big.Int        `json:"gasPrice"`
	Input       []byte          `json:"input"`
}

func (t *TransactionRes) MarshalJSON() ([]byte, error) {
	var ext struct {
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

	ext.Hash = newHexData(t.Hash.Bytes())
	ext.Nonce = newHexNum(big.NewInt(int64(t.Nonce)).Bytes())
	ext.BlockHash = newHexData(t.BlockHash.Bytes())
	ext.BlockNumber = newHexNum(big.NewInt(t.BlockNumber).Bytes())
	ext.TxIndex = newHexNum(big.NewInt(t.TxIndex).Bytes())
	ext.From = newHexData(t.From.Bytes())
	ext.To = newHexData(t.To.Bytes())
	ext.Value = newHexNum(t.Value.Bytes())
	ext.Gas = newHexNum(t.Gas.Bytes())
	ext.GasPrice = newHexNum(t.GasPrice.Bytes())
	ext.Input = newHexData(t.Input)

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

type LogRes struct {
	Address          common.Address `json:"address"`
	Topics           []common.Hash  `json:"topics"`
	Data             []byte         `json:"data"`
	BlockNumber      uint64         `json:"blockNumber"`
	Hash             common.Hash    `json:"hash"`
	LogIndex         uint64         `json:"logIndex"`
	BlockHash        common.Hash    `json:"blockHash"`
	TransactionHash  common.Hash    `json:"transactionHash"`
	TransactionIndex uint64         `json:"transactionIndex"`
}

func NewLogRes(log state.Log) LogRes {
	var l LogRes
	l.Topics = make([]common.Hash, len(log.Topics()))
	l.Address = log.Address()
	l.Data = log.Data()
	l.BlockNumber = log.Number()
	for j, topic := range log.Topics() {
		l.Topics[j] = topic
	}
	return l
}

func (l *LogRes) MarshalJSON() ([]byte, error) {
	var ext struct {
		Address          *hexdata   `json:"address"`
		Topics           []*hexdata `json:"topics"`
		Data             *hexdata   `json:"data"`
		BlockNumber      *hexnum    `json:"blockNumber"`
		Hash             *hexdata   `json:"hash"`
		LogIndex         *hexnum    `json:"logIndex"`
		BlockHash        *hexdata   `json:"blockHash"`
		TransactionHash  *hexdata   `json:"transactionHash"`
		TransactionIndex *hexnum    `json:"transactionIndex"`
	}

	ext.Address = newHexData(l.Address.Bytes())
	ext.Data = newHexData(l.Data)
	ext.BlockNumber = newHexNum(l.BlockNumber)
	ext.Topics = make([]*hexdata, len(l.Topics))
	for i, v := range l.Topics {
		ext.Topics[i] = newHexData(v)
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
