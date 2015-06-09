package shared

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

// RPC request
type Request struct {
	Id      interface{}     `json:"id"`
	Jsonrpc string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// RPC response
type Response struct {
	Id      interface{} `json:"id"`
	Jsonrpc string      `json:"jsonrpc"`
}

// RPC success response
type SuccessResponse struct {
	Id      interface{} `json:"id"`
	Jsonrpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
}

// RPC error response
type ErrorResponse struct {
	Id      interface{}  `json:"id"`
	Jsonrpc string       `json:"jsonrpc"`
	Error   *ErrorObject `json:"error"`
}

// RPC error response details
type ErrorObject struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	// Data    interface{} `json:"data"`
}

func NewRpcResponse(id interface{}, jsonrpcver string, reply interface{}, err error) *interface{} {
	var response interface{}

	switch err.(type) {
	case nil:
		response = &SuccessResponse{Jsonrpc: jsonrpcver, Id: id, Result: reply}
	case *NotImplementedError:
		jsonerr := &ErrorObject{-32601, err.Error()}
		response = &ErrorResponse{Jsonrpc: jsonrpcver, Id: id, Error: jsonerr}
	case *DecodeParamError, *InsufficientParamsError, *ValidationError, *InvalidTypeError:
		jsonerr := &ErrorObject{-32602, err.Error()}
		response = &ErrorResponse{Jsonrpc: jsonrpcver, Id: id, Error: jsonerr}
	default:
		jsonerr := &ErrorObject{-32603, err.Error()}
		response = &ErrorResponse{Jsonrpc: jsonrpcver, Id: id, Error: jsonerr}
	}

	glog.V(logger.Detail).Infof("Generated response: %T %s", response, response)
	return &response
}
