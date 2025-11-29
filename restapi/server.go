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

package restapi

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/elnormous/contenttype"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/gorilla/mux"
)

type Server struct {
	router *mux.Router
}

type API func(*mux.Router)

type WrappedHandler func(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int)

func NewServer(node *node.Node) *Server {
	s := &Server{
		router: mux.NewRouter(),
	}
	node.RegisterHandler("REST API", "/eth/", s.router)
	return s
}

func (s *Server) Register(regAPI API) {
	regAPI(s.router)
}

func mediaType(mt contenttype.MediaType, allowBinary bool) (binary, valid bool) {
	switch {
	case mt.Type == "" && mt.Subtype == "":
		return false, true // if content type is not specified then assume JSON
	case mt.Type == "application" && mt.Subtype == "json":
		return false, true
	case mt.Type == "application" && mt.Subtype == "octet-stream":
		return allowBinary, allowBinary
	default:
		return false, false
	}
}

var allAvailableMediaTypes = []contenttype.MediaType{
	contenttype.NewMediaType("application/json"),
	contenttype.NewMediaType("application/octet-stream"),
}

func (s *Server) WrapHandler(handler WrappedHandler, expectBody, allowRlpBody, allowRlpResponse bool) func(resp http.ResponseWriter, req *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		var decodeBody func(*any) error
		if expectBody {
			contentType, err := contenttype.GetMediaType(req)
			if err != nil {
				http.Error(resp, "invalid content type", http.StatusUnsupportedMediaType)
				return
			}
			binary, valid := mediaType(contentType, allowRlpBody)
			if !valid {
				http.Error(resp, "invalid content type", http.StatusUnsupportedMediaType)
				return
			}
			if req.Body == nil {
				http.Error(resp, "missing request body", http.StatusBadRequest)
				return
			}
			data, err := ioutil.ReadAll(req.Body)
			if err != nil {
				http.Error(resp, "could not read request body", http.StatusInternalServerError)
				return
			}
			if binary {
				decodeBody = func(body *any) error {
					return rlp.DecodeBytes(data, body)
				}
			} else {
				decodeBody = func(body *any) error {
					return json.Unmarshal(data, body)
				}
			}
		}

		availableMediaTypes := allAvailableMediaTypes
		if !allowRlpResponse {
			availableMediaTypes = availableMediaTypes[:1]
		}
		acceptType, _, err := contenttype.GetAcceptableMediaType(req, availableMediaTypes)
		if err != nil {
			http.Error(resp, "invalid accepted media type", http.StatusNotAcceptable)
			return
		}
		binary, valid := mediaType(acceptType, allowRlpResponse)
		if !valid {
			http.Error(resp, "invalid accepted media type", http.StatusNotAcceptable)
			return
		}
		response, errorStr, errorCode := handler(req.Context(), req.URL.Query(), mux.Vars(req), decodeBody)
		if errorCode != 0 {
			http.Error(resp, errorStr, errorCode)
			return
		}
		if binary {
			respRlp, err := rlp.EncodeToBytes(response)
			if err != nil {
				http.Error(resp, "response encoding error", http.StatusInternalServerError)
				return
			}
			resp.Header().Set("content-type", "application/octet-stream")
			resp.Write(respRlp)
		} else {
			respJson, err := json.Marshal(response)
			if err != nil {
				http.Error(resp, "response encoding error", http.StatusInternalServerError)
				return
			}
			resp.Header().Set("content-type", "application/json")
			resp.Write(respJson)
		}
	}
}

func (s *Server) WrapEventHandler(handler func(resp http.ResponseWriter, req *http.Request)) func(resp http.ResponseWriter, req *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		rc := http.NewResponseController(resp)
		if err := rc.SetReadDeadline(time.Time{}); err != nil {
			log.Error("Could not set read deadline for events request", "error", err)
		}
		if err := rc.SetWriteDeadline(time.Time{}); err != nil {
			log.Error("Could not set read deadline for events request", "error", err)
		}
		handler(resp, req)
	}
}
