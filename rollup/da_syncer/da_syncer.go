package da_syncer

import (
	"fmt"

	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/da"
)

var (
	ErrBlockTooLow  = fmt.Errorf("block number is too low")
	ErrBlockTooHigh = fmt.Errorf("block number is too high")
)

type DASyncer struct {
	blockchain *core.BlockChain
}

func NewDASyncer(blockchain *core.BlockChain) *DASyncer {
	return &DASyncer{
		blockchain: blockchain,
	}
}

// SyncOneBlock receives a PartialBlock, makes sure it's the next block in the chain, executes it and inserts it to the blockchain.
func (s *DASyncer) SyncOneBlock(block *da.PartialBlock) error {
	currentBlock := s.blockchain.CurrentBlock()

	// we expect blocks to be consecutive. block.PartialHeader.Number == parentBlock.Number+1.
	if block.PartialHeader.Number <= currentBlock.Number().Uint64() {
		log.Debug("block number is too low", "block number", block.PartialHeader.Number, "parent block number", currentBlock.Number().Uint64())
		return ErrBlockTooLow
	} else if block.PartialHeader.Number > currentBlock.Number().Uint64()+1 {
		log.Debug("block number is too high", "block number", block.PartialHeader.Number, "parent block number", currentBlock.Number().Uint64())
		return ErrBlockTooHigh
	}

	parentBlock := s.blockchain.GetBlockByNumber(currentBlock.Number().Uint64())
	if parentBlock == nil {
		return fmt.Errorf("parent block not found at height %d", currentBlock.Number().Uint64())
	}

	if _, err := s.blockchain.BuildAndWriteBlock(parentBlock, block.PartialHeader.ToHeader(), block.Transactions); err != nil {
		return fmt.Errorf("failed building and writing block, number: %d, error: %v", block.PartialHeader.Number, err)
	}

	if s.blockchain.CurrentBlock().Number().Uint64()%1000 == 0 {
		log.Info("L1 sync progress", "blockchain height", s.blockchain.CurrentBlock().Number().Uint64(), "block hash", s.blockchain.CurrentBlock().Hash(), "root", s.blockchain.CurrentBlock().Root())
	}

	return nil
}
