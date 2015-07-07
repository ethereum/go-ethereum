// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package shared

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

// Ethereum RPC API interface
type EthereumApi interface {
	// API identifier
	Name() string

	// API version
	ApiVersion() string

	// Execute the given request and returns the response or an error
	Execute(*Request) (interface{}, error)

	// List of supported RCP methods this API provides
	Methods() []string
}

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

// Create RPC error response, this allows for custom error codes
func NewRpcErrorResponse(id interface{}, jsonrpcver string, errCode int, err error) *interface{} {
	var response interface{}

	jsonerr := &ErrorObject{errCode, err.Error()}
	response = ErrorResponse{Jsonrpc: jsonrpcver, Id: id, Error: jsonerr}

	glog.V(logger.Detail).Infof("Generated error response: %s", response)
	return &response
}

// Create RPC response
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
