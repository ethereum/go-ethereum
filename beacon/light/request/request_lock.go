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

// SingleLock ensures that a certain request (or a certain type of request) is not
// sent again while there is one that has not been answered or timed out yet.
// If Trigger is not nil then it is triggered whenever the request is unlocked.
type SingleLock struct {
	sentRequest
	Trigger *ModuleTrigger
}

// CanRequest returns true if the request is not locked.
func (s *SingleLock) CanRequest() bool {
	if s.sentTo != nil && s.sentTo.hasTimedOut(s.reqId) {
		s.sentTo = nil
	}
	return s.sentTo == nil
}

// Send acquires the request lock and returns a request ID.
// Note: since Module.Process is never called concurrently, there is no risk of
// two processes simultaneously acquiring the same lock so we can always assume
// that Send is successful after CanRequest returned true.
func (s *SingleLock) Send(srv *Server) uint64 {
	reqId := srv.sendRequest(s.Trigger)
	s.sentTo, s.reqId = srv, reqId
	return reqId
}

// Returned releases the request lock.
func (s *SingleLock) Returned(srv *Server, reqId uint64) {
	if srv == s.sentTo && reqId == s.reqId {
		s.sentTo = nil
	}
	srv.returned(reqId)
	if s.Trigger != nil {
		s.Trigger.Trigger()
	}
}

// MultiLock ensures that no request with a given lockId is sent again while there
// is one sent with the same lockId that has not been answered or timed out yet.
// If Trigger is not nil then it is triggered whenever one of the requests is unlocked.
// Note that the lockId is different from the request ID of actually sent requests
// and it can be of any comparable type.
type MultiLock struct {
	// locks are only present in the map when sentTo != nil
	locks   map[interface{}]sentRequest
	Trigger *ModuleTrigger
}

// CanRequest returns true if the request identified by lockId is not locked.
func (s *MultiLock) CanRequest(lockId interface{}) bool {
	if s.locks == nil {
		s.locks = make(map[interface{}]sentRequest)
	}
	if sl, ok := s.locks[lockId]; ok {
		if sl.sentTo.hasTimedOut(sl.reqId) {
			delete(s.locks, lockId)
		} else {
			return false
		}
	}
	return true
}

// Send acquires the request lock identified by lockId and returns a request ID.
// Note: since Module.Process is never called concurrently, there is no risk of
// two processes simultaneously acquiring the same lock so we can always assume
// that Send is successful after CanRequest returned true.
func (s *MultiLock) Send(srv *Server, lockId interface{}) uint64 {
	reqId := srv.sendRequest(s.Trigger)
	s.locks[lockId] = sentRequest{sentTo: srv, reqId: reqId}
	return reqId
}

// Returned releases the request lock identified by lockId.
func (s *MultiLock) Returned(srv *Server, reqId uint64, lockId interface{}) {
	if s.locks[lockId] == (sentRequest{sentTo: srv, reqId: reqId}) {
		delete(s.locks, lockId)
	}
	srv.returned(reqId)
	if s.Trigger != nil {
		s.Trigger.Trigger()
	}
}
