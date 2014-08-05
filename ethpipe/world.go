package ethpipe

import (
	"container/list"

	"github.com/ethereum/eth-go/ethstate"
)

type World struct {
	pipe *Pipe
	cfg  *Config
}

func NewWorld(pipe *Pipe) *World {
	world := &World{pipe, nil}
	world.cfg = &Config{pipe}

	return world
}

func (self *Pipe) World() *World {
	return self.world
}

func (self *World) State() *ethstate.State {
	return self.pipe.stateManager.CurrentState()
}

func (self *World) Get(addr []byte) *Object {
	return &Object{self.State().GetStateObject(addr)}
}

func (self *World) safeGet(addr []byte) *ethstate.StateObject {
	object := self.State().GetStateObject(addr)
	if object == nil {
		object = ethstate.NewStateObject(addr)
	}

	return object
}

func (self *World) Coinbase() *ethstate.StateObject {
	return nil
}

func (self *World) IsMining() bool {
	return self.pipe.obj.IsMining()
}

func (self *World) IsListening() bool {
	return self.pipe.obj.IsListening()
}

func (self *World) Peers() *list.List {
	return self.pipe.obj.Peers()
}

func (self *World) Config() *Config {
	return self.cfg
}
