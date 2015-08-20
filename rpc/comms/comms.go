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
	"io"
	"net"

	"fmt"
	"strings"

	"strconv"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
)

const (
	maxHttpSizeReqLength = 1024 * 1024 // 1MB
)

var (
	// List with all API's which are offered over the in proc interface by default
	DefaultInProcApis = shared.AllApis

	// List with all API's which are offered over the IPC interface by default
	DefaultIpcApis = shared.AllApis

	// List with API's which are offered over thr HTTP/RPC interface by default
	DefaultHttpRpcApis = strings.Join([]string{
		shared.DbApiName, shared.EthApiName, shared.NetApiName, shared.Web3ApiName,
	}, ",")
)

type EthereumClient interface {
	// Close underlying connection
	Close()
	// Send request
	Send(interface{}) error
	// Receive response
	Recv() (interface{}, error)
	// List with modules this client supports
	SupportedModules() (map[string]string, error)
}

func handle(id int, conn net.Conn, api shared.EthereumApi, c codec.Codec) {
	codec := c.New(conn)

	for {
		requests, isBatch, err := codec.ReadRequest()
		if err == io.EOF {
			codec.Close()
			return
		} else if err != nil {
			codec.Close()
			glog.V(logger.Debug).Infof("Closed IPC Conn %06d recv err - %v\n", id, err)
			return
		}

		if isBatch {
			responses := make([]*interface{}, len(requests))
			responseCount := 0
			for _, req := range requests {
				res, err := api.Execute(req)
				if req.Id != nil {
					rpcResponse := shared.NewRpcResponse(req.Id, req.Jsonrpc, res, err)
					responses[responseCount] = rpcResponse
					responseCount += 1
				}
			}

			err = codec.WriteResponse(responses[:responseCount])
			if err != nil {
				codec.Close()
				glog.V(logger.Debug).Infof("Closed IPC Conn %06d send err - %v\n", id, err)
				return
			}
		} else {
			var rpcResponse interface{}
			res, err := api.Execute(requests[0])

			rpcResponse = shared.NewRpcResponse(requests[0].Id, requests[0].Jsonrpc, res, err)
			err = codec.WriteResponse(rpcResponse)
			if err != nil {
				codec.Close()
				glog.V(logger.Debug).Infof("Closed IPC Conn %06d send err - %v\n", id, err)
				return
			}
		}
	}
}

// Endpoint must be in the form of:
// ${protocol}:${path}
// e.g. ipc:/tmp/geth.ipc
//      rpc:localhost:8545
func ClientFromEndpoint(endpoint string, c codec.Codec) (EthereumClient, error) {
	if strings.HasPrefix(endpoint, "ipc:") {
		cfg := IpcConfig{
			Endpoint: endpoint[4:],
		}
		return NewIpcClient(cfg, codec.JSON)
	}

	if strings.HasPrefix(endpoint, "rpc:") {
		parts := strings.Split(endpoint, ":")
		addr := "http://localhost"
		port := uint(8545)
		if len(parts) >= 3 {
			addr = parts[1] + ":" + parts[2]
		}

		if len(parts) >= 4 {
			p, err := strconv.Atoi(parts[3])

			if err != nil {
				return nil, err
			}
			port = uint(p)
		}

		cfg := HttpConfig{
			ListenAddress: addr,
			ListenPort:    port,
		}

		return NewHttpClient(cfg, codec.JSON), nil
	}

	return nil, fmt.Errorf("Invalid endpoint")
}
