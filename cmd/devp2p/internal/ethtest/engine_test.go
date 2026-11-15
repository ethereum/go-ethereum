// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package ethtest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
)

func TestEngineClientCall(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   string // substring of the expected error, empty for success
	}{
		{"ok", http.StatusOK, `{"jsonrpc":"2.0","id":1,"result":{"payloadStatus":{"status":"VALID"}}}`, ""},
		{"rpc error", http.StatusOK, `{"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"Invalid params"}}`, "Invalid params"},
		{"http error", http.StatusUnauthorized, `token is expired`, "401"},
		{"garbage", http.StatusOK, `not json`, "invalid response"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			ec := NewEngineClient(srv.URL, "", nil)
			err := ec.call(nil, "engine_test")
			if tt.want == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got error %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestEngineClientForkchoiceRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"payloadStatus":{"status":"SYNCING","validationError":"behind"}}}`))
	}))
	defer srv.Close()

	chain := &Chain{blocks: []*types.Block{types.NewBlockWithHeader(&types.Header{})}}
	err := NewEngineClient(srv.URL, "", chain).sendForkchoiceUpdated(nil)
	if err == nil || !strings.Contains(err.Error(), "SYNCING: behind") {
		t.Fatalf("got error %v, want SYNCING rejection", err)
	}
}
