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

// request represents a new request to be sent. CanSendTo checks request-specific
// conditions under which it can be sent to a certain server while SendTo sends it
// to the selected one. Both should be non-blocking and are never called concurrently.
type request interface {
	CanSendTo(server *Server, moduleData *interface{}) (canSend bool, priority uint64)
	SendTo(server *Server, moduleData *interface{})
}

// Environment allows modules to start network requests when triggered. It is
// responsible for selecting a server for each request. The set of potential
// servers to choose from depends on the type of the trigger; module triggers
// trigger certain modules and allow all available servers to be selected while
// server triggers triggers all modules but only allow triggering servers to be
// selected.
type Environment struct {
	*HeadTracker
	scheduler     *Scheduler
	module        Module
	allServers    []*Server
	canRequestNow map[*Server]struct{}
}

// TryRequest tries to send the given request and returns true in case of success.
func (s *Environment) TryRequest(req request) bool {
	var (
		maxServerPriority, maxRequestPriority uint64
		bestServer                            *Server
	)
	for server := range s.canRequestNow {
		canRequest, serverPriority := server.canRequestNow()
		if !canRequest {
			delete(s.canRequestNow, server)
			continue
		}
		canSend, requestPriority := req.CanSendTo(server, server.moduleData[s.module])
		if !canSend || requestPriority < maxRequestPriority ||
			(requestPriority == maxRequestPriority && serverPriority <= maxServerPriority) {
			continue
		}
		maxServerPriority, maxRequestPriority = serverPriority, requestPriority
		bestServer = server
	}
	if bestServer != nil {
		req.SendTo(bestServer, bestServer.moduleData[s.module])
		return true
	}
	return false
}

// CanRequestNow returns true if there are any servers where a request could be
// sent at the moment.
// Note: when triggered by a module trigger, Module.Process is still called if
// the environment has no servers ready at the moment because it might process
// existing data that does not require further requests to be made. Checking
// CanRequestNow before requesting is optional as a failed TryRequest is also
// cheap; doing a check makes sense if building the request or finding out what to
// request has a significant cost, or if many requests are going to be attempted.
func (s *Environment) CanRequestNow() bool {
	return len(s.canRequestNow) > 0
}

// CanRequestLater returns true if there are connected servers (even if not ready
// at the moment) that could serve the given request.
func (s *Environment) CanRequestLater(req request) bool {
	for _, server := range s.allServers {
		if canSend, _ := req.CanSendTo(server, server.moduleData[s.module]); canSend {
			return true
		}
	}
	return false
}
