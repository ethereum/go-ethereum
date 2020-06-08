// Copyright 2019 The go-ethereum Authors
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

package graphql

import (
	"errors"
	"net/http"

	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

// Service encapsulates a GraphQL service.
type Service struct {
	backend       ethapi.Backend // The backend that queries will operate on.
	graphqlServer *node.HTTPServer
}

// New constructs a new GraphQL service instance.
func New(stack *node.Node, backend ethapi.Backend, endpoint string, cors, vhosts []string, timeouts rpc.HTTPTimeouts) error {
	service := new(Service)
	if backend == nil {
		return errors.New("No backend found") // TODO should this be a fatal error?
	}
	service.backend = backend
	// check if http server with given endpoint exists and enable graphQL on it
	server := stack.ExistingHTTPServer(endpoint)
	if server != nil {
		server.GQLAllowed = true
		server.Vhosts = append(server.Vhosts, vhosts...)
		server.CorsAllowedOrigins = append(server.CorsAllowedOrigins, cors...)
		server.Timeouts = timeouts
		// create handler
		handler, err := createHandler(service.backend, cors, vhosts)
		if err != nil {
			return err
		}
		server.GQLHandler = handler
		// don't register lifecycle if registering on existing http server
		return nil
	}
	// otherwise create a new server
	handler, err := createHandler(service.backend, cors, vhosts)
	if err != nil {
		return err
	}
	// create the http server
	gqlServer := &node.HTTPServer{
		Vhosts:             vhosts,
		CorsAllowedOrigins: cors,
		Timeouts:           timeouts,
		GQLAllowed:         true,
		GQLHandler:         handler,
		Srv:                rpc.NewServer(),
	}
	gqlServer.SetEndpoint(endpoint)
	stack.RegisterHTTPServer(endpoint, gqlServer)

	service.graphqlServer = gqlServer

	return nil
}

func createHandler(backend ethapi.Backend, cors, vhosts []string) (http.Handler, error) {
	// create handler stack and wrap the graphql handler
	handler, err := newHandler(backend)
	if err != nil {
		return nil, err
	}
	handler = node.NewHTTPHandlerStack(handler, cors, vhosts)

	return handler, nil
}

// newHandler returns a new `http.Handler` that will answer GraphQL queries.
// It additionally exports an interactive query browser on the / endpoint.
func newHandler(backend ethapi.Backend) (http.Handler, error) {
	q := Resolver{backend}

	s, err := graphql.ParseSchema(schema, &q)
	if err != nil {
		return nil, err
	}
	h := &relay.Handler{Schema: s}

	mux := http.NewServeMux()
	mux.Handle("/", GraphiQL{})
	mux.Handle("/graphql", h)
	mux.Handle("/graphql/", h)
	return mux, nil
}

// Start is called after all services have been constructed and the networking
// layer was also initialized to spawn any goroutines required by the service.
func (s *Service) Start() error {

	return nil
}

// Stop terminates all goroutines belonging to the service, blocking until they
// are all terminated.
func (s *Service) Stop() error { return nil }

func (s *Service) Server() *node.HTTPServer {
	return s.graphqlServer
}
