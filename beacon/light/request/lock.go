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

type sentRequest struct {
	sentTo *Server
	reqId  uint64
}

type SingleLock struct {
	sentRequest
	Trigger *ModuleTrigger
}

func (s *SingleLock) CanRequest() bool {
	if s.sentTo != nil && s.sentTo.hasTimedOut(s.reqId) {
		s.sentTo = nil
	}
	return s.sentTo == nil
}

func (s *SingleLock) Send(srv *Server) uint64 {
	reqId := srv.sendRequest(s.Trigger)
	s.sentTo, s.reqId = srv, reqId
	return reqId
}

func (s *SingleLock) Returned(srv *Server, reqId uint64) {
	if srv == s.sentTo && reqId == s.reqId {
		s.sentTo = nil
	}
	srv.returned(reqId)
	if s.Trigger != nil {
		s.Trigger.Trigger()
	}
}

type MultiLock struct {
	locks   map[interface{}]sentRequest // locks are only present in the map when sentTo != nil
	Trigger *ModuleTrigger
}

func (s *MultiLock) CanRequest(id interface{}) bool {
	if s.locks == nil {
		s.locks = make(map[interface{}]sentRequest)
	}
	if sl, ok := s.locks[id]; ok {
		if sl.sentTo.hasTimedOut(sl.reqId) {
			delete(s.locks, id)
		} else {
			return false
		}
	}
	return true
}

func (s *MultiLock) Send(srv *Server, id interface{}) uint64 {
	reqId := srv.sendRequest(s.Trigger)
	s.locks[id] = sentRequest{sentTo: srv, reqId: reqId}
	return reqId
}

func (s *MultiLock) Returned(srv *Server, reqId uint64, id interface{}) {
	if s.locks[id] == (sentRequest{sentTo: srv, reqId: reqId}) {
		delete(s.locks, id)
	}
	srv.returned(reqId)
	if s.Trigger != nil {
		s.Trigger.Trigger()
	}
}
