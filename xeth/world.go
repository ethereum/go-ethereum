package xeth

import (
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/state"
)

type World struct {
	pipe *XEth
	cfg  *Config
}

func NewWorld(pipe *XEth) *World {
	world := &World{pipe, nil}
	world.cfg = &Config{pipe}

	return world
}

func (self *XEth) World() *World {
	return self.world
}

func (self *World) State() *state.StateDB {
	return self.pipe.chainManager.State()
}

func (self *World) Get(addr []byte) *Object {
	return &Object{self.State().GetStateObject(addr)}
}

func (self *World) SafeGet(addr []byte) *Object {
	return &Object{self.safeGet(addr)}
}

func (self *World) safeGet(addr []byte) *state.StateObject {
	object := self.State().GetStateObject(addr)
	if object == nil {
		object = state.NewStateObject(addr)
	}

	return object
}

func (self *World) Coinbase() *state.StateObject {
	return nil
}

func (self *World) IsMining() bool {
	return self.pipe.obj.IsMining()
}

func (self *World) IsListening() bool {
	return self.pipe.obj.IsListening()
}

func (self *World) Peers() []*p2p.Peer {
	return self.pipe.obj.Peers()
}

func (self *World) Config() *Config {
	return self.cfg
}
