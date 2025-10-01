// Copyright 2025 The go-ethereum Authors
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

package rest

import (
	"fmt"
	"net/http"
)

const (
	defaultRequestLimit  = 5 * 1024 * 1024
	defaultResponseLimit = 5 * 1024 * 1024
)

// Server is a REST API server.
type Server struct {
	itemLimit, requestLimit, responseLimit int
	mux                                    http.ServeMux
}

// NewServer creates a new server instance with no registered handlers.
func NewServer() *Server {
	return &Server{
		requestLimit:  defaultRequestLimit,
		responseLimit: defaultResponseLimit,
	}
}

func (s *Server) Stop() {} //TODO is this required?

func (s *Server) Register(api API) {
	api.Register(&s.mux, s.responseLimit)
}

// SetBatchLimits sets limits applied to batch requests. There are two limits: 'itemLimit'
// is the maximum number of items in a batch. 'maxResponseSize' is the maximum number of
// response bytes across all requests in a batch.
//
// This method should be called before processing any requests via ServeCodec, ServeHTTP,
// ServeListener etc.
/*func (s *Server) SetBatchLimits(itemLimit, maxResponseSize int) {
	s.batchItemLimit = itemLimit
	s.batchResponseLimit = maxResponseSize
}*/

// SetHTTPBodyLimit sets the size limit for HTTP requests.
//
// This method should be called before processing any requests via ServeHTTP.
/*func (s *Server) SetHTTPBodyLimit(limit int) {
	s.httpBodyLimit = limit
}*/

// ServeHTTP serves REST API requests over HTTP.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength < 0 {
		http.Error(w, "request size unknown", http.StatusRequestEntityTooLarge)
		return
	}
	if reqLen := int64(len(r.URL.RawQuery)) + r.ContentLength; reqLen > int64(s.requestLimit) {
		http.Error(w, fmt.Sprintf("request too large (%d>%d)", reqLen, s.requestLimit), http.StatusRequestEntityTooLarge)
		return
	}
	s.mux.ServeHTTP(w, r)
}
