package xeth

import (
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/state"
)

type State struct {
	xeth  *XEth
	state *state.StateDB
}

func NewState(xeth *XEth, statedb *state.StateDB) *State {
	return &State{xeth, statedb}
}

func (self *State) State() *state.StateDB {
	return self.state
}

func (self *State) Get(addr string) *Object {
	return &Object{self.state.GetStateObject(ethutil.FromHex(addr))}
}

func (self *State) SafeGet(addr string) *Object {
	return &Object{self.safeGet(addr)}
}

func (self *State) safeGet(addr string) *state.StateObject {
	object := self.state.GetStateObject(ethutil.FromHex(addr))
	if object == nil {
		object = state.NewStateObject(ethutil.FromHex(addr), self.xeth.eth.StateDb())
	}

	return object
}
