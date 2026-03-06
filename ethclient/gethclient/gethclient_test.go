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
	"os"
	"path/filepath"
	"strings"
	"testing"

	ethereum "github.com/XinFinOrg/XDPoSChain"
	"github.com/XinFinOrg/XDPoSChain/XDCx"
	"github.com/XinFinOrg/XDPoSChain/XDCxlending"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/ethash"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/eth"
	"github.com/XinFinOrg/XDPoSChain/eth/ethconfig"
	"github.com/XinFinOrg/XDPoSChain/eth/filters"
	"github.com/XinFinOrg/XDPoSChain/eth/tracers"
	_ "github.com/XinFinOrg/XDPoSChain/eth/tracers/native"
	"github.com/XinFinOrg/XDPoSChain/ethclient"
	"github.com/XinFinOrg/XDPoSChain/node"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rpc"
)

var (
	testKey, _   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testAddr     = crypto.PubkeyToAddress(testKey.PublicKey)
	testContract = common.HexToAddress("0xbeef")
	testEmpty    = common.HexToAddress("0xeeee")
	testSlot     = common.HexToHash("0xdeadbeef")
	testValue    = crypto.Keccak256Hash(testSlot[:])
	testBalance  = big.NewInt(2e15)
)

func newTestBackend(t *testing.T) (*node.Node, []*types.Block, []common.Hash) {
	// Generate test chain.
	genesis, blocks, txHashes := generateTestChain()
	// Create node
	n, err := node.New(&node.Config{
		DataDir:     t.TempDir(),
		HTTPModules: []string{"debug", "eth", "admin"},
	})
	if err != nil {
		t.Fatalf("can't create new node: %v", err)
	}
	xdcxDir := filepath.Join(t.TempDir(), "xdcx")
	if err := os.MkdirAll(xdcxDir, 0o755); err != nil {
		t.Fatalf("can't create XDCx data dir: %v", err)
	}
	xdcx := XDCx.New(n, &XDCx.Config{DataDir: xdcxDir})
	lending := XDCxlending.New(n, xdcx)
	// Create Ethereum Service
	config := &ethconfig.Config{Genesis: genesis, RPCGasCap: 1000000}
	ethservice, err := eth.New(n, config, xdcx, lending)
	if err != nil {
		t.Fatalf("can't create new ethereum service: %v", err)
	}
	n.RegisterAPIs(tracers.APIs(ethservice.APIBackend))

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
	return n, blocks, txHashes
}

func generateTestChain() (*core.Genesis, []*types.Block, []common.Hash) {
	chainConfig := *params.AllEthashProtocolChanges
	chainConfig.Eip1559Block = big.NewInt(0)
	// Ensure global chain constants match the test chain config before block generation.
	common.CopyConstants(chainConfig.ChainID.Uint64())
	genesis := &core.Genesis{
		Config: &chainConfig,
		Alloc: types.GenesisAlloc{
			testAddr:     {Balance: testBalance, Storage: map[common.Hash]common.Hash{testSlot: testValue}},
			testContract: {Nonce: 1, Code: []byte{0x13, 0x37}},
			testEmpty:    {Balance: big.NewInt(1)},
		},
		ExtraData: []byte("test genesis"),
		Timestamp: 9000,
	}
	txHashes := make([]common.Hash, 0)
	generate := func(i int, g *core.BlockGen) {
		g.OffsetTime(5)
		g.SetExtra([]byte("test"))

		to := common.BytesToAddress([]byte{byte(i + 1)})
		tx := types.NewTx(&types.LegacyTx{
			Nonce:    uint64(i),
			To:       &to,
			Value:    big.NewInt(int64(2*i + 1)),
			Gas:      params.TxGas,
			GasPrice: big.NewInt(params.InitialBaseFee),
			Data:     nil,
		})
		tx, _ = types.SignTx(tx, types.LatestSignerForChainID(chainConfig.ChainID), testKey)
		g.AddTx(tx)
		txHashes = append(txHashes, tx.Hash())
	}
	_, blocks, _ := core.GenerateChainWithGenesis(genesis, ethash.NewFaker(), 1, generate)
	blocks = append([]*types.Block{genesis.ToBlock()}, blocks...)
	return genesis, blocks, txHashes
}

func TestGethClient(t *testing.T) {
	backend, _, txHashes := newTestBackend(t)
	client := backend.Attach()
	defer backend.Close()
	defer client.Close()

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
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
			"TestCallContract",
			func(t *testing.T) { testCallContract(t, client) },
		}, {
			"TestCallContractWithBlockOverrides",
			func(t *testing.T) { testCallContractWithBlockOverrides(t, client) },
		}, {
			"TestTraceTransactionWithCallTracer",
			func(t *testing.T) { testTraceTransactionWithCallTracer(t, client, txHashes) },
		}, {
			"TestTraceCallWithCallTracer",
			func(t *testing.T) { testTraceCallWithCallTracer(t, client) },
		},
		// The testaccesslist is a bit time-sensitive: the newTestBackend imports
		// one block. The `testAccessList` fails if the miner has not yet created a
		// new pending-block after the import event.
		// Hence: this test should be last, execute the tests serially.
		{
			"TestAccessList",
			func(t *testing.T) { testAccessList(t, client) },
		},
		{
			"TestTraceTransaction",
			func(t *testing.T) { testTraceTransactions(t, client, txHashes) },
		},
		{
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
	baseFee := new(big.Int).SetUint64(params.InitialBaseFee)

	for i, tc := range []struct {
		msg       ethereum.CallMsg
		wantGas   uint64
		wantErr   string
		wantVMErr string
		wantAL    string
	}{
		{ // Test transfer
			msg: ethereum.CallMsg{
				From:     testAddr,
				To:       &common.Address{},
				Gas:      21000,
				GasPrice: baseFee,
				Value:    big.NewInt(1),
			},
			wantGas: 21000,
			wantAL:  `[]`,
		},
		{ // Test reverting transaction
			msg: ethereum.CallMsg{
				From:     testAddr,
				To:       nil,
				Gas:      100000,
				GasPrice: baseFee,
				Value:    big.NewInt(1),
				Data:     common.FromHex("0x608060806080608155fd"),
			},
			wantGas:   78018,
			wantVMErr: "execution reverted",
			wantAL: `[
  {
    "address": "0xdb7d6ab1f17c6b31909ae466702703daef9269cf",
    "storageKeys": [
      "0x0000000000000000000000000000000000000000000000000000000000000081"
    ]
  }
]`,
		},
		{ // error when gasPrice is less than baseFee
			msg: ethereum.CallMsg{
				From:     testAddr,
				To:       &common.Address{},
				Gas:      21000,
				GasPrice: big.NewInt(1), // less than baseFee
				Value:    big.NewInt(1),
			},
			wantErr: "max fee per gas less than block base fee",
		},
		{ // when gasPrice is not specified
			msg: ethereum.CallMsg{
				From:  testAddr,
				To:    &common.Address{},
				Gas:   21000,
				Value: big.NewInt(1),
			},
			wantGas: 21000,
			wantAL:  `[]`,
		},
	} {
		al, gas, vmErr, err := ec.CreateAccessList(context.Background(), tc.msg)
		if tc.wantErr != "" {
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("test %d: wrong error: %v", i, err)
			}
			continue
		} else if err != nil {
			t.Fatalf("test %d: wrong error: %v", i, err)
		}
		if have, want := vmErr, tc.wantVMErr; have != want {
			t.Fatalf("test %d: vmErr wrong, have %v want %v", i, have, want)
		}
		if have, want := gas, tc.wantGas; have != want {
			t.Fatalf("test %d: gas wrong, have %v want %v", i, have, want)
		}
		haveList, _ := json.MarshalIndent(al, "", "  ")
		if have, want := string(haveList), tc.wantAL; have != want {
			t.Fatalf("test %d: access list wrong, have:\n%v\nwant:\n%v", i, have, want)
		}
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
	baseFee := new(big.Int).SetUint64(params.InitialBaseFee)

	// Subscribe to Transactions
	ch1 := make(chan common.Hash)
	ec.SubscribePendingTransactions(context.Background(), ch1)

	// Subscribe to Transactions
	ch2 := make(chan *types.Transaction)
	ec.SubscribeFullPendingTransactions(context.Background(), ch2)

	// Send a transaction
	chainID, err := ethcl.ChainID(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	nonce, err := ethcl.NonceAt(context.Background(), testAddr, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Create transaction
	tx := types.NewTransaction(nonce, common.Address{1}, big.NewInt(1), 22000, baseFee, nil)
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
	hash := <-ch1
	if hash != signedTx.Hash() {
		t.Fatalf("Invalid tx hash received, got %v, want %v", hash, signedTx.Hash())
	}
	// Check that the transaction was sent over the channel
	tx = <-ch2
	if tx.Hash() != signedTx.Hash() {
		t.Fatalf("Invalid tx hash received, got %v, want %v", tx.Hash(), signedTx.Hash())
	}
}

func testCallContract(t *testing.T, client *rpc.Client) {
	ec := New(client)
	baseFee := new(big.Int).SetUint64(params.InitialBaseFee)
	msg := ethereum.CallMsg{
		From:     testAddr,
		To:       &common.Address{},
		Gas:      21000,
		GasPrice: baseFee,
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

func testTraceTransactions(t *testing.T, client *rpc.Client, txHashes []common.Hash) {
	ec := New(client)
	for _, txHash := range txHashes {
		// Struct logger
		_, err := ec.TraceTransaction(context.Background(), txHash, nil)
		if err != nil {
			t.Fatal(err)
		}

		// Struct logger
		_, err = ec.TraceTransaction(context.Background(), txHash,
			&tracers.TraceConfig{},
		)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestOverrideAccountMarshal(t *testing.T) {
	om := map[common.Address]OverrideAccount{
		{0x11}: {
			// Zero-valued nonce is not overridden, but simply dropped by the encoder.
			Nonce: 0,
		},
		{0xaa}: {
			Nonce: 5,
		},
		{0xbb}: {
			Code: []byte{1},
		},
		{0xcc}: {
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

func TestBlockOverridesMarshal(t *testing.T) {
	for i, tt := range []struct {
		bo   BlockOverrides
		want string
	}{
		{
			bo:   BlockOverrides{},
			want: `{}`,
		},
		{
			bo: BlockOverrides{
				Coinbase: common.HexToAddress("0x1111111111111111111111111111111111111111"),
			},
			want: `{"feeRecipient":"0x1111111111111111111111111111111111111111"}`,
		},
		{
			bo: BlockOverrides{
				Number:     big.NewInt(1),
				Difficulty: big.NewInt(2),
				Time:       3,
				GasLimit:   4,
				BaseFee:    big.NewInt(5),
			},
			want: `{"number":"0x1","difficulty":"0x2","time":"0x3","gasLimit":"0x4","baseFeePerGas":"0x5"}`,
		},
	} {
		marshalled, err := json.Marshal(&tt.bo)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(marshalled) != tt.want {
			t.Errorf("Testcase #%d failed. expected\n%s\ngot\n%s", i, tt.want, string(marshalled))
		}
	}
}

func testCallContractWithBlockOverrides(t *testing.T, client *rpc.Client) {
	ec := New(client)
	baseFee := new(big.Int).SetUint64(params.InitialBaseFee)
	msg := ethereum.CallMsg{
		From:     testAddr,
		To:       &common.Address{},
		Gas:      50000,
		GasPrice: baseFee,
		Value:    big.NewInt(1),
	}
	override := OverrideAccount{
		// Returns coinbase address.
		Code: common.FromHex("0x41806000526014600cf3"),
	}
	mapAcc := make(map[common.Address]OverrideAccount)
	mapAcc[common.Address{}] = override
	res, err := ec.CallContract(context.Background(), msg, big.NewInt(0), &mapAcc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(res, common.FromHex("0x0000000000000000000000000000000000000000")) {
		t.Fatalf("unexpected result: %x", res)
	}

	// Now test with block overrides
	bo := BlockOverrides{
		Coinbase: common.HexToAddress("0x1111111111111111111111111111111111111111"),
	}
	res, err = ec.CallContractWithBlockOverrides(context.Background(), msg, big.NewInt(0), &mapAcc, bo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(res, common.FromHex("0x1111111111111111111111111111111111111111")) {
		t.Fatalf("unexpected result: %x", res)
	}
}

func testTraceTransactionWithCallTracer(t *testing.T, client *rpc.Client, txHashes []common.Hash) {
	ec := New(client)
	for _, txHash := range txHashes {
		// With nil config (defaults).
		result, err := ec.TraceTransactionWithCallTracer(context.Background(), txHash, nil)
		if err != nil {
			t.Fatalf("nil config: %v", err)
		}
		if result.Type != "CALL" {
			t.Fatalf("unexpected type: %s", result.Type)
		}
		if result.From == (common.Address{}) {
			t.Fatal("from is zero")
		}
		if result.Gas == 0 {
			t.Fatal("gas is zero")
		}

		// With explicit config.
		result, err = ec.TraceTransactionWithCallTracer(context.Background(), txHash,
			&CallTracerConfig{},
		)
		if err != nil {
			t.Fatalf("explicit config: %v", err)
		}
		if result.Type != "CALL" {
			t.Fatalf("unexpected type: %s", result.Type)
		}
	}
}

func testTraceCallWithCallTracer(t *testing.T, client *rpc.Client) {
	ec := New(client)
	baseFee := new(big.Int).SetUint64(params.InitialBaseFee)
	msg := ethereum.CallMsg{
		From:     testAddr,
		To:       &common.Address{},
		Gas:      21000,
		GasPrice: baseFee,
		Value:    big.NewInt(1),
	}
	result, err := ec.TraceCallWithCallTracer(context.Background(), msg,
		rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber), nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != "CALL" {
		t.Fatalf("unexpected type: %s", result.Type)
	}
	if result.From == (common.Address{}) {
		t.Fatal("from is zero")
	}
	if result.Gas == 0 {
		t.Fatal("gas is zero")
	}
}
