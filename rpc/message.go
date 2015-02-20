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
	"fmt"

	"github.com/ethereum/go-ethereum/xeth"
)

const (
	ErrorArguments      = "Error: Insufficient arguments"
	ErrorNotImplemented = "Error: Method not implemented"
	ErrorUnknown        = "Error: Unknown error"
	ErrorParseRequest   = "Error: Could not parse request"
	ErrorDecodeArgs     = "Error: Could not decode arguments"
)

type RpcRequest struct {
	ID      interface{}       `json:"id"`
	JsonRpc string            `json:"jsonrpc"`
	Method  string            `json:"method"`
	Params  []json.RawMessage `json:"params"`
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

func NewErrorResponse(msg string) error {
	return errors.New(msg)
}

func NewErrorResponseWithError(msg string, err error) error {
	return fmt.Errorf("%s: %v", msg, err)
}

func (req *RpcRequest) ToSha3Args() (*Sha3Args, error) {
	if len(req.Params) < 1 {
		return nil, NewErrorResponse(ErrorArguments)
	}

	args := new(Sha3Args)
	r := bytes.NewReader(req.Params[0])
	if err := json.NewDecoder(r).Decode(args); err != nil {
		return nil, NewErrorResponse(ErrorDecodeArgs)
	}
	rpclogger.DebugDetailf("%T %v", args, args)
	return args, nil
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
	if len(req.Params) < 1 {
		return nil, NewErrorResponse(ErrorArguments)
	}

	args := new(NewTxArgs)
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(args)
	if err != nil {
		return nil, NewErrorResponseWithError(ErrorDecodeArgs, err)
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

func (req *RpcRequest) ToGetStateArgs() (*GetStateArgs, error) {
	if len(req.Params) < 1 {
		return nil, NewErrorResponse(ErrorArguments)
	}

	args := new(GetStateArgs)
	// TODO need to pass both arguments
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(args)
	if err != nil {
		return nil, NewErrorResponse(ErrorDecodeArgs)
	}
	rpclogger.DebugDetailf("%T %v", args, args)
	return args, nil
}

func (req *RpcRequest) ToStorageAtArgs() (*GetStorageArgs, error) {
	if len(req.Params) < 1 {
		return nil, NewErrorResponse(ErrorArguments)
	}

	args := new(GetStorageArgs)
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

func (req *RpcRequest) ToBoolArgs() (bool, error) {
	if len(req.Params) < 1 {
		return false, NewErrorResponse(ErrorArguments)
	}

	var args bool
	err := json.Unmarshal(req.Params[0], &args)
	if err != nil {
		return false, NewErrorResponse(ErrorDecodeArgs)
	}

	rpclogger.DebugDetailf("%T %v", args, args)
	return args, nil
}

func (req *RpcRequest) ToCompileArgs() (string, error) {
	if len(req.Params) < 1 {
		return "", NewErrorResponse(ErrorArguments)
	}

	var args string
	err := json.Unmarshal(req.Params[0], &args)
	if err != nil {
		return "", NewErrorResponse(ErrorDecodeArgs)
	}

	rpclogger.DebugDetailf("%T %v", args, args)
	return args, nil
}

func (req *RpcRequest) ToFilterArgs() (*FilterOptions, error) {
	if len(req.Params) < 1 {
		return nil, NewErrorResponse(ErrorArguments)
	}

	args := new(FilterOptions)
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(args)
	if err != nil {
		return nil, NewErrorResponse(ErrorDecodeArgs)
	}
	rpclogger.DebugDetailf("%T %v", args, args)
	return args, nil
}

func (req *RpcRequest) ToFilterStringArgs() (string, error) {
	if len(req.Params) < 1 {
		return "", NewErrorResponse(ErrorArguments)
	}

	var args string
	err := json.Unmarshal(req.Params[0], &args)
	if err != nil {
		return "", NewErrorResponse(ErrorDecodeArgs)
	}

	rpclogger.DebugDetailf("%T %v", args, args)
	return args, nil
}

func (req *RpcRequest) ToUninstallFilterArgs() (int, error) {
	if len(req.Params) < 1 {
		return 0, NewErrorResponse(ErrorArguments)
	}

	var args int
	err := json.Unmarshal(req.Params[0], &args)
	if err != nil {
		return 0, NewErrorResponse(ErrorDecodeArgs)
	}

	rpclogger.DebugDetailf("%T %v", args, args)
	return args, nil
}

func (req *RpcRequest) ToFilterChangedArgs() (int, error) {
	if len(req.Params) < 1 {
		return 0, NewErrorResponse(ErrorArguments)
	}

	var id int
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(&id)
	if err != nil {
		return 0, NewErrorResponse(ErrorDecodeArgs)
	}
	rpclogger.DebugDetailf("%T %v", id, id)
	return id, nil
}

func (req *RpcRequest) ToDbPutArgs() (*DbArgs, error) {
	if len(req.Params) < 3 {
		return nil, NewErrorResponse(ErrorArguments)
	}

	var args DbArgs
	err := json.Unmarshal(req.Params[0], &args.Database)
	if err != nil {
		return nil, NewErrorResponseWithError(ErrorDecodeArgs, err)
	}
	err = json.Unmarshal(req.Params[1], &args.Key)
	if err != nil {
		return nil, NewErrorResponseWithError(ErrorDecodeArgs, err)
	}
	err = json.Unmarshal(req.Params[2], &args.Value)
	if err != nil {
		return nil, NewErrorResponseWithError(ErrorDecodeArgs, err)
	}
	rpclogger.DebugDetailf("%T %v", args, args)
	return &args, nil
}

func (req *RpcRequest) ToDbGetArgs() (*DbArgs, error) {
	if len(req.Params) < 2 {
		return nil, NewErrorResponse(ErrorArguments)
	}

	var args DbArgs
	err := json.Unmarshal(req.Params[0], &args.Database)
	if err != nil {
		return nil, NewErrorResponseWithError(ErrorDecodeArgs, err)
	}

	err = json.Unmarshal(req.Params[1], &args.Key)
	if err != nil {
		return nil, NewErrorResponseWithError(ErrorDecodeArgs, err)
	}
	rpclogger.DebugDetailf("%T %v", args, args)
	return &args, nil
}

func (req *RpcRequest) ToWhisperFilterArgs() (*xeth.Options, error) {
	if len(req.Params) < 1 {
		return nil, NewErrorResponse(ErrorArguments)
	}

	var args xeth.Options
	err := json.Unmarshal(req.Params[0], &args)
	if err != nil {
		return nil, NewErrorResponseWithError(ErrorDecodeArgs, err)
	}
	rpclogger.DebugDetailf("%T %v", args, args)
	return &args, nil
}

func (req *RpcRequest) ToIdArgs() (int, error) {
	if len(req.Params) < 1 {
		return 0, NewErrorResponse(ErrorArguments)
	}

	var id int
	err := json.Unmarshal(req.Params[0], &id)
	if err != nil {
		return 0, NewErrorResponse(ErrorDecodeArgs)
	}
	rpclogger.DebugDetailf("%T %v", id, id)
	return id, nil
}

func (req *RpcRequest) ToWhisperPostArgs() (*WhisperMessageArgs, error) {
	if len(req.Params) < 1 {
		return nil, NewErrorResponse(ErrorArguments)
	}

	var args WhisperMessageArgs
	err := json.Unmarshal(req.Params[0], &args)
	if err != nil {
		return nil, err
	}
	rpclogger.DebugDetailf("%T %v", args, args)
	return &args, nil
}

func (req *RpcRequest) ToWhisperHasIdentityArgs() (string, error) {
	if len(req.Params) < 1 {
		return "", NewErrorResponse(ErrorArguments)
	}

	var args string
	err := json.Unmarshal(req.Params[0], &args)
	if err != nil {
		return "", err
	}
	rpclogger.DebugDetailf("%T %v", args, args)
	return args, nil
}

func (req *RpcRequest) ToRegisterArgs() (string, error) {
	if len(req.Params) < 1 {
		return "", NewErrorResponse(ErrorArguments)
	}

	var args string
	err := json.Unmarshal(req.Params[0], &args)
	if err != nil {
		return "", err
	}
	rpclogger.DebugDetailf("%T %v", args, args)
	return args, nil
}

func (req *RpcRequest) ToWatchTxArgs() (string, error) {
	if len(req.Params) < 1 {
		return "", NewErrorResponse(ErrorArguments)
	}

	var args string
	err := json.Unmarshal(req.Params[0], &args)
	if err != nil {
		return "", err
	}
	rpclogger.DebugDetailf("%T %v", args, args)
	return args, nil
}
