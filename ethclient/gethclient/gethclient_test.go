// Copyright 2021 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package gethclient

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	testKey, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testAddr    = crypto.PubkeyToAddress(testKey.PublicKey)
	testSlot    = common.HexToHash("0xdeadbeef")
	testValue   = crypto.Keccak256Hash(testSlot[:])
	testBalance = big.NewInt(2e15)
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
	filterSystem := filters.NewFilterSystem(ethservice.APIBackend, filters.Config{})
	n.RegisterAPIs([]rpc.API{{
		Namespace: "eth",
		Service:   filters.NewFilterAPI(filterSystem, false),
	}})

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
	genesis := &core.Genesis{
		Config:    params.AllEthashProtocolChanges,
		Alloc:     core.GenesisAlloc{testAddr: {Balance: testBalance, Storage: map[common.Hash]common.Hash{testSlot: testValue}}},
		ExtraData: []byte("test genesis"),
		Timestamp: 9000,
	}
	generate := func(i int, g *core.BlockGen) {
		g.OffsetTime(5)
		g.SetExtra([]byte("test"))
	}
	_, blocks, _ := core.GenerateChainWithGenesis(genesis, ethash.NewFaker(), 1, generate)
	blocks = append([]*types.Block{genesis.ToBlock()}, blocks...)
	return genesis, blocks
}

func TestGethClient(t *testing.T) {
	backend, _ := newTestBackend(t)
	client, err := backend.Attach()
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	defer client.Close()

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			"TestGetProof",
			func(t *testing.T) { testGetProof(t, client) },
		}, {
			"TestGCStats",
			func(t *testing.T) { testGCStats(t, client) },
		}, {
			"TestMemStats",
			func(t *testing.T) { testMemStats(t, client) },
		}, {
			"TestGetNodeInfo",
			func(t *testing.T) { testGetNodeInfo(t, client) },
		}, {
			"TestSubscribePendingTxHashes",
			func(t *testing.T) { testSubscribePendingTransactions(t, client) },
		}, {
			"TestSubscribePendingTxs",
			func(t *testing.T) { testSubscribeFullPendingTransactions(t, client) },
		}, {
			"TestCallContract",
			func(t *testing.T) { testCallContract(t, client) },
		},
		// The testaccesslist is a bit time-sensitive: the newTestBackend imports
		// one block. The `testAcessList` fails if the miner has not yet created a
		// new pending-block after the import event.
		// Hence: this test should be last, execute the tests serially.
		{
			"TestAccessList",
			func(t *testing.T) { testAccessList(t, client) },
		}, {
			"TestSetHead",
			func(t *testing.T) { testSetHead(t, client) },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func testAccessList(t *testing.T, client *rpc.Client) {
	ec := New(client)
	// Test transfer
	msg := ethereum.CallMsg{
		From:     testAddr,
		To:       &common.Address{},
		Gas:      21000,
		GasPrice: big.NewInt(765625000),
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
		GasPrice: big.NewInt(1000000000),
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

func testGetProof(t *testing.T, client *rpc.Client) {
	ec := New(client)
	ethcl := ethclient.NewClient(client)
	result, err := ec.GetProof(context.Background(), testAddr, []string{testSlot.String()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result.Address[:], testAddr[:]) {
		t.Fatalf("unexpected address, want: %v got: %v", testAddr, result.Address)
	}
	// test nonce
	nonce, _ := ethcl.NonceAt(context.Background(), result.Address, nil)
	if result.Nonce != nonce {
		t.Fatalf("invalid nonce, want: %v got: %v", nonce, result.Nonce)
	}
	// test balance
	balance, _ := ethcl.BalanceAt(context.Background(), result.Address, nil)
	if result.Balance.Cmp(balance) != 0 {
		t.Fatalf("invalid balance, want: %v got: %v", balance, result.Balance)
	}
	// test storage
	if len(result.StorageProof) != 1 {
		t.Fatalf("invalid storage proof, want 1 proof, got %v proof(s)", len(result.StorageProof))
	}
	proof := result.StorageProof[0]
	slotValue, _ := ethcl.StorageAt(context.Background(), testAddr, testSlot, nil)
	if !bytes.Equal(slotValue, proof.Value.Bytes()) {
		t.Fatalf("invalid storage proof value, want: %v, got: %v", slotValue, proof.Value.Bytes())
	}
	if proof.Key != testSlot.String() {
		t.Fatalf("invalid storage proof key, want: %v, got: %v", testSlot.String(), proof.Key)
	}
}

func testGCStats(t *testing.T, client *rpc.Client) {
	ec := New(client)
	_, err := ec.GCStats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func testMemStats(t *testing.T, client *rpc.Client) {
	ec := New(client)
	stats, err := ec.MemStats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stats.Alloc == 0 {
		t.Fatal("Invalid mem stats retrieved")
	}
}

func testGetNodeInfo(t *testing.T, client *rpc.Client) {
	ec := New(client)
	info, err := ec.GetNodeInfo(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if info.Name == "" {
		t.Fatal("Invalid node info retrieved")
	}
}

func testSetHead(t *testing.T, client *rpc.Client) {
	ec := New(client)
	err := ec.SetHead(context.Background(), big.NewInt(0))
	if err != nil {
		t.Fatal(err)
	}
}

func testSubscribePendingTransactions(t *testing.T, client *rpc.Client) {
	ec := New(client)
	ethcl := ethclient.NewClient(client)
	// Subscribe to Transactions
	ch := make(chan common.Hash)
	ec.SubscribePendingTransactions(context.Background(), ch)
	// Send a transaction
	chainID, err := ethcl.ChainID(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Create transaction
	tx := types.NewTransaction(0, common.Address{1}, big.NewInt(1), 22000, big.NewInt(1), nil)
	signer := types.LatestSignerForChainID(chainID)
	signature, err := crypto.Sign(signer.Hash(tx).Bytes(), testKey)
	if err != nil {
		t.Fatal(err)
	}
	signedTx, err := tx.WithSignature(signer, signature)
	if err != nil {
		t.Fatal(err)
	}
	// Send transaction
	err = ethcl.SendTransaction(context.Background(), signedTx)
	if err != nil {
		t.Fatal(err)
	}
	// Check that the transaction was sent over the channel
	hash := <-ch
	if hash != signedTx.Hash() {
		t.Fatalf("Invalid tx hash received, got %v, want %v", hash, signedTx.Hash())
	}
}

func testSubscribeFullPendingTransactions(t *testing.T, client *rpc.Client) {
	ec := New(client)
	ethcl := ethclient.NewClient(client)
	// Subscribe to Transactions
	ch := make(chan *types.Transaction)
	ec.SubscribeFullPendingTransactions(context.Background(), ch)
	// Send a transaction
	chainID, err := ethcl.ChainID(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Create transaction
	tx := types.NewTransaction(1, common.Address{1}, big.NewInt(1), 22000, big.NewInt(1), nil)
	signer := types.LatestSignerForChainID(chainID)
	signature, err := crypto.Sign(signer.Hash(tx).Bytes(), testKey)
	if err != nil {
		t.Fatal(err)
	}
	signedTx, err := tx.WithSignature(signer, signature)
	if err != nil {
		t.Fatal(err)
	}
	// Send transaction
	err = ethcl.SendTransaction(context.Background(), signedTx)
	if err != nil {
		t.Fatal(err)
	}
	// Check that the transaction was sent over the channel
	tx = <-ch
	if tx.Hash() != signedTx.Hash() {
		t.Fatalf("Invalid tx hash received, got %v, want %v", tx.Hash(), signedTx.Hash())
	}
}

func testCallContract(t *testing.T, client *rpc.Client) {
	ec := New(client)
	msg := ethereum.CallMsg{
		From:     testAddr,
		To:       &common.Address{},
		Gas:      21000,
		GasPrice: big.NewInt(1000000000),
		Value:    big.NewInt(1),
	}
	// CallContract without override
	if _, err := ec.CallContract(context.Background(), msg, big.NewInt(0), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// CallContract with override
	override := OverrideAccount{
		Nonce: 1,
	}
	mapAcc := make(map[common.Address]OverrideAccount)
	mapAcc[testAddr] = override
	if _, err := ec.CallContract(context.Background(), msg, big.NewInt(0), &mapAcc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOverrideAccountMarshal(t *testing.T) {
	om := map[common.Address]OverrideAccount{
		common.Address{0x11}: OverrideAccount{
			// Zero-valued nonce is not overriddden, but simply dropped by the encoder.
			Nonce: 0,
		},
		common.Address{0xaa}: OverrideAccount{
			Nonce: 5,
		},
		common.Address{0xbb}: OverrideAccount{
			Code: []byte{1},
		},
		common.Address{0xcc}: OverrideAccount{
			// 'code', 'balance', 'state' should be set when input is
			// a non-nil but empty value.
			Code:    []byte{},
			Balance: big.NewInt(0),
			State:   map[common.Hash]common.Hash{},
			// For 'stateDiff' the behavior is different, empty map
			// is ignored because it makes no difference.
			StateDiff: map[common.Hash]common.Hash{},
		},
	}

	marshalled, err := json.MarshalIndent(&om, "", "  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `{
  "0x1100000000000000000000000000000000000000": {},
  "0xaa00000000000000000000000000000000000000": {
    "nonce": "0x5"
  },
  "0xbb00000000000000000000000000000000000000": {
    "code": "0x01"
  },
  "0xcc00000000000000000000000000000000000000": {
    "code": "0x",
    "balance": "0x0",
    "state": {}
  }
}`

	if string(marshalled) != expected {
		t.Error("wrong output:", string(marshalled))
		t.Error("want:", expected)
	}
}
