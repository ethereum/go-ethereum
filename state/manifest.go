package state

import (
	"fmt"
	"math/big"
)

// Object manifest
//
// The object manifest is used to keep changes to the state so we can keep track of the changes
// that occurred during a state transitioning phase.
type Manifest struct {
	Messages Messages
}

func NewManifest() *Manifest {
	m := &Manifest{}
	m.Reset()

	return m
}

func (m *Manifest) Reset() {
	m.Messages = nil
}

func (self *Manifest) AddMessage(msg *Message) *Message {
	self.Messages = append(self.Messages, msg)

	return msg
}

type Messages []*Message
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
	Value     *big.Int

	ChangedAddresses [][]byte
}

func (self *Message) AddStorageChange(addr []byte) {
	self.ChangedAddresses = append(self.ChangedAddresses, addr)
}

func (self *Message) String() string {
	return fmt.Sprintf("Message{to: %x from: %x input: %x output: %x origin: %x coinbase: %x block: %x number: %v timestamp: %d path: %d value: %v", self.To, self.From, self.Input, self.Output, self.Origin, self.Coinbase, self.Block, self.Number, self.Timestamp, self.Path, self.Value)
}
