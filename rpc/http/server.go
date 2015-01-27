/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
package rpchttp

import (
	"fmt"
	"net"
	"net/http"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/xeth"
)

var rpchttplogger = logger.NewLogger("RPC-HTTP")
var JSON rpc.JsonWrapper

func NewRpcHttpServer(pipe *xeth.JSXEth, port int) (*RpcHttpServer, error) {
	sport := fmt.Sprintf(":%d", port)
	l, err := net.Listen("tcp", sport)
	if err != nil {
		return nil, err
	}

	return &RpcHttpServer{
		listener: l,
		quit:     make(chan bool),
		pipe:     pipe,
		port:     port,
	}, nil
}

type RpcHttpServer struct {
	quit     chan bool
	listener net.Listener
	pipe     *xeth.JSXEth
	port     int
}

func (s *RpcHttpServer) exitHandler() {
out:
	for {
		select {
		case <-s.quit:
			s.listener.Close()
			break out
		}
	}

	rpchttplogger.Infoln("Shutdown RPC-HTTP server")
}

func (s *RpcHttpServer) Stop() {
	close(s.quit)
}

func (s *RpcHttpServer) Start() {
	rpchttplogger.Infof("Starting RPC-HTTP server on port %d", s.port)
	go s.exitHandler()

	api := rpc.NewEthereumApi(s.pipe)
	h := s.apiHandler(api)
	http.Handle("/", h)

	err := http.Serve(s.listener, nil)
	// FIX Complains on shutdown due to listner already being closed
	if err != nil {
		rpchttplogger.Errorln("Error on RPC-HTTP interface:", err)
	}
}

func (s *RpcHttpServer) apiHandler(api *rpc.EthereumApi) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		rpchttplogger.Debugln("Handling request")

		reqParsed, reqerr := JSON.ParseRequestBody(req)
		if reqerr != nil {
			JSON.Send(w, &rpc.RpcErrorResponse{JsonRpc: reqParsed.JsonRpc, ID: reqParsed.ID, Error: true, ErrorText: rpc.ErrorParseRequest})
			return
		}

		var response interface{}
		reserr := api.GetRequestReply(&reqParsed, &response)
		if reserr != nil {
			rpchttplogger.Errorln(reserr)
			JSON.Send(w, &rpc.RpcErrorResponse{JsonRpc: reqParsed.JsonRpc, ID: reqParsed.ID, Error: true, ErrorText: reserr.Error()})
			return
		}

		rpchttplogger.Debugf("Generated response: %T %s", response, response)
		JSON.Send(w, &rpc.RpcSuccessResponse{JsonRpc: reqParsed.JsonRpc, ID: reqParsed.ID, Error: false, Result: response})
	}

	return http.HandlerFunc(fn)
}
