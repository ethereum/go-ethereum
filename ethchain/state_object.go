package ethchain

import (
	"fmt"
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
)

type StateObject struct {
	// Address of the object
	address []byte
	// Shared attributes
	Amount     *big.Int
	ScriptHash []byte
	Nonce      uint64
	// Contract related attributes
	state      *State
	script     []byte
	initScript []byte
}

// Converts an transaction in to a state object
func MakeContract(tx *Transaction, state *State) *StateObject {
	// Create contract if there's no recipient
	if tx.IsContract() {
		addr := tx.CreationAddress()

		value := tx.Value
		contract := NewContract(addr, value, ZeroHash256)

		contract.initScript = tx.Data

		state.UpdateStateObject(contract)

		return contract
	}

	return nil
}

func NewContract(address []byte, Amount *big.Int, root []byte) *StateObject {
	contract := &StateObject{address: address, Amount: Amount, Nonce: 0}
	contract.state = NewState(ethutil.NewTrie(ethutil.Config.Db, string(root)))

	return contract
}

// Returns a newly created account
func NewAccount(address []byte, amount *big.Int) *StateObject {
	account := &StateObject{address: address, Amount: amount, Nonce: 0}

	return account
}

func NewStateObjectFromBytes(address, data []byte) *StateObject {
	object := &StateObject{address: address}
	object.RlpDecode(data)

	return object
}

func (c *StateObject) State() *State {
	return c.state
}

func (c *StateObject) N() *big.Int {
	return big.NewInt(int64(c.Nonce))
}

func (c *StateObject) Addr(addr []byte) *ethutil.Value {
	return ethutil.NewValueFromBytes([]byte(c.state.trie.Get(string(addr))))
}

func (c *StateObject) SetAddr(addr []byte, value interface{}) {
	c.state.trie.Update(string(addr), string(ethutil.NewValue(value).Encode()))
}

func (c *StateObject) SetStorage(num *big.Int, val *ethutil.Value) {
	addr := ethutil.BigToBytes(num, 256)
	//fmt.Println("storing", val.BigInt(), "@", num)
	c.SetAddr(addr, val)
}

func (c *StateObject) GetStorage(num *big.Int) *ethutil.Value {
	nb := ethutil.BigToBytes(num, 256)

	return c.Addr(nb)
}

/* DEPRECATED */
func (c *StateObject) GetMem(num *big.Int) *ethutil.Value {
	return c.GetStorage(num)
}

func (c *StateObject) GetInstr(pc *big.Int) *ethutil.Value {
	if int64(len(c.script)-1) < pc.Int64() {
		return ethutil.NewValue(0)
	}

	return ethutil.NewValueFromBytes([]byte{c.script[pc.Int64()]})
}

// Return the gas back to the origin. Used by the Virtual machine or Closures
func (c *StateObject) ReturnGas(gas, price *big.Int, state *State) {
	remainder := new(big.Int).Mul(gas, price)
	c.AddAmount(remainder)
}

func (c *StateObject) AddAmount(amount *big.Int) {
	c.SetAmount(new(big.Int).Add(c.Amount, amount))

	ethutil.Config.Log.Debugf("%x: #%d %v (+ %v)", c.Address(), c.Nonce, c.Amount, amount)
}

func (c *StateObject) SubAmount(amount *big.Int) {
	c.SetAmount(new(big.Int).Sub(c.Amount, amount))

	ethutil.Config.Log.Debugf("%x: #%d %v (- %v)", c.Address(), c.Nonce, c.Amount, amount)
}

func (c *StateObject) SetAmount(amount *big.Int) {
	c.Amount = amount
}

func (c *StateObject) ConvertGas(gas, price *big.Int) error {
	total := new(big.Int).Mul(gas, price)
	if total.Cmp(c.Amount) > 0 {
		return fmt.Errorf("insufficient amount: %v, %v", c.Amount, total)
	}

	c.SubAmount(total)

	return nil
}

func (self *StateObject) BuyGas(gas, price *big.Int) error {
	rGas := new(big.Int).Set(gas)
	rGas.Mul(gas, price)

	self.AddAmount(rGas)

	// TODO Do sub from TotalGasPool
	// and check if enough left
	return nil
}

// Returns the address of the contract/account
func (c *StateObject) Address() []byte {
	return c.address
}

// Returns the main script body
func (c *StateObject) Script() []byte {
	return c.script
}

// Returns the initialization script
func (c *StateObject) Init() []byte {
	return c.initScript
}

// State object encoding methods
func (c *StateObject) RlpEncode() []byte {
	var root interface{}
	if c.state != nil {
		root = c.state.trie.Root
	} else {
		root = ""
	}

	return ethutil.Encode([]interface{}{c.Nonce, c.Amount, root, ethutil.Sha3Bin(c.script)})
}

func (c *StateObject) RlpDecode(data []byte) {
	decoder := ethutil.NewValueFromBytes(data)

	c.Nonce = decoder.Get(0).Uint()
	c.Amount = decoder.Get(1).BigInt()
	c.state = NewState(ethutil.NewTrie(ethutil.Config.Db, decoder.Get(2).Interface()))

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
