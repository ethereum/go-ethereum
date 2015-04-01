package rpc

import (
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	reHash       = `"0x[0-9a-f]{64}"`                    // 32 bytes
	reHashOpt    = `"(0x[0-9a-f]{64})"|null`             // 32 bytes or null
	reAddress    = `"0x[0-9a-f]{40}"`                    // 20 bytes
	reAddressOpt = `"0x[0-9a-f]{40}"|null`               // 20 bytes or null
	reNum        = `"0x([1-9a-f][0-9a-f]{0,15})|0"`      // must not have left-padded zeros
	reNumOpt     = `"0x([1-9a-f][0-9a-f]{0,15})|0"|null` // must not have left-padded zeros or null
	reData       = `"0x[0-9a-f]*"`                       // can be "empty"
)

func TestNewBlockRes(t *testing.T) {
	parentHash := common.HexToHash("0x01")
	coinbase := common.HexToAddress("0x01")
	root := common.HexToHash("0x01")
	difficulty := common.Big1
	nonce := uint64(1)
	extra := ""
	block := types.NewBlock(parentHash, coinbase, root, difficulty, nonce, extra)
	tests := map[string]string{
		"number":          reNum,
		"hash":            reHash,
		"parentHash":      reHash,
		"nonce":           reData,
		"sha3Uncles":      reHash,
		"logsBloom":       reData,
		"transactionRoot": reHash,
		"stateRoot":       reHash,
		"miner":           reAddress,
		"difficulty":      `"0x1"`,
		"totalDifficulty": reNum,
		"size":            reNum,
		"extraData":       reData,
		"gasLimit":        reNum,
		// "minGasPrice":  "0x",
		"gasUsed":   reNum,
		"timestamp": reNum,
	}

	v := NewBlockRes(block, false)
	j, _ := json.Marshal(v)

	for k, re := range tests {
		match, _ := regexp.MatchString(fmt.Sprintf(`{.*"%s":%s.*}`, k, re), string(j))
		if !match {
			t.Error(fmt.Sprintf("%s output json does not match format %s. Got %s", k, re, j))
		}
	}
}

func TestNewTransactionRes(t *testing.T) {
	to := common.HexToAddress("0x02")
	amount := big.NewInt(1)
	gasAmount := big.NewInt(1)
	gasPrice := big.NewInt(1)
	data := []byte{1, 2, 3}
	tx := types.NewTransactionMessage(to, amount, gasAmount, gasPrice, data)

	tests := map[string]string{
		"hash":             reHash,
		"nonce":            reNum,
		"blockHash":        reHashOpt,
		"blockNum":         reNumOpt,
		"transactionIndex": reNumOpt,
		"from":             reAddress,
		"to":               reAddressOpt,
		"value":            reNum,
		"gas":              reNum,
		"gasPrice":         reNum,
		"input":            reData,
	}

	v := NewTransactionRes(tx)
	v.BlockHash = newHexData(common.HexToHash("0x030201"))
	v.BlockNumber = newHexNum(5)
	v.TxIndex = newHexNum(0)
	j, _ := json.Marshal(v)
	for k, re := range tests {
		match, _ := regexp.MatchString(fmt.Sprintf(`{.*"%s":%s.*}`, k, re), string(j))
		if !match {
			t.Error(fmt.Sprintf("`%s` output json does not match format %s. Source %s", k, re, j))
		}
	}

}

func TestNewLogRes(t *testing.T) {
	log := makeStateLog(0)
	tests := map[string]string{
		"address": reAddress,
		// "topics": "[.*]"
		"data":        reData,
		"blockNumber": reNum,
		// "hash":             reHash,
		// "logIndex":         reNum,
		// "blockHash":        reHash,
		// "transactionHash":  reHash,
		"transactionIndex": reNum,
	}

	v := NewLogRes(log)
	j, _ := json.Marshal(v)

	for k, re := range tests {
		match, _ := regexp.MatchString(fmt.Sprintf(`{.*"%s":%s.*}`, k, re), string(j))
		if !match {
			t.Error(fmt.Sprintf("`%s` output json does not match format %s. Got %s", k, re, j))
		}
	}

}

func TestNewLogsRes(t *testing.T) {
	logs := make([]state.Log, 3)
	logs[0] = makeStateLog(1)
	logs[1] = makeStateLog(2)
	logs[2] = makeStateLog(3)
	tests := map[string]string{}

	v := NewLogsRes(logs)
	j, _ := json.Marshal(v)

	for k, re := range tests {
		match, _ := regexp.MatchString(fmt.Sprintf(`[{.*"%s":%s.*}]`, k, re), string(j))
		if !match {
			t.Error(fmt.Sprintf("%s output json does not match format %s. Got %s", k, re, j))
		}
	}

}

func makeStateLog(num int) state.Log {
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
