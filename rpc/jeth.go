package rpc

import (
	"encoding/json"
	"fmt"

	"reflect"

	"github.com/ethereum/go-ethereum/jsre"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/comms"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/robertkrimen/otto"
	"github.com/ethereum/go-ethereum/rpc/comms"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"reflect"
)

type Jeth struct {
	ethApi  *EthereumApi
	re      *jsre.JSRE
	ipcpath string
}

func NewJeth(ethApi *EthereumApi, re *jsre.JSRE, ipcpath string) *Jeth {
	return &Jeth{ethApi, re, ipcpath}
}

func (self *Jeth) err(call otto.FunctionCall, code int, msg string, id interface{}) (response otto.Value) {
	rpcerr := &RpcErrorObject{code, msg}
	call.Otto.Set("ret_jsonrpc", jsonrpcver)
	call.Otto.Set("ret_id", id)
	call.Otto.Set("ret_error", rpcerr)
	response, _ = call.Otto.Run(`
		ret_response = { jsonrpc: ret_jsonrpc, id: ret_id, error: ret_error };
	`)
	return
}

func (self *Jeth) Send(call otto.FunctionCall) (response otto.Value) {
	reqif, err := call.Argument(0).Export()
	if err != nil {
		return self.err(call, -32700, err.Error(), nil)
	}

	client, err := comms.NewIpcClient(comms.IpcConfig{self.ipcpath}, codec.JSON)
	if err != nil {
		fmt.Println("Unable to connect to geth.")
		return self.err(call, -32603, err.Error(), -1)
	}
	defer client.Close()

	jsonreq, err := json.Marshal(reqif)
	var reqs []RpcRequest
	batch := true
	err = json.Unmarshal(jsonreq, &reqs)
	if err != nil {
		reqs = make([]RpcRequest, 1)
		err = json.Unmarshal(jsonreq, &reqs[0])
		batch = false
	}

	call.Otto.Set("response_len", len(reqs))
	call.Otto.Run("var ret_response = new Array(response_len);")

	for i, req := range reqs {
		err := client.Send(&req)
		if err != nil {
			fmt.Println("Error send request:", err)
			return self.err(call, -32603, err.Error(), req.Id)
		}

		respif, err := client.Recv()
		if err != nil {
			fmt.Println("Error recv response:", err)
			return self.err(call, -32603, err.Error(), req.Id)
		}

		if res, ok := respif.(shared.SuccessResponse); ok {
			call.Otto.Set("ret_id", res.Id)
			call.Otto.Set("ret_jsonrpc", res.Jsonrpc)
			resObj, _ := json.Marshal(res.Result)
			call.Otto.Set("ret_result", string(resObj))
			call.Otto.Set("response_idx", i)

			response, err = call.Otto.Run(`
				ret_response[response_idx] = { jsonrpc: ret_jsonrpc, id: ret_id, result: JSON.parse(ret_result) };
			`)
		} else if res, ok := respif.(shared.ErrorResponse); ok {
			fmt.Printf("Error: %s (%d)\n", res.Error.Message, res.Error.Code)

			call.Otto.Set("ret_id", res.Id)
			call.Otto.Set("ret_jsonrpc", res.Jsonrpc)
			call.Otto.Set("ret_error", res.Error)
			call.Otto.Set("response_idx", i)

			response, _ = call.Otto.Run(`
				ret_response = { jsonrpc: ret_jsonrpc, id: ret_id, error: ret_error };
			`)
			return
		} else {
			fmt.Printf("unexpected response\n", reflect.TypeOf(respif))
		}
	}

	if !batch {
		call.Otto.Run("ret_response = ret_response[0];")
	}

	if call.Argument(1).IsObject() {
		call.Otto.Set("callback", call.Argument(1))
		call.Otto.Run(`
	    if (Object.prototype.toString.call(callback) == '[object Function]') {
			callback(null, ret_response);
		}
		`)
	}

	return
}

func (self *Jeth) SendIpc(call otto.FunctionCall) (response otto.Value) {
	reqif, err := call.Argument(0).Export()
	if err != nil {
		return self.err(call, -32700, err.Error(), nil)
	}

	client, err := comms.NewIpcClient(comms.IpcConfig{self.ipcpath}, codec.JSON)
	if err != nil {
		fmt.Println("Unable to connect to geth.")
		return self.err(call, -32603, err.Error(), -1)
	}
	defer client.Close()

	jsonreq, err := json.Marshal(reqif)
	var reqs []RpcRequest
	batch := true
	err = json.Unmarshal(jsonreq, &reqs)
	if err != nil {
		reqs = make([]RpcRequest, 1)
		err = json.Unmarshal(jsonreq, &reqs[0])
		batch = false
	}

	call.Otto.Set("response_len", len(reqs))
	call.Otto.Run("var ret_response = new Array(response_len);")

	for i, req := range reqs {
		err := client.Send(&req)
		if err != nil {
			fmt.Println("Error send request:", err)
			return self.err(call, -32603, err.Error(), req.Id)
		}

		respif, err := client.Recv()
		if err != nil {
			fmt.Println("Error recv response:", err)
			return self.err(call, -32603, err.Error(), req.Id)
		}

		if res, ok := respif.(shared.SuccessResponse); ok {
			call.Otto.Set("ret_id", res.Id)
			call.Otto.Set("ret_jsonrpc", res.Jsonrpc)
			resObj, _ := json.Marshal(res.Result)
			call.Otto.Set("ret_result", string(resObj))
			call.Otto.Set("response_idx", i)

			response, err = call.Otto.Run(`
				ret_response[response_idx] = { jsonrpc: ret_jsonrpc, id: ret_id, result: JSON.parse(ret_result) };
			`)
		} else if res, ok := respif.(shared.ErrorResponse); ok {
			fmt.Printf("Error: %s (%d)\n", res.Error.Message, res.Error.Code)

			call.Otto.Set("ret_id", res.Id)
			call.Otto.Set("ret_jsonrpc", res.Jsonrpc)
			call.Otto.Set("ret_error", res.Error)
			call.Otto.Set("response_idx", i)

			response, _ = call.Otto.Run(`
				ret_response = { jsonrpc: ret_jsonrpc, id: ret_id, error: ret_error };
			`)
			return
		} else {
			fmt.Printf("unexpected response\n", reflect.TypeOf(respif))
		}
	}

	if !batch {
		call.Otto.Run("ret_response = ret_response[0];")
	}

	if call.Argument(1).IsObject() {
		call.Otto.Set("callback", call.Argument(1))
		call.Otto.Run(`
	    if (Object.prototype.toString.call(callback) == '[object Function]') {
			callback(null, ret_response);
		}
		`)
	}

	return
}
