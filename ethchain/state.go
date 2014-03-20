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
}

// Create a new state from a given trie
func NewState(trie *ethutil.Trie) *State {
	return &State{trie: trie, states: make(map[string]*State)}
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
	s.trie.Sync()

	// Sync all nested states
	for _, state := range s.states {
		state.Sync()
	}
}

// Purges the current trie.
func (s *State) Purge() int {
	return s.trie.NewIterator().Purge()
}

func (s *State) GetContract(addr []byte) *Contract {
	data := s.trie.Get(string(addr))
	if data == "" {
		return nil
	}

	// Whet get contract is called the retrieved value might
	// be an account. The StateManager uses this to check
	// to see if the address a tx was sent to is a contract
	// or an account
	value := ethutil.NewValueFromBytes([]byte(data))
	if value.Len() == 2 {
		return nil
	}

	// build contract
	contract := NewContractFromBytes(addr, []byte(data))

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

func (s *State) UpdateContract(contract *Contract) {
	addr := contract.Address()

	s.states[string(addr)] = contract.state
	s.trie.Update(string(addr), string(contract.RlpEncode()))
}

func (s *State) GetAccount(addr []byte) (account *Account) {
	data := s.trie.Get(string(addr))
	if data == "" {
		account = NewAccount(addr, big.NewInt(0))
	} else {
		account = NewAccountFromData(addr, []byte(data))
	}

	return
}

func (s *State) UpdateAccount(addr []byte, account *Account) {
	s.trie.Update(string(addr), string(account.RlpEncode()))
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

// Returns the object stored at key and the type stored at key
// Returns nil if nothing is stored
func (s *State) Get(key []byte) (*ethutil.Value, ObjType) {
	// Fetch data from the trie
	data := s.trie.Get(string(key))
	// Returns the nil type, indicating nothing could be retrieved.
	// Anything using this function should check for this ret val
	if data == "" {
		return nil, NilTy
	}

	var typ ObjType
	val := ethutil.NewValueFromBytes([]byte(data))
	// Check the length of the retrieved value.
	// Len 2 = Account
	// Len 3 = Contract
	// Other = invalid for now. If other types emerge, add them here
	if val.Len() == 2 {
		typ = AccountTy
	} else if val.Len() == 3 {
		typ = ContractTy
	} else {
		typ = UnknownTy
	}

	return val, typ
}

func (s *State) Put(key, object []byte) {
	s.trie.Update(string(key), string(object))
}

// Script compilation functions
// Compiles strings to machine code
func Compile(code []string) (script []string) {
	script = make([]string, len(code))
	for i, val := range code {
		instr, _ := ethutil.CompileInstr(val)

		script[i] = string(instr)
	}

	return
}

func CompileToValues(code []string) (script []*ethutil.Value) {
	script = make([]*ethutil.Value, len(code))
	for i, val := range code {
		instr, _ := ethutil.CompileInstr(val)

		script[i] = ethutil.NewValue(instr)
	}

	return
}
