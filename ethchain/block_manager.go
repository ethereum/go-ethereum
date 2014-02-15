package ethchain

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/obscuren/secp256k1-go"
	"log"
	"math"
	"math/big"
	"strconv"
	"sync"
	"time"
)

type BlockProcessor interface {
	ProcessBlock(block *Block)
}

func CalculateBlockReward(block *Block, uncleLength int) *big.Int {
	return BlockReward
}

type BlockManager struct {
	// Mutex for locking the block processor. Blocks can only be handled one at a time
	mutex sync.Mutex

	// The block chain :)
	bc *BlockChain

	// Stack for processing contracts
	stack *Stack
	// non-persistent key/value memory storage
	mem map[string]*big.Int

	TransactionPool *TxPool

	Pow PoW

	Speaker PublicSpeaker

	SecondaryBlockProcessor BlockProcessor
}

func AddTestNetFunds(block *Block) {
	for _, addr := range []string{
		"8a40bfaa73256b60764c1bf40675a99083efb075", // Gavin
		"93658b04240e4bd4046fd2d6d417d20f146f4b43", // Jeffrey
		"1e12515ce3e0f817a4ddef9ca55788a1d66bd2df", // Vit
		"80c01a26338f0d905e295fccb71fa9ea849ffa12", // Alex
	} {
		//log.Println("2^200 Wei to", addr)
		codedAddr, _ := hex.DecodeString(addr)
		addr := block.GetAddr(codedAddr)
		addr.Amount = ethutil.BigPow(2, 200)
		block.UpdateAddr(codedAddr, addr)
	}
}

func NewBlockManager(speaker PublicSpeaker) *BlockManager {
	bm := &BlockManager{
		//server: s,
		bc:      NewBlockChain(),
		stack:   NewStack(),
		mem:     make(map[string]*big.Int),
		Pow:     &EasyPow{},
		Speaker: speaker,
	}

	if bm.bc.CurrentBlock == nil {
		AddTestNetFunds(bm.bc.genesisBlock)
		// Prepare the genesis block
		bm.bc.Add(bm.bc.genesisBlock)

		log.Printf("Genesis: %x\n", bm.bc.genesisBlock.Hash())
		//log.Printf("root %x\n", bm.bc.genesisBlock.State().Root)
		//bm.bc.genesisBlock.PrintHash()
	}

	return bm
}

func (bm *BlockManager) BlockChain() *BlockChain {
	return bm.bc
}

func (bm *BlockManager) ApplyTransactions(block *Block, txs []*Transaction) {
	// Process each transaction/contract
	for _, tx := range txs {
		// If there's no recipient, it's a contract
		if tx.IsContract() {
			block.MakeContract(tx)
			bm.ProcessContract(tx, block)
		} else {
			bm.TransactionPool.ProcessTransaction(tx, block)
		}
	}
}

// Block processing and validating with a given (temporarily) state
func (bm *BlockManager) ProcessBlock(block *Block) error {
	// Processing a blocks may never happen simultaneously
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	hash := block.Hash()

	if bm.bc.HasBlock(hash) {
		return nil
	}

	/*
		if ethutil.Config.Debug {
			log.Printf("[BMGR] Processing block(%x)\n", hash)
		}
	*/

	// Check if we have the parent hash, if it isn't known we discard it
	// Reasons might be catching up or simply an invalid block
	if !bm.bc.HasBlock(block.PrevHash) && bm.bc.CurrentBlock != nil {
		return ParentError(block.PrevHash)
	}

	// Process the transactions on to current block
	bm.ApplyTransactions(bm.bc.CurrentBlock, block.Transactions())

	// Block validation
	if err := bm.ValidateBlock(block); err != nil {
		return err
	}

	// I'm not sure, but I don't know if there should be thrown
	// any errors at this time.
	if err := bm.AccumelateRewards(bm.bc.CurrentBlock, block); err != nil {
		return err
	}

	if !block.State().Cmp(bm.bc.CurrentBlock.State()) {
		//if block.State().Root != state.Root {
		return fmt.Errorf("Invalid merkle root. Expected %x, got %x", block.State().Root, bm.bc.CurrentBlock.State().Root)
	}

	// Calculate the new total difficulty and sync back to the db
	if bm.CalculateTD(block) {
		// Sync the current block's state to the database
		bm.bc.CurrentBlock.State().Sync()
		// Add the block to the chain
		bm.bc.Add(block)

		/*
			ethutil.Config.Db.Put(block.Hash(), block.RlpEncode())
			bm.bc.CurrentBlock = block
			bm.LastBlockHash = block.Hash()
			bm.writeBlockInfo(block)
		*/

		/*
			txs := bm.TransactionPool.Flush()
			var coded = []interface{}{}
			for _, tx := range txs {
				err := bm.TransactionPool.ValidateTransaction(tx)
				if err == nil {
					coded = append(coded, tx.RlpEncode())
				}
			}
		*/

		// Broadcast the valid block back to the wire
		//bm.Speaker.Broadcast(ethwire.MsgBlockTy, []interface{}{block.RlpValue().Value})

		// If there's a block processor present, pass in the block for further
		// processing
		if bm.SecondaryBlockProcessor != nil {
			bm.SecondaryBlockProcessor.ProcessBlock(block)
		}

		log.Printf("[BMGR] Added block #%d (%x)\n", block.BlockInfo().Number, block.Hash())
	} else {
		fmt.Println("total diff failed")
	}

	return nil
}

func (bm *BlockManager) CalculateTD(block *Block) bool {
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
		// Set the new total difficulty back to the block chain
		bm.bc.SetTotalDifficulty(td)

		/*
			if ethutil.Config.Debug {
				log.Println("[BMGR] TD(block) =", td)
			}
		*/

		return true
	}

	return false
}

// Validates the current block. Returns an error if the block was invalid,
// an uncle or anything that isn't on the current block chain.
// Validation validates easy over difficult (dagger takes longer time = difficult)
func (bm *BlockManager) ValidateBlock(block *Block) error {
	// TODO
	// 2. Check if the difficulty is correct

	// Check each uncle's previous hash. In order for it to be valid
	// is if it has the same block hash as the current
	previousBlock := bm.bc.GetBlock(block.PrevHash)
	for _, uncle := range block.Uncles {
		if bytes.Compare(uncle.PrevHash, previousBlock.PrevHash) != 0 {
			return ValidationError("Mismatch uncle's previous hash. Expected %x, got %x", previousBlock.PrevHash, uncle.PrevHash)
		}
	}

	diff := block.Time - bm.bc.CurrentBlock.Time
	if diff < 0 {
		return ValidationError("Block timestamp less then prev block %v", diff)
	}

	// New blocks must be within the 15 minute range of the last block.
	if diff > int64(15*time.Minute) {
		return ValidationError("Block is too far in the future of last block (> 15 minutes)")
	}

	// Verify the nonce of the block. Return an error if it's not valid
	if !bm.Pow.Verify(block.HashNoNonce(), block.Difficulty, block.Nonce) {
		return ValidationError("Block's nonce is invalid (= %v)", block.Nonce)
	}

	return nil
}

func (bm *BlockManager) AccumelateRewards(processor *Block, block *Block) error {
	// Get the coinbase rlp data
	addr := processor.GetAddr(block.Coinbase)
	// Reward amount of ether to the coinbase address
	addr.AddFee(CalculateBlockReward(block, len(block.Uncles)))

	processor.UpdateAddr(block.Coinbase, addr)

	// TODO Reward each uncle

	return nil
}

func (bm *BlockManager) Stop() {
	bm.bc.Stop()
}

func (bm *BlockManager) ProcessContract(tx *Transaction, block *Block) {
	// Recovering function in case the VM had any errors
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from VM execution with err =", r)
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
}

// Contract evaluation is done here.
func (bm *BlockManager) ProcContract(tx *Transaction, block *Block, cb TxCallback) {

	// Instruction pointer
	pc := 0
	blockInfo := bm.bc.BlockInfo(block)

	contract := block.GetContract(tx.Hash())
	if contract == nil {
		fmt.Println("Contract not found")
		return
	}

	Pow256 := ethutil.BigPow(2, 256)

	if ethutil.Config.Debug {
		fmt.Printf("#   op   arg\n")
	}
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

		if ethutil.Config.Debug {
			fmt.Printf("%-3d %-4s\n", pc, op.String())
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
			bm.stack.Push(base)
		case oSUB:
			x, y := bm.stack.Popn()
			// (x - y) % 2 ** 256
			base.Sub(x, y)
			base.Mod(base, Pow256)
			// Pop result back on the stack
			bm.stack.Push(base)
		case oMUL:
			x, y := bm.stack.Popn()
			// (x * y) % 2 ** 256
			base.Mul(x, y)
			base.Mod(base, Pow256)
			// Pop result back on the stack
			bm.stack.Push(base)
		case oDIV:
			x, y := bm.stack.Popn()
			// floor(x / y)
			base.Div(x, y)
			// Pop result back on the stack
			bm.stack.Push(base)
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
			bm.stack.Push(z)
		case oMOD:
			x, y := bm.stack.Popn()
			base.Mod(x, y)
			bm.stack.Push(base)
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
			bm.stack.Push(z)
		case oEXP:
			x, y := bm.stack.Popn()
			base.Exp(x, y, Pow256)

			bm.stack.Push(base)
		case oNEG:
			base.Sub(Pow256, bm.stack.Pop())
			bm.stack.Push(base)
		case oLT:
			x, y := bm.stack.Popn()
			// x < y
			if x.Cmp(y) < 0 {
				bm.stack.Push(ethutil.BigTrue)
			} else {
				bm.stack.Push(ethutil.BigFalse)
			}
		case oLE:
			x, y := bm.stack.Popn()
			// x <= y
			if x.Cmp(y) < 1 {
				bm.stack.Push(ethutil.BigTrue)
			} else {
				bm.stack.Push(ethutil.BigFalse)
			}
		case oGT:
			x, y := bm.stack.Popn()
			// x > y
			if x.Cmp(y) > 0 {
				bm.stack.Push(ethutil.BigTrue)
			} else {
				bm.stack.Push(ethutil.BigFalse)
			}
		case oGE:
			x, y := bm.stack.Popn()
			// x >= y
			if x.Cmp(y) > -1 {
				bm.stack.Push(ethutil.BigTrue)
			} else {
				bm.stack.Push(ethutil.BigFalse)
			}
		case oNOT:
			x, y := bm.stack.Popn()
			// x != y
			if x.Cmp(y) != 0 {
				bm.stack.Push(ethutil.BigTrue)
			} else {
				bm.stack.Push(ethutil.BigFalse)
			}

		// Please note  that the  following code contains some
		// ugly string casting. This will have to change to big
		// ints. TODO :)
		case oMYADDRESS:
			bm.stack.Push(ethutil.BigD(tx.Hash()))
		case oTXSENDER:
			bm.stack.Push(ethutil.BigD(tx.Sender()))
		case oTXVALUE:
			bm.stack.Push(tx.Value)
		case oTXDATAN:
			bm.stack.Push(big.NewInt(int64(len(tx.Data))))
		case oTXDATA:
			v := bm.stack.Pop()
			// v >= len(data)
			if v.Cmp(big.NewInt(int64(len(tx.Data)))) >= 0 {
				bm.stack.Push(ethutil.Big("0"))
			} else {
				bm.stack.Push(ethutil.Big(tx.Data[v.Uint64()]))
			}
		case oBLK_PREVHASH:
			bm.stack.Push(ethutil.BigD(block.PrevHash))
		case oBLK_COINBASE:
			bm.stack.Push(ethutil.BigD(block.Coinbase))
		case oBLK_TIMESTAMP:
			bm.stack.Push(big.NewInt(block.Time))
		case oBLK_NUMBER:
			bm.stack.Push(big.NewInt(int64(blockInfo.Number)))
		case oBLK_DIFFICULTY:
			bm.stack.Push(block.Difficulty)
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
			bm.stack.Push(x)
		case oSHA256, oSHA3, oRIPEMD160:
			// This is probably save
			// ceil(pop / 32)
			length := int(math.Ceil(float64(bm.stack.Pop().Uint64()) / 32.0))
			// New buffer which will contain the concatenated popped items
			data := new(bytes.Buffer)
			for i := 0; i < length; i++ {
				// Encode the number to bytes and have it 32bytes long
				num := ethutil.NumberToBytes(bm.stack.Pop().Bytes(), 256)
				data.WriteString(string(num))
			}

			if op == oSHA256 {
				bm.stack.Push(base.SetBytes(ethutil.Sha256Bin(data.Bytes())))
			} else if op == oSHA3 {
				bm.stack.Push(base.SetBytes(ethutil.Sha3Bin(data.Bytes())))
			} else {
				bm.stack.Push(base.SetBytes(ethutil.Ripemd160(data.Bytes())))
			}
		case oECMUL:
			y := bm.stack.Pop()
			x := bm.stack.Pop()
			//n := bm.stack.Pop()

			//if ethutil.Big(x).Cmp(ethutil.Big(y)) {
			data := new(bytes.Buffer)
			data.WriteString(x.String())
			data.WriteString(y.String())
			if secp256k1.VerifyPubkeyValidity(data.Bytes()) == 1 {
				// TODO
			} else {
				// Invalid, push infinity
				bm.stack.Push(ethutil.Big("0"))
				bm.stack.Push(ethutil.Big("0"))
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
		case oPUSH:
			pc++
			bm.stack.Push(bm.mem[strconv.Itoa(pc)])
		case oPOP:
			// Pop current value of the stack
			bm.stack.Pop()
		case oDUP:
			// Dup top stack
			x := bm.stack.Pop()
			bm.stack.Push(x)
			bm.stack.Push(x)
		case oSWAP:
			// Swap two top most values
			x, y := bm.stack.Popn()
			bm.stack.Push(y)
			bm.stack.Push(x)
		case oMLOAD:
			x := bm.stack.Pop()
			bm.stack.Push(bm.mem[x.String()])
		case oMSTORE:
			x, y := bm.stack.Popn()
			bm.mem[x.String()] = y
		case oSLOAD:
			// Load the value in storage and push it on the stack
			x := bm.stack.Pop()
			// decode the object as a big integer
			decoder := ethutil.NewValueFromBytes([]byte(contract.State().Get(x.String())))
			if !decoder.IsNil() {
				bm.stack.Push(decoder.BigInt())
			} else {
				bm.stack.Push(ethutil.BigFalse)
			}
		case oSSTORE:
			// Store Y at index X
			x, y := bm.stack.Popn()
			contract.State().Update(x.String(), string(ethutil.Encode(y)))
		case oJMP:
			x := int(bm.stack.Pop().Uint64())
			// Set pc to x - 1 (minus one so the incrementing at the end won't effect it)
			pc = x
			pc--
		case oJMPI:
			x := bm.stack.Pop()
			// Set pc to x if it's non zero
			if x.Cmp(ethutil.BigFalse) != 0 {
				pc = int(x.Uint64())
				pc--
			}
		case oIND:
			bm.stack.Push(big.NewInt(int64(pc)))
		case oEXTRO:
			memAddr := bm.stack.Pop()
			contractAddr := bm.stack.Pop().Bytes()

			// Push the contract's memory on to the stack
			bm.stack.Push(getContractMemory(block, contractAddr, memAddr))
		case oBALANCE:
			// Pushes the balance of the popped value on to the stack
			d := block.State().Get(bm.stack.Pop().String())
			ether := NewAddressFromData([]byte(d))
			bm.stack.Push(ether.Amount)
		case oMKTX:
			value, addr := bm.stack.Popn()
			from, length := bm.stack.Popn()

			j := 0
			dataItems := make([]string, int(length.Uint64()))
			for i := from.Uint64(); i < length.Uint64(); i++ {
				dataItems[j] = string(bm.mem[strconv.Itoa(int(i))].Bytes())
				j++
			}
			// TODO sign it?
			tx := NewTransaction(addr.Bytes(), value, dataItems)
			// Add the transaction to the tx pool
			bm.TransactionPool.QueueTransaction(tx)
		case oSUICIDE:
			//addr := bm.stack.Pop()
		}
		pc++
	}
}

// Returns an address from the specified contract's address
func getContractMemory(block *Block, contractAddr []byte, memAddr *big.Int) *big.Int {
	contract := block.GetContract(contractAddr)
	if contract == nil {
		log.Panicf("invalid contract addr %x", contractAddr)
	}
	val := contract.State().Get(memAddr.String())

	// decode the object as a big integer
	decoder := ethutil.NewValueFromBytes([]byte(val))
	if decoder.IsNil() {
		return ethutil.BigFalse
	}

	return decoder.BigInt()
}
