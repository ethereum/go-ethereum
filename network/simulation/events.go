// Copyright 2018 The go-ethereum Authors
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

package simulation

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/simulations"
)

// PeerEvent is the type of the channel returned by Simulation.PeerEvents.
type PeerEvent struct {
	// NodeID is the ID of node that the event is caught on.
	NodeID enode.ID
	// PeerID is the ID of the peer node that the event is caught on.
	PeerID enode.ID
	// Event is the event that is caught.
	Event *simulations.Event
	// Error is the error that may have happened during event watching.
	Error error
}

// PeerEventsFilter defines a filter on PeerEvents to exclude messages with
// defined properties. Use PeerEventsFilter methods to set required options.
type PeerEventsFilter struct {
	eventType simulations.EventType

	connUp *bool

	msgReceive *bool
	protocol   *string
	msgCode    *uint64
}

// NewPeerEventsFilter returns a new PeerEventsFilter instance.
func NewPeerEventsFilter() *PeerEventsFilter {
	return &PeerEventsFilter{}
}

// Connect sets the filter to events when two nodes connect.
func (f *PeerEventsFilter) Connect() *PeerEventsFilter {
	f.eventType = simulations.EventTypeConn
	b := true
	f.connUp = &b
	return f
}

// Drop sets the filter to events when two nodes disconnect.
func (f *PeerEventsFilter) Drop() *PeerEventsFilter {
	f.eventType = simulations.EventTypeConn
	b := false
	f.connUp = &b
	return f
}

// ReceivedMessages sets the filter to only messages that are received.
func (f *PeerEventsFilter) ReceivedMessages() *PeerEventsFilter {
	f.eventType = simulations.EventTypeMsg
	b := true
	f.msgReceive = &b
	return f
}

// SentMessages sets the filter to only messages that are sent.
func (f *PeerEventsFilter) SentMessages() *PeerEventsFilter {
	f.eventType = simulations.EventTypeMsg
	b := false
	f.msgReceive = &b
	return f
}

// Protocol sets the filter to only one message protocol.
func (f *PeerEventsFilter) Protocol(p string) *PeerEventsFilter {
	f.eventType = simulations.EventTypeMsg
	f.protocol = &p
	return f
}

// MsgCode sets the filter to only one msg code.
func (f *PeerEventsFilter) MsgCode(c uint64) *PeerEventsFilter {
	f.eventType = simulations.EventTypeMsg
	f.msgCode = &c
	return f
}

// PeerEvents returns a channel of events that are captured by admin peerEvents
// subscription nodes with provided NodeIDs. Additional filters can be set to ignore
// events that are not relevant.
func (s *Simulation) PeerEvents(ctx context.Context, ids []enode.ID, filters ...*PeerEventsFilter) <-chan PeerEvent {
	eventC := make(chan PeerEvent)

	// wait group to make sure all subscriptions to admin peerEvents are established
	// before this function returns.
	var subsWG sync.WaitGroup
	for _, id := range ids {
		s.shutdownWG.Add(1)
		subsWG.Add(1)
		go func(id enode.ID) {
			defer s.shutdownWG.Done()

			events := make(chan *simulations.Event)
			sub := s.Net.Events().Subscribe(events)
			defer sub.Unsubscribe()

			subsWG.Done()

			for {
				select {
				case <-ctx.Done():
					if err := ctx.Err(); err != nil {
						select {
						case eventC <- PeerEvent{NodeID: id, Error: err}:
						case <-s.Done():
						}
					}
					return
				case <-s.Done():
					return
				case e := <-events:
					// ignore control events
					if e.Control {
						continue
					}
					match := len(filters) == 0 // if there are no filters match all events
					for _, f := range filters {
						if f.eventType == simulations.EventTypeConn && e.Conn != nil {
							if *f.connUp != e.Conn.Up {
								continue
							}
							// all connection filter parameters matched, break the loop
							match = true
							break
						}
						if f.eventType == simulations.EventTypeMsg && e.Msg != nil {
							if f.msgReceive != nil && *f.msgReceive != e.Msg.Received {
								continue
							}
							if f.protocol != nil && *f.protocol != e.Msg.Protocol {
								continue
							}
							if f.msgCode != nil && *f.msgCode != e.Msg.Code {
								continue
							}
							// all message filter parameters matched, break the loop
							match = true
							break
						}
					}
					var peerID enode.ID
					switch e.Type {
					case simulations.EventTypeConn:
						peerID = e.Conn.One
						if peerID == id {
							peerID = e.Conn.Other
						}
					case simulations.EventTypeMsg:
						peerID = e.Msg.One
						if peerID == id {
							peerID = e.Msg.Other
						}
					}
					if match {
						select {
						case eventC <- PeerEvent{NodeID: id, PeerID: peerID, Event: e}:
						case <-ctx.Done():
							if err := ctx.Err(); err != nil {
								select {
								case eventC <- PeerEvent{NodeID: id, PeerID: peerID, Error: err}:
								case <-s.Done():
								}
							}
							return
						case <-s.Done():
							return
						}
					}
				case err := <-sub.Err():
					if err != nil {
						select {
						case eventC <- PeerEvent{NodeID: id, Error: err}:
						case <-ctx.Done():
							if err := ctx.Err(); err != nil {
								select {
								case eventC <- PeerEvent{NodeID: id, Error: err}:
								case <-s.Done():
								}
							}
							return
						case <-s.Done():
							return
						}
					}
				}
			}
		}(id)
	}

	// wait all subscriptions
	subsWG.Wait()
	return eventC
}
