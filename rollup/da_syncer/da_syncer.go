package da_syncer

import (
	"fmt"

	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/da"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/serrors"
)

var (
	ErrBlockTooLow  = fmt.Errorf("block number is too low")
	ErrBlockTooHigh = fmt.Errorf("block number is too high")
)

type DASyncer struct {
	l2EndBlock uint64
	blockchain *core.BlockChain
}

func NewDASyncer(blockchain *core.BlockChain, l2EndBlock uint64) *DASyncer {
	return &DASyncer{
		l2EndBlock: l2EndBlock,
		blockchain: blockchain,
	}
}

// SyncOneBlock receives a PartialBlock, makes sure it's the next block in the chain, executes it and inserts it to the blockchain.
func (s *DASyncer) SyncOneBlock(block *da.PartialBlock, override bool, sign bool) error {
	currentBlock := s.blockchain.CurrentBlock()

	// we expect blocks to be consecutive. block.PartialHeader.Number == parentBlock.Number+1.
	// if override is true, we allow blocks to be lower than the current block number and replace the blocks.
	if !override && block.PartialHeader.Number <= currentBlock.Number().Uint64() {
		log.Debug("block number is too low", "block number", block.PartialHeader.Number, "parent block number", currentBlock.Number().Uint64())
		return ErrBlockTooLow
	} else if block.PartialHeader.Number > currentBlock.Number().Uint64()+1 {
		log.Debug("block number is too high", "block number", block.PartialHeader.Number, "parent block number", currentBlock.Number().Uint64())
		return ErrBlockTooHigh
	}

	parentBlockNumber := currentBlock.Number().Uint64()
	if override {
		parentBlockNumber = block.PartialHeader.Number - 1
		// reset the chain head to the parent block so that the new block can be inserted as part of the new canonical chain.
		err := s.blockchain.SetHead(parentBlockNumber)
		if err != nil {
			return fmt.Errorf("failed setting head, number: %d, error: %v", parentBlockNumber, err)
		}
	}

	parentBlock := s.blockchain.GetBlockByNumber(parentBlockNumber)
	if parentBlock == nil {
		return fmt.Errorf("failed getting parent block, number: %d", parentBlockNumber)
	}

	fullBlock, writeStatus, err := s.blockchain.BuildAndWriteBlock(parentBlock, block.PartialHeader.ToHeader(), block.Transactions, sign)
	if err != nil {
		return fmt.Errorf("failed building and writing block, number: %d, error: %v", block.PartialHeader.Number, err)
	}
	if writeStatus != core.CanonStatTy {
		return fmt.Errorf("failed writing block as part of canonical chain, number: %d, status: %d", block.PartialHeader.Number, writeStatus)
	}

	currentBlock = s.blockchain.CurrentBlock()
	if currentBlock.Number().Uint64() != fullBlock.NumberU64() || currentBlock.Hash() != fullBlock.Hash() {
		return fmt.Errorf("failed to insert block: not part of canonical chain, number: %d, hash: %s - canonical: number: %d, hash: %s", fullBlock.NumberU64(), fullBlock.Hash(), currentBlock.Number().Uint64(), currentBlock.Hash())
	}

	if fullBlock.Number().Uint64()%100 == 0 {
		log.Info("L1 sync progress", "blockchain height", fullBlock.Number().Uint64(), "block hash", fullBlock.Hash(), "root", fullBlock.Root())
	}

	if s.l2EndBlock > 0 && s.l2EndBlock == block.PartialHeader.Number {
		log.Warn("L1 sync reached L2EndBlock: you can terminate recovery mode now", "L2EndBlock", fullBlock.NumberU64(), "block hash", fullBlock.Hash(), "root", fullBlock.Root())
		return serrors.Terminated
	}

	return nil
}
