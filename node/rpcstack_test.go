// Copyright 2020 The go-ethereum Authors
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

package node

import (
	"bytes"
	"github.com/ethereum/go-ethereum/internal/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestCorsHandler(t *testing.T) {
	// TODO put this in separate func
	srv := newHTTPServer(testlog.Logger(t, log.LvlDebug), rpc.DefaultHTTPTimeouts)
	assert.NoError(t, srv.enableRPC(nil, httpConfig{
		CorsAllowedOrigins: []string{"test", "test.com"},
	}))
	assert.NoError(t, srv.setListenAddr("localhost", 0))
	assert.NoError(t, srv.start())
	defer srv.stop()

	resp := testRequest(t, "test.com", srv)
	if resp.Header.Get("Access-Control-Allow-Origin") != "test.com" {
		t.Fatalf("cors not recognized")
	}
	resp2 := testRequest(t, "bad", srv)
	if resp2.Header.Get("Access-Control-Allow-Origin") != ""{
		t.Fatalf("cors not properly set, bad cors recognized")
	}
}

func testRequest(t *testing.T, origin string, srv *httpServer) *http.Response {
	t.Helper()

	body := bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,method":"rpc_modules"}`))
	req, _ := http.NewRequest("POST", "http://" + srv.listenAddr(), body)
	req.Header.Set("content-type", "application/json")
	req.Header.Set("origin", origin)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	return resp
}

func TestVhosts(t *testing.T) {
	// TODO
}

func TestWebsocketOrigins(t *testing.T) {
	// TODO
}


