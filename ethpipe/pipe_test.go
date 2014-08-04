package ethpipe

import (
	"testing"

	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethutil"
)

func Val(v interface{}) *ethutil.Value {
	return ethutil.NewValue(v)
}

func TestNew(t *testing.T) {
	pipe := New(nil)

	var addr, privy, recp, data []byte
	var object *ethstate.StateObject
	var key *ethcrypto.KeyPair

	world := pipe.World()
	world.Get(addr)
	world.Coinbase()
	world.IsMining()
	world.IsListening()
	world.State()
	peers := world.Peers()
	peers.Len()

	// Shortcut functions
	pipe.Balance(addr)
	pipe.Nonce(addr)
	pipe.Block(addr)
	pipe.Storage(addr, addr)
	pipe.ToAddress(privy)
	// Doesn't change state
	pipe.Execute(addr, nil, Val(0), Val(1000000), Val(10))
	// Doesn't change state
	pipe.ExecuteObject(object, nil, Val(0), Val(1000000), Val(10))

	conf := world.Config()
	namereg := conf.Get("NameReg")
	namereg.Storage(addr)

	var err error
	// Transact
	err = pipe.Transact(key, recp, ethutil.NewValue(0), ethutil.NewValue(0), ethutil.NewValue(0), nil)
	if err != nil {
		t.Error(err)
	}
	// Create
	err = pipe.Transact(key, nil, ethutil.NewValue(0), ethutil.NewValue(0), ethutil.NewValue(0), data)
	if err != nil {
		t.Error(err)
	}
}
