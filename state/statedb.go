package state

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/trie"
)

var statelogger = logger.NewLogger("STATE")

// StateDBs within the ethereum protocol are used to store anything
// within the merkle trie. StateDBs take care of caching and storing
// nested states. It's the general query interface to retrieve:
// * Contracts
// * Accounts
type StateDB struct {
	db   common.Database
	trie *trie.SecureTrie

	stateObjects map[string]*StateObject

	refund map[string]*big.Int

	logs Logs
}

// Create a new state from a given trie
func New(root []byte, db common.Database) *StateDB {
	trie := trie.NewSecure(common.CopyBytes(root), db)
	return &StateDB{db: db, trie: trie, stateObjects: make(map[string]*StateObject), refund: make(map[string]*big.Int)}
}

func (self *StateDB) PrintRoot() {
	self.trie.Trie.PrintRoot()
}

func (self *StateDB) EmptyLogs() {
	self.logs = nil
}

func (self *StateDB) AddLog(log Log) {
	self.logs = append(self.logs, log)
}

func (self *StateDB) Logs() Logs {
	return self.logs
}

func (self *StateDB) Refund(addr []byte, gas *big.Int) {
	if self.refund[string(addr)] == nil {
		self.refund[string(addr)] = new(big.Int)
	}
	self.refund[string(addr)].Add(self.refund[string(addr)], gas)
}

// Retrieve the balance from the given address or 0 if object not found
func (self *StateDB) GetBalance(addr []byte) *big.Int {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		return stateObject.balance
	}

	return common.Big0
}

func (self *StateDB) AddBalance(addr []byte, amount *big.Int) {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		stateObject.AddBalance(amount)
	}
}

func (self *StateDB) GetNonce(addr []byte) uint64 {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		return stateObject.nonce
	}

	return 0
}

func (self *StateDB) GetCode(addr []byte) []byte {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		return stateObject.code
	}

	return nil
}

func (self *StateDB) GetState(a, b []byte) []byte {
	stateObject := self.GetStateObject(a)
	if stateObject != nil {
		return stateObject.GetState(b).Bytes()
	}

	return nil
}

func (self *StateDB) SetNonce(addr []byte, nonce uint64) {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		stateObject.SetNonce(nonce)
	}
}

func (self *StateDB) SetCode(addr, code []byte) {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		stateObject.SetCode(code)
	}
}

func (self *StateDB) SetState(addr, key []byte, value interface{}) {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		stateObject.SetState(key, common.NewValue(value))
	}
}

func (self *StateDB) Delete(addr []byte) bool {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		stateObject.MarkForDeletion()
		stateObject.balance = new(big.Int)

		return true
	}

	return false
}

func (self *StateDB) IsDeleted(addr []byte) bool {
	stateObject := self.GetStateObject(addr)
	if stateObject != nil {
		return stateObject.remove
	}
	return false
}

//
// Setting, updating & deleting state object methods
//

// Update the given state object and apply it to state trie
func (self *StateDB) UpdateStateObject(stateObject *StateObject) {
	addr := stateObject.Address()

	if len(stateObject.CodeHash()) > 0 {
		self.db.Put(stateObject.CodeHash(), stateObject.code)
	}

	self.trie.Update(addr, stateObject.RlpEncode())
}

// Delete the given state object and delete it from the state trie
func (self *StateDB) DeleteStateObject(stateObject *StateObject) {
	self.trie.Delete(stateObject.Address())

	delete(self.stateObjects, string(stateObject.Address()))
}

// Retrieve a state object given my the address. Nil if not found
func (self *StateDB) GetStateObject(addr []byte) *StateObject {
	addr = common.Address(addr)

	stateObject := self.stateObjects[string(addr)]
	if stateObject != nil {
		return stateObject
	}

	data := self.trie.Get(addr)
	if len(data) == 0 {
		return nil
	}

	stateObject = NewStateObjectFromBytes(addr, []byte(data), self.db)
	self.SetStateObject(stateObject)

	return stateObject
}

func (self *StateDB) SetStateObject(object *StateObject) {
	self.stateObjects[string(object.address)] = object
}

// Retrieve a state object or create a new state object if nil
func (self *StateDB) GetOrNewStateObject(addr []byte) *StateObject {
	stateObject := self.GetStateObject(addr)
	if stateObject == nil {
		stateObject = self.NewStateObject(addr)
	}

	return stateObject
}

// Create a state object whether it exist in the trie or not
func (self *StateDB) NewStateObject(addr []byte) *StateObject {
	addr = common.Address(addr)

	statelogger.Debugf("(+) %x\n", addr)

	stateObject := NewStateObject(addr, self.db)
	self.stateObjects[string(addr)] = stateObject

	return stateObject
}

// Deprecated
func (self *StateDB) GetAccount(addr []byte) *StateObject {
	return self.GetOrNewStateObject(addr)
}

//
// Setting, copying of the state methods
//

func (s *StateDB) Cmp(other *StateDB) bool {
	return bytes.Equal(s.trie.Root(), other.trie.Root())
}

func (self *StateDB) Copy() *StateDB {
	state := New(nil, self.db)
	state.trie = self.trie.Copy()
	for k, stateObject := range self.stateObjects {
		state.stateObjects[k] = stateObject.Copy()
	}

	for addr, refund := range self.refund {
		state.refund[addr] = new(big.Int).Set(refund)
	}

	logs := make(Logs, len(self.logs))
	copy(logs, self.logs)
	state.logs = logs

	return state
}

func (self *StateDB) Set(state *StateDB) {
	self.trie = state.trie
	self.stateObjects = state.stateObjects

	self.refund = state.refund
	self.logs = state.logs
}

func (s *StateDB) Root() []byte {
	return s.trie.Root()
}

func (s *StateDB) Trie() *trie.SecureTrie {
	return s.trie
}

// Resets the trie and all siblings
func (s *StateDB) Reset() {
	s.trie.Reset()

	// Reset all nested states
	for _, stateObject := range s.stateObjects {
		if stateObject.State == nil {
			continue
		}

		stateObject.Reset()
	}

	s.Empty()
}

// Syncs the trie and all siblings
func (s *StateDB) Sync() {
	// Sync all nested states
	for _, stateObject := range s.stateObjects {
		if stateObject.State == nil {
			continue
		}

		stateObject.State.Sync()
	}

	s.trie.Commit()

	s.Empty()
}

func (self *StateDB) Empty() {
	self.stateObjects = make(map[string]*StateObject)
	self.refund = make(map[string]*big.Int)
}

func (self *StateDB) Refunds() map[string]*big.Int {
	return self.refund
}

func (self *StateDB) Update(gasUsed *big.Int) {
	self.refund = make(map[string]*big.Int)

	for _, stateObject := range self.stateObjects {
		if stateObject.dirty {
			if stateObject.remove {
				self.DeleteStateObject(stateObject)
			} else {
				stateObject.Sync()

				self.UpdateStateObject(stateObject)
			}
			stateObject.dirty = false
		}
	}
}

// Debug stuff
func (self *StateDB) CreateOutputForDiff() {
	for _, stateObject := range self.stateObjects {
		stateObject.CreateOutputForDiff()
	}
}
