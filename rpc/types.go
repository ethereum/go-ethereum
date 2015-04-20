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

	"errors"
	"net"
	"net/http"
	"time"

	"io"

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

func newHexData(input interface{}) *hexdata {
	d := new(hexdata)

	if input == nil {
		d.isNil = true
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
		if input == nil {
			d.isNil = true
		} else {
			d.data = input.Bytes()
		}
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
		buff := make([]byte, 2)
		binary.BigEndian.PutUint16(buff, input)
		d.data = buff
	case int32:
		d.data = big.NewInt(int64(input)).Bytes()
	case uint32:
		buff := make([]byte, 4)
		binary.BigEndian.PutUint32(buff, input)
		d.data = buff
	case string: // hexstring
		d.data = common.Big(input).Bytes()
	default:
		d.isNil = true
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

type listenerHasStoppedError struct {
	msg string
}

func (self listenerHasStoppedError) Error() string {
	return self.msg
}

var listenerStoppedError = listenerHasStoppedError{"Listener stopped"}

// When https://github.com/golang/go/issues/4674 is fixed this could be replaced
type stoppableTCPListener struct {
	*net.TCPListener
	stop chan struct{} // closed when the listener must stop
}

// Wraps the default handler and checks if the RPC service was stopped. In that case it returns an
// error indicating that the service was stopped. This will only happen for connections which are
// kept open (HTTP keep-alive) when the RPC service was shutdown.
func newStoppableHandler(h http.Handler, stop chan struct{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-stop:
			w.Header().Set("Content-Type", "application/json")
			jsonerr := &RpcErrorObject{-32603, "RPC service stopped"}
			send(w, &RpcErrorResponse{Jsonrpc: jsonrpcver, Id: nil, Error: jsonerr})
		default:
			h.ServeHTTP(w, r)
		}
	})
}

// Stop the listener and all accepted and still active connections.
func (self *stoppableTCPListener) Stop() {
	close(self.stop)
}

func newStoppableTCPListener(addr string) (*stoppableTCPListener, error) {
	wl, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	if tcpl, ok := wl.(*net.TCPListener); ok {
		stop := make(chan struct{})
		l := &stoppableTCPListener{tcpl, stop}
		return l, nil
	}

	return nil, errors.New("Unable to create TCP listener for RPC service")
}

func (self *stoppableTCPListener) Accept() (net.Conn, error) {
	for {
		self.SetDeadline(time.Now().Add(time.Duration(1 * time.Second)))
		c, err := self.TCPListener.AcceptTCP()

		select {
		case <-self.stop:
			if c != nil { // accept timeout
				c.Close()
			}
			self.TCPListener.Close()
			return nil, listenerStoppedError
		default:
		}

		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() && netErr.Temporary() {
				continue // regular timeout
			}
		}

		return &closableConnection{c, self.stop}, err
	}
}

type closableConnection struct {
	*net.TCPConn
	closed chan struct{}
}

func (self *closableConnection) Read(b []byte) (n int, err error) {
	select {
	case <-self.closed:
		self.TCPConn.Close()
		return 0, io.EOF
	default:
		return self.TCPConn.Read(b)
	}
}
