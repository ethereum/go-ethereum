// attempt to test Process, and how 7560 transaction affect normal TXs
package rip7560

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/tests"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

/**
Test that "Process" of 7560 transactions doesn't alter legacy transaction processing.
the idea:
1. Run "Process" with a set of transactions L1 [AA1..AAn] L2
2. Run "Process" just with the lagacy transactions L1,L2
3. if AA transactions revert validation - make sure the legacy processing is intact.
4. if AA transactions are executed, make sure the needed state changes of the legacy transactions is intact
*/

const addr1 = "f39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
const privKey1 = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

const addr2 = "70997970C51812dc3A010C7d01b50e0d17dc79C8"
const privKey2 = "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"

// initial minimal test that a valid AATX can be processed in a block
func TestProcess1(t *testing.T) {

	Sender := common.HexToAddress(DEFAULT_SENDER)
	err := runProcess(newTestContextBuilder(t).
		withAccount(addr1, 100000000000000).
		withCode(DEFAULT_SENDER, createAccountCode(), 1000000000000000000).
		build(), []*types.Rip7560AccountAbstractionTx{
		{
			Sender:        &Sender,
			ValidationGas: uint64(1000000000),
			GasFeeCap:     big.NewInt(1000000000),
			Data:          []byte{1, 2, 3},
		},
	})
	if err != nil {
		panic(err)
	}
}

// run a set of AA transactions, with a legacy TXs before and after.
func runProcess(t *testContext, aatxs []*types.Rip7560AccountAbstractionTx) error {
	var db ethdb.Database = rawdb.NewMemoryDatabase()
	var state = tests.MakePreState(db, t.genesisAlloc, false, rawdb.HashScheme)
	defer state.Close()

	cacheConfig := &core.CacheConfig{}
	chainOverrides := core.ChainOverrides{}
	engine := beacon.New(ethash.NewFaker())
	lookupLimit := uint64(0)
	blockchain, err := core.NewBlockChain(db, cacheConfig, t.genesis, &chainOverrides, engine,
		vm.Config{}, shouldPreserve, &lookupLimit)
	if err != nil {
		t.t.Fatalf("NewBlockChain failed: %v", err)
	}

	signer := types.MakeSigner(blockchain.Config(), new(big.Int), 0)
	key1, _ := crypto.HexToECDSA(privKey1)
	if crypto.PubkeyToAddress(key1.PublicKey) != common.HexToAddress(addr1) {
		t.t.Fatalf("sanity: addr1 doesn't match privKey1: should be %s", crypto.PubkeyToAddress(key1.PublicKey))
	}
	//addr1 := crypto.PubkeyToAddress(key1.PublicKey)

	key2, _ := crypto.HexToECDSA(privKey2)
	addr2 := crypto.PubkeyToAddress(key2.PublicKey)

	tx1, _ := types.SignTx(types.NewTx(&types.DynamicFeeTx{
		Nonce:     0,
		GasFeeCap: big.NewInt(1000000000),
		Value:     big.NewInt(1),
		Gas:       30000,
		To:        &addr2,
	}), signer, key1)

	tx3, _ := types.SignTx(types.NewTx(&types.DynamicFeeTx{
		Nonce:     1,
		GasFeeCap: big.NewInt(1000000000),
		Value:     big.NewInt(2),
		Gas:       30000,
		To:        &addr2,
	}), signer, key1)

	txs := []*types.Transaction{tx1}
	for _, aatx := range aatxs {
		txs = append(txs, types.NewTx(aatx))
	}
	txs = append(txs, tx3)

	body := types.Body{Transactions: txs}
	b := types.NewBlock(blockchain.CurrentBlock(), &body, nil, trie.NewStackTrie(nil))
	_, _, _, err = blockchain.Processor().Process(b, state.StateDB, vm.Config{})
	if err != nil {
		return err
	}
	assert.Equal(t.t, "0x3", state.StateDB.GetBalance(addr2).Hex(), "failed to process pre/post legacy transactions")
	return nil
}

func shouldPreserve(*types.Header) bool {
	return false
}
