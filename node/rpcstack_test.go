package node

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
)

func TestNewWebsocketUpgradeHandler_websocket(t *testing.T) {
	h := &HTTPServer{
		Srv:       rpc.NewServer(),
		WSAllowed: true,
	}
	handler := h.NewWebsocketUpgradeHandler(nil, h.Srv.WebsocketHandler([]string{}))
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
			t.Fatalf("could not issue a GET request to the test http server  %v", err)
		}
		responses <- resp
	}(responses)

	response := <-responses
	assert.Equal(t, "websocket", response.Header.Get("Upgrade"))
}

// Tests that a ws handler can be added to and enabled on an existing HTTPServers
func TestWSAllowed(t *testing.T) {
	stack, err := New(&Config{
		HTTPHost: DefaultHTTPHost,
		HTTPPort: 9393,
		WSHost:   DefaultHTTPHost,
		WSPort:   9393,
	})
	if err != nil {
		t.Fatalf("could not create node: %v", err)
	}
	defer stack.Close()
	// start node
	err = stack.Start()
	if err != nil {
		t.Fatalf("could not start node: %v", err)
	}
	// check that server was configured on the given endpoint
	server := stack.ExistingHTTPServer(fmt.Sprintf("%s:%d", DefaultHTTPHost, 9393))
	if server == nil {
		t.Fatalf("server was not started on the given endpoint: %v", err)
	}
	// assert that both RPC and WS are allowed on the HTTP Server
	assert.True(t, server.RPCAllowed)
	assert.True(t, server.WSAllowed)
}
