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
	"bytes"
	"encoding/json"
	"errors"
)

const (
	ErrorArguments      = "Error: Insufficient arguments"
	ErrorNotImplemented = "Error: Method not implemented"
	ErrorUnknown        = "Error: Unknown error"
	ErrorParseRequest   = "Error: Could not parse request"
	ErrorDecodeArgs     = "Error: Could not decode arguments"
)

type ErrorResponse struct {
	Error     bool   `json:"error"`
	ErrorText string `json:"errorText"`
}

type RpcSuccessResponse struct {
	ID      int         `json:"id"`
	JsonRpc string      `json:"jsonrpc"`
	Error   bool        `json:"error"`
	Result  interface{} `json:"result"`
}

type RpcErrorResponse struct {
	ID        int    `json:"id"`
	JsonRpc   string `json:"jsonrpc"`
	Error     bool   `json:"error"`
	ErrorText string `json:"errortext"`
}

type RpcRequest struct {
	JsonRpc string            `json:"jsonrpc"`
	ID      int               `json:"id"`
	Method  string            `json:"method"`
	Params  []json.RawMessage `json:"params"`
}

func (req *RpcRequest) ToGetBlockArgs() (*GetBlockArgs, error) {
	if len(req.Params) < 1 {
		return nil, NewErrorResponse(ErrorArguments)
	}

	args := new(GetBlockArgs)
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(args)
	if err != nil {
		return nil, NewErrorResponse(ErrorDecodeArgs)
	}
	rpclogger.DebugDetailf("%T %v", args, args)
	return args, nil
}

func (req *RpcRequest) ToNewTxArgs() (*NewTxArgs, error) {
	if len(req.Params) < 7 {
		return nil, NewErrorResponse(ErrorArguments)
	}

	args := new(NewTxArgs)
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(args)
	if err != nil {
		return nil, NewErrorResponse(ErrorDecodeArgs)
	}
	rpclogger.DebugDetailf("%T %v", args, args)
	return args, nil
}

func (req *RpcRequest) ToPushTxArgs() (*PushTxArgs, error) {
	if len(req.Params) < 1 {
		return nil, NewErrorResponse(ErrorArguments)
	}

	args := new(PushTxArgs)
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(args)
	if err != nil {
		return nil, NewErrorResponse(ErrorDecodeArgs)
	}
	rpclogger.DebugDetailf("%T %v", args, args)
	return args, nil
}

func (req *RpcRequest) ToGetStorageArgs() (*GetStorageArgs, error) {
	if len(req.Params) < 2 {
		return nil, NewErrorResponse(ErrorArguments)
	}

	args := new(GetStorageArgs)
	// TODO need to pass both arguments
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(args)
	if err != nil {
		return nil, NewErrorResponse(ErrorDecodeArgs)
	}
	rpclogger.DebugDetailf("%T %v", args, args)
	return args, nil
}

func (req *RpcRequest) ToGetTxCountArgs() (*GetTxCountArgs, error) {
	if len(req.Params) < 1 {
		return nil, NewErrorResponse(ErrorArguments)
	}

	args := new(GetTxCountArgs)
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(args)
	if err != nil {
		return nil, NewErrorResponse(ErrorDecodeArgs)
	}
	rpclogger.DebugDetailf("%T %v", args, args)
	return args, nil
}

func (req *RpcRequest) ToGetBalanceArgs() (*GetBalanceArgs, error) {
	if len(req.Params) < 1 {
		return nil, NewErrorResponse(ErrorArguments)
	}

	args := new(GetBalanceArgs)
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(args)
	if err != nil {
		return nil, NewErrorResponse(ErrorDecodeArgs)
	}
	rpclogger.DebugDetailf("%T %v", args, args)
	return args, nil
}

func (req *RpcRequest) ToGetCodeAtArgs() (*GetCodeAtArgs, error) {
	if len(req.Params) < 1 {
		return nil, NewErrorResponse(ErrorArguments)
	}

	args := new(GetCodeAtArgs)
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(args)
	if err != nil {
		return nil, NewErrorResponse(ErrorDecodeArgs)
	}
	rpclogger.DebugDetailf("%T %v", args, args)
	return args, nil
}

func NewErrorResponse(msg string) error {
	return errors.New(msg)
}
