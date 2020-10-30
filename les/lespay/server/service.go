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
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/les/lespay"
	"github.com/ethereum/go-ethereum/les/utils"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nodestate"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	maxRequestLength = 16
	costCutRatio     = 0.1
)

type (
	Server struct {
		ns                          *nodestate.NodeStateMachine
		db                          ethdb.Database
		costFilter                  *utils.CostFilter
		limiter                     *utils.Limiter
		services, alias             map[string]*serviceEntry
		priority                    []*serviceEntry
		sleepFactor, sizeCostFactor float64

		opService *serviceEntry
		opBatch   ethdb.Batch
	}

	Service interface {
		ServiceInfo() (string, string) // only called during registration
		Handle(id enode.ID, address string, name string, data []byte) []byte
	}

	serviceEntry struct {
		id, desc string
		backend  Service
	}

	DbAccess struct {
		server  *Server
		service *serviceEntry
		prefix  []byte
	}
)

func NewServer(ns *nodestate.NodeStateMachine, db ethdb.Database, maxThreadTime, maxBandwidth float64) *Server {
	return &Server{
		ns:             ns,
		db:             db,
		costFilter:     utils.NewCostFilter(costCutRatio, 0.01),
		limiter:        utils.NewLimiter(1000),
		sleepFactor:    (1/maxThreadTime - 1) / (1 - costCutRatio),
		sizeCostFactor: maxThreadTime * 1000000000 / maxBandwidth,
		services:       make(map[string]*serviceEntry),
	}
}

func (s *Server) Register(b Service) *DbAccess {
	srv := &serviceEntry{backend: b}
	srv.id, srv.desc = b.ServiceInfo()
	if strings.Contains(srv.id, ":") {
		panic("Service ID contains ':'")
	}
	s.services[srv.id] = srv
	s.priority = append(s.priority, srv)
	return &DbAccess{
		server:  s,
		service: srv,
		prefix:  append([]byte(srv.id), byte(':')),
	}
}

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

func (s *Server) HandleTalkRequest(id enode.ID, addr *net.UDPAddr, req []byte) []byte {
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

func (s *Server) Serve(id enode.ID, address string, requests lespay.Requests) lespay.Replies {
	if len(requests) == 0 || len(requests) > maxRequestLength {
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
	fWeight := float64(priorWeight) / maxRequestLength
	filteredCost, limit := s.costFilter.Filter(cost, fWeight)
	time.Sleep(time.Duration(filteredCost * s.sleepFactor))
	if limit*fWeight <= filteredCost {
		ch <- fWeight
	} else {
		ch <- filteredCost / limit
	}
	return results
}

func (s *Server) Stop() {
	s.limiter.Stop()
}

func (d *DbAccess) Operation(fn func(), write bool) {
	d.server.opService = d.service
	if write {
		d.server.opBatch = d.server.db.NewBatch()
	}
	d.server.ns.Operation(fn)
	d.server.opService = nil
	if write {
		d.server.opBatch.Write()
		d.server.opBatch = nil
	}
}

func (d *DbAccess) SubOperation(srv *serviceEntry, fn func()) {
	if d.server.opService != d.service {
		panic("Database access not allowed")
	}
	d.server.opService = srv
	fn()
	d.server.opService = d.service
}

func (d *DbAccess) Has(key []byte) (bool, error) {
	if d.server.opService != d.service {
		panic("Database access not allowed")
	}
	return d.server.db.Has(append(d.prefix, key...))
}

func (d *DbAccess) Get(key []byte) ([]byte, error) {
	if d.server.opService != d.service {
		panic("Database access not allowed")
	}
	return d.server.db.Get(append(d.prefix, key...))
}

func (d *DbAccess) Put(key []byte, value []byte) error {
	if d.server.opService != d.service {
		panic("Database access not allowed")
	}
	return d.server.opBatch.Put(append(d.prefix, key...), value)
}

func (d *DbAccess) Delete(key []byte) error {
	if d.server.opService != d.service {
		panic("Database access not allowed")
	}
	return d.server.opBatch.Delete(append(d.prefix, key...))
}

func (d *DbAccess) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	if d.server.opService != d.service {
		panic("Database access not allowed")
	}
	return d.server.db.NewIterator(append(d.prefix, prefix...), start)
}
