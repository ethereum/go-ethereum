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

// Contains a batch of utility type declarations used by the tests. As the node
// operates on unique types, a lot of them are needed to check various features.

package node

import (
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

// NoopService is a trivial implementation of the Service interface.
type NoopLifecycle struct{}

func (s *NoopLifecycle) Start() error   				{ return nil }
func (s *NoopLifecycle) Stop() error               	{ return nil }

func NewNoop(stack *Node) (*Noop, error) {
	noop := new(Noop)
	stack.RegisterLifecycle(noop)
	return noop, nil
}

// Set of services all wrapping the base NoopService resulting in the same method
// signatures but different outer types.
type Noop struct{ NoopLifecycle }

//func NewNoopServiceA(*ServiceContext) (Lifecycle, error) { return new(NoopServiceA), nil }
//func NewNoopServiceB(*ServiceContext) (Lifecycle, error) { return new(NoopServiceB), nil }
//func NewNoopServiceC(*ServiceContext) (Lifecycle, error) { return new(NoopServiceC), nil }

// InstrumentedService is an implementation of Service for which all interface
// methods can be instrumented both return value as well as event hook wise.
type InstrumentedService struct {
	protocols []p2p.Protocol
	apis      []rpc.API
	start     error
	stop      error

	server	*p2p.Server

	protocolsHook func()
	startHook     func(*p2p.Server)
	stopHook      func()
}

func NewInstrumentedService(server *p2p.Server) (Lifecycle, error) {
	is := &InstrumentedService{ server: server }
	return is, nil
}

func (s *InstrumentedService) Protocols() []p2p.Protocol {
	if s.protocolsHook != nil {
		s.protocolsHook()
	}
	return s.protocols
}

func (s *InstrumentedService) APIs() []rpc.API {
	return s.apis
}

func (s *InstrumentedService) Start() error {
	if s.startHook != nil {
		s.startHook(s.server)
	}
	return s.start
}

func (s *InstrumentedService) Stop() error {
	if s.stopHook != nil {
		s.stopHook()
	}
	return s.stop
}
