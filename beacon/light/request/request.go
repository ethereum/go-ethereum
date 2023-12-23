// Copyright 2023 The go-ethereum Authors
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

package request

import (
	"math"

	"github.com/ethereum/go-ethereum/log"
)

type (
	Request     any
	Response    any
	ID          uint64
	ServerAndId struct {
		Server Server
		Id     ID
	}
)

// one per sync process
type RequestTracker struct {
	servers       serverSet // one per trigger
	scheduler     *Scheduler
	module        Module
	requestEvents []RequestEvent
}

func (p *RequestTracker) TryRequest(requestFn func(server Server) (Request, float32)) (ServerAndId, Request) {
	var (
		maxServerPriority, maxRequestPriority float32
		bestServer                            Server
		bestRequest                           Request
	)
	maxServerPriority, maxRequestPriority = -math.MaxFloat32, -math.MaxFloat32
	serverCount := len(p.servers)
	var removed, candidates int
	for server, _ := range p.servers {
		canRequest, serverPriority := server.CanRequestNow()
		if !canRequest {
			delete(p.servers, server)
			removed++
			continue
		}
		request, requestPriority := requestFn(server)
		if request != nil {
			candidates++
		}
		if request == nil || requestPriority < maxRequestPriority ||
			(requestPriority == maxRequestPriority && serverPriority <= maxServerPriority) {
			continue
		}
		maxServerPriority, maxRequestPriority = serverPriority, requestPriority
		bestServer, bestRequest = server, request
	}
	log.Debug("Request attempt", "serverCount", serverCount, "removedServers", removed, "requestCandidates", candidates)
	if bestServer == nil {
		return ServerAndId{}, nil
	}
	id := ServerAndId{Server: bestServer, Id: bestServer.SendRequest(bestRequest)}
	p.scheduler.pending[id] = pendingRequest{request: bestRequest, module: p.module}
	return id, bestRequest
}
