package ethchain

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/eth-go/ethutil"
	_ "github.com/ethereum/eth-go/ethwire"
	"log"
	"math/big"
	"sync"
	"time"
)

type BlockProcessor interface {
	ProcessBlock(block *Block)
}

// TODO rename to state manager
type BlockManager struct {
	// Mutex for locking the block processor. Blocks can only be handled one at a time
	mutex sync.Mutex

	// The block chain :)
	bc *BlockChain

	// States for addresses. You can watch any address
	// at any given time
	addrStateStore *AddrStateStore

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
		"e6716f9544a56c530d868e4bfbacb172315bdead", // Jeffrey
		"1e12515ce3e0f817a4ddef9ca55788a1d66bd2df", // Vit
		"1a26338f0d905e295fccb71fa9ea849ffa12aaf4", // Alex
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
		bc:             NewBlockChain(),
		stack:          NewStack(),
		mem:            make(map[string]*big.Int),
		Pow:            &EasyPow{},
		Speaker:        speaker,
		addrStateStore: NewAddrStateStore(),
	}

	if bm.bc.CurrentBlock == nil {
		AddTestNetFunds(bm.bc.genesisBlock)

		bm.bc.genesisBlock.State().Sync()
		// Prepare the genesis block
		bm.bc.Add(bm.bc.genesisBlock)

		//log.Printf("root %x\n", bm.bc.genesisBlock.State().Root)
		//bm.bc.genesisBlock.PrintHash()
	}

	log.Printf("Last block: %x\n", bm.bc.CurrentBlock.Hash())

	return bm
}

// Watches any given address and puts it in the address state store
func (bm *BlockManager) WatchAddr(addr []byte) *AddressState {
	account := bm.bc.CurrentBlock.GetAddr(addr)

	return bm.addrStateStore.Add(addr, account)
}

func (bm *BlockManager) GetAddrState(addr []byte) *AddressState {
	account := bm.addrStateStore.Get(addr)
	if account == nil {
		a := bm.bc.CurrentBlock.GetAddr(addr)
		account = &AddressState{Nonce: a.Nonce, Account: a}
	}

	return account
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
		} else {
			if contract := block.GetContract(tx.Recipient); contract != nil {
				bm.ProcessContract(contract, tx, block)
			} else {
				err := bm.TransactionPool.ProcessTransaction(tx, block)
				if err != nil {
					ethutil.Config.Log.Infoln("[BMGR]", err)
				}
			}
		}
	}
}

// Block processing and validating with a given (temporarily) state
func (bm *BlockManager) ProcessBlock(block *Block) error {
	// Processing a blocks may never happen simultaneously
	bm.mutex.Lock()
	defer bm.mutex.Unlock()
	// Defer the Undo on the Trie. If the block processing happened
	// we don't want to undo but since undo only happens on dirty
	// nodes this won't happen because Commit would have been called
	// before that.
	defer bm.bc.CurrentBlock.Undo()

	hash := block.Hash()

	if bm.bc.HasBlock(hash) {
		return nil
	}

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
		return fmt.Errorf("Invalid merkle root. Expected %x, got %x", block.State().Root, bm.bc.CurrentBlock.State().Root)
	}

	// Calculate the new total difficulty and sync back to the db
	if bm.CalculateTD(block) {
		// Sync the current block's state to the database and cancelling out the deferred Undo
		bm.bc.CurrentBlock.Sync()

		// Broadcast the valid block back to the wire
		//bm.Speaker.Broadcast(ethwire.MsgBlockTy, []interface{}{block.Value().Val})

		// Add the block to the chain
		bm.bc.Add(block)

		// If there's a block processor present, pass in the block for further
		// processing
		if bm.SecondaryBlockProcessor != nil {
			bm.SecondaryBlockProcessor.ProcessBlock(block)
		}

		ethutil.Config.Log.Infof("[BMGR] Added block #%d (%x)\n", block.BlockInfo().Number, block.Hash())
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

func CalculateBlockReward(block *Block, uncleLength int) *big.Int {
	base := new(big.Int)
	for i := 0; i < uncleLength; i++ {
		base.Add(base, UncleInclusionReward)
	}
	return base.Add(base, BlockReward)
}

func CalculateUncleReward(block *Block) *big.Int {
	return UncleReward
}

func (bm *BlockManager) AccumelateRewards(processor *Block, block *Block) error {
	// Get the coinbase rlp data
	addr := processor.GetAddr(block.Coinbase)
	// Reward amount of ether to the coinbase address
	addr.AddFee(CalculateBlockReward(block, len(block.Uncles)))

	processor.UpdateAddr(block.Coinbase, addr)

	for _, uncle := range block.Uncles {
		uncleAddr := processor.GetAddr(uncle.Coinbase)
		uncleAddr.AddFee(CalculateUncleReward(uncle))

		processor.UpdateAddr(uncle.Coinbase, uncleAddr)
	}

	return nil
}

func (bm *BlockManager) Stop() {
	bm.bc.Stop()
}

func (bm *BlockManager) ProcessContract(contract *Contract, tx *Transaction, block *Block) {
	// Recovering function in case the VM had any errors
	/*
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from VM execution with err =", r)
			}
		}()
	*/

	vm := &Vm{}
	vm.Process(contract, NewState(block.state), RuntimeVars{
		address:     tx.Hash()[12:],
		blockNumber: block.BlockInfo().Number,
		sender:      tx.Sender(),
		prevHash:    block.PrevHash,
		coinbase:    block.Coinbase,
		time:        block.Time,
		diff:        block.Difficulty,
		txValue:     tx.Value,
		txData:      tx.Data,
	})
}
