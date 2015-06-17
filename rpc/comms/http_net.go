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
	"github.com/ethereum/go-ethereum/rpc/api"
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
			response := shared.NewRpcResponse(-1, api.JsonRpcVersion, nil, err)
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

func gethHttpHandler(codec codec.Codec, a api.EthereumApi) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Limit request size to resist DoS
		if req.ContentLength > maxHttpSizeReqLength {
			err := fmt.Errorf("Request too large")
			response := shared.NewRpcErrorResponse(-1, api.JsonRpcVersion, -32700, err)
			httpSend(w, &response)
			return
		}

		defer req.Body.Close()
		payload, err := ioutil.ReadAll(req.Body)
		if err != nil {
			err := fmt.Errorf("Could not read request body")
			response := shared.NewRpcErrorResponse(-1, api.JsonRpcVersion, -32700, err)
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
		res := shared.NewRpcErrorResponse(-1, api.JsonRpcVersion, -32600, err)
		httpSend(w, res)
	})
}
