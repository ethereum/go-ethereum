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

package rpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"go.opentelemetry.io/otel/codes"
)

type notificationTimeoutService struct {
	canceled bool
}

func (s *notificationTimeoutService) Wait(ctx context.Context) {
	<-ctx.Done()
	s.canceled = true
	// Let the timeout callback finish before the method returns. synctest
	// advances this delay without relying on wall-clock scheduling.
	time.Sleep(time.Second)
}

func TestNotificationTimeout(t *testing.T) {
	for _, tc := range []struct {
		name, request, response string
	}{
		{
			name:    "notification",
			request: `{"jsonrpc":"2.0","method":"timeout_wait"}`,
		},
		{
			name:     "call",
			request:  `{"jsonrpc":"2.0","id":1,"method":"timeout_wait"}`,
			response: `{"jsonrpc":"2.0","id":1,"error":{"code":-32002,"message":"request timed out"}}`,
		},
		{
			name:     "null ID call",
			request:  `{"jsonrpc":"2.0","id":null,"method":"timeout_wait"}`,
			response: `{"jsonrpc":"2.0","id":null,"error":{"code":-32002,"message":"request timed out"}}`,
		},
		{
			name:    "notification batch",
			request: `[{"jsonrpc":"2.0","method":"timeout_wait"}]`,
		},
		{
			name:     "mixed batch",
			request:  `[{"jsonrpc":"2.0","method":"timeout_wait"},{"jsonrpc":"2.0","id":1,"method":"test_echo"}]`,
			response: `[{"jsonrpc":"2.0","id":1,"error":{"code":-32002,"message":"request timed out"}}]`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				server, _, exporter := newTracingServer(t)
				service := new(notificationTimeoutService)
				if err := server.RegisterName("timeout", service); err != nil {
					t.Fatal(err)
				}
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.request))
				req.Header.Set("Content-Type", "application/json")
				// Supply the HTTP timeout context without an actual socket deadline.
				// The RPC timer fires after one second of synctest's virtual time.
				ctx := context.WithValue(req.Context(), http.ServerContextKey, &http.Server{WriteTimeout: 1100 * time.Millisecond})
				w := httptest.NewRecorder()
				server.ServeHTTP(w, req.WithContext(ctx))
				if !service.canceled {
					t.Fatal("timeout did not cancel the running method")
				}
				if got := strings.TrimSpace(w.Body.String()); got != tc.response {
					t.Fatalf("response: got %q, want %q", got, tc.response)
				}
				if tc.name == "notification" {
					found := false
					for _, span := range exporter.GetSpans() {
						if span.Name == "rpc.writeJSON" {
							t.Error("notification timeout wrote a JSON-RPC response")
						}
						if span.Name == "jsonrpc.timeout/wait" {
							found = true
							if span.Status.Code != codes.Error || span.Status.Description != errMsgTimeout {
								t.Errorf("timeout span status: %v", span.Status)
							}
						}
					}
					if !found {
						t.Error("notification timeout span missing")
					}
				}
			})
		})
	}
}
