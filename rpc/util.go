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
	"fmt"
	"io"
	"math/big"
	"net/http"
	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/xeth"
)

var rpclogger = logger.NewLogger("RPC")

type JsonWrapper struct{}

// Unmarshal state is a helper method which has the ability to decode messsages
// that use the `defaultBlock` (https://github.com/ethereum/wiki/wiki/JSON-RPC#the-default-block-parameter)
// For example a `call`: [{to: "0x....", data:"0x..."}, "latest"]. The first argument is the transaction
// message and the second one refers to the block height (or state) to which to apply this `call`.
func UnmarshalRawMessages(b []byte, iface interface{}, number *int64) (err error) {
	var data []json.RawMessage
	if err = json.Unmarshal(b, &data); err != nil && len(data) == 0 {
		return errDecodeArgs
	}

	// Number index determines the index in the array for a possible block number
	numberIndex := 0

	value := reflect.ValueOf(iface)
	rvalue := reflect.Indirect(value)

	switch rvalue.Kind() {
	case reflect.Slice:
		// This is a bit of a cheat, but `data` is expected to be larger than 2 if iface is a slice
		if number != nil {
			numberIndex = len(data) - 1
		} else {
			numberIndex = len(data)
		}

		slice := reflect.MakeSlice(rvalue.Type(), numberIndex, numberIndex)
		for i, raw := range data[0:numberIndex] {
			v := slice.Index(i).Interface()
			if err = json.Unmarshal(raw, &v); err != nil {
				fmt.Println(err, v)
				return err
			}
			slice.Index(i).Set(reflect.ValueOf(v))
		}
		reflect.Indirect(rvalue).Set(slice) //value.Set(slice)
	case reflect.Struct:
		fallthrough
	default:
		if err = json.Unmarshal(data[0], iface); err != nil {
			return errDecodeArgs
		}
		numberIndex = 1
	}

	// <0 index means out of bound for block number
	if numberIndex >= 0 && len(data) > numberIndex {
		if err = blockNumber(data[numberIndex], number); err != nil {
			return errDecodeArgs
		}
	}

	return nil
}

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
	hex := ethutil.Bytes2Hex(b)
	// Prefer output of "0x0" instead of "0x"
	if len(hex) == 0 {
		hex = "0"
	}
	return "0x" + hex
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

func i2hex(n int) string {
	return toHex(big.NewInt(int64(n)).Bytes())
}

type RpcServer interface {
	Start()
	Stop()
}

type Log struct {
	Address string   `json:"address"`
	Topic   []string `json:"topic"`
	Data    string   `json:"data"`
	Number  uint64   `json:"number"`
}

func toLogs(logs state.Logs) (ls []Log) {
	ls = make([]Log, len(logs))

	for i, log := range logs {
		var l Log
		l.Topic = make([]string, len(log.Topics()))
		l.Address = toHex(log.Address())
		l.Data = toHex(log.Data())
		l.Number = log.Number()
		for j, topic := range log.Topics() {
			l.Topic[j] = toHex(topic)
		}
		ls[i] = l
	}

	return
}

type whisperFilter struct {
	messages []xeth.WhisperMessage
	timeout  time.Time
	id       int
}

func (w *whisperFilter) add(msgs ...xeth.WhisperMessage) {
	w.messages = append(w.messages, msgs...)
}
func (w *whisperFilter) get() []xeth.WhisperMessage {
	w.timeout = time.Now()
	tmp := w.messages
	w.messages = nil
	return tmp
}

type logFilter struct {
	logs    state.Logs
	timeout time.Time
	id      int
}

func (l *logFilter) add(logs ...state.Log) {
	l.logs = append(l.logs, logs...)
}

func (l *logFilter) get() state.Logs {
	l.timeout = time.Now()
	tmp := l.logs
	l.logs = nil
	return tmp
}
