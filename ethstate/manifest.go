package ethstate

import (
	"fmt"
	"math/big"
)

// Object manifest
//
// The object manifest is used to keep changes to the state so we can keep track of the changes
// that occurred during a state transitioning phase.
type Manifest struct {
	// XXX These will be handy in the future. Not important for now.
	objectAddresses  map[string]bool
	storageAddresses map[string]map[string]bool

	ObjectChanges  map[string]*StateObject
	StorageChanges map[string]map[string]*big.Int

	Messages []*Message
}

func NewManifest() *Manifest {
	m := &Manifest{objectAddresses: make(map[string]bool), storageAddresses: make(map[string]map[string]bool)}
	m.Reset()

	return m
}

func (m *Manifest) Reset() {
	m.ObjectChanges = make(map[string]*StateObject)
	m.StorageChanges = make(map[string]map[string]*big.Int)
}

func (m *Manifest) AddObjectChange(stateObject *StateObject) {
	m.ObjectChanges[string(stateObject.Address())] = stateObject
}

func (m *Manifest) AddStorageChange(stateObject *StateObject, storageAddr []byte, storage *big.Int) {
	if m.StorageChanges[string(stateObject.Address())] == nil {
		m.StorageChanges[string(stateObject.Address())] = make(map[string]*big.Int)
	}

	m.StorageChanges[string(stateObject.Address())][string(storageAddr)] = storage
}

func (self *Manifest) AddMessage(msg *Message) *Message {
	self.Messages = append(self.Messages, msg)

	return msg
}

type Message struct {
	To, From  []byte
	Input     []byte
	Output    []byte
	Path      int
	Origin    []byte
	Timestamp int64
	Coinbase  []byte
	Block     []byte
	Number    *big.Int
}

func (self *Message) String() string {
	return fmt.Sprintf("Message{to: %x from: %x input: %x output: %x origin: %x coinbase: %x block: %x number: %v timestamp: %d path: %d", self.To, self.From, self.Input, self.Output, self.Origin, self.Coinbase, self.Block, self.Number, self.Timestamp, self.Path)
}
