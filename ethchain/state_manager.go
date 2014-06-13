package ethchain

import (
	"bytes"
	"container/list"
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

type Peer interface {
	Inbound() bool
	LastSend() time.Time
	LastPong() int64
	Host() []byte
	Port() uint16
	Version() string
	PingTime() string
	Connected() *int32
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
	Peers() *list.List
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
	// Transiently state. The trans state isn't ever saved, validated and
	// it could be used for setting account nonces without effecting
	// the main states.
	transState *State
	// Mining state. The mining state is used purely and solely by the mining
	// operation.
	miningState *State
}

func NewStateManager(ethereum EthManager) *StateManager {
	sm := &StateManager{
		stack:    NewStack(),
		mem:      make(map[string]*big.Int),
		Pow:      &EasyPow{},
		Ethereum: ethereum,
		bc:       ethereum.BlockChain(),
	}
	sm.transState = ethereum.BlockChain().CurrentBlock.State().Copy()
	sm.miningState = ethereum.BlockChain().CurrentBlock.State().Copy()

	return sm
}

func (sm *StateManager) CurrentState() *State {
	return sm.Ethereum.BlockChain().CurrentBlock.State()
}

func (sm *StateManager) TransState() *State {
	return sm.transState
}

func (sm *StateManager) MiningState() *State {
	return sm.miningState
}

func (sm *StateManager) NewMiningState() *State {
	sm.miningState = sm.Ethereum.BlockChain().CurrentBlock.State().Copy()

	return sm.miningState
}

func (sm *StateManager) BlockChain() *BlockChain {
	return sm.bc
}

func (sm *StateManager) MakeStateObject(state *State, tx *Transaction) *StateObject {
	contract := MakeContract(tx, state)
	if contract != nil {
		state.states[string(tx.CreationAddress())] = contract.state

		return contract
	}

	return nil
}

type StateTransition struct {
	coinbase []byte
	tx       *Transaction
	gas      *big.Int
	state    *State
	block    *Block

	cb, rec, sen *StateObject
}

func NewStateTransition(coinbase []byte, gas *big.Int, tx *Transaction, state *State, block *Block) *StateTransition {
	return &StateTransition{coinbase, tx, new(big.Int), state, block, nil, nil, nil}
}

func (self *StateTransition) Coinbase() *StateObject {
	if self.cb != nil {
		return self.cb
	}

	self.cb = self.state.GetAccount(self.coinbase)
	return self.cb
}
func (self *StateTransition) Sender() *StateObject {
	if self.sen != nil {
		return self.sen
	}

	self.sen = self.state.GetAccount(self.tx.Sender())
	return self.sen
}
func (self *StateTransition) Receiver() *StateObject {
	if self.tx.CreatesContract() {
		return nil
	}

	if self.rec != nil {
		return self.rec
	}

	self.rec = self.state.GetAccount(self.tx.Recipient)
	return self.rec
}

func (self *StateTransition) UseGas(amount *big.Int) error {
	if self.gas.Cmp(amount) < 0 {
		return OutOfGasError()
	}
	self.gas.Sub(self.gas, amount)

	return nil
}

func (self *StateTransition) AddGas(amount *big.Int) {
	self.gas.Add(self.gas, amount)
}

func (self *StateTransition) BuyGas() error {
	var err error

	sender := self.Sender()
	if sender.Amount.Cmp(self.tx.GasValue()) < 0 {
		return fmt.Errorf("Insufficient funds to pre-pay gas. Req %v, has %v", self.tx.GasValue(), self.tx.Value)
	}

	coinbase := self.Coinbase()
	err = coinbase.BuyGas(self.tx.Gas, self.tx.GasPrice)
	if err != nil {
		return err
	}
	self.state.UpdateStateObject(coinbase)

	self.AddGas(self.tx.Gas)
	sender.SubAmount(self.tx.GasValue())

	return nil
}

func (self *StateManager) TransitionState(st *StateTransition) (err error) {
	//snapshot := st.state.Snapshot()

	defer func() {
		if r := recover(); r != nil {
			ethutil.Config.Log.Infoln(r)
			err = fmt.Errorf("%v", r)
		}
	}()

	var (
		tx       = st.tx
		sender   = st.Sender()
		receiver *StateObject
	)

	if sender.Nonce != tx.Nonce {
		return NonceError(tx.Nonce, sender.Nonce)
	}

	sender.Nonce += 1
	defer func() {
		// Notify all subscribers
		self.Ethereum.Reactor().Post("newTx:post", tx)
	}()

	if err = st.BuyGas(); err != nil {
		return err
	}

	receiver = st.Receiver()

	if err = st.UseGas(GasTx); err != nil {
		return err
	}

	dataPrice := big.NewInt(int64(len(tx.Data)))
	dataPrice.Mul(dataPrice, GasData)
	if err = st.UseGas(dataPrice); err != nil {
		return err
	}

	if receiver == nil { // Contract
		receiver = self.MakeStateObject(st.state, tx)
		if receiver == nil {
			return fmt.Errorf("ERR. Unable to create contract with transaction %v", tx)
		}
	}

	if err = self.transferValue(st, sender, receiver); err != nil {
		return err
	}

	if tx.CreatesContract() {
		fmt.Println(Disassemble(receiver.Init()))
		// Evaluate the initialization script
		// and use the return value as the
		// script section for the state object.
		//script, gas, err = sm.Eval(state, contract.Init(), contract, tx, block)
		code, err := self.Eval(st, receiver.Init(), receiver)
		if err != nil {
			return fmt.Errorf("Error during init script run %v", err)
		}

		receiver.script = code
	}

	st.state.UpdateStateObject(sender)
	st.state.UpdateStateObject(receiver)

	return nil
}

func (self *StateManager) transferValue(st *StateTransition, sender, receiver *StateObject) error {
	if sender.Amount.Cmp(st.tx.Value) < 0 {
		return fmt.Errorf("Insufficient funds to transfer value. Req %v, has %v", st.tx.Value, sender.Amount)
	}

	// Subtract the amount from the senders account
	sender.SubAmount(st.tx.Value)
	// Add the amount to receivers account which should conclude this transaction
	receiver.AddAmount(st.tx.Value)

	ethutil.Config.Log.Debugf("%x => %x (%v) %x\n", sender.Address()[:4], receiver.Address()[:4], st.tx.Value, st.tx.Hash())

	return nil
}

func (self *StateManager) ProcessTransactions(coinbase []byte, state *State, block, parent *Block, txs Transactions) (Receipts, Transactions, Transactions, error) {
	var (
		receipts           Receipts
		handled, unhandled Transactions
		totalUsedGas       = big.NewInt(0)
		err                error
	)

done:
	for i, tx := range txs {
		txGas := new(big.Int).Set(tx.Gas)
		st := NewStateTransition(coinbase, tx.Gas, tx, state, block)
		err = self.TransitionState(st)
		if err != nil {
			switch {
			case IsNonceErr(err):
				err = nil // ignore error
				continue
			case IsGasLimitErr(err):
				unhandled = txs[i:]

				break done
			default:
				ethutil.Config.Log.Infoln(err)
			}
		}

		txGas.Sub(txGas, st.gas)
		accumelative := new(big.Int).Set(totalUsedGas.Add(totalUsedGas, txGas))
		receipt := &Receipt{tx, ethutil.CopyBytes(state.Root().([]byte)), accumelative}

		receipts = append(receipts, receipt)
		handled = append(handled, tx)
	}

	fmt.Println("################# MADE\n", receipts, "\n############################")

	parent.GasUsed = totalUsedGas

	return receipts, handled, unhandled, err
}

func (self *StateManager) Eval(st *StateTransition, script []byte, context *StateObject) (ret []byte, err error) {
	var (
		tx        = st.tx
		block     = st.block
		initiator = st.Sender()
	)

	closure := NewClosure(initiator, context, script, st.state, st.gas, tx.GasPrice)
	vm := NewVm(st.state, self, RuntimeVars{
		Origin:      initiator.Address(),
		BlockNumber: block.BlockInfo().Number,
		PrevHash:    block.PrevHash,
		Coinbase:    block.Coinbase,
		Time:        block.Time,
		Diff:        block.Difficulty,
		Value:       tx.Value,
	})
	ret, _, err = closure.Call(vm, tx.Data, nil)

	return
}

func (sm *StateManager) Process(block *Block, dontReact bool) error {
	if !sm.bc.HasBlock(block.PrevHash) {
		return ParentError(block.PrevHash)
	}

	parent := sm.bc.GetBlock(block.PrevHash)

	return sm.ProcessBlock(parent.State(), parent, block, dontReact)

}

// Block processing and validating with a given (temporarily) state
func (sm *StateManager) ProcessBlock(state *State, parent, block *Block, dontReact bool) error {
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
	defer state.Reset()

	// Check if we have the parent hash, if it isn't known we discard it
	// Reasons might be catching up or simply an invalid block
	if !sm.bc.HasBlock(block.PrevHash) && sm.bc.CurrentBlock != nil {
		return ParentError(block.PrevHash)
	}
	fmt.Println(block.Receipts())

	// Process the transactions on to current block
	//sm.ApplyTransactions(block.Coinbase, state, parent, block.Transactions())
	sm.ProcessTransactions(block.Coinbase, state, block, parent, block.Transactions())

	// Block validation
	if err := sm.ValidateBlock(block); err != nil {
		fmt.Println("[SM] Error validating block:", err)
		return err
	}

	// I'm not sure, but I don't know if there should be thrown
	// any errors at this time.
	if err := sm.AccumelateRewards(state, block); err != nil {
		fmt.Println("[SM] Error accumulating reward", err)
		return err
	}

	//if !sm.compState.Cmp(state) {
	if !block.State().Cmp(state) {
		return fmt.Errorf("Invalid merkle root.\nrec: %x\nis:  %x", block.State().trie.Root, state.trie.Root)
	}

	// Calculate the new total difficulty and sync back to the db
	if sm.CalculateTD(block) {
		// Sync the current block's state to the database and cancelling out the deferred Undo
		state.Sync()

		// Add the block to the chain
		sm.bc.Add(block)
		sm.notifyChanges(state)

		ethutil.Config.Log.Infof("[STATE] Added block #%d (%x)\n", block.Number, block.Hash())
		if dontReact == false {
			sm.Ethereum.Reactor().Post("newBlock", block)

			state.manifest.Reset()
		}

		sm.Ethereum.Broadcast(ethwire.MsgBlockTy, []interface{}{block.Value().Val})

		sm.Ethereum.TxPool().RemoveInvalid(state)
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

func (sm *StateManager) AccumelateRewards(state *State, block *Block) error {
	// Get the account associated with the coinbase
	account := state.GetAccount(block.Coinbase)
	// Reward amount of ether to the coinbase address
	account.AddAmount(CalculateBlockReward(block, len(block.Uncles)))

	addr := make([]byte, len(block.Coinbase))
	copy(addr, block.Coinbase)
	state.UpdateStateObject(account)

	for _, uncle := range block.Uncles {
		uncleAccount := state.GetAccount(uncle.Coinbase)
		uncleAccount.AddAmount(CalculateUncleReward(uncle))

		state.UpdateStateObject(uncleAccount)
	}

	return nil
}

func (sm *StateManager) Stop() {
	sm.bc.Stop()
}

func (sm *StateManager) notifyChanges(state *State) {
	for addr, stateObject := range state.manifest.objectChanges {
		sm.Ethereum.Reactor().Post("object:"+addr, stateObject)
	}

	for stateObjectAddr, mappedObjects := range state.manifest.storageChanges {
		for addr, value := range mappedObjects {
			sm.Ethereum.Reactor().Post("storage:"+stateObjectAddr+":"+addr, &StorageState{[]byte(stateObjectAddr), []byte(addr), value})
		}
	}
}
