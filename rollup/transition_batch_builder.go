package rollup

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"sync"
	"time"
)

var (
	logger                     = log.New(TransitionBatchBuilder{})
	ErrTransactionLimitReached = errors.New("transaction limit reached")
	ErrMoreThanOneTxInBlock    = errors.New("block contains more than one transaction")
	LastProcessedDBKey         = []byte("lastProcessedRollupBlock")
)

type ActiveBatch struct {
	firstBlockNumber uint64
	lastBlockNumber  uint64
	gasUsed          uint64

	transitionBatch *TransitionBatch
}

func newActiveBatch(defaultTxCapacity int) *ActiveBatch {
	return &ActiveBatch{
		firstBlockNumber: 0,
		lastBlockNumber:  0,
		gasUsed:          TransitionBatchGasBuffer,
		transitionBatch:  NewTransitionBatch(defaultTxCapacity),
	}
}

// addBlock adds a Geth Block to the ActiveBatch in question, only if it fits.
// Cases in which it would not fit are if it would put the block above the configured
// max number of transactions or max block gas, resulting in
// ErrTransactionLimitReached and core.ErrGasLimitReached, respectively.
func (b *ActiveBatch) addBlock(block *types.Block, maxBlockGas uint64, maxBlockTransactions int) error {
	if maxBlockTransactions < len(b.transitionBatch.transitions)+1 {
		return ErrTransactionLimitReached
	}
	blockGasCost := GetBlockRollupGasUsage(block)
	if maxBlockGas < b.gasUsed+blockGasCost {
		return core.ErrGasLimitReached
	}

	b.transitionBatch.addBlock(block)
	b.gasUsed += blockGasCost
	if b.firstBlockNumber == 0 {
		b.firstBlockNumber = block.NumberU64()
	}
	b.lastBlockNumber = block.NumberU64()

	return nil
}

type TransitionBatchBuilder struct {
	db                   ethdb.Database
	blockProvider        BlockStore
	rollupBatchSubmitter RollupTransitionBatchSubmitter
	pendingMu            sync.RWMutex

	newBlockCh chan *types.Block

	maxTransitionBatchTime         time.Duration
	maxTransitionBatchGas          uint64
	maxTransitionBatchTransactions int

	lastProcessedBlockNumber uint64
	activeBatch              *ActiveBatch
}

func NewTransitionBatchBuilder(db ethdb.Database, blockStore interface{}, rollupBlockSubmitter interface{}, maxBlockTime time.Duration, maxBlockGas uint64, maxBlockTransactions int) (*TransitionBatchBuilder, error) {
	lastBlock, err := fetchLastProcessedBlockNumber(db)
	if err != nil {
		return nil, err
	}

	builder := &TransitionBatchBuilder{
		db:                   db,
		blockProvider:        blockStore.(BlockStore),
		rollupBatchSubmitter: rollupBlockSubmitter.(RollupTransitionBatchSubmitter),
		newBlockCh:           make(chan *types.Block, 10_000),

		maxTransitionBatchTime:         maxBlockTime,
		maxTransitionBatchGas:          maxBlockGas,
		maxTransitionBatchTransactions: maxBlockTransactions,

		lastProcessedBlockNumber: lastBlock,
		activeBatch:              newActiveBatch(maxBlockTransactions),
	}

	go builder.buildLoop(maxBlockTime)

	return builder, nil
}

// NewBlock handles new blocks from Geth by adding them to the newBlockCh channel
// for processing and returning so as to not delay the caller.
func (b *TransitionBatchBuilder) NewBlock(block *types.Block) {
	b.newBlockCh <- block
}

// Stop handles graceful shutdown of the TransitionBatchBuilder.
func (b *TransitionBatchBuilder) Stop() {
	close(b.newBlockCh)
}

// buildLoop initiates TransitionBatch production and submission either based on
// a new Geth Block being received or the maxBlockTime being reached.
func (b *TransitionBatchBuilder) buildLoop(maxBlockTime time.Duration) {
	lastProcessed := b.lastProcessedBlockNumber

	if err := b.sync(); err != nil {
		panic(fmt.Errorf("error syncing: %+v", err))
	}

	timer := time.NewTimer(maxBlockTime)

	for {
		select {
		case block, ok := <-b.newBlockCh:
			if !ok {
				timer.Stop()
				logger.Info("Closing transition batch builder new block channel. If not shutting down, this is an error")
				return
			}

			built, err := b.handleNewBlock(block)
			if err != nil {
				panic(fmt.Errorf("error handling new block. Error: %v. Block: %+v", err, block))
			}
			if timer != nil && built {
				timer.Reset(b.maxTransitionBatchTime)
			}
		case <-timer.C:
			if lastProcessed != b.lastProcessedBlockNumber && b.activeBatch.firstBlockNumber != 0 {
				if _, err := b.buildRollupBlock(true); err != nil {
					panic(fmt.Errorf("error buidling block: %v", err))
				}
			}

			lastProcessed = b.lastProcessedBlockNumber
			timer.Reset(maxBlockTime)
		}
	}
}

// handleNewBlock processes a newly received Geth Block, ignoring old / future blocks
// and building and submitting TransitionBatches if the pending TransitionBatch is full.
func (b *TransitionBatchBuilder) handleNewBlock(block *types.Block) (bool, error) {
	logger.Debug("handling new block in transition batch builder", "block", block)
	if block.NumberU64() <= b.lastProcessedBlockNumber {
		logger.Debug("handling old block -- ignoring", "block", block)
		return false, nil
	}
	if block.NumberU64() > b.lastProcessedBlockNumber+1 {
		logger.Error("received future block", "block", block, "expectedNumber", b.lastProcessedBlockNumber+1)
		// TODO: add to queue and/or try to fetch blocks in between.
		return false, nil
	}

	if txCount := len(block.Transactions()); txCount > 1 {
		// should never happen
		logger.Error("received block with more than one transaction", "block", block)
		return false, ErrMoreThanOneTxInBlock
	} else if txCount == 0 {
		logger.Debug("handling empty block -- ignoring", "block", block)
		b.lastProcessedBlockNumber = block.NumberU64()
		return false, nil
	}

	switch err := b.addBlock(block); err {
	case core.ErrGasLimitReached, ErrTransactionLimitReached:
		if _, e := b.buildRollupBlock(false); e != nil {
			logger.Error("unable to build transition batch", "error", e, "transition batch", b.activeBatch)
			return false, e
		}
		if addErr := b.addBlock(block); addErr != nil {
			// TODO: Retry and whatnot instead of instant panic
			logger.Error("unable to build transition batch", "error", addErr, "transition batch", b.activeBatch)
			return false, addErr
		}
	default:
		if err != nil {
			logger.Error("unrecognized error adding to transition batch in progress", "error", err, "transition batch", b.activeBatch)
			return false, err
		} else {
			logger.Debug("successfully added block to transition batch in progress", "number", block.NumberU64())
		}
	}

	built, err := b.tryBuildRollupBlock()
	if err != nil {
		logger.Error("error building block", "error", err, "block", block)
		return false, err
	}

	return built, nil
}

// sync catches the TransitionBatchBuilder up to the Geth chain by fetching all Geth Blocks between
// its last processed Block and the current Block, building and submitting RollupBlocks if/when
// they are full.
func (b *TransitionBatchBuilder) sync() error {
	logger.Info("syncing blocks in transition batch builder", "starting block", b.lastProcessedBlockNumber)

	for {
		blockNum := b.lastProcessedBlockNumber + uint64(1)
		block := b.blockProvider.GetBlockByNumber(blockNum)
		logger.Info("got block number", "number", blockNum, "block", block)
		if block == nil {
			logger.Info("done syncing blocks in transition batch builder", "number", b.lastProcessedBlockNumber)
			return nil
		}
		if _, err := b.handleNewBlock(block); err != nil {
			logger.Error("Error handling new block", "error", err)
			return err
		} else {
			logger.Debug("successfully synced block", "number", blockNum, "last processed", b.lastProcessedBlockNumber)
		}
	}
}

// addBlock adds a Geth Block to the TransitionBatch if it fits. If not, it will return an error.
func (b *TransitionBatchBuilder) addBlock(block *types.Block) error {
	b.pendingMu.Lock()
	defer b.pendingMu.Unlock()
	if err := b.activeBatch.addBlock(block, b.maxTransitionBatchGas, b.maxTransitionBatchTransactions); err != nil {
		return err
	}
	b.lastProcessedBlockNumber = block.NumberU64()
	return nil
}

// tryBuildRollupBlock builds and submits a TransitionBatch if the pending TransitionBatch is full.
func (b *TransitionBatchBuilder) tryBuildRollupBlock() (bool, error) {
	txCount := len(b.activeBatch.transitionBatch.transitions)
	gasAfterOneMoreTx := b.activeBatch.gasUsed + MinTxGas
	if txCount < b.maxTransitionBatchTransactions && gasAfterOneMoreTx <= b.maxTransitionBatchGas {
		logger.Debug("transition batch is not full, so not finalizing it", "txCount", txCount, "gasAfterOneMoreTx", gasAfterOneMoreTx)
		return false, nil
	}
	logger.Debug("transition batch is full, finalizing it", "txCount", txCount, "gasAfterOneMoreTx", gasAfterOneMoreTx)

	return b.buildRollupBlock(false)
}

// buildRollupBlock builds a TransitionBatch if the pending TransitionBatch is full or if force is true
// and the pending TransitionBatch is not empty.
func (b *TransitionBatchBuilder) buildRollupBlock(force bool) (bool, error) {
	var toSubmit *ActiveBatch
	b.pendingMu.Lock()
	defer b.pendingMu.Unlock()

	txCount := len(b.activeBatch.transitionBatch.transitions)

	if force && txCount == 0 {
		logger.Debug("transition batch is empty so not finalizing it, even though force = true")
		return false, nil
	}
	if !force && txCount < b.maxTransitionBatchTransactions && b.activeBatch.gasUsed+MinTxGas <= b.maxTransitionBatchGas {
		logger.Debug("transition batch is not full, so not finalizing it")
		return false, nil
	}
	logger.Debug("building transition batch")

	toSubmit = b.activeBatch
	b.activeBatch = newActiveBatch(b.maxTransitionBatchTransactions)

	if err := b.submitBlock(toSubmit); err != nil {
		logger.Error("error submitting transition batch", "lastBlockNumber", toSubmit.lastBlockNumber, "error", err)
		return false, err
	}
	logger.Debug("successfully built transition batch", "lastBlockNumber", toSubmit.lastBlockNumber)

	return true, nil
}

// submitBlock submits a TransitionBatch to the RollupTransitionBatchSubmitter and updates the DB
// to indicate the last processed Geth Block included in the TransitionBatch.
func (b *TransitionBatchBuilder) submitBlock(block *ActiveBatch) error {
	// TODO: Submit to chain & get hash
	logger.Debug("submitting transition batch", "block", block)

	if err := b.rollupBatchSubmitter.submit(block.transitionBatch); err != nil {
		return err
	}

	if err := b.db.Put(LastProcessedDBKey, SerializeBlockNumber(block.lastBlockNumber)); err != nil {
		logger.Error("error saving last processed transition batch", "block", block)
		// TODO: Something here
	}
	logger.Debug("transition batch submitted", "block", block)
	return nil
}

// fetchLastProcessedBlockNumber fetches the last processed Geth Block # from the DB.
func fetchLastProcessedBlockNumber(db ethdb.Database) (uint64, error) {
	has, err := db.Has(LastProcessedDBKey)
	if err != nil {
		logger.Error("received error checking if LastProcessedDBKey exists in DB", "error", err)
		return 0, err
	}
	if has {
		lastProcessedBytes, e := db.Get(LastProcessedDBKey)
		if e != nil {
			logger.Error("error fetching LastProcessedDBKey from DB", "error", err)
			return 0, err
		}
		lastProcessedBlock := DeserializeBlockNumber(lastProcessedBytes)
		logger.Info("fetched last processed block from database", "number", lastProcessedBlock)
		return lastProcessedBlock, nil
	} else {
		logger.Info("no last processed block found in the db -- returning 0")
		return 0, nil
	}
}

// SerializeBlockNumber serializes the number for DB storage
func SerializeBlockNumber(blockNumber uint64) []byte {
	numberAsByteArray := make([]byte, 8)
	binary.LittleEndian.PutUint64(numberAsByteArray, blockNumber)
	return numberAsByteArray
}

// DeserializeBlockNumber deserializes the number from DB storage
func DeserializeBlockNumber(blockNumber []byte) uint64 {
	return binary.LittleEndian.Uint64(blockNumber)
}

// GetBlockRollupGasUsage determines the amount of L1 gas the provided Geth Block will use
// when submitted to mainnet.
func GetBlockRollupGasUsage(block *types.Block) uint64 {
	return params.SstoreSetGas + uint64(len(block.Transactions()[0].Data()))*params.TxDataNonZeroGasEIP2028
}
