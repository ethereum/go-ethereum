package ethchain

import (
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
)

type State struct {
	trie *ethutil.Trie
}

func NewState(trie *ethutil.Trie) *State {
	return &State{trie: trie}
}

func (s *State) GetContract(addr []byte) *Contract {
	data := s.trie.Get(string(addr))
	if data == "" {
		return nil
	}

	contract := &Contract{}
	contract.RlpDecode([]byte(data))

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
