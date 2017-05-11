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
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/rpc"
)

// rpcMux is an RPC multiplexer which allows many clients to make RPC requests
// over a single connection by changing each request's ID to a unique value.
//
// This is used by node adapters so that simulations can create many RPC
// clients all sending requests over the underlying node's stdin / stdout.
type rpcMux struct {
	conn net.Conn

	mtx       sync.Mutex
	idCounter uint64
	msgMap    map[uint64]*rpcMsg
	subMap    map[string]*rpcReply
	send      chan *rpcMsg
}

type rpcMsg struct {
	Method  string          `json:"method,omitempty"`
	Version string          `json:"jsonrpc,omitempty"`
	Id      json.RawMessage `json:"id,omitempty"`
	Payload json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`

	id    uint64
	reply *rpcReply
}

// rpcSub is the payload or result of a subscription RPC message
type rpcSub struct {
	Subscription string          `json:"subscription"`
	Result       json.RawMessage `json:"result,omitempty"`
}

// rpcReply receives replies to RPC messages for a particular client
type rpcReply struct {
	ch        chan *rpcMsg
	closeOnce sync.Once
}

func (r *rpcReply) close() {
	r.closeOnce.Do(func() { close(r.ch) })
}

func newRPCMux(conn net.Conn) *rpcMux {
	mux := &rpcMux{
		msgMap: make(map[uint64]*rpcMsg),
		subMap: make(map[string]*rpcReply),
		send:   make(chan *rpcMsg),
	}
	go mux.sendLoop(conn)
	go mux.recvLoop(conn)
	return mux
}

// Client creates a new RPC client which sends messages through the multiplexer
func (mux *rpcMux) Client() *rpc.Client {
	pipe1, pipe2 := net.Pipe()
	go mux.Serve(pipe1)
	return rpc.NewClientWithConn(pipe2)
}

// Serve reads RPC messages from the given connection, forwards them to the
// multiplexed connnection and writes replies back to the given connection
func (mux *rpcMux) Serve(conn net.Conn) {
	// reply will receive replies to any messages we send
	reply := &rpcReply{ch: make(chan *rpcMsg)}
	defer func() {
		// drain the channel to prevent blocking the recvLoop
		for range reply.ch {
		}
	}()

	// start a goroutine to read RPC messages from the connection and
	// forward them to the sendLoop
	done := make(chan struct{})
	go func() {
		defer close(done)
		dec := json.NewDecoder(conn)
		for {
			msg := &rpcMsg{}
			if err := dec.Decode(msg); err != nil {
				return
			}
			msg.reply = reply
			mux.send <- msg
		}
	}()

	// write message replies to the connection
	enc := json.NewEncoder(conn)
	for {
		select {
		case msg, ok := <-reply.ch:
			if !ok {
				return
			}
			if err := enc.Encode(msg); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}

// sendLoop receives messages from the send channel, changes their ID and
// writes them to the given connection
func (mux *rpcMux) sendLoop(conn net.Conn) {
	enc := json.NewEncoder(conn)
	for msg := range mux.send {
		if err := enc.Encode(mux.newMsg(msg)); err != nil {
			return
		}
	}
}

// recvLoop reads messages from the given connection, changes their ID back
// to the oringal value and sends them to the message's reply channel
func (mux *rpcMux) recvLoop(conn net.Conn) {
	// close all reply channels if we get an error
	defer func() {
		mux.mtx.Lock()
		defer mux.mtx.Unlock()
		for _, msg := range mux.msgMap {
			msg.reply.close()
		}
	}()

	dec := json.NewDecoder(conn)
	for {
		msg := &rpcMsg{}
		if err := dec.Decode(msg); err != nil {
			return
		}
		if reply := mux.lookup(msg); reply != nil {
			reply.ch <- msg
		}
	}
}

// newMsg copies the given message and changes it's ID to a unique value
func (mux *rpcMux) newMsg(msg *rpcMsg) *rpcMsg {
	mux.mtx.Lock()
	defer mux.mtx.Unlock()
	id := mux.idCounter
	mux.idCounter++
	mux.msgMap[id] = msg
	newMsg := *msg
	newMsg.Id = json.RawMessage(strconv.FormatUint(id, 10))
	return &newMsg
}

// lookup looks up the original message for which the given message is a reply
func (mux *rpcMux) lookup(msg *rpcMsg) *rpcReply {
	mux.mtx.Lock()
	defer mux.mtx.Unlock()

	// if the message has no ID, it is a subscription notification so
	// lookup the original subscribe message
	if msg.Id == nil {
		sub := &rpcSub{}
		if err := json.Unmarshal(msg.Payload, sub); err != nil {
			return nil
		}
		return mux.subMap[sub.Subscription]
	}

	// lookup the original message and restore the ID
	id, err := strconv.ParseUint(string(msg.Id), 10, 64)
	if err != nil {
		return nil
	}
	origMsg, ok := mux.msgMap[id]
	if !ok {
		return nil
	}
	delete(mux.msgMap, id)
	msg.Id = origMsg.Id

	// if the original message was a subscription, store the subscription
	// ID so we can detect notifications
	if strings.HasSuffix(string(origMsg.Method), "_subscribe") {
		var result string
		if err := json.Unmarshal(msg.Result, &result); err == nil {
			mux.subMap[result] = origMsg.reply
		}
	}

	return origMsg.reply
}
