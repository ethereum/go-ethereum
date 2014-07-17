package ethchain

import (
	"fmt"
	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethtrie"
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
	"strings"
)

type Code []byte

func (self Code) String() string {
	return strings.Join(Disassemble(self), " ")
}

type Storage map[string]*ethutil.Value

func (self Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range self {
		// XXX Do we need a 'value' copy or is this sufficient?
		cpy[key] = value
	}

	return cpy
}

type StateObject struct {
	// Address of the object
	address []byte
	// Shared attributes
	Amount     *big.Int
	ScriptHash []byte
	Nonce      uint64
	// Contract related attributes
	state      *State
	script     Code
	initScript Code

	storage Storage

	// Total gas pool is the total amount of gas currently
	// left if this object is the coinbase. Gas is directly
	// purchased of the coinbase.
	gasPool *big.Int

	// Mark for deletion
	// When an object is marked for deletion it will be delete from the trie
	// during the "update" phase of the state transition
	remove bool
}

func (self *StateObject) Reset() {
	self.storage = make(Storage)
	self.state.Reset()
}

// Converts an transaction in to a state object
func MakeContract(tx *Transaction, state *State) *StateObject {
	// Create contract if there's no recipient
	if tx.IsContract() {
		addr := tx.CreationAddress()

		contract := state.NewStateObject(addr)
		contract.initScript = tx.Data
		contract.state = NewState(ethtrie.NewTrie(ethutil.Config.Db, ""))

		return contract
	}

	return nil
}

func NewStateObject(addr []byte) *StateObject {
	// This to ensure that it has 20 bytes (and not 0 bytes), thus left or right pad doesn't matter.
	address := ethutil.Address(addr)

	object := &StateObject{address: address, Amount: new(big.Int), gasPool: new(big.Int)}
	object.state = NewState(ethtrie.NewTrie(ethutil.Config.Db, ""))
	object.storage = make(Storage)
	object.gasPool = new(big.Int)

	return object
}

func NewContract(address []byte, Amount *big.Int, root []byte) *StateObject {
	contract := NewStateObject(address)
	contract.Amount = Amount
	contract.state = NewState(ethtrie.NewTrie(ethutil.Config.Db, string(root)))

	return contract
}

func NewStateObjectFromBytes(address, data []byte) *StateObject {
	object := &StateObject{address: address}
	object.RlpDecode(data)

	return object
}

func (self *StateObject) MarkForDeletion() {
	self.remove = true
	statelogger.DebugDetailf("%x: #%d %v (deletion)\n", self.Address(), self.Nonce, self.Amount)
}

func (c *StateObject) GetAddr(addr []byte) *ethutil.Value {
	return ethutil.NewValueFromBytes([]byte(c.state.trie.Get(string(addr))))
}

func (c *StateObject) SetAddr(addr []byte, value interface{}) {
	c.state.trie.Update(string(addr), string(ethutil.NewValue(value).Encode()))
}

func (self *StateObject) GetStorage(key *big.Int) *ethutil.Value {
	return self.getStorage(key.Bytes())
}
func (self *StateObject) SetStorage(key *big.Int, value *ethutil.Value) {
	self.setStorage(key.Bytes(), value)
}

func (self *StateObject) getStorage(k []byte) *ethutil.Value {
	key := ethutil.LeftPadBytes(k, 32)

	value := self.storage[string(key)]
	if value == nil {
		value = self.GetAddr(key)

		if !value.IsNil() {
			self.storage[string(key)] = value
		}
	}

	return value

	//return self.GetAddr(key)
}

func (self *StateObject) setStorage(k []byte, value *ethutil.Value) {
	key := ethutil.LeftPadBytes(k, 32)
	self.storage[string(key)] = value.Copy()

	/*
		if value.BigInt().Cmp(ethutil.Big0) == 0 {
			self.state.trie.Delete(string(key))
			return
		}

		self.SetAddr(key, value)
	*/
}

func (self *StateObject) Sync() {
	/*
		fmt.Println("############# BEFORE ################")
		self.state.EachStorage(func(key string, value *ethutil.Value) {
			fmt.Printf("%x %x %x\n", self.Address(), []byte(key), value.Bytes())
		})
		fmt.Printf("%x @:%x\n", self.Address(), self.state.Root())
		fmt.Println("#####################################")
	*/
	for key, value := range self.storage {
		if value.Len() == 0 { // value.BigInt().Cmp(ethutil.Big0) == 0 {
			//data := self.getStorage([]byte(key))
			//fmt.Printf("deleting %x %x 0x%x\n", self.Address(), []byte(key), data)
			self.state.trie.Delete(string(key))
			continue
		}

		self.SetAddr([]byte(key), value)
	}

	valid, t2 := ethtrie.ParanoiaCheck(self.state.trie)
	if !valid {
		statelogger.Infof("Warn: PARANOIA: Different state storage root during copy %x vs %x\n", self.state.trie.Root, t2.Root)

		self.state.trie = t2
	}

	/*
		fmt.Println("############# AFTER ################")
		self.state.EachStorage(func(key string, value *ethutil.Value) {
			fmt.Printf("%x %x %x\n", self.Address(), []byte(key), value.Bytes())
		})
	*/
	fmt.Printf("%x @:%x\n", self.Address(), self.state.Root())
}

func (c *StateObject) GetInstr(pc *big.Int) *ethutil.Value {
	if int64(len(c.script)-1) < pc.Int64() {
		return ethutil.NewValue(0)
	}

	return ethutil.NewValueFromBytes([]byte{c.script[pc.Int64()]})
}

func (c *StateObject) AddAmount(amount *big.Int) {
	c.SetAmount(new(big.Int).Add(c.Amount, amount))

	statelogger.Debugf("%x: #%d %v (+ %v)\n", c.Address(), c.Nonce, c.Amount, amount)
}

func (c *StateObject) SubAmount(amount *big.Int) {
	c.SetAmount(new(big.Int).Sub(c.Amount, amount))

	statelogger.Debugf("%x: #%d %v (- %v)\n", c.Address(), c.Nonce, c.Amount, amount)
}

func (c *StateObject) SetAmount(amount *big.Int) {
	c.Amount = amount
}

//
// Gas setters and getters
//

// Return the gas back to the origin. Used by the Virtual machine or Closures
func (c *StateObject) ReturnGas(gas, price *big.Int, state *State) {}
func (c *StateObject) ConvertGas(gas, price *big.Int) error {
	total := new(big.Int).Mul(gas, price)
	if total.Cmp(c.Amount) > 0 {
		return fmt.Errorf("insufficient amount: %v, %v", c.Amount, total)
	}

	c.SubAmount(total)

	return nil
}

func (self *StateObject) SetGasPool(gasLimit *big.Int) {
	self.gasPool = new(big.Int).Set(gasLimit)

	statelogger.DebugDetailf("%x: fuel (+ %v)", self.Address(), self.gasPool)
}

func (self *StateObject) BuyGas(gas, price *big.Int) error {
	if self.gasPool.Cmp(gas) < 0 {
		return GasLimitError(self.gasPool, gas)
	}

	rGas := new(big.Int).Set(gas)
	rGas.Mul(rGas, price)

	self.AddAmount(rGas)

	return nil
}

func (self *StateObject) RefundGas(gas, price *big.Int) {
	self.gasPool.Add(self.gasPool, gas)

	rGas := new(big.Int).Set(gas)
	rGas.Mul(rGas, price)

	self.Amount.Sub(self.Amount, rGas)
}

func (self *StateObject) Copy() *StateObject {
	stateObject := NewStateObject(self.Address())
	stateObject.Amount.Set(self.Amount)
	stateObject.ScriptHash = ethutil.CopyBytes(self.ScriptHash)
	stateObject.Nonce = self.Nonce
	if self.state != nil {
		stateObject.state = self.state.Copy()
	}
	stateObject.script = ethutil.CopyBytes(self.script)
	stateObject.initScript = ethutil.CopyBytes(self.initScript)
	stateObject.storage = self.storage.Copy()
	stateObject.gasPool.Set(self.gasPool)

	return stateObject
}

func (self *StateObject) Set(stateObject *StateObject) {
	*self = *stateObject
}

//
// Attribute accessors
//

func (c *StateObject) State() *State {
	return c.state
}

func (c *StateObject) N() *big.Int {
	return big.NewInt(int64(c.Nonce))
}

// Returns the address of the contract/account
func (c *StateObject) Address() []byte {
	return c.address
}

// Returns the main script body
func (c *StateObject) Script() Code {
	return c.script
}

// Returns the initialization script
func (c *StateObject) Init() Code {
	return c.initScript
}

//
// Encoding
//

// State object encoding methods
func (c *StateObject) RlpEncode() []byte {
	var root interface{}
	if c.state != nil {
		root = c.state.trie.Root
	} else {
		root = ""
	}

	return ethutil.Encode([]interface{}{c.Nonce, c.Amount, root, ethcrypto.Sha3Bin(c.script)})
}

func (c *StateObject) RlpDecode(data []byte) {
	decoder := ethutil.NewValueFromBytes(data)

	c.Nonce = decoder.Get(0).Uint()
	c.Amount = decoder.Get(1).BigInt()
	c.state = NewState(ethtrie.NewTrie(ethutil.Config.Db, decoder.Get(2).Interface()))
	c.storage = make(map[string]*ethutil.Value)
	c.gasPool = new(big.Int)

	c.ScriptHash = decoder.Get(3).Bytes()

	c.script, _ = ethutil.Config.Db.Get(c.ScriptHash)
}

// Storage change object. Used by the manifest for notifying changes to
// the sub channels.
type StorageState struct {
	StateAddress []byte
	Address      []byte
	Value        *big.Int
}
