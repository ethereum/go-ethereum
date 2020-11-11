// Copyright 2020 The go-ethereum Authors
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

package server

import (
	"net"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/les/lespay"
	"github.com/ethereum/go-ethereum/les/utils"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
)

type (
	// Server serves lespay requests
	Server struct {
		costFilter                  *utils.CostFilter
		limiter                     *utils.Limiter
		services, alias             map[string]*serviceEntry
		priority                    []*serviceEntry
		sleepFactor, sizeCostFactor float64
	}

	// Service is a service registered at the Server and identified by a string id
	Service interface {
		ServiceInfo() (id, desc string)                                      // only called during registration
		Handle(id enode.ID, address string, name string, data []byte) []byte // never called concurrently
	}

	serviceEntry struct {
		id, desc string
		backend  Service
	}
)

// NewServer creates a new Server
func NewServer(maxThreadTime, maxBandwidth float64) *Server {
	return &Server{
		costFilter:     utils.NewCostFilter(0.1, 0.01),
		limiter:        utils.NewLimiter(1000),
		sleepFactor:    (1/maxThreadTime - 1) / 0.9,
		sizeCostFactor: maxThreadTime * 1000000000 / maxBandwidth,
		services:       make(map[string]*serviceEntry),
	}
}

// Register registers a Service
func (s *Server) Register(b Service) {
	srv := &serviceEntry{backend: b}
	srv.id, srv.desc = b.ServiceInfo()
	if strings.Contains(srv.id, ":") {
		panic("Service ID contains ':'")
	}
	s.services[srv.id] = srv
	s.priority = append(s.priority, srv)
}

// Resolve returns the Service registered under the given id
func (s *Server) Resolve(serviceID string) Service {
	var srv *serviceEntry
	if len(serviceID) > 0 && serviceID[0] == ':' {
		// service alias
		srv = s.alias[serviceID[1:]]
	} else {
		srv = s.services[serviceID]
	}
	if srv != nil {
		return srv.backend
	}
	return nil
}

// Serve serves a lespay request batch
// Note: requests are served by the Handle functions of the registered services. Serve
// may be called concurrently but the Handle functions are called sequentially and
// therefore thread safety is guaranteed.
func (s *Server) Serve(id enode.ID, address string, requests lespay.Requests) lespay.Replies {
	if len(requests) == 0 || len(requests) > lespay.MaxRequestLength {
		return nil
	}
	priorWeight := uint64(len(requests))
	if priorWeight == 0 {
		return nil
	}
	ch := <-s.limiter.Add(id, address, 0, priorWeight)
	if ch == nil {
		return nil
	}
	// Note: the following section is protected from concurrency by the limiter
	start := mclock.Now()
	results := make(lespay.Replies, len(requests))
	s.alias = make(map[string]*serviceEntry)
	var size int
	for i, req := range requests {
		if service := s.Resolve(req.Service); service != nil {
			results[i] = service.Handle(id, address, req.Name, req.Params)
			size += len(results[i]) + 2
		}
	}
	s.alias = nil
	cost := float64(mclock.Now() - start)
	sizeCost := float64(size+100) * s.sizeCostFactor
	if sizeCost > cost {
		cost = sizeCost
	}
	fWeight := float64(priorWeight) / lespay.MaxRequestLength
	filteredCost, limit := s.costFilter.Filter(cost, fWeight)
	time.Sleep(time.Duration(filteredCost * s.sleepFactor))
	// The protected section ends by sending the cost value to the channel and thereby
	// allowing the limiter to start the next request
	if limit*fWeight <= filteredCost {
		ch <- fWeight
	} else {
		ch <- filteredCost / limit
	}
	return results
}

// ServeEncoded serves an encoded lespay request batch and returns the encoded replies
func (s *Server) ServeEncoded(id enode.ID, addr *net.UDPAddr, req []byte) []byte {
	var requests lespay.Requests
	if err := rlp.DecodeBytes(req, &requests); err != nil {
		return nil
	}
	results := s.Serve(id, addr.String(), requests)
	if results == nil {
		return nil
	}
	res, _ := rlp.EncodeToBytes(&results)
	return res
}

// Stop shuts downs the server
func (s *Server) Stop() {
	s.limiter.Stop()
}
