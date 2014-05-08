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
	// Nested states
	states map[string]*State

	manifest *Manifest
}

// Create a new state from a given trie
func NewState(trie *ethutil.Trie) *State {
	return &State{trie: trie, states: make(map[string]*State), manifest: NewManifest()}
}

// Resets the trie and all siblings
func (s *State) Reset() {
	s.trie.Undo()

	// Reset all nested states
	for _, state := range s.states {
		state.Reset()
	}
}

// Syncs the trie and all siblings
func (s *State) Sync() {
	// Sync all nested states
	for _, state := range s.states {
		state.Sync()
	}

	s.trie.Sync()
}

// Purges the current trie.
func (s *State) Purge() int {
	return s.trie.NewIterator().Purge()
}

// XXX Deprecated
func (s *State) GetContract(addr []byte) *StateObject {
	data := s.trie.Get(string(addr))
	if data == "" {
		return nil
	}

	// build contract
	contract := NewStateObjectFromBytes(addr, []byte(data))

	// Check if there's a cached state for this contract
	cachedState := s.states[string(addr)]
	if cachedState != nil {
		contract.state = cachedState
	} else {
		// If it isn't cached, cache the state
		s.states[string(addr)] = contract.state
	}

	return contract
}

func (s *State) GetStateObject(addr []byte) *StateObject {
	data := s.trie.Get(string(addr))
	if data == "" {
		return nil
	}

	stateObject := NewStateObjectFromBytes(addr, []byte(data))

	// Check if there's a cached state for this contract
	cachedStateObject := s.states[string(addr)]
	if cachedStateObject != nil {
		stateObject.state = cachedStateObject
	} else {
		// If it isn't cached, cache the state
		s.states[string(addr)] = stateObject.state
	}

	return stateObject
}

func (s *State) SetStateObject(stateObject *StateObject) {
	s.states[string(stateObject.address)] = stateObject.state

	s.UpdateStateObject(stateObject)
}

func (s *State) GetAccount(addr []byte) (account *StateObject) {
	data := s.trie.Get(string(addr))
	if data == "" {
		account = NewAccount(addr, big.NewInt(0))
	} else {
		account = NewStateObjectFromBytes(addr, []byte(data))
	}

	return
}

func (s *State) Cmp(other *State) bool {
	return s.trie.Cmp(other.trie)
}

func (s *State) Copy() *State {
	return NewState(s.trie.Copy())
}

type ObjType byte

const (
	NilTy ObjType = iota
	AccountTy
	ContractTy

	UnknownTy
)

// Updates any given state object
func (s *State) UpdateStateObject(object *StateObject) {
	addr := object.Address()

	if object.state != nil {
		s.states[string(addr)] = object.state
	}

	s.trie.Update(string(addr), string(object.RlpEncode()))
	s.manifest.AddObjectChange(object)
}

func (s *State) Put(key, object []byte) {
	s.trie.Update(string(key), string(object))
}

func (s *State) Root() interface{} {
	return s.trie.Root
}
