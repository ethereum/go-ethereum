package engine_v1_tests

import (
	"math/big"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

// Snapshot try to read before blockchain is written
func TestRaceConditionOnBlockchainReadAndWrite(t *testing.T) {

	blockchain, backend, parentBlock, signer, signFn := PrepareXDCTestBlockChain(t, GAP-1, params.TestXDPoSMockChainConfig)

	state, err := blockchain.State()
	if err != nil {
		t.Fatalf("Failed while trying to get blockchain state")
	}
	t.Logf("Account %v have balance of: %v", acc1Addr.String(), state.GetBalance(acc1Addr))
	// Check initial signer
	signers, err := GetSnapshotSigner(blockchain, blockchain.CurrentBlock().Header())
	if err != nil {
		t.Fatal(err)
	}
	if signers[acc3Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("acc3Addr should sit in the signer list")
	}

	// Insert first Block 450 A
	t.Logf("Inserting block with propose and transfer at 450 A...")
	blockCoinbaseA := "0xaaa0000000000000000000000000000000000450"
	tx, err := voteTX(58117, 0, acc1Addr.String())
	if err != nil {
		t.Fatal(err)
	}

	transferTransaction := transferTx(t, acc1Addr, 999)

	merkleRoot := "ea465415b60d88429f181fec9fae67c0f19cbf5a4fa10971d96d4faa57d96ffa"

	header := &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(450)),
		ParentHash: parentBlock.Hash(),
		Coinbase:   common.HexToAddress(blockCoinbaseA),
	}

	blockA, err := createBlockFromHeader(blockchain, header, []*types.Transaction{tx, transferTransaction}, signer, signFn, blockchain.Config())
	if err != nil {
		t.Fatal(err)
	}
	err = blockchain.InsertBlock(blockA)
	assert.Nil(t, err)
	state, err = blockchain.State()
	if err != nil {
		t.Fatalf("Failed while trying to get blockchain state")
	}
	t.Log("After transfer transaction at block 450 A, Account 1 have balance of: ", state.GetBalance(acc1Addr))

	if state.GetBalance(acc1Addr).Cmp(new(big.Int).SetUint64(10000000999)) != 0 {
		t.Fatalf("account 1 should have 10000000999 in balance")
	}

	signers, err = GetSnapshotSigner(blockchain, blockA.Header())
	if err != nil {
		t.Fatal(err)
	}
	if signers[acc1Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 1 should sit in the signer list")
	}

	// Insert forked Block 450 B
	t.Logf("Inserting block with propose at 450 B...")

	blockCoinBase450B := "0xbbb0000000000000000000000000000000000450"
	tx, err = voteTX(37117, 0, acc2Addr.String())
	if err != nil {
		t.Fatal(err)
	}

	transferTransaction = transferTx(t, acc1Addr, 888)

	merkleRoot = "184edaddeafc2404248f896ae46be503ae68949896c8eb6b6ad43695581e5022"

	header = &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(450)),
		ParentHash: parentBlock.Hash(),
		Coinbase:   common.HexToAddress(blockCoinBase450B),
		Difficulty: big.NewInt(2),
	}

	block450B, err := createBlockFromHeader(blockchain, header, []*types.Transaction{tx, transferTransaction}, signer, signFn, blockchain.Config())
	if err != nil {
		t.Fatal(err)
	}
	err = blockchain.InsertBlock(block450B)
	assert.Nil(t, err)
	if blockchain.CurrentHeader().Hash() != block450B.Hash() {
		t.Fatalf("the block with higher difficulty should be current header")
	}
	state, err = blockchain.State()
	if err != nil {
		t.Fatalf("Failed while trying to get blockchain state")
	}
	if state.GetBalance(acc1Addr).Cmp(new(big.Int).SetUint64(10000000888)) != 0 {
		t.Fatalf("account 1 should have 10000000888 in balance as the block replace previous head block at number 450")
	}

	signers, err = GetSnapshotSigner(blockchain, block450B.Header())
	if err != nil {
		t.Fatal(err)
	}
	// Should run the `updateM1` for forked chain as it's now the mainchain, hence account2 should exist
	if signers[acc2Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 2 should sit in the signer list")
	}

	//Insert block 451 parent is 451 B
	t.Logf("Inserting block with propose at 451 B...")

	blockCoinBase451B := "0xbbb0000000000000000000000000000000000451"
	merkleRoot = "184edaddeafc2404248f896ae46be503ae68949896c8eb6b6ad43695581e5022"
	header = &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(451)),
		ParentHash: block450B.Hash(),
		Coinbase:   common.HexToAddress(blockCoinBase451B),
		Difficulty: big.NewInt(3),
	}
	block451B, err := createBlockFromHeader(blockchain, header, nil, signer, signFn, blockchain.Config())
	if err != nil {
		t.Fatal(err)
	}
	err = blockchain.InsertBlock(block451B)
	assert.Nil(t, err)

	signers, err = GetSnapshotSigner(blockchain, block450B.Header())
	if err != nil {
		t.Fatal(err)
	}
	if signers[acc2Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 2 should sit in the signer list")
	}

	signers, err = GetSnapshotSigner(blockchain, block451B.Header())
	if err != nil {
		t.Fatal(err)
	}
	if signers[acc2Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 2 should sit in the signer list")
	}

	signers, err = GetSnapshotSigner(blockchain, blockchain.CurrentBlock().Header())
	if err != nil {
		t.Fatal(err)
	}
	if signers[acc2Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("acc2Addr should sit in the signer list")
	}
	state, err = blockchain.State()
	if err != nil {
		t.Fatalf("Failed while trying to get blockchain state")
	}
	t.Log("After transfer transaction at block 450 B and the B fork has been merged into main chain, Account 1 have balance of: ", state.GetBalance(acc1Addr))

	if state.GetBalance(acc1Addr).Cmp(new(big.Int).SetUint64(10000000888)) != 0 {
		t.Fatalf("account 1 should have 10000000888 in balance")
	}
}
