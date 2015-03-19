package rpc

import (
	"encoding/json"
	// "fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type BlockRes struct {
	fullTx bool

	BlockNumber     int64             `json:"number"`
	BlockHash       common.Hash       `json:"hash"`
	ParentHash      common.Hash       `json:"parentHash"`
	Nonce           [8]byte           `json:"nonce"`
	Sha3Uncles      common.Hash       `json:"sha3Uncles"`
	LogsBloom       types.Bloom       `json:"logsBloom"`
	TransactionRoot common.Hash       `json:"transactionRoot"`
	StateRoot       common.Hash       `json:"stateRoot"`
	Miner           common.Address    `json:"miner"`
	Difficulty      int64             `json:"difficulty"`
	TotalDifficulty int64             `json:"totalDifficulty"`
	Size            int64             `json:"size"`
	ExtraData       []byte            `json:"extraData"`
	GasLimit        int64             `json:"gasLimit"`
	MinGasPrice     int64             `json:"minGasPrice"`
	GasUsed         int64             `json:"gasUsed"`
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
	ext.BlockNumber = common.ToHex(big.NewInt(b.BlockNumber).Bytes())
	ext.BlockHash = b.BlockHash.Hex()
	ext.ParentHash = b.ParentHash.Hex()
	ext.Nonce = common.ToHex(b.Nonce[:])
	ext.Sha3Uncles = b.Sha3Uncles.Hex()
	ext.LogsBloom = common.ToHex(b.LogsBloom[:])
	ext.TransactionRoot = b.TransactionRoot.Hex()
	ext.StateRoot = b.StateRoot.Hex()
	ext.Miner = b.Miner.Hex()
	ext.Difficulty = common.ToHex(big.NewInt(b.Difficulty).Bytes())
	ext.TotalDifficulty = common.ToHex(big.NewInt(b.TotalDifficulty).Bytes())
	ext.Size = common.ToHex(big.NewInt(b.Size).Bytes())
	// ext.ExtraData = common.ToHex(b.ExtraData)
	ext.GasLimit = common.ToHex(big.NewInt(b.GasLimit).Bytes())
	// ext.MinGasPrice = common.ToHex(big.NewInt(b.MinGasPrice).Bytes())
	ext.GasUsed = common.ToHex(big.NewInt(b.GasUsed).Bytes())
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
	res.Uncles = make([]common.Hash, len(block.Uncles()))
	for i, uncle := range block.Uncles() {
		res.Uncles[i] = uncle.Hash()
	}
	return res
}

type TransactionRes struct {
	Hash        common.Hash     `json:"hash"`
	Nonce       int64           `json:"nonce"`
	BlockHash   common.Hash     `json:"blockHash,omitempty"`
	BlockNumber int64           `json:"blockNumber,omitempty"`
	TxIndex     int64           `json:"transactionIndex,omitempty"`
	From        common.Address  `json:"from"`
	To          *common.Address `json:"to"`
	Value       int64           `json:"value"`
	Gas         int64           `json:"gas"`
	GasPrice    int64           `json:"gasPrice"`
	Input       []byte          `json:"input"`
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

	ext.Hash = t.Hash.Hex()
	ext.Nonce = common.ToHex(big.NewInt(t.Nonce).Bytes())
	ext.BlockHash = t.BlockHash.Hex()
	ext.BlockNumber = common.ToHex(big.NewInt(t.BlockNumber).Bytes())
	ext.TxIndex = common.ToHex(big.NewInt(t.TxIndex).Bytes())
	ext.From = t.From.Hex()
	if t.To == nil {
		ext.To = "0x00"
	} else {
		ext.To = t.To.Hex()
	}
	ext.Value = common.ToHex(big.NewInt(t.Value).Bytes())
	ext.Gas = common.ToHex(big.NewInt(t.Gas).Bytes())
	ext.GasPrice = common.ToHex(big.NewInt(t.GasPrice).Bytes())
	ext.Input = common.ToHex(t.Input)

	return json.Marshal(ext)
}

func NewTransactionRes(tx *types.Transaction) *TransactionRes {
	var v = new(TransactionRes)
	v.Hash = tx.Hash()
	v.Nonce = int64(tx.Nonce())
	v.From, _ = tx.From()
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
