// Copyright 2015 The go-ethereum Authors
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

package runtime

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

func TestDefaults(t *testing.T) {
	cfg := new(Config)
	setDefaults(cfg)

	if cfg.Difficulty == nil {
		t.Error("expected difficulty to be non nil")
	}

	if cfg.Time == nil {
		t.Error("expected time to be non nil")
	}
	if cfg.GasLimit == 0 {
		t.Error("didn't expect gaslimit to be zero")
	}
	if cfg.GasPrice == nil {
		t.Error("expected time to be non nil")
	}
	if cfg.Value == nil {
		t.Error("expected time to be non nil")
	}
	if cfg.GetHashFn == nil {
		t.Error("expected time to be non nil")
	}
	if cfg.BlockNumber == nil {
		t.Error("expected block number to be non nil")
	}
}

func TestEVM(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("crashed with: %v", r)
		}
	}()

	Execute([]byte{
		byte(vm.DIFFICULTY),
		byte(vm.TIMESTAMP),
		byte(vm.GASLIMIT),
		byte(vm.PUSH1),
		byte(vm.ORIGIN),
		byte(vm.BLOCKHASH),
		byte(vm.COINBASE),
	}, nil, nil)
}

func TestExecute(t *testing.T) {
	ret, _, err := Execute([]byte{
		byte(vm.PUSH1), 10,
		byte(vm.PUSH1), 0,
		byte(vm.MSTORE),
		byte(vm.PUSH1), 32,
		byte(vm.PUSH1), 0,
		byte(vm.RETURN),
	}, nil, nil)
	if err != nil {
		t.Fatal("didn't expect error", err)
	}

	num := new(big.Int).SetBytes(ret)
	if num.Cmp(big.NewInt(10)) != 0 {
		t.Error("Expected 10, got", num)
	}
}

func TestCall(t *testing.T) {
	state, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	address := common.HexToAddress("0x0a")
	state.SetCode(address, []byte{
		byte(vm.PUSH1), 10,
		byte(vm.PUSH1), 0,
		byte(vm.MSTORE),
		byte(vm.PUSH1), 32,
		byte(vm.PUSH1), 0,
		byte(vm.RETURN),
	})

	ret, _, err := Call(address, nil, &Config{State: state})
	if err != nil {
		t.Fatal("didn't expect error", err)
	}

	num := new(big.Int).SetBytes(ret)
	if num.Cmp(big.NewInt(10)) != 0 {
		t.Error("Expected 10, got", num)
	}
}

func BenchmarkCall(b *testing.B) {
	var definition = `[{"constant":true,"inputs":[],"name":"seller","outputs":[{"name":"","type":"address"}],"type":"function"},{"constant":false,"inputs":[],"name":"abort","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"value","outputs":[{"name":"","type":"uint256"}],"type":"function"},{"constant":false,"inputs":[],"name":"refund","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"buyer","outputs":[{"name":"","type":"address"}],"type":"function"},{"constant":false,"inputs":[],"name":"confirmReceived","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"state","outputs":[{"name":"","type":"uint8"}],"type":"function"},{"constant":false,"inputs":[],"name":"confirmPurchase","outputs":[],"type":"function"},{"inputs":[],"type":"constructor"},{"anonymous":false,"inputs":[],"name":"Aborted","type":"event"},{"anonymous":false,"inputs":[],"name":"PurchaseConfirmed","type":"event"},{"anonymous":false,"inputs":[],"name":"ItemReceived","type":"event"},{"anonymous":false,"inputs":[],"name":"Refunded","type":"event"}]`

	var code = common.Hex2Bytes("6060604052361561006c5760e060020a600035046308551a53811461007457806335a063b4146100865780633fa4f245146100a6578063590e1ae3146100af5780637150d8ae146100cf57806373fac6f0146100e1578063c19d93fb146100fe578063d696069714610112575b610131610002565b610133600154600160a060020a031681565b610131600154600160a060020a0390811633919091161461015057610002565b61014660005481565b610131600154600160a060020a039081163391909116146102d557610002565b610133600254600160a060020a031681565b610131600254600160a060020a0333811691161461023757610002565b61014660025460ff60a060020a9091041681565b61013160025460009060ff60a060020a9091041681146101cc57610002565b005b600160a060020a03166060908152602090f35b6060908152602090f35b60025460009060a060020a900460ff16811461016b57610002565b600154600160a060020a03908116908290301631606082818181858883f150506002805460a060020a60ff02191660a160020a179055506040517f72c874aeff0b183a56e2b79c71b46e1aed4dee5e09862134b8821ba2fddbf8bf9250a150565b80546002023414806101dd57610002565b6002805460a060020a60ff021973ffffffffffffffffffffffffffffffffffffffff1990911633171660a060020a1790557fd5d55c8a68912e9a110618df8d5e2e83b8d83211c57a8ddd1203df92885dc881826060a15050565b60025460019060a060020a900460ff16811461025257610002565b60025460008054600160a060020a0390921691606082818181858883f150508354604051600160a060020a0391821694503090911631915082818181858883f150506002805460a060020a60ff02191660a160020a179055506040517fe89152acd703c9d8c7d28829d443260b411454d45394e7995815140c8cbcbcf79250a150565b60025460019060a060020a900460ff1681146102f057610002565b6002805460008054600160a060020a0390921692909102606082818181858883f150508354604051600160a060020a0391821694503090911631915082818181858883f150506002805460a060020a60ff02191660a160020a179055506040517f8616bbbbad963e4e65b1366f1d75dfb63f9e9704bbbf91fb01bec70849906cf79250a15056")

	abi, err := abi.JSON(strings.NewReader(definition))
	if err != nil {
		b.Fatal(err)
	}

	cpurchase, err := abi.Pack("confirmPurchase")
	if err != nil {
		b.Fatal(err)
	}
	creceived, err := abi.Pack("confirmReceived")
	if err != nil {
		b.Fatal(err)
	}
	refund, err := abi.Pack("refund")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 400; j++ {
			Execute(code, cpurchase, nil)
			Execute(code, creceived, nil)
			Execute(code, refund, nil)
		}
	}
}
func benchmarkEVM_Create(bench *testing.B, code string) {
	var (
		statedb, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
		sender     = common.BytesToAddress([]byte("sender"))
		receiver   = common.BytesToAddress([]byte("receiver"))
	)

	statedb.CreateAccount(sender)
	statedb.SetCode(receiver, common.FromHex(code))
	runtimeConfig := Config{
		Origin:      sender,
		State:       statedb,
		GasLimit:    10000000,
		Difficulty:  big.NewInt(0x200000),
		Time:        new(big.Int).SetUint64(0),
		Coinbase:    common.Address{},
		BlockNumber: new(big.Int).SetUint64(1),
		ChainConfig: &params.ChainConfig{
			ChainID:             big.NewInt(1),
			HomesteadBlock:      new(big.Int),
			ByzantiumBlock:      new(big.Int),
			ConstantinopleBlock: new(big.Int),
			DAOForkBlock:        new(big.Int),
			DAOForkSupport:      false,
			EIP150Block:         new(big.Int),
			EIP155Block:         new(big.Int),
			EIP158Block:         new(big.Int),
		},
		EVMConfig: vm.Config{},
	}
	// Warm up the intpools and stuff
	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		Call(receiver, []byte{}, &runtimeConfig)
	}
	bench.StopTimer()
}

func BenchmarkEVM_CREATE_500(bench *testing.B) {
	// initcode size 500K, repeatedly calls CREATE and then modifies the mem contents
	benchmarkEVM_Create(bench, "5b6207a120600080f0600152600056")
}
func BenchmarkEVM_CREATE2_500(bench *testing.B) {
	// initcode size 500K, repeatedly calls CREATE2 and then modifies the mem contents
	benchmarkEVM_Create(bench, "5b586207a120600080f5600152600056")
}
func BenchmarkEVM_CREATE_1200(bench *testing.B) {
	// initcode size 1200K, repeatedly calls CREATE and then modifies the mem contents
	benchmarkEVM_Create(bench, "5b62124f80600080f0600152600056")
}
func BenchmarkEVM_CREATE2_1200(bench *testing.B) {
	// initcode size 1200K, repeatedly calls CREATE2 and then modifies the mem contents
	benchmarkEVM_Create(bench, "5b5862124f80600080f5600152600056")
}

func fakeHeader(n uint64, parentHash common.Hash) *types.Header {
	header := types.Header{
		Coinbase:   common.HexToAddress("0x00000000000000000000000000000000deadbeef"),
		Number:     big.NewInt(int64(n)),
		ParentHash: parentHash,
		Time:       1000,
		Nonce:      types.BlockNonce{0x1},
		Extra:      []byte{},
		Difficulty: big.NewInt(0),
		GasLimit:   100000,
	}
	return &header
}

type dummyChain struct {
	counter int
}

// Engine retrieves the chain's consensus engine.
func (d *dummyChain) Engine() consensus.Engine {
	return nil
}

// GetHeader returns the hash corresponding to their hash.
func (d *dummyChain) GetHeader(h common.Hash, n uint64) *types.Header {
	d.counter++
	parentHash := common.Hash{}
	s := common.LeftPadBytes(big.NewInt(int64(n-1)).Bytes(), 32)
	copy(parentHash[:], s)

	//parentHash := common.Hash{byte(n - 1)}
	//fmt.Printf("GetHeader(%x, %d) => header with parent %x\n", h, n, parentHash)
	return fakeHeader(n, parentHash)
}

// TestBlockhash tests the blockhash operation. It's a bit special, since it internally
// requires access to a chain reader.
func TestBlockhash(t *testing.T) {
	// Current head
	n := uint64(1000)
	parentHash := common.Hash{}
	s := common.LeftPadBytes(big.NewInt(int64(n-1)).Bytes(), 32)
	copy(parentHash[:], s)
	header := fakeHeader(n, parentHash)

	// This is the contract we're using. It requests the blockhash for current num (should be all zeroes),
	// then iteratively fetches all blockhashes back to n-260.
	// It returns
	// 1. the first (should be zero)
	// 2. the second (should be the parent hash)
	// 3. the last non-zero hash
	// By making the chain reader return hashes which correlate to the number, we can
	// verify that it obtained the right hashes where it should

	/*

		pragma solidity ^0.5.3;
		contract Hasher{

			function test() public view returns (bytes32, bytes32, bytes32){
				uint256 x = block.number;
				bytes32 first;
				bytes32 last;
				bytes32 zero;
				zero = blockhash(x); // Should be zeroes
				first = blockhash(x-1);
				for(uint256 i = 2 ; i < 260; i++){
					bytes32 hash = blockhash(x - i);
					if (uint256(hash) != 0){
						last = hash;
					}
				}
				return (zero, first, last);
			}
		}

	*/
	// The contract above
	data := common.Hex2Bytes("6080604052348015600f57600080fd5b50600436106045576000357c010000000000000000000000000000000000000000000000000000000090048063f8a8fd6d14604a575b600080fd5b60506074565b60405180848152602001838152602001828152602001935050505060405180910390f35b600080600080439050600080600083409050600184034092506000600290505b61010481101560c35760008186034090506000816001900414151560b6578093505b5080806001019150506094565b508083839650965096505050505090919256fea165627a7a72305820462d71b510c1725ff35946c20b415b0d50b468ea157c8c77dff9466c9cb85f560029")
	// The method call to 'test()'
	input := common.Hex2Bytes("f8a8fd6d")
	chain := &dummyChain{}
	ret, _, err := Execute(data, input, &Config{
		GetHashFn:   core.GetHashFn(header, chain),
		BlockNumber: new(big.Int).Set(header.Number),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(ret) != 96 {
		t.Fatalf("expected returndata to be 96 bytes, got %d", len(ret))
	}

	zero := new(big.Int).SetBytes(ret[0:32])
	first := new(big.Int).SetBytes(ret[32:64])
	last := new(big.Int).SetBytes(ret[64:96])
	if zero.BitLen() != 0 {
		t.Fatalf("expected zeroes, got %x", ret[0:32])
	}
	if first.Uint64() != 999 {
		t.Fatalf("second block should be 999, got %d (%x)", first, ret[32:64])
	}
	if last.Uint64() != 744 {
		t.Fatalf("last block should be 744, got %d (%x)", last, ret[64:96])
	}
	if exp, got := 255, chain.counter; exp != got {
		t.Errorf("suboptimal; too much chain iteration, expected %d, got %d", exp, got)
	}
}

// BenchmarkSimpleLoop test a pretty simple loop which loops
// 1M (1 048 575) times.
// Takes about 200 ms
func BenchmarkSimpleLoop(b *testing.B) {
	// 0xfffff = 1048575 loops
	code := []byte{
		byte(vm.PUSH3), 0x0f, 0xff, 0xff,
		byte(vm.JUMPDEST), //  [ count ]
		byte(vm.PUSH1), 1, // [count, 1]
		byte(vm.SWAP1),    // [1, count]
		byte(vm.SUB),      // [ count -1 ]
		byte(vm.DUP1),     //  [ count -1 , count-1]
		byte(vm.PUSH1), 4, // [count-1, count -1, label]
		byte(vm.JUMPI), // [ 0 ]
		byte(vm.STOP),
	}
	//tracer := vm.NewJSONLogger(nil, os.Stdout)
	//Execute(code, nil, &Config{
	//	EVMConfig: vm.Config{
	//		Debug:  true,
	//		Tracer: tracer,
	//	}})

	for i := 0; i < b.N; i++ {
		Execute(code, nil, nil)
	}
}
