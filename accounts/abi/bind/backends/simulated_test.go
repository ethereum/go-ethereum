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

package backends

import (
	"bytes"
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

func TestSimulatedBackend(t *testing.T) {
	var gasLimit uint64 = 8000029
	key, _ := crypto.GenerateKey() // nolint: gosec
	auth := bind.NewKeyedTransactor(key)
	genAlloc := make(core.GenesisAlloc)
	genAlloc[auth.From] = core.GenesisAccount{Balance: big.NewInt(9223372036854775807)}

	sim := NewSimulatedBackend(genAlloc, gasLimit)
	defer sim.Close()

	// should return an error if the tx is not found
	txHash := common.HexToHash("2")
	_, isPending, err := sim.TransactionByHash(context.Background(), txHash)

	if isPending {
		t.Fatal("transaction should not be pending")
	}
	if err != ethereum.NotFound {
		t.Fatalf("err should be `ethereum.NotFound` but received %v", err)
	}

	// generate a transaction and confirm you can retrieve it
	code := `6060604052600a8060106000396000f360606040526008565b00`
	var gas uint64 = 3000000
	tx := types.NewContractCreation(0, big.NewInt(0), gas, big.NewInt(1), common.FromHex(code))
	tx, _ = types.SignTx(tx, types.HomesteadSigner{}, key)

	err = sim.SendTransaction(context.Background(), tx)
	if err != nil {
		t.Fatal("error sending transaction")
	}

	txHash = tx.Hash()
	_, isPending, err = sim.TransactionByHash(context.Background(), txHash)
	if err != nil {
		t.Fatalf("error getting transaction with hash: %v", txHash.String())
	}
	if !isPending {
		t.Fatal("transaction should have pending status")
	}

	sim.Commit()
	_, isPending, err = sim.TransactionByHash(context.Background(), txHash)
	if err != nil {
		t.Fatalf("error getting transaction with hash: %v", txHash.String())
	}
	if isPending {
		t.Fatal("transaction should not have pending status")
	}
}

var testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

//  the following is based on this contract:
//  contract T {
//  	event received(address sender, uint amount, bytes memo);
//  	event receivedAddr(address sender);
//
//  	function receive(bytes calldata memo) external payable returns (string memory res) {
//  		emit received(msg.sender, msg.value, memo);
//  		emit receivedAddr(msg.sender);
//		    return "hello world";
//  	}
//  }
const abiJSON = `[ { "constant": false, "inputs": [ { "name": "memo", "type": "bytes" } ], "name": "receive", "outputs": [ { "name": "res", "type": "string" } ], "payable": true, "stateMutability": "payable", "type": "function" }, { "anonymous": false, "inputs": [ { "indexed": false, "name": "sender", "type": "address" }, { "indexed": false, "name": "amount", "type": "uint256" }, { "indexed": false, "name": "memo", "type": "bytes" } ], "name": "received", "type": "event" }, { "anonymous": false, "inputs": [ { "indexed": false, "name": "sender", "type": "address" } ], "name": "receivedAddr", "type": "event" } ]`
const abiBin = `0x608060405234801561001057600080fd5b506102a0806100206000396000f3fe60806040526004361061003b576000357c010000000000000000000000000000000000000000000000000000000090048063a69b6ed014610040575b600080fd5b6100b76004803603602081101561005657600080fd5b810190808035906020019064010000000081111561007357600080fd5b82018360208201111561008557600080fd5b803590602001918460018302840111640100000000831117156100a757600080fd5b9091929391929390505050610132565b6040518080602001828103825283818151815260200191508051906020019080838360005b838110156100f75780820151818401526020810190506100dc565b50505050905090810190601f1680156101245780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b60607f75fd880d39c1daf53b6547ab6cb59451fc6452d27caa90e5b6649dd8293b9eed33348585604051808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001848152602001806020018281038252848482818152602001925080828437600081840152601f19601f8201169050808301925050509550505050505060405180910390a17f46923992397eac56cf13058aced2a1871933622717e27b24eabc13bf9dd329c833604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390a16040805190810160405280600b81526020017f68656c6c6f20776f726c6400000000000000000000000000000000000000000081525090509291505056fea165627a7a72305820ff0c57dad254cfeda48c9cfb47f1353a558bccb4d1bc31da1dae69315772d29e0029`
const deployedCode = `60806040526004361061003b576000357c010000000000000000000000000000000000000000000000000000000090048063a69b6ed014610040575b600080fd5b6100b76004803603602081101561005657600080fd5b810190808035906020019064010000000081111561007357600080fd5b82018360208201111561008557600080fd5b803590602001918460018302840111640100000000831117156100a757600080fd5b9091929391929390505050610132565b6040518080602001828103825283818151815260200191508051906020019080838360005b838110156100f75780820151818401526020810190506100dc565b50505050905090810190601f1680156101245780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b60607f75fd880d39c1daf53b6547ab6cb59451fc6452d27caa90e5b6649dd8293b9eed33348585604051808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001848152602001806020018281038252848482818152602001925080828437600081840152601f19601f8201169050808301925050509550505050505060405180910390a17f46923992397eac56cf13058aced2a1871933622717e27b24eabc13bf9dd329c833604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390a16040805190810160405280600b81526020017f68656c6c6f20776f726c6400000000000000000000000000000000000000000081525090509291505056fea165627a7a72305820ff0c57dad254cfeda48c9cfb47f1353a558bccb4d1bc31da1dae69315772d29e0029`

// expected return value contains "hello world"
var expectedReturn = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 11, 104, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

func TestNewSimulatedBackend(t *testing.T) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	expectedBal := big.NewInt(10000000000)
	sim := NewSimulatedBackend(
		core.GenesisAlloc{
			testAddr: {Balance: expectedBal},
		}, 10000000,
	)
	defer sim.Close()

	if sim.config != params.AllEthashProtocolChanges {
		t.Errorf("expected sim config to equal params.AllEthashProtocolChanges, got %v", sim.config)
	}

	if sim.blockchain.Config() != params.AllEthashProtocolChanges {
		t.Errorf("expected sim blockchain config to equal params.AllEthashProtocolChanges, got %v", sim.config)
	}

	statedb, _ := sim.blockchain.State()
	bal := statedb.GetBalance(testAddr)
	if bal.Cmp(expectedBal) != 0 {
		t.Errorf("expected balance for test address not received. expected: %v actual: %v", expectedBal, bal)
	}
}

func TestSimulatedBackend_AdjustTime(t *testing.T) {
	sim := NewSimulatedBackend(
		core.GenesisAlloc{}, 10000000,
	)
	defer sim.Close()

	prevTime := sim.pendingBlock.Time()
	err := sim.AdjustTime(time.Second)
	if err != nil {
		t.Error(err)
	}
	newTime := sim.pendingBlock.Time()

	if newTime-prevTime != uint64(time.Second.Seconds()) {
		t.Errorf("adjusted time not equal to a second. prev: %v, new: %v", prevTime, newTime)
	}
}

func TestSimulatedBackend_BalanceAt(t *testing.T) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	expectedBal := big.NewInt(10000000000)
	sim := NewSimulatedBackend(
		core.GenesisAlloc{
			testAddr: {Balance: expectedBal},
		}, 10000000,
	)
	defer sim.Close()
	bgCtx := context.Background()

	bal, err := sim.BalanceAt(bgCtx, testAddr, nil)
	if err != nil {
		t.Error(err)
	}

	if bal.Cmp(expectedBal) != 0 {
		t.Errorf("expected balance for test address not received. expected: %v actual: %v", expectedBal, bal)
	}
}

func TestSimulatedBackend_BlockByHash(t *testing.T) {
	sim := NewSimulatedBackend(
		core.GenesisAlloc{}, 10000000,
	)
	defer sim.Close()
	bgCtx := context.Background()

	block, err := sim.BlockByNumber(bgCtx, nil)
	if err != nil {
		t.Errorf("could not get recent block: %v", err)
	}
	blockByHash, err := sim.BlockByHash(bgCtx, block.Hash())
	if err != nil {
		t.Errorf("could not get recent block: %v", err)
	}

	if block.Hash() != blockByHash.Hash() {
		t.Errorf("did not get expected block")
	}
}

func TestSimulatedBackend_BlockByNumber(t *testing.T) {
	sim := NewSimulatedBackend(
		core.GenesisAlloc{}, 10000000,
	)
	defer sim.Close()
	bgCtx := context.Background()

	block, err := sim.BlockByNumber(bgCtx, nil)
	if err != nil {
		t.Errorf("could not get recent block: %v", err)
	}
	if block.NumberU64() != 0 {
		t.Errorf("did not get most recent block, instead got block number %v", block.NumberU64())
	}

	// create one block
	sim.Commit()

	block, err = sim.BlockByNumber(bgCtx, nil)
	if err != nil {
		t.Errorf("could not get recent block: %v", err)
	}
	if block.NumberU64() != 1 {
		t.Errorf("did not get most recent block, instead got block number %v", block.NumberU64())
	}

	blockByNumber, err := sim.BlockByNumber(bgCtx, big.NewInt(1))
	if err != nil {
		t.Errorf("could not get block by number: %v", err)
	}
	if blockByNumber.Hash() != block.Hash() {
		t.Errorf("did not get the same block with height of 1 as before")
	}
}

func TestSimulatedBackend_NonceAt(t *testing.T) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)

	sim := NewSimulatedBackend(
		core.GenesisAlloc{
			testAddr: {Balance: big.NewInt(10000000000)},
		}, 10000000,
	)
	defer sim.Close()
	bgCtx := context.Background()

	nonce, err := sim.NonceAt(bgCtx, testAddr, big.NewInt(0))
	if err != nil {
		t.Errorf("could not get nonce for test addr: %v", err)
	}

	if nonce != uint64(0) {
		t.Errorf("received incorrect nonce. expected 0, got %v", nonce)
	}

	// create a signed transaction to send
	tx := types.NewTransaction(nonce, testAddr, big.NewInt(1000), params.TxGas, big.NewInt(1), nil)
	signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, testKey)
	if err != nil {
		t.Errorf("could not sign tx: %v", err)
	}

	// send tx to simulated backend
	err = sim.SendTransaction(bgCtx, signedTx)
	if err != nil {
		t.Errorf("could not add tx to pending block: %v", err)
	}
	sim.Commit()

	newNonce, err := sim.NonceAt(bgCtx, testAddr, big.NewInt(1))
	if err != nil {
		t.Errorf("could not get nonce for test addr: %v", err)
	}

	if newNonce != nonce+uint64(1) {
		t.Errorf("received incorrect nonce. expected 1, got %v", nonce)
	}
}

func TestSimulatedBackend_SendTransaction(t *testing.T) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)

	sim := NewSimulatedBackend(
		core.GenesisAlloc{
			testAddr: {Balance: big.NewInt(10000000000)},
		}, 10000000,
	)
	defer sim.Close()
	bgCtx := context.Background()

	// create a signed transaction to send
	tx := types.NewTransaction(uint64(0), testAddr, big.NewInt(1000), params.TxGas, big.NewInt(1), nil)
	signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, testKey)
	if err != nil {
		t.Errorf("could not sign tx: %v", err)
	}

	// send tx to simulated backend
	err = sim.SendTransaction(bgCtx, signedTx)
	if err != nil {
		t.Errorf("could not add tx to pending block: %v", err)
	}
	sim.Commit()

	block, err := sim.BlockByNumber(bgCtx, big.NewInt(1))
	if err != nil {
		t.Errorf("could not get block at height 1: %v", err)
	}

	if signedTx.Hash() != block.Transactions()[0].Hash() {
		t.Errorf("did not commit sent transaction. expected hash %v got hash %v", block.Transactions()[0].Hash(), signedTx.Hash())
	}
}

func TestSimulatedBackend_TransactionByHash(t *testing.T) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)

	sim := NewSimulatedBackend(
		core.GenesisAlloc{
			testAddr: {Balance: big.NewInt(10000000000)},
		}, 10000000,
	)
	defer sim.Close()
	bgCtx := context.Background()

	// create a signed transaction to send
	tx := types.NewTransaction(uint64(0), testAddr, big.NewInt(1000), params.TxGas, big.NewInt(1), nil)
	signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, testKey)
	if err != nil {
		t.Errorf("could not sign tx: %v", err)
	}

	// send tx to simulated backend
	err = sim.SendTransaction(bgCtx, signedTx)
	if err != nil {
		t.Errorf("could not add tx to pending block: %v", err)
	}

	// ensure tx is committed pending
	receivedTx, pending, err := sim.TransactionByHash(bgCtx, signedTx.Hash())
	if err != nil {
		t.Errorf("could not get transaction by hash %v: %v", signedTx.Hash(), err)
	}
	if !pending {
		t.Errorf("expected transaction to be in pending state")
	}
	if receivedTx.Hash() != signedTx.Hash() {
		t.Errorf("did not received committed transaction. expected hash %v got hash %v", signedTx.Hash(), receivedTx.Hash())
	}

	sim.Commit()

	// ensure tx is not and committed pending
	receivedTx, pending, err = sim.TransactionByHash(bgCtx, signedTx.Hash())
	if err != nil {
		t.Errorf("could not get transaction by hash %v: %v", signedTx.Hash(), err)
	}
	if pending {
		t.Errorf("expected transaction to not be in pending state")
	}
	if receivedTx.Hash() != signedTx.Hash() {
		t.Errorf("did not received committed transaction. expected hash %v got hash %v", signedTx.Hash(), receivedTx.Hash())
	}
}

func TestSimulatedBackend_EstimateGas(t *testing.T) {
	sim := NewSimulatedBackend(
		core.GenesisAlloc{}, 10000000,
	)
	defer sim.Close()
	bgCtx := context.Background()
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)

	gas, err := sim.EstimateGas(bgCtx, ethereum.CallMsg{
		From:  testAddr,
		To:    &testAddr,
		Value: big.NewInt(1000),
		Data:  []byte{},
	})
	if err != nil {
		t.Errorf("could not estimate gas: %v", err)
	}

	if gas != params.TxGas {
		t.Errorf("expected 21000 gas cost for a transaction got %v", gas)
	}
}

func TestSimulatedBackend_HeaderByHash(t *testing.T) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)

	sim := NewSimulatedBackend(
		core.GenesisAlloc{
			testAddr: {Balance: big.NewInt(10000000000)},
		}, 10000000,
	)
	defer sim.Close()
	bgCtx := context.Background()

	header, err := sim.HeaderByNumber(bgCtx, nil)
	if err != nil {
		t.Errorf("could not get recent block: %v", err)
	}
	headerByHash, err := sim.HeaderByHash(bgCtx, header.Hash())
	if err != nil {
		t.Errorf("could not get recent block: %v", err)
	}

	if header.Hash() != headerByHash.Hash() {
		t.Errorf("did not get expected block")
	}
}

func TestSimulatedBackend_HeaderByNumber(t *testing.T) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)

	sim := NewSimulatedBackend(
		core.GenesisAlloc{
			testAddr: {Balance: big.NewInt(10000000000)},
		}, 10000000,
	)
	defer sim.Close()
	bgCtx := context.Background()

	latestBlockHeader, err := sim.HeaderByNumber(bgCtx, nil)
	if err != nil {
		t.Errorf("could not get header for tip of chain: %v", err)
	}
	if latestBlockHeader == nil {
		t.Errorf("received a nil block header")
	}
	if latestBlockHeader.Number.Uint64() != uint64(0) {
		t.Errorf("expected block header number 0, instead got %v", latestBlockHeader.Number.Uint64())
	}

	sim.Commit()

	latestBlockHeader, err = sim.HeaderByNumber(bgCtx, nil)
	if err != nil {
		t.Errorf("could not get header for blockheight of 1: %v", err)
	}

	blockHeader, err := sim.HeaderByNumber(bgCtx, big.NewInt(1))
	if err != nil {
		t.Errorf("could not get header for blockheight of 1: %v", err)
	}

	if blockHeader.Hash() != latestBlockHeader.Hash() {
		t.Errorf("block header and latest block header are not the same")
	}
	if blockHeader.Number.Int64() != int64(1) {
		t.Errorf("did not get blockheader for block 1. instead got block %v", blockHeader.Number.Int64())
	}

	block, err := sim.BlockByNumber(bgCtx, big.NewInt(1))
	if err != nil {
		t.Errorf("could not get block for blockheight of 1: %v", err)
	}

	if block.Hash() != blockHeader.Hash() {
		t.Errorf("block hash and block header hash do not match. expected %v, got %v", block.Hash(), blockHeader.Hash())
	}
}

func TestSimulatedBackend_TransactionCount(t *testing.T) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)

	sim := NewSimulatedBackend(
		core.GenesisAlloc{
			testAddr: {Balance: big.NewInt(10000000000)},
		}, 10000000,
	)
	defer sim.Close()
	bgCtx := context.Background()
	currentBlock, err := sim.BlockByNumber(bgCtx, nil)
	if err != nil || currentBlock == nil {
		t.Error("could not get current block")
	}

	count, err := sim.TransactionCount(bgCtx, currentBlock.Hash())
	if err != nil {
		t.Error("could not get current block's transaction count")
	}

	if count != 0 {
		t.Errorf("expected transaction count of %v does not match actual count of %v", 0, count)
	}

	// create a signed transaction to send
	tx := types.NewTransaction(uint64(0), testAddr, big.NewInt(1000), params.TxGas, big.NewInt(1), nil)
	signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, testKey)
	if err != nil {
		t.Errorf("could not sign tx: %v", err)
	}

	// send tx to simulated backend
	err = sim.SendTransaction(bgCtx, signedTx)
	if err != nil {
		t.Errorf("could not add tx to pending block: %v", err)
	}

	sim.Commit()

	lastBlock, err := sim.BlockByNumber(bgCtx, nil)
	if err != nil {
		t.Errorf("could not get header for tip of chain: %v", err)
	}

	count, err = sim.TransactionCount(bgCtx, lastBlock.Hash())
	if err != nil {
		t.Error("could not get current block's transaction count")
	}

	if count != 1 {
		t.Errorf("expected transaction count of %v does not match actual count of %v", 1, count)
	}
}

func TestSimulatedBackend_TransactionInBlock(t *testing.T) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)

	sim := NewSimulatedBackend(
		core.GenesisAlloc{
			testAddr: {Balance: big.NewInt(10000000000)},
		}, 10000000,
	)
	defer sim.Close()
	bgCtx := context.Background()

	transaction, err := sim.TransactionInBlock(bgCtx, sim.pendingBlock.Hash(), uint(0))
	if err == nil && err != errTransactionDoesNotExist {
		t.Errorf("expected a transaction does not exist error to be received but received %v", err)
	}
	if transaction != nil {
		t.Errorf("expected transaction to be nil but received %v", transaction)
	}

	// expect pending nonce to be 0 since account has not been used
	pendingNonce, err := sim.PendingNonceAt(bgCtx, testAddr)
	if err != nil {
		t.Errorf("did not get the pending nonce: %v", err)
	}

	if pendingNonce != uint64(0) {
		t.Errorf("expected pending nonce of 0 got %v", pendingNonce)
	}

	// create a signed transaction to send
	tx := types.NewTransaction(uint64(0), testAddr, big.NewInt(1000), params.TxGas, big.NewInt(1), nil)
	signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, testKey)
	if err != nil {
		t.Errorf("could not sign tx: %v", err)
	}

	// send tx to simulated backend
	err = sim.SendTransaction(bgCtx, signedTx)
	if err != nil {
		t.Errorf("could not add tx to pending block: %v", err)
	}

	sim.Commit()

	lastBlock, err := sim.BlockByNumber(bgCtx, nil)
	if err != nil {
		t.Errorf("could not get header for tip of chain: %v", err)
	}

	transaction, err = sim.TransactionInBlock(bgCtx, lastBlock.Hash(), uint(1))
	if err == nil && err != errTransactionDoesNotExist {
		t.Errorf("expected a transaction does not exist error to be received but received %v", err)
	}
	if transaction != nil {
		t.Errorf("expected transaction to be nil but received %v", transaction)
	}

	transaction, err = sim.TransactionInBlock(bgCtx, lastBlock.Hash(), uint(0))
	if err != nil {
		t.Errorf("could not get transaction in the lastest block with hash %v: %v", lastBlock.Hash().String(), err)
	}

	if signedTx.Hash().String() != transaction.Hash().String() {
		t.Errorf("received transaction that did not match the sent transaction. expected hash %v, got hash %v", signedTx.Hash().String(), transaction.Hash().String())
	}
}

func TestSimulatedBackend_PendingNonceAt(t *testing.T) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)

	sim := NewSimulatedBackend(
		core.GenesisAlloc{
			testAddr: {Balance: big.NewInt(10000000000)},
		}, 10000000,
	)
	defer sim.Close()
	bgCtx := context.Background()

	// expect pending nonce to be 0 since account has not been used
	pendingNonce, err := sim.PendingNonceAt(bgCtx, testAddr)
	if err != nil {
		t.Errorf("did not get the pending nonce: %v", err)
	}

	if pendingNonce != uint64(0) {
		t.Errorf("expected pending nonce of 0 got %v", pendingNonce)
	}

	// create a signed transaction to send
	tx := types.NewTransaction(uint64(0), testAddr, big.NewInt(1000), params.TxGas, big.NewInt(1), nil)
	signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, testKey)
	if err != nil {
		t.Errorf("could not sign tx: %v", err)
	}

	// send tx to simulated backend
	err = sim.SendTransaction(bgCtx, signedTx)
	if err != nil {
		t.Errorf("could not add tx to pending block: %v", err)
	}

	// expect pending nonce to be 1 since account has submitted one transaction
	pendingNonce, err = sim.PendingNonceAt(bgCtx, testAddr)
	if err != nil {
		t.Errorf("did not get the pending nonce: %v", err)
	}

	if pendingNonce != uint64(1) {
		t.Errorf("expected pending nonce of 1 got %v", pendingNonce)
	}

	// make a new transaction with a nonce of 1
	tx = types.NewTransaction(uint64(1), testAddr, big.NewInt(1000), params.TxGas, big.NewInt(1), nil)
	signedTx, err = types.SignTx(tx, types.HomesteadSigner{}, testKey)
	if err != nil {
		t.Errorf("could not sign tx: %v", err)
	}
	err = sim.SendTransaction(bgCtx, signedTx)
	if err != nil {
		t.Errorf("could not send tx: %v", err)
	}

	// expect pending nonce to be 2 since account now has two transactions
	pendingNonce, err = sim.PendingNonceAt(bgCtx, testAddr)
	if err != nil {
		t.Errorf("did not get the pending nonce: %v", err)
	}

	if pendingNonce != uint64(2) {
		t.Errorf("expected pending nonce of 2 got %v", pendingNonce)
	}
}

func TestSimulatedBackend_TransactionReceipt(t *testing.T) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)

	sim := NewSimulatedBackend(
		core.GenesisAlloc{
			testAddr: {Balance: big.NewInt(10000000000)},
		}, 10000000,
	)
	defer sim.Close()
	bgCtx := context.Background()

	// create a signed transaction to send
	tx := types.NewTransaction(uint64(0), testAddr, big.NewInt(1000), params.TxGas, big.NewInt(1), nil)
	signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, testKey)
	if err != nil {
		t.Errorf("could not sign tx: %v", err)
	}

	// send tx to simulated backend
	err = sim.SendTransaction(bgCtx, signedTx)
	if err != nil {
		t.Errorf("could not add tx to pending block: %v", err)
	}
	sim.Commit()

	receipt, err := sim.TransactionReceipt(bgCtx, signedTx.Hash())
	if err != nil {
		t.Errorf("could not get transaction receipt: %v", err)
	}

	if receipt.ContractAddress != testAddr && receipt.TxHash != signedTx.Hash() {
		t.Errorf("received receipt is not correct: %v", receipt)
	}
}

func TestSimulatedBackend_SuggestGasPrice(t *testing.T) {
	sim := NewSimulatedBackend(
		core.GenesisAlloc{},
		10000000,
	)
	defer sim.Close()
	bgCtx := context.Background()
	gasPrice, err := sim.SuggestGasPrice(bgCtx)
	if err != nil {
		t.Errorf("could not get gas price: %v", err)
	}
	if gasPrice.Uint64() != uint64(1) {
		t.Errorf("gas price was not expected value of 1. actual: %v", gasPrice.Uint64())
	}
}

func TestSimulatedBackend_PendingCodeAt(t *testing.T) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	sim := NewSimulatedBackend(
		core.GenesisAlloc{
			testAddr: {Balance: big.NewInt(10000000000)},
		},
		10000000,
	)
	defer sim.Close()
	bgCtx := context.Background()
	code, err := sim.CodeAt(bgCtx, testAddr, nil)
	if err != nil {
		t.Errorf("could not get code at test addr: %v", err)
	}
	if len(code) != 0 {
		t.Errorf("got code for account that does not have contract code")
	}

	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		t.Errorf("could not get code at test addr: %v", err)
	}
	auth := bind.NewKeyedTransactor(testKey)
	contractAddr, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(abiBin), sim)
	if err != nil {
		t.Errorf("could not deploy contract: %v tx: %v contract: %v", err, tx, contract)
	}

	code, err = sim.PendingCodeAt(bgCtx, contractAddr)
	if err != nil {
		t.Errorf("could not get code at test addr: %v", err)
	}
	if len(code) == 0 {
		t.Errorf("did not get code for account that has contract code")
	}
	// ensure code received equals code deployed
	if !bytes.Equal(code, common.FromHex(deployedCode)) {
		t.Errorf("code received did not match expected deployed code:\n expected %v\n actual %v", common.FromHex(deployedCode), code)
	}
}

func TestSimulatedBackend_CodeAt(t *testing.T) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	sim := NewSimulatedBackend(
		core.GenesisAlloc{
			testAddr: {Balance: big.NewInt(10000000000)},
		},
		10000000,
	)
	defer sim.Close()
	bgCtx := context.Background()
	code, err := sim.CodeAt(bgCtx, testAddr, nil)
	if err != nil {
		t.Errorf("could not get code at test addr: %v", err)
	}
	if len(code) != 0 {
		t.Errorf("got code for account that does not have contract code")
	}

	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		t.Errorf("could not get code at test addr: %v", err)
	}
	auth := bind.NewKeyedTransactor(testKey)
	contractAddr, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(abiBin), sim)
	if err != nil {
		t.Errorf("could not deploy contract: %v tx: %v contract: %v", err, tx, contract)
	}

	sim.Commit()
	code, err = sim.CodeAt(bgCtx, contractAddr, nil)
	if err != nil {
		t.Errorf("could not get code at test addr: %v", err)
	}
	if len(code) == 0 {
		t.Errorf("did not get code for account that has contract code")
	}
	// ensure code received equals code deployed
	if !bytes.Equal(code, common.FromHex(deployedCode)) {
		t.Errorf("code received did not match expected deployed code:\n expected %v\n actual %v", common.FromHex(deployedCode), code)
	}
}

// When receive("X") is called with sender 0x00... and value 1, it produces this tx receipt:
//   receipt{status=1 cgas=23949 bloom=00000000004000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000040200000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000 logs=[log: b6818c8064f645cd82d99b59a1a267d6d61117ef [75fd880d39c1daf53b6547ab6cb59451fc6452d27caa90e5b6649dd8293b9eed] 000000000000000000000000376c47978271565f56deb45495afa69e59c16ab200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000158 9ae378b6d4409eada347a5dc0c180f186cb62dc68fcc0f043425eb917335aa28 0 95d429d309bb9d753954195fe2d69bd140b4ae731b9b5b605c34323de162cf00 0]}
func TestSimulatedBackend_PendingAndCallContract(t *testing.T) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	sim := NewSimulatedBackend(
		core.GenesisAlloc{
			testAddr: {Balance: big.NewInt(10000000000)},
		},
		10000000,
	)
	defer sim.Close()
	bgCtx := context.Background()

	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		t.Errorf("could not get code at test addr: %v", err)
	}
	contractAuth := bind.NewKeyedTransactor(testKey)
	addr, _, _, err := bind.DeployContract(contractAuth, parsed, common.FromHex(abiBin), sim)
	if err != nil {
		t.Errorf("could not deploy contract: %v", err)
	}

	input, err := parsed.Pack("receive", []byte("X"))
	if err != nil {
		t.Errorf("could pack receive function on contract: %v", err)
	}

	// make sure you can call the contract in pending state
	res, err := sim.PendingCallContract(bgCtx, ethereum.CallMsg{
		From: testAddr,
		To:   &addr,
		Data: input,
	})
	if err != nil {
		t.Errorf("could not call receive method on contract: %v", err)
	}
	if len(res) == 0 {
		t.Errorf("result of contract call was empty: %v", res)
	}

	// while comparing against the byte array is more exact, also compare against the human readable string for readability
	if !bytes.Equal(res, expectedReturn) || !strings.Contains(string(res), "hello world") {
		t.Errorf("response from calling contract was expected to be 'hello world' instead received %v", string(res))
	}

	sim.Commit()

	// make sure you can call the contract
	res, err = sim.CallContract(bgCtx, ethereum.CallMsg{
		From: testAddr,
		To:   &addr,
		Data: input,
	}, nil)
	if err != nil {
		t.Errorf("could not call receive method on contract: %v", err)
	}
	if len(res) == 0 {
		t.Errorf("result of contract call was empty: %v", res)
	}

	if !bytes.Equal(res, expectedReturn) || !strings.Contains(string(res), "hello world") {
		t.Errorf("response from calling contract was expected to be 'hello world' instead received %v", string(res))
	}
}
