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

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

// PeerEvent is the type of the channel returned by Simulation.PeerEvents.
type PeerEvent struct {
	// NodeID is the ID of node that the event is caught on.
	NodeID enode.ID
	// Event is the event that is caught.
	Event *p2p.PeerEvent
	// Error is the error that may have happened during event watching.
	Error error
}

// PeerEventsFilter defines a filter on PeerEvents to exclude messages with
// defined properties. Use PeerEventsFilter methods to set required options.
type PeerEventsFilter struct {
	t        *p2p.PeerEventType
	protocol *string
	msgCode  *uint64
}

// NewPeerEventsFilter returns a new PeerEventsFilter instance.
func NewPeerEventsFilter() *PeerEventsFilter {
	return &PeerEventsFilter{}
}

// Type sets the filter to only one peer event type.
func (f *PeerEventsFilter) Type(t p2p.PeerEventType) *PeerEventsFilter {
	f.t = &t
	return f
}

// Protocol sets the filter to only one message protocol.
func (f *PeerEventsFilter) Protocol(p string) *PeerEventsFilter {
	f.protocol = &p
	return f
}

// MsgCode sets the filter to only one msg code.
func (f *PeerEventsFilter) MsgCode(c uint64) *PeerEventsFilter {
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

			client, err := s.Net.GetNode(id).Client()
			if err != nil {
				subsWG.Done()
				eventC <- PeerEvent{NodeID: id, Error: err}
				return
			}
			events := make(chan *p2p.PeerEvent)
			sub, err := client.Subscribe(ctx, "admin", events, "peerEvents")
			if err != nil {
				subsWG.Done()
				eventC <- PeerEvent{NodeID: id, Error: err}
				return
			}
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
					match := len(filters) == 0 // if there are no filters match all events
					for _, f := range filters {
						if f.t != nil && *f.t != e.Type {
							continue
						}
						if f.protocol != nil && *f.protocol != e.Protocol {
							continue
						}
						if f.msgCode != nil && e.MsgCode != nil && *f.msgCode != *e.MsgCode {
							continue
						}
						// all filter parameters matched, break the loop
						match = true
						break
					}
					if match {
						select {
						case eventC <- PeerEvent{NodeID: id, Event: e}:
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
