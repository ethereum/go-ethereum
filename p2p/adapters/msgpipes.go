// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package adapters

import (
	"github.com/ethereum/go-ethereum/p2p"
)

//network adapter's messenger interace
// NewPipe() (p2p.MsgReadWriter, p2p.MsgReadWriter)
// ClosePipe(rw p2p.MsgReadWriter)

// protocol Messenger interface
// SendMsg(p2p.MsgWriter, uint64, interface{}) error
// ReadMsg(p2p.MsgReader) (p2p.Msg, error)

// peer session test
// ExpectMsg(p2p.MsgReader, uint64, interface{}) error
// SendMsg(p2p.MsgWriter, uint64, interface{}) error
type SimPipe struct{
	rw p2p.MsgReadWriter
}

func (self *SimPipe) SendMsg(code uint64, msg interface{}) error {
	return p2p.Send(self.rw, code, msg)
}

func (self *SimPipe) ReadMsg() (p2p.Msg, error) {
	return self.rw.ReadMsg()
}

func (self *SimPipe) TriggerMsg(code uint64, msg interface{}) error {
	return p2p.Send(self.rw, code, msg)
}

func (self *SimPipe) ExpectMsg(code uint64, msg interface{}) error {
	return p2p.ExpectMsg(self.rw, code, msg)
}

func (self *SimPipe) Close() {
	self.rw.(*p2p.MsgPipeRW).Close()
}

func NewSimPipe(rw p2p.MsgReadWriter) Messenger {
	return Messenger(&SimPipe{rw})
}
