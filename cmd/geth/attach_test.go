package main

import (
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
)

type testHandler struct {
	body func(http.ResponseWriter, *http.Request)
}

func (t *testHandler) ServeHTTP(out http.ResponseWriter, in *http.Request) {
	t.body(out, in)
}

// TestAttachWithHeaders tests that 'geth attach' with custom headers works, i.e
// that custom headers are forwarded to the target.
func TestAttachWithHeaders(t *testing.T) {
	t.Parallel()
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	testReceiveHeaders(t, ln, "attach", "-H", "first: one", "-H", "second: two", fmt.Sprintf("http://localhost:%d", port))
	// This way to do it fails due to flag ordering:
	//
	// testReceiveHeaders(t, ln, "-H", "first: one", "-H", "second: two", "attach", fmt.Sprintf("http://localhost:%d", port))
	// This is fixed in a follow-up PR.
}

// TestAttachWithHeaders tests that 'geth db --remotedb' with custom headers works, i.e
// that custom headers are forwarded to the target.
func TestRemoteDbWithHeaders(t *testing.T) {
	t.Parallel()
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	testReceiveHeaders(t, ln, "db", "metadata", "--remotedb", fmt.Sprintf("http://localhost:%d", port), "-H", "first: one", "-H", "second: two")
}

func testReceiveHeaders(t *testing.T, ln net.Listener, gethArgs ...string) {
	var ok uint32
	server := &http.Server{
		Addr: "localhost:0",
		Handler: &testHandler{func(w http.ResponseWriter, r *http.Request) {
			// We expect two headers
			if have, want := r.Header.Get("first"), "one"; have != want {
				t.Fatalf("missing header, have %v want %v", have, want)
			}
			if have, want := r.Header.Get("second"), "two"; have != want {
				t.Fatalf("missing header, have %v want %v", have, want)
			}
			atomic.StoreUint32(&ok, 1)
		}}}
	go server.Serve(ln)
	defer server.Close()
	runGeth(t, gethArgs...).WaitExit()
	if atomic.LoadUint32(&ok) != 1 {
		t.Fatal("Test fail, expected invocation to succeed")
	}
}
