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

package node

import (
	"path/filepath"
	"reflect"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
)

// ServiceContext is a collection of service independent options inherited from
// the protocol stack, that is passed to all constructors to be optionally used;
// as well as utility methods to operate on the service environment.
type ServiceContext struct {
	datadir  string             // Data directory for protocol persistence
	services map[string]Service // Index of the already constructed services
	EventMux *event.TypeMux     // Event multiplexer used for decoupled notifications
}

// Database opens an existing database with the given name (or creates one if no
// previous can be found) from within the node's data directory. If the node is
// an ephemeral one, a memory database is returned.
func (ctx *ServiceContext) Database(name string, cache int) (ethdb.Database, error) {
	if ctx.datadir == "" {
		return ethdb.NewMemDatabase()
	}
	return ethdb.NewLDBDatabase(filepath.Join(ctx.datadir, name), cache)
}

// Service retrieves an already constructed service registered under a given id.
func (ctx *ServiceContext) Service(id string) Service {
	return ctx.services[id]
}

// SingletonService retrieves an already constructed service using a specific type
// implementing the Service interface. This is a utility function for scenarios
// where it is known that only one instance of a given service type is running,
// allowing to access services without needing to know their specific id with
// which they were registered.
func (ctx *ServiceContext) SingletonService(service interface{}) (string, error) {
	for id, running := range ctx.services {
		if reflect.TypeOf(running) == reflect.ValueOf(service).Elem().Type() {
			reflect.ValueOf(service).Elem().Set(reflect.ValueOf(running))
			return id, nil
		}
	}
	return "", ErrServiceUnknown
}

// ServiceConstructor is the function signature of the constructors needed to be
// registered for service instantiation.
type ServiceConstructor func(ctx *ServiceContext) (Service, error)

// Service is an individual protocol that can be registered into a node.
//
// Notes:
//  - Service life-cycle management is delegated to the node. The service is
//    allowed to initialize itself upon creation, but no goroutines should be
//    spun up outside of the Start method.
//  - Restart logic is not required as the node will create a fresh instance
//    every time a service is started.
type Service interface {
	// Protocol retrieves the P2P protocols the service wishes to start.
	Protocols() []p2p.Protocol

	// Start is called after all services have been constructed and the networking
	// layer was also initialized to spawn any goroutines required by the service.
	Start(server *p2p.Server) error

	// Stop terminates all goroutines belonging to the service, blocking until they
	// are all terminated.
	Stop() error
}
