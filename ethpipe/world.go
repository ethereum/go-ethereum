package ethpipe

import (
	"container/list"

	"github.com/ethereum/eth-go/ethstate"
)

type world struct {
	pipe *Pipe
	cfg  *config
}

func NewWorld(pipe *Pipe) *world {
	world := &world{pipe, nil}
	world.cfg = &config{pipe}

	return world
}

func (self *Pipe) World() *world {
	return self.world
}

func (self *world) State() *ethstate.State {
	return self.pipe.stateManager.CurrentState()
}

func (self *world) Get(addr []byte) *Object {
	return &Object{self.State().GetStateObject(addr)}
}

func (self *world) safeGet(addr []byte) *ethstate.StateObject {
	object := self.State().GetStateObject(addr)
	if object == nil {
		object = ethstate.NewStateObject(addr)
	}

	return object
}

func (self *world) Coinbase() *ethstate.StateObject {
	return nil
}

func (self *world) IsMining() bool {
	return self.pipe.obj.IsMining()
}

func (self *world) IsListening() bool {
	return self.pipe.obj.IsListening()
}

func (self *world) Peers() *list.List {
	return self.pipe.obj.Peers()
}

func (self *world) Config() *config {
	return self.cfg
}
