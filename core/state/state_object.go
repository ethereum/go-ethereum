package state

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

type Code []byte

func (self Code) String() string {
	return string(self) //strings.Join(Disassemble(self), " ")
}

type Storage map[string]common.Hash

func (self Storage) String() (str string) {
	for key, value := range self {
		str += fmt.Sprintf("%X : %X\n", key, value)
	}

	return
}

func (self Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range self {
		cpy[key] = value
	}

	return cpy
}

type StateObject struct {
	// State database for storing state changes
	db common.Database
	// The state object
	State *StateDB

	// Address belonging to this account
	address common.Address
	// The balance of the account
	balance *big.Int
	// The nonce of the account
	nonce uint64
	// The code hash if code is present (i.e. a contract)
	codeHash []byte
	// The code for this account
	code Code
	// Temporarily initialisation code
	initCode Code
	// Cached storage (flushed when updated)
	storage Storage
	// Temporary prepaid gas, reward after transition
	prepaid *big.Int

	// Total gas pool is the total amount of gas currently
	// left if this object is the coinbase. Gas is directly
	// purchased of the coinbase.
	gasPool *big.Int

	// Mark for deletion
	// When an object is marked for deletion it will be delete from the trie
	// during the "update" phase of the state transition
	remove bool
	dirty  bool
}

func (self *StateObject) Reset() {
	self.storage = make(Storage)
	self.State.Reset()
}

func NewStateObject(address common.Address, db common.Database) *StateObject {
	// This to ensure that it has 20 bytes (and not 0 bytes), thus left or right pad doesn't matter.
	//address := common.ToAddress(addr)

	object := &StateObject{db: db, address: address, balance: new(big.Int), gasPool: new(big.Int), dirty: true}
	object.State = New(common.Hash{}, db) //New(trie.New(common.Config.Db, ""))
	object.storage = make(Storage)
	object.gasPool = new(big.Int)
	object.prepaid = new(big.Int)

	return object
}

func NewStateObjectFromBytes(address common.Address, data []byte, db common.Database) *StateObject {
	// TODO clean me up
	var extobject struct {
		Nonce    uint64
		Balance  *big.Int
		Root     common.Hash
		CodeHash []byte
	}
	err := rlp.Decode(bytes.NewReader(data), &extobject)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	object := &StateObject{address: address, db: db}
	//object.RlpDecode(data)
	object.nonce = extobject.Nonce
	object.balance = extobject.Balance
	object.codeHash = extobject.CodeHash
	object.State = New(extobject.Root, db)
	object.storage = make(map[string]common.Hash)
	object.gasPool = new(big.Int)
	object.prepaid = new(big.Int)
	object.code, _ = db.Get(extobject.CodeHash)

	return object
}

func (self *StateObject) MarkForDeletion() {
	self.remove = true
	self.dirty = true

	if glog.V(logger.Core) {
		glog.Infof("%x: #%d %v X\n", self.Address(), self.nonce, self.balance)
	}
}

func (c *StateObject) getAddr(addr common.Hash) common.Hash {
	var ret []byte
	rlp.DecodeBytes(c.State.trie.Get(addr[:]), &ret)
	return common.BytesToHash(ret)
}

func (c *StateObject) setAddr(addr []byte, value common.Hash) {
	v, err := rlp.EncodeToBytes(bytes.TrimLeft(value[:], "\x00"))
	if err != nil {
		// if RLPing failed we better panic and not fail silently. This would be considered a consensus issue
		panic(err)
	}
	c.State.trie.Update(addr, v)
}

func (self *StateObject) Storage() Storage {
	return self.storage
}

func (self *StateObject) GetState(key common.Hash) common.Hash {
	strkey := key.Str()
	value, exists := self.storage[strkey]
	if !exists {
		value = self.getAddr(key)
		if (value != common.Hash{}) {
			self.storage[strkey] = value
		}
	}

	return value
}

func (self *StateObject) SetState(k, value common.Hash) {
	self.storage[k.Str()] = value
	self.dirty = true
}

func (self *StateObject) Sync() {
	for key, value := range self.storage {
		if (value == common.Hash{}) {
			self.State.trie.Delete([]byte(key))
			continue
		}

		self.setAddr([]byte(key), value)
	}
	self.storage = make(Storage)
}

func (c *StateObject) GetInstr(pc *big.Int) *common.Value {
	if int64(len(c.code)-1) < pc.Int64() {
		return common.NewValue(0)
	}

	return common.NewValueFromBytes([]byte{c.code[pc.Int64()]})
}

func (c *StateObject) AddBalance(amount *big.Int) {
	c.SetBalance(new(big.Int).Add(c.balance, amount))

	if glog.V(logger.Core) {
		glog.Infof("%x: #%d %v (+ %v)\n", c.Address(), c.nonce, c.balance, amount)
	}
}

func (c *StateObject) SubBalance(amount *big.Int) {
	c.SetBalance(new(big.Int).Sub(c.balance, amount))

	if glog.V(logger.Core) {
		glog.Infof("%x: #%d %v (- %v)\n", c.Address(), c.nonce, c.balance, amount)
	}
}

func (c *StateObject) SetBalance(amount *big.Int) {
	c.balance = amount
	c.dirty = true
}

func (c *StateObject) St() Storage {
	return c.storage
}

//
// Gas setters and getters
//

// Return the gas back to the origin. Used by the Virtual machine or Closures
func (c *StateObject) ReturnGas(gas, price *big.Int) {}
func (c *StateObject) ConvertGas(gas, price *big.Int) error {
	total := new(big.Int).Mul(gas, price)
	if total.Cmp(c.balance) > 0 {
		return fmt.Errorf("insufficient amount: %v, %v", c.balance, total)
	}

	c.SubBalance(total)

	c.dirty = true

	return nil
}

func (self *StateObject) SetGasPool(gasLimit *big.Int) {
	self.gasPool = new(big.Int).Set(gasLimit)

	if glog.V(logger.Core) {
		glog.Infof("%x: gas (+ %v)", self.Address(), self.gasPool)
	}
}

func (self *StateObject) BuyGas(gas, price *big.Int) error {
	if self.gasPool.Cmp(gas) < 0 {
		return GasLimitError(self.gasPool, gas)
	}

	self.gasPool.Sub(self.gasPool, gas)

	rGas := new(big.Int).Set(gas)
	rGas.Mul(rGas, price)

	self.dirty = true

	return nil
}

func (self *StateObject) RefundGas(gas, price *big.Int) {
	self.gasPool.Add(self.gasPool, gas)
}

func (self *StateObject) Copy() *StateObject {
	stateObject := NewStateObject(self.Address(), self.db)
	stateObject.balance.Set(self.balance)
	stateObject.codeHash = common.CopyBytes(self.codeHash)
	stateObject.nonce = self.nonce
	if self.State != nil {
		stateObject.State = self.State.Copy()
	}
	stateObject.code = common.CopyBytes(self.code)
	stateObject.initCode = common.CopyBytes(self.initCode)
	stateObject.storage = self.storage.Copy()
	stateObject.gasPool.Set(self.gasPool)
	stateObject.remove = self.remove
	stateObject.dirty = self.dirty

	return stateObject
}

func (self *StateObject) Set(stateObject *StateObject) {
	*self = *stateObject
}

//
// Attribute accessors
//

func (self *StateObject) Balance() *big.Int {
	return self.balance
}

func (c *StateObject) N() *big.Int {
	return big.NewInt(int64(c.nonce))
}

// Returns the address of the contract/account
func (c *StateObject) Address() common.Address {
	return c.address
}

// Returns the initialization Code
func (c *StateObject) Init() Code {
	return c.initCode
}

func (self *StateObject) Trie() *trie.SecureTrie {
	return self.State.trie
}

func (self *StateObject) Root() []byte {
	return self.Trie().Root()
}

func (self *StateObject) Code() []byte {
	return self.code
}

func (self *StateObject) SetCode(code []byte) {
	self.code = code
	self.dirty = true
}

func (self *StateObject) SetInitCode(code []byte) {
	self.initCode = code
	self.dirty = true
}

func (self *StateObject) SetNonce(nonce uint64) {
	self.nonce = nonce
	self.dirty = true
}

func (self *StateObject) Nonce() uint64 {
	return self.nonce
}

func (self *StateObject) EachStorage(cb func(key, value []byte)) {
	// When iterating over the storage check the cache first
	for h, v := range self.storage {
		cb([]byte(h), v.Bytes())
	}

	it := self.State.trie.Iterator()
	for it.Next() {
		// ignore cached values
		key := self.State.trie.GetKey(it.Key)
		if _, ok := self.storage[string(key)]; !ok {
			cb(key, it.Value)
		}
	}
}

//
// Encoding
//

// State object encoding methods
func (c *StateObject) RlpEncode() []byte {
	return common.Encode([]interface{}{c.nonce, c.balance, c.Root(), c.CodeHash()})
}

func (c *StateObject) CodeHash() common.Bytes {
	return crypto.Sha3(c.code)
}

func (c *StateObject) RlpDecode(data []byte) {
	decoder := common.NewValueFromBytes(data)
	c.nonce = decoder.Get(0).Uint()
	c.balance = decoder.Get(1).BigInt()
	c.State = New(common.BytesToHash(decoder.Get(2).Bytes()), c.db) //New(trie.New(common.Config.Db, decoder.Get(2).Interface()))
	c.storage = make(map[string]common.Hash)
	c.gasPool = new(big.Int)

	c.codeHash = decoder.Get(3).Bytes()

	c.code, _ = c.db.Get(c.codeHash)
}

// Storage change object. Used by the manifest for notifying changes to
// the sub channels.
type StorageState struct {
	StateAddress []byte
	Address      []byte
	Value        *big.Int
}
