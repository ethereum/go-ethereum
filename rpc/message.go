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

	"github.com/ethereum/go-ethereum/state"
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

type Log struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}

func toLogs(logs state.Logs) (ls []Log) {
	ls = make([]Log, len(logs))

	for i, log := range logs {
		var l Log
		l.Topics = make([]string, len(log.Topics()))
		l.Address = toHex(log.Address())
		l.Data = toHex(log.Data())
		for j, topic := range log.Topics() {
			l.Topics[j] = toHex(topic)
		}
		ls[i] = l
	}

	return
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
