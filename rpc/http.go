package rpc

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/xeth"
)

var rpchttplogger = logger.NewLogger("RPC-HTTP")

const (
	jsonrpcver       = "2.0"
	maxSizeReqLength = 1024 * 1024 // 1MB
)

// JSONRPC returns a handler that implements the Ethereum JSON-RPC API.
func JSONRPC(pipe *xeth.XEth, dataDir string) http.Handler {
	var jsw JsonWrapper
	api := NewEthereumApi(pipe, dataDir)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// TODO this needs to be configurable
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Limit request size to resist DoS
		if req.ContentLength > maxSizeReqLength {
			jsonerr := &RpcErrorObject{-32700, "Request too large"}
			jsw.Send(w, &RpcErrorResponse{Jsonrpc: jsonrpcver, Id: nil, Error: jsonerr})
			return
		}

		defer req.Body.Close()
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			jsonerr := &RpcErrorObject{-32700, "Could not read request body"}
			jsw.Send(w, &RpcErrorResponse{Jsonrpc: jsonrpcver, Id: nil, Error: jsonerr})
		}

		// Try to parse the request as a single
		var reqSingle RpcRequest
		if err := json.Unmarshal(body, &reqSingle); err == nil {
			response := RpcResponse(api, &reqSingle)
			jsw.Send(w, &response)
			return
		}

		// Try to parse the request to batch
		var reqBatch []RpcRequest
		if err := json.Unmarshal(body, &reqBatch); err == nil {
			// Build response batch
			resBatch := make([]*interface{}, len(reqBatch))
			for i, request := range reqBatch {
				response := RpcResponse(api, &request)
				resBatch[i] = response
			}
			jsw.Send(w, resBatch)
			return
		}

		// Not a batch or single request, error
		jsonerr := &RpcErrorObject{-32600, "Could not decode request"}
		jsw.Send(w, &RpcErrorResponse{Jsonrpc: jsonrpcver, Id: nil, Error: jsonerr})
	})
}

func RpcResponse(api *EthereumApi, request *RpcRequest) *interface{} {
	var reply, response interface{}
	reserr := api.GetRequestReply(request, &reply)
	switch reserr.(type) {
	case nil:
		response = &RpcSuccessResponse{Jsonrpc: jsonrpcver, Id: request.Id, Result: reply}
	case *NotImplementedError:
		jsonerr := &RpcErrorObject{-32601, reserr.Error()}
		response = &RpcErrorResponse{Jsonrpc: jsonrpcver, Id: request.Id, Error: jsonerr}
	case *DecodeParamError, *InsufficientParamsError, *ValidationError:
		jsonerr := &RpcErrorObject{-32602, reserr.Error()}
		response = &RpcErrorResponse{Jsonrpc: jsonrpcver, Id: request.Id, Error: jsonerr}
	default:
		jsonerr := &RpcErrorObject{-32603, reserr.Error()}
		response = &RpcErrorResponse{Jsonrpc: jsonrpcver, Id: request.Id, Error: jsonerr}
	}

	rpchttplogger.DebugDetailf("Generated response: %T %s", response, response)
	return &response
}
