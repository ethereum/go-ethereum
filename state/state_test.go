package state

import (
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethtrie"
	"github.com/ethereum/go-ethereum/ethutil"
)

var ZeroHash256 = make([]byte, 32)

func TestSnapshot(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	ethutil.ReadConfig(".ethtest", "/tmp/ethtest", "")
	ethutil.Config.Db = db

	state := New(ethtrie.New(db, ""))

	stateObject := state.GetOrNewStateObject([]byte("aa"))

	stateObject.SetStorage(ethutil.Big("0"), ethutil.NewValue(42))

	snapshot := state.Copy()

	stateObject = state.GetStateObject([]byte("aa"))
	stateObject.SetStorage(ethutil.Big("0"), ethutil.NewValue(43))

	state.Set(snapshot)

	stateObject = state.GetStateObject([]byte("aa"))
	res := stateObject.GetStorage(ethutil.Big("0"))
	if !res.Cmp(ethutil.NewValue(42)) {
		t.Error("Expected storage 0 to be 42", res)
	}
}
