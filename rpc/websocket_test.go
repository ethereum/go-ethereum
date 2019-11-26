// Copyright 2018 The go-ethereum Authors
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
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebsocketClientHeaders(t *testing.T) {
	t.Parallel()

	endpoint, header, err := wsClientHeaders("wss://testuser:test-PASS_01@example.com:1234", "https://example.com")
	if err != nil {
		t.Fatalf("wsGetConfig failed: %s", err)
	}
	if endpoint != "wss://example.com:1234" {
		t.Fatal("User should have been stripped from the URL")
	}
	if header.Get("authorization") != "Basic dGVzdHVzZXI6dGVzdC1QQVNTXzAx" {
		t.Fatal("Basic auth header is incorrect")
	}
	if header.Get("origin") != "https://example.com" {
		t.Fatal("Origin not set")
	}
}

// This test checks that the server rejects connections from disallowed origins.
func TestWebsocketOriginCheck(t *testing.T) {
	t.Parallel()

	var (
		srv     = newTestServer()
		httpsrv = httptest.NewServer(srv.WebsocketHandler([]string{"http://example.com"}))
		wsURL   = "ws:" + strings.TrimPrefix(httpsrv.URL, "http:")
	)
	defer srv.Stop()
	defer httpsrv.Close()

	client, err := DialWebsocket(context.Background(), wsURL, "http://ekzample.com")
	if err == nil {
		client.Close()
		t.Fatal("no error for wrong origin")
	}
	wantErr := wsHandshakeError{websocket.ErrBadHandshake, "403 Forbidden"}
	if !reflect.DeepEqual(err, wantErr) {
		t.Fatalf("wrong error for wrong origin: %q", err)
	}

	// Connections without origin header should work.
	client, err = DialWebsocket(context.Background(), wsURL, "")
	if err != nil {
		t.Fatal("error for empty origin")
	}
	client.Close()
}

// This test checks whether calls exceeding the request size limit are rejected.
func TestWebsocketLargeCall(t *testing.T) {
	t.Parallel()

	var (
		srv     = newTestServer()
		httpsrv = httptest.NewServer(srv.WebsocketHandler([]string{"*"}))
		wsURL   = "ws:" + strings.TrimPrefix(httpsrv.URL, "http:")
	)
	defer srv.Stop()
	defer httpsrv.Close()

	client, err := DialWebsocket(context.Background(), wsURL, "")
	if err != nil {
		t.Fatalf("can't dial: %v", err)
	}
	defer client.Close()

	// This call sends slightly less than the limit and should work.
	var result echoResult
	arg := strings.Repeat("x", maxRequestContentLength-200)
	if err := client.Call(&result, "test_echo", arg, 1); err != nil {
		t.Fatalf("valid call didn't work: %v", err)
	}
	if result.String != arg {
		t.Fatal("wrong string echoed")
	}

	// This call sends twice the allowed size and shouldn't work.
	arg = strings.Repeat("x", maxRequestContentLength*2)
	err = client.Call(&result, "test_echo", arg)
	if err == nil {
		t.Fatal("no error for too large call")
	}
}

// This test checks that client handles WebSocket ping frames correctly.
func TestClientWebsocketPing(t *testing.T) {
	t.Parallel()

	var (
		sendPing    = make(chan struct{})
		server      = wsPingTestServer(t, sendPing)
		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	)
	defer cancel()
	defer server.Shutdown(ctx)

	client, err := DialContext(ctx, "ws://"+server.Addr)
	if err != nil {
		t.Fatalf("client dial error: %v", err)
	}
	resultChan := make(chan int)
	sub, err := client.EthSubscribe(ctx, resultChan, "foo")
	if err != nil {
		t.Fatalf("client subscribe error: %v", err)
	}

	// Wait for the context's deadline to be reached before proceeding.
	// This is important for reproducing https://github.com/ethereum/go-ethereum/issues/19798
	<-ctx.Done()
	close(sendPing)

	// Wait for the subscription result.
	timeout := time.NewTimer(5 * time.Second)
	for {
		select {
		case err := <-sub.Err():
			t.Error("client subscription error:", err)
		case result := <-resultChan:
			t.Log("client got result:", result)
			return
		case <-timeout.C:
			t.Error("didn't get any result within the test timeout")
			return
		}
	}
}

// wsPingTestServer runs a WebSocket server which accepts a single subscription request.
// When a value arrives on sendPing, the server sends a ping frame, waits for a matching
// pong and finally delivers a single subscription result.
func wsPingTestServer(t *testing.T, sendPing <-chan struct{}) *http.Server {
	var srv http.Server
	shutdown := make(chan struct{})
	srv.RegisterOnShutdown(func() {
		close(shutdown)
	})
	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Upgrade to WebSocket.
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("server WS upgrade error: %v", err)
			return
		}
		defer conn.Close()

		// Handle the connection.
		wsPingTestHandler(t, conn, shutdown, sendPing)
	})

	// Start the server.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("can't listen:", err)
	}
	srv.Addr = listener.Addr().String()
	go srv.Serve(listener)
	return &srv
}

func wsPingTestHandler(t *testing.T, conn *websocket.Conn, shutdown, sendPing <-chan struct{}) {
	// Canned responses for the eth_subscribe call in TestClientWebsocketPing.
	const (
		subResp   = `{"jsonrpc":"2.0","id":1,"result":"0x00"}`
		subNotify = `{"jsonrpc":"2.0","method":"eth_subscription","params":{"subscription":"0x00","result":1}}`
	)

	// Handle subscribe request.
	if _, _, err := conn.ReadMessage(); err != nil {
		t.Errorf("server read error: %v", err)
		return
	}
	if err := conn.WriteMessage(websocket.TextMessage, []byte(subResp)); err != nil {
		t.Errorf("server write error: %v", err)
		return
	}

	// Read from the connection to process control messages.
	var pongCh = make(chan string)
	conn.SetPongHandler(func(d string) error {
		t.Logf("server got pong: %q", d)
		pongCh <- d
		return nil
	})
	go func() {
		for {
			typ, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			t.Logf("server got message (%d): %q", typ, msg)
		}
	}()

	// Write messages.
	var (
		sendResponse <-chan time.Time
		wantPong     string
	)
	for {
		select {
		case _, open := <-sendPing:
			if !open {
				sendPing = nil
			}
			t.Logf("server sending ping")
			conn.WriteMessage(websocket.PingMessage, []byte("ping"))
			wantPong = "ping"
		case data := <-pongCh:
			if wantPong == "" {
				t.Errorf("unexpected pong")
			} else if data != wantPong {
				t.Errorf("got pong with wrong data %q", data)
			}
			wantPong = ""
			sendResponse = time.NewTimer(200 * time.Millisecond).C
		case <-sendResponse:
			t.Logf("server sending response")
			conn.WriteMessage(websocket.TextMessage, []byte(subNotify))
			sendResponse = nil
		case <-shutdown:
			conn.Close()
			return
		}
	}
}
