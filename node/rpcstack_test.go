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
