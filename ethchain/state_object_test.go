package ethchain

import (
	"fmt"
	"github.com/ethereum/eth-go/ethdb"
	"github.com/ethereum/eth-go/ethutil"
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
	fmt.Printf("%x\n", object.Script())
}
