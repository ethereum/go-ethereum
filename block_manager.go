
  // Blocks, blocks will have transactions.
  // Transactions/contracts are updated in goroutines
  // Each contract should send a message on a channel with usage statistics
  // The statics can be used for fee calculation within the block update method
  // Statistics{transaction, /* integers */ normal_ops, store_load, extro_balance, crypto, steps}
  // The block updater will wait for all goroutines to be finished and update the block accordingly
  // in one go and should use minimal IO overhead.
  // The actual block updating will happen within a goroutine as well so normal operation may continue

package main

import (
  _"fmt"
)

type BlockManager struct {
  vm *Vm
}

func NewBlockManager() *BlockManager {
  bm := &BlockManager{vm: NewVm()}

  return bm
}

// Process a block.
func (bm *BlockManager) ProcessBlock(block *Block) error {
  txCount  := len(block.transactions)
  lockChan := make(chan bool, txCount)

  for _, tx := range block.transactions {
    go bm.ProcessTransaction(tx, lockChan)
  }

  // Wait for all Tx to finish processing
  for i := 0; i < txCount; i++ {
    <- lockChan
  }

  return nil
}

func (bm *BlockManager) ProcessTransaction(tx *Transaction, lockChan chan bool) {
  if tx.recipient == "\x00" {
    bm.vm.RunTransaction(tx, func(opType OpType) bool {
      // TODO calculate fees

      return true // Continue
    })
  }

  // Broadcast we're done
  lockChan <- true
}
