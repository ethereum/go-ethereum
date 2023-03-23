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

type request interface {
	CanSendTo(server *Server) (canSend bool, priority uint64)
	SendTo(server *Server)
}

// Environment allows Module.Process to send requests to a set of servers. The enabled server set can either be all servers that are not delayed or timed out (in case of a module trigger) or a subset of them that have been triggered by a server trigger.
type Environment struct {
	*HeadTracker
	scheduler     *Scheduler
	allServers    []*Server
	canRequestNow map[*Server]struct{}
}

func (s *Environment) TryRequest(req request) (sent, tryMore bool) {
	var (
		maxServerPriority, maxRequestPriority uint64
		bestServer                            *Server
	)
	for server := range s.canRequestNow {
		canRequest, serverPriority := server.CanRequestNow()
		if !canRequest {
			delete(s.canRequestNow, server)
			continue
		}
		canSend, requestPriority := req.CanSendTo(server)
		if !canSend || requestPriority < maxRequestPriority ||
			(requestPriority == maxRequestPriority && serverPriority <= maxServerPriority) {
			continue
		}
		maxServerPriority, maxRequestPriority = serverPriority, requestPriority
		bestServer = server
	}
	if bestServer != nil {
		req.SendTo(bestServer)
		return true, true
	}
	return false, len(s.canRequestNow) > 0
}

func (s *Environment) CanRequestNow() bool {
	return len(s.canRequestNow) > 0
}

func (s *Environment) CanRequestLater(req request) bool {
	for _, server := range s.allServers {
		if canSend, _ := req.CanSendTo(server); canSend {
			return true
		}
	}
	return false
}
