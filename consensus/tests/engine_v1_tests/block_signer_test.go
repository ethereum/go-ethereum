package engine_v1_tests

import (
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

// Should NOT update signerList if not on the gap block
func TestNotUpdateSignerListIfNotOnGapBlock(t *testing.T) {
	blockchain, backend, parentBlock, signer, signFn := PrepareXDCTestBlockChain(t, 400, params.TestXDPoSMockChainConfig)
	parentSigners, err := GetSnapshotSigner(blockchain, parentBlock.Header())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Inserting block with propose at 401")
	blockCoinbaseA := "0xaaa0000000000000000000000000000000000401"
	tx, err := voteTX(37117, 0, acc1Addr.String())
	if err != nil {
		t.Fatal(err)
	}

	//Get from block validator error message
	merkleRoot := "46234e9cd7e85a267f7f0435b15256a794a2f6d65cc98cdbd21dcd10a01d9772"
	header := &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(401)),
		ParentHash: parentBlock.Hash(),
		Coinbase:   common.HexToAddress(blockCoinbaseA),
	}
	blockA, err := createBlockFromHeader(blockchain, header, []*types.Transaction{tx}, signer, signFn, blockchain.Config())
	if err != nil {
		t.Fatal(err)
	}
	err = blockchain.InsertBlock(blockA)
	assert.Nil(t, err)
	signers, err := GetSnapshotSigner(blockchain, blockA.Header())
	if err != nil {
		t.Fatal(err)
	}

	if signers[acc1Addr.Hex()] == true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 1 should NOT sit in the signer list")
	}
	eq := reflect.DeepEqual(parentSigners, signers)
	if eq {
		t.Logf("Signers unchanged")
	} else {
		t.Fatalf("Singers should not be changed!")
	}
}

// Should call updateM1 at the gap block, and have the same snapshot values as the parent block if no SM transaction is involved
func TestNotChangeSingerListIfNothingProposedOrVoted(t *testing.T) {
	blockchain, _, parentBlock, signer, signFn := PrepareXDCTestBlockChain(t, GAP-1, params.TestXDPoSMockChainConfig)
	// Insert block 450
	blockCoinBase := fmt.Sprintf("0x111000000000000000000000000000000%03d", 450)
	merkleRoot := "35999dded35e8db12de7e6c1471eb9670c162eec616ecebbaf4fddd4676fb930"
	header := &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(450)),
		ParentHash: parentBlock.Hash(),
		Coinbase:   common.HexToAddress(blockCoinBase),
	}
	block, err := createBlockFromHeader(blockchain, header, nil, signer, signFn, blockchain.Config())
	if err != nil {
		t.Fatal(err)
	}
	err = blockchain.InsertBlock(block)
	assert.Nil(t, err)
	parentSigners, err := GetSnapshotSigner(blockchain, parentBlock.Header())
	if err != nil {
		t.Fatal(err)
	}
	signers, err := GetSnapshotSigner(blockchain, block.Header())
	if err != nil {
		t.Fatal(err)
	}

	eq := reflect.DeepEqual(parentSigners, signers)
	if eq {
		t.Logf("Signers unchanged")
	} else {
		t.Fatalf("Singers should not be changed!")
	}
}

// Should call updateM1 at gap block, and update the snapshot if there are SM transactions involved
func TestUpdateSignerListIfVotedBeforeGap(t *testing.T) {

	blockchain, backend, parentBlock, signer, signFn := PrepareXDCTestBlockChain(t, GAP-2, params.TestXDPoSMockChainConfig)
	// Insert first Block 449
	t.Logf("Inserting block with propose at 449...")
	blockCoinbaseA := "0xaaa0000000000000000000000000000000000449"
	tx, err := voteTX(37117, 0, acc1Addr.String())
	if err != nil {
		t.Fatal(err)
	}

	//Get from block validator error message
	merkleRoot := "46234e9cd7e85a267f7f0435b15256a794a2f6d65cc98cdbd21dcd10a01d9772"
	header := &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(449)),
		ParentHash: parentBlock.Hash(),
		Coinbase:   common.HexToAddress(blockCoinbaseA),
	}
	block449, err := createBlockFromHeader(blockchain, header, []*types.Transaction{tx}, signer, signFn, blockchain.Config())
	if err != nil {
		t.Fatal(err)
	}
	err = blockchain.InsertBlock(block449)
	assert.Nil(t, err)
	parentBlock = block449

	signers, err := GetSnapshotSigner(blockchain, block449.Header())
	if err != nil {
		t.Fatal(err)
	}
	// At block 449, we should not update signerList. we need to update it till block 450 gap block.
	// Acc3 is the default account that is on the signerList
	if signers[acc3Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 3 should sit in the signer list")
	}
	if signers[acc1Addr.Hex()] == true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 1 should NOT sit in the signer list")
	}

	// Now, let's mine another block to trigger the GAP block signerList update
	block450CoinbaseAddress := "0xaaa0000000000000000000000000000000000450"
	merkleRoot = "46234e9cd7e85a267f7f0435b15256a794a2f6d65cc98cdbd21dcd10a01d9772"
	header = &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(450)),
		ParentHash: parentBlock.Hash(),
		Coinbase:   common.HexToAddress(block450CoinbaseAddress),
	}
	block450, err := createBlockFromHeader(blockchain, header, nil, signer, signFn, blockchain.Config())
	if err != nil {
		t.Fatal(err)
	}
	err = blockchain.InsertBlock(block450)
	assert.Nil(t, err)
	signers, err = GetSnapshotSigner(blockchain, block450.Header())
	if err != nil {
		t.Fatalf("Failed while trying to get signers")
	}
	assert.Equal(t, common.MaxMasternodes, len(signers))
	// Now, we voted acc 1 to be in the signerList, which will kick out acc3 because it has less funds
	if signers[acc3Addr.Hex()] == true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 3 should NOT sit in the signer list")
	}
	if signers[acc1Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 1 should sit in the signer list")
	}
}

// Should call updateM1 before gap block, and update the snapshot if there are SM transactions involved
func TestCallUpdateM1WithSmartContractTranscation(t *testing.T) {

	blockchain, backend, currentBlock, signer, signFn := PrepareXDCTestBlockChain(t, GAP-1, params.TestXDPoSMockChainConfig)
	// Insert first Block 450 A
	t.Logf("Inserting block with propose at 450 A...")
	blockCoinbaseA := "0xaaa0000000000000000000000000000000000450"
	tx, err := voteTX(37117, 0, acc1Addr.String())
	if err != nil {
		t.Fatal(err)
	}

	//Get from block validator error message
	merkleRoot := "46234e9cd7e85a267f7f0435b15256a794a2f6d65cc98cdbd21dcd10a01d9772"
	header := &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(450)),
		ParentHash: currentBlock.Hash(),
		Coinbase:   common.HexToAddress(blockCoinbaseA),
	}
	blockA, err := createBlockFromHeader(blockchain, header, []*types.Transaction{tx}, signer, signFn, blockchain.Config())
	if err != nil {
		t.Fatal(err)
	}
	err = blockchain.InsertBlock(blockA)
	assert.Nil(t, err)
	signers, err := GetSnapshotSigner(blockchain, blockA.Header())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, common.MaxMasternodes, len(signers))
	if signers[acc1Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 1 should sit in the signer list")
	}
}

// Should call updateM1 and update snapshot when a forked block(at gap block number) is inserted back into main chain (Edge case)
func TestCallUpdateM1WhenForkedBlockBackToMainChain(t *testing.T) {

	blockchain, backend, currentBlock, signer, signFn := PrepareXDCTestBlockChain(t, GAP-1, params.TestXDPoSMockChainConfig)
	// Check initial signer, by default, acc3 is in the signerList
	signers, err := GetSnapshotSigner(blockchain, blockchain.CurrentBlock().Header())
	if err != nil {
		t.Fatal(err)
	}
	if signers[acc3Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("acc3 should sit in the signer list")
	}
	if (signers[acc1Addr.Hex()] == true) || (signers[acc2Addr.Hex()] == true) {
		debugMessage(backend, signers, t)
		t.Fatalf("acc1,2should NOT sit in the signer list")
	}

	// Insert first Block 450 A
	t.Logf("Inserting block with propose at 450 A...")
	blockCoinbaseA := "0xaaa0000000000000000000000000000000000450"
	tx, err := voteTX(37117, 0, acc1Addr.String())
	if err != nil {
		t.Fatal(err)
	}

	merkleRoot := "46234e9cd7e85a267f7f0435b15256a794a2f6d65cc98cdbd21dcd10a01d9772"
	header := &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(450)),
		ParentHash: currentBlock.Hash(),
		Coinbase:   common.HexToAddress(blockCoinbaseA),
	}
	blockA, err := createBlockFromHeader(blockchain, header, []*types.Transaction{tx}, signer, signFn, blockchain.Config())
	if err != nil {
		t.Fatal(err)
	}
	err = blockchain.InsertBlock(blockA)
	assert.Nil(t, err)
	signers, err = GetSnapshotSigner(blockchain, blockA.Header())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, common.MaxMasternodes, len(signers))
	if signers[acc1Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 1 should sit in the signer list")
	}
	if signers[acc2Addr.Hex()] == true || signers[acc3Addr.Hex()] == true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 2,3 should NOT sit in the signer list")
	}

	// Insert forked Block 450 B
	t.Logf("Inserting block with propose for acc2 at 450 B...")

	blockCoinBase450B := "0xbbb0000000000000000000000000000000000450"
	tx, err = voteTX(37117, 0, acc2Addr.String())
	if err != nil {
		t.Fatal(err)
	}

	merkleRoot = "068dfa09d7b4093441c0cc4d9807a71bc586f6101c072d939b214c21cd136eb3"
	header = &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(450)),
		ParentHash: currentBlock.Hash(),
		Coinbase:   common.HexToAddress(blockCoinBase450B),
	}
	block450B, err := createBlockFromHeader(blockchain, header, []*types.Transaction{tx}, signer, signFn, blockchain.Config())
	if err != nil {
		t.Fatal(err)
	}
	err = blockchain.InsertBlock(block450B)
	assert.Nil(t, err)
	signers, err = GetSnapshotSigner(blockchain, block450B.Header())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, common.MaxMasternodes, len(signers))
	// Should not run the `updateM1` for forked chain, hence account3 still exit
	if signers[acc3Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 3 should sit in the signer list as previos block result")
	}
	if (signers[acc1Addr.Hex()] == true) || (signers[acc2Addr.Hex()] == true) {
		debugMessage(backend, signers, t)
		t.Fatalf("acc1,2should NOT sit in the signer list")
	}

	//Insert block 451 parent is 451 B
	t.Logf("Inserting block with propose at 451 B...")

	blockCoinBase451B := "0xbbb0000000000000000000000000000000000451"
	merkleRoot = "068dfa09d7b4093441c0cc4d9807a71bc586f6101c072d939b214c21cd136eb3"
	header = &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(451)),
		ParentHash: block450B.Hash(),
		Coinbase:   common.HexToAddress(blockCoinBase451B),
	}
	block451B, err := createBlockFromHeader(blockchain, header, nil, signer, signFn, blockchain.Config())
	assert.Nil(t, err)
	err = blockchain.InsertBlock(block451B)
	assert.Nil(t, err)
	if err != nil {
		t.Fatal(err)
	}

	signers, err = GetSnapshotSigner(blockchain, block450B.Header())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, common.MaxMasternodes, len(signers))
	if signers[acc2Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 2 should sit in the signer list")
	}
	if signers[acc1Addr.Hex()] == true || signers[acc3Addr.Hex()] == true {
		debugMessage(backend, signers, t)
		t.Fatalf("acc1,3should NOT sit in the signer list")
	}

	signers, err = GetSnapshotSigner(blockchain, block451B.Header())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, common.MaxMasternodes, len(signers))
	if signers[acc2Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 2 should sit in the signer list")
	}
	if signers[acc1Addr.Hex()] == true || signers[acc3Addr.Hex()] == true {
		debugMessage(backend, signers, t)
		t.Fatalf("acc1,3should NOT sit in the signer list")
	}

	signers, err = GetSnapshotSigner(blockchain, blockchain.CurrentBlock().Header())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, common.MaxMasternodes, len(signers))
	if signers[acc2Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("acc2Addr should sit in the signer list")
	}
	if signers[acc1Addr.Hex()] == true || signers[acc3Addr.Hex()] == true {
		debugMessage(backend, signers, t)
		t.Fatalf("acc1,3should NOT sit in the signer list")
	}
}

func TestStatesShouldBeUpdatedWhenForkedBlockBecameMainChainAtGapBlock(t *testing.T) {

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
	assert.Equal(t, common.MaxMasternodes, len(signers))
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
	}
	block450B, err := createBlockFromHeader(blockchain, header, []*types.Transaction{tx, transferTransaction}, signer, signFn, blockchain.Config())
	if err != nil {
		t.Fatal(err)
	}
	err = blockchain.InsertBlock(block450B)
	assert.Nil(t, err)
	state, err = blockchain.State()
	if err != nil {
		t.Fatalf("Failed while trying to get blockchain state")
	}
	if state.GetBalance(acc1Addr).Cmp(new(big.Int).SetUint64(10000000999)) != 0 {
		t.Fatalf("account 1 should have 10000000999 in balance as the block is forked, not on the main chain")
	}

	signers, err = GetSnapshotSigner(blockchain, block450B.Header())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, common.MaxMasternodes, len(signers))
	// Should not run the `updateM1` for forked chain, hence account3 still exit
	if signers[acc3Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 3 should sit in the signer list as previos block result")
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
	assert.Equal(t, common.MaxMasternodes, len(signers))
	if signers[acc2Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 2 should sit in the signer list")
	}

	signers, err = GetSnapshotSigner(blockchain, block451B.Header())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, common.MaxMasternodes, len(signers))
	if signers[acc2Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 2 should sit in the signer list")
	}

	signers, err = GetSnapshotSigner(blockchain, blockchain.CurrentBlock().Header())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, common.MaxMasternodes, len(signers))
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

func TestVoteShouldNotBeAffectedByFork(t *testing.T) {
	blockchain, backend, parentBlock, signer, signFn := PrepareXDCTestBlockChain(t, GAP-1, params.TestXDPoSMockChainConfig)
	// Check initial signer, by default, acc3 is in the signerList
	signers, err := GetSnapshotSigner(blockchain, blockchain.CurrentBlock().Header())
	if err != nil {
		t.Fatal(err)
	}
	if signers[acc3Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("acc3 should sit in the signer list")
	}
	if (signers[acc1Addr.Hex()] == true) || (signers[acc2Addr.Hex()] == true) {
		debugMessage(backend, signers, t)
		t.Fatalf("acc1,2should NOT sit in the signer list")
	}

	// Insert normal blocks 450 A
	blockCoinBase450A := "0xaaa0000000000000000000000000000000000450"
	merkleRoot := "35999dded35e8db12de7e6c1471eb9670c162eec616ecebbaf4fddd4676fb930"
	header := &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(450)),
		ParentHash: parentBlock.Hash(),
		Coinbase:   common.HexToAddress(blockCoinBase450A),
	}
	block450A, err := createBlockFromHeader(blockchain, header, nil, signer, signFn, blockchain.Config())
	if err != nil {
		t.Fatal(err)
	}
	err = blockchain.InsertBlock(block450A)
	assert.Nil(t, err)

	// Insert 451 A with vote
	blockCoinbase451A := "0xaaa0000000000000000000000000000000000451"
	tx, err := voteTX(37117, 0, acc1Addr.String())
	if err != nil {
		t.Fatal(err)
	}

	merkleRoot = "46234e9cd7e85a267f7f0435b15256a794a2f6d65cc98cdbd21dcd10a01d9772"
	header = &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(451)),
		ParentHash: block450A.Hash(),
		Coinbase:   common.HexToAddress(blockCoinbase451A),
	}
	block451A, err := createBlockFromHeader(blockchain, header, []*types.Transaction{tx}, signer, signFn, blockchain.Config())
	if err != nil {
		t.Fatal(err)
	}
	err = blockchain.InsertBlock(block451A)
	assert.Nil(t, err)

	// SignerList should be unchanged as the vote happen after GAP block
	signers, err = GetSnapshotSigner(blockchain, block451A.Header())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, common.MaxMasternodes, len(signers))
	if signers[acc1Addr.Hex()] == true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 1 should NOT sit in the signer list")
	}
	if signers[acc3Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 3 should sit in the signer list")
	}

	// Now, we going to inject normal blocks of 450B, 451B and 452B. Because it's the longest, it will become the mainchain
	// Insert forked Block 450 B
	blockCoinBase450B := "0xbbb0000000000000000000000000000000000450"
	merkleRoot = "35999dded35e8db12de7e6c1471eb9670c162eec616ecebbaf4fddd4676fb930"
	header = &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(450)),
		ParentHash: parentBlock.Hash(),
		Coinbase:   common.HexToAddress(blockCoinBase450B),
	}
	block450B, err := createBlockFromHeader(blockchain, header, nil, signer, signFn, blockchain.Config())
	if err != nil {
		t.Fatal(err)
	}
	err = blockchain.InsertBlock(block450B)
	assert.Nil(t, err)

	blockCoinBase451B := "0xbbb0000000000000000000000000000000000451"
	merkleRoot = "35999dded35e8db12de7e6c1471eb9670c162eec616ecebbaf4fddd4676fb930"
	header = &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(451)),
		ParentHash: block450B.Hash(),
		Coinbase:   common.HexToAddress(blockCoinBase451B),
	}
	block451B, err := createBlockFromHeader(blockchain, header, nil, signer, signFn, blockchain.Config())
	if err != nil {
		t.Fatal(err)
	}
	err = blockchain.InsertBlock(block451B)
	assert.Nil(t, err)

	blockCoinBase452B := "0xbbb0000000000000000000000000000000000452"
	merkleRoot = "35999dded35e8db12de7e6c1471eb9670c162eec616ecebbaf4fddd4676fb930"
	header = &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(452)),
		ParentHash: block451B.Hash(),
		Coinbase:   common.HexToAddress(blockCoinBase452B),
	}
	block452B, err := createBlockFromHeader(blockchain, header, nil, signer, signFn, blockchain.Config())
	if err != nil {
		t.Fatal(err)
	}
	err = blockchain.InsertBlock(block452B)
	assert.Nil(t, err)
	signers, err = GetSnapshotSigner(blockchain, block452B.Header())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, common.MaxMasternodes, len(signers))

	// Should run the `updateM1` for forked chain, but it should not be affected by the voted block 451A which is not on the mainchain anymore
	if signers[acc3Addr.Hex()] != true {
		debugMessage(backend, signers, t)
		t.Fatalf("account 3 should sit in the signer list as previos block result")
	}
}
