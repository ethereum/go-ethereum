package rpc

import (
// "encoding/json"
// "math/big"
// "testing"

// "github.com/ethereum/go-ethereum/common"
// "github.com/ethereum/go-ethereum/core/state"
// "github.com/ethereum/go-ethereum/core/types"
)

// func TestNewBlockRes(t *testing.T) {
// 	parentHash := common.HexToHash("0x01")
// 	coinbase := common.HexToAddress("0x01")
// 	root := common.HexToHash("0x01")
// 	difficulty := common.Big1
// 	nonce := uint64(1)
// 	extra := ""
// 	block := types.NewBlock(parentHash, coinbase, root, difficulty, nonce, extra)

// 	_ = NewBlockRes(block)
// }

// func TestBlockRes(t *testing.T) {
// 	v := &BlockRes{
// 		BlockNumber:     big.NewInt(0),
// 		BlockHash:       common.HexToHash("0x0"),
// 		ParentHash:      common.HexToHash("0x0"),
// 		Nonce:           [8]byte{0, 0, 0, 0, 0, 0, 0, 0},
// 		Sha3Uncles:      common.HexToHash("0x0"),
// 		LogsBloom:       types.BytesToBloom([]byte{0}),
// 		TransactionRoot: common.HexToHash("0x0"),
// 		StateRoot:       common.HexToHash("0x0"),
// 		Miner:           common.HexToAddress("0x0"),
// 		Difficulty:      big.NewInt(0),
// 		TotalDifficulty: big.NewInt(0),
// 		Size:            big.NewInt(0),
// 		ExtraData:       []byte{},
// 		GasLimit:        big.NewInt(0),
// 		MinGasPrice:     int64(0),
// 		GasUsed:         big.NewInt(0),
// 		UnixTimestamp:   int64(0),
// 		// Transactions    []*TransactionRes `json:"transactions"`
// 		// Uncles          []common.Hash     `json:"uncles"`
// 	}

// 	_, _ = json.Marshal(v)

// 	// fmt.Println(string(j))

// }

// func TestTransactionRes(t *testing.T) {
// 	a := common.HexToAddress("0x0")
// 	v := &TransactionRes{
// 		Hash:        common.HexToHash("0x0"),
// 		Nonce:       uint64(0),
// 		BlockHash:   common.HexToHash("0x0"),
// 		BlockNumber: int64(0),
// 		TxIndex:     int64(0),
// 		From:        common.HexToAddress("0x0"),
// 		To:          &a,
// 		Value:       big.NewInt(0),
// 		Gas:         big.NewInt(0),
// 		GasPrice:    big.NewInt(0),
// 		Input:       []byte{0},
// 	}

// 	_, _ = json.Marshal(v)
// }

// func TestNewTransactionRes(t *testing.T) {
// 	to := common.HexToAddress("0x02")
// 	amount := big.NewInt(1)
// 	gasAmount := big.NewInt(1)
// 	gasPrice := big.NewInt(1)
// 	data := []byte{1, 2, 3}
// 	tx := types.NewTransactionMessage(to, amount, gasAmount, gasPrice, data)

// 	_ = NewTransactionRes(tx)
// }

// func TestLogRes(t *testing.T) {
// 	topics := make([]common.Hash, 3)
// 	topics = append(topics, common.HexToHash("0x00"))
// 	topics = append(topics, common.HexToHash("0x10"))
// 	topics = append(topics, common.HexToHash("0x20"))

// 	v := &LogRes{
// 		Topics:      topics,
// 		Address:     common.HexToAddress("0x0"),
// 		Data:        []byte{1, 2, 3},
// 		BlockNumber: uint64(5),
// 	}

// 	_, _ = json.Marshal(v)
// }

// func MakeStateLog(num int) state.Log {
// 	address := common.HexToAddress("0x0")
// 	data := []byte{1, 2, 3}
// 	number := uint64(num)
// 	topics := make([]common.Hash, 3)
// 	topics = append(topics, common.HexToHash("0x00"))
// 	topics = append(topics, common.HexToHash("0x10"))
// 	topics = append(topics, common.HexToHash("0x20"))
// 	log := state.NewLog(address, topics, data, number)
// 	return log
// }

// func TestNewLogRes(t *testing.T) {
// 	log := MakeStateLog(0)
// 	_ = NewLogRes(log)
// }

// func TestNewLogsRes(t *testing.T) {
// 	logs := make([]state.Log, 3)
// 	logs[0] = MakeStateLog(1)
// 	logs[1] = MakeStateLog(2)
// 	logs[2] = MakeStateLog(3)
// 	_ = NewLogsRes(logs)
// }
