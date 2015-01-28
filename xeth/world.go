package xeth

import "github.com/ethereum/go-ethereum/state"

type State struct {
	xeth *JSXEth
}

func NewState(xeth *JSXEth) *State {
	return &State{xeth}
}

func (self *State) State() *state.StateDB {
	return self.xeth.chainManager.State()
}

func (self *State) Get(addr string) *Object {
	return &Object{self.State().GetStateObject(fromHex(addr))}
}

func (self *State) SafeGet(addr string) *Object {
	return &Object{self.safeGet(addr)}
}

func (self *State) safeGet(addr string) *state.StateObject {
	object := self.State().GetStateObject(fromHex(addr))
	if object == nil {
		object = state.NewStateObject(fromHex(addr), self.xeth.eth.Db())
	}

	return object
}
