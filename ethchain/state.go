package ethchain

import (
	"fmt"
	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethtrie"
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
)

// States within the ethereum protocol are used to store anything
// within the merkle trie. States take care of caching and storing
// nested states. It's the general query interface to retrieve:
// * Contracts
// * Accounts
type State struct {
	// The trie for this structure
	trie *ethtrie.Trie

	stateObjects map[string]*StateObject

	manifest *Manifest
}

// Create a new state from a given trie
func NewState(trie *ethtrie.Trie) *State {
	return &State{trie: trie, stateObjects: make(map[string]*StateObject), manifest: NewManifest()}
}

// Iterate over each storage address and yield callback
func (s *State) EachStorage(cb ethtrie.EachCallback) {
	it := s.trie.NewIterator()
	it.Each(cb)
}

// Retrieve the balance from the given address or 0 if object not found
func (self *State) GetBalance(addr []byte) *big.Int {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		return stateObject.Amount
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

	ethutil.Config.Db.Put(ethcrypto.Sha3Bin(stateObject.Script()), stateObject.Script())

	self.trie.Update(string(addr), string(stateObject.RlpEncode()))

	self.manifest.AddObjectChange(stateObject)
}

// Delete the given state object and delete it from the state trie
func (self *State) DeleteStateObject(stateObject *StateObject) {
	self.trie.Delete(string(stateObject.Address()))

	delete(self.stateObjects, string(stateObject.Address()))
}

// Retrieve a state object given my the address. Nil if not found
func (self *State) GetStateObject(addr []byte) *StateObject {
	stateObject := self.stateObjects[string(addr)]
	if stateObject != nil {
		return stateObject
	}

	data := self.trie.Get(string(addr))
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
	statelogger.Infof("(+) %x\n", addr)

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
	return s.trie.Cmp(other.trie)
}

func (self *State) Copy() *State {
	if self.trie != nil {
		state := NewState(self.trie.Copy())
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

	self.trie = state.trie
	self.stateObjects = state.stateObjects
}

func (s *State) Root() interface{} {
	return s.trie.Root
}

// Resets the trie and all siblings
func (s *State) Reset() {
	s.trie.Undo()

	// Reset all nested states
	for _, stateObject := range s.stateObjects {
		if stateObject.state == nil {
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
		s.UpdateStateObject(stateObject)

		if stateObject.state == nil {
			continue
		}

		stateObject.state.Sync()
	}

	s.trie.Sync()

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
	valid, t2 := ethtrie.ParanoiaCheck(self.trie)
	if !valid {
		self.trie = t2
	}
}

// Debug stuff
func (self *State) CreateOutputForDiff() {
	for addr, stateObject := range self.stateObjects {
		fmt.Printf("%x %x %x %x\n", addr, stateObject.state.Root(), stateObject.Amount.Bytes(), stateObject.Nonce)
		stateObject.state.EachStorage(func(addr string, value *ethutil.Value) {
			fmt.Printf("%x %x\n", addr, value.Bytes())
		})
	}
}

// Object manifest
//
// The object manifest is used to keep changes to the state so we can keep track of the changes
// that occurred during a state transitioning phase.
type Manifest struct {
	// XXX These will be handy in the future. Not important for now.
	objectAddresses  map[string]bool
	storageAddresses map[string]map[string]bool

	objectChanges  map[string]*StateObject
	storageChanges map[string]map[string]*big.Int
}

func NewManifest() *Manifest {
	m := &Manifest{objectAddresses: make(map[string]bool), storageAddresses: make(map[string]map[string]bool)}
	m.Reset()

	return m
}

func (m *Manifest) Reset() {
	m.objectChanges = make(map[string]*StateObject)
	m.storageChanges = make(map[string]map[string]*big.Int)
}

func (m *Manifest) AddObjectChange(stateObject *StateObject) {
	m.objectChanges[string(stateObject.Address())] = stateObject
}

func (m *Manifest) AddStorageChange(stateObject *StateObject, storageAddr []byte, storage *big.Int) {
	if m.storageChanges[string(stateObject.Address())] == nil {
		m.storageChanges[string(stateObject.Address())] = make(map[string]*big.Int)
	}

	m.storageChanges[string(stateObject.Address())][string(storageAddr)] = storage
}
