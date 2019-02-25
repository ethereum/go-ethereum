// Copyright 2019 The go-ethereum Authors
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

package explorer

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
	"github.com/rs/cors"
)

const jsonContentType = "application/json; charset=utf-8"

// NewHandler constructs an http.Handler with router
// that servers requests required by chunk explorer.
//
//   /api/has-key/{node}/{key}
//   /api/keys?start={key}&node={node}&limit={int[0..1000]}
//   /api/nodes?start={node}&key={key}&limit={int[0..1000]}
//
// Data from global store will be served and appropriate
// CORS headers will be sent if allowed origins are provided.
func NewHandler(store mock.GlobalStorer, corsOrigins []string) (handler http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/api/has-key/", newHasKeyHandler(store))
	mux.Handle("/api/keys", newKeysHandler(store))
	mux.Handle("/api/nodes", newNodesHandler(store))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		jsonStatusResponse(w, http.StatusNotFound)
	})
	handler = noCacheHandler(mux)
	if corsOrigins != nil {
		handler = cors.New(cors.Options{
			AllowedOrigins: corsOrigins,
			AllowedMethods: []string{"GET"},
			MaxAge:         600,
		}).Handler(handler)
	}
	return handler
}

// newHasKeyHandler returns a new handler that serves
// requests for HasKey global store method.
// Possible responses are StatusResponse with
// status codes 200 or 404 if the chunk is found or not.
func newHasKeyHandler(store mock.GlobalStorer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		addr, key, ok := parseHasKeyPath(r.URL.Path)
		if !ok {
			jsonStatusResponse(w, http.StatusNotFound)
			return
		}
		found := store.HasKey(addr, key)
		if !found {
			jsonStatusResponse(w, http.StatusNotFound)
			return
		}
		jsonStatusResponse(w, http.StatusOK)
	}
}

// KeysResponse is a JSON-encoded response for global store
// Keys and NodeKeys methods.
type KeysResponse struct {
	Keys []string `json:"keys"`
	Next string   `json:"next,omitempty"`
}

// newKeysHandler returns a new handler that serves
// requests for Key global store method.
// HTTP response body will be JSON-encoded KeysResponse.
func newKeysHandler(store mock.GlobalStorer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		node := q.Get("node")
		start, limit := listingPage(q)

		var keys mock.Keys
		if node == "" {
			var err error
			keys, err = store.Keys(common.Hex2Bytes(start), limit)
			if err != nil {
				log.Error("chunk explorer: keys handler: get keys", "start", start, "err", err)
				jsonStatusResponse(w, http.StatusInternalServerError)
				return
			}
		} else {
			var err error
			keys, err = store.NodeKeys(common.HexToAddress(node), common.Hex2Bytes(start), limit)
			if err != nil {
				log.Error("chunk explorer: keys handler: get node keys", "node", node, "start", start, "err", err)
				jsonStatusResponse(w, http.StatusInternalServerError)
				return
			}
		}
		ks := make([]string, len(keys.Keys))
		for i, k := range keys.Keys {
			ks[i] = common.Bytes2Hex(k)
		}
		data, err := json.Marshal(KeysResponse{
			Keys: ks,
			Next: common.Bytes2Hex(keys.Next),
		})
		if err != nil {
			log.Error("chunk explorer: keys handler: json marshal", "err", err)
			jsonStatusResponse(w, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", jsonContentType)
		_, err = io.Copy(w, bytes.NewReader(data))
		if err != nil {
			log.Error("chunk explorer: keys handler: write response", "err", err)
		}
	}
}

// NodesResponse is a JSON-encoded response for global store
// Nodes and KeyNodes methods.
type NodesResponse struct {
	Nodes []string `json:"nodes"`
	Next  string   `json:"next,omitempty"`
}

// newNodesHandler returns a new handler that serves
// requests for Nodes global store method.
// HTTP response body will be JSON-encoded NodesResponse.
func newNodesHandler(store mock.GlobalStorer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		key := q.Get("key")
		var start *common.Address
		queryStart, limit := listingPage(q)
		if queryStart != "" {
			s := common.HexToAddress(queryStart)
			start = &s
		}

		var nodes mock.Nodes
		if key == "" {
			var err error
			nodes, err = store.Nodes(start, limit)
			if err != nil {
				log.Error("chunk explorer: nodes handler: get nodes", "start", queryStart, "err", err)
				jsonStatusResponse(w, http.StatusInternalServerError)
				return
			}
		} else {
			var err error
			nodes, err = store.KeyNodes(common.Hex2Bytes(key), start, limit)
			if err != nil {
				log.Error("chunk explorer: nodes handler: get key nodes", "key", key, "start", queryStart, "err", err)
				jsonStatusResponse(w, http.StatusInternalServerError)
				return
			}
		}
		ns := make([]string, len(nodes.Addrs))
		for i, n := range nodes.Addrs {
			ns[i] = n.Hex()
		}
		var next string
		if nodes.Next != nil {
			next = nodes.Next.Hex()
		}
		data, err := json.Marshal(NodesResponse{
			Nodes: ns,
			Next:  next,
		})
		if err != nil {
			log.Error("chunk explorer: nodes handler", "err", err)
			jsonStatusResponse(w, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", jsonContentType)
		_, err = io.Copy(w, bytes.NewReader(data))
		if err != nil {
			log.Error("chunk explorer: nodes handler: write response", "err", err)
		}
	}
}

// parseHasKeyPath extracts address and key from HTTP request
// path for HasKey route: /api/has-key/{node}/{key}.
// If ok is false, the provided path is not matched.
func parseHasKeyPath(p string) (addr common.Address, key []byte, ok bool) {
	p = strings.TrimPrefix(p, "/api/has-key/")
	parts := strings.SplitN(p, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return addr, nil, false
	}
	addr = common.HexToAddress(parts[0])
	key = common.Hex2Bytes(parts[1])
	return addr, key, true
}

// listingPage returns start value and listing limit
// from url query values.
func listingPage(q url.Values) (start string, limit int) {
	// if limit is not a valid integer (or blank string),
	// ignore the error and use the returned 0 value
	limit, _ = strconv.Atoi(q.Get("limit"))
	return q.Get("start"), limit
}

// StatusResponse is a standardized JSON-encoded response
// that contains information about HTTP response code
// for easier status identification.
type StatusResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// jsonStatusResponse writes to the response writer
// JSON-encoded StatusResponse based on the provided status code.
func jsonStatusResponse(w http.ResponseWriter, code int) {
	w.Header().Set("Content-Type", jsonContentType)
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(StatusResponse{
		Message: http.StatusText(code),
		Code:    code,
	})
	if err != nil {
		log.Error("chunk explorer: json status response", "err", err)
	}
}

// noCacheHandler sets required HTTP headers to prevent
// response caching at the client side.
func noCacheHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		h.ServeHTTP(w, r)
	})
}
