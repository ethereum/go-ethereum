package main

import (
	"fmt"
	"github.com/ethereum/ethutil-go"
	"errors"
	"log"
	"math/big"
)

type BlockChain struct {
	LastBlock *ethutil.Block

	genesisBlock *ethutil.Block

	TD *big.Int
}

func NewBlockChain() *BlockChain {
	bc := &BlockChain{}
	bc.genesisBlock = ethutil.NewBlock(ethutil.Encode(ethutil.Genesis))

	// Set the last know difficulty (might be 0x0 as initial value, Genesis)
	bc.TD = new(big.Int)
	bc.TD.SetBytes(ethutil.Config.Db.LastKnownTD())

	return bc
}

func (bc *BlockChain) HasBlock(hash string) bool {
	return bc.LastBlock.State().Get(hash) != ""
}

type BlockManager struct {
	// Ethereum virtual machine for processing contracts
	vm *Vm
	// The block chain :)
	bc *BlockChain
}

func NewBlockManager() *BlockManager {
	bm := &BlockManager{
		vm: NewVm(),
		bc: NewBlockChain(),
	}

	return bm
}

// Process a block.
func (bm *BlockManager) ProcessBlock(block *ethutil.Block) error {
	// Block validation
	if err := bm.ValidateBlock(block); err != nil {
		return err
	}

	// I'm not sure, but I don't know if there should be thrown
	// any errors at this time.
	if err := bm.AccumelateRewards(block); err != nil {
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

	if bm.CalculateTD(block) {
		ethutil.Config.Db.Put(block.Hash(), block.MarshalRlp())
		bm.bc.LastBlock = block
	}

	return nil
}

func (bm *BlockManager) CalculateTD(block *ethutil.Block) bool {
	uncleDiff := new(big.Int)
	for _, uncle := range block.Uncles {
		uncleDiff = uncleDiff.Add(uncleDiff, uncle.Difficulty)
	}

	// TD(genesis_block) = 0 and TD(B) = TD(B.parent) + sum(u.difficulty for u in B.uncles) + B.difficulty
	td := new(big.Int)
	td = td.Add(bm.bc.TD, uncleDiff)
	td = td.Add(td, block.Difficulty)

	// The new TD will only be accepted if the new difficulty is
	// is greater than the previous.
	if td.Cmp(bm.bc.TD) > 0 {
		bm.bc.LastBlock = block
		// Set the new total difficulty back to the block chain
		bm.bc.TD = td

		return true
	}

	return false
}

// Validates the current block. Returns an error if the block was invalid,
// an uncle or anything that isn't on the current block chain
func (bm *BlockManager) ValidateBlock(block *ethutil.Block) error {
	// TODO
	// 1. Check if the nonce of the block is valid
	// 2. Check if the difficulty is correct

	// Check if we have the parent hash, if it isn't known we discard it
	// Reasons might be catching up or simply an invalid block
	if bm.bc.HasBlock(block.PrevHash) {
		// Check each uncle's previous hash. In order for it to be valid
		// is if it has the same block hash as the current
		for _, uncle := range block.Uncles {
			if uncle.PrevHash != block.PrevHash {
				if Debug {
					log.Printf("Uncle prvhash mismatch %x %x\n", block.PrevHash, uncle.PrevHash)
				}

				return errors.New("Mismatching Prvhash from uncle")
			}
		}
	} else {
		return errors.New("Block's parent unknown")
	}


	return nil
}

func (bm *BlockManager) AccumelateRewards(block *ethutil.Block) error {
	// Get the coinbase rlp data
	d := block.State().Get(block.Coinbase)

	ether := ethutil.NewEtherFromData([]byte(d))

	// Reward amount of ether to the coinbase address
	ether.AddFee(ethutil.CalculateBlockReward(block, len(block.Uncles)))
	block.State().Update(block.Coinbase, string(ether.MarshalRlp()))

	// TODO Reward each uncle


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
