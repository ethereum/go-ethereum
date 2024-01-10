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
	// Server identifies a server without allowing any direct interaction.
	// Note: server interface is used by Scheduler and Tracker but not used by
	// the modules that do not interact with them directly.
	// In order to make module testing easier, Server interface is used in
	// events and modules.
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

// Tracker allows Modules to start requests and provide feedback about responses
// that were found to be invalid during processing.
type Tracker interface {
	// TryRequest iterates through currently available servers and selects the
	// best server and request to send. The caller provides a callback function
	// that generates a request candidate for each available server. Note that
	// the module may keep track of relevant server specific info, such as assumed
	// available range of data to request, and therefore it may generate different
	// request candidates for different servers. The callback also returns a
	// priority value. TryRequest selects the request candidate with the highest
	// priority value. If multiple candidates belonging to multiple servers have
	// the same highest priority then it selects based on server priority.
	// If a request candidate and a server has been selected, the request is sent
	// and also returned along with the target server and request ID.
	TryRequest(requestFn func(server Server) (Request, float32)) (RequestWithID, bool)
	// InvalidResponse signals that the given response was invalid. Note that
	// certain responses can only be judged by modules, in the context of existing,
	// partially synced data structures. Giving this signal results in blocking
	// the given server for a certain amount of time, ensuring that the same
	// request will not be instantly sent again to the same server.
	InvalidResponse(id ServerAndID, desc string)
}

// tracker implements Tracker. A separate instance is created for each Module.
type tracker struct {
	// servers is a set of currently available servers; it is recreated at every
	// processModule round.
	servers   serverSet
	scheduler *Scheduler
	module    Module
	// requestEvents is a list of events related to requests sent by the given
	// module. It is reset before module processing and the previous contents are
	// passed to Module.Process along with the globally collected server events.
	requestEvents []Event
}

// TryRequest implements Tracker.
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

// InvalidResponse implements Tracker.
func (p *tracker) InvalidResponse(id ServerAndID, desc string) {
	id.Server.(server).fail(desc)
}
