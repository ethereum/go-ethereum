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

	"golang.org/x/net/context"
)

// NoOdr is the default context when ODR is not used
var NoOdr = context.Background()

// NewContext creates an ODR context, carrying channel ID and a "terminated" flag
func NewContext(id *OdrChannelID) (context.Context, context.CancelFunc) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, 0, id)
	ctx = context.WithValue(ctx, 1, new(int32))
	return context.WithTimeout(ctx, id.timeout)
}

// IsOdrContext returns true if ctx is an ODR-enabled context
func IsOdrContext(ctx context.Context) bool {
	_, ok := ctx.Value(0).(*OdrChannelID)
	return ok
}

// setTerminated switches the "terminated" flag on, notifying caller that any
// result it received should be considered invalid
func setTerminated(ctx context.Context) {
	ptr, ok := ctx.Value(1).(*int32)
	if ok {
		atomic.StoreInt32(ptr, 1)
	}
}

// Terminated returns true if the "terminated" flag was switched on, meaning
// that any result received should be considered invalid
func Terminated(ctx context.Context) bool {
	ptr, ok := ctx.Value(1).(*int32)
	if ok {
		return atomic.LoadInt32(ptr) == 1
	} else {
		return false
	}
}

// OdrChannelID is a permanent identifier of a source from where
// requests can come (like an RPC channel).
// (needed for future functions like "list of my waiting requests" and
// "cancel all requests from this channel")
type OdrChannelID struct {
	timeout time.Duration
}

// NewChannelID creates a new OdrChannelID with a channel-specific timeout parameter
func NewChannelID(timeout time.Duration) *OdrChannelID {
	return &OdrChannelID{timeout: timeout}
}
