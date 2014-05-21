package ethchain

import (
	"fmt"
	"github.com/ethereum/eth-go/ethdb"
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
	"testing"
)

func TestSync(t *testing.T) {
	ethutil.ReadConfig("", ethutil.LogStd)

	db, _ := ethdb.NewMemDatabase()
	state := NewState(ethutil.NewTrie(db, ""))

	contract := NewContract([]byte("aa"), ethutil.Big1, ZeroHash256)

	contract.script = []byte{42}

	state.UpdateStateObject(contract)
	state.Sync()

	object := state.GetStateObject([]byte("aa"))
	if len(object.Script()) == 0 {
		t.Fail()
	}
}

func TestObjectGet(t *testing.T) {
	ethutil.ReadConfig("", ethutil.LogStd)

	db, _ := ethdb.NewMemDatabase()
	ethutil.Config.Db = db

	state := NewState(ethutil.NewTrie(db, ""))

	contract := NewContract([]byte("aa"), ethutil.Big1, ZeroHash256)
	state.UpdateStateObject(contract)

	contract = state.GetStateObject([]byte("aa"))
	contract.SetStorage(big.NewInt(0), ethutil.NewValue("hello"))
	o := contract.GetMem(big.NewInt(0))
	fmt.Println(o)

	state.UpdateStateObject(contract)
	contract.SetStorage(big.NewInt(0), ethutil.NewValue("hello00"))

	contract = state.GetStateObject([]byte("aa"))
	o = contract.GetMem(big.NewInt(0))
	fmt.Println("after", o)
}
