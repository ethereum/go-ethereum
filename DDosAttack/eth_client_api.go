package DDosAttack

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/asm"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"log"
	"math/big"
)

func GetBlockNumber() *big.Int {
	client, err := Connect("http://localhost:8545")
	if err != nil {
		log.Fatalf("Failed to connect with blockchain")
	}

	blockNumber, err := client.GetBlockNumber(context.TODO())
	return blockNumber
}

func TraceBlockByNumber(int64) *big.Int {
	client, err := Connect("http://localhost:8545")
	if err != nil {
		log.Fatalf("Failed to connect with blockchain")
	}

	blockNumber, err := client.GetBlockNumber(context.TODO())
	return blockNumber
}

type Client struct {
	rpcClient *rpc.Client
	EthClient *ethclient.Client
}

// GetBlockNumber returns the block number.
func (ec *Client) GetBlockNumber(ctx context.Context) (*big.Int, error) {
	var result hexutil.Big
	err := ec.rpcClient.CallContext(ctx, &result, "eth_blockNumber")
	return (*big.Int)(&result), err
}

// txTraceResult is the result of a single transaction trace.
type txTraceResult struct {
	Result interface{} `json:"result,omitempty"` // Trace results produced by the tracer
	Error  string      `json:"error,omitempty"`  // Trace failure produced by the tracer
}

func TraceBlock(number int64) error {
	var result []*txTraceResult

	client, err := Connect("http://localhost:8545")
	if err != nil {
		log.Fatalf("Failed to connect with blockchain")
	}

	err = client.rpcClient.CallContext(context.TODO(), &result, "debug_traceBlockByNumber",
		hexutil.EncodeUint64(317))
	fmt.Print(result)
	return err
}

func Connect(host string) (*Client, error) {
	rpcClient, err := rpc.Dial(host)
	if err != nil {
		return nil, err
	}
	ethClient := ethclient.NewClient(rpcClient)
	return &Client{rpcClient, ethClient}, nil
}

func PrintDisasm(code string) error {
	fmt.Printf("%v\n", code)
	return asm.PrintDisassembled(code)
}
