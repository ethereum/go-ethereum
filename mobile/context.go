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

// Contains all the wrappers from the golang.org/x/net/context package to support
// client side context management on mobile platforms.

package geth

import (
	"context"
	"time"
)

// Context carries a deadline, a cancelation signal, and other values across API
// boundaries.
type Context struct {
	context context.Context
	cancel  context.CancelFunc
}

// NewContext returns a non-nil, empty Context. It is never canceled, has no
// values, and has no deadline. It is typically used by the main function,
// initialization, and tests, and as the top-level Context for incoming requests.
func NewContext() *Context {
	return &Context{
		context: context.Background(),
	}
}

// WithCancel returns a copy of the original context with cancellation mechanism
// included.
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete.
func (c *Context) WithCancel() *Context {
	child, cancel := context.WithCancel(c.context)
	return &Context{
		context: child,
		cancel:  cancel,
	}
}

// WithDeadline returns a copy of the original context with the deadline adjusted
// to be no later than the specified time.
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete.
func (c *Context) WithDeadline(sec int64, nsec int64) *Context {
	child, cancel := context.WithDeadline(c.context, time.Unix(sec, nsec))
	return &Context{
		context: child,
		cancel:  cancel,
	}
}

// WithTimeout returns a copy of the original context with the deadline adjusted
// to be no later than now + the duration specified.
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete.
func (c *Context) WithTimeout(nsec int64) *Context {
	child, cancel := context.WithTimeout(c.context, time.Duration(nsec))
	return &Context{
		context: child,
		cancel:  cancel,
	}
}
