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
	"net/http"
	"testing"

	"github.com/ethereum/go-ethereum/internal/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

// TestCorsHandler makes sure CORS are properly handled on the http server.
func TestCorsHandler(t *testing.T) {
	srv := createAndStartServer(t, httpConfig{CorsAllowedOrigins: []string{"test", "test.com"}}, false, wsConfig{})
	defer srv.stop()

	resp := testRequest(t, "origin", "test.com", "", srv)
	assert.Equal(t, "test.com", resp.Header.Get("Access-Control-Allow-Origin"))

	resp2 := testRequest(t, "origin", "bad", "", srv)
	assert.Equal(t, "", resp2.Header.Get("Access-Control-Allow-Origin"))
}

// TestVhosts makes sure vhosts are properly handled on the http server.
func TestVhosts(t *testing.T) {
	srv := createAndStartServer(t, httpConfig{Vhosts: []string{"test"}}, false, wsConfig{})
	defer srv.stop()

	resp := testRequest(t, "", "", "test", srv)
	assert.Equal(t, resp.StatusCode, http.StatusOK)

	resp2 := testRequest(t, "", "", "bad", srv)
	assert.Equal(t, resp2.StatusCode, http.StatusForbidden)
}

// TestWebsocketOrigins makes sure the websocket origins are properly handled on the websocket server.
func TestWebsocketOrigins(t *testing.T) {
	tryWebsocketOriginsSimpleRule(t)
	tryWebsocketOriginsRuleWithScheme(t)
	tryWebsocketOriginsIPRuleWithScheme(t)
	tryWebsocketOriginsRuleWithPort(t)
	tryWebsocketOriginsRuleWithSchemeAndPort(t)
}

func tryWebsocketOriginsSimpleRule(t *testing.T) {
	srv := createAndStartServer(t, httpConfig{}, true, wsConfig{Origins: []string{"test"}})
	defer srv.stop()

	assert.NoError(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://test"))
	assert.NoError(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://test"))
	assert.NoError(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://test:8540"))
	assert.NoError(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://test:8540"))

	// Host mismatch (set):
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://bad"))
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://bad"))
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://bad:8540"))
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://bad:8540"))
}

func tryWebsocketOriginsRuleWithScheme(t *testing.T) {
	srv := createAndStartServer(t, httpConfig{}, true, wsConfig{Origins: []string{"https://test"}})
	defer srv.stop()

	// Scheme mismatch:
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://test"))
	assert.NoError(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://test"))

	// Scheme mismatch:
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://test:8540"))
	assert.NoError(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://test:8540"))

	// Host mismatch (set):
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://bad"))
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://bad"))
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://bad:8540"))
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://bad:8540"))
}

func tryWebsocketOriginsIPRuleWithScheme(t *testing.T) {
	srv := createAndStartServer(t, httpConfig{}, true, wsConfig{Origins: []string{"https://12.34.56.78"}})
	defer srv.stop()

	// Scheme mismatch:
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://12.34.56.78"))
	assert.NoError(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://12.34.56.78"))

	// Scheme mismatch:
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://12.34.56.78:8540"))
	assert.NoError(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://12.34.56.78:8540"))

	// Host mismatch (set):
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://87.65.43.21"))
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://87.65.43.21"))
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://87.65.43.21:8540"))
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://87.65.43.21:8540"))
}

func tryWebsocketOriginsRuleWithPort(t *testing.T) {
	srv := createAndStartServer(t, httpConfig{}, true, wsConfig{Origins: []string{"test:8540"}})
	defer srv.stop()

	assert.NoError(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://test"))
	assert.NoError(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://test"))
	assert.NoError(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://test:8540"))
	assert.NoError(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://test:8540"))

	// Port mismatch (set):
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://test:8541"))
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://test:8541"))

	// Host mismatch (set):
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://bad"))
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://bad"))
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://bad:8540"))
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://bad:8540"))
}

func tryWebsocketOriginsRuleWithSchemeAndPort(t *testing.T) {
	srv := createAndStartServer(t, httpConfig{}, true, wsConfig{Origins: []string{"https://test:8540"}})
	defer srv.stop()

	// Scheme mismatch:
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://test"))
	assert.NoError(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://test"))
	// Scheme mismatch:
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://test:8540"))
	assert.NoError(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://test:8540"))

	// Port mismatch (set):
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://test:8541"))
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://test:8541"))

	// Host mismatch (set):
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://bad"))
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://bad"))
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "http://bad:8540"))
	assert.Error(t, attemptWebsocketConnectionFromOrigin(t, srv, "https://bad:8540"))
}

// TestIsWebsocket tests if an incoming websocket upgrade request is handled properly.
func TestIsWebsocket(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)

	assert.False(t, isWebsocket(r))
	r.Header.Set("upgrade", "websocket")
	assert.False(t, isWebsocket(r))
	r.Header.Set("connection", "upgrade")
	assert.True(t, isWebsocket(r))
	r.Header.Set("connection", "upgrade,keep-alive")
	assert.True(t, isWebsocket(r))
	r.Header.Set("connection", " UPGRADE,keep-alive")
	assert.True(t, isWebsocket(r))
}

func createAndStartServer(t *testing.T, conf httpConfig, ws bool, wsConf wsConfig) *httpServer {
	t.Helper()

	srv := newHTTPServer(testlog.Logger(t, log.LvlDebug), rpc.DefaultHTTPTimeouts)

	assert.NoError(t, srv.enableRPC(nil, conf))
	if ws {
		assert.NoError(t, srv.enableWS(nil, wsConf))
	}
	assert.NoError(t, srv.setListenAddr("localhost", 0))
	assert.NoError(t, srv.start())

	return srv
}

func attemptWebsocketConnectionFromOrigin(t *testing.T, srv *httpServer, browserOrigin string) error {
	t.Helper()
	dialer := websocket.DefaultDialer
	_, _, err := dialer.Dial("ws://"+srv.listenAddr(), http.Header{
		"Content-type":          []string{"application/json"},
		"Sec-WebSocket-Version": []string{"13"},
		"Origin":                []string{browserOrigin},
	})
	return err
}

func testRequest(t *testing.T, key, value, host string, srv *httpServer) *http.Response {
	t.Helper()

	body := bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,method":"rpc_modules"}`))
	req, _ := http.NewRequest("POST", "http://"+srv.listenAddr(), body)
	req.Header.Set("content-type", "application/json")
	if key != "" && value != "" {
		req.Header.Set(key, value)
	}
	if host != "" {
		req.Host = host
	}

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}
