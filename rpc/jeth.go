package rpc

import (
	"encoding/json"
	// "fmt"
	"github.com/obscuren/otto"
)

type Jeth struct {
	ethApi *EthereumApi
	toVal  func(interface{}) otto.Value
}

func NewJeth(ethApi *EthereumApi, toVal func(interface{}) otto.Value) *Jeth {
	return &Jeth{ethApi, toVal}
}

func (self *Jeth) err(code int, msg string, id interface{}) otto.Value {
	rpcerr := &RpcErrorObject{code, msg}
	rpcresponse := &RpcErrorResponse{Jsonrpc: jsonrpcver, Id: id, Error: rpcerr}
	return self.toVal(rpcresponse)
}

func (self *Jeth) Send(call otto.FunctionCall) (response otto.Value) {
	reqif, err := call.Argument(0).Export()
	if err != nil {
		return self.err(-32700, err.Error(), nil)
	}

	jsonreq, err := json.Marshal(reqif)

	var req RpcRequest
	err = json.Unmarshal(jsonreq, &req)

	var respif interface{}
	err = self.ethApi.GetRequestReply(&req, &respif)
	if err != nil {
		return self.err(-32603, err.Error(), req.Id)
	}
	rpcresponse := &RpcSuccessResponse{Jsonrpc: jsonrpcver, Id: req.Id, Result: respif}
	response = self.toVal(rpcresponse)
	return
}
