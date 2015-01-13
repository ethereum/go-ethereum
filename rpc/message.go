package rpc

import (
	"bytes"
	"encoding/json"
	"errors"

	// "github.com/ethereum/go-ethereum/ethutil"
)

const (
	ErrorArguments      = "Error: Insufficient arguments"
	ErrorNotImplemented = "Error: Method not implemented"
	ErrorUnknown        = "Error: Unknown error"
	ErrorParseRequest   = "Error: Could not parse request"
	ErrorDecodeArgs     = "Error: Could not decode arguments"
)

// type JsonResponse interface {
// }

type ErrorResponse struct {
	Error     bool   `json:"error"`
	ErrorText string `json:"errorText"`
}

// type SuccessRes struct {
// 	Error bool `json:"error"`
// 	Result JsonResponse `json:"result"`
// }

// type Message struct {
// 	Call string        `json:"call"`
// 	Args []interface{} `json:"args"`
// 	Id   int           `json:"_id"`
// 	Data interface{}   `json:"data"`
// }

// func (self *Message) Arguments() *ethutil.Value {
// 	return ethutil.NewValue(self.Args)
// }

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
	jsonlogger.DebugDetailf("%T %v", args, args)
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
	jsonlogger.DebugDetailf("%T %v", args, args)
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
	jsonlogger.DebugDetailf("%T %v", args, args)
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
	jsonlogger.DebugDetailf("%T %v", args, args)
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
	jsonlogger.DebugDetailf("%T %v", args, args)
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
	jsonlogger.DebugDetailf("%T %v", args, args)
	return args, nil
}

// func NewSuccessRes(object JsonResponse) string {
// 	e := SuccessRes{Error: false, Result: object}
// 	res, err := json.Marshal(e)
// 	if err != nil {
// 		// This should never happen
// 		panic("Creating json error response failed, help")
// 	}
// 	success := string(res)
// 	return success
// 	// return res
// }

func NewErrorResponse(msg string) error {
	return errors.New(msg)
}
