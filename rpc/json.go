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
)

type jsonWrapper struct{}

func (self jsonWrapper) Send(writer io.Writer, v interface{}) (n int, err error) {
	var payload []byte
	payload, err = json.Marshal(v)
	if err != nil {
		jsonlogger.Fatalln("Error marshalling JSON", err)
		return 0, err
	}
	jsonlogger.Infof("Sending payload: %s", payload)

	return writer.Write(payload)
}

func (self jsonWrapper) ParseRequestBody(req *http.Request) (RpcRequest, error) {
	var reqParsed RpcRequest

	// Convert JSON to native types
	d := json.NewDecoder(req.Body)
	// d.UseNumber()
	defer req.Body.Close()
	err := d.Decode(&reqParsed)

	if err != nil {
		jsonlogger.Errorln("Error decoding JSON: ", err)
		return reqParsed, err
	}
	jsonlogger.DebugDetailf("Parsed request: %s", reqParsed)

	return reqParsed, nil
}

var JSON jsonWrapper
