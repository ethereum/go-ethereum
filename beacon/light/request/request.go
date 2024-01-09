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
	Server      any
	Request     any
	Response    any
	ID          uint64
	ServerAndID struct {
		Server Server
		ID     ID
	}
	RequestWithID struct {
		ServerAndID
		Request Request
	}
)

// one per sync process
type tracker struct {
	servers       serverSet // one per trigger
	scheduler     *Scheduler
	module        Module
	requestEvents []RequestEvent
}

func (p *tracker) TryRequest(requestFn func(server Server) (Request, float32)) (RequestWithID, bool) {
	var (
		maxServerPriority, maxRequestPriority float32
		bestServer                            server
		bestRequest                           Request
	)
	maxServerPriority, maxRequestPriority = -math.MaxFloat32, -math.MaxFloat32
	serverCount := len(p.servers)
	var removed, candidates int
	for server, _ := range p.servers {
		canRequest, serverPriority := server.canRequestNow()
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
		return RequestWithID{}, false
	}
	id := ServerAndID{Server: bestServer, ID: bestServer.sendRequest(bestRequest)}
	p.scheduler.pending[id] = pendingRequest{request: bestRequest, module: p.module}
	return RequestWithID{ServerAndID: id, Request: bestRequest}, true
}

func (p *tracker) InvalidResponse(id ServerAndID, desc string) {
	id.Server.(server).fail(desc)
}
