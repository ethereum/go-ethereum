package node

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/rpc"
	"github.com/stretchr/testify/assert"
)

func TestNewWebsocketUpgradeHandler_websocket(t *testing.T) {
	srv := rpc.NewServer()

	handler := NewWebsocketUpgradeHandler(nil, srv.WebsocketHandler([]string{}))
	ts := httptest.NewServer(handler)
	defer ts.Close()

	responses := make(chan *http.Response)
	go func(responses chan *http.Response) {
		client := &http.Client{}

		req, _ := http.NewRequest(http.MethodGet, ts.URL, nil)
		req.Header.Set("Connection", "upgrade")
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Sec-WebSocket-Version", "13")
		req.Header.Set("Sec-Websocket-Key", "SGVsbG8sIHdvcmxkIQ==")

		resp, err := client.Do(req)
		if err != nil {
			t.Error("could not issue a GET request to the test http server", err)
		}
		responses <- resp
	}(responses)

	response := <-responses
	assert.Equal(t, "websocket", response.Header.Get("Upgrade"))
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
