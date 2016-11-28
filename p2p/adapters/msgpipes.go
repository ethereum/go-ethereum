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
type SimPipe struct{}

func (*SimPipe) SendMsg(w p2p.MsgWriter, code uint64, msg interface{}) error {
	return p2p.Send(w, code, msg)
}

func (*SimPipe) ReadMsg(r p2p.MsgReader) (p2p.Msg, error) {
	return r.ReadMsg()
}

func (*SimPipe) TriggerMsg(w p2p.MsgWriter, code uint64, msg interface{}) error {
	return p2p.Send(w, code, msg)
}

func (*SimPipe) ExpectMsg(r p2p.MsgReader, code uint64, msg interface{}) error {
	return p2p.ExpectMsg(r, code, msg)
}
