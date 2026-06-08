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
const BasePath = "/engine/v2"

// supportedForks lists the fork URL segments the router recognises. Order is
// chronological; the router does an exact prefix match so /amsterdam/payloads
// is not confused with /amsterdam-foo/payloads.
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

// handleForkScoped routes /{fork}/<endpoint>... paths.
func (rt *Router) handleForkScoped(w http.ResponseWriter, r *http.Request, path string) {
	if len(path) < 2 || path[0] != '/' {
		writeProblem(w, http.StatusNotFound, ErrMethodNotFound, "")
		return
	}
	rest := path[1:]
	slash := strings.IndexByte(rest, '/')
	if slash < 0 {
		writeProblem(w, http.StatusNotFound, ErrMethodNotFound, "")
		return
	}
	forkName, sub := rest[:slash], rest[slash:]
	fork, ok := supportedForks[forkName]
	if !ok {
		writeProblem(w, http.StatusBadRequest, ErrUnsupportedFork, "unknown fork "+forkName)
		return
	}
	switch {
	case sub == "/payloads" && r.Method == http.MethodPost:
		rt.handleNewPayload(w, r, fork)
	case sub == "/forkchoice" && r.Method == http.MethodPost:
		rt.handleForkchoice(w, r, fork)
	case strings.HasPrefix(sub, "/payloads/") && r.Method == http.MethodGet:
		rt.handleGetPayload(w, r, fork, sub[len("/payloads/"):])
	case sub == "/bodies/hash" && r.Method == http.MethodPost:
		rt.handleBodiesByHash(w, r, fork)
	case sub == "/bodies" && r.Method == http.MethodGet:
		rt.handleBodiesByRange(w, r, fork)
	default:
		writeProblem(w, http.StatusNotFound, ErrMethodNotFound, "")
	}
}
