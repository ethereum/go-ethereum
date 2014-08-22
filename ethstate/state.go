package ethstate

import (
	"math/big"

	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethtrie"
	"github.com/ethereum/eth-go/ethutil"
)

var statelogger = ethlog.NewLogger("STATE")

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
}

// Create a new state from a given trie
func New(trie *ethtrie.Trie) *State {
	return &State{Trie: trie, stateObjects: make(map[string]*StateObject), manifest: NewManifest()}
}

// Retrieve the balance from the given address or 0 if object not found
func (self *State) GetBalance(addr []byte) *big.Int {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		return stateObject.Balance
	}

	return ethutil.Big0
}

func (self *State) GetNonce(addr []byte) uint64 {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		return stateObject.Nonce
	}

	return 0
}

//
// Setting, updating & deleting state object methods
//

// Update the given state object and apply it to state trie
func (self *State) UpdateStateObject(stateObject *StateObject) {
	addr := stateObject.Address()

	ethutil.Config.Db.Put(ethcrypto.Sha3Bin(stateObject.Code), stateObject.Code)

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
	self.stateObjects[string(addr)] = stateObject

	return stateObject
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
		//s.UpdateStateObject(stateObject)

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
}

func (self *State) Update() {
	for _, stateObject := range self.stateObjects {
		if stateObject.remove {
			self.DeleteStateObject(stateObject)
		} else {
			stateObject.Sync()

			self.UpdateStateObject(stateObject)
		}
	}

	// FIXME trie delete is broken
	valid, t2 := ethtrie.ParanoiaCheck(self.Trie)
	if !valid {
		statelogger.Infof("Warn: PARANOIA: Different state root during copy %x vs %x\n", self.Trie.Root, t2.Root)

		self.Trie = t2
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
