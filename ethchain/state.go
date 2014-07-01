package ethchain

import (
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
	trie *ethutil.Trie

	stateObjects map[string]*StateObject

	manifest *Manifest
}

// Create a new state from a given trie
func NewState(trie *ethutil.Trie) *State {
	return &State{trie: trie, stateObjects: make(map[string]*StateObject), manifest: NewManifest()}
}

// Resets the trie and all siblings
func (s *State) Reset() {
	s.trie.Undo()

	// Reset all nested states
	for _, stateObject := range s.stateObjects {
		if stateObject.state == nil {
			continue
		}

		stateObject.state.Reset()
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
		self.UpdateStateObject(stateObject)
	}
}

// Purges the current trie.
func (s *State) Purge() int {
	return s.trie.NewIterator().Purge()
}

func (s *State) EachStorage(cb ethutil.EachCallback) {
	it := s.trie.NewIterator()
	it.Each(cb)
}

func (self *State) ResetStateObject(stateObject *StateObject) {
	delete(self.stateObjects, string(stateObject.Address()))

	stateObject.state.Reset()
}

func (self *State) UpdateStateObject(stateObject *StateObject) {
	addr := stateObject.Address()

	if self.stateObjects[string(addr)] == nil {
		self.stateObjects[string(addr)] = stateObject
	}

	ethutil.Config.Db.Put(ethutil.Sha3Bin(stateObject.Script()), stateObject.Script())

	self.trie.Update(string(addr), string(stateObject.RlpEncode()))

	self.manifest.AddObjectChange(stateObject)
}

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

func (self *State) GetOrNewStateObject(addr []byte) *StateObject {
	stateObject := self.GetStateObject(addr)
	if stateObject == nil {
		stateObject = self.NewStateObject(addr)
	}

	return stateObject
}

func (self *State) NewStateObject(addr []byte) *StateObject {
	//statelogger.Infof("(+) %x\n", addr)

	stateObject := NewStateObject(addr)
	self.stateObjects[string(addr)] = stateObject

	return stateObject
}

func (self *State) GetAccount(addr []byte) *StateObject {
	return self.GetOrNewStateObject(addr)
}

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
	//s.trie = snapshot.trie
	//s.stateObjects = snapshot.stateObjects
	self = state
}

func (s *State) Put(key, object []byte) {
	s.trie.Update(string(key), string(object))
}

func (s *State) Root() interface{} {
	return s.trie.Root
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
