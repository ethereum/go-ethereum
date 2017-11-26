// Copyright 2015 The go-ethereum Authors
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
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/rs/cors"
	"golang.org/x/net/http2"
)

const (
	contentType                 = "application/json"
	maxHTTPRequestContentLength = 1024 * 128
)

var nullAddr, _ = net.ResolveTCPAddr("tcp", "127.0.0.1:0")

type httpConn struct {
	client *http.Client
	req    *http.Request

	respOnce    sync.Once
	resp        *http.Response
	errChan     chan error
	established chan struct{}

	closeOnce sync.Once
	closed    chan struct{}

	pw io.Writer
}

// httpConn is treated specially by Client.
func (hc *httpConn) LocalAddr() net.Addr              { return nullAddr }
func (hc *httpConn) RemoteAddr() net.Addr             { return nullAddr }
func (hc *httpConn) SetReadDeadline(time.Time) error  { return nil }
func (hc *httpConn) SetWriteDeadline(time.Time) error { return nil }
func (hc *httpConn) SetDeadline(time.Time) error      { return nil }
func (hc *httpConn) Write(b []byte) (int, error) {
	hc.respOnce.Do(func() {
		go func() {
			var err error
			hc.resp, err = hc.client.Do(hc.req)
			hc.errChan <- err
			close(hc.errChan)
			close(hc.established)
		}()
	})
	written, err := hc.pw.Write(b)
	if err != nil {
		return 0, err
	}
	if err := <-hc.errChan; err != nil {
		return 0, err
	}
	return written, nil
}

func (hc *httpConn) Read(b []byte) (int, error) {
	<-hc.established
	return hc.resp.Body.Read(b)
}

func (hc *httpConn) Close() error {
	<-hc.established
	hc.resp.Body.Close()
	hc.closeOnce.Do(func() { close(hc.closed) })
	return nil
}

// DialHTTP creates a new RPC clients that connection to an RPC server over HTTP.
func DialHTTP(endpoint string) (*Client, error) {
	pr, pw := io.Pipe()
	req, err := http.NewRequest("POST", endpoint, pr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", contentType)

	initctx := context.Background()
	client := http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}
	return newClient(initctx, func(context.Context) (net.Conn, error) {
		return &httpConn{pw: pw,
			client:      &client,
			errChan:     make(chan error, 1),
			established: make(chan struct{}),
			req:         req,
			closed:      make(chan struct{})}, nil
	})
}

type flushWriter struct {
	w io.Writer
}

func (fw flushWriter) Write(p []byte) (n int, err error) {
	n, err = fw.w.Write(p)
	if f, ok := fw.w.(http.Flusher); ok {
		f.Flush()
	}
	return
}

// httpReadWriteNopCloser wraps a io.Reader and io.Writer with a NOP Close method.
type httpReadWriteNopCloser struct {
	io.Reader
	io.Writer
}

// Close does nothing and returns always nil
func (t *httpReadWriteNopCloser) Close() error {
	return nil
}

// NewHTTPServer creates a new HTTP RPC server around an API provider.
//
// Deprecated: Server implements http.Handler
func NewHTTPServer(l net.Listener, cors []string, srv *Server) error {
	h2 := &http2.Server{}
	handler := newCorsHandler(srv, cors)
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		go h2.ServeConn(conn, &http2.ServeConnOpts{Handler: handler})
	}

}

// ServeHTTP serves JSON-RPC requests over HTTP.
func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Permit dumb empty requests for remote health-checks (AWS)
	if r.Method == "GET" && r.ContentLength == 0 && r.URL.RawQuery == "" {
		return
	}
	if code, err := validateRequest(r); err != nil {
		http.Error(w, err.Error(), code)
		return
	}
	// All checks passed, create a codec that reads direct from the request body
	// untilEOF and writes the response to w and order the server to process a
	// single request.
	codec := NewJSONCodec(&httpReadWriteNopCloser{r.Body, flushWriter{w}})
	w.Header().Set("content-type", contentType)
	srv.ServeCodec(codec, OptionMethodInvocation|OptionSubscriptions)
}

// validateRequest returns a non-zero response code and error message if the
// request is invalid.
func validateRequest(r *http.Request) (int, error) {
	if r.Method == "PUT" || r.Method == "DELETE" {
		return http.StatusMethodNotAllowed, errors.New("method not allowed")
	}
	if r.ContentLength > maxHTTPRequestContentLength {
		err := fmt.Errorf("content length too large (%d>%d)", r.ContentLength, maxHTTPRequestContentLength)
		return http.StatusRequestEntityTooLarge, err
	}
	mt, _, err := mime.ParseMediaType(r.Header.Get("content-type"))
	if err != nil || mt != contentType {
		err := fmt.Errorf("invalid content type, only %s is supported", contentType)
		return http.StatusUnsupportedMediaType, err
	}
	return 0, nil
}

func newCorsHandler(srv *Server, allowedOrigins []string) http.Handler {
	// disable CORS support if user has not specified a custom CORS configuration
	if len(allowedOrigins) == 0 {
		return srv
	}

	c := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"POST", "GET"},
		MaxAge:         600,
		AllowedHeaders: []string{"*"},
	})
	return c.Handler(srv)
}
