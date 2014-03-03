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
	contract := &Contract{}
	contract.RlpDecode([]byte(data))

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

func (s *State) UpdateContract(addr []byte, contract *Contract) {
	s.trie.Update(string(addr), string(contract.RlpEncode()))
}

func Compile(code []string) (script []string) {
	script = make([]string, len(code))
	for i, val := range code {
		instr, _ := ethutil.CompileInstr(val)

		script[i] = string(instr)
	}

	return
}

func (s *State) GetAccount(addr []byte) (account *Address) {
	data := s.trie.Get(string(addr))
	if data == "" {
		account = NewAddress(big.NewInt(0))
	} else {
		account = NewAddressFromData([]byte(data))
	}

	return
}

func (s *State) UpdateAccount(addr []byte, account *Address) {
	s.trie.Update(string(addr), string(account.RlpEncode()))
}

func (s *State) Cmp(other *State) bool {
	return s.trie.Cmp(other.trie)
}
