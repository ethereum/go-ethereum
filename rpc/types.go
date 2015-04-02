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
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type hexdata struct {
	data  []byte
	isNil bool
}

func (d *hexdata) String() string {
	return "0x" + common.Bytes2Hex(d.data)
}

func (d *hexdata) MarshalJSON() ([]byte, error) {
	if d.isNil {
		return json.Marshal(nil)
	}
	return json.Marshal(d.String())
}

func (d *hexdata) UnmarshalJSON(b []byte) (err error) {
	d.data = common.FromHex(string(b))
	return nil
}

func newHexData(input interface{}) *hexdata {
	d := new(hexdata)

	if input == nil {
		d.data = nil
		return d
	}
	switch input := input.(type) {
	case []byte:
		d.data = input
	case common.Hash:
		d.data = input.Bytes()
	case *common.Hash:
		if input == nil {
			d.isNil = true
		} else {
			d.data = input.Bytes()
		}
	case common.Address:
		d.data = input.Bytes()
	case *common.Address:
		if input == nil {
			d.isNil = true
		} else {
			d.data = input.Bytes()
		}
	case types.Bloom:
		d.data = input.Bytes()
	case *types.Bloom:
		if input == nil {
			d.isNil = true
		} else {
			d.data = input.Bytes()
		}
	case *big.Int:
		d.data = input.Bytes()
	case int64:
		d.data = big.NewInt(input).Bytes()
	case uint64:
		buff := make([]byte, 8)
		binary.BigEndian.PutUint64(buff, input)
		d.data = buff
	case int:
		d.data = big.NewInt(int64(input)).Bytes()
	case uint:
		d.data = big.NewInt(int64(input)).Bytes()
	case int8:
		d.data = big.NewInt(int64(input)).Bytes()
	case uint8:
		d.data = big.NewInt(int64(input)).Bytes()
	case int16:
		d.data = big.NewInt(int64(input)).Bytes()
	case uint16:
		d.data = big.NewInt(int64(input)).Bytes()
	case int32:
		d.data = big.NewInt(int64(input)).Bytes()
	case uint32:
		d.data = big.NewInt(int64(input)).Bytes()
	case string: // hexstring
		d.data = common.Big(input).Bytes()
	default:
		d.data = nil
	}

	return d
}

type hexnum struct {
	data  []byte
	isNil bool
}

func (d *hexnum) String() string {
	// Get hex string from bytes
	out := common.Bytes2Hex(d.data)
	// Trim leading 0s
	out = strings.TrimLeft(out, "0")
	// Output "0x0" when value is 0
	if len(out) == 0 {
		out = "0"
	}
	return "0x" + out
}

func (d *hexnum) MarshalJSON() ([]byte, error) {
	if d.isNil {
		return json.Marshal(nil)
	}
	return json.Marshal(d.String())
}

func (d *hexnum) UnmarshalJSON(b []byte) (err error) {
	d.data = common.FromHex(string(b))
	return nil
}

func newHexNum(input interface{}) *hexnum {
	d := new(hexnum)

	d.data = newHexData(input).data

	return d
}

type RpcConfig struct {
	ListenAddress string
	ListenPort    uint
	CorsDomain    string
}

type InvalidTypeError struct {
	method string
	msg    string
}

func (e *InvalidTypeError) Error() string {
	return fmt.Sprintf("invalid type on field %s: %s", e.method, e.msg)
}

func NewInvalidTypeError(method, msg string) *InvalidTypeError {
	return &InvalidTypeError{
		method: method,
		msg:    msg,
	}
}

type InsufficientParamsError struct {
	have int
	want int
}

func (e *InsufficientParamsError) Error() string {
	return fmt.Sprintf("insufficient params, want %d have %d", e.want, e.have)
}

func NewInsufficientParamsError(have int, want int) *InsufficientParamsError {
	return &InsufficientParamsError{
		have: have,
		want: want,
	}
}

type NotImplementedError struct {
	Method string
}

func (e *NotImplementedError) Error() string {
	return fmt.Sprintf("%s method not implemented", e.Method)
}

func NewNotImplementedError(method string) *NotImplementedError {
	return &NotImplementedError{
		Method: method,
	}
}

type DecodeParamError struct {
	err string
}

func (e *DecodeParamError) Error() string {
	return fmt.Sprintf("could not decode, %s", e.err)

}

func NewDecodeParamError(errstr string) error {
	return &DecodeParamError{
		err: errstr,
	}
}

type ValidationError struct {
	ParamName string
	msg       string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s not valid, %s", e.ParamName, e.msg)
}

func NewValidationError(param string, msg string) error {
	return &ValidationError{
		ParamName: param,
		msg:       msg,
	}
}

type RpcRequest struct {
	Id      interface{}     `json:"id"`
	Jsonrpc string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type RpcSuccessResponse struct {
	Id      interface{} `json:"id"`
	Jsonrpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
}

type RpcErrorResponse struct {
	Id      interface{}     `json:"id"`
	Jsonrpc string          `json:"jsonrpc"`
	Error   *RpcErrorObject `json:"error"`
}

type RpcErrorObject struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	// Data    interface{} `json:"data"`
}
