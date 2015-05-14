package rpc

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/xeth"
	"github.com/rs/cors"
)

var rpclistener *stoppableTCPListener

const (
	jsonrpcver       = "2.0"
	maxSizeReqLength = 1024 * 1024 // 1MB
)

func Start(pipe *xeth.XEth, config RpcConfig) error {
	if rpclistener != nil {
		if fmt.Sprintf("%s:%d", config.ListenAddress, config.ListenPort) != rpclistener.Addr().String() {
			return fmt.Errorf("RPC service already running on %s ", rpclistener.Addr().String())
		}
		return nil // RPC service already running on given host/port
	}

	l, err := newStoppableTCPListener(fmt.Sprintf("%s:%d", config.ListenAddress, config.ListenPort))
	if err != nil {
		glog.V(logger.Error).Infof("Can't listen on %s:%d: %v", config.ListenAddress, config.ListenPort, err)
		return err
	}
	rpclistener = l

	var handler http.Handler
	if len(config.CorsDomain) > 0 {
		var opts cors.Options
		opts.AllowedMethods = []string{"POST"}
		opts.AllowedOrigins = []string{config.CorsDomain}

		c := cors.New(opts)
		handler = newStoppableHandler(c.Handler(JSONRPC(pipe)), l.stop)
	} else {
		handler = newStoppableHandler(JSONRPC(pipe), l.stop)
	}

	go http.Serve(l, handler)

	return nil
}

func Stop() error {
	if rpclistener != nil {
		rpclistener.Stop()
		rpclistener = nil
	}

	return nil
}

// JSONRPC returns a handler that implements the Ethereum JSON-RPC API.
func JSONRPC(pipe *xeth.XEth) http.Handler {
	api := NewEthereumApi(pipe)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Limit request size to resist DoS
		if req.ContentLength > maxSizeReqLength {
			jsonerr := &RpcErrorObject{-32700, "Request too large"}
			send(w, &RpcErrorResponse{Jsonrpc: jsonrpcver, Id: nil, Error: jsonerr})
			return
		}

		// Read request body
		defer req.Body.Close()
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			jsonerr := &RpcErrorObject{-32700, "Could not read request body"}
			send(w, &RpcErrorResponse{Jsonrpc: jsonrpcver, Id: nil, Error: jsonerr})
		}

		// Try to parse the request as a single
		var reqSingle RpcRequest
		if err := json.Unmarshal(body, &reqSingle); err == nil {
			response := RpcResponse(api, &reqSingle)
			if reqSingle.Id != nil {
				send(w, &response)
			}
			return
		}

		// Try to parse the request to batch
		var reqBatch []RpcRequest
		if err := json.Unmarshal(body, &reqBatch); err == nil {
			// Build response batch
			resBatch := make([]*interface{}, len(reqBatch))
			resCount := 0

			for i, request := range reqBatch {
				response := RpcResponse(api, &request)
				// this leaves nil entries in the response batch for later removal
				if request.Id != nil {
					resBatch[i] = response
					resCount = resCount + 1
				}
			}

			// make response omitting nil entries
			respBatchComp := make([]*interface{}, resCount)
			resCount = resCount - 1
			for _, v := range resBatch {
				if v != nil {
					respBatchComp[resCount] = v
					resCount = resCount - 1
				}
			}

			send(w, respBatchComp)
			return
		}

		// Not a batch or single request, error
		jsonerr := &RpcErrorObject{-32600, "Could not decode request"}
		send(w, &RpcErrorResponse{Jsonrpc: jsonrpcver, Id: nil, Error: jsonerr})
	})
}

func RpcResponse(api *EthereumApi, request *RpcRequest) *interface{} {
	var reply, response interface{}
	reserr := api.GetRequestReply(request, &reply)
	switch reserr.(type) {
	case nil:
		response = &RpcSuccessResponse{Jsonrpc: jsonrpcver, Id: request.Id, Result: reply}
	case *NotImplementedError, *NotAvailableError:
		jsonerr := &RpcErrorObject{-32601, reserr.Error()}
		response = &RpcErrorResponse{Jsonrpc: jsonrpcver, Id: request.Id, Error: jsonerr}
	case *DecodeParamError, *InsufficientParamsError, *ValidationError, *InvalidTypeError:
		jsonerr := &RpcErrorObject{-32602, reserr.Error()}
		response = &RpcErrorResponse{Jsonrpc: jsonrpcver, Id: request.Id, Error: jsonerr}
	default:
		jsonerr := &RpcErrorObject{-32603, reserr.Error()}
		response = &RpcErrorResponse{Jsonrpc: jsonrpcver, Id: request.Id, Error: jsonerr}
	}

	glog.V(logger.Detail).Infof("Generated response: %T %s", response, response)
	return &response
}

func send(writer io.Writer, v interface{}) (n int, err error) {
	var payload []byte
	payload, err = json.MarshalIndent(v, "", "\t")
	if err != nil {
		glog.V(logger.Error).Infoln("Error marshalling JSON", err)
		return 0, err
	}
	glog.V(logger.Detail).Infof("Sending payload: %s", payload)

	return writer.Write(payload)
}
