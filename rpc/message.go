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

var (
	errArguments      = errors.New("Error: Insufficient arguments")
	errNotImplemented = errors.New("Error: Method not implemented")
	errUnknown        = errors.New("Error: Unknown error")
	errParseRequest   = errors.New("Error: Could not parse request")
	errDecodeArgs     = errors.New("Error: Could not decode arguments")
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

func NewErrorWithMessage(err error, msg string) error {
	return fmt.Errorf("%s: %s", err.Error(), msg)
}

func (req *RpcRequest) ToSha3Args() (*Sha3Args, error) {
	if len(req.Params) < 1 {
		return nil, errArguments
	}

	args := new(Sha3Args)
	r := bytes.NewReader(req.Params[0])
	if err := json.NewDecoder(r).Decode(args); err != nil {
		return nil, errDecodeArgs
	}

	return args, nil
}

func (req *RpcRequest) ToGetBlockArgs() (*GetBlockArgs, error) {
	if len(req.Params) < 1 {
		return nil, errArguments
	}

	args := new(GetBlockArgs)
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(args)
	if err != nil {
		return nil, errDecodeArgs
	}

	return args, nil
}

func (req *RpcRequest) ToNewTxArgs() (*NewTxArgs, error) {
	if len(req.Params) < 1 {
		return nil, errArguments
	}

	args := new(NewTxArgs)
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(args)
	if err != nil {
		return nil, NewErrorWithMessage(errDecodeArgs, err.Error())
	}

	return args, nil
}

func (req *RpcRequest) ToPushTxArgs() (*PushTxArgs, error) {
	if len(req.Params) < 1 {
		return nil, errArguments
	}

	args := new(PushTxArgs)
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(args)
	if err != nil {
		return nil, errDecodeArgs
	}

	return args, nil
}

func (req *RpcRequest) ToGetStateArgs() (*GetStateArgs, error) {
	if len(req.Params) < 1 {
		return nil, errArguments
	}

	args := new(GetStateArgs)
	// TODO need to pass both arguments
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(args)
	if err != nil {
		return nil, errDecodeArgs
	}

	return args, nil
}

func (req *RpcRequest) ToStorageAtArgs() (*GetStorageArgs, error) {
	if len(req.Params) < 1 {
		return nil, errArguments
	}

	args := new(GetStorageArgs)
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(args)
	if err != nil {
		return nil, errDecodeArgs
	}

	return args, nil
}

func (req *RpcRequest) ToGetTxCountArgs() (*GetTxCountArgs, error) {
	if len(req.Params) < 1 {
		return nil, errArguments
	}

	args := new(GetTxCountArgs)
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(args)
	if err != nil {
		return nil, errDecodeArgs
	}

	return args, nil
}

func (req *RpcRequest) ToGetBalanceArgs() (*GetBalanceArgs, error) {
	if len(req.Params) < 1 {
		return nil, errArguments
	}

	args := new(GetBalanceArgs)
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(args)
	if err != nil {
		return nil, errDecodeArgs
	}

	return args, nil
}

func (req *RpcRequest) ToGetCodeAtArgs() (*GetCodeAtArgs, error) {
	if len(req.Params) < 1 {
		return nil, errArguments
	}

	args := new(GetCodeAtArgs)
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(args)
	if err != nil {
		return nil, errDecodeArgs
	}

	return args, nil
}

func (req *RpcRequest) ToBoolArgs() (bool, error) {
	if len(req.Params) < 1 {
		return false, errArguments
	}

	var args bool
	err := json.Unmarshal(req.Params[0], &args)
	if err != nil {
		return false, errDecodeArgs
	}

	return args, nil
}

func (req *RpcRequest) ToCompileArgs() (string, error) {
	if len(req.Params) < 1 {
		return "", errArguments
	}

	var args string
	err := json.Unmarshal(req.Params[0], &args)
	if err != nil {
		return "", errDecodeArgs
	}

	return args, nil
}

func (req *RpcRequest) ToFilterArgs() (*FilterOptions, error) {
	if len(req.Params) < 1 {
		return nil, errArguments
	}

	args := new(FilterOptions)
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(args)
	if err != nil {
		return nil, errDecodeArgs
	}
	return args, nil
}

func (req *RpcRequest) ToFilterStringArgs() (string, error) {
	if len(req.Params) < 1 {
		return "", errArguments
	}

	var args string
	err := json.Unmarshal(req.Params[0], &args)
	if err != nil {
		return "", errDecodeArgs
	}

	return args, nil
}

func (req *RpcRequest) ToUninstallFilterArgs() (int, error) {
	if len(req.Params) < 1 {
		return 0, errArguments
	}

	var args int
	err := json.Unmarshal(req.Params[0], &args)
	if err != nil {
		return 0, errDecodeArgs
	}

	return args, nil
}

func (req *RpcRequest) ToFilterChangedArgs() (int, error) {
	if len(req.Params) < 1 {
		return 0, errArguments
	}

	var id int
	r := bytes.NewReader(req.Params[0])
	err := json.NewDecoder(r).Decode(&id)
	if err != nil {
		return 0, errDecodeArgs
	}
	return id, nil
}

func (req *RpcRequest) ToDbPutArgs() (*DbArgs, error) {
	if len(req.Params) < 3 {
		return nil, errArguments
	}

	var args DbArgs
	err := json.Unmarshal(req.Params[0], &args.Database)
	if err != nil {
		return nil, NewErrorWithMessage(errDecodeArgs, err.Error())
	}
	err = json.Unmarshal(req.Params[1], &args.Key)
	if err != nil {
		return nil, NewErrorWithMessage(errDecodeArgs, err.Error())
	}
	err = json.Unmarshal(req.Params[2], &args.Value)
	if err != nil {
		return nil, NewErrorWithMessage(errDecodeArgs, err.Error())
	}

	return &args, nil
}

func (req *RpcRequest) ToDbGetArgs() (*DbArgs, error) {
	if len(req.Params) < 2 {
		return nil, errArguments
	}

	var args DbArgs
	err := json.Unmarshal(req.Params[0], &args.Database)
	if err != nil {
		return nil, NewErrorWithMessage(errDecodeArgs, err.Error())
	}

	err = json.Unmarshal(req.Params[1], &args.Key)
	if err != nil {
		return nil, NewErrorWithMessage(errDecodeArgs, err.Error())
	}

	return &args, nil
}

func (req *RpcRequest) ToWhisperFilterArgs() (*xeth.Options, error) {
	if len(req.Params) < 1 {
		return nil, errArguments
	}

	var args xeth.Options
	err := json.Unmarshal(req.Params[0], &args)
	if err != nil {
		return nil, NewErrorWithMessage(errDecodeArgs, err.Error())
	}

	return &args, nil
}

func (req *RpcRequest) ToIdArgs() (int, error) {
	if len(req.Params) < 1 {
		return 0, errArguments
	}

	var id int
	err := json.Unmarshal(req.Params[0], &id)
	if err != nil {
		return 0, errDecodeArgs
	}

	return id, nil
}

func (req *RpcRequest) ToWhisperPostArgs() (*WhisperMessageArgs, error) {
	if len(req.Params) < 1 {
		return nil, errArguments
	}

	var args WhisperMessageArgs
	err := json.Unmarshal(req.Params[0], &args)
	if err != nil {
		return nil, err
	}

	return &args, nil
}

func (req *RpcRequest) ToWhisperHasIdentityArgs() (string, error) {
	if len(req.Params) < 1 {
		return "", errArguments
	}

	var args string
	err := json.Unmarshal(req.Params[0], &args)
	if err != nil {
		return "", err
	}

	return args, nil
}

func (req *RpcRequest) ToRegisterArgs() (string, error) {
	if len(req.Params) < 1 {
		return "", errArguments
	}

	var args string
	err := json.Unmarshal(req.Params[0], &args)
	if err != nil {
		return "", err
	}

	return args, nil
}

func (req *RpcRequest) ToWatchTxArgs() (string, error) {
	if len(req.Params) < 1 {
		return "", errArguments
	}

	var args string
	err := json.Unmarshal(req.Params[0], &args)
	if err != nil {
		return "", err
	}

	return args, nil
}
