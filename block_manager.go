package main

import (
	"fmt"
	"github.com/ethereum/ethutil-go"
)

type BlockChain struct {
	lastBlock *ethutil.Block

	genesisBlock *ethutil.Block
}

func NewBlockChain() *BlockChain {
	bc := &BlockChain{}
	bc.genesisBlock = ethutil.NewBlock(ethutil.Encode(ethutil.Genesis))

	return bc
}

type BlockManager struct {
	vm *Vm

	blockChain *BlockChain
}

func NewBlockManager() *BlockManager {
	bm := &BlockManager{vm: NewVm()}

	return bm
}

// Process a block.
func (bm *BlockManager) ProcessBlock(block *ethutil.Block) error {
	// TODO Validation (Or move to other part of the application)
	if err := bm.ValidateBlock(block); err != nil {
		return err
	}

	// Get the tx count. Used to create enough channels to 'join' the go routines
	txCount := len(block.Transactions())
	// Locking channel. When it has been fully buffered this method will return
	lockChan := make(chan bool, txCount)

	// Process each transaction/contract
	for _, tx := range block.Transactions() {
		// If there's no recipient, it's a contract
		if tx.IsContract() {
			go bm.ProcessContract(tx, block, lockChan)
		} else {
			// "finish" tx which isn't a contract
			lockChan <- true
		}
	}

	// Wait for all Tx to finish processing
	for i := 0; i < txCount; i++ {
		<-lockChan
	}

	return nil
}

func (bm *BlockManager) ValidateBlock(block *ethutil.Block) error {
	return nil
}

func (bm *BlockManager) ProcessContract(tx *ethutil.Transaction, block *ethutil.Block, lockChan chan bool) {
	// Recovering function in case the VM had any errors
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from VM execution with err =", r)
			// Let the channel know where done even though it failed (so the execution may resume normally)
			lockChan <- true
		}
	}()

	// Process contract
	bm.vm.ProcContract(tx, block, func(opType OpType) bool {
		// TODO turn on once big ints are in place
		//if !block.PayFee(tx.Hash(), StepFee.Uint64()) {
		//  return false
		//}

		return true // Continue
	})

	// Broadcast we're done
	lockChan <- true
}
