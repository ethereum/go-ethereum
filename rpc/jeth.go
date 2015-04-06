package rpc

import (
	"encoding/json"
	"fmt"
	// "fmt"
	"github.com/ethereum/go-ethereum/jsre"
	"github.com/robertkrimen/otto"
)

type Jeth struct {
	ethApi *EthereumApi
	toVal  func(interface{}) otto.Value
	re     *jsre.JSRE
}

func NewJeth(ethApi *EthereumApi, toVal func(interface{}) otto.Value, re *jsre.JSRE) *Jeth {
	return &Jeth{ethApi, toVal, re}
}

func (self *Jeth) err(code int, msg string, id interface{}) (response otto.Value) {
	rpcerr := &RpcErrorObject{code, msg}
	self.re.Set("ret_jsonrpc", jsonrpcver)
	self.re.Set("ret_id", id)
	self.re.Set("ret_error", rpcerr)
	response, _ = self.re.Run(`
		ret_response = { jsonrpc: ret_jsonrpc, id: ret_id, error: ret_error };
	`)
	return
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
		fmt.Printf("error: %s\n", err)
		return self.err(-32603, err.Error(), req.Id)
	}
	self.re.Set("ret_jsonrpc", jsonrpcver)
	self.re.Set("ret_id", req.Id)

	res, _ := json.Marshal(respif)
	self.re.Set("ret_result", string(res))
	response, err = self.re.Run(`
		ret_response = { jsonrpc: ret_jsonrpc, id: ret_id, result: JSON.parse(ret_result) };
	`)
	return
}
