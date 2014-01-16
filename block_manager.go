package main

import (
	"errors"
	"fmt"
	"github.com/ethereum/ethutil-go"
	"log"
	"math/big"
	"strconv"
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

	// TODO get last block from the database
	//bc.LastBlock = bc.genesisBlock

	return bc
}

func (bc *BlockChain) HasBlock(hash string) bool {
	data, _ := ethutil.Config.Db.Get([]byte(hash))
	return len(data) != 0
}

func (bc *BlockChain) GenesisBlock() *ethutil.Block {
	return bc.genesisBlock
}

type BlockManager struct {
	// The block chain :)
	bc *BlockChain

	// Stack for processing contracts
	stack *Stack
}

func NewBlockManager() *BlockManager {
	bm := &BlockManager{
		bc:    NewBlockChain(),
		stack: NewStack(),
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

		if Debug {
			log.Println("TD(block) =", td)
		}

		return true
	}

	return false
}

// Validates the current block. Returns an error if the block was invalid,
// an uncle or anything that isn't on the current block chain.
// Validation validates easy over difficult (dagger takes longer time = difficult)
func (bm *BlockManager) ValidateBlock(block *ethutil.Block) error {
	// TODO
	// 2. Check if the difficulty is correct

	// Check if we have the parent hash, if it isn't known we discard it
	// Reasons might be catching up or simply an invalid block
	if bm.bc.LastBlock != nil && block.PrevHash == "" &&
		!bm.bc.HasBlock(block.PrevHash) {
		return errors.New("Block's parent unknown")
	}

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

	// Verify the nonce of the block. Return an error if it's not valid
	if bm.bc.LastBlock != nil && block.PrevHash == "" &&
		!DaggerVerify(ethutil.BigD(block.Hash()), block.Difficulty, block.Nonce) {

		return errors.New("Block's nonce is invalid")
	}

	log.Println("Block validation PASSED")

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
	bm.ProcContract(tx, block, func(opType OpType) bool {
		// TODO turn on once big ints are in place
		//if !block.PayFee(tx.Hash(), StepFee.Uint64()) {
		//  return false
		//}

		return true // Continue
	})

	// Broadcast we're done
	lockChan <- true
}

func (bm *BlockManager) ProcContract(tx *ethutil.Transaction,
	block *ethutil.Block, cb TxCallback) {
	// Instruction pointer
	pc := 0

	contract := block.GetContract(tx.Hash())
	if contract == nil {
		fmt.Println("Contract not found")
		return
	}

	Pow256 := ethutil.BigPow(2, 256)

	//fmt.Printf("#   op   arg\n")
out:
	for {
		// The base big int for all calculations. Use this for any results.
		base := new(big.Int)
		// XXX Should Instr return big int slice instead of string slice?
		// Get the next instruction from the contract
		//op, _, _ := Instr(contract.state.Get(string(Encode(uint32(pc)))))
		nb := ethutil.NumberToBytes(uint64(pc), 32)
		o, _, _ := ethutil.Instr(contract.State().Get(string(nb)))
		op := OpCode(o)

		if !cb(0) {
			break
		}

		if Debug {
			//fmt.Printf("%-3d %-4s\n", pc, op.String())
		}

		switch op {
		case oADD:
			x, y := bm.stack.Popn()
			// (x + y) % 2 ** 256
			base.Add(x, y)
			base.Mod(base, Pow256)
			// Pop result back on the stack
			bm.stack.Push(base.String())
		case oSUB:
			x, y := bm.stack.Popn()
			// (x - y) % 2 ** 256
			base.Sub(x, y)
			base.Mod(base, Pow256)
			// Pop result back on the stack
			bm.stack.Push(base.String())
		case oMUL:
			x, y := bm.stack.Popn()
			// (x * y) % 2 ** 256
			base.Mul(x, y)
			base.Mod(base, Pow256)
			// Pop result back on the stack
			bm.stack.Push(base.String())
		case oDIV:
			x, y := bm.stack.Popn()
			// floor(x / y)
			base.Div(x, y)
			// Pop result back on the stack
			bm.stack.Push(base.String())
		case oSDIV:
			x, y := bm.stack.Popn()
			// n > 2**255
			if x.Cmp(Pow256) > 0 {
				x.Sub(Pow256, x)
			}
			if y.Cmp(Pow256) > 0 {
				y.Sub(Pow256, y)
			}
			z := new(big.Int)
			z.Div(x, y)
			if z.Cmp(Pow256) > 0 {
				z.Sub(Pow256, z)
			}
			// Push result on to the stack
			bm.stack.Push(z.String())
		case oMOD:
			x, y := bm.stack.Popn()
			base.Mod(x, y)
			bm.stack.Push(base.String())
		case oSMOD:
			x, y := bm.stack.Popn()
			// n > 2**255
			if x.Cmp(Pow256) > 0 {
				x.Sub(Pow256, x)
			}
			if y.Cmp(Pow256) > 0 {
				y.Sub(Pow256, y)
			}
			z := new(big.Int)
			z.Mod(x, y)
			if z.Cmp(Pow256) > 0 {
				z.Sub(Pow256, z)
			}
			// Push result on to the stack
			bm.stack.Push(z.String())
		case oEXP:
			x, y := bm.stack.Popn()
			base.Exp(x, y, Pow256)

			bm.stack.Push(base.String())
		case oNEG:
			base.Sub(Pow256, ethutil.Big(bm.stack.Pop()))
			bm.stack.Push(base.String())
		case oLT:
			x, y := bm.stack.Popn()
			// x < y
			if x.Cmp(y) < 0 {
				bm.stack.Push("1")
			} else {
				bm.stack.Push("0")
			}
		case oLE:
			x, y := bm.stack.Popn()
			// x <= y
			if x.Cmp(y) < 1 {
				bm.stack.Push("1")
			} else {
				bm.stack.Push("0")
			}
		case oGT:
			x, y := bm.stack.Popn()
			// x > y
			if x.Cmp(y) > 0 {
				bm.stack.Push("1")
			} else {
				bm.stack.Push("0")
			}
		case oGE:
			x, y := bm.stack.Popn()
			// x >= y
			if x.Cmp(y) > -1 {
				bm.stack.Push("1")
			} else {
				bm.stack.Push("0")
			}
		case oNOT:
			x, y := bm.stack.Popn()
			// x != y
			if x.Cmp(y) != 0 {
				bm.stack.Push("1")
			} else {
				bm.stack.Push("0")
			}

		// Please note  that the  following code contains some
		// ugly string casting. This will have to change to big
		// ints. TODO :)
		case oMYADDRESS:
			bm.stack.Push(string(tx.Hash()))
		case oTXSENDER:
			bm.stack.Push(string(tx.Sender()))
		case oTXVALUE:
			bm.stack.Push(tx.Value.String())
		case oTXDATAN:
			bm.stack.Push(big.NewInt(int64(len(tx.Data))).String())
		case oTXDATA:
			v := ethutil.Big(bm.stack.Pop())
			// v >= len(data)
			if v.Cmp(big.NewInt(int64(len(tx.Data)))) >= 0 {
				//I know this will change. It makes no
				//sense. Read comment above
				bm.stack.Push(ethutil.Big("0").String())
			} else {
				bm.stack.Push(ethutil.Big(tx.Data[v.Uint64()]).String())
			}
		case oBLK_PREVHASH:
			bm.stack.Push(string(block.PrevHash))
		case oBLK_COINBASE:
			bm.stack.Push(block.Coinbase)
		case oBLK_TIMESTAMP:
			bm.stack.Push(big.NewInt(block.Time).String())
		case oBLK_NUMBER:

		case oPUSH:
			// Get the next entry and pushes the value on the stack
			pc++
			bm.stack.Push(contract.State().Get(string(ethutil.NumberToBytes(uint64(pc), 32))))
		case oPOP:
			// Pop current value of the stack
			bm.stack.Pop()
		case oLOAD:
			// Load instruction X on the stack
			i, _ := strconv.Atoi(bm.stack.Pop())
			bm.stack.Push(contract.State().Get(string(ethutil.NumberToBytes(uint64(i), 32))))
		case oSTOP:
			break out
		}
		pc++
	}

	bm.stack.Print()
}
