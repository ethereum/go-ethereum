package state

import (
	. "gopkg.in/check.v1"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/trie"
)

func Test(t *testing.T) { TestingT(t) }

type StateSuite struct {
	state *State
}

var _ = Suite(&StateSuite{})

const expectedasbytes = "Expected % x Got % x"

// var ZeroHash256 = make([]byte, 32)

func (s *StateSuite) SetUpTest(c *C) {
	db, _ := ethdb.NewMemDatabase()
	ethutil.ReadConfig(".ethtest", "/tmp/ethtest", "")
	ethutil.Config.Db = db
	s.state = New(trie.New(db, ""))
}

func (s *StateSuite) TestSnapshot(c *C) {
	data1 := ethutil.NewValue(42)
	data2 := ethutil.NewValue(43)
	storageaddr := ethutil.Big("0")
	stateobjaddr := []byte("aa")

	stateObject := s.state.GetOrNewStateObject(stateobjaddr)
	stateObject.SetStorage(storageaddr, data1)
	snapshot := s.state.Copy()

	stateObject = s.state.GetStateObject(stateobjaddr)
	stateObject.SetStorage(storageaddr, data2)
	s.state.Set(snapshot)

	stateObject = s.state.GetStateObject(stateobjaddr)
	res := stateObject.GetStorage(storageaddr)

	c.Assert(data1, DeepEquals, res, Commentf(expectedasbytes, data1, res))
}
