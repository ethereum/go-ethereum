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
package rpc

import (
	"fmt"
	"net"
	"net/http"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/xeth"
)

var jsonlogger = logger.NewLogger("JSON")

type JsonRpcServer struct {
	quit     chan bool
	listener net.Listener
	pipe     *xeth.JSXEth
}

func (s *JsonRpcServer) exitHandler() {
out:
	for {
		select {
		case <-s.quit:
			s.listener.Close()
			break out
		}
	}

	jsonlogger.Infoln("Shutdown JSON-RPC server")
}

func (s *JsonRpcServer) Stop() {
	close(s.quit)
}

func (s *JsonRpcServer) Start() {
	jsonlogger.Infoln("Starting JSON-RPC server")
	go s.exitHandler()

	h := apiHandler(&EthereumApi{pipe: s.pipe})
	http.Handle("/", h)

	err := http.Serve(s.listener, nil)
	// FIX Complains on shutdown due to listner already being closed
	if err != nil {
		jsonlogger.Errorln("Error on JSON-RPC interface:", err)
	}
}

func NewJsonRpcServer(pipe *xeth.JSXEth, port int) (*JsonRpcServer, error) {
	sport := fmt.Sprintf(":%d", port)
	l, err := net.Listen("tcp", sport)
	if err != nil {
		return nil, err
	}

	return &JsonRpcServer{
		listener: l,
		quit:     make(chan bool),
		pipe:     pipe,
	}, nil
}

func apiHandler(xeth *EthereumApi) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		jsonlogger.Debugln("Handling request")

		reqParsed, reqerr := JSON.ParseRequestBody(req)
		if reqerr != nil {
			JSON.Send(w, &RpcErrorResponse{JsonRpc: reqParsed.JsonRpc, ID: reqParsed.ID, Error: true, ErrorText: ErrorParseRequest})
			return
		}

		var response interface{}
		reserr := xeth.GetRequestReply(&reqParsed, &response)
		if reserr != nil {
			jsonlogger.Errorln(reserr)
			JSON.Send(w, &RpcErrorResponse{JsonRpc: reqParsed.JsonRpc, ID: reqParsed.ID, Error: true, ErrorText: reserr.Error()})
			return
		}

		jsonlogger.Debugf("Generated response: %T %s", response, response)
		JSON.Send(w, &RpcSuccessResponse{JsonRpc: reqParsed.JsonRpc, ID: reqParsed.ID, Error: false, Result: response})
	}

	return http.HandlerFunc(fn)
}
