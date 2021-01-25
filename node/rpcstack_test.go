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
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/internal/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

// TestCorsHandler makes sure CORS are properly handled on the http server.
func TestCorsHandler(t *testing.T) {
	srv := createAndStartServer(t, &httpConfig{CorsAllowedOrigins: []string{"test", "test.com"}}, false, &wsConfig{})
	defer srv.stop()

	resp := testRequest(t, "origin", "test.com", "", srv, "")
	assert.Equal(t, "test.com", resp.Header.Get("Access-Control-Allow-Origin"))

	resp2 := testRequest(t, "origin", "bad", "", srv, "")
	assert.Equal(t, "", resp2.Header.Get("Access-Control-Allow-Origin"))
}

// TestVhosts makes sure vhosts are properly handled on the http server.
func TestVhosts(t *testing.T) {
	srv := createAndStartServer(t, &httpConfig{Vhosts: []string{"test"}}, false, &wsConfig{})
	defer srv.stop()

	resp := testRequest(t, "", "", "test", srv, "")
	assert.Equal(t, resp.StatusCode, http.StatusOK)

	resp2 := testRequest(t, "", "", "bad", srv, "")
	assert.Equal(t, resp2.StatusCode, http.StatusForbidden)
}

type originTest struct {
	spec    string
	expOk   []string
	expFail []string
}

// splitAndTrim splits input separated by a comma
// and trims excessive white space from the substrings.
// Copied over from flags.go
func splitAndTrim(input string) (ret []string) {
	l := strings.Split(input, ",")
	for _, r := range l {
		r = strings.TrimSpace(r)
		if len(r) > 0 {
			ret = append(ret, r)
		}
	}
	return ret
}

// TestWebsocketOrigins makes sure the websocket origins are properly handled on the websocket server.
func TestWebsocketOrigins(t *testing.T) {
	tests := []originTest{
		{
			spec: "*", // allow all
			expOk: []string{"", "http://test", "https://test", "http://test:8540", "https://test:8540",
				"http://test.com", "https://foo.test", "http://testa", "http://atestb:8540", "https://atestb:8540"},
		},
		{
			spec:    "test",
			expOk:   []string{"http://test", "https://test", "http://test:8540", "https://test:8540"},
			expFail: []string{"http://test.com", "https://foo.test", "http://testa", "http://atestb:8540", "https://atestb:8540"},
		},
		// scheme tests
		{
			spec:  "https://test",
			expOk: []string{"https://test", "https://test:9999"},
			expFail: []string{
				"test",                                // no scheme, required by spec
				"http://test",                         // wrong scheme
				"http://test.foo", "https://a.test.x", // subdomain variatoins
				"http://testx:8540", "https://xtest:8540"},
		},
		// ip tests
		{
			spec:  "https://12.34.56.78",
			expOk: []string{"https://12.34.56.78", "https://12.34.56.78:8540"},
			expFail: []string{
				"http://12.34.56.78",     // wrong scheme
				"http://12.34.56.78:443", // wrong scheme
				"http://1.12.34.56.78",   // wrong 'domain name'
				"http://12.34.56.78.a",   // wrong 'domain name'
				"https://87.65.43.21", "http://87.65.43.21:8540", "https://87.65.43.21:8540"},
		},
		// port tests
		{
			spec:  "test:8540",
			expOk: []string{"http://test:8540", "https://test:8540"},
			expFail: []string{
				"http://test", "https://test", // spec says port required
				"http://test:8541", "https://test:8541", // wrong port
				"http://bad", "https://bad", "http://bad:8540", "https://bad:8540"},
		},
		// scheme and port
		{
			spec:  "https://test:8540",
			expOk: []string{"https://test:8540"},
			expFail: []string{
				"https://test",                          // missing port
				"http://test",                           // missing port, + wrong scheme
				"http://test:8540",                      // wrong scheme
				"http://test:8541", "https://test:8541", // wrong port
				"http://bad", "https://bad", "http://bad:8540", "https://bad:8540"},
		},
		// several allowed origins
		{
			spec: "localhost,http://127.0.0.1",
			expOk: []string{"localhost", "http://localhost", "https://localhost:8443",
				"http://127.0.0.1", "http://127.0.0.1:8080"},
			expFail: []string{
				"https://127.0.0.1", // wrong scheme
				"http://bad", "https://bad", "http://bad:8540", "https://bad:8540"},
		},
	}
	for _, tc := range tests {
		srv := createAndStartServer(t, &httpConfig{}, true, &wsConfig{Origins: splitAndTrim(tc.spec)})
		for _, origin := range tc.expOk {
			if err := attemptWebsocketConnectionFromOrigin(t, srv, origin); err != nil {
				t.Errorf("spec '%v', origin '%v': expected ok, got %v", tc.spec, origin, err)
			}
		}
		for _, origin := range tc.expFail {
			if err := attemptWebsocketConnectionFromOrigin(t, srv, origin); err == nil {
				t.Errorf("spec '%v', origin '%v': expected not to allow,  got ok", tc.spec, origin)
			}
		}
		srv.stop()
	}
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

func Test_checkPath(t *testing.T) {
	tests := []struct {
		req      *http.Request
		prefix   string
		expected bool
	}{
		{
			req:      &http.Request{URL: &url.URL{Path: "/test"}},
			prefix:   "/test",
			expected: true,
		},
		{
			req:      &http.Request{URL: &url.URL{Path: "/testing"}},
			prefix:   "/test",
			expected: true,
		},
		{
			req:      &http.Request{URL: &url.URL{Path: "/"}},
			prefix:   "/test",
			expected: false,
		},
		{
			req:      &http.Request{URL: &url.URL{Path: "/fail"}},
			prefix:   "/test",
			expected: false,
		},
		{
			req:      &http.Request{URL: &url.URL{Path: "/"}},
			prefix:   "",
			expected: true,
		},
		{
			req:      &http.Request{URL: &url.URL{Path: "/fail"}},
			prefix:   "",
			expected: false,
		},
		{
			req:      &http.Request{URL: &url.URL{Path: "/"}},
			prefix:   "/",
			expected: true,
		},
		{
			req:      &http.Request{URL: &url.URL{Path: "/testing"}},
			prefix:   "/",
			expected: true,
		},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.expected, checkPath(tt.req, tt.prefix))
		})
	}
}

// Test_prettyPath makes sure that an acceptable path prefix is returned,
// if one has been specified.
func Test_prettyPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{
			path:     "test",
			expected: "/test",
		},
		{
			path:     "/",
			expected: "/",
		},
		{
			path:     "/test",
			expected: "/test",
		},
		{
			path:     "",
			expected: "",
		},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.expected, prettyPath(tt.path))
		})
	}
}

// Test_queryString tests to make sure query strings are ignored by checkPath
func Test_queryString(t *testing.T) {
	tests := []struct {
		rawPath  string
		prefix   string
		expected bool
	}{
		{
			rawPath:  "/test?test=test",
			prefix:   "/test",
			expected: true,
		},
		{
			rawPath:  "/testing?test=test",
			prefix:   "/test",
			expected: true,
		},
		{
			rawPath:  "/testing?test=test",
			prefix:   "/",
			expected: true,
		},
		{
			rawPath:  "/?blah=blah",
			prefix:   "/test",
			expected: false,
		},
		{
			rawPath:  "/?blah=blah",
			prefix:   "",
			expected: true,
		},
		{
			rawPath:  "/?blah=blah",
			prefix:   "/",
			expected: true,
		},
		{
			rawPath:  "/fail?blah=blah",
			prefix:   "",
			expected: false,
		},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			// create server
			srv := createAndStartServer(t, &httpConfig{prefix: tt.prefix}, true, &wsConfig{prefix: tt.prefix})

			resp := testRequest(t, "", "", "", srv, tt.rawPath)
			if tt.expected {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			} else {
				assert.Equal(t, http.StatusNotFound, resp.StatusCode)
			}

			srv.stop()
		})
	}
}

// TestRPCCall_CustomPath tests whether an RPC call on a custom path prefix
// will be successfully completed.
func TestRPCCall_CustomPath(t *testing.T) {
	tests := []struct {
		httpConf  httpConfig
		wsConf    wsConfig
		wsEnabled bool
	}{
		{
			httpConf: httpConfig{
				prefix: "/",
			},
			wsConf: wsConfig{
				prefix: "/test",
			},
			wsEnabled: false,
		},
		{
			httpConf: httpConfig{
				prefix: "/test",
			},
			wsConf: wsConfig{
				prefix: "/test",
			},
			wsEnabled: false,
		},
		{
			httpConf: httpConfig{
				prefix: "test",
			},
			wsConf: wsConfig{
				prefix: "/test",
			},
			wsEnabled: true,
		},
		{
			httpConf: httpConfig{
				prefix: "/testing/test/123",
			},
			wsConf: wsConfig{
				prefix: "/test",
			},
			wsEnabled: true,
		},
		{
			httpConf: httpConfig{
				prefix: "/",
			},
			wsConf: wsConfig{
				prefix: "/test",
			},
			wsEnabled: true,
		},
		{
			httpConf: httpConfig{
				prefix: "",
			},
			wsConf: wsConfig{
				prefix: "",
			},
			wsEnabled: true,
		},
		{
			httpConf: httpConfig{
				prefix: "",
			},
			wsConf: wsConfig{
				prefix: "",
			},
			wsEnabled: false,
		},
		{
			httpConf: httpConfig{
				prefix: "",
			},
			wsConf: wsConfig{
				prefix: "",
			},
			wsEnabled: true,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			srv := createAndStartServer(t, &test.httpConf, test.wsEnabled, &test.wsConf)

			resp := testRequest(t, "", "", "", srv, test.httpConf.prefix)
			assert.True(t, resp.StatusCode != http.StatusNotFound)

			// if / prefix specified, make sure to serve on all paths
			if test.httpConf.prefix == "/" {
				resp = testRequest(t, "", "", "", srv, "/AnyPath=ThisShouldWork?Test")
				assert.True(t, resp.StatusCode != http.StatusNotFound)

				resp = testRequest(t, "", "", "", srv, "/")
				assert.True(t, resp.StatusCode != http.StatusNotFound)
			} else {
				resp = testRequest(t, "", "", "", srv, "/fail")
				assert.True(t, resp.StatusCode == http.StatusNotFound)
			}

			if test.wsEnabled {
				dialer := websocket.DefaultDialer
				// make sure ws dial is successful
				_, _, err := dialer.Dial("ws://"+srv.listenAddr()+test.wsConf.prefix, http.Header{
					"Content-type":          []string{"application/json"},
					"Sec-WebSocket-Version": []string{"13"},
				})
				assert.NoError(t, err)
				// if all paths specified, make sure to serve on all paths
				if test.wsConf.prefix == "/" {
					_, _, err = dialer.Dial("ws://"+srv.listenAddr()+"/AnyPath=ThisShouldWork?Test", http.Header{
						"Content-type":          []string{"application/json"},
						"Sec-WebSocket-Version": []string{"13"},
					})
					assert.NoError(t, err)

					_, _, err = dialer.Dial("ws://"+srv.listenAddr()+"/", http.Header{
						"Content-type":          []string{"application/json"},
						"Sec-WebSocket-Version": []string{"13"},
					})
					assert.NoError(t, err)
				} else {
					// make sure ws dial fails
					_, _, err = dialer.Dial("ws://"+srv.listenAddr()+"/fail", http.Header{
						"Content-type":          []string{"application/json"},
						"Sec-WebSocket-Version": []string{"13"},
					})
					assert.Error(t, err)
				}
			}
		})
	}
}

func createAndStartServer(t *testing.T, conf *httpConfig, ws bool, wsConf *wsConfig) *httpServer {
	t.Helper()

	// set http path prefix
	if len(conf.prefix) >= 1 && strings.Split(conf.prefix, "")[0] != "/" {
		conf.prefix = "/" + conf.prefix
	}
	// set ws prefix
	if len(wsConf.prefix) >= 1 && strings.Split(wsConf.prefix, "")[0] != "/" {
		wsConf.prefix = "/" + wsConf.prefix
	}

	srv := newHTTPServer(testlog.Logger(t, log.LvlDebug), rpc.DefaultHTTPTimeouts)

	assert.NoError(t, srv.enableRPC(nil, *conf))

	if ws {
		assert.NoError(t, srv.enableWS(nil, *wsConf))
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

func testRequest(t *testing.T, key, value, host string, srv *httpServer, path string) *http.Response {
	t.Helper()

	body := bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,method":"rpc_modules"}`))
	req, err := http.NewRequest("POST", "http://"+srv.listenAddr()+path, body)
	if err != nil {
		t.Fatal("could not create http request: ", err)
	}

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
