package rpc

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestNewBlockRes(t *testing.T) {
	parentHash := common.HexToHash("0x01")
	coinbase := common.HexToAddress("0x01")
	root := common.HexToHash("0x01")
	difficulty := common.Big1
	nonce := uint64(1)
	extra := ""
	block := types.NewBlock(parentHash, coinbase, root, difficulty, nonce, extra)

	_ = NewBlockRes(block)
}

func TestNewTransactionRes(t *testing.T) {
	to := common.HexToAddress("0x02")
	amount := big.NewInt(1)
	gasAmount := big.NewInt(1)
	gasPrice := big.NewInt(1)
	data := []byte{1, 2, 3}
	tx := types.NewTransactionMessage(to, amount, gasAmount, gasPrice, data)

	_ = NewTransactionRes(tx)
}

func MakeStateLog(num int) state.Log {
	address := common.HexToAddress("0x0")
	data := []byte{1, 2, 3}
	number := uint64(num)
	topics := make([]common.Hash, 3)
	topics = append(topics, common.HexToHash("0x00"))
	topics = append(topics, common.HexToHash("0x10"))
	topics = append(topics, common.HexToHash("0x20"))
	log := state.NewLog(address, topics, data, number)
	return log
}

func TestNewLogRes(t *testing.T) {
	log := MakeStateLog(0)
	_ = NewLogRes(log)
}

func TestNewLogsRes(t *testing.T) {
	logs := make([]state.Log, 3)
	logs[0] = MakeStateLog(1)
	logs[1] = MakeStateLog(2)
	logs[2] = MakeStateLog(3)
	_ = NewLogsRes(logs)
}
