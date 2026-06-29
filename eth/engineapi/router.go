// Copyright 2026 The go-ethereum Authors
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

package engineapi

import (
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/params/forks"
)

// BasePath is the mount prefix for the REST Engine API. Handlers below assume
// requests have already been routed under this prefix.
const BasePath = "/engine/v1"

// forkHeader carries the fork name on hot-path requests, replacing the former
// /{fork}/ URL segment. Mirrors the Beacon API's Eth-Consensus-Version idiom.
const forkHeader = "Eth-Execution-Version"

// supportedForks lists the fork names the router recognises in the
// Eth-Execution-Version header. Lookup is case-insensitive (keys are lower).
var supportedForks = map[string]forks.Fork{
	"paris":     forks.Paris,
	"shanghai":  forks.Shanghai,
	"cancun":    forks.Cancun,
	"prague":    forks.Prague,
	"osaka":     forks.Osaka,
	"amsterdam": forks.Amsterdam,
}

// Router is the http.Handler implementing the REST Engine API.
type Router struct {
	backend Backend
}

// NewRouter constructs a Router backed by b.
func NewRouter(b Backend) *Router {
	return &Router{backend: b}
}

// ServeHTTP dispatches to the per-endpoint handler. The caller is expected to
// have stripped the BasePath prefix and applied authentication.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	// The spec forbids trailing slashes; net/http's ServeMux would redirect,
	// so police it here.
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		writeProblem(w, http.StatusNotFound, ErrMethodNotFound, "")
		return
	}
	// Unscoped endpoints ignore the Eth-Execution-Version header; fork-scoped
	// endpoints resolve the fork from it (see handleForkScoped).
	switch {
	case path == "/capabilities":
		rt.handleCapabilities(w, r)
	case path == "/identity":
		rt.handleIdentity(w, r)
	case strings.HasPrefix(path, "/blobs/"):
		rt.handleBlobs(w, r, path[len("/blobs/"):])
	default:
		rt.handleForkScoped(w, r, path)
	}
}

// handleForkScoped routes the fork-scoped endpoints (/payloads, /forkchoice,
// /bodies). The fork is selected by the Eth-Execution-Version request header
// rather than a URL segment. The path is matched first so an entirely unknown
// route yields method-not-found rather than masking it as a fork error.
func (rt *Router) handleForkScoped(w http.ResponseWriter, r *http.Request, path string) {
	var handler func(w http.ResponseWriter, r *http.Request, fork forks.Fork)
	switch {
	case path == "/payloads" && r.Method == http.MethodPost:
		handler = rt.handleNewPayload
	case path == "/forkchoice" && r.Method == http.MethodPost:
		handler = rt.handleForkchoice
	case strings.HasPrefix(path, "/payloads/") && r.Method == http.MethodGet:
		id := path[len("/payloads/"):]
		handler = func(w http.ResponseWriter, r *http.Request, fork forks.Fork) {
			rt.handleGetPayload(w, r, fork, id)
		}
	case path == "/bodies/hash" && r.Method == http.MethodPost:
		handler = rt.handleBodiesByHash
	case path == "/bodies" && r.Method == http.MethodGet:
		handler = rt.handleBodiesByRange
	default:
		writeProblem(w, http.StatusNotFound, ErrMethodNotFound, "")
		return
	}
	fork, ok := rt.forkFromHeader(w, r)
	if !ok {
		return
	}
	handler(w, r, fork)
}

// forkFromHeader resolves the Eth-Execution-Version request header to a fork.
// A missing or unrecognised value writes a 400 unsupported-fork problem and
// returns ok=false; callers must stop on false.
func (rt *Router) forkFromHeader(w http.ResponseWriter, r *http.Request) (forks.Fork, bool) {
	name := strings.ToLower(strings.TrimSpace(r.Header.Get(forkHeader)))
	if name == "" {
		writeProblem(w, http.StatusBadRequest, ErrUnsupportedFork, "missing "+forkHeader+" header")
		return 0, false
	}
	fork, ok := supportedForks[name]
	if !ok {
		writeProblem(w, http.StatusBadRequest, ErrUnsupportedFork, "unknown fork "+name)
		return 0, false
	}
	return fork, true
}
