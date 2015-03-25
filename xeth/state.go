package xeth

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
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
	return &Object{self.state.GetStateObject(common.HexToAddress(addr))}
}

func (self *State) SafeGet(addr string) *Object {
	return &Object{self.safeGet(addr)}
}

func (self *State) safeGet(addr string) *state.StateObject {
	object := self.state.GetStateObject(common.HexToAddress(addr))
	if object == nil {
		object = state.NewStateObject(common.HexToAddress(addr), self.xeth.backend.StateDb())
	}

	return object
}
