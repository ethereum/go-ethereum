// Copyright 2017 The go-ethereum Authors
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

package tracers

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/json"
	"math/big"
	"reflect"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/common/hexutil"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/tests"
)

// To generate a new callTracer test, copy paste the makeTest method below into
// a Geth console and call it with a transaction hash you which to export.

/*
// makeTest generates a callTracer test by running a prestate reassembled and a
// call trace run, assembling all the gathered information into a test case.
var makeTest = function(tx, rewind) {
  // Generate the genesis block from the block, transaction and prestate data
  var block   = eth.getBlock(eth.getTransaction(tx).blockHash);
  var genesis = eth.getBlock(block.parentHash);

  delete genesis.gasUsed;
  delete genesis.logsBloom;
  delete genesis.parentHash;
  delete genesis.receiptsRoot;
  delete genesis.sha3Uncles;
  delete genesis.size;
  delete genesis.transactions;
  delete genesis.transactionsRoot;
  delete genesis.uncles;

  genesis.gasLimit  = genesis.gasLimit.toString();
  genesis.number    = genesis.number.toString();
  genesis.timestamp = genesis.timestamp.toString();

  genesis.alloc = debug.traceTransaction(tx, {tracer: "prestateTracer", rewind: rewind});
  for (var key in genesis.alloc) {
    genesis.alloc[key].nonce = genesis.alloc[key].nonce.toString();
  }
  genesis.config = admin.nodeInfo.protocols.eth.config;

  // Generate the call trace and produce the test input
  var result = debug.traceTransaction(tx, {tracer: "callTracer", rewind: rewind});
  delete result.time;

  console.log(JSON.stringify({
    genesis: genesis,
    context: {
      number:     block.number.toString(),
      difficulty: block.difficulty,
      timestamp:  block.timestamp.toString(),
      gasLimit:   block.gasLimit.toString(),
      miner:      block.miner,
    },
    input:  eth.getRawTransaction(tx),
    result: result,
  }, null, 2));
}
*/

// callTrace is the result of a callTracer run.
type callTrace struct {
	Type    string          `json:"type"`
	From    common.Address  `json:"from"`
	To      common.Address  `json:"to"`
	Input   hexutil.Bytes   `json:"input"`
	Output  hexutil.Bytes   `json:"output"`
	Gas     *hexutil.Uint64 `json:"gas,omitempty"`
	GasUsed *hexutil.Uint64 `json:"gasUsed,omitempty"`
	Value   *hexutil.Big    `json:"value,omitempty"`
	Error   string          `json:"error,omitempty"`
	Calls   []callTrace     `json:"calls,omitempty"`
}

// TestZeroValueToNotExitCall tests the calltracer(s) on the following:
// Tx to A, A calls B with zero value. B does not already exist.
// Expected: that enter/exit is invoked and the inner call is shown in the result
func TestZeroValueToNotExitCall(t *testing.T) {
	var to = common.HexToAddress("0x00000000000000000000000000000000deadbeef")
	privkey, err := crypto.HexToECDSA("0000000000000000deadbeef00000000000000000000000000000000deadbeef")
	if err != nil {
		t.Fatalf("err %v", err)
	}
	signer := types.NewEIP155Signer(big.NewInt(1))
	tx, err := types.SignNewTx(privkey, signer, &types.LegacyTx{
		GasPrice: big.NewInt(0),
		Gas:      50000,
		To:       &to,
	})
	if err != nil {
		t.Fatalf("err %v", err)
	}
	origin, _ := signer.Sender(tx)
	context := vm.Context{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    common.Address{},
		BlockNumber: new(big.Int).SetUint64(8000000),
		Time:        new(big.Int).SetUint64(5),
		Difficulty:  big.NewInt(0x30000),
		GasLimit:    uint64(6000000),
		Origin:      origin,
		GasPrice:    big.NewInt(1),
	}
	var code = []byte{
		byte(vm.PUSH1), 0x0, byte(vm.DUP1), byte(vm.DUP1), byte(vm.DUP1), // in and outs zero
		byte(vm.DUP1), byte(vm.PUSH1), 0xff, byte(vm.GAS), // value=0,address=0xff, gas=GAS
		byte(vm.CALL),
	}
	var alloc = core.GenesisAlloc{
		to: core.GenesisAccount{
			Nonce: 1,
			Code:  code,
		},
		origin: core.GenesisAccount{
			Nonce:   0,
			Balance: big.NewInt(500000000000000),
		},
	}
	statedb := tests.MakePreState(rawdb.NewMemoryDatabase(), alloc)
	// Create the tracer, the EVM environment and run it
	tracer, err := New("callTracer", new(Context))
	if err != nil {
		t.Fatalf("failed to create call tracer: %v", err)
	}
	evm := vm.NewEVM(context, statedb, nil, params.MainnetChainConfig, vm.Config{Debug: true, Tracer: tracer})
	msg, err := tx.AsMessage(signer, nil, nil)
	if err != nil {
		t.Fatalf("failed to prepare transaction for tracing: %v", err)
	}
	st := core.NewStateTransition(evm, msg, new(core.GasPool).AddGas(tx.Gas()))
	if _, _, _, err, _ = st.TransitionDb(common.Address{}); err != nil {
		t.Fatalf("failed to execute transaction: %v", err)
	}
	// Retrieve the trace result and compare against the etalon
	res, err := tracer.GetResult()
	if err != nil {
		t.Fatalf("failed to retrieve trace result: %v", err)
	}
	have := new(callTrace)
	if err := json.Unmarshal(res, have); err != nil {
		t.Fatalf("failed to unmarshal trace result: %v", err)
	}
	wantStr := `{"type":"CALL","from":"0x682a80a6f560eec50d54e63cbeda1c324c5f8d1b","to":"0x00000000000000000000000000000000deadbeef","value":"0x0","gas":"0x7148","gasUsed":"0x2d0","input":"0x","output":"0x","calls":[{"type":"CALL","from":"0x00000000000000000000000000000000deadbeef","to":"0x00000000000000000000000000000000000000ff","value":"0x0","gas":"0x6cbf","gasUsed":"0x0","input":"0x","output":"0x"}]}`
	want := new(callTrace)
	json.Unmarshal([]byte(wantStr), want)
	if !jsonEqual(have, want) {
		t.Error("have != want")
	}
}

func TestPrestateTracerCreate2(t *testing.T) {
	common.TIPXDCXCancellationFee = big.NewInt(10000000000000)
	unsignedTx := types.NewTransaction(1, common.HexToAddress("0x00000000000000000000000000000000deadbeef"),
		new(big.Int), 5000000, big.NewInt(1), []byte{})

	privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		t.Fatalf("err %v", err)
	}
	signer := types.NewEIP155Signer(big.NewInt(1))
	tx, err := types.SignTx(unsignedTx, signer, privateKeyECDSA)
	if err != nil {
		t.Fatalf("err %v", err)
	}
	/**
		This comes from one of the test-vectors on the Skinny Create2 - EIP

	    address 0x00000000000000000000000000000000deadbeef
	    salt 0x00000000000000000000000000000000000000000000000000000000cafebabe
	    init_code 0xdeadbeef
	    gas (assuming no mem expansion): 32006
	    result: 0x60f3f640a8508fC6a86d45DF051962668E1e8AC7
	*/
	origin, _ := signer.Sender(tx)
	context := vm.Context{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Origin:      origin,
		Coinbase:    common.Address{},
		BlockNumber: new(big.Int).SetUint64(8000000),
		Time:        new(big.Int).SetUint64(5),
		Difficulty:  big.NewInt(0x30000),
		GasLimit:    uint64(6000000),
		GasPrice:    big.NewInt(1),
	}
	alloc := core.GenesisAlloc{}

	// The code pushes 'deadbeef' into memory, then the other params, and calls CREATE2, then returns
	// the address
	alloc[common.HexToAddress("0x00000000000000000000000000000000deadbeef")] = core.GenesisAccount{
		Nonce:   1,
		Code:    hexutil.MustDecode("0x63deadbeef60005263cafebabe6004601c6000F560005260206000F3"),
		Balance: big.NewInt(1),
	}
	alloc[origin] = core.GenesisAccount{
		Nonce:   1,
		Code:    []byte{},
		Balance: big.NewInt(500000000000000),
	}
	db := rawdb.NewMemoryDatabase()
	statedb := tests.MakePreState(db, alloc)

	// Create the tracer, the EVM environment and run it
	tracer, err := New("prestateTracer", new(Context))
	if err != nil {
		t.Fatalf("failed to create call tracer: %v", err)
	}
	evm := vm.NewEVM(context, statedb, nil, params.MainnetChainConfig, vm.Config{Debug: true, Tracer: tracer})

	msg, err := tx.AsMessage(signer, nil, nil)
	if err != nil {
		t.Fatalf("failed to prepare transaction for tracing: %v", err)
	}
	st := core.NewStateTransition(evm, msg, new(core.GasPool).AddGas(tx.Gas()))
	if _, _, _, err, _ = st.TransitionDb(common.Address{}); err != nil {
		t.Fatalf("failed to execute transaction: %v", err)
	}
	// Retrieve the trace result and compare against the etalon
	res, err := tracer.GetResult()
	if err != nil {
		t.Fatalf("failed to retrieve trace result: %v", err)
	}
	ret := make(map[string]interface{})
	if err := json.Unmarshal(res, &ret); err != nil {
		t.Fatalf("failed to unmarshal trace result: %v", err)
	}
	if _, has := ret["0x60f3f640a8508fc6a86d45df051962668e1e8ac7"]; !has {
		t.Fatalf("Expected 0x60f3f640a8508fc6a86d45df051962668e1e8ac7 in result")
	}
}

// jsonEqual is similar to reflect.DeepEqual, but does a 'bounce' via json prior to
// comparison
func jsonEqual(x, y interface{}) bool {
	xTrace := new(callTrace)
	yTrace := new(callTrace)
	if xj, err := json.Marshal(x); err == nil {
		json.Unmarshal(xj, xTrace)
	} else {
		return false
	}
	if yj, err := json.Marshal(y); err == nil {
		json.Unmarshal(yj, yTrace)
	} else {
		return false
	}
	return reflect.DeepEqual(xTrace, yTrace)
}

func BenchmarkTransactionTrace(b *testing.B) {
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	from := crypto.PubkeyToAddress(key.PublicKey)
	gas := uint64(1000000) // 1M gas
	to := common.HexToAddress("0x00000000000000000000000000000000deadbeef")
	signer := types.LatestSignerForChainID(big.NewInt(1337))
	tx, err := types.SignNewTx(key, signer,
		&types.LegacyTx{
			Nonce:    1,
			GasPrice: big.NewInt(500),
			Gas:      gas,
			To:       &to,
		})
	if err != nil {
		b.Fatal(err)
	}
	context := vm.Context{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    common.Address{},
		BlockNumber: new(big.Int).SetUint64(uint64(5)),
		Time:        new(big.Int).SetUint64(uint64(5)),
		Difficulty:  big.NewInt(0xffffffff),
		GasLimit:    gas,
		// BaseFee:     big.NewInt(8),
		Origin:   from,
		GasPrice: tx.GasPrice(),
	}
	alloc := core.GenesisAlloc{}
	// The code pushes 'deadbeef' into memory, then the other params, and calls CREATE2, then returns
	// the address
	loop := []byte{
		byte(vm.JUMPDEST), //  [ count ]
		byte(vm.PUSH1), 0, // jumpdestination
		byte(vm.JUMP),
	}
	alloc[common.HexToAddress("0x00000000000000000000000000000000deadbeef")] = core.GenesisAccount{
		Nonce:   1,
		Code:    loop,
		Balance: big.NewInt(1),
	}
	alloc[from] = core.GenesisAccount{
		Nonce:   1,
		Code:    []byte{},
		Balance: big.NewInt(500000000000000),
	}
	statedb := tests.MakePreState(rawdb.NewMemoryDatabase(), alloc)
	// Create the tracer, the EVM environment and run it
	tracer := vm.NewStructLogger(&vm.LogConfig{
		Debug: false,
		//DisableStorage: true,
		//EnableMemory: false,
		//EnableReturnData: false,
	})
	evm := vm.NewEVM(context, statedb, nil, params.AllEthashProtocolChanges, vm.Config{Debug: true, Tracer: tracer})
	msg, err := tx.AsMessage(signer, nil, nil)
	if err != nil {
		b.Fatalf("failed to prepare transaction for tracing: %v", err)
	}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		snap := statedb.Snapshot()
		st := core.NewStateTransition(evm, msg, new(core.GasPool).AddGas(tx.Gas()))
		_, _, _, err, _ = st.TransitionDb(common.Address{})
		if err != nil {
			b.Fatal(err)
		}
		statedb.RevertToSnapshot(snap)
		if have, want := len(tracer.StructLogs()), 244752; have != want {
			b.Fatalf("trace wrong, want %d steps, have %d", want, have)
		}
		tracer.Reset()
	}
}
