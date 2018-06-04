// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"bytes"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

var emptyCodeHash = crypto.Keccak256(nil)

type Code []byte

func (c Code) String() string {
	return string(c) //strings.Join(Disassemble(c), " ")
}

type Storage map[common.Hash]common.Hash

func (s Storage) String() (str string) {
	for key, value := range s {
		str += fmt.Sprintf("%X : %X\n", key, value)
	}

	return
}

func (s Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range s {
		cpy[key] = value
	}

	return cpy
}

// stateObject represents an Ethereum account which is being modified.
//
// The usage pattern is as follows:
// First you need to obtain a state object.
// Account values can be accessed and modified through the object.
// Finally, call CommitTrie to write the modified storage trie into a database.
type stateObject struct {
	address  common.Address
	addrHash common.Hash // hash of ethereum address of the account
	data     Account
	db       *StateDB

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error

	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access
	code Code // contract bytecode, which gets set when code is loaded

	cachedStorage Storage // Storage entry cache to avoid duplicate reads
	dirtyStorage  Storage // Storage entries that need to be flushed to disk

	// Cache flags.
	// When an object is marked suicided it will be delete from the trie
	// during the "update" phase of the state transition.
	dirtyCode bool // true if the code was updated
	suicided  bool
	deleted   bool
}

// empty returns whether the account is considered empty.
func (s *stateObject) empty() bool {
	return s.data.Nonce == 0 && s.data.Balance.Sign() == 0 && bytes.Equal(s.data.CodeHash, emptyCodeHash)
}

// Account is the Ethereum consensus representation of accounts.
// These objects are stored in the main account trie.
type Account struct {
	Nonce    uint64
	Balance  *big.Int
	Root     common.Hash // merkle root of the storage trie
	CodeHash []byte
}

// newObject creates a state object.
func newObject(db *StateDB, address common.Address, data Account) *stateObject {
	if data.Balance == nil {
		data.Balance = new(big.Int)
	}
	if data.CodeHash == nil {
		data.CodeHash = emptyCodeHash
	}
	return &stateObject{
		db:            db,
		address:       address,
		addrHash:      crypto.Keccak256Hash(address[:]),
		data:          data,
		cachedStorage: make(Storage),
		dirtyStorage:  make(Storage),
	}
}

// EncodeRLP implements rlp.Encoder.
func (c *stateObject) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, c.data)
}

// setError remembers the first non-nil error it is called with.
func (object *stateObject) setError(err error) {
	if object.dbErr == nil {
		object.dbErr = err
	}
}

func (object *stateObject) markSuicided() {
	object.suicided = true
}

func (c *stateObject) touch() {
	c.db.journal.append(touchChange{
		account: &c.address,
	})
	if c.address == ripemd {
		// Explicitly put it in the dirty-cache, which is otherwise generated from
		// flattened journals.
		c.db.journal.dirty(c.address)
	}
}

func (c *stateObject) getTrie(db Database) Trie {
	if c.trie == nil {
		var err error
		c.trie, err = db.OpenStorageTrie(c.addrHash, c.data.Root)
		if err != nil {
			c.trie, _ = db.OpenStorageTrie(c.addrHash, common.Hash{})
			c.setError(fmt.Errorf("can't create storage trie: %v", err))
		}
	}
	return c.trie
}

// GetState returns a value in account storage.
func (object *stateObject) GetState(db Database, key common.Hash) common.Hash {
	value, exists := object.cachedStorage[key]
	if exists {
		return value
	}
	// Load from DB in case it is missing.
	enc, err := object.getTrie(db).TryGet(key[:])
	if err != nil {
		object.setError(err)
		return common.Hash{}
	}
	if len(enc) > 0 {
		_, content, _, err := rlp.Split(enc)
		if err != nil {
			object.setError(err)
		}
		value.SetBytes(content)
	}
	object.cachedStorage[key] = value
	return value
}

// SetState updates a value in account storage.
func (object *stateObject) SetState(db Database, key, value common.Hash) {
	object.db.journal.append(storageChange{
		account:  &object.address,
		key:      key,
		prevalue: object.GetState(db, key),
	})
	object.setState(key, value)
}

func (object *stateObject) setState(key, value common.Hash) {
	object.cachedStorage[key] = value
	object.dirtyStorage[key] = value
}

// updateTrie writes cached storage modifications into the object's storage trie.
func (object *stateObject) updateTrie(db Database) Trie {
	tr := object.getTrie(db)
	for key, value := range object.dirtyStorage {
		delete(object.dirtyStorage, key)
		if (value == common.Hash{}) {
			object.setError(tr.TryDelete(key[:]))
			continue
		}
		// Encoding []byte cannot fail, ok to ignore the error.
		v, _ := rlp.EncodeToBytes(bytes.TrimLeft(value[:], "\x00"))
		object.setError(tr.TryUpdate(key[:], v))
	}
	return tr
}

// UpdateRoot sets the trie root to the current root hash of
func (object *stateObject) updateRoot(db Database) {
	object.updateTrie(db)
	object.data.Root = object.trie.Hash()
}

// CommitTrie the storage trie of the object to dwb.
// This updates the trie root.
func (object *stateObject) CommitTrie(db Database) error {
	object.updateTrie(db)
	if object.dbErr != nil {
		return object.dbErr
	}
	root, err := object.trie.Commit(nil)
	if err == nil {
		object.data.Root = root
	}
	return err
}

// AddBalance removes amount from c's balance.
// It is used to add funds to the destination account of a transfer.
func (c *stateObject) AddBalance(amount *big.Int) {
	// EIP158: We must check emptiness for the objects such that the account
	// clearing (0,0,0 objects) can take effect.
	if amount.Sign() == 0 {
		if c.empty() {
			c.touch()
		}

		return
	}
	c.SetBalance(new(big.Int).Add(c.Balance(), amount))
}

// SubBalance removes amount from c's balance.
// It is used to remove funds from the origin account of a transfer.
func (c *stateObject) SubBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	c.SetBalance(new(big.Int).Sub(c.Balance(), amount))
}

func (object *stateObject) SetBalance(amount *big.Int) {
	object.db.journal.append(balanceChange{
		account: &object.address,
		prev:    new(big.Int).Set(object.data.Balance),
	})
	object.setBalance(amount)
}

func (object *stateObject) setBalance(amount *big.Int) {
	object.data.Balance = amount
}

// Return the gas back to the origin. Used by the Virtual machine or Closures
func (c *stateObject) ReturnGas(gas *big.Int) {}

func (object *stateObject) deepCopy(db *StateDB) *stateObject {
	stateObject := newObject(db, object.address, object.data)
	if object.trie != nil {
		stateObject.trie = db.db.CopyTrie(object.trie)
	}
	stateObject.code = object.code
	stateObject.dirtyStorage = object.dirtyStorage.Copy()
	stateObject.cachedStorage = object.dirtyStorage.Copy()
	stateObject.suicided = object.suicided
	stateObject.dirtyCode = object.dirtyCode
	stateObject.deleted = object.deleted
	return stateObject
}

//
// Attribute accessors
//

// Returns the address of the contract/account
func (c *stateObject) Address() common.Address {
	return c.address
}

// Code returns the contract code associated with this object, if any.
func (object *stateObject) Code(db Database) []byte {
	if object.code != nil {
		return object.code
	}
	if bytes.Equal(object.CodeHash(), emptyCodeHash) {
		return nil
	}
	code, err := db.ContractCode(object.addrHash, common.BytesToHash(object.CodeHash()))
	if err != nil {
		object.setError(fmt.Errorf("can't load code hash %x: %v", object.CodeHash(), err))
	}
	object.code = code
	return code
}

func (object *stateObject) SetCode(codeHash common.Hash, code []byte) {
	prevcode := object.Code(object.db.db)
	object.db.journal.append(codeChange{
		account:  &object.address,
		prevhash: object.CodeHash(),
		prevcode: prevcode,
	})
	object.setCode(codeHash, code)
}

func (object *stateObject) setCode(codeHash common.Hash, code []byte) {
	object.code = code
	object.data.CodeHash = codeHash[:]
	object.dirtyCode = true
}

func (object *stateObject) SetNonce(nonce uint64) {
	object.db.journal.append(nonceChange{
		account: &object.address,
		prev:    object.data.Nonce,
	})
	object.setNonce(nonce)
}

func (object *stateObject) setNonce(nonce uint64) {
	object.data.Nonce = nonce
}

func (object *stateObject) CodeHash() []byte {
	return object.data.CodeHash
}

func (object *stateObject) Balance() *big.Int {
	return object.data.Balance
}

func (object *stateObject) Nonce() uint64 {
	return object.data.Nonce
}

// Never called, but must be present to allow stateObject to be used
// as a vm.Account interface that also satisfies the vm.ContractRef
// interface. Interfaces are awesome.
func (object *stateObject) Value() *big.Int {
	panic("Value on stateObject should never be called")
}
