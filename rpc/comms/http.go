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

package comms

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"bytes"
	"io"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/rs/cors"
)

const (
	serverIdleTimeout  = 10 * time.Second // idle keep-alive connections
	serverReadTimeout  = 15 * time.Second // per-request read timeout
	serverWriteTimeout = 15 * time.Second // per-request read timeout
)

var (
	httpServerMu sync.Mutex
	httpServer   *stopServer
)

type HttpConfig struct {
	ListenAddress string
	ListenPort    uint
	CorsDomain    string
}

// stopServer augments http.Server with idle connection tracking.
// Idle keep-alive connections are shut down when Close is called.
type stopServer struct {
	*http.Server
	l net.Listener
	// connection tracking state
	mu       sync.Mutex
	shutdown bool // true when Stop has returned
	idle     map[net.Conn]struct{}
}

type handler struct {
	codec codec.Codec
	api   shared.EthereumApi
}

// StartHTTP starts listening for RPC requests sent via HTTP.
func StartHttp(cfg HttpConfig, codec codec.Codec, api shared.EthereumApi) error {
	httpServerMu.Lock()
	defer httpServerMu.Unlock()

	addr := fmt.Sprintf("%s:%d", cfg.ListenAddress, cfg.ListenPort)
	if httpServer != nil {
		if addr != httpServer.Addr {
			return fmt.Errorf("RPC service already running on %s ", httpServer.Addr)
		}
		return nil // RPC service already running on given host/port
	}
	// Set up the request handler, wrapping it with CORS headers if configured.
	handler := http.Handler(&handler{codec, api})
	if len(cfg.CorsDomain) > 0 {
		opts := cors.Options{
			AllowedMethods: []string{"POST"},
			AllowedOrigins: strings.Split(cfg.CorsDomain, " "),
		}
		handler = cors.New(opts).Handler(handler)
	}
	// Start the server.
	s, err := listenHTTP(addr, handler)
	if err != nil {
		glog.V(logger.Error).Infof("Can't listen on %s:%d: %v", cfg.ListenAddress, cfg.ListenPort, err)
		return err
	}
	httpServer = s
	return nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Limit request size to resist DoS
	if req.ContentLength > maxHttpSizeReqLength {
		err := fmt.Errorf("Request too large")
		response := shared.NewRpcErrorResponse(-1, shared.JsonRpcVersion, -32700, err)
		sendJSON(w, &response)
		return
	}

	defer req.Body.Close()
	payload, err := ioutil.ReadAll(req.Body)
	if err != nil {
		err := fmt.Errorf("Could not read request body")
		response := shared.NewRpcErrorResponse(-1, shared.JsonRpcVersion, -32700, err)
		sendJSON(w, &response)
		return
	}

	c := h.codec.New(nil)
	var rpcReq shared.Request
	if err = c.Decode(payload, &rpcReq); err == nil {
		reply, err := h.api.Execute(&rpcReq)
		res := shared.NewRpcResponse(rpcReq.Id, rpcReq.Jsonrpc, reply, err)
		sendJSON(w, &res)
		return
	}

	var reqBatch []shared.Request
	if err = c.Decode(payload, &reqBatch); err == nil {
		resBatch := make([]*interface{}, len(reqBatch))
		resCount := 0
		for i, rpcReq := range reqBatch {
			reply, err := h.api.Execute(&rpcReq)
			if rpcReq.Id != nil { // this leaves nil entries in the response batch for later removal
				resBatch[i] = shared.NewRpcResponse(rpcReq.Id, rpcReq.Jsonrpc, reply, err)
				resCount += 1
			}
		}
		// make response omitting nil entries
		sendJSON(w, resBatch[:resCount])
		return
	}

	// invalid request
	err = fmt.Errorf("Could not decode request")
	res := shared.NewRpcErrorResponse(-1, shared.JsonRpcVersion, -32600, err)
	sendJSON(w, res)
}

func sendJSON(w io.Writer, v interface{}) {
	if glog.V(logger.Detail) {
		if payload, err := json.MarshalIndent(v, "", "\t"); err == nil {
			glog.Infof("Sending payload: %s", payload)
		}
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		glog.V(logger.Error).Infoln("Error sending JSON:", err)
	}
}

// Stop closes all active HTTP connections and shuts down the server.
func StopHttp() {
	httpServerMu.Lock()
	defer httpServerMu.Unlock()
	if httpServer != nil {
		httpServer.Close()
		httpServer = nil
	}
}

func listenHTTP(addr string, h http.Handler) (*stopServer, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	s := &stopServer{l: l, idle: make(map[net.Conn]struct{})}
	s.Server = &http.Server{
		Addr:         addr,
		Handler:      h,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
		ConnState:    s.connState,
	}
	go s.Serve(l)
	return s, nil
}

func (s *stopServer) connState(c net.Conn, state http.ConnState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Close c immediately if we're past shutdown.
	if s.shutdown {
		if state != http.StateClosed {
			c.Close()
		}
		return
	}
	if state == http.StateIdle {
		s.idle[c] = struct{}{}
	} else {
		delete(s.idle, c)
	}
}

func (s *stopServer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Shut down the acceptor. No new connections can be created.
	s.l.Close()
	// Drop all idle connections. Non-idle connections will be
	// closed by connState as soon as they become idle.
	s.shutdown = true
	for c := range s.idle {
		glog.V(logger.Detail).Infof("closing idle connection %v", c.RemoteAddr())
		c.Close()
		delete(s.idle, c)
	}
}

type httpClient struct {
	address string
	port    uint
	codec   codec.ApiCoder
	lastRes interface{}
	lastErr error
}

// Create a new in process client
func NewHttpClient(cfg HttpConfig, c codec.Codec) *httpClient {
	return &httpClient{
		address: cfg.ListenAddress,
		port:    cfg.ListenPort,
		codec:   c.New(nil),
	}
}

func (self *httpClient) Close() {
	// do nothing
}

func (self *httpClient) Send(req interface{}) error {
	var body []byte
	var err error

	self.lastRes = nil
	self.lastErr = nil

	if body, err = self.codec.Encode(req); err != nil {
		return err
	}

	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s:%d", self.address, self.port), bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.Status == "200 OK" {
		reply, _ := ioutil.ReadAll(resp.Body)
		var rpcSuccessResponse shared.SuccessResponse
		if err = self.codec.Decode(reply, &rpcSuccessResponse); err == nil {
			self.lastRes = &rpcSuccessResponse
			self.lastErr = err
			return nil
		} else {
			var rpcErrorResponse shared.ErrorResponse
			if err = self.codec.Decode(reply, &rpcErrorResponse); err == nil {
				self.lastRes = &rpcErrorResponse
				self.lastErr = err
				return nil
			} else {
				return err
			}
		}
	}

	return fmt.Errorf("Not implemented")
}

func (self *httpClient) Recv() (interface{}, error) {
	return self.lastRes, self.lastErr
}

func (self *httpClient) SupportedModules() (map[string]string, error) {
	var body []byte
	var err error

	payload := shared.Request{
		Id:      1,
		Jsonrpc: "2.0",
		Method:  "modules",
	}

	if body, err = self.codec.Encode(payload); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s:%d", self.address, self.port), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.Status == "200 OK" {
		reply, _ := ioutil.ReadAll(resp.Body)
		var rpcRes shared.SuccessResponse
		if err = self.codec.Decode(reply, &rpcRes); err != nil {
			return nil, err
		}

		result := make(map[string]string)
		if modules, ok := rpcRes.Result.(map[string]interface{}); ok {
			for a, v := range modules {
				result[a] = fmt.Sprintf("%s", v)
			}
			return result, nil
		}
		err = fmt.Errorf("Unable to parse module response - %v", rpcRes.Result)
	} else {
		fmt.Printf("resp.Status = %s\n", resp.Status)
		fmt.Printf("err = %v\n", err)
	}

	return nil, err
}
