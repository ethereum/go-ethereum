// Copyright 2015 The go-ethereum Authors
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

// Package access provides a layer to handle local blockchain database and
// on-demand network retrieval
package access

import (
	"sync/atomic"
	"time"
)

// Context for ODR requests, can be timed out or cancelled manually
type OdrContext struct {
	cancel, cancelOrTimeout chan struct{}
	id                      *OdrChannelID
	cancelled               int32
}

var NullCtx = (*OdrContext)(nil) // used when creating states
var NoOdr = (*OdrContext)(nil)   // used for individual requests

func NewContext(id *OdrChannelID) *OdrContext {
	ctx := &OdrContext{
		cancel:          make(chan struct{}),
		cancelOrTimeout: make(chan struct{}),
		id:              id,
	}
	go func() {
		select {
		case <-ctx.cancel:
		case <-time.After(id.timeout):
		}
		ctx.setCancelled()
		close(ctx.cancelOrTimeout)
	}()
	return ctx
}

func (self *OdrContext) Cancel() {
	close(self.cancel)
}

func (self *OdrContext) setCancelled() {
	atomic.StoreInt32(&self.cancelled, 1)
}

func (self *OdrContext) IsCancelled() bool {
	if self == nil {
		return false
	}
	return atomic.LoadInt32(&self.cancelled) == 1
}

// While contexts are created for each request, channel ID is a permanent
//  identifier of a source from where requests can come (like an RPC channel).
//  (needed for future functions like "list of my waiting requests" and
//  "cancel all requests from this channel")
type OdrChannelID struct {
	timeout time.Duration
}

func NewChannelID(timeout time.Duration) *OdrChannelID {
	return &OdrChannelID{timeout: timeout}
}
