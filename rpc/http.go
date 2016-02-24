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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"io"

	"github.com/rs/cors"
)

const (
	maxHTTPRequestContentLength = 1024 * 128
)

// httpClient connects to a geth RPC server over HTTP.
type httpClient struct {
	endpoint *url.URL // HTTP-RPC server endpoint
	lastRes  []byte   // HTTP requests are synchronous, store last response
}

// NewHTTPClient create a new RPC clients that connection to a geth RPC server
// over HTTP.
func NewHTTPClient(endpoint string) (Client, error) {
	url, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	return &httpClient{endpoint: url}, nil
}

// Send will serialize the given msg to JSON and sends it to the RPC server.
// Since HTTP is synchronous the response is stored until Recv is called.
func (client *httpClient) Send(msg interface{}) error {
	var body []byte
	var err error

	client.lastRes = nil

	if body, err = json.Marshal(msg); err != nil {
		return err
	}

	httpReq, err := http.NewRequest("POST", client.endpoint.String(), bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpClient := http.Client{}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		client.lastRes, err = ioutil.ReadAll(resp.Body)
		return err
	}

	return fmt.Errorf("unable to handle request")
}

// Recv will try to deserialize the last received response into the given msg.
func (client *httpClient) Recv(msg interface{}) error {
	return json.Unmarshal(client.lastRes, &msg)
}

// Close is not necessary for httpClient
func (client *httpClient) Close() {
}

// SupportedModules will return the collection of offered RPC modules.
func (client *httpClient) SupportedModules() (map[string]string, error) {
	return SupportedModules(client)
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

// newJSONHTTPHandler creates a HTTP handler that will parse incoming JSON requests,
// send the request to the given API provider and sends the response back to the caller.
func newJSONHTTPHandler(srv *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > maxHTTPRequestContentLength {
			http.Error(w,
				fmt.Sprintf("content length too large (%d>%d)", r.ContentLength, maxHTTPRequestContentLength),
				http.StatusRequestEntityTooLarge)
			return
		}

		w.Header().Set("content-type", "application/json")

		// create a codec that reads direct from the request body until
		// EOF and writes the response to w and order the server to process
		// a single request.
		codec := NewJSONCodec(&httpReadWriteNopCloser{r.Body, w})
		defer codec.Close()
		srv.ServeSingleRequest(codec)
	}
}

// NewHTTPServer creates a new HTTP RPC server around an API provider.
func NewHTTPServer(corsString string, srv *Server) *http.Server {
	var allowedOrigins []string
	for _, domain := range strings.Split(corsString, ",") {
		allowedOrigins = append(allowedOrigins, strings.TrimSpace(domain))
	}

	c := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"POST", "GET"},
	})

	handler := c.Handler(newJSONHTTPHandler(srv))

	return &http.Server{
		Handler: handler,
	}
}
