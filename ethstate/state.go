package ethstate

import (
	"math/big"

	"github.com/ethereum/go-ethereum/ethtrie"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
)

var statelogger = logger.NewLogger("STATE")

// States within the ethereum protocol are used to store anything
// within the merkle trie. States take care of caching and storing
// nested states. It's the general query interface to retrieve:
// * Contracts
// * Accounts
type State struct {
	// The trie for this structure
	Trie *ethtrie.Trie

	stateObjects map[string]*StateObject

	manifest *Manifest

	refund map[string]*big.Int

	logs Logs
}

// Create a new state from a given trie
func New(trie *ethtrie.Trie) *State {
	return &State{Trie: trie, stateObjects: make(map[string]*StateObject), manifest: NewManifest(), refund: make(map[string]*big.Int)}
}

func (self *State) EmptyLogs() {
	self.logs = nil
}

func (self *State) AddLog(log Log) {
	self.logs = append(self.logs, log)
}

func (self *State) Logs() Logs {
	return self.logs
}

// Retrieve the balance from the given address or 0 if object not found
func (self *State) GetBalance(addr []byte) *big.Int {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		return stateObject.balance
	}

	return ethutil.Big0
}

func (self *State) Refund(addr []byte, gas, price *big.Int) {
	amount := new(big.Int).Mul(gas, price)

	if self.refund[string(addr)] == nil {
		self.refund[string(addr)] = new(big.Int)
	}

	self.refund[string(addr)] = new(big.Int).Add(self.refund[string(addr)], amount)
}

func (self *State) AddBalance(addr []byte, amount *big.Int) {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		stateObject.AddBalance(amount)
	}
}

func (self *State) GetNonce(addr []byte) uint64 {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		return stateObject.Nonce
	}

	return 0
}

func (self *State) SetNonce(addr []byte, nonce uint64) {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		stateObject.Nonce = nonce
	}
}

func (self *State) GetCode(addr []byte) []byte {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		return stateObject.Code
	}

	return nil
}

func (self *State) GetState(a, b []byte) []byte {
	stateObject := self.GetStateObject(a)
	if stateObject != nil {
		return stateObject.GetState(b).Bytes()
	}

	return nil
}

func (self *State) SetState(addr, key []byte, value interface{}) {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		stateObject.SetState(key, ethutil.NewValue(value))
	}
}

func (self *State) Delete(addr []byte) bool {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		stateObject.MarkForDeletion()

		return true
	}

	return false
}

//
// Setting, updating & deleting state object methods
//

// Update the given state object and apply it to state trie
func (self *State) UpdateStateObject(stateObject *StateObject) {
	addr := stateObject.Address()

	if len(stateObject.CodeHash()) > 0 {
		ethutil.Config.Db.Put(stateObject.CodeHash(), stateObject.Code)
	}

	self.Trie.Update(string(addr), string(stateObject.RlpEncode()))
}

// Delete the given state object and delete it from the state trie
func (self *State) DeleteStateObject(stateObject *StateObject) {
	self.Trie.Delete(string(stateObject.Address()))

	delete(self.stateObjects, string(stateObject.Address()))
}

// Retrieve a state object given my the address. Nil if not found
func (self *State) GetStateObject(addr []byte) *StateObject {
	addr = ethutil.Address(addr)

	stateObject := self.stateObjects[string(addr)]
	if stateObject != nil {
		return stateObject
	}

	data := self.Trie.Get(string(addr))
	if len(data) == 0 {
		return nil
	}

	stateObject = NewStateObjectFromBytes(addr, []byte(data))
	self.SetStateObject(stateObject)

	return stateObject
}

func (self *State) SetStateObject(object *StateObject) {
	self.stateObjects[string(object.address)] = object
}

// Retrieve a state object or create a new state object if nil
func (self *State) GetOrNewStateObject(addr []byte) *StateObject {
	stateObject := self.GetStateObject(addr)
	if stateObject == nil {
		stateObject = self.NewStateObject(addr)
	}

	return stateObject
}

// Create a state object whether it exist in the trie or not
func (self *State) NewStateObject(addr []byte) *StateObject {
	addr = ethutil.Address(addr)

	statelogger.Debugf("(+) %x\n", addr)

	stateObject := NewStateObject(addr)
	self.stateObjects[string(addr)] = stateObject

	return stateObject
}

// Deprecated
func (self *State) GetAccount(addr []byte) *StateObject {
	return self.GetOrNewStateObject(addr)
}

//
// Setting, copying of the state methods
//

func (s *State) Cmp(other *State) bool {
	return s.Trie.Cmp(other.Trie)
}

func (self *State) Copy() *State {
	if self.Trie != nil {
		state := New(self.Trie.Copy())
		for k, stateObject := range self.stateObjects {
			state.stateObjects[k] = stateObject.Copy()
		}

		for addr, refund := range self.refund {
			state.refund[addr] = refund
		}

		logs := make(Logs, len(self.logs))
		copy(logs, self.logs)
		state.logs = logs

		return state
	}

	return nil
}

func (self *State) Set(state *State) {
	if state == nil {
		panic("Tried setting 'state' to nil through 'Set'")
	}

	self.Trie = state.Trie
	self.stateObjects = state.stateObjects
	self.refund = state.refund
	self.logs = state.logs
}

func (s *State) Root() interface{} {
	return s.Trie.Root
}

// Resets the trie and all siblings
func (s *State) Reset() {
	s.Trie.Undo()

	// Reset all nested states
	for _, stateObject := range s.stateObjects {
		if stateObject.State == nil {
			continue
		}

		//stateObject.state.Reset()
		stateObject.Reset()
	}

	s.Empty()
}

// Syncs the trie and all siblings
func (s *State) Sync() {
	// Sync all nested states
	for _, stateObject := range s.stateObjects {
		if stateObject.State == nil {
			continue
		}

		stateObject.State.Sync()
	}

	s.Trie.Sync()

	s.Empty()
}

func (self *State) Empty() {
	self.stateObjects = make(map[string]*StateObject)
	self.refund = make(map[string]*big.Int)
}

func (self *State) Update() {
	var deleted bool

	// Refund any gas that's left
	for addr, amount := range self.refund {
		self.GetStateObject([]byte(addr)).AddBalance(amount)
	}

	for _, stateObject := range self.stateObjects {
		if stateObject.remove {
			self.DeleteStateObject(stateObject)
			deleted = true
		} else {
			stateObject.Sync()

			self.UpdateStateObject(stateObject)
		}
	}

	// FIXME trie delete is broken
	if deleted {
		valid, t2 := ethtrie.ParanoiaCheck(self.Trie)
		if !valid {
			statelogger.Infof("Warn: PARANOIA: Different state root during copy %x vs %x\n", self.Trie.Root, t2.Root)

			self.Trie = t2
		}
	}
}

func (self *State) Manifest() *Manifest {
	return self.manifest
}

// Debug stuff
func (self *State) CreateOutputForDiff() {
	for _, stateObject := range self.stateObjects {
		stateObject.CreateOutputForDiff()
	}
}
