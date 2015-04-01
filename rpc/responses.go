package rpc

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

type BlockRes struct {
	fullTx bool

	BlockNumber     *hexnum           `json:"number"`
	BlockHash       *hexdata          `json:"hash"`
	ParentHash      *hexdata          `json:"parentHash"`
	Nonce           *hexdata          `json:"nonce"`
	Sha3Uncles      *hexdata          `json:"sha3Uncles"`
	LogsBloom       *hexdata          `json:"logsBloom"`
	TransactionRoot *hexdata          `json:"transactionRoot"`
	StateRoot       *hexdata          `json:"stateRoot"`
	Miner           *hexdata          `json:"miner"`
	Difficulty      *hexnum           `json:"difficulty"`
	TotalDifficulty *hexnum           `json:"totalDifficulty"`
	Size            *hexnum           `json:"size"`
	ExtraData       *hexdata          `json:"extraData"`
	GasLimit        *hexnum           `json:"gasLimit"`
	MinGasPrice     *hexnum           `json:"minGasPrice"`
	GasUsed         *hexnum           `json:"gasUsed"`
	UnixTimestamp   *hexnum           `json:"timestamp"`
	Transactions    []*TransactionRes `json:"transactions"`
	Uncles          []*hexdata        `json:"uncles"`
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
			TransactionRoot *hexdata          `json:"transactionRoot"`
			StateRoot       *hexdata          `json:"stateRoot"`
			Miner           *hexdata          `json:"miner"`
			Difficulty      *hexnum           `json:"difficulty"`
			TotalDifficulty *hexnum           `json:"totalDifficulty"`
			Size            *hexnum           `json:"size"`
			ExtraData       *hexdata          `json:"extraData"`
			GasLimit        *hexnum           `json:"gasLimit"`
			MinGasPrice     *hexnum           `json:"minGasPrice"`
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
		ext.MinGasPrice = b.MinGasPrice
		ext.GasUsed = b.GasUsed
		ext.UnixTimestamp = b.UnixTimestamp
		ext.Transactions = b.Transactions
		ext.Uncles = b.Uncles
		return json.Marshal(ext)
	} else {
		var ext struct {
			BlockNumber     *hexnum    `json:"number"`
			BlockHash       *hexdata   `json:"hash"`
			ParentHash      *hexdata   `json:"parentHash"`
			Nonce           *hexdata   `json:"nonce"`
			Sha3Uncles      *hexdata   `json:"sha3Uncles"`
			LogsBloom       *hexdata   `json:"logsBloom"`
			TransactionRoot *hexdata   `json:"transactionRoot"`
			StateRoot       *hexdata   `json:"stateRoot"`
			Miner           *hexdata   `json:"miner"`
			Difficulty      *hexnum    `json:"difficulty"`
			TotalDifficulty *hexnum    `json:"totalDifficulty"`
			Size            *hexnum    `json:"size"`
			ExtraData       *hexdata   `json:"extraData"`
			GasLimit        *hexnum    `json:"gasLimit"`
			MinGasPrice     *hexnum    `json:"minGasPrice"`
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
		ext.MinGasPrice = b.MinGasPrice
		ext.GasUsed = b.GasUsed
		ext.UnixTimestamp = b.UnixTimestamp
		ext.Transactions = make([]*hexdata, len(b.Transactions))
		for i, tx := range b.Transactions {
			ext.Transactions[i] = tx.Hash
		}
		ext.Uncles = b.Uncles
		return json.Marshal(ext)
	}
}

func NewBlockRes(block *types.Block, fullTx bool) *BlockRes {
	// TODO respect fullTx flag

	if block == nil {
		return &BlockRes{}
	}

	res := new(BlockRes)
	res.fullTx = fullTx
	res.BlockNumber = newHexNum(block.Number())
	res.BlockHash = newHexData(block.Hash())
	res.ParentHash = newHexData(block.ParentHash())
	res.Nonce = newHexData(block.Header().Nonce)
	res.Sha3Uncles = newHexData(block.Header().UncleHash)
	res.LogsBloom = newHexData(block.Bloom())
	res.TransactionRoot = newHexData(block.Header().TxHash)
	res.StateRoot = newHexData(block.Root())
	res.Miner = newHexData(block.Header().Coinbase)
	res.Difficulty = newHexNum(block.Difficulty())
	res.TotalDifficulty = newHexNum(block.Td)
	res.Size = newHexNum(block.Size().Int64())
	res.ExtraData = newHexData(block.Header().Extra)
	res.GasLimit = newHexNum(block.GasLimit())
	// res.MinGasPrice =
	res.GasUsed = newHexNum(block.GasUsed())
	res.UnixTimestamp = newHexNum(block.Time())

	res.Transactions = make([]*TransactionRes, len(block.Transactions()))
	for i, tx := range block.Transactions() {
		res.Transactions[i] = NewTransactionRes(tx)
		res.Transactions[i].BlockHash = res.BlockHash
		res.Transactions[i].BlockNumber = res.BlockNumber
		res.Transactions[i].TxIndex = newHexNum(i)
	}

	res.Uncles = make([]*hexdata, len(block.Uncles()))
	for i, uncle := range block.Uncles() {
		res.Uncles[i] = newHexData(uncle.Hash())
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

func NewLogRes(log state.Log) LogRes {
	var l LogRes
	l.Topics = make([]*hexdata, len(log.Topics()))
	for j, topic := range log.Topics() {
		l.Topics[j] = newHexData(topic)
	}
	l.Address = newHexData(log.Address())
	l.Data = newHexData(log.Data())
	l.BlockNumber = newHexNum(log.Number())

	return l
}

func NewLogsRes(logs state.Logs) (ls []LogRes) {
	ls = make([]LogRes, len(logs))

	for i, log := range logs {
		ls[i] = NewLogRes(log)
	}

	return
}
