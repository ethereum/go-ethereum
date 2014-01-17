package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/ethereum/ethutil-go"
	"github.com/obscuren/secp256k1-go"
	"log"
	"math"
	"math/big"
)

type BlockChain struct {
	// Last block
	LastBlock *ethutil.Block
	// The famous, the fabulous Mister GENESIIIIIIS (block)
	genesisBlock *ethutil.Block
	// Last known total difficulty
	TD *big.Int
}

func NewBlockChain() *BlockChain {
	bc := &BlockChain{}
	bc.genesisBlock = ethutil.NewBlock(ethutil.Encode(ethutil.Genesis))

	// Set the last know difficulty (might be 0x0 as initial value, Genesis)
	bc.TD = ethutil.BigD(ethutil.Config.Db.LastKnownTD())

	// TODO get last block from the database
	bc.LastBlock = bc.genesisBlock

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

	// Last known block number
	LastBlockNumber *big.Int

	// Stack for processing contracts
	stack *Stack
	// non-persistent key/value memory storage
	mem map[string]string
}

func NewBlockManager() *BlockManager {
	bm := &BlockManager{
		bc:    NewBlockChain(),
		stack: NewStack(),
		mem:   make(map[string]string),
	}

	// Set the last known block number based on the blockchains last
	// block
	bm.LastBlockNumber = bm.BlockInfo(bm.bc.LastBlock).Number

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

	// Calculate the new total difficulty and sync back to the db
	if bm.CalculateTD(block) {
		ethutil.Config.Db.Put(block.Hash(), block.RlpEncode())
		bm.bc.LastBlock = block
	}

	return nil
}

// Unexported method for writing extra non-essential block info to the db
func (bm *BlockManager) writeBlockInfo(block *ethutil.Block) {
	bi := ethutil.BlockInfo{Number: bm.LastBlockNumber.Add(bm.LastBlockNumber, big.NewInt(1))}

	// For now we use the block hash with the words "info" appended as key
	ethutil.Config.Db.Put(append(block.Hash(), []byte("Info")...), bi.RlpEncode())
}

func (bm *BlockManager) BlockInfo(block *ethutil.Block) ethutil.BlockInfo {
	bi := ethutil.BlockInfo{}
	data, _ := ethutil.Config.Db.Get(append(block.Hash(), []byte("Info")...))
	bi.RlpDecode(data)

	return bi
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
	block.State().Update(block.Coinbase, string(ether.RlpEncode()))

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

// Contract evaluation is done here.
func (bm *BlockManager) ProcContract(tx *ethutil.Transaction, block *ethutil.Block, cb TxCallback) {
	// Instruction pointer
	pc := 0
	blockInfo := bm.BlockInfo(block)

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
		case oSTOP:
			break out
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
			bm.stack.Push(blockInfo.Number.String())
		case oBLK_DIFFICULTY:
			bm.stack.Push(block.Difficulty.String())
		case oBASEFEE:
			// e = 10^21
			e := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(21), big.NewInt(0))
			d := new(big.Rat)
			d.SetInt(block.Difficulty)
			c := new(big.Rat)
			c.SetFloat64(0.5)
			// d = diff / 0.5
			d.Quo(d, c)
			// base = floor(d)
			base.Div(d.Num(), d.Denom())

			x := new(big.Int)
			x.Div(e, base)

			// x = floor(10^21 / floor(diff^0.5))
			bm.stack.Push(x.String())
		case oSHA256, oRIPEMD160:
			// This is probably save
			// ceil(pop / 32)
			length := int(math.Ceil(float64(ethutil.Big(bm.stack.Pop()).Uint64()) / 32.0))
			// New buffer which will contain the concatenated popped items
			data := new(bytes.Buffer)
			for i := 0; i < length; i++ {
				// Encode the number to bytes and have it 32bytes long
				num := ethutil.NumberToBytes(ethutil.Big(bm.stack.Pop()).Bytes(), 256)
				data.WriteString(string(num))
			}

			if op == oSHA256 {
				bm.stack.Push(base.SetBytes(ethutil.Sha256Bin(data.Bytes())).String())
			} else {
				bm.stack.Push(base.SetBytes(ethutil.Ripemd160(data.Bytes())).String())
			}
		case oECMUL:
			y := bm.stack.Pop()
			x := bm.stack.Pop()
			//n := bm.stack.Pop()

			//if ethutil.Big(x).Cmp(ethutil.Big(y)) {
			data := new(bytes.Buffer)
			data.WriteString(x)
			data.WriteString(y)
			if secp256k1.VerifyPubkeyValidity(data.Bytes()) == 1 {
				// TODO
			} else {
				// Invalid, push infinity
				bm.stack.Push("0")
				bm.stack.Push("0")
			}
			//} else {
			//	// Invalid, push infinity
			//	bm.stack.Push("0")
			//	bm.stack.Push("0")
			//}

		case oECADD:
		case oECSIGN:
		case oECRECOVER:
		case oECVALID:
		case oSHA3:
		case oPUSH:
			// Get the next entry and pushes the value on the stack
			pc++
			bm.stack.Push(contract.State().Get(string(ethutil.NumberToBytes(uint64(pc), 32))))
		case oPOP:
			// Pop current value of the stack
			bm.stack.Pop()
		case oDUP:
		case oSWAP:
		case oMLOAD:
		case oMSTORE:
		case oSLOAD:
		case oSSTORE:
		case oJMP:
		case oJMPI:
		case oIND:
		case oEXTRO:
		case oBALANCE:
		case oMKTX:
		case oSUICIDE:
			/*
				case oLOAD:
					// Load instruction X on the stack
					i, _ := strconv.Atoi(bm.stack.Pop())
					bm.stack.Push(contract.State().Get(string(ethutil.NumberToBytes(uint64(i), 32))))
			*/
		}
		pc++
	}

	bm.stack.Print()
}
