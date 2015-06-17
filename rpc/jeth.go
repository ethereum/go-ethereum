package rpc

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/jsre"
	"github.com/ethereum/go-ethereum/rpc/api"
	"github.com/ethereum/go-ethereum/rpc/comms"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/robertkrimen/otto"
)

type Jeth struct {
	ethApi api.EthereumApi
	re     *jsre.JSRE
	client comms.EthereumClient
}

func NewJeth(ethApi api.EthereumApi, re *jsre.JSRE, client comms.EthereumClient) *Jeth {
	return &Jeth{ethApi, re, client}
}

func (self *Jeth) err(call otto.FunctionCall, code int, msg string, id interface{}) (response otto.Value) {
	rpcerr := &shared.ErrorObject{code, msg}
	call.Otto.Set("ret_jsonrpc", api.JsonRpcVersion)
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

	jsonreq, err := json.Marshal(reqif)
	var reqs []shared.Request
	batch := true
	err = json.Unmarshal(jsonreq, &reqs)
	if err != nil {
		reqs = make([]shared.Request, 1)
		err = json.Unmarshal(jsonreq, &reqs[0])
		batch = false
	}

	call.Otto.Set("response_len", len(reqs))
	call.Otto.Run("var ret_response = new Array(response_len);")

	for i, req := range reqs {
		var respif interface{}
		err := self.client.Send(&req)
		if err != nil {
			return self.err(call, -32603, err.Error(), req.Id)
		}
		respif, err = self.client.Recv()
		if err != nil {
			return self.err(call, -32603, err.Error(), req.Id)
		}

		call.Otto.Set("ret_jsonrpc", api.JsonRpcVersion)
		call.Otto.Set("ret_id", req.Id)

		res, _ := json.Marshal(respif)

		call.Otto.Set("ret_result", string(res))
		call.Otto.Set("response_idx", i)
		response, err = call.Otto.Run(`
		ret_response[response_idx] = { jsonrpc: ret_jsonrpc, id: ret_id, result: JSON.parse(ret_result) };
		`)
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
