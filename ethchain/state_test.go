package ethchain

import (
	"github.com/ethereum/eth-go/ethdb"
	"github.com/ethereum/eth-go/ethutil"
	"testing"
)

func TestSnapshot(t *testing.T) {
	ethutil.ReadConfig(".ethtest", "/tmp/ethtest", "", "ETH")

	db, _ := ethdb.NewMemDatabase()
	state := NewState(ethutil.NewTrie(db, ""))

	stateObject := NewContract([]byte("aa"), ethutil.Big1, ZeroHash256)
	state.UpdateStateObject(stateObject)
	stateObject.SetStorage(ethutil.Big("0"), ethutil.NewValue(42))

	snapshot := state.Copy()

	stateObject = state.GetStateObject([]byte("aa"))
	stateObject.SetStorage(ethutil.Big("0"), ethutil.NewValue(43))

	state.Set(snapshot)

	stateObject = state.GetStateObject([]byte("aa"))
	if !stateObject.GetStorage(ethutil.Big("0")).Cmp(ethutil.NewValue(42)) {
		t.Error("Expected storage 0 to be 42")
	}
}
