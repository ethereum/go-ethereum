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
	reNumNonZero = `"0x([1-9a-f][0-9a-f]{0,15})"`        // non-zero required must not have left-padded zeros
	reNumOpt     = `"0x([1-9a-f][0-9a-f]{0,15})|0"|null` // must not have left-padded zeros or null
	reData       = `"0x[0-9a-f]*"`                       // can be "empty"
	// reListHash   = `[("\w":"0x[0-9a-f]{64}",?)*]`
	// reListObj    = `[("\w":(".+"|null),?)*]`
)

func TestNewBlockRes(t *testing.T) {
	tests := map[string]string{
		"number":           reNum,
		"hash":             reHash,
		"parentHash":       reHash,
		"nonce":            reData,
		"sha3Uncles":       reHash,
		"logsBloom":        reData,
		"transactionsRoot": reHash,
		"stateRoot":        reHash,
		"miner":            reAddress,
		"difficulty":       `"0x1"`,
		"totalDifficulty":  reNum,
		"size":             reNumNonZero,
		"extraData":        reData,
		"gasLimit":         reNum,
		// "minGasPrice":  "0x",
		"gasUsed":   reNum,
		"timestamp": reNum,
		// "transactions": reListHash,
		// "uncles":       reListHash,
	}

	block := makeBlock()
	v := NewBlockRes(block, false)
	j, _ := json.Marshal(v)

	for k, re := range tests {
		match, _ := regexp.MatchString(fmt.Sprintf(`{.*"%s":%s.*}`, k, re), string(j))
		if !match {
			t.Error(fmt.Sprintf("%s output json does not match format %s. Got %s", k, re, j))
		}
	}
}

func TestNewBlockResTxFull(t *testing.T) {
	tests := map[string]string{
		"number":           reNum,
		"hash":             reHash,
		"parentHash":       reHash,
		"nonce":            reData,
		"sha3Uncles":       reHash,
		"logsBloom":        reData,
		"transactionsRoot": reHash,
		"stateRoot":        reHash,
		"miner":            reAddress,
		"difficulty":       `"0x1"`,
		"totalDifficulty":  reNum,
		"size":             reNumNonZero,
		"extraData":        reData,
		"gasLimit":         reNum,
		// "minGasPrice":  "0x",
		"gasUsed":   reNum,
		"timestamp": reNum,
		// "transactions": reListHash,
		// "uncles":       reListHash,
	}

	block := makeBlock()
	v := NewBlockRes(block, true)
	j, _ := json.Marshal(v)

	for k, re := range tests {
		match, _ := regexp.MatchString(fmt.Sprintf(`{.*"%s":%s.*}`, k, re), string(j))
		if !match {
			t.Error(fmt.Sprintf("%s output json does not match format %s. Got %s", k, re, j))
		}
	}
}

func TestBlockNil(t *testing.T) {
	var block *types.Block
	block = nil
	u := NewBlockRes(block, false)
	j, _ := json.Marshal(u)
	if string(j) != "null" {
		t.Errorf("Expected null but got %v", string(j))
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

func TestTransactionNil(t *testing.T) {
	var tx *types.Transaction
	tx = nil
	u := NewTransactionRes(tx)
	j, _ := json.Marshal(u)
	if string(j) != "null" {
		t.Errorf("Expected null but got %v", string(j))
	}
}

func TestNewUncleRes(t *testing.T) {
	header := makeHeader()
	u := NewUncleRes(header)
	tests := map[string]string{
		"number":           reNum,
		"hash":             reHash,
		"parentHash":       reHash,
		"nonce":            reData,
		"sha3Uncles":       reHash,
		"receiptHash":      reHash,
		"transactionsRoot": reHash,
		"stateRoot":        reHash,
		"miner":            reAddress,
		"difficulty":       reNum,
		"extraData":        reData,
		"gasLimit":         reNum,
		"gasUsed":          reNum,
		"timestamp":        reNum,
	}

	j, _ := json.Marshal(u)
	for k, re := range tests {
		match, _ := regexp.MatchString(fmt.Sprintf(`{.*"%s":%s.*}`, k, re), string(j))
		if !match {
			t.Error(fmt.Sprintf("`%s` output json does not match format %s. Source %s", k, re, j))
		}
	}
}

func TestUncleNil(t *testing.T) {
	var header *types.Header
	header = nil
	u := NewUncleRes(header)
	j, _ := json.Marshal(u)
	if string(j) != "null" {
		t.Errorf("Expected null but got %v", string(j))
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
	logs := make([]*state.Log, 3)
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

func makeStateLog(num int) *state.Log {
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

func makeHeader() *types.Header {
	header := &types.Header{
		ParentHash:  common.StringToHash("0x00"),
		UncleHash:   common.StringToHash("0x00"),
		Coinbase:    common.StringToAddress("0x00"),
		Root:        common.StringToHash("0x00"),
		TxHash:      common.StringToHash("0x00"),
		ReceiptHash: common.StringToHash("0x00"),
		// Bloom:
		Difficulty: big.NewInt(88888888),
		Number:     big.NewInt(16),
		GasLimit:   big.NewInt(70000),
		GasUsed:    big.NewInt(25000),
		Time:       124356789,
		Extra:      nil,
		MixDigest:  common.StringToHash("0x00"),
		Nonce:      [8]byte{0, 1, 2, 3, 4, 5, 6, 7},
	}
	return header
}

func makeBlock() *types.Block {
	parentHash := common.HexToHash("0x01")
	coinbase := common.HexToAddress("0x01")
	root := common.HexToHash("0x01")
	difficulty := common.Big1
	nonce := uint64(1)
	block := types.NewBlock(parentHash, coinbase, root, difficulty, nonce, nil)

	txto := common.HexToAddress("0x02")
	txamount := big.NewInt(1)
	txgasAmount := big.NewInt(1)
	txgasPrice := big.NewInt(1)
	txdata := []byte{1, 2, 3}

	tx := types.NewTransactionMessage(txto, txamount, txgasAmount, txgasPrice, txdata)
	txs := make([]*types.Transaction, 1)
	txs[0] = tx
	block.SetTransactions(txs)

	uncles := make([]*types.Header, 1)
	uncles[0] = makeHeader()
	block.SetUncles(uncles)

	return block
}
