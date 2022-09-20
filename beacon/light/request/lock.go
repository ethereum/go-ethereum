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

type SingleLock struct {
	requestLock map[*Server]uint64 // servers where the request has been sent and not timed out yet
}

func (s *SingleLock) CanSend(server *Server) bool {
	reqId, ok := s.requestLock[server]
	if ok && server.Timeout(reqId) {
		delete(s.requestLock, server)
		return false
	}
	return !ok && server.CanSend()
}

// assumes that canSend returned true (no request lock)
func (s *SingleLock) TrySend(srv *Server) (uint64, bool) {
	if s.requestLock == nil {
		s.requestLock = make(map[*Server]uint64)
	}
	if reqId, ok := srv.TrySend(); ok {
		s.requestLock[srv] = reqId
		return reqId, true
	}
	return 0, false
}

func (s *SingleLock) Returned(srv *Server, reqId uint64) {
	delete(s.requestLock, srv)
	srv.Returned(reqId)
}
