package engineclient

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
)

var (
	testKey, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testAddr    = crypto.PubkeyToAddress(testKey.PublicKey)
	testBalance = big.NewInt(2e15)
)

var genesis = &core.Genesis{
	Config:    params.AllEthashProtocolChanges,
	Alloc:     types.GenesisAlloc{testAddr: {Balance: testBalance}},
	ExtraData: []byte("test genesis"),
	Timestamp: 9000,
	BaseFee:   big.NewInt(params.InitialBaseFee),
}

var testTx1 = types.MustSignNewTx(testKey, types.LatestSigner(genesis.Config), &types.LegacyTx{
	Nonce:    0,
	Value:    big.NewInt(12),
	GasPrice: big.NewInt(params.InitialBaseFee),
	Gas:      params.TxGas,
	To:       &common.Address{2},
})

var testTx2 = types.MustSignNewTx(testKey, types.LatestSigner(genesis.Config), &types.LegacyTx{
	Nonce:    1,
	Value:    big.NewInt(8),
	GasPrice: big.NewInt(params.InitialBaseFee),
	Gas:      params.TxGas,
	To:       &common.Address{2},
})

func newTestBackend(t *testing.T) (*node.Node, []*types.Block) {
	// Generate test chain.
	blocks := generateTestChain()

	// Create node
	n, err := node.New(&node.Config{})
	if err != nil {
		t.Fatalf("can't create new node: %v", err)
	}
	// Create Ethereum Service
	config := &ethconfig.Config{Genesis: genesis}
	ethservice, err := eth.New(n, config)
	if err != nil {
		t.Fatalf("can't create new ethereum service: %v", err)
	}

	// Register the engine api namespace.
	catalyst.Register(n, ethservice)

	// Import the test chain.
	if err := n.Start(); err != nil {
		t.Fatalf("can't start test node: %v", err)
	}
	if _, err := ethservice.BlockChain().InsertChain(blocks[1:]); err != nil {
		t.Fatalf("can't import test blocks: %v", err)
	}
	// Ensure the tx indexing is fully generated
	for ; ; time.Sleep(time.Millisecond * 100) {
		progress, err := ethservice.BlockChain().TxIndexProgress()
		if err == nil && progress.Done() {
			break
		}
	}
	return n, blocks
}

func generateTestChain() []*types.Block {
	generate := func(i int, g *core.BlockGen) {
		g.OffsetTime(5)
		g.SetExtra([]byte("test"))
		if i == 1 {
			// Test transactions are included in block #2.
			g.AddTx(testTx1)
			g.AddTx(testTx2)
		}
	}
	_, blocks, _ := core.GenerateChainWithGenesis(genesis, ethash.NewFaker(), 2, generate)
	return append([]*types.Block{genesis.ToBlock()}, blocks...)
}

func TestEngineClient(t *testing.T) {
	backend, chain := newTestBackend(t)
	client := New(ethclient.NewClient(backend.Attach()))
	defer backend.Close()
	defer client.Close()

	tests := map[string]struct {
		test func(t *testing.T)
	}{
		"ExchangeCapabilities": {
			func(t *testing.T) { testExchangeCapabilities(t, chain, client) },
		},
		"GetClientVersionV1": {
			func(t *testing.T) { testGetClientV1(t, chain, client) },
		},
		"GetPayloadBodiesByHashV1": {
			func(t *testing.T) { testGetPayloadBodiesByHashV1(t, chain, client) },
		},
		"GetPayloadBodiesByRangeV1": {
			func(t *testing.T) { testGetPayloadBodiesByRangeV1(t, chain, client) },
		},
		"NewPayloadV1": {
			func(t *testing.T) { testNewPayloadV1(t, chain, client) },
		},
		"NewPayloadV2": {
			func(t *testing.T) { testNewPayloadV2(t, chain, client) },
		},
		"NewPayloadV3": {
			func(t *testing.T) { testNewPayloadV3(t, chain, client) },
		},
		"ForkchoiceUpdatedV1": {
			func(t *testing.T) { testForkchoiceUpdatedV1(t, chain, client) },
		},
		"ForkchoiceUpdatedV2": {
			func(t *testing.T) { testForkchoiceUpdatedV2(t, chain, client) },
		},
		"ForkchoiceUpdatedV3": {
			func(t *testing.T) { testForkchoiceUpdatedV3(t, chain, client) },
		},
		"GetPayloadV3": {
			func(t *testing.T) { testGetPayloadV3(t, chain, client) },
		},
		"GetPayloadV2": {
			func(t *testing.T) { testGetPayloadV2(t, chain, client) },
		},
		"GetPayloadV1": {
			func(t *testing.T) { testGetPayloadV1(t, chain, client) },
		},
	}

	t.Parallel()
	for name, tt := range tests {
		t.Run(name, tt.test)
	}
}
func testExchangeCapabilities(t *testing.T, chain []*types.Block, client *Client) {
	expected := catalyst.Caps
	capabilities := []string{"random", "ignored", "strings"}
	actual, err := client.ExchangeCapabilities(context.Background(), capabilities)
	if err != nil {
		t.Fatalf("ExchangeCapabilitiesV1 failed: %v", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Expected capabilities %v, got %v", expected, actual)
	}
}

func testGetClientV1(t *testing.T, chain []*types.Block, client *Client) {
	actual, err := client.GetClientVersionV1(context.Background())
	if err != nil {
		t.Fatalf("GetClientVersionV1 failed: %v", err)
	}
	if !strings.Contains(fmt.Sprint(actual), "go-ethereum") {
		t.Fatalf("Expected go-ethereum client version, got %v", actual)
	}
}

func testGetPayloadBodiesByHashV1(t *testing.T, chain []*types.Block, client *Client) {
	actual, err := client.GetPayloadBodiesByHashV1(context.Background(), []common.Hash{chain[2].Hash()})
	if err != nil {
		t.Fatalf("GetPayloadBodiesByHashV1 failed: %v", err)
	}
	if len(actual) != 1 {
		t.Fatalf("Expected 1 payload body, got %v", actual)
	}

	if actual[0].TransactionData == nil {
		t.Fatalf("Expected payload body, got %v", actual[0])
	}

	tx := &types.Transaction{}
	if err := tx.UnmarshalBinary(actual[0].TransactionData[0]); err != nil {
		t.Fatalf("Failed to unmarshal transaction: %v", err)
	}
	if tx.Hash() != testTx1.Hash() {
		t.Fatalf("Expected transaction %v, got %v", testTx1, tx)
	}
}

func testGetPayloadBodiesByRangeV1(t *testing.T, chain []*types.Block, client *Client) {
	actual, err := client.GetPayloadBodiesByRangeV1(context.Background(), hexutil.Uint64(chain[2].NumberU64()), hexutil.Uint64(1))
	if err != nil {
		t.Fatalf("GetPayloadBodiesByRangeV1 failed: %v", err)
	}
	if len(actual) != 1 {
		t.Fatalf("Expected 1 payload body, got %v", len(actual))
	}

	if actual[0].TransactionData == nil {
		t.Fatalf("Expected payload body, got %v", actual[0])
	}

	tx := &types.Transaction{}
	fmt.Println(actual[0].TransactionData)
	tx.UnmarshalBinary(actual[0].TransactionData[0])
	if tx.Hash() != testTx1.Hash() {
		t.Fatalf("Expected transaction %v, got %v", testTx1, tx)
	}
}

func testNewPayloadV1(t *testing.T, chain []*types.Block, client *Client) {
	ctx := context.Background()

	// Create a mock payload
	payload := createMockPayload(chain[len(chain)-1])

	// Call NewPayloadV1
	status, err := client.NewPayloadV1(ctx, payload)
	if err != nil {
		t.Fatalf("NewPayloadV1 failed: %v", err)
	}
	if status.Status != engine.INVALID {
		t.Fatalf("Expected payload status to be INVALID, got %v", status.Status)
	}
}

func testNewPayloadV2(t *testing.T, chain []*types.Block, client *Client) {
	ctx := context.Background()

	// Create a mock payload
	payload := createMockPayload(chain[len(chain)-1])

	// Call NewPayloadV1
	status, err := client.NewPayloadV1(ctx, payload)
	if err != nil {
		t.Fatalf("NewPayloadV1 failed: %v", err)
	}
	if status.Status != engine.INVALID {
		t.Fatalf("Expected payload status to be INVALID, got %v", status.Status)
	}
}

func testNewPayloadV3(t *testing.T, chain []*types.Block, client *Client) {
	ctx := context.Background()

	// Create a mock payload
	payload := createMockPayload(chain[len(chain)-1])

	// Call NewPayloadV1
	status, err := client.NewPayloadV1(ctx, payload)
	if err != nil {
		t.Fatalf("NewPayloadV1 failed: %v", err)
	}
	if status.Status != engine.INVALID {
		t.Fatalf("Expected payload status to be INVALID, got %v", status.Status)
	}
}

func createMockPayload(parent *types.Block) *engine.ExecutionPayloadEnvelope {
	// Assuming createMockPayload creates and returns a mock ExecutionPayloadEnvelope
	// This is a placeholder for actual payload creation code
	return &engine.ExecutionPayloadEnvelope{
		ExecutionPayload: &engine.ExecutableData{
			ParentHash:    parent.Hash(),
			BlockHash:     common.BytesToHash(crypto.Keccak256([]byte("randomBlockHash"))),
			FeeRecipient:  common.BytesToAddress(crypto.Keccak256([]byte("randomFeeRecipient"))),
			StateRoot:     common.BytesToHash(crypto.Keccak256([]byte("randomStateRoot"))),
			ReceiptsRoot:  common.BytesToHash(crypto.Keccak256([]byte("randomReceiptsRoot"))),
			LogsBloom:     crypto.Keccak256([]byte("randomLogsBloom")),
			Random:        common.BytesToHash(crypto.Keccak256([]byte("random"))),
			Number:        parent.NumberU64() + 1,
			GasLimit:      21000,
			GasUsed:       10500,
			Timestamp:     1630425600,
			ExtraData:     []byte("randomExtraData"),
			BaseFeePerGas: big.NewInt(1000000000),
			Transactions: [][]byte{
				crypto.Keccak256([]byte("randomTransaction1")),
				crypto.Keccak256([]byte("randomTransaction2"))},
		}}
}

func testForkchoiceUpdatedV1(t *testing.T, chain []*types.Block, client *Client) {
	// Call ForkchoiceUpdatedV2
	resp, err := client.ForkchoiceUpdatedV1(context.Background(), &engine.ForkchoiceStateV1{
		HeadBlockHash: common.Hash{},
	}, nil)
	if err != nil {
		t.Fatalf("ForkchoiceUpdatedV2 failed: %v", err)
	}
	if resp.Status == nil {
		t.Fatalf("Expected status, got %v", resp.Status)
	}

	if resp.Status.Status != engine.INVALID {
		t.Fatalf("Expected status to be INVALID, got %v", resp.Status.Status)
	}
}

func testForkchoiceUpdatedV2(t *testing.T, chain []*types.Block, client *Client) {
	// Call ForkchoiceUpdatedV2
	resp, err := client.ForkchoiceUpdatedV2(context.Background(), &engine.ForkchoiceStateV1{
		HeadBlockHash: common.Hash{},
	}, nil)
	if err != nil {
		t.Fatalf("ForkchoiceUpdatedV2 failed: %v", err)
	}
	if resp.Status == nil {
		t.Fatalf("Expected status, got %v", resp.Status)
	}

	if resp.Status.Status != engine.INVALID {
		t.Fatalf("Expected status to be INVALID, got %v", resp.Status.Status)
	}
}

func testForkchoiceUpdatedV3(t *testing.T, chain []*types.Block, client *Client) {
	// Call ForkchoiceUpdatedV3
	resp, err := client.ForkchoiceUpdatedV3(context.Background(), &engine.ForkchoiceStateV1{
		HeadBlockHash: common.Hash{},
	}, nil)
	if err != nil {
		t.Fatalf("ForkchoiceUpdatedV3 failed: %v", err)
	}
	if resp.Status == nil {
		t.Fatalf("Expected status, got %v", resp.Status)
	}

	if resp.Status.Status != engine.INVALID {
		t.Fatalf("Expected status to be INVALID, got %v", resp.Status.Status)
	}
}

func testGetPayloadV3(t *testing.T, chain []*types.Block, client *Client) {
	payloadID := engine.PayloadID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08} // Example PayloadID, adjust as necessary
	_, err := client.GetPayloadV3(context.Background(), &payloadID)
	if err.Error() != "Unsupported fork" {
		t.Fatalf("GetPayloadV3 failed: %v", err)
	}
}

func testGetPayloadV2(t *testing.T, chain []*types.Block, client *Client) {
	payloadID := engine.PayloadID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08} // Example PayloadID, adjust as necessary
	_, err := client.GetPayloadV2(context.Background(), &payloadID)
	if err.Error() != "Unknown payload" {
		t.Fatalf("Expected unknown payload error, got: %v", err)
	}
}

func testGetPayloadV1(t *testing.T, chain []*types.Block, client *Client) {
	payloadID := engine.PayloadID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08} // Example PayloadID, adjust as necessary
	_, err := client.GetPayloadV1(context.Background(), &payloadID)
	if err.Error() != "Unknown payload" {
		t.Fatalf("Expected unknown payload error, got: %v", err)
	}
}
