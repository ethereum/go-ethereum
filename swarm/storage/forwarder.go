package storage

// implements CloudStore
// noop placeholder for netstore functionality

type Forwarder struct {
}

func (self *Forwarder) Store(chunk *Chunk) {
}

func (self *Forwarder) Retrieve(chunk *Chunk) {
}

func (self *Forwarder) Deliver(chunk *Chunk) {
}
