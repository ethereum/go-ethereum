package ethchain

import (
	"bytes"
	"fmt"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethwire"
	"math/big"
	"sync"
	"time"
)

type BlockProcessor interface {
	ProcessBlock(block *Block)
}

type EthManager interface {
	StateManager() *BlockManager
	BlockChain() *BlockChain
	TxPool() *TxPool
	Broadcast(msgType ethwire.MsgType, data []interface{})
}

// TODO rename to state manager
type BlockManager struct {
	// Mutex for locking the block processor. Blocks can only be handled one at a time
	mutex sync.Mutex

	// Canonical block chain
	bc *BlockChain
	// States for addresses. You can watch any address
	// at any given time
	addrStateStore *AddrStateStore

	// Stack for processing contracts
	stack *Stack
	// non-persistent key/value memory storage
	mem map[string]*big.Int

	Pow PoW

	Ethereum EthManager

	SecondaryBlockProcessor BlockProcessor

	// The managed states
	// Processor state. Anything processed will be applied to this
	// state
	procState *State
	// Comparative state it used for comparing and validating end
	// results
	compState *State
}

func NewBlockManager(ethereum EthManager) *BlockManager {
	bm := &BlockManager{
		stack:          NewStack(),
		mem:            make(map[string]*big.Int),
		Pow:            &EasyPow{},
		Ethereum:       ethereum,
		addrStateStore: NewAddrStateStore(),
		bc:             ethereum.BlockChain(),
	}

	return bm
}

func (bm *BlockManager) ProcState() *State {
	return bm.procState
}

// Watches any given address and puts it in the address state store
func (bm *BlockManager) WatchAddr(addr []byte) *AccountState {
	//FIXME account := bm.procState.GetAccount(addr)
	account := bm.bc.CurrentBlock.state.GetAccount(addr)

	return bm.addrStateStore.Add(addr, account)
}

func (bm *BlockManager) GetAddrState(addr []byte) *AccountState {
	account := bm.addrStateStore.Get(addr)
	if account == nil {
		a := bm.bc.CurrentBlock.state.GetAccount(addr)
		account = &AccountState{Nonce: a.Nonce, Account: a}
	}

	return account
}

func (bm *BlockManager) BlockChain() *BlockChain {
	return bm.bc
}

func (bm *BlockManager) MakeContract(tx *Transaction) {
	contract := MakeContract(tx, bm.procState)
	if contract != nil {
		bm.procState.states[string(tx.Hash()[12:])] = contract.state
	}
}

func (bm *BlockManager) ApplyTransactions(block *Block, txs []*Transaction) {
	// Process each transaction/contract
	for _, tx := range txs {
		// If there's no recipient, it's a contract
		if tx.IsContract() {
			//FIXME bm.MakeContract(tx)
			block.MakeContract(tx)
		} else {
			//FIXME if contract := procState.GetContract(tx.Recipient); contract != nil {
			if contract := block.state.GetContract(tx.Recipient); contract != nil {
				bm.ProcessContract(contract, tx, block)
			} else {
				err := bm.Ethereum.TxPool().ProcessTransaction(tx, block)
				if err != nil {
					ethutil.Config.Log.Infoln("[BMGR]", err)
				}
			}
		}
	}
}

// The prepare function, prepares the state manager for the next
// "ProcessBlock" action.
func (bm *BlockManager) Prepare(processer *State, comparative *State) {
	bm.compState = comparative
	bm.procState = processer
}

// Default prepare function
func (bm *BlockManager) PrepareDefault(block *Block) {
	bm.Prepare(bm.BlockChain().CurrentBlock.State(), block.State())
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

	// if !bm.compState.Cmp(bm.procState)
	if !block.state.Cmp(bm.bc.CurrentBlock.state) {
		return fmt.Errorf("Invalid merkle root. Expected %x, got %x", block.State().trie.Root, bm.bc.CurrentBlock.State().trie.Root)
		//FIXME return fmt.Errorf("Invalid merkle root. Expected %x, got %x", bm.compState.trie.Root, bm.procState.trie.Root)
	}

	// Calculate the new total difficulty and sync back to the db
	if bm.CalculateTD(block) {
		// Sync the current block's state to the database and cancelling out the deferred Undo
		bm.bc.CurrentBlock.Sync()
		//FIXME bm.procState.Sync()

		// Broadcast the valid block back to the wire
		//bm.Ethereum.Broadcast(ethwire.MsgBlockTy, []interface{}{block.Value().Val})

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
	addr := processor.state.GetAccount(block.Coinbase)
	//FIXME addr := proc.GetAccount(block.Coinbase)
	// Reward amount of ether to the coinbase address
	addr.AddFee(CalculateBlockReward(block, len(block.Uncles)))

	processor.state.UpdateAccount(block.Coinbase, addr)
	//FIXME proc.UpdateAccount(block.Coinbase, addr)

	for _, uncle := range block.Uncles {
		uncleAddr := processor.state.GetAccount(uncle.Coinbase)
		uncleAddr.AddFee(CalculateUncleReward(uncle))

		processor.state.UpdateAccount(uncle.Coinbase, uncleAddr)
		//FIXME proc.UpdateAccount(uncle.Coinbase, uncleAddr)
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
	//vm.Process(contract, bm.procState, RuntimeVars{
	vm.Process(contract, block.state, RuntimeVars{
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
