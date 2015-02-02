package state

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/ethutil"
)

type Account struct {
	Balance  string            `json:"balance"`
	Nonce    uint64            `json:"nonce"`
	Root     string            `json:"root"`
	CodeHash string            `json:"codeHash"`
	Storage  map[string]string `json:"storage"`
}

type World struct {
	Root     string             `json:"root"`
	Accounts map[string]Account `json:"accounts"`
}

func (self *StateDB) Dump() []byte {
	world := World{
		Root:     ethutil.Bytes2Hex(self.trie.Root()),
		Accounts: make(map[string]Account),
	}

	it := self.trie.Iterator()
	for it.Next() {
		stateObject := NewStateObjectFromBytes(it.Key, it.Value, self.db)

		account := Account{Balance: stateObject.balance.String(), Nonce: stateObject.Nonce, Root: ethutil.Bytes2Hex(stateObject.Root()), CodeHash: ethutil.Bytes2Hex(stateObject.codeHash)}
		account.Storage = make(map[string]string)

		storageIt := stateObject.State.trie.Iterator()
		for storageIt.Next() {
			account.Storage[ethutil.Bytes2Hex(it.Key)] = ethutil.Bytes2Hex(it.Value)
		}
		world.Accounts[ethutil.Bytes2Hex(it.Key)] = account
	}

	json, err := json.MarshalIndent(world, "", "    ")
	if err != nil {
		fmt.Println("dump err", err)
	}

	return json
}

// Debug stuff
func (self *StateObject) CreateOutputForDiff() {
	fmt.Printf("%x %x %x %x\n", self.Address(), self.State.Root(), self.balance.Bytes(), self.Nonce)
	it := self.State.trie.Iterator()
	for it.Next() {
		fmt.Printf("%x %x\n", it.Key, it.Value)
	}
}
