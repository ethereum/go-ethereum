package rpc

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

type BlockRes struct {
	fullTx bool

	BlockNumber     int64             `json:"number"`
	BlockHash       []byte            `json:"hash"`
	ParentHash      []byte            `json:"parentHash"`
	Nonce           []byte            `json:"nonce"`
	Sha3Uncles      []byte            `json:"sha3Uncles"`
	LogsBloom       []byte            `json:"logsBloom"`
	TransactionRoot []byte            `json:"transactionRoot"`
	StateRoot       []byte            `json:"stateRoot"`
	Miner           []byte            `json:"miner"`
	Difficulty      int64             `json:"difficulty"`
	TotalDifficulty int64             `json:"totalDifficulty"`
	Size            int64             `json:"size"`
	ExtraData       []byte            `json:"extraData"`
	GasLimit        int64             `json:"gasLimit"`
	MinGasPrice     int64             `json:"minGasPrice"`
	GasUsed         int64             `json:"gasUsed"`
	UnixTimestamp   int64             `json:"timestamp"`
	Transactions    []*TransactionRes `json:"transactions"`
	Uncles          [][]byte          `json:"uncles"`
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
	ext.BlockNumber = toHex(big.NewInt(b.BlockNumber).Bytes())
	ext.BlockHash = toHex(b.BlockHash)
	ext.ParentHash = toHex(b.ParentHash)
	ext.Nonce = toHex(b.Nonce)
	ext.Sha3Uncles = toHex(b.Sha3Uncles)
	ext.LogsBloom = toHex(b.LogsBloom)
	ext.TransactionRoot = toHex(b.TransactionRoot)
	ext.StateRoot = toHex(b.StateRoot)
	ext.Miner = toHex(b.Miner)
	ext.Difficulty = toHex(big.NewInt(b.Difficulty).Bytes())
	ext.TotalDifficulty = toHex(big.NewInt(b.TotalDifficulty).Bytes())
	ext.Size = toHex(big.NewInt(b.Size).Bytes())
	// ext.ExtraData = toHex(b.ExtraData)
	ext.GasLimit = toHex(big.NewInt(b.GasLimit).Bytes())
	// ext.MinGasPrice = toHex(big.NewInt(b.MinGasPrice).Bytes())
	ext.GasUsed = toHex(big.NewInt(b.GasUsed).Bytes())
	ext.UnixTimestamp = toHex(big.NewInt(b.UnixTimestamp).Bytes())
	ext.Transactions = make([]interface{}, len(b.Transactions))
	if b.fullTx {
		for i, tx := range b.Transactions {
			ext.Transactions[i] = tx
		}
	} else {
		for i, tx := range b.Transactions {
			ext.Transactions[i] = toHex(tx.Hash)
		}
	}
	ext.Uncles = make([]string, len(b.Uncles))
	for i, v := range b.Uncles {
		ext.Uncles[i] = toHex(v)
	}

	return json.Marshal(ext)
}

func NewBlockRes(block *types.Block) *BlockRes {
	res := new(BlockRes)
	res.BlockNumber = block.Number().Int64()
	res.BlockHash = block.Hash()
	res.ParentHash = block.ParentHash()
	res.Nonce = block.Header().Nonce
	res.Sha3Uncles = block.Header().UncleHash
	res.LogsBloom = block.Bloom()
	res.TransactionRoot = block.Header().TxHash
	res.StateRoot = block.Root()
	res.Miner = block.Header().Coinbase
	res.Difficulty = block.Difficulty().Int64()
	if block.Td != nil {
		res.TotalDifficulty = block.Td.Int64()
	}
	res.Size = int64(block.Size())
	// res.ExtraData =
	res.GasLimit = block.GasLimit().Int64()
	// res.MinGasPrice =
	res.GasUsed = block.GasUsed().Int64()
	res.UnixTimestamp = block.Time()
	res.Transactions = make([]*TransactionRes, len(block.Transactions()))
	for i, tx := range block.Transactions() {
		v := NewTransactionRes(tx)
		v.BlockHash = block.Hash()
		v.BlockNumber = block.Number().Int64()
		v.TxIndex = int64(i)
		res.Transactions[i] = v
	}
	res.Uncles = make([][]byte, len(block.Uncles()))
	for i, uncle := range block.Uncles() {
		res.Uncles[i] = uncle.Hash()
	}
	return res
}

type TransactionRes struct {
	Hash        []byte `json:"hash"`
	Nonce       int64  `json:"nonce"`
	BlockHash   []byte `json:"blockHash,omitempty"`
	BlockNumber int64  `json:"blockNumber,omitempty"`
	TxIndex     int64  `json:"transactionIndex,omitempty"`
	From        []byte `json:"from"`
	To          []byte `json:"to"`
	Value       int64  `json:"value"`
	Gas         int64  `json:"gas"`
	GasPrice    int64  `json:"gasPrice"`
	Input       []byte `json:"input"`
}

func (t *TransactionRes) MarshalJSON() ([]byte, error) {
	var ext struct {
		Hash        string `json:"hash"`
		Nonce       string `json:"nonce"`
		BlockHash   string `json:"blockHash,omitempty"`
		BlockNumber string `json:"blockNumber,omitempty"`
		TxIndex     string `json:"transactionIndex,omitempty"`
		From        string `json:"from"`
		To          string `json:"to"`
		Value       string `json:"value"`
		Gas         string `json:"gas"`
		GasPrice    string `json:"gasPrice"`
		Input       string `json:"input"`
	}

	ext.Hash = toHex(t.Hash)
	ext.Nonce = toHex(big.NewInt(t.Nonce).Bytes())
	ext.BlockHash = toHex(t.BlockHash)
	ext.BlockNumber = toHex(big.NewInt(t.BlockNumber).Bytes())
	ext.TxIndex = toHex(big.NewInt(t.TxIndex).Bytes())
	ext.From = toHex(t.From)
	ext.To = toHex(t.To)
	ext.Value = toHex(big.NewInt(t.Value).Bytes())
	ext.Gas = toHex(big.NewInt(t.Gas).Bytes())
	ext.GasPrice = toHex(big.NewInt(t.GasPrice).Bytes())
	ext.Input = toHex(t.Input)

	return json.Marshal(ext)
}

func NewTransactionRes(tx *types.Transaction) *TransactionRes {
	var v = new(TransactionRes)
	v.Hash = tx.Hash()
	v.Nonce = int64(tx.Nonce())
	v.From = tx.From()
	v.To = tx.To()
	v.Value = tx.Value().Int64()
	v.Gas = tx.Gas().Int64()
	v.GasPrice = tx.GasPrice().Int64()
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
