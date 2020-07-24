package node

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/internal/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

// func TestNewWebsocketUpgradeHandler_websocket(t *testing.T) {
// 	h := &httpServer{
// 		Srv:       rpc.NewServer(),
// 		WSAllowed: 1,
// 	}
// 	handler := h.NewWebsocketUpgradeHandler(nil, h.Srv.WebsocketHandler([]string{}))
// 	ts := httptest.NewServer(handler)
// 	defer ts.Close()
//
// 	responses := make(chan *http.Response)
// 	go func(responses chan *http.Response) {
// 		client := &http.Client{}
//
// 		req, _ := http.NewRequest(http.MethodGet, ts.URL, nil)
// 		req.Header.Set("Connection", "upgrade")
// 		req.Header.Set("Upgrade", "websocket")
// 		req.Header.Set("Sec-WebSocket-Version", "13")
// 		req.Header.Set("Sec-Websocket-Key", "SGVsbG8sIHdvcmxkIQ==")
//
// 		resp, err := client.Do(req)
// 		if err != nil {
// 			t.Fatalf("could not issue a GET request to the test http server  %v", err)
// 		}
// 		responses <- resp
// 	}(responses)
//
// 	response := <-responses
// 	assert.Equal(t, "websocket", response.Header.Get("Upgrade"))
// }

// Tests that a ws handler can be added to and enabled on an existing HTTPServer
func TestWSAllowed(t *testing.T) {
	stack, err := New(&Config{
		HTTPHost: "127.0.0.1",
		HTTPPort: 0,
		WSHost:   "127.0.0.1",
		WSPort:   0,
		Logger:   testlog.Logger(t, log.LvlDebug),
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

	// check that HTTP works on the endpoint.
	url := stack.HTTPEndpoint()
	if err := checkModules(url, stack.Config().WSModules); err != nil {
		t.Fatal(err)
	}

	// check that WS works on the same endpoint.
	wsURL := strings.Replace(url, "http://", "ws://", 1)
	if err := checkModules(wsURL, stack.Config().WSModules); err != nil {
		t.Fatal(err)
	}
}

func checkModules(url string, want []string) error {
	c, err := rpc.Dial(url)
	if err != nil {
		return fmt.Errorf("can't create RPC client: %v", err)
	}
	defer c.Close()

	_, err = c.SupportedModules()
	if err != nil {
		return fmt.Errorf("can't get modules: %v", err)
	}

	// TODO: check module list
	return nil
}
