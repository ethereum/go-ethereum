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

package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync/atomic"

	mapset "github.com/deckarep/golang-set"
	"github.com/ubiq/go-ubiq/log"
)

const MetadataApi = "rpc"

var (
	// defaultOpenRPCSchemaRaw can be used to establish a default (package-wide) OpenRPC schema from raw JSON.
	// Methods will be cross referenced with actual registered method names in order to serve
	// only server-enabled methods, enabling user and on-the-fly server endpoint availability configuration.
	defaultOpenRPCSchemaRaw string

	errOpenRPCDiscoverUnavailable   = errors.New("openrpc discover data unavailable")
	errOpenRPCDiscoverSchemaInvalid = errors.New("openrpc discover data invalid")
)

// CodecOption specifies which type of messages a codec supports.
//
// Deprecated: this option is no longer honored by Server.
type CodecOption int

const (
	// OptionMethodInvocation is an indication that the codec supports RPC method calls
	OptionMethodInvocation CodecOption = 1 << iota

	// OptionSubscriptions is an indication that the codec suports RPC notifications
	OptionSubscriptions = 1 << iota // support pub sub
)

// Server is an RPC server.
type Server struct {
	services         serviceRegistry
	idgen            func() ID
	run              int32
	codecs           mapset.Set
	OpenRPCSchemaRaw string
}

// NewServer creates a new server instance with no registered handlers.
func NewServer() *Server {
	server := &Server{
		idgen:            randomIDGenerator(),
		codecs:           mapset.NewSet(),
		run:              1,
		OpenRPCSchemaRaw: defaultOpenRPCSchemaRaw,
	}
	// Register the default service providing meta information about the RPC service such
	// as the services and methods it offers.
	rpcService := &RPCService{server}
	server.RegisterName(MetadataApi, rpcService)
	return server
}

func validateOpenRPCSchemaRaw(schemaJSON string) error {
	if schemaJSON == "" {
		return errOpenRPCDiscoverSchemaInvalid
	}
	var schema OpenRPCDiscoverSchemaT
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return fmt.Errorf("%v: %v", errOpenRPCDiscoverSchemaInvalid, err)
	}
	return nil
}

// SetDefaultOpenRPCSchemaRaw validates and sets the package-wide OpenRPC schema data.
func SetDefaultOpenRPCSchemaRaw(schemaJSON string) error {
	if err := validateOpenRPCSchemaRaw(schemaJSON); err != nil {
		return err
	}
	defaultOpenRPCSchemaRaw = schemaJSON
	return nil
}

// SetOpenRPCSchemaRaw validates and sets the raw OpenRPC schema data for a server.
func (s *Server) SetOpenRPCSchemaRaw(schemaJSON string) error {
	if err := validateOpenRPCSchemaRaw(schemaJSON); err != nil {
		return err
	}
	s.OpenRPCSchemaRaw = schemaJSON
	return nil
}

// RegisterName creates a service for the given receiver type under the given name. When no
// methods on the given receiver match the criteria to be either a RPC method or a
// subscription an error is returned. Otherwise a new service is created and added to the
// service collection this server provides to clients.
func (s *Server) RegisterName(name string, receiver interface{}) error {
	return s.services.registerName(name, receiver)
}

// ServeCodec reads incoming requests from codec, calls the appropriate callback and writes
// the response back using the given codec. It will block until the codec is closed or the
// server is stopped. In either case the codec is closed.
//
// Note that codec options are no longer supported.
func (s *Server) ServeCodec(codec ServerCodec, options CodecOption) {
	defer codec.Close()

	// Don't serve if server is stopped.
	if atomic.LoadInt32(&s.run) == 0 {
		return
	}

	// Add the codec to the set so it can be closed by Stop.
	s.codecs.Add(codec)
	defer s.codecs.Remove(codec)

	c := initClient(codec, s.idgen, &s.services)
	<-codec.Closed()
	c.Close()
}

// serveSingleRequest reads and processes a single RPC request from the given codec. This
// is used to serve HTTP connections. Subscriptions and reverse calls are not allowed in
// this mode.
func (s *Server) serveSingleRequest(ctx context.Context, codec ServerCodec) {
	// Don't serve if server is stopped.
	if atomic.LoadInt32(&s.run) == 0 {
		return
	}

	h := newHandler(ctx, codec, s.idgen, &s.services)
	h.allowSubscribe = false
	defer h.close(io.EOF, nil)

	reqs, batch, err := codec.Read()
	if err != nil {
		if err != io.EOF {
			codec.Write(ctx, errorMessage(&invalidMessageError{"parse error"}))
		}
		return
	}
	if batch {
		h.handleBatch(reqs)
	} else {
		h.handleMsg(reqs[0])
	}
}

// Stop stops reading new requests, waits for stopPendingRequestTimeout to allow pending
// requests to finish, then closes all codecs which will cancel pending requests and
// subscriptions.
func (s *Server) Stop() {
	if atomic.CompareAndSwapInt32(&s.run, 1, 0) {
		log.Debug("RPC server shutting down")
		s.codecs.Each(func(c interface{}) bool {
			c.(ServerCodec).Close()
			return true
		})
	}
}

// RPCService gives meta information about the server.
// e.g. gives information about the loaded modules.
type RPCService struct {
	server *Server
}

// Modules returns the list of RPC services with their version number
func (s *RPCService) Modules() map[string]string {
	s.server.services.mu.Lock()
	defer s.server.services.mu.Unlock()

	modules := make(map[string]string)
	for name := range s.server.services.services {
		modules[name] = "1.0"
	}
	return modules
}

func (s *RPCService) methods() map[string][]string {
	s.server.services.mu.Lock()
	defer s.server.services.mu.Unlock()

	methods := make(map[string][]string)
	for name, ser := range s.server.services.services {
		for s := range ser.callbacks {
			_, ok := methods[name]
			if !ok {
				methods[name] = []string{s}
			} else {
				methods[name] = append(methods[name], s)
			}
		}
	}
	return methods
}

// Discover returns a configured schema that is audited for actual server availability.
// Only methods that the server makes available are included in the 'methods' array of
// the discover schema. Components are not audited.
func (s *RPCService) Discover() (schema *OpenRPCDiscoverSchemaT, err error) {
	if s.server.OpenRPCSchemaRaw == "" {
		return nil, errOpenRPCDiscoverUnavailable
	}
	schema = &OpenRPCDiscoverSchemaT{
		Servers: make([]map[string]interface{}, 0),
	}
	err = json.Unmarshal([]byte(s.server.OpenRPCSchemaRaw), schema)
	if err != nil {
		log.Crit("openrpc json umarshal", "error", err)
	}

	// Audit documented schema methods vs. actual server availability
	// This removes methods described in the OpenRPC JSON schema document
	// which are not currently exposed on the server's API.
	// This is done on the fly (as opposed to at servre init or schema setting)
	// because it's possible that exposed APIs could be modified in proc.
	schemaMethodsAvailable := []map[string]interface{}{}
	serverMethodsAvailable := s.methods()

	for _, m := range schema.Methods {
		module, path, err := elementizeMethodName(m["name"].(string))
		if err != nil {
			return nil, err
		}
		paths, ok := serverMethodsAvailable[module]
		if !ok {
			continue
		}

		// the module exists, does the path exist?
		for _, pa := range paths {
			if pa == path {
				schemaMethodsAvailable = append(schemaMethodsAvailable, m)
				break
			}
		}
	}
	schema.Methods = schemaMethodsAvailable
	return
}
