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
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/les/lespay"
	"github.com/ethereum/go-ethereum/les/utils"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	maxRequestLength = 16
	costCutRatio     = 0.1
)

type Server struct {
	costFilter                  *utils.CostFilter
	limiter                     *utils.Limiter
	handlers                    map[string]Handler
	sleepFactor, sizeCostFactor float64
}

type Handler func(id enode.ID, address string, data []byte) []byte

func NewServer(maxThreadTime, maxBandwidth float64) *Server {
	return &Server{
		costFilter:     utils.NewCostFilter(costCutRatio, 0.01),
		limiter:        utils.NewLimiter(1000),
		sleepFactor:    (1/maxThreadTime - 1) / (1 - costCutRatio),
		sizeCostFactor: maxThreadTime * 1000000000 / maxBandwidth,
		handlers:       make(map[string]Handler),
	}
}

// Note: register every handler before serving requests
func (s *Server) RegisterHandler(name string, handler Handler) {
	s.handlers[name] = handler
}

func (s *Server) Serve(id enode.ID, addr *net.UDPAddr, req []byte) []byte {
	var requests lespay.Requests
	if err := rlp.DecodeBytes(req, &requests); err != nil || len(requests) == 0 || len(requests) > maxRequestLength {
		return nil
	}
	priorWeight := uint64(len(requests))
	if priorWeight == 0 {
		return nil
	}
	address := addr.String()
	ch := <-s.limiter.Add(id, address, 0, priorWeight)
	if ch == nil {
		return nil
	}
	start := mclock.Now()
	results := make([][]byte, len(requests))
	for i, req := range requests {
		if handler, ok := s.handlers[req.Name]; ok {
			results[i] = handler(id, address, req.Params)
		}
	}
	res, err := rlp.EncodeToBytes(&results)
	cost := float64(mclock.Now() - start)
	sizeCost := float64(len(res)+100) * s.sizeCostFactor
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
	if err != nil {
		return nil
	}
	return res
}

func (s *Server) Stop() {
	s.limiter.Stop()
}
