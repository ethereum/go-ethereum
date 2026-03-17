// Copyright 2019 The go-ethereum Authors
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

package simulated

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"

	ethereum "github.com/XinFinOrg/XDPoSChain"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/params"
)

var _ bind.ContractBackend = (Client)(nil)

var (
	testKey, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testAddr    = crypto.PubkeyToAddress(testKey.PublicKey)
	testKey2, _ = crypto.HexToECDSA("7ee346e3f7efc685250053bfbafbfc880d58dc6145247053d4fb3cb0f66dfcb2")
	testAddr2   = crypto.PubkeyToAddress(testKey2.PublicKey)
)

const callableAbi = "[{\"anonymous\":false,\"inputs\":[],\"name\":\"Called\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"Call\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

const callableBin = "6080604052348015600f57600080fd5b5060998061001e6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c806334e2292114602d575b600080fd5b60336035565b005b7f81fab7a4a0aa961db47eefc81f143a5220e8c8495260dd65b1356f1d19d3c7b860405160405180910390a156fea2646970667358221220029436d24f3ac598ceca41d4d712e13ced6d70727f4cdc580667de66d2f51d8b64736f6c63430008010033"

const abiJSON = `[ { "constant": false, "inputs": [ { "name": "memo", "type": "bytes" } ], "name": "receive", "outputs": [ { "name": "res", "type": "string" } ], "payable": true, "stateMutability": "payable", "type": "function" }, { "anonymous": false, "inputs": [ { "indexed": false, "name": "sender", "type": "address" }, { "indexed": false, "name": "amount", "type": "uint256" }, { "indexed": false, "name": "memo", "type": "bytes" } ], "name": "received", "type": "event" }, { "anonymous": false, "inputs": [ { "indexed": false, "name": "sender", "type": "address" } ], "name": "receivedAddr", "type": "event" } ]`

const abiBin = `0x608060405234801561001057600080fd5b506102a0806100206000396000f3fe60806040526004361061003b576000357c010000000000000000000000000000000000000000000000000000000090048063a69b6ed014610040575b600080fd5b6100b76004803603602081101561005657600080fd5b810190808035906020019064010000000081111561007357600080fd5b82018360208201111561008557600080fd5b803590602001918460018302840111640100000000831117156100a757600080fd5b9091929391929390505050610132565b6040518080602001828103825283818151815260200191508051906020019080838360005b838110156100f75780820151818401526020810190506100dc565b50505050905090810190601f1680156101245780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b60607f75fd880d39c1daf53b6547ab6cb59451fc6452d27caa90e5b6649dd8293b9eed33348585604051808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001848152602001806020018281038252848482818152602001925080828437600081840152601f19601f8201169050808301925050509550505050505060405180910390a17f46923992397eac56cf13058aced2a1871933622717e27b24eabc13bf9dd329c833604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390a16040805190810160405280600b81526020017f68656c6c6f20776f726c6400000000000000000000000000000000000000000081525090509291505056fea165627a7a72305820ff0c57dad254cfeda48c9cfb47f1353a558bccb4d1bc31da1dae69315772d29e0029`

const deployedCode = `60806040526004361061003b576000357c010000000000000000000000000000000000000000000000000000000090048063a69b6ed014610040575b600080fd5b6100b76004803603602081101561005657600080fd5b810190808035906020019064010000000081111561007357600080fd5b82018360208201111561008557600080fd5b803590602001918460018302840111640100000000831117156100a757600080fd5b9091929391929390505050610132565b6040518080602001828103825283818151815260200191508051906020019080838360005b838110156100f75780820151818401526020810190506100dc565b50505050905090810190601f1680156101245780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b60607f75fd880d39c1daf53b6547ab6cb59451fc6452d27caa90e5b6649dd8293b9eed33348585604051808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001848152602001806020018281038252848482818152602001925080828437600081840152601f19601f8201169050808301925050509550505050505060405180910390a17f46923992397eac56cf13058aced2a1871933622717e27b24eabc13bf9dd329c833604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390a16040805190810160405280600b81526020017f68656c6c6f20776f726c6400000000000000000000000000000000000000000081525090509291505056fea165627a7a72305820ff0c57dad254cfeda48c9cfb47f1353a558bccb4d1bc31da1dae69315772d29e0029`

var expectedReturn = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 11, 104, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

func simTestBackend(testAddr common.Address) *Backend {
	return New(
		types.GenesisAlloc{
			testAddr: {Balance: big.NewInt(10000000000000000)},
		}, 10000000,
	)
}

func newTx(sim *Backend, key *ecdsa.PrivateKey) (*types.Transaction, error) {
	client := sim.Client()

	// create a signed transaction to send
	head, _ := client.HeaderByNumber(context.Background(), nil) // Should be child's, good enough
	gasPrice := new(big.Int).Add(head.BaseFee, big.NewInt(1))
	addr := crypto.PubkeyToAddress(key.PublicKey)
	chainid, _ := client.ChainID(context.Background())
	nonce, err := client.PendingNonceAt(context.Background(), addr)
	if err != nil {
		return nil, err
	}
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainid,
		Nonce:     nonce,
		GasTipCap: big.NewInt(1),
		GasFeeCap: gasPrice,
		Gas:       21000,
		To:        &addr,
	})
	return types.SignTx(tx, types.LatestSignerForChainID(chainid), key)
}

func newContractCreationTx(sim *Backend, key *ecdsa.PrivateKey, bytecode []byte, gas uint64) (*types.Transaction, common.Address, error) {
	client := sim.Client()

	head, _ := client.HeaderByNumber(context.Background(), nil) // Should be child's, good enough
	gasFeeCap := new(big.Int).Add(head.BaseFee, big.NewInt(1))
	from := crypto.PubkeyToAddress(key.PublicKey)
	chainID, _ := client.ChainID(context.Background())
	nonce, err := client.PendingNonceAt(context.Background(), from)
	if err != nil {
		return nil, common.Address{}, err
	}
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: big.NewInt(1),
		GasFeeCap: gasFeeCap,
		Gas:       gas,
		Data:      bytecode,
	})
	signed, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), key)
	if err != nil {
		return nil, common.Address{}, err
	}
	return signed, crypto.CreateAddress(from, nonce), nil
}

func TestSimulatedBackend(t *testing.T) {
	t.Parallel()
	key, _ := crypto.GenerateKey()
	auth, _ := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
	alloc := types.GenesisAlloc{auth.From: {Balance: big.NewInt(9223372036854775807)}}
	sim := New(alloc, 8000029)
	defer sim.Close()

	client := sim.Client()
	ctx := context.Background()

	_, pending, err := client.TransactionByHash(ctx, common.HexToHash("0x2"))
	if pending || !errors.Is(err, ethereum.ErrNotFound) {
		t.Fatalf("expected not found and not pending, got err=%v pending=%v", err, pending)
	}

	head, _ := client.HeaderByNumber(ctx, nil)
	gasPrice := new(big.Int).Add(head.BaseFee, big.NewInt(1))
	tx := types.NewContractCreation(0, big.NewInt(0), 3000000, gasPrice, common.FromHex("6060604052600a8060106000396000f360606040526008565b00"))
	tx, _ = types.SignTx(tx, types.HomesteadSigner{}, key)
	if err := client.SendTransaction(ctx, tx); err != nil {
		t.Fatalf("send tx failed: %v", err)
	}
	_, pending, err = client.TransactionByHash(ctx, tx.Hash())
	if err != nil || !pending {
		t.Fatalf("expected pending tx, err=%v pending=%v", err, pending)
	}
	sim.Commit()
	_, pending, err = client.TransactionByHash(ctx, tx.Hash())
	if err != nil || pending {
		t.Fatalf("expected mined tx, err=%v pending=%v", err, pending)
	}
}

func TestNewSimulatedBackend(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()
	bal, err := sim.Client().BalanceAt(context.Background(), testAddr, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bal.Cmp(big.NewInt(10000000000000000)) != 0 {
		t.Fatalf("unexpected balance %v", bal)
	}
}

func TestAdjustTime(t *testing.T) {
	sim := New(types.GenesisAlloc{}, 10_000_000)
	defer sim.Close()

	client := sim.Client()
	block1, _ := client.BlockByNumber(context.Background(), nil)

	// Create a block
	if err := sim.AdjustTime(time.Minute); err != nil {
		t.Fatal(err)
	}
	block2, _ := client.BlockByNumber(context.Background(), nil)
	prevTime := block1.Time()
	newTime := block2.Time()
	if newTime-prevTime != uint64(time.Minute.Seconds()) {
		t.Errorf("adjusted time not equal to 60 seconds. prev: %v, new: %v", prevTime, newTime)
	}
}

func TestNewAdjustTimeFail(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()
	client := sim.Client()
	ctx := context.Background()

	head, _ := client.HeaderByNumber(ctx, nil)
	gasPrice := new(big.Int).Add(head.BaseFee, big.NewInt(1))
	tx := types.NewTransaction(0, testAddr, big.NewInt(1000), params.TxGas, gasPrice, nil)
	signedTx, _ := types.SignTx(tx, types.HomesteadSigner{}, testKey)
	_ = client.SendTransaction(ctx, signedTx)
	if err := sim.AdjustTime(time.Second); err == nil {
		t.Fatal("expected adjust time to fail on non-empty block")
	}
	sim.Commit()

	prevTime := sim.pendingBlock.Time()
	if err := sim.AdjustTime(time.Minute); err != nil {
		t.Fatal(err)
	}
	newTime := sim.pendingBlock.Time()
	if newTime-prevTime != uint64(time.Minute.Seconds()) {
		t.Fatalf("adjusted time mismatch")
	}

	tx2 := types.NewTransaction(1, testAddr, big.NewInt(1000), params.TxGas, gasPrice, nil)
	signedTx2, _ := types.SignTx(tx2, types.HomesteadSigner{}, testKey)
	_ = client.SendTransaction(ctx, signedTx2)
	sim.Commit()
	newTime = sim.pendingBlock.Time()
	if newTime < prevTime {
		t.Fatalf("time moved backwards unexpectedly")
	}
}

func TestBalanceAt(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()
	bal, err := sim.Client().BalanceAt(context.Background(), testAddr, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bal.Cmp(big.NewInt(10000000000000000)) != 0 {
		t.Fatalf("unexpected balance %v", bal)
	}
}

func TestBlockByHash(t *testing.T) {
	t.Parallel()
	sim := New(types.GenesisAlloc{}, 10000000)
	defer sim.Close()
	client := sim.Client()
	ctx := context.Background()
	block, _ := client.BlockByNumber(ctx, nil)
	byHash, err := client.BlockByHash(ctx, block.Hash())
	if err != nil || byHash.Hash() != block.Hash() {
		t.Fatalf("block by hash mismatch: err=%v", err)
	}
}

func TestBlockByNumber(t *testing.T) {
	t.Parallel()
	sim := New(types.GenesisAlloc{}, 10000000)
	defer sim.Close()
	client := sim.Client()
	ctx := context.Background()
	block, _ := client.BlockByNumber(ctx, nil)
	if block.NumberU64() != 0 {
		t.Fatalf("expected block 0")
	}
	sim.Commit()
	latest, _ := client.BlockByNumber(ctx, nil)
	if latest.NumberU64() != 1 {
		t.Fatalf("expected block 1")
	}
	one, err := client.BlockByNumber(ctx, big.NewInt(1))
	if err != nil || one.Hash() != latest.Hash() {
		t.Fatalf("block by number mismatch: err=%v", err)
	}
}

func TestNonceAt(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()
	client := sim.Client()
	ctx := context.Background()
	nonce, _ := client.NonceAt(ctx, testAddr, big.NewInt(0))
	if nonce != 0 {
		t.Fatalf("expected nonce 0")
	}
	head, _ := client.HeaderByNumber(ctx, nil)
	gasPrice := new(big.Int).Add(head.BaseFee, big.NewInt(1))
	tx := types.NewTransaction(nonce, testAddr, big.NewInt(1000), params.TxGas, gasPrice, nil)
	signed, _ := types.SignTx(tx, types.HomesteadSigner{}, testKey)
	_ = client.SendTransaction(ctx, signed)
	sim.Commit()
	n1, _ := client.NonceAt(ctx, testAddr, big.NewInt(1))
	if n1 != 1 {
		t.Fatalf("expected nonce 1")
	}
	sim.Commit()
	n1Again, _ := client.NonceAt(ctx, testAddr, big.NewInt(1))
	if n1Again != 1 {
		t.Fatalf("expected historical nonce 1")
	}
}

func TestSendTransaction(t *testing.T) {
	sim := simTestBackend(testAddr)
	defer sim.Close()

	client := sim.Client()
	ctx := context.Background()

	signedTx, err := newTx(sim, testKey)
	if err != nil {
		t.Errorf("could not create transaction: %v", err)
	}
	// send tx to simulated backend
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		t.Errorf("could not add tx to pending block: %v", err)
	}
	sim.Commit()
	block, err := client.BlockByNumber(ctx, big.NewInt(1))
	if err != nil {
		t.Errorf("could not get block at height 1: %v", err)
	}

	if signedTx.Hash() != block.Transactions()[0].Hash() {
		t.Errorf("did not commit sent transaction. expected hash %v got hash %v", block.Transactions()[0].Hash(), signedTx.Hash())
	}
}

func TestTransactionByHash(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()

	client := sim.Client()
	ctx := context.Background()

	head, _ := client.HeaderByNumber(ctx, nil)
	gasPrice := new(big.Int).Add(head.BaseFee, big.NewInt(1))
	tx := types.NewTransaction(0, testAddr, big.NewInt(1000), params.TxGas, gasPrice, nil)
	signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, testKey)
	if err != nil {
		t.Fatalf("could not sign tx: %v", err)
	}
	if err := client.SendTransaction(ctx, signedTx); err != nil {
		t.Fatalf("could not send tx: %v", err)
	}

	receivedTx, pending, err := client.TransactionByHash(ctx, signedTx.Hash())
	if err != nil || !pending || receivedTx.Hash() != signedTx.Hash() {
		t.Fatalf("expected pending tx by hash, err=%v pending=%v", err, pending)
	}

	sim.Commit()
	receivedTx, pending, err = client.TransactionByHash(ctx, signedTx.Hash())
	if err != nil || pending || receivedTx.Hash() != signedTx.Hash() {
		t.Fatalf("expected mined tx by hash, err=%v pending=%v", err, pending)
	}
}

func TestEstimateGas(t *testing.T) {
	t.Parallel()
	const contractAbi = "[{\"inputs\":[],\"name\":\"Assert\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"OOG\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"PureRevert\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"Revert\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"Valid\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"
	const contractBin = "0x60806040523480156100115760006000fd5b50610017565b61016e806100266000396000f3fe60806040523480156100115760006000fd5b506004361061005c5760003560e01c806350f6fe3414610062578063aa8b1d301461006c578063b9b046f914610076578063d8b9839114610080578063e09fface1461008a5761005c565b60006000fd5b61006a610094565b005b6100746100ad565b005b61007e6100b5565b005b6100886100c2565b005b610092610135565b005b6000600090505b5b808060010191505061009b565b505b565b60006000fd5b565b600015156100bf57fe5b5b565b6040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252600d8152602001807f72657665727420726561736f6e0000000000000000000000000000000000000081526020015060200191505060405180910390fd5b565b5b56fea2646970667358221220345bbcbb1a5ecf22b53a78eaebf95f8ee0eceff6d10d4b9643495084d2ec934a64736f6c63430006040033"

	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	opts, _ := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
	sim := New(types.GenesisAlloc{addr: {Balance: big.NewInt(params.Ether)}}, 10000000)
	defer sim.Close()

	parsed, _ := abi.JSON(strings.NewReader(contractAbi))
	contractAddr, _, _, _ := bind.DeployContract(opts, parsed, common.FromHex(contractBin), sim)
	sim.Commit()

	cases := []struct {
		message     ethereum.CallMsg
		expect      uint64
		expectError error
		expectData  interface{}
	}{
		{ethereum.CallMsg{From: addr, To: &addr, GasPrice: big.NewInt(0), Value: big.NewInt(1)}, params.TxGas, nil, nil},
		{ethereum.CallMsg{From: addr, To: &contractAddr, GasPrice: big.NewInt(0), Value: big.NewInt(1)}, 0, errors.New("execution reverted"), nil},
		{ethereum.CallMsg{From: addr, To: &contractAddr, GasPrice: big.NewInt(0), Data: common.Hex2Bytes("d8b98391")}, 0, errors.New("execution reverted: revert reason"), "0x08c379a00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d72657665727420726561736f6e00000000000000000000000000000000000000"},
		{ethereum.CallMsg{From: addr, To: &contractAddr, GasPrice: big.NewInt(0), Data: common.Hex2Bytes("aa8b1d30")}, 0, errors.New("execution reverted"), nil},
		{ethereum.CallMsg{From: addr, To: &contractAddr, Gas: 100000, GasPrice: big.NewInt(0), Data: common.Hex2Bytes("50f6fe34")}, 0, errors.New("gas required exceeds allowance (100000)"), nil},
		{ethereum.CallMsg{From: addr, To: &contractAddr, Gas: 100000, GasPrice: big.NewInt(0), Data: common.Hex2Bytes("b9b046f9")}, 0, errors.New("invalid opcode: INVALID"), nil},
		{ethereum.CallMsg{From: addr, To: &contractAddr, Gas: 100000, GasPrice: big.NewInt(0), Data: common.Hex2Bytes("e09fface")}, 21483, nil, nil},
	}
	for _, c := range cases {
		got, err := sim.EstimateGas(context.Background(), c.message)
		if c.expectError != nil {
			if err == nil || err.Error() != c.expectError.Error() {
				t.Fatalf("expected error %v, got %v", c.expectError, err)
			}
			if c.expectData != nil {
				rerr, ok := err.(*revertError)
				if !ok || !reflect.DeepEqual(rerr.ErrorData(), c.expectData) {
					t.Fatalf("revert data mismatch")
				}
			}
			continue
		}
		if got != c.expect {
			t.Fatalf("gas mismatch, want %d got %d", c.expect, got)
		}
	}
}

func TestEstimateGasWithPrice(t *testing.T) {
	t.Parallel()
	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	sim := New(types.GenesisAlloc{addr: {Balance: big.NewInt(params.Ether*2 + 2e17)}}, 10000000)
	defer sim.Close()

	recipient := common.HexToAddress("deadbeef")
	cases := []ethereum.CallMsg{
		{From: addr, To: &recipient, GasPrice: big.NewInt(0), Value: big.NewInt(100000000000)},
		{From: addr, To: &recipient, GasPrice: big.NewInt(100000000000), Value: big.NewInt(100000000000)},
		{From: addr, To: &recipient, GasPrice: big.NewInt(1e14), Value: big.NewInt(1e17)},
	}
	for i, c := range cases {
		got, err := sim.EstimateGas(context.Background(), c)
		if err != nil || got != 21000 {
			t.Fatalf("case %d failed, gas=%d err=%v", i, got, err)
		}
	}
}

func TestHeaderByHash(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()
	ctx := context.Background()
	client := sim.Client()
	head, _ := client.HeaderByNumber(ctx, nil)
	byHash, err := client.HeaderByHash(ctx, head.Hash())
	if err != nil || byHash.Hash() != head.Hash() {
		t.Fatalf("header by hash mismatch: err=%v", err)
	}
}

func TestHeaderByNumber(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()
	ctx := context.Background()
	client := sim.Client()
	head0, _ := client.HeaderByNumber(ctx, nil)
	if head0.Number.Uint64() != 0 {
		t.Fatalf("expected head 0")
	}
	sim.Commit()
	latest, _ := client.HeaderByNumber(ctx, nil)
	head1, _ := client.HeaderByNumber(ctx, big.NewInt(1))
	if head1.Hash() != latest.Hash() || head1.Number.Uint64() != 1 {
		t.Fatalf("header by number mismatch")
	}
	block1, _ := client.BlockByNumber(ctx, big.NewInt(1))
	if block1.Hash() != head1.Hash() {
		t.Fatalf("block/header hash mismatch")
	}
}

func TestTransactionCount(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()
	ctx := context.Background()
	client := sim.Client()
	cur, _ := client.BlockByNumber(ctx, nil)
	count, _ := sim.TransactionCount(ctx, cur.Hash())
	if count != 0 {
		t.Fatalf("expected 0 tx count")
	}
	head, _ := client.HeaderByNumber(ctx, nil)
	gasPrice := new(big.Int).Add(head.BaseFee, big.NewInt(1))
	tx := types.NewTransaction(0, testAddr, big.NewInt(1000), params.TxGas, gasPrice, nil)
	signed, _ := types.SignTx(tx, types.HomesteadSigner{}, testKey)
	_ = client.SendTransaction(ctx, signed)
	sim.Commit()
	last, _ := client.BlockByNumber(ctx, nil)
	count, _ = sim.TransactionCount(ctx, last.Hash())
	if count != 1 {
		t.Fatalf("expected 1 tx count")
	}
}

func TestTransactionInBlock(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()
	ctx := context.Background()
	client := sim.Client()
	if tx, err := sim.TransactionInBlock(ctx, sim.pendingBlock.Hash(), 0); err == nil || tx != nil {
		t.Fatalf("expected missing tx in empty pending block")
	}
	head, _ := client.HeaderByNumber(ctx, nil)
	gasPrice := new(big.Int).Add(head.BaseFee, big.NewInt(1))
	tx := types.NewTransaction(0, testAddr, big.NewInt(1000), params.TxGas, gasPrice, nil)
	signed, _ := types.SignTx(tx, types.HomesteadSigner{}, testKey)
	_ = client.SendTransaction(ctx, signed)
	sim.Commit()
	last, _ := client.BlockByNumber(ctx, nil)
	if tx1, err := sim.TransactionInBlock(ctx, last.Hash(), 1); err == nil || tx1 != nil {
		t.Fatalf("expected missing tx at index 1")
	}
	tx0, err := sim.TransactionInBlock(ctx, last.Hash(), 0)
	if err != nil || tx0.Hash() != signed.Hash() {
		t.Fatalf("tx in block mismatch: err=%v", err)
	}
}

func TestPendingNonceAt(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()
	ctx := context.Background()
	client := sim.Client()
	p0, _ := client.PendingNonceAt(ctx, testAddr)
	if p0 != 0 {
		t.Fatalf("expected pending nonce 0")
	}
	head, _ := client.HeaderByNumber(ctx, nil)
	gasPrice := new(big.Int).Add(head.BaseFee, big.NewInt(1))
	tx0 := types.NewTransaction(0, testAddr, big.NewInt(1000), params.TxGas, gasPrice, nil)
	signed0, _ := types.SignTx(tx0, types.HomesteadSigner{}, testKey)
	_ = client.SendTransaction(ctx, signed0)
	p1, _ := client.PendingNonceAt(ctx, testAddr)
	if p1 != 1 {
		t.Fatalf("expected pending nonce 1")
	}
	tx1 := types.NewTransaction(1, testAddr, big.NewInt(1000), params.TxGas, gasPrice, nil)
	signed1, _ := types.SignTx(tx1, types.HomesteadSigner{}, testKey)
	_ = client.SendTransaction(ctx, signed1)
	p2, _ := client.PendingNonceAt(ctx, testAddr)
	if p2 != 2 {
		t.Fatalf("expected pending nonce 2")
	}
}

func TestTransactionReceipt(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()
	ctx := context.Background()
	client := sim.Client()
	head, _ := client.HeaderByNumber(ctx, nil)
	gasPrice := new(big.Int).Add(head.BaseFee, big.NewInt(1))
	tx := types.NewTransaction(0, testAddr, big.NewInt(1000), params.TxGas, gasPrice, nil)
	signed, _ := types.SignTx(tx, types.HomesteadSigner{}, testKey)
	_ = client.SendTransaction(ctx, signed)
	sim.Commit()
	receipt, err := client.TransactionReceipt(ctx, signed.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.TxHash != signed.Hash() {
		t.Fatalf("receipt tx hash mismatch")
	}
}

func TestSuggestGasPrice(t *testing.T) {
	t.Parallel()
	sim := New(types.GenesisAlloc{}, 10000000)
	defer sim.Close()
	price, err := sim.Client().SuggestGasPrice(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	baseFee := sim.pendingBlock.Header().BaseFee
	if baseFee == nil {
		baseFee = big.NewInt(1)
	}
	if price.Cmp(baseFee) != 0 {
		t.Fatalf("unexpected gas price %v want %v", price, baseFee)
	}
}

func TestPendingCodeAt(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()
	ctx := context.Background()
	client := sim.Client()
	code, _ := client.CodeAt(ctx, testAddr, nil)
	if len(code) != 0 {
		t.Fatalf("expected no code at EOA")
	}
	parsed, _ := abi.JSON(strings.NewReader(abiJSON))
	auth, _ := bind.NewKeyedTransactorWithChainID(testKey, big.NewInt(1337))
	contractAddr, _, _, err := bind.DeployContract(auth, parsed, common.FromHex(abiBin), sim)
	if err != nil {
		t.Fatal(err)
	}
	pendingCode, err := client.PendingCodeAt(ctx, contractAddr)
	if err != nil || len(pendingCode) == 0 {
		t.Fatalf("pending code unavailable: err=%v", err)
	}
	if !bytes.Equal(pendingCode, common.FromHex(deployedCode)) {
		t.Fatalf("pending code mismatch")
	}
}

func TestCodeAt(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()
	ctx := context.Background()
	client := sim.Client()
	code, _ := client.CodeAt(ctx, testAddr, nil)
	if len(code) != 0 {
		t.Fatalf("expected no code at EOA")
	}
	parsed, _ := abi.JSON(strings.NewReader(abiJSON))
	auth, _ := bind.NewKeyedTransactorWithChainID(testKey, big.NewInt(1337))
	contractAddr, _, _, err := bind.DeployContract(auth, parsed, common.FromHex(abiBin), sim)
	if err != nil {
		t.Fatal(err)
	}
	sim.Commit()
	code, err = client.CodeAt(ctx, contractAddr, nil)
	if err != nil || len(code) == 0 {
		t.Fatalf("code unavailable: err=%v", err)
	}
	if !bytes.Equal(code, common.FromHex(deployedCode)) {
		t.Fatalf("code mismatch")
	}
}

func TestCodeAtHash(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()
	ctx := context.Background()
	client := sim.Client()
	head, _ := client.HeaderByNumber(ctx, nil)
	code, err := sim.CodeAtHash(ctx, testAddr, head.Hash())
	if err != nil || len(code) != 0 {
		t.Fatalf("expected no code at EOA: err=%v", err)
	}
	parsed, _ := abi.JSON(strings.NewReader(abiJSON))
	auth, _ := bind.NewKeyedTransactorWithChainID(testKey, big.NewInt(1337))
	contractAddr, _, _, err := bind.DeployContract(auth, parsed, common.FromHex(abiBin), sim)
	if err != nil {
		t.Fatal(err)
	}
	blockHash := sim.Commit()
	code, err = sim.CodeAtHash(ctx, contractAddr, blockHash)
	if err != nil || len(code) == 0 {
		t.Fatalf("code at hash unavailable: err=%v", err)
	}
	if !bytes.Equal(code, common.FromHex(deployedCode)) {
		t.Fatalf("code at hash mismatch")
	}
}

func TestPendingAndCallContract(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()
	ctx := context.Background()
	parsed, _ := abi.JSON(strings.NewReader(abiJSON))
	auth, _ := bind.NewKeyedTransactorWithChainID(testKey, big.NewInt(1337))
	addr, _, _, err := bind.DeployContract(auth, parsed, common.FromHex(abiBin), sim)
	if err != nil {
		t.Fatal(err)
	}
	input, _ := parsed.Pack("receive", []byte("X"))
	res, err := sim.PendingCallContract(ctx, ethereum.CallMsg{From: testAddr, To: &addr, Data: input})
	if err != nil || len(res) == 0 {
		t.Fatalf("pending call failed: err=%v", err)
	}
	if !bytes.Equal(res, expectedReturn) || !strings.Contains(string(res), "hello world") {
		t.Fatalf("unexpected pending call return")
	}
	blockHash := sim.Commit()
	res, err = sim.CallContract(ctx, ethereum.CallMsg{From: testAddr, To: &addr, Data: input}, nil)
	if err != nil || len(res) == 0 {
		t.Fatalf("call failed: err=%v", err)
	}
	if !bytes.Equal(res, expectedReturn) || !strings.Contains(string(res), "hello world") {
		t.Fatalf("unexpected call return")
	}
	res, err = sim.CallContractAtHash(ctx, ethereum.CallMsg{From: testAddr, To: &addr, Data: input}, blockHash)
	if err != nil || len(res) == 0 {
		t.Fatalf("call at hash failed: err=%v", err)
	}
	if !bytes.Equal(res, expectedReturn) || !strings.Contains(string(res), "hello world") {
		t.Fatalf("unexpected call at hash return")
	}
}

func TestCallContractRevert(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()

	client := sim.Client()
	ctx := context.Background()

	bytecode := common.FromHex("6005600c60003960056000f360006000fd")
	tx, contractAddr, err := newContractCreationTx(sim, testKey, bytecode, 300000)
	if err != nil {
		t.Fatalf("could not create deploy tx: %v", err)
	}
	if err := client.SendTransaction(ctx, tx); err != nil {
		t.Fatalf("could not send deploy tx: %v", err)
	}
	sim.Commit()

	_, err = client.CallContract(ctx, ethereum.CallMsg{From: testAddr, To: &contractAddr}, nil)
	if err == nil || !strings.Contains(err.Error(), "execution reverted") {
		t.Fatalf("expected execution reverted error, got: %v", err)
	}
}

func TestFork(t *testing.T) {
	t.Parallel()
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	sim := simTestBackend(testAddr)
	defer sim.Close()

	client := sim.Client()
	ctx := context.Background()

	// 1.
	parent, _ := client.HeaderByNumber(ctx, nil)

	// 2.
	n := int(rand.Int31n(21))
	for i := 0; i < n; i++ {
		sim.Commit()
	}

	// 3.
	b, _ := client.BlockNumber(ctx)
	if b != uint64(n) {
		t.Error("wrong chain length")
	}

	// 4.
	sim.Fork(parent.Hash())

	// 5.
	for i := 0; i < n+1; i++ {
		sim.Commit()
	}

	// 6.
	b, _ = client.BlockNumber(ctx)
	if b != uint64(n+1) {
		t.Error("wrong chain length")
	}
}

func TestForkLogsReborn(t *testing.T) {
	t.Parallel()
	sim, client, ctx, auth, contract, _, _, parentHash := setupForkLogsRebornScenario(t)
	defer sim.Close()

	var err error

	logs, sub, err := contract.WatchLogs(nil, "Called")
	if err != nil {
		t.Fatalf("watching logs: %v", err)
	}
	defer sub.Unsubscribe()

	tx, err := contract.Transact(auth, "Call")
	if err != nil {
		t.Fatalf("sending contract tx: %v", err)
	}
	sim.Commit()

	lg := mustReadWatchedLog(t, logs, sub, "included")
	if lg.TxHash != tx.Hash() {
		t.Fatalf("wrong included event tx hash: got %s want %s", lg.TxHash, tx.Hash())
	}
	if lg.Removed {
		t.Fatalf("event should be included")
	}

	if err := sim.Fork(parentHash); err != nil {
		t.Fatalf("forking: %v", err)
	}
	sim.Commit()
	sim.Commit()

	lg = mustReadWatchedLog(t, logs, sub, "removed")
	if lg.TxHash != tx.Hash() {
		t.Fatalf("wrong removed event tx hash: got %s want %s", lg.TxHash, tx.Hash())
	}
	if !lg.Removed {
		t.Fatalf("event should be removed after reorg")
	}

	if err := client.SendTransaction(ctx, tx); err != nil {
		t.Fatalf("re-sending transaction: %v", err)
	}
	sim.Commit()

	lg = mustReadWatchedLog(t, logs, sub, "reborn")
	if lg.TxHash != tx.Hash() {
		t.Fatalf("wrong reborn event tx hash: got %s want %s", lg.TxHash, tx.Hash())
	}
	if lg.Removed {
		t.Fatalf("event should be reborn as included")
	}
}

func TestForkResendTx(t *testing.T) {
	t.Parallel()
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	sim := simTestBackend(testAddr)
	defer sim.Close()

	client := sim.Client()
	ctx := context.Background()

	// 1.
	parent, _ := client.HeaderByNumber(ctx, nil)

	// 2.
	tx, err := newTx(sim, testKey)
	if err != nil {
		t.Fatalf("could not create transaction: %v", err)
	}
	client.SendTransaction(ctx, tx)
	sim.Commit()

	// 3.
	receipt, _ := client.TransactionReceipt(ctx, tx.Hash())
	if h := receipt.BlockNumber.Uint64(); h != 1 {
		t.Errorf("TX included in wrong block: %d", h)
	}

	// 4.
	if err := sim.Fork(parent.Hash()); err != nil {
		t.Errorf("forking: %v", err)
	}

	// 5.
	sim.Commit()
	if err := client.SendTransaction(ctx, tx); err != nil {
		t.Fatalf("sending transaction: %v", err)
	}
	sim.Commit()
	receipt, _ = client.TransactionReceipt(ctx, tx.Hash())
	if h := receipt.BlockNumber.Uint64(); h != 2 {
		t.Errorf("TX included in wrong block: %d", h)
	}
}

func TestCommitReturnValue(t *testing.T) {
	t.Parallel()
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	sim := simTestBackend(testAddr)
	defer sim.Close()

	client := sim.Client()
	ctx := context.Background()

	// Test if Commit returns the correct block hash
	h1 := sim.Commit()
	cur, _ := client.HeaderByNumber(ctx, nil)
	if h1 != cur.Hash() {
		t.Error("Commit did not return the hash of the last block.")
	}

	// Create a block in the original chain (containing a transaction to force different block hashes)
	head, _ := client.HeaderByNumber(ctx, nil) // Should be child's, good enough
	gasPrice := new(big.Int).Add(head.BaseFee, big.NewInt(1))
	_tx := types.NewTransaction(0, testAddr, big.NewInt(1000), params.TxGas, gasPrice, nil)
	tx, _ := types.SignTx(_tx, types.HomesteadSigner{}, testKey)
	client.SendTransaction(ctx, tx)

	h2 := sim.Commit()

	// Create another block in the original chain
	sim.Commit()

	// Fork at the first bock
	if err := sim.Fork(h1); err != nil {
		t.Errorf("forking: %v", err)
	}

	// Test if Commit returns the correct block hash after the reorg
	h2fork := sim.Commit()
	if h2 == h2fork {
		t.Error("The block in the fork and the original block are the same block!")
	}
	if header, err := client.HeaderByHash(ctx, h2fork); err != nil || header == nil {
		t.Error("Could not retrieve the just created block (side-chain)")
	}
}

func TestAdjustTimeAfterFork(t *testing.T) {
	t.Parallel()
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	sim := simTestBackend(testAddr)
	defer sim.Close()

	client := sim.Client()
	ctx := context.Background()

	sim.Commit() // h1
	h1, _ := client.HeaderByNumber(ctx, nil)

	sim.Commit() // h2
	sim.Fork(h1.Hash())
	sim.AdjustTime(1 * time.Second)
	sim.Commit()

	head, _ := client.HeaderByNumber(ctx, nil)
	if head.Number.Uint64() == 2 && head.ParentHash != h1.Hash() {
		t.Errorf("failed to build block on fork")
	}
}

func TestNewSim(t *testing.T) {
	sim := New(types.GenesisAlloc{}, 30_000_000)
	defer sim.Close()

	client := sim.Client()
	num, err := client.BlockNumber(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if num != 0 {
		t.Fatalf("expected 0 got %v", num)
	}
	// Create a block
	sim.Commit()
	num, err = client.BlockNumber(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if num != 1 {
		t.Fatalf("expected 1 got %v", num)
	}
}

func TestTransactionByHashLifecycle(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()

	client := sim.Client()
	ctx := context.Background()

	if _, pending, err := client.TransactionByHash(ctx, common.HexToHash("0x1234")); !errors.Is(err, ethereum.ErrNotFound) || pending {
		t.Fatalf("expected not found and not pending, got err=%v pending=%v", err, pending)
	}

	tx, err := newTx(sim, testKey)
	if err != nil {
		t.Fatalf("could not create tx: %v", err)
	}
	if err := client.SendTransaction(ctx, tx); err != nil {
		t.Fatalf("could not send tx: %v", err)
	}

	got, pending, err := client.TransactionByHash(ctx, tx.Hash())
	if err != nil || !pending || got.Hash() != tx.Hash() {
		t.Fatalf("expected pending tx before commit, got err=%v pending=%v", err, pending)
	}

	sim.Commit()
	got, pending, err = client.TransactionByHash(ctx, tx.Hash())
	if err != nil || pending || got.Hash() != tx.Hash() {
		t.Fatalf("expected mined tx after commit, got err=%v pending=%v", err, pending)
	}
}

func TestTransactionReceiptLifecycle(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()

	client := sim.Client()
	ctx := context.Background()

	if _, err := client.TransactionReceipt(ctx, common.HexToHash("0x1234")); !errors.Is(err, ethereum.ErrNotFound) {
		t.Fatalf("expected not found before tx mining, got %v", err)
	}

	tx, err := newTx(sim, testKey)
	if err != nil {
		t.Fatalf("could not create tx: %v", err)
	}
	if err := client.SendTransaction(ctx, tx); err != nil {
		t.Fatalf("could not send tx: %v", err)
	}
	sim.Commit()

	receipt, err := client.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		t.Fatalf("could not fetch receipt: %v", err)
	}
	if receipt.TxHash != tx.Hash() || receipt.BlockNumber == nil || receipt.BlockNumber.Uint64() != 1 {
		t.Fatalf("unexpected receipt content: tx=%s block=%v", receipt.TxHash, receipt.BlockNumber)
	}
}

func TestSuggestGasPriceAndTipCap(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()

	client := sim.Client()
	ctx := context.Background()

	head, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		t.Fatalf("could not fetch header: %v", err)
	}
	price, err := client.SuggestGasPrice(ctx)
	if err != nil {
		t.Fatalf("could not suggest gas price: %v", err)
	}
	if head.BaseFee != nil && price.Cmp(head.BaseFee) != 0 {
		t.Fatalf("unexpected suggested gas price: got %v want %v", price, head.BaseFee)
	}
	tip, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		t.Fatalf("could not suggest tip cap: %v", err)
	}
	if tip.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("unexpected tip cap: got %v want 1", tip)
	}
}

func TestEstimateGasSimpleTransfer(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()

	client := sim.Client()
	ctx := context.Background()

	gas, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From: testAddr,
		To:   &testAddr,
	})
	if err != nil {
		t.Fatalf("estimate gas failed: %v", err)
	}
	if gas < params.TxGas {
		t.Fatalf("estimated gas too low: got %d want >= %d", gas, params.TxGas)
	}
}

func TestFeeHistoryBasic(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()

	client := sim.Client()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		tx, err := newTx(sim, testKey)
		if err != nil {
			t.Fatalf("could not create tx: %v", err)
		}
		if err := client.SendTransaction(ctx, tx); err != nil {
			t.Fatalf("could not send tx: %v", err)
		}
		sim.Commit()
	}

	history, err := client.FeeHistory(ctx, 2, nil, []float64{50, 90})
	if err != nil {
		t.Fatalf("fee history failed: %v", err)
	}
	if history == nil || history.OldestBlock == nil {
		t.Fatalf("fee history response is incomplete")
	}
	if len(history.BaseFee) != 3 || len(history.GasUsedRatio) != 2 || len(history.Reward) != 2 {
		t.Fatalf("unexpected fee history lengths: base=%d gas=%d reward=%d", len(history.BaseFee), len(history.GasUsedRatio), len(history.Reward))
	}
	for i, rewards := range history.Reward {
		if len(rewards) != 2 {
			t.Fatalf("unexpected reward percentiles at block %d: got %d", i, len(rewards))
		}
	}
}

func TestPendingCodeAtAndCodeAt(t *testing.T) {
	t.Parallel()
	sim := simTestBackend(testAddr)
	defer sim.Close()

	client := sim.Client()
	ctx := context.Background()

	bytecode := common.FromHex("6005600c60003960056000f360006000fd")
	tx, contractAddr, err := newContractCreationTx(sim, testKey, bytecode, 300000)
	if err != nil {
		t.Fatalf("could not create deploy tx: %v", err)
	}

	code, err := client.CodeAt(ctx, contractAddr, nil)
	if err != nil {
		t.Fatalf("could not query code before deploy: %v", err)
	}
	if len(code) != 0 {
		t.Fatalf("expected empty code before deployment, got length %d", len(code))
	}

	if err := client.SendTransaction(ctx, tx); err != nil {
		t.Fatalf("could not send deploy tx: %v", err)
	}
	pendingCode, err := client.PendingCodeAt(ctx, contractAddr)
	if err != nil {
		t.Fatalf("could not query pending code: %v", err)
	}
	if len(pendingCode) == 0 {
		t.Fatalf("expected pending code for contract")
	}

	sim.Commit()
	code, err = client.CodeAt(ctx, contractAddr, nil)
	if err != nil {
		t.Fatalf("could not query code after deploy: %v", err)
	}
	if len(code) == 0 {
		t.Fatalf("expected non-empty code after deployment")
	}
}

func TestForkLogsRebornFilterLogs(t *testing.T) {
	t.Parallel()
	sim, client, ctx, auth, contract, contractAddr, calledEventID, parentHash := setupForkLogsRebornScenario(t)
	defer sim.Close()

	var err error
	tx, err := contract.Transact(auth, "Call")
	if err != nil {
		t.Fatalf("sending contract tx: %v", err)
	}
	sim.Commit()

	query := calledEventQuery(contractAddr, calledEventID, nil, nil)
	logs, err := client.FilterLogs(ctx, query)
	if err != nil {
		t.Fatalf("filter logs before fork: %v", err)
	}
	if len(logs) != 1 || logs[0].TxHash != tx.Hash() {
		t.Fatalf("expected exactly one canonical log before fork, got len=%d tx=%v", len(logs), logs)
	}

	if err := sim.Fork(parentHash); err != nil {
		t.Fatalf("forking: %v", err)
	}
	sim.Commit()
	sim.Commit()

	logs, err = client.FilterLogs(ctx, query)
	if err != nil {
		t.Fatalf("filter logs after reorg removal: %v", err)
	}
	if len(logs) != 0 {
		t.Fatalf("expected no canonical logs after reorg removal, got len=%d", len(logs))
	}

	if err := client.SendTransaction(ctx, tx); err != nil {
		t.Fatalf("re-sending transaction: %v", err)
	}
	sim.Commit()

	logs, err = client.FilterLogs(ctx, query)
	if err != nil {
		t.Fatalf("filter logs after reborn: %v", err)
	}
	if len(logs) != 1 || logs[0].TxHash != tx.Hash() {
		t.Fatalf("expected exactly one canonical log after reborn, got len=%d tx=%v", len(logs), logs)
	}
}

func TestForkLogsRebornFilterLogsWithRange(t *testing.T) {
	t.Parallel()
	sim, client, ctx, auth, contract, contractAddr, calledEventID, parentHash := setupForkLogsRebornScenario(t)
	defer sim.Close()

	var err error
	tx, err := contract.Transact(auth, "Call")
	if err != nil {
		t.Fatalf("sending contract tx: %v", err)
	}
	sim.Commit() // block 2

	queryBlock2 := calledEventQuery(contractAddr, calledEventID, big.NewInt(2), big.NewInt(2))
	logs, err := client.FilterLogs(ctx, queryBlock2)
	if err != nil {
		t.Fatalf("filter logs in [2,2] before reorg: %v", err)
	}
	if len(logs) != 1 || logs[0].TxHash != tx.Hash() || logs[0].BlockNumber != 2 {
		t.Fatalf("expected one log at block 2 before reorg, got len=%d logs=%v", len(logs), logs)
	}

	if err := sim.Fork(parentHash); err != nil {
		t.Fatalf("forking: %v", err)
	}
	sim.Commit() // block 2 on new branch
	sim.Commit() // block 3 on new branch

	logs, err = client.FilterLogs(ctx, queryBlock2)
	if err != nil {
		t.Fatalf("filter logs in [2,2] after reorg removal: %v", err)
	}
	if len(logs) != 0 {
		t.Fatalf("expected zero logs at block 2 after reorg removal, got len=%d", len(logs))
	}

	if err := client.SendTransaction(ctx, tx); err != nil {
		t.Fatalf("re-sending transaction: %v", err)
	}
	sim.Commit() // block 4 on new branch

	queryBlock4 := calledEventQuery(contractAddr, calledEventID, big.NewInt(4), big.NewInt(4))
	logs, err = client.FilterLogs(ctx, queryBlock4)
	if err != nil {
		t.Fatalf("filter logs in [4,4] after reborn: %v", err)
	}
	if len(logs) != 1 || logs[0].TxHash != tx.Hash() || logs[0].BlockNumber != 4 {
		t.Fatalf("expected one log at block 4 after reborn, got len=%d logs=%v", len(logs), logs)
	}
}

// TestFork check that the chain length after a reorg is correct.
// Steps:
//  1. Save the current block which will serve as parent for the fork.
//  2. Mine n blocks with n ∈ [0, 20].
//  3. Assert that the chain length is n.
//  4. Fork by using the parent block as ancestor.
//  5. Mine n+1 blocks which should trigger a reorg.
//  6. Assert that the chain length is n+1.
//     Since Commit() was called 2n+1 times in total,
//     having a chain length of just n+1 means that a reorg occurred.

// TestForkResendTx checks that re-sending a TX after a fork
// is possible and does not cause a "nonce mismatch" panic.
// Steps:
//  1. Save the current block which will serve as parent for the fork.
//  2. Send a transaction.
//  3. Check that the TX is included in block 1.
//  4. Fork by using the parent block as ancestor.
//  5. Mine a block, Re-send the transaction and mine another one.
//  6. Check that the TX is now included in block 2.

// TestAdjustTimeAfterFork ensures that after a fork, AdjustTime uses the pending fork
// block's parent rather than the canonical head's parent.

func setupForkLogsRebornScenario(t *testing.T) (*Backend, Client, context.Context, *bind.TransactOpts, *bind.BoundContract, common.Address, common.Hash, common.Hash) {
	t.Helper()

	sim := simTestBackend(testAddr)
	client := sim.Client()
	ctx := context.Background()

	parsed, err := abi.JSON(strings.NewReader(callableAbi))
	if err != nil {
		t.Fatalf("parsing callable ABI: %v", err)
	}
	calledEvent, ok := parsed.Events["Called"]
	if !ok {
		t.Fatalf("missing Called event in ABI")
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		t.Fatalf("fetching chain id: %v", err)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(testKey, chainID)
	if err != nil {
		t.Fatalf("creating transactor: %v", err)
	}
	contractAddr, _, contract, err := bind.DeployContract(auth, parsed, common.FromHex(callableBin), client)
	if err != nil {
		t.Fatalf("deploying contract: %v", err)
	}
	sim.Commit()

	parent, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		t.Fatalf("fetching parent header: %v", err)
	}

	return sim, client, ctx, auth, contract, contractAddr, calledEvent.ID, parent.Hash()
}

func mustReadWatchedLog(t *testing.T, logs <-chan types.Log, sub ethereum.Subscription, step string) types.Log {
	t.Helper()

	select {
	case lg := <-logs:
		return lg
	case err := <-sub.Err():
		t.Fatalf("subscription error at %s: %v", step, err)
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for log at %s", step)
	}
	return types.Log{}
}

func calledEventQuery(contractAddr common.Address, calledEventID common.Hash, fromBlock, toBlock *big.Int) ethereum.FilterQuery {
	return ethereum.FilterQuery{
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Addresses: []common.Address{contractAddr},
		Topics:    [][]common.Hash{{calledEventID}},
	}
}
