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
	"encoding/json"
	"errors"
	"fmt"
)

var (
	errArguments      = errors.New("Error: Insufficient arguments")
	errNotImplemented = errors.New("Error: Method not implemented")
	errUnknown        = errors.New("Error: Unknown error")
	errDecodeArgs     = errors.New("Error: Could not decode arguments")
)

type RpcRequest struct {
	ID      interface{}     `json:"id"`
	JsonRpc string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type RpcSuccessResponse struct {
	ID      interface{} `json:"id"`
	JsonRpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
}

type RpcErrorResponse struct {
	ID      interface{}     `json:"id"`
	JsonRpc string          `json:"jsonrpc"`
	Error   *RpcErrorObject `json:"error"`
}

type RpcErrorObject struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	// Data    interface{} `json:"data"`
}

func NewErrorWithMessage(err error, msg string) error {
	return fmt.Errorf("%s: %s", err.Error(), msg)
}

// func (req *RpcRequest) ToRegisterArgs() (string, error) {
// 	if len(req.Params) < 1 {
// 		return "", errArguments
// 	}

// 	var args string
// 	err := json.Unmarshal(req.Params, &args)
// 	if err != nil {
// 		return "", err
// 	}

// 	return args, nil
// }

// func (req *RpcRequest) ToWatchTxArgs() (string, error) {
// 	if len(req.Params) < 1 {
// 		return "", errArguments
// 	}

// 	var args string
// 	err := json.Unmarshal(req.Params, &args)
// 	if err != nil {
// 		return "", err
// 	}

// 	return args, nil
// }
