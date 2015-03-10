package rpc

import (
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
	var json JsonWrapper
	api := NewEthereumApi(pipe, dataDir)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		rpchttplogger.DebugDetailln("Handling request")

		if req.ContentLength > maxSizeReqLength {
			jsonerr := &RpcErrorObject{-32700, "Error: Request too large"}
			json.Send(w, &RpcErrorResponse{JsonRpc: jsonrpcver, ID: nil, Error: jsonerr})
			return
		}

		reqParsed, reqerr := json.ParseRequestBody(req)
		if reqerr != nil {
			jsonerr := &RpcErrorObject{-32700, "Error: Could not parse request"}
			json.Send(w, &RpcErrorResponse{JsonRpc: jsonrpcver, ID: nil, Error: jsonerr})
			return
		}

		var response interface{}
		reserr := api.GetRequestReply(&reqParsed, &response)
		if reserr != nil {
			rpchttplogger.Warnln(reserr)
			jsonerr := &RpcErrorObject{-32603, reserr.Error()}
			json.Send(w, &RpcErrorResponse{JsonRpc: jsonrpcver, ID: reqParsed.ID, Error: jsonerr})
			return
		}

		rpchttplogger.DebugDetailf("Generated response: %T %s", response, response)
		json.Send(w, &RpcSuccessResponse{JsonRpc: jsonrpcver, ID: reqParsed.ID, Result: response})
	})
}
