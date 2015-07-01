package state

import (
	"math/big"
	"testing"

	checker "gopkg.in/check.v1"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

type StateSuite struct {
	state *StateDB
}

var _ = checker.Suite(&StateSuite{})

var toAddr = common.BytesToAddress

func (s *StateSuite) TestDump(c *checker.C) {
	return
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

	// check that dump contains the state objects that are in trie
	got := string(s.state.Dump())
	want := `{
    "root": "6e277ae8357d013e50f74eedb66a991f6922f93ae03714de58b3d0c5e9eee53f",
    "accounts": {
        "1468288056310c82aa4c01a7e12a10f8111a0560e72b700555479031b86c357d": {
            "balance": "22",
            "nonce": 0,
            "root": "56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
            "codeHash": "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
            "storage": {}
        },
        "a17eacbc25cda025e81db9c5c62868822c73ce097cee2a63e33a2e41268358a1": {
            "balance": "0",
            "nonce": 0,
            "root": "56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
            "codeHash": "87874902497a5bb968da31a2998d8f22e949d1ef6214bcdedd8bae24cca4b9e3",
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
	s.state = New(common.Hash{}, db)
}

func TestNull(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	state := New(common.Hash{}, db)

	address := common.HexToAddress("0x823140710bf13990e4500136726d8b55")
	state.CreateAccount(address)
	//value := common.FromHex("0x823140710bf13990e4500136726d8b55")
	var value common.Hash
	state.SetState(address, common.Hash{}, value)
	state.SyncIntermediate()
	state.Sync()
	value = state.GetState(address, common.Hash{})
	if !common.EmptyHash(value) {
		t.Errorf("expected empty hash. got %x", value)
	}
}

func (s *StateSuite) TestSnapshot(c *checker.C) {
	stateobjaddr := toAddr([]byte("aa"))
	var storageaddr common.Hash
	data1 := common.BytesToHash([]byte{42})
	data2 := common.BytesToHash([]byte{43})

	// set inital state object value
	s.state.SetState(stateobjaddr, storageaddr, data1)
	// get snapshot of current state
	snapshot := s.state.Copy()

	// set new state object value
	s.state.SetState(stateobjaddr, storageaddr, data2)
	// restore snapshot
	s.state.Set(snapshot)

	// get state storage value
	res := s.state.GetState(stateobjaddr, storageaddr)

	c.Assert(data1, checker.DeepEquals, res)
}
