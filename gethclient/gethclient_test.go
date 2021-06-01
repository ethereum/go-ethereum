package gethclient

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	testKey, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testAddr    = crypto.PubkeyToAddress(testKey.PublicKey)
	testBalance = big.NewInt(2e10)
)

func newTestBackend(t *testing.T) (*node.Node, []*types.Block) {
	// Generate test chain.
	genesis, blocks := generateTestChain()
	// Create node
	n, err := node.New(&node.Config{})
	if err != nil {
		t.Fatalf("can't create new node: %v", err)
	}
	// Create Ethereum Service
	config := &ethconfig.Config{Genesis: genesis}
	config.Ethash.PowMode = ethash.ModeFake
	ethservice, err := eth.New(n, config)
	if err != nil {
		t.Fatalf("can't create new ethereum service: %v", err)
	}
	// Import the test chain.
	if err := n.Start(); err != nil {
		t.Fatalf("can't start test node: %v", err)
	}
	if _, err := ethservice.BlockChain().InsertChain(blocks[1:]); err != nil {
		t.Fatalf("can't import test blocks: %v", err)
	}
	return n, blocks
}

func generateTestChain() (*core.Genesis, []*types.Block) {
	db := rawdb.NewMemoryDatabase()
	config := params.AllEthashProtocolChanges
	genesis := &core.Genesis{
		Config:    config,
		Alloc:     core.GenesisAlloc{testAddr: {Balance: testBalance}},
		ExtraData: []byte("test genesis"),
		Timestamp: 9000,
	}
	generate := func(i int, g *core.BlockGen) {
		g.OffsetTime(5)
		g.SetExtra([]byte("test"))
	}
	gblock := genesis.ToBlock(db)
	engine := ethash.NewFaker()
	blocks, _ := core.GenerateChain(config, gblock, engine, db, 1, generate)
	blocks = append([]*types.Block{gblock}, blocks...)
	return genesis, blocks
}

func TestEthClient(t *testing.T) {
	backend, _ := newTestBackend(t)
	client, _ := backend.Attach()
	defer backend.Close()
	defer client.Close()

	tests := map[string]struct {
		test func(t *testing.T)
	}{"TestAccessList": {
		func(t *testing.T) { testAccessList(t, client) },
	},
	}
	t.Parallel()
	for name, tt := range tests {
		t.Run(name, tt.test)
	}
}

func testAccessList(t *testing.T, client *rpc.Client) {
	ec := NewClient(client)
	// Test transfer
	msg := ethereum.CallMsg{
		From:     testAddr,
		To:       &common.Address{},
		Gas:      21000,
		GasPrice: big.NewInt(1),
		Value:    big.NewInt(1),
	}
	al, gas, vmErr, err := ec.CreateAccessList(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vmErr != "" {
		t.Fatalf("unexpected vm error: %v", vmErr)
	}
	if gas != 21000 {
		t.Fatalf("unexpected gas used: %v", gas)
	}
	if len(*al) != 0 {
		t.Fatalf("unexpected length of accesslist: %v", len(*al))
	}
	// Test reverting transaction
	msg = ethereum.CallMsg{
		From:     testAddr,
		To:       nil,
		Gas:      100000,
		GasPrice: big.NewInt(1),
		Value:    big.NewInt(1),
		Data:     common.FromHex("0x608060806080608155fd"),
	}
	al, gas, vmErr, err = ec.CreateAccessList(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vmErr == "" {
		t.Fatalf("wanted vmErr, got none")
	}
	if gas == 21000 {
		t.Fatalf("unexpected gas used: %v", gas)
	}
	if len(*al) != 1 || al.StorageKeys() != 1 {
		t.Fatalf("unexpected length of accesslist: %v", len(*al))
	}
	// address changes between calls, so we can't test for it.
	if (*al)[0].Address == common.HexToAddress("0x0") {
		t.Fatalf("unexpected address: %v", (*al)[0].Address)
	}
	if (*al)[0].StorageKeys[0] != common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000081") {
		t.Fatalf("unexpected storage key: %v", (*al)[0].StorageKeys[0])
	}
}
