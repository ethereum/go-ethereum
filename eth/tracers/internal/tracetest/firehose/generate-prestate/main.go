package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/exp/maps"
)

var configByName = map[string]*params.ChainConfig{
	"mainnet": params.MainnetChainConfig,
	"goerli":  params.GoerliChainConfig,
	"sepolia": params.SepoliaChainConfig,
	"holesky": params.HoleskyChainConfig,
}

func main() {
	ensure(len(os.Args) == 3, "Usage: generate-prestate <network> <tx-hash>")

	config, found := configByName[os.Args[1]]
	ensure(found, "Unknown network %q, valid networks are %q", os.Args[1], strings.Join(maps.Keys(configByName), ", "))

	endpoint := os.Getenv("ARCHIVE_ENDPOINT")
	ensure(endpoint != "", "ARCHIVE_ENDPOINT environment variable is not set")

	txHash := common.HexToHash(os.Args[2])
	ensure(txHash.Big().Sign() != 0, "Argument %q is not a valid transaction hash", os.Args[1])

	client, err := rpc.DialOptions(context.Background(), endpoint)
	noError(err, "Failed to connect to RPC server")

	log("Fetching transaction %q...", txHash.Hex())
	tx, blockHash, err := transactionByHash(context.Background(), client, txHash)
	noError(err, "Failed to get transaction %q by hash", txHash)

	log("Fetching block %q...", txHash.Hex())
	block, err := blockByHash(context.Background(), client, blockHash)
	noError(err, "Failed to get block %q by hash", blockHash)

	log("Fetching parent block %q...", txHash.Hex())
	parentBlock, err := blockByHash(context.Background(), client, block.ParentHash())
	noError(err, "Failed to get parent block %q by hash", block.ParentHash())

	log("Collecting transaction prestate trace...")
	var prestateTrace types.GenesisAlloc
	err = client.Call(&prestateTrace, "debug_traceTransaction", txHash, map[string]interface{}{
		"tracer": "prestateTracer",
	})
	noError(err, "Failed to trace transaction %q with prestateTracer", txHash)

	prestateTest := &PrestateTest{
		// Our genesis block is the parent block
		Genesis: parentBlock,
		Block:   block,
		Alloc:   prestateTrace,
		Config:  config,
		Tx:      tx,
	}

	output, err := json.MarshalIndent(prestateTest, "", "  ")
	noError(err, "Failed to marshal prestate test to JSON")

	fmt.Println(string(output))
}

type PrestateTest struct {
	Genesis *rpcBlock
	Block   *rpcBlock
	Alloc   types.GenesisAlloc
	Config  *params.ChainConfig
	Tx      *types.Transaction
}

func (b *PrestateTest) MarshalJSON() ([]byte, error) {
	genesis := b.Genesis.Fields

	for k, v := range genesis {
		if v == nil {
			delete(genesis, k)
		}
	}

	delete(genesis, "gasUsed")
	delete(genesis, "logsBloom")
	delete(genesis, "parentHash")
	delete(genesis, "receiptsRoot")
	delete(genesis, "sha3Uncles")
	delete(genesis, "size")
	delete(genesis, "transactions")
	delete(genesis, "transactionsRoot")
	delete(genesis, "uncles")

	genesis["gasLimit"] = hexToIntegerString(genesis["gasLimit"].(string))
	genesis["number"] = hexToIntegerString(genesis["number"].(string))
	genesis["timestamp"] = hexToIntegerString(genesis["timestamp"].(string))
	genesis["difficulty"] = hexToIntegerString(genesis["difficulty"].(string))
	genesis["totalDifficulty"] = hexToIntegerString(genesis["totalDifficulty"].(string))

	if found := genesis["baseFeePerGas"]; found != nil {
		genesis["baseFeePerGas"] = hexToIntegerString(genesis["baseFeePerGas"].(string))
	}

	marshaledAlloc, err := json.Marshal(b.Alloc)
	if err != nil {
		return nil, err
	}

	var allocOut map[string]any
	if err = json.Unmarshal(marshaledAlloc, &allocOut); err != nil {
		return nil, err
	}

	for _, account := range allocOut {
		details := account.(map[string]any)
		for k, v := range details {
			if k == "nonce" {
				details["nonce"] = hexToIntegerString(v.(string))
			}
		}
	}

	genesis["alloc"] = allocOut
	genesis["config"] = b.Config

	context := map[string]any{
		"number":     hexToIntegerString(b.Block.Fields["number"].(string)),
		"difficulty": hexToIntegerString(b.Block.Fields["difficulty"].(string)),
		"timestamp":  hexToIntegerString(b.Block.Fields["timestamp"].(string)),
		"gasLimit":   hexToIntegerString(b.Block.Fields["gasLimit"].(string)),
		"miner":      b.Block.Fields["miner"],
	}

	if found := b.Block.Fields["baseFeePerGas"]; found != nil {
		context["baseFeePerGas"] = hexToIntegerString(b.Block.Fields["baseFeePerGas"].(string))
	}

	txBuffer := bytes.NewBuffer(nil)
	rlp.Encode(txBuffer, b.Tx)

	out := map[string]any{}
	out["genesis"] = genesis
	out["context"] = context
	out["input"] = "0x" + hex.EncodeToString(txBuffer.Bytes())

	return json.Marshal(out)
}

func hexToIntegerString(in string) string {
	value := math.HexOrDecimal256{}
	noError(value.UnmarshalText([]byte(in)), "Failed to parse hex value %q", in)

	return (*big.Int)(&value).String()
}

type CallTrace struct {
	From    string `json:"from"`
	Gas     string `json:"gas"`
	GasUsed string `json:"gasUsed"`
	To      string `json:"to"`
	Input   string `json:"input"`
	Value   string `json:"value"`
	Type    string `json:"type"`
	Output  string `json:"output,omitempty"`

	Calls []CallTrace `json:"calls"`
}

func blockByHash(ctx context.Context, ec *rpc.Client, hash common.Hash) (*rpcBlock, error) {
	var out rpcBlock
	err := ec.CallContext(ctx, &out.Fields, "eth_getBlockByHash", hash, false)
	if err != nil {
		return nil, err
	} else if out.Fields == nil {
		return nil, ethereum.NotFound
	}

	return &out, nil
}

type rpcBlock struct {
	Fields map[string]any
}

func (b *rpcBlock) ParentHash() common.Hash {
	return common.HexToHash(b.Fields["parentHash"].(string))
}

func transactionByHash(ctx context.Context, ec *rpc.Client, hash common.Hash) (tx *types.Transaction, blockHash common.Hash, err error) {
	var json *rpcTransaction
	err = ec.CallContext(ctx, &json, "eth_getTransactionByHash", hash)
	if err != nil {
		return nil, common.Hash{}, err
	} else if json == nil {
		return nil, common.Hash{}, ethereum.NotFound
	} else if _, r, _ := json.tx.RawSignatureValues(); r == nil {
		return nil, common.Hash{}, errors.New("server returned transaction without signature")
	}

	if json.BlockHash == nil {
		return nil, common.Hash{}, errors.New("server returned transaction without block hash")
	}

	return json.tx, *json.BlockHash, nil
}

type rpcTransaction struct {
	tx *types.Transaction
	txExtraInfo
}

type txExtraInfo struct {
	BlockNumber *string         `json:"blockNumber,omitempty"`
	BlockHash   *common.Hash    `json:"blockHash,omitempty"`
	From        *common.Address `json:"from,omitempty"`
}

func (tx *rpcTransaction) UnmarshalJSON(msg []byte) error {
	if err := json.Unmarshal(msg, &tx.tx); err != nil {
		return err
	}
	return json.Unmarshal(msg, &tx.txExtraInfo)
}

func log(message string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, message+"\n", args...)
}

func ensure(condition bool, message string, args ...interface{}) {
	if !condition {
		quit(message, args...)
	}
}

func noError(err error, message string, args ...interface{}) {
	if err != nil {
		quit(message+": "+err.Error(), args...)
	}
}

func quit(message string, args ...interface{}) {
	fmt.Printf(message+"\n", args...)
	os.Exit(1)
}
