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
	"encoding/json"
	"io"
	"net/http"

	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/state"
)

var rpclogger = logger.NewLogger("RPC")

type JsonWrapper struct{}

func (self JsonWrapper) Send(writer io.Writer, v interface{}) (n int, err error) {
	var payload []byte
	payload, err = json.Marshal(v)
	if err != nil {
		rpclogger.Fatalln("Error marshalling JSON", err)
		return 0, err
	}
	rpclogger.DebugDetailf("Sending payload: %s", payload)

	return writer.Write(payload)
}

func (self JsonWrapper) ParseRequestBody(req *http.Request) (RpcRequest, error) {
	var reqParsed RpcRequest

	// Convert JSON to native types
	d := json.NewDecoder(req.Body)
	defer req.Body.Close()
	err := d.Decode(&reqParsed)

	if err != nil {
		rpclogger.Errorln("Error decoding JSON: ", err)
		return reqParsed, err
	}

	rpclogger.DebugDetailf("Parsed request: %s", reqParsed)

	return reqParsed, nil
}

func toHex(b []byte) string {
	return "0x" + ethutil.Bytes2Hex(b)
}
func fromHex(s string) []byte {
	if len(s) > 1 {
		if s[0:2] == "0x" {
			s = s[2:]
		}
		return ethutil.Hex2Bytes(s)
	}
	return nil
}

type RpcServer interface {
	Start()
	Stop()
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
