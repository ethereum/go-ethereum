package node

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/internal/testlog"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/rpc"
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
	srv := createAndStartServer(t, httpConfig{}, true, wsConfig{Origins: []string{"test"}})
	defer srv.stop()

	dialer := websocket.DefaultDialer
	_, _, err := dialer.Dial("ws://"+srv.listenAddr(), http.Header{
		"Content-type":          []string{"application/json"},
		"Sec-WebSocket-Version": []string{"13"},
		"Origin":                []string{"test"},
	})
	assert.NoError(t, err)

	_, _, err = dialer.Dial("ws://"+srv.listenAddr(), http.Header{
		"Content-type":          []string{"application/json"},
		"Sec-WebSocket-Version": []string{"13"},
		"Origin":                []string{"bad"},
	})
	assert.Error(t, err)
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

// TestIsWebsocket tests if an incoming websocket upgrade request is handled properly.
func TestIsWebsocket(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)

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
