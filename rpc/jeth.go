// Copyright 2015 The go-expanse Authors
// This file is part of the go-expanse library.
//
// The go-expanse library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-expanse library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-expanse library. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"encoding/json"

	"fmt"

	"github.com/expanse-project/go-expanse/jsre"
	"github.com/expanse-project/go-expanse/rpc/comms"
	"github.com/expanse-project/go-expanse/rpc/shared"
	"github.com/robertkrimen/otto"
)

type Jeth struct {
	ethApi shared.ExpanseApi
	re     *jsre.JSRE
	client comms.ExpanseClient
}

func NewJeth(ethApi shared.ExpanseApi, re *jsre.JSRE, client comms.ExpanseClient) *Jeth {
	return &Jeth{ethApi, re, client}
}

func (self *Jeth) err(call otto.FunctionCall, code int, msg string, id interface{}) (response otto.Value) {
	errObj := fmt.Sprintf("{\"message\": \"%s\", \"code\": %d}", msg, code)
	retResponse := fmt.Sprintf("ret_response = JSON.parse('{\"jsonrpc\": \"%s\", \"id\": %v, \"error\": %s}');", shared.JsonRpcVersion, id, errObj)

	call.Otto.Run("ret_error = " + errObj)
	res, _ := call.Otto.Run(retResponse)

	return res
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

		call.Otto.Set("ret_jsonrpc", shared.JsonRpcVersion)
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
