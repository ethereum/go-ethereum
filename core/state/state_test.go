package state

import (
	"bytes"
	"math/big"
	"testing"

	checker "gopkg.in/check.v1"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
)

func TestStateForking(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	state, _ := New(common.Hash{}, db)

	var address common.Address

	object := NewStateObject(address, db)
	object.SetBalance(big.NewInt(1))

	state.StateObjects[address] = object

	ss1 := Fork(state)
	ss2 := Fork(ss1)
	ss2.AddBalance(address, big.NewInt(1))

	if ss1.GetBalance(address).Cmp(big.NewInt(1)) != 0 {
		t.Error("expected ss2 balance to be 1")
	}

	if ss2.GetBalance(address).Cmp(big.NewInt(2)) != 0 {
		t.Error("expected ss2 balance to be 2")
	}
}

func TestCaching(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	state, _ := New(common.Hash{}, db)

	var address common.Address

	object := NewStateObject(address, db)
	object.SetBalance(big.NewInt(1))

	state.StateObjects[address] = object

	ss1 := Fork(state)
	ss2 := Fork(ss1)

	if o := ss2.GetStateObject(address); o == nil {
		t.Error("expected object to exist")
	}

	ss1objects := ss1.StateObjects
	ss2objects := ss2.StateObjects

	if len(ss1objects) != 0 {
		t.Error("expected ss1 objects to be empty")
	}

	if len(ss2objects) != 1 {
		t.Error("expected ss2 objects to be 1")
	}
}

func TestFlatten(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	state, _ := New(common.Hash{}, db)

	var address common.Address

	object := NewStateObject(address, db)
	object.SetBalance(big.NewInt(1))

	state.StateObjects[address] = object

	ss1 := Fork(state)
	ss1.AddBalance(address, big.NewInt(1))

	ss2 := Fork(ss1)
	ss2.AddBalance(address, big.NewInt(1))

	flat1 := Flatten(ss1)
	if flat1.StateObjects[address].Balance().Cmp(big.NewInt(2)) != 0 {
		t.Error("expected flat1 to have balance of 2")
	}

	flat2 := Flatten(ss2)
	if flat2.StateObjects[address].Balance().Cmp(big.NewInt(3)) != 0 {
		t.Error("expected flat2 to have balance of 3")
	}
}

func TestLogs(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	state, _ := New(common.Hash{}, db)
	state.PrepareIntermediate(common.Hash{}, common.Hash{}, 0)
	state.AddLog(&vm.Log{})
	unforkedLogs := state.Logs()

	fork := Fork(state)
	fork.PrepareIntermediate(common.Hash{}, common.Hash{}, 0)
	fork.AddLog(&vm.Log{})
	fork.AddLog(&vm.Log{})
	forkedLogs := fork.Logs()

	if len(unforkedLogs) != 1 {
		t.Error("expected unforked state to have 1 log")
	}

	if len(forkedLogs) != 2 {
		t.Error("expected forked state to have 2 log")
	}
}

func BenchmarkFork(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		db, _ := ethdb.NewMemDatabase()
		state, _ := New(common.Hash{}, db)
		b.StartTimer()

		revertEvery := 5
		nesting := 20
		for x := 0; x < nesting; x++ {
			pstate := state
			nstate := Fork(state)
			nstate.AddBalance(common.Address{byte(x)}, big.NewInt(10))
			// at the very least get the balance of each previous object
			nstate.GetBalance(common.Address{byte(x - 1)})
			if x%revertEvery == 0 {
				state = pstate
			}
		}
	}
}

type StateSuite struct {
	state *State
}

var _ = checker.Suite(&StateSuite{})

var toAddr = common.BytesToAddress

func (s *StateSuite) TestDump(c *checker.C) {
	// generate a few entries
	obj1 := s.state.GetOrNewStateObject(toAddr([]byte{0x01}))
	obj1.AddBalance(big.NewInt(22))
	obj2 := s.state.GetOrNewStateObject(toAddr([]byte{0x01, 0x02}))
	obj2.SetCode([]byte{3, 3, 3, 3, 3, 3, 3})
	obj3 := s.state.GetOrNewStateObject(toAddr([]byte{0x02}))
	obj3.SetBalance(big.NewInt(44))

	// write some of them to the trie
	s.state.UpdateStateObject(obj1)
	s.state.UpdateStateObject(obj2)

	Commit(s.state)

	// check that dump contains the state objects that are in trie
	got := string(s.state.Dump())
	want := `{
    "root": "71edff0130dd2385947095001c73d9e28d862fc286fca2b922ca6f6f3cddfdd2",
    "accounts": {
        "0000000000000000000000000000000000000001": {
            "balance": "22",
            "nonce": 0,
            "root": "56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
            "codeHash": "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
            "code": "",
            "storage": {}
        },
        "0000000000000000000000000000000000000002": {
            "balance": "44",
            "nonce": 0,
            "root": "56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
            "codeHash": "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
            "code": "",
            "storage": {}
        },
        "0000000000000000000000000000000000000102": {
            "balance": "0",
            "nonce": 0,
            "root": "56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
            "codeHash": "87874902497a5bb968da31a2998d8f22e949d1ef6214bcdedd8bae24cca4b9e3",
            "code": "03030303030303",
            "storage": {}
        }
    }
}`
	if got != want {
		c.Errorf("dump mismatch:\ngot: %s\nwant: %s\n", got, want)
	}
}

func (s *StateSuite) SetUpTest(c *checker.C) {
	db, _ := ethdb.NewMemDatabase()
	s.state, _ = New(common.Hash{}, db)
}

func TestNull(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	state, _ := New(common.Hash{}, db)

	address := common.HexToAddress("0x823140710bf13990e4500136726d8b55")
	state.CreateAccount(address)
	//value := common.FromHex("0x823140710bf13990e4500136726d8b55")
	var value common.Hash
	state.SetState(address, common.Hash{}, value)
	Commit(state)
	value = state.GetState(address, common.Hash{})
	if !common.EmptyHash(value) {
		t.Errorf("expected empty hash. got %x", value)
	}
}

// use testing instead of checker because checker does not support
// printing/logging in tests (-check.vv does not work)
func TestSnapshot2(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	state, _ := New(common.Hash{}, db)

	stateobjaddr0 := toAddr([]byte("so0"))
	stateobjaddr1 := toAddr([]byte("so1"))
	var storageaddr common.Hash

	data0 := common.BytesToHash([]byte{17})
	data1 := common.BytesToHash([]byte{18})

	state.SetState(stateobjaddr0, storageaddr, data0)
	state.SetState(stateobjaddr1, storageaddr, data1)

	// db, trie are already non-empty values
	so0 := state.GetStateObject(stateobjaddr0)
	so0.balance = big.NewInt(42)
	so0.nonce = 43
	so0.SetCode([]byte{'c', 'a', 'f', 'e'})
	so0.remove = false
	so0.deleted = false
	so0.dirty = true
	state.StateObjects[so0.Address()] = so0

	// and one with deleted == true
	so1 := state.GetStateObject(stateobjaddr1)
	so1.balance = big.NewInt(52)
	so1.nonce = 53
	so1.SetCode([]byte{'c', 'a', 'f', 'e', '2'})
	so1.remove = true
	so1.deleted = true
	so1.dirty = true
	state.StateObjects[so1.Address()] = so1

	so1 = state.GetStateObject(stateobjaddr1)
	if so1 != nil {
		t.Fatalf("deleted object not nil when getting")
	}

	snapshot := state
	state = Fork(state)
	state.AddBalance(stateobjaddr0, big.NewInt(1))
	state.AddBalance(stateobjaddr1, big.NewInt(1))
	state.Set(snapshot)

	so0Restored := state.GetStateObject(stateobjaddr0)
	so0Restored.GetState(storageaddr)
	so1Restored := state.GetStateObject(stateobjaddr1)
	// non-deleted is equal (restored)
	compareStateObjects(so0Restored, so0, t)
	// deleted should be nil, both before and after restore of state copy
	if so1Restored != nil {
		t.Fatalf("deleted object not nil after restoring snapshot")
	}
}

func compareStateObjects(so0, so1 *StateObject, t *testing.T) {
	if so0.address != so1.address {
		t.Fatalf("Address mismatch: have %v, want %v", so0.address, so1.address)
	}
	if so0.balance.Cmp(so1.balance) != 0 {
		t.Fatalf("Balance mismatch: have %v, want %v", so0.balance, so1.balance)
	}
	if so0.nonce != so1.nonce {
		t.Fatalf("Nonce mismatch: have %v, want %v", so0.nonce, so1.nonce)
	}
	if !bytes.Equal(so0.codeHash, so1.codeHash) {
		t.Fatalf("CodeHash mismatch: have %v, want %v", so0.codeHash, so1.codeHash)
	}
	if !bytes.Equal(so0.code, so1.code) {
		t.Fatalf("Code mismatch: have %v, want %v", so0.code, so1.code)
	}

	for k, v := range so1.storage {
		if so0.storage[k] != v {
			t.Fatalf("Storage key %s mismatch: have %v, want %v", k, so0.storage[k], v)
		}
	}
	for k, v := range so0.storage {
		if so1.storage[k] != v {
			t.Fatalf("Storage key %s mismatch: have %v, want none.", k, v)
		}
	}

	if so0.remove != so1.remove {
		t.Fatalf("Remove mismatch: have %v, want %v", so0.remove, so1.remove)
	}
	if so0.deleted != so1.deleted {
		t.Fatalf("Deleted mismatch: have %v, want %v", so0.deleted, so1.deleted)
	}
	if so0.dirty != so1.dirty {
		t.Fatalf("Dirty mismatch: have %v, want %v", so0.dirty, so1.dirty)
	}
}
