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
package websocket

import (
	"fmt"
	"net"
	"net/http"

	ws "code.google.com/p/go.net/websocket"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/event/filter"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/xeth"
)

var wslogger = logger.NewLogger("RPC-WS")

type WebSocketServer struct {
	eth           *eth.Ethereum
	filterManager *filter.FilterManager
	port          int
	doneCh        chan bool
	listener      net.Listener
}

func NewWebSocketServer(eth *eth.Ethereum, port int) (*WebSocketServer, error) {
	sport := fmt.Sprintf(":%d", port)
	l, err := net.Listen("tcp", sport)
	if err != nil {
		return nil, err
	}

	filterManager := filter.NewFilterManager(eth.EventMux())
	go filterManager.Start()

	return &WebSocketServer{eth,
		filterManager,
		port,
		make(chan bool),
		l,
	}, nil
}

func (self *WebSocketServer) handlerLoop() {
	for {
		select {
		case <-self.doneCh:
			wslogger.Infoln("Shutdown RPC-WS server")
			return
		}
	}
}

func (self *WebSocketServer) Stop() {
	close(self.doneCh)
}

func (self *WebSocketServer) Start() {
	wslogger.Infof("Starting RPC-WS server on port %d", self.port)
	go self.handlerLoop()

	api := rpc.NewEthereumApi(xeth.NewJSXEth(self.eth))
	h := self.apiHandler(api)
	http.Handle("/ws", h)

	err := http.Serve(self.listener, nil)
	if err != nil {
		wslogger.Errorln("Error on RPC-WS interface:", err)
	}
}

func (s *WebSocketServer) apiHandler(xeth *rpc.EthereumApi) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		h := sockHandler(xeth)
		s := ws.Server{Handler: h}
		s.ServeHTTP(w, req)
	}

	return http.HandlerFunc(fn)
}

func sockHandler(xeth *rpc.EthereumApi) ws.Handler {
	fn := func(conn *ws.Conn) {
		for {
			// FIX wslogger does not output to console
			wslogger.Debugln("Handling request")
			var reqParsed rpc.RpcRequest

			if err := ws.JSON.Receive(conn, &reqParsed); err != nil {
				wslogger.Debugln(rpc.ErrorParseRequest)
				ws.JSON.Send(conn, rpc.RpcErrorResponse{JsonRpc: reqParsed.JsonRpc, ID: reqParsed.ID, Error: true, ErrorText: rpc.ErrorParseRequest})
				continue
			}

			var response interface{}
			reserr := xeth.GetRequestReply(&reqParsed, &response)
			if reserr != nil {
				wslogger.Errorln(reserr)
				ws.JSON.Send(conn, rpc.RpcErrorResponse{JsonRpc: reqParsed.JsonRpc, ID: reqParsed.ID, Error: true, ErrorText: reserr.Error()})
				continue
			}

			wslogger.Debugf("Generated response: %T %s", response, response)
			ws.JSON.Send(conn, rpc.RpcSuccessResponse{JsonRpc: reqParsed.JsonRpc, ID: reqParsed.ID, Error: false, Result: response})
		}
	}
	return fn
}
