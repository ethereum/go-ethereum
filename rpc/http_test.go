// Copyright 2017 The go-ethereum Authors
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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func confirmStatusCode(t *testing.T, got, want int) {
	t.Helper()
	if got == want {
		return
	}
	if gotName := http.StatusText(got); len(gotName) > 0 {
		if wantName := http.StatusText(want); len(wantName) > 0 {
			t.Fatalf("response status code: got %d (%s), want %d (%s)", got, gotName, want, wantName)
		}
	}
	t.Fatalf("response status code: got %d, want %d", got, want)
}

func confirmRequestValidationCode(t *testing.T, method, contentType, body string, expectedStatusCode int) {
	t.Helper()

	s := NewServer()
	request := httptest.NewRequest(method, "http://url.com", strings.NewReader(body))
	if len(contentType) > 0 {
		request.Header.Set("Content-Type", contentType)
	}
	code, err := s.validateRequest(request)
	if code == 0 {
		if err != nil {
			t.Errorf("validation: got error %v, expected nil", err)
		}
	} else if err == nil {
		t.Errorf("validation: code %d: got nil, expected error", code)
	}
	confirmStatusCode(t, code, expectedStatusCode)
}

func TestHTTPErrorResponseWithDelete(t *testing.T) {
	t.Parallel()

	confirmRequestValidationCode(t, http.MethodDelete, contentType, "", http.StatusMethodNotAllowed)
}

func TestHTTPErrorResponseWithPut(t *testing.T) {
	t.Parallel()

	confirmRequestValidationCode(t, http.MethodPut, contentType, "", http.StatusMethodNotAllowed)
}

func TestHTTPErrorResponseWithMaxContentLength(t *testing.T) {
	t.Parallel()

	body := make([]rune, defaultBodyLimit+1)
	confirmRequestValidationCode(t,
		http.MethodPost, contentType, string(body), http.StatusRequestEntityTooLarge)
}

func TestHTTPErrorResponseWithEmptyContentType(t *testing.T) {
	t.Parallel()

	confirmRequestValidationCode(t, http.MethodPost, "", "", http.StatusUnsupportedMediaType)
}

func TestHTTPErrorResponseWithValidRequest(t *testing.T) {
	t.Parallel()

	confirmRequestValidationCode(t, http.MethodPost, contentType, "", 0)
}

func confirmHTTPRequestYieldsStatusCode(t *testing.T, method, contentType, body string, expectedStatusCode int) {
	t.Helper()
	s := Server{}
	ts := httptest.NewServer(&s)
	defer ts.Close()

	request, err := http.NewRequest(method, ts.URL, strings.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create a valid HTTP request: %v", err)
	}
	if len(contentType) > 0 {
		request.Header.Set("Content-Type", contentType)
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	cleanlyCloseBody(resp.Body)
	confirmStatusCode(t, resp.StatusCode, expectedStatusCode)
}

func TestHTTPResponseWithEmptyGet(t *testing.T) {
	t.Parallel()

	confirmHTTPRequestYieldsStatusCode(t, http.MethodGet, "", "", http.StatusOK)
}

// This checks that maxRequestContentLength is not applied to the response of a request.
func TestHTTPRespBodyUnlimited(t *testing.T) {
	t.Parallel()

	const respLength = defaultBodyLimit * 3

	s := NewServer()
	defer s.Stop()
	s.RegisterName("test", largeRespService{respLength})
	ts := httptest.NewServer(s)
	defer ts.Close()

	c, err := DialHTTP(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	var r string
	if err := c.Call(&r, "test_largeResp"); err != nil {
		t.Fatal(err)
	}
	if len(r) != respLength {
		t.Fatalf("response has wrong length %d, want %d", len(r), respLength)
	}
}

// Tests that an HTTP error results in an HTTPError instance
// being returned with the expected attributes.
func TestHTTPErrorResponse(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "error has occurred!", http.StatusTeapot)
	}))
	defer ts.Close()

	c, err := DialHTTP(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	var r string
	err = c.Call(&r, "test_method")
	if err == nil {
		t.Fatal("error was expected")
	}

	httpErr, ok := err.(HTTPError)
	if !ok {
		t.Fatalf("unexpected error type %T", err)
	}

	if httpErr.StatusCode != http.StatusTeapot {
		t.Error("unexpected status code", httpErr.StatusCode)
	}
	if httpErr.Status != "418 I'm a teapot" {
		t.Error("unexpected status text", httpErr.Status)
	}
	if body := string(httpErr.Body); body != "error has occurred!\n" {
		t.Error("unexpected body", body)
	}

	if errMsg := httpErr.Error(); errMsg != "418 I'm a teapot: error has occurred!\n" {
		t.Error("unexpected error message", errMsg)
	}
}

func TestHTTPPeerInfo(t *testing.T) {
	t.Parallel()

	s := newTestServer()
	defer s.Stop()
	ts := httptest.NewServer(s)
	defer ts.Close()

	c, err := Dial(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	c.SetHeader("user-agent", "ua-testing")
	c.SetHeader("origin", "origin.example.com")

	// Request peer information.
	var info PeerInfo
	if err := c.Call(&info, "test_peerInfo"); err != nil {
		t.Fatal(err)
	}

	if info.RemoteAddr == "" {
		t.Error("RemoteAddr not set")
	}
	if info.Transport != "http" {
		t.Errorf("wrong Transport %q", info.Transport)
	}
	if info.HTTP.Version != "HTTP/1.1" {
		t.Errorf("wrong HTTP.Version %q", info.HTTP.Version)
	}
	if info.HTTP.UserAgent != "ua-testing" {
		t.Errorf("wrong HTTP.UserAgent %q", info.HTTP.UserAgent)
	}
	if info.HTTP.Origin != "origin.example.com" {
		t.Errorf("wrong HTTP.Origin %q", info.HTTP.UserAgent)
	}
}

func TestNewContextWithHeaders(t *testing.T) {
	t.Parallel()

	expectedHeaders := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		for i := 0; i < expectedHeaders; i++ {
			key, want := fmt.Sprintf("key-%d", i), fmt.Sprintf("val-%d", i)
			if have := request.Header.Get(key); have != want {
				t.Errorf("wrong request headers for %s, want: %s, have: %s", key, want, have)
			}
		}
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, err := Dial(server.URL)
	if err != nil {
		t.Fatalf("failed to dial: %s", err)
	}
	defer client.Close()

	newHdr := func(k, v string) http.Header {
		header := http.Header{}
		header.Set(k, v)
		return header
	}
	ctx1 := NewContextWithHeaders(context.Background(), newHdr("key-0", "val-0"))
	ctx2 := NewContextWithHeaders(ctx1, newHdr("key-1", "val-1"))
	ctx3 := NewContextWithHeaders(ctx2, newHdr("key-2", "val-2"))

	expectedHeaders = 3
	if err := client.CallContext(ctx3, nil, "test"); err != ErrNoResult {
		t.Error("call failed", err)
	}

	expectedHeaders = 2
	if err := client.CallContext(ctx2, nil, "test"); err != ErrNoResult {
		t.Error("call failed:", err)
	}
}

// TestHTTPRequestFraming covers what the server answers for the shapes of body
// that reach it, including the ones that are not valid JSON.
func TestHTTPRequestFraming(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string // substring the response must contain, empty means no response
	}{
		{
			name: "call",
			body: `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["x",3]}`,
			want: `"result"`,
		},
		{
			name: "call with surrounding space",
			body: "  \n{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"test_echo\",\"params\":[\"x\",3]}\n ",
			want: `"result"`,
		},
		{
			name: "batch",
			body: `[{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["x",3]}]`,
			want: `"result"`,
		},
		{
			name: "empty body",
			body: ``,
			want: ``,
		},
		{
			name: "whitespace only",
			body: "   \n\t ",
			want: ``,
		},
		{
			name: "truncated object",
			body: `{"jsonrpc":"2.0","id":1,"method":"test_echo"`,
			want: `parse error`,
		},
		{
			name: "not json",
			body: `hello`,
			want: `parse error`,
		},
		{
			name: "unbalanced bracket",
			body: `[{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["x",3]}`,
			want: `parse error`,
		},
		{
			name: "control character in string",
			body: "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"test_\x01echo\",\"params\":[]}",
			want: `parse error`,
		},
		{
			// A body holding more than one value is rejected. The decoder this
			// replaced stopped after the first value and ignored the rest.
			name: "trailing second value",
			body: `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["x",3]}{"a":1}`,
			want: `parse error`,
		},
		{
			name: "trailing garbage",
			body: `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["x",3]} oops`,
			want: `parse error`,
		},
	}

	srv := newTestServer()
	defer srv.Stop()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			req.Header.Set("content-type", "application/json")
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)

			body := rec.Body.String()
			if tc.want == "" {
				if strings.TrimSpace(body) != "" {
					t.Fatalf("want no response, got %q", body)
				}
				return
			}
			if !strings.Contains(body, tc.want) {
				t.Fatalf("want response containing %q, got %q", tc.want, body)
			}
		})
	}
}

// TestHTTPRequestFramingChunked checks a body with no content length, which is
// the case the size hint cannot help with.
func TestHTTPRequestFramingChunked(t *testing.T) {
	srv := newTestServer()
	defer srv.Stop()

	body := `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["x",3]}`
	req := httptest.NewRequest(http.MethodPost, "/", io.NopCloser(strings.NewReader(body)))
	req.Header.Set("content-type", "application/json")
	req.ContentLength = -1
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if got := rec.Body.String(); !strings.Contains(got, `"result"`) {
		t.Fatalf("want a result, got %q", got)
	}
}

// TestReadAllBody checks the body reader against io.ReadAll for both a helpful
// and an unhelpful size hint.
func TestReadAllBody(t *testing.T) {
	for _, size := range []int{0, 1, 511, 512, 513, 4096, 100000} {
		want := bytes.Repeat([]byte("ab"), size/2)
		for _, hint := range []int{0, 1, size, size + 1, size * 2} {
			got, err := readAllBody(bytes.NewReader(want), hint)
			if err != nil {
				t.Fatalf("size %d hint %d: %v", size, hint, err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("size %d hint %d: got %d bytes, want %d", size, hint, len(got), len(want))
			}
		}
	}
}

// TestReadAllBodyError checks that a read failure is reported rather than
// treated as the end of the body.
func TestReadAllBodyError(t *testing.T) {
	r := io.MultiReader(strings.NewReader(`{"a":`), &errReader{})
	if _, err := readAllBody(r, 0); err == nil {
		t.Fatal("want an error")
	}
}

type errReader struct{}

func (*errReader) Read([]byte) (int, error) { return 0, io.ErrClosedPipe }

// TestHTTPBatchRequestFraming checks a batch whose items each carry a sizeable
// argument. Every message in a batch points into the same buffer, so this would
// catch one item's arguments bleeding into another's.
func TestHTTPBatchRequestFraming(t *testing.T) {
	srv := newTestServer()
	defer srv.Stop()

	const items = 12
	var body strings.Builder
	body.WriteByte('[')
	for i := 0; i < items; i++ {
		if i > 0 {
			body.WriteByte(',')
		}
		// A distinct payload per item, large enough to span several reads.
		pad := strings.Repeat(string(rune('a'+i)), 4096)
		fmt.Fprintf(&body, `{"jsonrpc":"2.0","id":%d,"method":"test_echo","params":["%s",%d,{"S":"%s"}]}`,
			i, pad, i, pad)
	}
	body.WriteByte(']')

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body.String()))
	req.Header.Set("content-type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	confirmStatusCode(t, rec.Code, http.StatusOK)

	var resps []struct {
		ID     int `json:"id"`
		Result struct {
			String string
			Int    int
			Args   *echoArgs
		} `json:"result"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resps); err != nil {
		t.Fatalf("decoding the batch response failed: %v", err)
	}
	if len(resps) != items {
		t.Fatalf("got %d responses, want %d", len(resps), items)
	}
	for _, r := range resps {
		want := strings.Repeat(string(rune('a'+r.ID)), 4096)
		if r.Result.Int != r.ID {
			t.Errorf("id %d: Int = %d", r.ID, r.Result.Int)
		}
		if r.Result.String != want {
			t.Errorf("id %d: String is not its own argument", r.ID)
		}
		if r.Result.Args == nil || r.Result.Args.S != want {
			t.Errorf("id %d: Args is not its own argument", r.ID)
		}
	}
}
