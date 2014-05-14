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
	StateManager() *StateManager
	BlockChain() *BlockChain
	TxPool() *TxPool
	Broadcast(msgType ethwire.MsgType, data []interface{})
	Reactor() *ethutil.ReactorEngine
	PeerCount() int
	IsMining() bool
	IsListening() bool
}

type StateManager struct {
	// Mutex for locking the block processor. Blocks can only be handled one at a time
	mutex sync.Mutex
	// Canonical block chain
	bc *BlockChain
	// Stack for processing contracts
	stack *Stack
	// non-persistent key/value memory storage
	mem map[string]*big.Int
	// Proof of work used for validating
	Pow PoW
	// The ethereum manager interface
	Ethereum EthManager
	// The managed states
	// Processor state. Anything processed will be applied to this
	// state
	procState *State
	// Comparative state it used for comparing and validating end
	// results
	compState *State
	// Transiently state. The trans state isn't ever saved, validated and
	// it could be used for setting account nonces without effecting
	// the main states.
	transState *State
	// Manifest for keeping changes regarding state objects. See `notify`
	// XXX Should we move the manifest to the State object. Benefit:
	// * All states can keep their own local changes
	//manifest *Manifest
}

func NewStateManager(ethereum EthManager) *StateManager {
	sm := &StateManager{
		stack:    NewStack(),
		mem:      make(map[string]*big.Int),
		Pow:      &EasyPow{},
		Ethereum: ethereum,
		bc:       ethereum.BlockChain(),
		//manifest: NewManifest(),
	}
	sm.procState = ethereum.BlockChain().CurrentBlock.State()
	sm.transState = sm.procState.Copy()

	return sm
}

func (sm *StateManager) ProcState() *State {
	return sm.procState
}

func (sm *StateManager) TransState() *State {
	return sm.transState
}

func (sm *StateManager) BlockChain() *BlockChain {
	return sm.bc
}

func (sm *StateManager) MakeContract(tx *Transaction) *StateObject {
	contract := MakeContract(tx, sm.procState)
	if contract != nil {
		sm.procState.states[string(tx.Hash()[12:])] = contract.state

		return contract
	}

	return nil
}

// Apply transactions uses the transaction passed to it and applies them onto
// the current processing state.
func (sm *StateManager) ApplyTransactions(block *Block, txs []*Transaction) {
	// Process each transaction/contract
	for _, tx := range txs {
		// If there's no recipient, it's a contract
		// Check if this is a contract creation traction and if so
		// create a contract of this tx.
		if tx.IsContract() {
			err := sm.Ethereum.TxPool().ProcessTransaction(tx, block, false)
			if err == nil {
				contract := sm.MakeContract(tx)
				if contract != nil {
					sm.EvalScript(contract.Init(), contract, tx, block)
				} else {
					ethutil.Config.Log.Infoln("[STATE] Unable to create contract")
				}
			} else {
				ethutil.Config.Log.Infoln("[STATE] contract create:", err)
			}
		} else {
			err := sm.Ethereum.TxPool().ProcessTransaction(tx, block, false)
			contract := sm.procState.GetContract(tx.Recipient)
			if err == nil && len(contract.Script()) > 0 {
				sm.EvalScript(contract.Script(), contract, tx, block)
			} else if err != nil {
				ethutil.Config.Log.Infoln("[STATE] process:", err)
			}
		}
	}
}

// The prepare function, prepares the state manager for the next
// "ProcessBlock" action.
func (sm *StateManager) Prepare(processor *State, comparative *State) {
	sm.compState = comparative
	sm.procState = processor
}

// Default prepare function
func (sm *StateManager) PrepareDefault(block *Block) {
	sm.Prepare(sm.BlockChain().CurrentBlock.State(), block.State())
}

// Block processing and validating with a given (temporarily) state
func (sm *StateManager) ProcessBlock(block *Block, dontReact bool) error {
	// Processing a blocks may never happen simultaneously
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	hash := block.Hash()

	if sm.bc.HasBlock(hash) {
		//fmt.Println("[STATE] We already have this block, ignoring")
		return nil
	}

	// Defer the Undo on the Trie. If the block processing happened
	// we don't want to undo but since undo only happens on dirty
	// nodes this won't happen because Commit would have been called
	// before that.
	defer sm.bc.CurrentBlock.Undo()

	// Check if we have the parent hash, if it isn't known we discard it
	// Reasons might be catching up or simply an invalid block
	if !sm.bc.HasBlock(block.PrevHash) && sm.bc.CurrentBlock != nil {
		return ParentError(block.PrevHash)
	}

	// Process the transactions on to current block
	sm.ApplyTransactions(sm.bc.CurrentBlock, block.Transactions())

	// Block validation
	if err := sm.ValidateBlock(block); err != nil {
		fmt.Println("[SM] Error validating block:", err)
		return err
	}

	// I'm not sure, but I don't know if there should be thrown
	// any errors at this time.
	if err := sm.AccumelateRewards(block); err != nil {
		fmt.Println("[SM] Error accumulating reward", err)
		return err
	}

	if !sm.compState.Cmp(sm.procState) {
		return fmt.Errorf("Invalid merkle root. Expected %x, got %x", sm.compState.trie.Root, sm.procState.trie.Root)
	}

	// Calculate the new total difficulty and sync back to the db
	if sm.CalculateTD(block) {
		// Sync the current block's state to the database and cancelling out the deferred Undo
		sm.procState.Sync()

		// Add the block to the chain
		sm.bc.Add(block)

		ethutil.Config.Log.Infof("[STATE] Added block #%d (%x)\n", block.BlockInfo().Number, block.Hash())
		if dontReact == false {
			sm.Ethereum.Reactor().Post("newBlock", block)

			sm.notifyChanges()

			sm.procState.manifest.Reset()
		}

		sm.Ethereum.Broadcast(ethwire.MsgBlockTy, []interface{}{block.Value().Val})

		sm.Ethereum.TxPool().RemoveInvalid(sm.procState)
	} else {
		fmt.Println("total diff failed")
	}

	return nil
}
func (sm *StateManager) CalculateTD(block *Block) bool {
	uncleDiff := new(big.Int)
	for _, uncle := range block.Uncles {
		uncleDiff = uncleDiff.Add(uncleDiff, uncle.Difficulty)
	}

	// TD(genesis_block) = 0 and TD(B) = TD(B.parent) + sum(u.difficulty for u in B.uncles) + B.difficulty
	td := new(big.Int)
	td = td.Add(sm.bc.TD, uncleDiff)
	td = td.Add(td, block.Difficulty)

	// The new TD will only be accepted if the new difficulty is
	// is greater than the previous.
	if td.Cmp(sm.bc.TD) > 0 {
		// Set the new total difficulty back to the block chain
		sm.bc.SetTotalDifficulty(td)

		return true
	}

	return false
}

// Validates the current block. Returns an error if the block was invalid,
// an uncle or anything that isn't on the current block chain.
// Validation validates easy over difficult (dagger takes longer time = difficult)
func (sm *StateManager) ValidateBlock(block *Block) error {
	// TODO
	// 2. Check if the difficulty is correct

	// Check each uncle's previous hash. In order for it to be valid
	// is if it has the same block hash as the current
	previousBlock := sm.bc.GetBlock(block.PrevHash)
	for _, uncle := range block.Uncles {
		if bytes.Compare(uncle.PrevHash, previousBlock.PrevHash) != 0 {
			return ValidationError("Mismatch uncle's previous hash. Expected %x, got %x", previousBlock.PrevHash, uncle.PrevHash)
		}
	}

	diff := block.Time - sm.bc.CurrentBlock.Time
	if diff < 0 {
		return ValidationError("Block timestamp less then prev block %v", diff)
	}

	// New blocks must be within the 15 minute range of the last block.
	if diff > int64(15*time.Minute) {
		return ValidationError("Block is too far in the future of last block (> 15 minutes)")
	}

	// Verify the nonce of the block. Return an error if it's not valid
	if !sm.Pow.Verify(block.HashNoNonce(), block.Difficulty, block.Nonce) {
		return ValidationError("Block's nonce is invalid (= %v)", ethutil.Hex(block.Nonce))
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

func (sm *StateManager) AccumelateRewards(block *Block) error {
	// Get the account associated with the coinbase
	account := sm.procState.GetAccount(block.Coinbase)
	// Reward amount of ether to the coinbase address
	account.AddAmount(CalculateBlockReward(block, len(block.Uncles)))

	addr := make([]byte, len(block.Coinbase))
	copy(addr, block.Coinbase)
	sm.procState.UpdateStateObject(account)

	for _, uncle := range block.Uncles {
		uncleAccount := sm.procState.GetAccount(uncle.Coinbase)
		uncleAccount.AddAmount(CalculateUncleReward(uncle))

		sm.procState.UpdateStateObject(uncleAccount)
	}

	return nil
}

func (sm *StateManager) Stop() {
	sm.bc.Stop()
}

func (sm *StateManager) EvalScript(script []byte, object *StateObject, tx *Transaction, block *Block) {
	account := sm.procState.GetAccount(tx.Sender())

	err := account.ConvertGas(tx.Gas, tx.GasPrice)
	if err != nil {
		ethutil.Config.Log.Debugln(err)
		return
	}

	closure := NewClosure(account, object, script, sm.procState, tx.Gas, tx.GasPrice)
	vm := NewVm(sm.procState, sm, RuntimeVars{
		Origin:      account.Address(),
		BlockNumber: block.BlockInfo().Number,
		PrevHash:    block.PrevHash,
		Coinbase:    block.Coinbase,
		Time:        block.Time,
		Diff:        block.Difficulty,
		Value:       tx.Value,
		//Price:       tx.GasPrice,
	})
	closure.Call(vm, tx.Data, nil)

	// Update the account (refunds)
	sm.procState.UpdateStateObject(account)
	sm.procState.UpdateStateObject(object)
}

func (sm *StateManager) notifyChanges() {
	for addr, stateObject := range sm.procState.manifest.objectChanges {
		sm.Ethereum.Reactor().Post("object:"+addr, stateObject)
	}

	for stateObjectAddr, mappedObjects := range sm.procState.manifest.storageChanges {
		for addr, value := range mappedObjects {
			sm.Ethereum.Reactor().Post("storage:"+stateObjectAddr+":"+addr, &StorageState{[]byte(stateObjectAddr), []byte(addr), value})
		}
	}
}
