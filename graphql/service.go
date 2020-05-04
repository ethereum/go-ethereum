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
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

// Service encapsulates a GraphQL service.
type Service struct {
	backend  ethapi.Backend   // The backend that queries will operate on.
	graphqlServer *node.HTTPServer
}

// New constructs a new GraphQL service instance.
func New(backend ethapi.Backend, endpoint string, cors, vhosts []string, timeouts rpc.HTTPTimeouts) (*Service, error) {
	service := &Service{
		backend:  backend,
		graphqlServer: &node.HTTPServer{
			Timeouts: timeouts,
			Vhosts: vhosts,
			CorsAllowedOrigins: cors,
		},
	}
	service.graphqlServer.SetEndpoint(endpoint)
	return service, nil
}

// Start is called after all services have been constructed and the networking
// layer was also initialized to spawn any goroutines required by the service.
func (s *Service) Start() error {
	// create handler stack and wrap the graphql handler
	handler, err := newHandler(s.backend)
	if err != nil {
		return err
	}
	handler = node.NewHTTPHandlerStack(handler, s.graphqlServer.CorsAllowedOrigins, s.graphqlServer.Vhosts)
	s.graphqlServer.SetHandler(handler)

	listener, err := net.Listen("tcp", s.graphqlServer.Endpoint())
	if err != nil {
		return err
	}

	// make sure timeout values are meaningful
	node.CheckTimeouts(&s.graphqlServer.Timeouts)
	// create http server
	httpSrv := &http.Server{
		Handler:      handler,
		ReadTimeout:  s.graphqlServer.Timeouts.ReadTimeout,
		WriteTimeout: s.graphqlServer.Timeouts.WriteTimeout,
		IdleTimeout:  s.graphqlServer.Timeouts.IdleTimeout,
	}
	go httpSrv.Serve(listener)
	log.Info("GraphQL endpoint opened", "url", fmt.Sprintf("http://%s", s.graphqlServer.Endpoint))
	// add information to graphql http server
	s.graphqlServer.Server = httpSrv
	s.graphqlServer.ListenerAddr = listener.Addr()
	s.graphqlServer.SetHandler(handler)

	return nil
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

// Stop terminates all goroutines belonging to the service, blocking until they
// are all terminated.
func (s *Service) Stop() error {
	if s.graphqlServer.Server != nil {
		s.graphqlServer.Server.Shutdown(context.Background())
		log.Info("GraphQL endpoint closed", "url", fmt.Sprintf("http://%v", s.graphqlServer.ListenerAddr))
	}
	return nil
}

func (s *Service) Server() *node.HTTPServer {
	return s.graphqlServer
}
