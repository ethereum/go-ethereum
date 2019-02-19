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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/storage/mock/mem"
)

// TestHandler_CORSOrigin validates that the correct Access-Control-Allow-Origin
// header is served with various allowed origin settings.
func TestHandler_CORSOrigin(t *testing.T) {
	notAllowedOrigin := "http://not-allowed-origin.com/"

	for _, tc := range []struct {
		name    string
		origins []string
	}{
		{
			name:    "no origin",
			origins: nil,
		},
		{
			name:    "single origin",
			origins: []string{"http://localhost/"},
		},
		{
			name:    "multiple origins",
			origins: []string{"http://localhost/", "http://ethereum.org/"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewHandler(mem.NewGlobalStore(), tc.origins)

			origins := tc.origins
			if origins == nil {
				// handle the "no origin" test case
				origins = []string{""}
			}

			for _, origin := range origins {
				t.Run(fmt.Sprintf("get %q", origin), newTestCORSOrigin(handler, origin, origin))
				t.Run(fmt.Sprintf("preflight %q", origin), newTestCORSPreflight(handler, origin, origin))
			}

			t.Run(fmt.Sprintf("get %q", notAllowedOrigin), newTestCORSOrigin(handler, notAllowedOrigin, ""))
			t.Run(fmt.Sprintf("preflight %q", notAllowedOrigin), newTestCORSPreflight(handler, notAllowedOrigin, ""))
		})
	}

	t.Run("wildcard", func(t *testing.T) {
		handler := NewHandler(mem.NewGlobalStore(), []string{"*"})

		for _, origin := range []string{
			"http://example.com/",
			"http://ethereum.org",
			"http://localhost",
		} {
			t.Run(fmt.Sprintf("get %q", origin), newTestCORSOrigin(handler, origin, origin))
			t.Run(fmt.Sprintf("preflight %q", origin), newTestCORSPreflight(handler, origin, origin))
		}
	})
}

// newTestCORSOrigin returns a test function that validates if wantOrigin CORS header is
// served by the handler for a GET request.
func newTestCORSOrigin(handler http.Handler, origin, wantOrigin string) func(t *testing.T) {
	return func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", origin)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()

		header := resp.Header.Get("Access-Control-Allow-Origin")
		if header != wantOrigin {
			t.Errorf("got Access-Control-Allow-Origin header %q, want %q", header, wantOrigin)
		}
	}
}

// newTestCORSPreflight returns a test function that validates if wantOrigin CORS header is
// served by the handler for an OPTIONS CORS preflight request.
func newTestCORSPreflight(handler http.Handler, origin, wantOrigin string) func(t *testing.T) {
	return func(t *testing.T) {
		req, err := http.NewRequest(http.MethodOptions, "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "GET")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()

		header := resp.Header.Get("Access-Control-Allow-Origin")
		if header != wantOrigin {
			t.Errorf("got Access-Control-Allow-Origin header %q, want %q", header, wantOrigin)
		}
	}
}

// TestHandler_noCacheHeaders validates that no cache headers are server.
func TestHandler_noCacheHeaders(t *testing.T) {
	handler := NewHandler(mem.NewGlobalStore(), nil)

	for _, tc := range []struct {
		url string
	}{
		{
			url: "/",
		},
		{
			url: "/api/nodes",
		},
		{
			url: "/api/keys",
		},
	} {
		req, err := http.NewRequest(http.MethodGet, tc.url, nil)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		resp := w.Result()

		for header, want := range map[string]string{
			"Cache-Control": "no-cache, no-store, must-revalidate",
			"Pragma":        "no-cache",
			"Expires":       "0",
		} {
			got := resp.Header.Get(header)
			if got != want {
				t.Errorf("got %q header %q for url %q, want %q", header, tc.url, got, want)
			}
		}
	}
}
