package state

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

var StartingNonce uint64 // StartingNonce determines the nonce used when a new account is initialised.

// State keeps a Log of changes that occured during the sessions such as state object changes, Refunds and Logs. It also keeps a reference to it's parent state to retrieve data if anything is missing from this snapshot. This will allow us to copy on demand.
type State struct {
	Db   ethdb.Database
	Trie *trie.SecureTrie

	parent *State

	StateObjects      map[common.Address]*StateObject
	ownedStateObjects map[common.Address]bool
	refund            *big.Int

	logIdx uint
	logs   []*vm.Log

	interInfo        intermediateInfo
	MarkedTransition bool // marks the transition between transactions
}

type intermediateInfo struct {
	txHash, blockHash common.Hash
	txIdx             uint
}

// Create a new state from a given trie
func New(root common.Hash, db ethdb.Database) (*State, error) {
	tr, err := trie.NewSecure(root, db)
	if err != nil {
		return nil, err
	}

	return &State{
		Db:                db,
		Trie:              tr,
		StateObjects:      make(map[common.Address]*StateObject),
		ownedStateObjects: make(map[common.Address]bool),
		refund:            new(big.Int),
	}, nil
}

func (s *State) PrepareIntermediate(txHash, blockHash common.Hash, txIdx int) {
	if s.parent != nil {
		s.parent.MarkedTransition = true
	}
	s.interInfo = intermediateInfo{txHash: txHash, blockHash: blockHash, txIdx: uint(txIdx)}
	s.refund = new(big.Int)
}

// Flatten flattens one's own state
func (s *State) Flatten() {
	*s = *Flatten(s)
}

// Read fetches the state object with the given address by first checking its own cache of state objects. If that didn't result in an object it will attempt to check it's line of ancestors.
func (s *State) Read(address common.Address) (object *StateObject, inCache bool) {
	stateObject := s.StateObjects[address]
	if stateObject == nil && s.parent != nil {
		stateObject, _ = s.parent.Read(address)
	} else if stateObject != nil {
		inCache = true
	}

	if stateObject != nil {
		if stateObject.deleted {
			return nil, false
		}
		return stateObject, inCache
	}

	data := s.Trie.Get(address[:])
	if len(data) == 0 {
		return nil, false
	}
	var err error
	stateObject, err = DecodeObject(address, s.Db, data)
	if err != nil {
		glog.Errorf("can't decode object at %x: %v", address[:], err)
		return nil, false
	}

	return stateObject, false
}

func (s *State) GetOrNewStateObject(address common.Address) *StateObject {
	stateObject, inCache := s.Read(address)
	if stateObject != nil {
		if !inCache || !s.ownedStateObjects[address] {
			stateObject = stateObject.Copy()
			s.StateObjects[address] = stateObject
			s.ownedStateObjects[address] = true
		}
		return stateObject
	}

	if stateObject == nil || stateObject.deleted {
		stateObject = NewStateObject(address, s.Db)
		stateObject.SetNonce(StartingNonce)

		s.StateObjects[address] = stateObject
		s.ownedStateObjects[address] = true
	}
	return stateObject
}

/*
func (s *State) GetOrNewStateObject(address common.Address) *StateObject {
	stateObject := s.GetStateObject(address)
	if stateObject == nil || stateObject.deleted {
		stateObject = NewStateObject(address, s.Db)
		stateObject.SetNonce(StartingNonce)

		s.StateObjects[address] = stateObject
	}

	return stateObject
}
*/

func (s *State) GetStateObject(address common.Address) *StateObject {
	account, inCache := s.Read(address)
	if account != nil {
		if !inCache {
			s.StateObjects[address] = account
		}
		return account
	}
	return nil
}

func (s *State) CreateStateObject(address common.Address) *StateObject {
	// Get previous (if any)
	so := s.GetStateObject(address)
	// Create a new one
	stateObject := NewStateObject(address, s.Db)
	stateObject.SetNonce(StartingNonce)

	s.StateObjects[address] = stateObject

	// If it existed set the balance to the new account
	if so != nil {
		stateObject.balance = so.balance
	}

	return stateObject
}

func (s *State) SubBalance(address common.Address, amount *big.Int) {
	stateObject := s.GetOrNewStateObject(address)
	if stateObject != nil {
		stateObject.SubBalance(amount)
	}
}

func (s *State) AddBalance(address common.Address, amount *big.Int) {
	stateObject := s.GetOrNewStateObject(address)
	if stateObject != nil {
		stateObject.AddBalance(amount)
	}
}

func (s *State) GetBalance(address common.Address) *big.Int {
	stateObject := s.GetStateObject(address)
	if stateObject != nil {
		return stateObject.balance
	}

	return common.Big0
}

func (s *State) GetNonce(address common.Address) uint64 {
	stateObject := s.GetStateObject(address)
	if stateObject != nil {
		return stateObject.nonce
	}

	return StartingNonce
}

func (s *State) SetNonce(address common.Address, nonce uint64) {
	stateObject := s.GetOrNewStateObject(address)
	if stateObject != nil {
		stateObject.SetNonce(nonce)
	}
}

func (s *State) GetCode(address common.Address) []byte {
	stateObject := s.GetStateObject(address)
	if stateObject != nil {
		return stateObject.code
	}

	return nil
}
func (s *State) SetCode(address common.Address, code []byte) {
	stateObject := s.GetOrNewStateObject(address)
	if stateObject != nil {
		stateObject.SetCode(code)
	}
}

func (s *State) AddRefund(gas *big.Int) {
	s.refund.Add(s.refund, gas)
}

func (s *State) GetRefund() *big.Int {
	var refund *big.Int
	if s.MarkedTransition {
		return new(big.Int)
	} else if s.parent == nil {
		refund = new(big.Int)
	} else {
		refund = s.parent.GetRefund()
	}
	return refund.Add(refund, s.refund)
}

func (s *State) GetState(address common.Address, stateAddress common.Hash) common.Hash {
	stateObject := s.GetStateObject(address)
	if stateObject != nil {
		return stateObject.GetState(stateAddress)
	}

	return common.Hash{}
}
func (s *State) SetState(address common.Address, stateAddress common.Hash, value common.Hash) {
	stateObject := s.GetOrNewStateObject(address)
	if stateObject != nil {
		stateObject.SetState(stateAddress, value)
	}
}

func (s *State) Delete(address common.Address) bool {
	stateObject := s.GetStateObject(address)
	if stateObject != nil {
		stateObject.MarkForDeletion()
		stateObject.balance = new(big.Int)

		return true
	}

	return false
}

func (s *State) HasAccount(addr common.Address) bool {
	return s.GetStateObject(addr) != nil
}

func (s *State) Exist(address common.Address) bool {
	return s.GetStateObject(address) != nil
}

func (s *State) IsDeleted(address common.Address) bool {
	stateObject := s.GetStateObject(address)
	if stateObject != nil {
		return stateObject.remove
	}
	return false
}

func (s *State) GetAccount(address common.Address) vm.Account {
	return s.GetStateObject(address)
}

func (s *State) CreateAccount(address common.Address) vm.Account {
	return s.CreateStateObject(address)
}

func (s *State) AddLog(log *vm.Log) {
	log.TxIndex = s.interInfo.txIdx
	log.TxHash = s.interInfo.txHash
	log.BlockHash = s.interInfo.blockHash
	log.Index = s.logIdx

	s.logs = append(s.logs, log)

	s.logIdx++
}

// Logs returns the logs of it's entire ancestory chain or until
// the marked transition is found (state between transactions).
func (s *State) Logs() []*vm.Log {
	var logs []*vm.Log
	if s.MarkedTransition {
		return nil
	} else if s.parent != nil {
		logs = s.parent.Logs()
	}
	return append(logs, s.logs...)
}

func (s *State) DeleteStateObject(stateObject *StateObject) {
	stateObject.deleted = true

	addr := stateObject.Address()
	s.Trie.Delete(addr[:])
}

func (s *State) UpdateStateObject(stateObject *StateObject) {
	addr := stateObject.Address()
	data, err := rlp.EncodeToBytes(stateObject)
	if err != nil {
		panic(fmt.Errorf("can't encode object at %x: %v", addr[:], err))
	}
	s.Trie.Update(addr[:], data)
}

func (s *State) Reset(root common.Hash) error {
	fmt.Println("reset")
	var (
		err error
		tr  = s.Trie
	)
	if s.Trie.Hash() != root {
		if tr, err = trie.NewSecure(root, s.Db); err != nil {
			return err
		}
	}
	*s = State{
		Db:                s.Db,
		Trie:              tr,
		StateObjects:      s.StateObjects,
		ownedStateObjects: s.ownedStateObjects,
		//StateObjects:      make(map[common.Address]*StateObject),
		//ownedStateObjects: make(map[common.Address]bool),
		refund: new(big.Int),
	}
	return nil
}

// DeleteSuicides flags the suicided objects for deletion so that it
// won't be referenced again when called / queried up on.
//
// DeleteSuicides should not be used for consensus related updates
// under any circumstances.
func (s *State) DeleteSuicides() {
	// Delete parents first
	if s.parent != nil {
		s.DeleteSuicides()
	}

	// Reset refund so that any used-gas calculations can use
	// this method.
	s.refund = new(big.Int)
	for _, stateObject := range s.StateObjects {
		if stateObject.dirty {
			// If the object has been removed by a suicide
			// flag the object as deleted.
			if stateObject.remove {
				stateObject.deleted = true
			}
			stateObject.dirty = false
		}
	}
}

// Fork preserve the given state and returns a handle to a new modifiable state
// that does not affect the preserved state.
func Fork(parent *State) *State {
	return &State{
		Db:   parent.Db,
		Trie: parent.Trie,

		parent:            parent,
		StateObjects:      make(map[common.Address]*StateObject),
		ownedStateObjects: make(map[common.Address]bool),
		refund:            new(big.Int),
		logIdx:            parent.logIdx,
		logs:              nil,
	}
}

func (s *State) Set(o *State) {
	*s = *o
}

// Flatten flattens the state in to a single new state, including all changes of all ancestors.
func Flatten(s *State) *State {
	// first commit the parent so we can overwrite changes
	// later.
	var flattenedState *State
	if s.parent != nil {
		flattenedState = Flatten(s.parent)
	} else {
		flattenedState = &State{
			Db:                s.Db,
			Trie:              s.Trie,
			refund:            new(big.Int),
			StateObjects:      make(map[common.Address]*StateObject),
			ownedStateObjects: make(map[common.Address]bool),
		}
	}

	for address, object := range s.StateObjects {
		flattenedState.StateObjects[address] = object
		if s.ownedStateObjects[address] {
			flattenedState.ownedStateObjects[address] = true
		}
	}

	flattenedState.logs = append(flattenedState.logs, s.logs...)
	flattenedState.refund.Add(flattenedState.refund, s.refund)

	return flattenedState
}

func (s *State) String() string {
	return fmt.Sprintf("objects: %d owned: %d", len(s.StateObjects), len(s.ownedStateObjects))
}

func IntermediateRoot(state *State) common.Hash {
	s := Flatten(state)

	for _, stateObject := range s.StateObjects {
		if stateObject.dirty {
			if stateObject.remove {
				s.DeleteStateObject(stateObject)
			} else {
				stateObject.Update()
				s.UpdateStateObject(stateObject)
			}
			stateObject.dirty = false
		}
	}
	return s.Trie.Hash()
}
