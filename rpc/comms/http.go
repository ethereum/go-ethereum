// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package comms

import (
	"fmt"
	"net/http"
	"strings"

	"bytes"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/rs/cors"
)

var (
	// main HTTP rpc listener
	httpListener         *stoppableTCPListener
	listenerStoppedError = fmt.Errorf("Listener has stopped")
)

type HttpConfig struct {
	ListenAddress string
	ListenPort    uint
	CorsDomain    string
}

func StartHttp(cfg HttpConfig, codec codec.Codec, api shared.EthereumApi) error {
	if httpListener != nil {
		if fmt.Sprintf("%s:%d", cfg.ListenAddress, cfg.ListenPort) != httpListener.Addr().String() {
			return fmt.Errorf("RPC service already running on %s ", httpListener.Addr().String())
		}
		return nil // RPC service already running on given host/port
	}

	l, err := newStoppableTCPListener(fmt.Sprintf("%s:%d", cfg.ListenAddress, cfg.ListenPort))
	if err != nil {
		glog.V(logger.Error).Infof("Can't listen on %s:%d: %v", cfg.ListenAddress, cfg.ListenPort, err)
		return err
	}
	httpListener = l

	var handler http.Handler
	if len(cfg.CorsDomain) > 0 {
		var opts cors.Options
		opts.AllowedMethods = []string{"POST"}
		opts.AllowedOrigins = strings.Split(cfg.CorsDomain, " ")

		c := cors.New(opts)
		handler = newStoppableHandler(c.Handler(gethHttpHandler(codec, api)), l.stop)
	} else {
		handler = newStoppableHandler(gethHttpHandler(codec, api), l.stop)
	}

	go http.Serve(l, handler)

	return nil
}

func StopHttp() {
	if httpListener != nil {
		httpListener.Stop()
		httpListener = nil
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
			self.lastRes = rpcSuccessResponse.Result
			self.lastErr = err
			return nil
		} else {
			var rpcErrorResponse shared.ErrorResponse
			if err = self.codec.Decode(reply, &rpcErrorResponse); err == nil {
				self.lastRes = rpcErrorResponse.Error
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
