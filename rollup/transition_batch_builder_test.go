package rollup

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

var (
	timeoutDuration = time.Millisecond * 100

	cliqueChainConfig *params.ChainConfig

	// Test accounts
	testBankKey, _ = crypto.GenerateKey()

	testUserKey, _  = crypto.GenerateKey()
	testUserAddress = crypto.PubkeyToAddress(testUserKey.PublicKey)
	testRollupTxId  = hexutil.Uint64(2)
)

func init() {
	cliqueChainConfig = params.AllCliqueProtocolChanges
	cliqueChainConfig.Clique = &params.CliqueConfig{
		Period: 10,
		Epoch:  30000,
	}
}

type TestBlockStore struct {
	blocks map[uint64]*types.Block
}

func newTestBlockStore(blocks []*types.Block) *TestBlockStore {
	store := &TestBlockStore{blocks: make(map[uint64]*types.Block, len(blocks))}
	for _, block := range blocks {
		store.blocks[block.NumberU64()] = block
	}

	return store
}

func (t *TestBlockStore) GetBlockByNumber(number uint64) *types.Block {
	if block, found := t.blocks[number]; found {
		return block
	}
	return nil
}

type TestTransitionBatchSubmitter struct {
	submittedTransitions []*TransitionBatch
	submitCh             chan *TransitionBatch
}

func newTestBlockSubmitter(submittedBlocks []*TransitionBatch, submitCh chan *TransitionBatch) *TestTransitionBatchSubmitter {
	return &TestTransitionBatchSubmitter{
		submittedTransitions: submittedBlocks,
		submitCh:             submitCh,
	}
}

func (t *TestTransitionBatchSubmitter) submit(block *TransitionBatch) error {
	t.submittedTransitions = append(t.submittedTransitions, block)
	t.submitCh <- block
	return nil
}

func createBlocks(number int, startIndex int, withTx bool) types.Blocks {
	blocks := make(types.Blocks, number)
	for i := 0; i < number; i++ {
		header := &types.Header{Number: big.NewInt(int64(i + startIndex))}
		txs := make(types.Transactions, 0)
		if withTx {
			tx, _ := types.SignTx(types.NewTransaction(uint64(i), testUserAddress, big.NewInt(1), params.TxGas, big.NewInt(0), nil, &testUserAddress, &testRollupTxId), types.HomesteadSigner{}, testBankKey)
			txs = append(txs, tx)
		}
		block := types.NewBlock(header, txs, make([]*types.Header, 0), make([]*types.Receipt, 0))
		blocks[i] = block
	}
	return blocks
}

func assertTransitionFromBlock(t *testing.T, transition *Transition, block *types.Block) {
	if transition.postState != block.Root() {
		t.Fatal("expecting transitionBatch postState to equal block root", "postState", transition.postState, "block.Hash()", block.Root())
	}
	if transition.transaction.Hash() != block.Transactions()[0].Hash() {
		t.Fatal("expecting transitionBatch tx hash to equal block tx hash", "transitionBatch tx", transition.transaction.Hash(), "block tx", block.Transactions()[0].Hash())
	}
}

func newTestTransitionBatchBuilder(blockStore *TestBlockStore, batchSubmitter *TestTransitionBatchSubmitter, lastProcessedBlock uint64, maxBlockTime time.Duration, maxBlockGas uint64, maxBlockTransactions int) (*TransitionBatchBuilder, error) {
	db := rawdb.NewMemoryDatabase()

	if lastProcessedBlock != 0 {
		if err := db.Put(LastProcessedDBKey, SerializeBlockNumber(lastProcessedBlock)); err != nil {
			return nil, err
		}
	}

	return NewTransitionBatchBuilder(db, blockStore, batchSubmitter, maxBlockTime, maxBlockGas, maxBlockTransactions)
}

func getSubmitChBlockStoreAndSubmitter() (chan *TransitionBatch, *TestBlockStore, *TestTransitionBatchSubmitter) {
	submitCh := make(chan *TransitionBatch, 10)
	return submitCh, newTestBlockStore(make([]*types.Block, 0)), newTestBlockSubmitter(make([]*TransitionBatch, 0), submitCh)
}

/***************
 * Tests Start *
 ***************/

/********************
 * Submission Tests *
 ********************/

// Single block submission tests

func TestBatchSubmissionMaxTransactions(t *testing.T) {
	batchSubmitCh, blockStore, batchSubmitter := getSubmitChBlockStoreAndSubmitter()
	blockBuilder, err := newTestTransitionBatchBuilder(blockStore, batchSubmitter, 0, time.Minute*1, 1_000_000_000, 1)
	if err != nil {
		t.Fatalf("unable to make test batch builder, error: %v", err)
	}

	blocks := createBlocks(1, 1, true)
	blockBuilder.NewBlock(blocks[0])

	timeout := time.After(timeoutDuration)
	select {
	case transitionBatch := <-batchSubmitCh:
		assertTransitionFromBlock(t, transitionBatch.transitions[0], blocks[0])
		if len(batchSubmitter.submittedTransitions) > 1 {
			t.Fatal("Expected 1 batch to have been submitted", "numSubmitted", len(batchSubmitter.submittedTransitions))
		}
	case <-timeout:
		t.Fatalf("test timeout")
	}
}

func TestBlockLessThanMaxTransactions(t *testing.T) {
	batchSubmitCh, blockStore, batchSubmitter := getSubmitChBlockStoreAndSubmitter()
	blockBuilder, err := newTestTransitionBatchBuilder(blockStore, batchSubmitter, 0, time.Minute*1, 1_000_000_000, 2)
	if err != nil {
		t.Fatalf("unable to make test batch builder, error: %v", err)
	}

	blocks := createBlocks(1, 1, true)
	blockBuilder.NewBlock(blocks[0])

	timeout := time.After(timeoutDuration)
	select {
	case <-batchSubmitCh:
		t.Fatalf("should not have submitted a block")
	case <-timeout:
	}
}

func TestBatchSubmissionMaxGas(t *testing.T) {
	batchSubmitCh, blockStore, batchSubmitter := getSubmitChBlockStoreAndSubmitter()

	blocks := createBlocks(1, 1, true)
	gasLimit := GetBlockRollupGasUsage(blocks[0]) + TransitionBatchGasBuffer

	blockBuilder, err := newTestTransitionBatchBuilder(blockStore, batchSubmitter, 0, time.Minute*1, gasLimit, 2)
	if err != nil {
		t.Fatalf("unable to make test batch builder, error: %v", err)
	}

	blockBuilder.NewBlock(blocks[0])

	timeout := time.After(timeoutDuration)
	select {
	case transitionBatch := <-batchSubmitCh:
		assertTransitionFromBlock(t, transitionBatch.transitions[0], blocks[0])
		if len(batchSubmitter.submittedTransitions) > 1 {
			t.Fatal("Expected 1 batch to have been submitted", "numSubmitted", len(batchSubmitter.submittedTransitions))
		}
	case <-timeout:
		t.Fatalf("test timeout")
	}
}

func TestBlockLessThanMaxGas(t *testing.T) {
	batchSubmitCh, blockStore, batchSubmitter := getSubmitChBlockStoreAndSubmitter()

	blocks := createBlocks(1, 1, true)
	gasLimit := GetBlockRollupGasUsage(blocks[0]) + TransitionBatchGasBuffer + MinTxGas

	blockBuilder, err := newTestTransitionBatchBuilder(blockStore, batchSubmitter, 0, time.Minute*1, gasLimit, 2)
	if err != nil {
		t.Fatalf("unable to make test batch builder, error: %v", err)
	}

	blockBuilder.NewBlock(blocks[0])

	timeout := time.After(timeoutDuration)
	select {
	case <-batchSubmitCh:
		t.Fatalf("should not have submitted a block")
	case <-timeout:
	}
}

// Multiple block submission tests

func TestMultipleBatchSubmissionMaxTransactions(t *testing.T) {
	batchSubmitCh, blockStore, batchSubmitter := getSubmitChBlockStoreAndSubmitter()
	blockBuilder, err := newTestTransitionBatchBuilder(blockStore, batchSubmitter, 0, time.Minute*1, 1_000_000_000, 1)
	if err != nil {
		t.Fatalf("unable to make test batch builder, error: %v", err)
	}

	blocks := createBlocks(2, 1, true)
	blockBuilder.NewBlock(blocks[0])
	blockBuilder.NewBlock(blocks[1])

	timeout := time.After(timeoutDuration)
	select {
	case transitionBatch := <-batchSubmitCh:
		assertTransitionFromBlock(t, transitionBatch.transitions[0], blocks[0])
		time.Sleep(time.Microsecond * 10)
		if len(batchSubmitter.submittedTransitions) != 2 {
			t.Fatal("Expected 2 batch to have been submitted", "numSubmitted", len(batchSubmitter.submittedTransitions))
		}
		assertTransitionFromBlock(t, batchSubmitter.submittedTransitions[1].transitions[0], blocks[1])
	case <-timeout:
		t.Fatalf("test timeout")
	}
}

func TestMultipleBlocksLessThanMaxTransactions(t *testing.T) {
	batchSubmitCh, blockStore, batchSubmitter := getSubmitChBlockStoreAndSubmitter()
	blockBuilder, err := newTestTransitionBatchBuilder(blockStore, batchSubmitter, 0, time.Minute*1, 1_000_000_000, 3)
	if err != nil {
		t.Fatalf("unable to make test batch builder, error: %v", err)
	}

	blocks := createBlocks(2, 1, true)
	blockBuilder.NewBlock(blocks[0])
	blockBuilder.NewBlock(blocks[1])

	timeout := time.After(timeoutDuration)
	select {
	case <-batchSubmitCh:
		t.Fatalf("should not have submitted a block")
	case <-timeout:
	}
}

func TestMultipleBatchSubmissionMaxGas(t *testing.T) {
	batchSubmitCh, blockStore, batchSubmitter := getSubmitChBlockStoreAndSubmitter()

	blocks := createBlocks(2, 1, true)
	gasLimit := GetBlockRollupGasUsage(blocks[0]) + TransitionBatchGasBuffer

	blockBuilder, err := newTestTransitionBatchBuilder(blockStore, batchSubmitter, 0, time.Minute*1, gasLimit, 3)
	if err != nil {
		t.Fatalf("unable to make test batch builder, error: %v", err)
	}

	blockBuilder.NewBlock(blocks[0])
	blockBuilder.NewBlock(blocks[1])

	timeout := time.After(timeoutDuration)
	select {
	case transitionBatch := <-batchSubmitCh:
		assertTransitionFromBlock(t, transitionBatch.transitions[0], blocks[0])
		time.Sleep(time.Microsecond * 10)
		if len(batchSubmitter.submittedTransitions) != 2 {
			t.Fatal("Expected 2 batch to have been submitted", "numSubmitted", len(batchSubmitter.submittedTransitions))
		}
		assertTransitionFromBlock(t, batchSubmitter.submittedTransitions[1].transitions[0], blocks[1])
	case <-timeout:
		t.Fatalf("test timeout")
	}
}

func TestMultipleBlocksLessThanMaxGas(t *testing.T) {
	batchSubmitCh, blockStore, batchSubmitter := getSubmitChBlockStoreAndSubmitter()

	blocks := createBlocks(2, 1, true)
	gasLimit := 2 * (GetBlockRollupGasUsage(blocks[0]) + TransitionBatchGasBuffer + MinTxGas)

	blockBuilder, err := newTestTransitionBatchBuilder(blockStore, batchSubmitter, 0, time.Minute*1, gasLimit, 3)
	if err != nil {
		t.Fatalf("unable to make test batch builder, error: %v", err)
	}

	blockBuilder.NewBlock(blocks[0])
	blockBuilder.NewBlock(blocks[1])

	timeout := time.After(timeoutDuration)
	select {
	case <-batchSubmitCh:
		t.Fatalf("should not have submitted a block")
	case <-timeout:
	}
}

// Empty block tests

func TestEmptyBlocksIgnored(t *testing.T) {
	batchSubmitCh, blockStore, batchSubmitter := getSubmitChBlockStoreAndSubmitter()
	blockBuilder, err := newTestTransitionBatchBuilder(blockStore, batchSubmitter, 0, time.Minute*1, 1_000_000_000, 1)
	if err != nil {
		t.Fatalf("unable to make test batch builder, error: %v", err)
	}

	blocks := createBlocks(2, 1, false)
	blockBuilder.NewBlock(blocks[0])
	blockBuilder.NewBlock(blocks[1])

	timeout := time.After(timeoutDuration)
	select {
	case <-batchSubmitCh:
		t.Fatalf("should not have submitted a block")
	case <-timeout:
	}
}

func TestEmptyBlocksIgnoredWithNonEmpty(t *testing.T) {
	batchSubmitCh, blockStore, batchSubmitter := getSubmitChBlockStoreAndSubmitter()

	blockBuilder, err := newTestTransitionBatchBuilder(blockStore, batchSubmitter, 0, time.Minute*1, 1_000_000_000, 1)
	if err != nil {
		t.Fatalf("unable to make test batch builder, error: %v", err)
	}

	emptyBlocks := createBlocks(2, 1, false)

	blockBuilder.NewBlock(emptyBlocks[0])
	blockBuilder.NewBlock(emptyBlocks[1])

	nonEmpty := createBlocks(1, 3, true)[0]
	blockBuilder.NewBlock(nonEmpty)

	timeout := time.After(timeoutDuration)
	select {
	case transitionBatch := <-batchSubmitCh:
		assertTransitionFromBlock(t, transitionBatch.transitions[0], nonEmpty)
		if len(batchSubmitter.submittedTransitions) > 1 {
			t.Fatal("Expected 1 batch to have been submitted", "numSubmitted", len(batchSubmitter.submittedTransitions))
		}
	case <-timeout:
		t.Fatalf("test timeout")
	}
}

// timer submission

func TestBatchSubmissionMaxTimeBetweenBlocks(t *testing.T) {
	batchSubmitCh, blockStore, batchSubmitter := getSubmitChBlockStoreAndSubmitter()
	blockBuilder, err := newTestTransitionBatchBuilder(blockStore, batchSubmitter, 0, time.Microsecond*1, 1_000_000_000, 10)
	if err != nil {
		t.Fatalf("unable to make test batch builder, error: %v", err)
	}

	blocks := createBlocks(2, 1, true)
	blockBuilder.NewBlock(blocks[0])
	blockBuilder.NewBlock(blocks[1])

	timeout := time.After(timeoutDuration)
	select {
	case transitionBatch := <-batchSubmitCh:
		assertTransitionFromBlock(t, transitionBatch.transitions[0], blocks[0])
		time.Sleep(time.Microsecond * 10)
		if len(batchSubmitter.submittedTransitions) != 2 && len(transitionBatch.transitions) != 2 {
			t.Fatal("Expected 2 transitions to have been submitted", "blocksSubmitted", len(batchSubmitter.submittedTransitions), "transitionsInFirst", len(transitionBatch.transitions))
		}
		var secondTransition *Transition
		switch true {
		case len(batchSubmitter.submittedTransitions) == 2:
			secondTransition = batchSubmitter.submittedTransitions[1].transitions[0]
		case len(transitionBatch.transitions) == 2:
			secondTransition = transitionBatch.transitions[1]
		}
		assertTransitionFromBlock(t, secondTransition, blocks[1])
	case <-timeout:
		t.Fatalf("test timeout")
	}
}

func TestBatchSubmissionMaxTimeBetweenBlocksReset(t *testing.T) {
	batchSubmitCh, blockStore, batchSubmitter := getSubmitChBlockStoreAndSubmitter()
	blockBuilder, err := newTestTransitionBatchBuilder(blockStore, batchSubmitter, 0, time.Microsecond*1, 1_000_000_000, 10)
	if err != nil {
		t.Fatalf("unable to make test batch builder, error: %v", err)
	}

	blocks := createBlocks(2, 1, true)
	blockBuilder.NewBlock(blocks[0])

	timeout := time.After(timeoutDuration)
	select {
	case transitionBatch := <-batchSubmitCh:
		assertTransitionFromBlock(t, transitionBatch.transitions[0], blocks[0])
		if len(batchSubmitter.submittedTransitions) != 1 {
			t.Fatal("Expected 1 batch to have been submitted", "blocksSubmitted", len(batchSubmitter.submittedTransitions))
		}
	case <-timeout:
		t.Fatalf("test timeout")
	}

	blockBuilder.NewBlock(blocks[1])

	select {
	case transitionBatch := <-batchSubmitCh:
		assertTransitionFromBlock(t, transitionBatch.transitions[0], blocks[1])
		if len(batchSubmitter.submittedTransitions) != 2 {
			t.Fatal("Expected 2 batches to have been submitted", "blocksSubmitted", len(batchSubmitter.submittedTransitions))
		}
	case <-timeout:
		t.Fatalf("test timeout")
	}
}

/***********************
 * Existing Data Tests *
 ***********************/

func TestBatchSubmissionWithExistingData(t *testing.T) {
	batchSubmitCh, blockStore, batchSubmitter := getSubmitChBlockStoreAndSubmitter()
	blockBuilder, err := newTestTransitionBatchBuilder(blockStore, batchSubmitter, 1, time.Minute*1, 1_000_000_000, 1)
	if err != nil {
		t.Fatalf("unable to make test batch builder, error: %v", err)
	}

	blocks := createBlocks(1, 2, true)
	blockBuilder.NewBlock(blocks[0])

	timeout := time.After(timeoutDuration)
	select {
	case transitionBatch := <-batchSubmitCh:
		assertTransitionFromBlock(t, transitionBatch.transitions[0], blocks[0])
		if len(batchSubmitter.submittedTransitions) > 1 {
			t.Fatal("Expected 1 batch to have been submitted", "numSubmitted", len(batchSubmitter.submittedTransitions))
		}
	case <-timeout:
		t.Fatalf("test timeout")
	}
}

func TestBatchSubmissionWithExistingDataNoRepeats(t *testing.T) {
	batchSubmitCh, blockStore, batchSubmitter := getSubmitChBlockStoreAndSubmitter()
	blockBuilder, err := newTestTransitionBatchBuilder(blockStore, batchSubmitter, 1, time.Minute*1, 1_000_000_000, 1)
	if err != nil {
		t.Fatalf("unable to make test batch builder, error: %v", err)
	}

	blocks := createBlocks(1, 1, true)
	blockBuilder.NewBlock(blocks[0])

	timeout := time.After(timeoutDuration)
	select {
	case <-batchSubmitCh:
		t.Fatalf("block should not have been submitted")
	case <-timeout:
	}
}

func TestBatchSubmissionWithExistingDataNewBlocks(t *testing.T) {
	existingBlocks := createBlocks(2, 1, true)

	batchSubmitCh := make(chan *TransitionBatch, 10)
	blockStore, batchSubmitter := newTestBlockStore(existingBlocks), newTestBlockSubmitter(make([]*TransitionBatch, 0), batchSubmitCh)

	_, err := newTestTransitionBatchBuilder(blockStore, batchSubmitter, 1, time.Minute*1, 1_000_000_000, 1)
	if err != nil {
		t.Fatalf("unable to make test batch builder, error: %v", err)
	}

	timeout := time.After(timeoutDuration)
	select {
	case transitionBatch := <-batchSubmitCh:
		assertTransitionFromBlock(t, transitionBatch.transitions[0], existingBlocks[1])
		if len(batchSubmitter.submittedTransitions) != 1 {
			t.Fatal("Expected 1 batch to have been submitted", "numSubmitted", len(batchSubmitter.submittedTransitions))
		}
	case <-timeout:
		t.Fatalf("test timeout")

	}
}
