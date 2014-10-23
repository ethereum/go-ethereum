package ethstate

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/ethutil"
)

type Account struct {
	Balance  string            `json:"balance"`
	Nonce    uint64            `json:"nonce"`
	CodeHash string            `json:"codeHash"`
	Storage  map[string]string `json:"storage"`
}

type World struct {
	Root     string             `json:"root"`
	Accounts map[string]Account `json:"accounts"`
}

func (self *State) Dump() []byte {
	world := World{
		Root:     ethutil.Bytes2Hex(self.Trie.Root.([]byte)),
		Accounts: make(map[string]Account),
	}

	self.Trie.NewIterator().Each(func(key string, value *ethutil.Value) {
		stateObject := NewStateObjectFromBytes([]byte(key), value.Bytes())

		account := Account{Balance: stateObject.balance.String(), Nonce: stateObject.Nonce, CodeHash: ethutil.Bytes2Hex(stateObject.codeHash)}
		account.Storage = make(map[string]string)

		stateObject.EachStorage(func(key string, value *ethutil.Value) {
			value.Decode()
			account.Storage[ethutil.Bytes2Hex([]byte(key))] = ethutil.Bytes2Hex(value.Bytes())
		})
		world.Accounts[ethutil.Bytes2Hex([]byte(key))] = account
	})

	json, err := json.MarshalIndent(world, "", "    ")
	if err != nil {
		fmt.Println("dump err", err)
	}

	return json
}
