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
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
)

// When https://github.com/golang/go/issues/4674 is implemented this could be replaced
type stoppableTCPListener struct {
	*net.TCPListener
	stop chan struct{} // closed when the listener must stop
}

func newStoppableTCPListener(addr string) (*stoppableTCPListener, error) {
	wl, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	if tcpl, ok := wl.(*net.TCPListener); ok {
		stop := make(chan struct{})
		return &stoppableTCPListener{tcpl, stop}, nil
	}

	return nil, fmt.Errorf("Unable to create TCP listener for RPC service")
}

// Stop the listener and all accepted and still active connections.
func (self *stoppableTCPListener) Stop() {
	close(self.stop)
}

func (self *stoppableTCPListener) Accept() (net.Conn, error) {
	for {
		self.SetDeadline(time.Now().Add(time.Duration(1 * time.Second)))
		c, err := self.TCPListener.AcceptTCP()

		select {
		case <-self.stop:
			if c != nil { // accept timeout
				c.Close()
			}
			self.TCPListener.Close()
			return nil, listenerStoppedError
		default:
		}

		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() && netErr.Temporary() {
				continue // regular timeout
			}
		}

		return &closableConnection{c, self.stop}, err
	}
}

type closableConnection struct {
	*net.TCPConn
	closed chan struct{}
}

func (self *closableConnection) Read(b []byte) (n int, err error) {
	select {
	case <-self.closed:
		self.TCPConn.Close()
		return 0, io.EOF
	default:
		return self.TCPConn.Read(b)
	}
}

// Wraps the default handler and checks if the RPC service was stopped. In that case it returns an
// error indicating that the service was stopped. This will only happen for connections which are
// kept open (HTTP keep-alive) when the RPC service was shutdown.
func newStoppableHandler(h http.Handler, stop chan struct{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-stop:
			w.Header().Set("Content-Type", "application/json")
			err := fmt.Errorf("RPC service stopped")
			response := shared.NewRpcResponse(-1, shared.JsonRpcVersion, nil, err)
			httpSend(w, response)
		default:
			h.ServeHTTP(w, r)
		}
	})
}

func httpSend(writer io.Writer, v interface{}) (n int, err error) {
	var payload []byte
	payload, err = json.MarshalIndent(v, "", "\t")
	if err != nil {
		glog.V(logger.Error).Infoln("Error marshalling JSON", err)
		return 0, err
	}
	glog.V(logger.Detail).Infof("Sending payload: %s", payload)

	return writer.Write(payload)
}

func gethHttpHandler(codec codec.Codec, a shared.EthereumApi) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Limit request size to resist DoS
		if req.ContentLength > maxHttpSizeReqLength {
			err := fmt.Errorf("Request too large")
			response := shared.NewRpcErrorResponse(-1, shared.JsonRpcVersion, -32700, err)
			httpSend(w, &response)
			return
		}

		defer req.Body.Close()
		payload, err := ioutil.ReadAll(req.Body)
		if err != nil {
			err := fmt.Errorf("Could not read request body")
			response := shared.NewRpcErrorResponse(-1, shared.JsonRpcVersion, -32700, err)
			httpSend(w, &response)
			return
		}

		c := codec.New(nil)
		var rpcReq shared.Request
		if err = c.Decode(payload, &rpcReq); err == nil {
			reply, err := a.Execute(&rpcReq)
			res := shared.NewRpcResponse(rpcReq.Id, rpcReq.Jsonrpc, reply, err)
			httpSend(w, &res)
			return
		}

		var reqBatch []shared.Request
		if err = c.Decode(payload, &reqBatch); err == nil {
			resBatch := make([]*interface{}, len(reqBatch))
			resCount := 0

			for i, rpcReq := range reqBatch {
				reply, err := a.Execute(&rpcReq)
				if rpcReq.Id != nil { // this leaves nil entries in the response batch for later removal
					resBatch[i] = shared.NewRpcResponse(rpcReq.Id, rpcReq.Jsonrpc, reply, err)
					resCount += 1
				}
			}

			// make response omitting nil entries
			resBatch = resBatch[:resCount]
			httpSend(w, resBatch)
			return
		}

		// invalid request
		err = fmt.Errorf("Could not decode request")
		res := shared.NewRpcErrorResponse(-1, shared.JsonRpcVersion, -32600, err)
		httpSend(w, res)
	})
}
