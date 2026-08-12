// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/fjl/jsonw"
)

const (
	vsn                      = "2.0"
	serviceMethodSeparator   = "_"
	subscribeMethodSuffix    = "_subscribe"
	unsubscribeMethodSuffix  = "_unsubscribe"
	notificationMethodSuffix = "_subscription"
	maxMethodNameLength      = 2048

	defaultWriteTimeout = 10 * time.Second // used if context has no deadline
)

var null = json.RawMessage("null")

type subscriptionResult struct {
	ID     string          `json:"subscription"`
	Result json.RawMessage `json:"result,omitempty"`
}

type subscriptionResultEnc struct {
	ID     string `json:"subscription"`
	Result any    `json:"result"`
}

// A value of this type can a JSON-RPC request, notification, successful response or
// error response. Which one it is depends on the fields.
type jsonrpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

func (msg *jsonrpcMessage) isNotification() bool {
	return msg.hasValidVersion() && msg.ID == nil && msg.Method != ""
}

func (msg *jsonrpcMessage) isCall() bool {
	return msg.hasValidVersion() && msg.hasValidID() && msg.Method != ""
}

func (msg *jsonrpcMessage) isResponse() bool {
	return msg.hasValidVersion() && msg.hasValidID() && msg.Method == "" && msg.Params == nil && (msg.Result != nil || msg.Error != nil)
}

func (msg *jsonrpcMessage) hasValidID() bool {
	return len(msg.ID) > 0 && msg.ID[0] != '{' && msg.ID[0] != '['
}

func (msg *jsonrpcMessage) hasValidVersion() bool {
	return msg.Version == vsn
}

func (msg *jsonrpcMessage) isSubscribe() bool {
	return strings.HasSuffix(msg.Method, subscribeMethodSuffix)
}

func (msg *jsonrpcMessage) isUnsubscribe() bool {
	return strings.HasSuffix(msg.Method, unsubscribeMethodSuffix)
}

func (msg *jsonrpcMessage) namespace() string {
	before, _, _ := strings.Cut(msg.Method, serviceMethodSeparator)
	return before
}

func (msg *jsonrpcMessage) String() string {
	b, _ := json.Marshal(msg)
	return string(b)
}

func (msg *jsonrpcMessage) errorResponse(err error) *jsonrpcMessage {
	resp := errorMessage(err)
	resp.ID = msg.ID
	return resp
}

// decodeError decodes the Error field into a jsonError struct.
func (msg *jsonrpcMessage) decodeError() *jsonError {
	if msg.Error == nil {
		return nil
	}
	je := new(jsonError)
	json.Unmarshal(msg.Error, je)
	return je
}

func (msg *jsonrpcMessage) response(result interface{}) *jsonrpcMessage {
	var (
		enc []byte
		err error
	)
	// Call MarshalJSON directly for types that implement it. This avoids the
	// expensive validation/compaction pass that json.Marshal performs on
	// encoder output.
	if m, ok := result.(json.Marshaler); ok {
		enc, err = m.MarshalJSON()
	} else {
		enc, err = json.Marshal(result)
	}
	if err != nil {
		return msg.errorResponse(&internalServerError{errcodeMarshalError, err.Error()})
	}
	return &jsonrpcMessage{Version: vsn, ID: msg.ID, Result: enc}
}

func errorMessage(err error) *jsonrpcMessage {
	je := &jsonError{
		Code:    errcodeDefault,
		Message: err.Error(),
	}
	if ec, ok := err.(Error); ok {
		je.Code = ec.ErrorCode()
	}
	if de, ok := err.(DataError); ok {
		je.Data = de.ErrorData()
	}
	enc, _ := json.Marshal(je)
	return &jsonrpcMessage{Version: vsn, ID: null, Error: enc}
}

type jsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (err *jsonError) Error() string {
	if err.Message == "" {
		return fmt.Sprintf("json-rpc error %d", err.Code)
	}
	return err.Message
}

func (err *jsonError) ErrorCode() int {
	return err.Code
}

func (err *jsonError) ErrorData() interface{} {
	return err.Data
}

// Conn is a subset of the methods of net.Conn which are sufficient for ServerCodec.
type Conn interface {
	io.ReadWriteCloser
	SetWriteDeadline(time.Time) error
}

type deadlineCloser interface {
	io.Closer
	SetWriteDeadline(time.Time) error
}

// ConnRemoteAddr wraps the RemoteAddr operation, which returns a description
// of the peer address of a connection. If a Conn also implements ConnRemoteAddr, this
// description is used in log messages.
type ConnRemoteAddr interface {
	RemoteAddr() string
}

// jsonCodec reads and writes JSON-RPC messages to the underlying connection. It also has
// support for parsing arguments and serializing (result) objects.
type jsonCodec struct {
	remote      string
	closer      sync.Once        // close closed channel once
	closeCh     chan interface{} // closed on Close
	decode      decodeFunc       // decoder to allow multiple transports
	readFrame   readFrameFunc    // set when the transport delimits messages itself
	encMu       sync.Mutex       // guards the encoder
	encodeMsg   encodeMsgFunc    // single-message encoder
	encodeBatch encodeBatchFunc  // batch encoder
	conn        deadlineCloser
}

type encodeMsgFunc = func(ctx context.Context, msg *jsonrpcMessage, isError bool) error

type encodeBatchFunc = func(ctx context.Context, msgs []*jsonrpcMessage, isError bool) error

type decodeFunc = func(v interface{}) error

// readFrameFunc returns the bytes of the next message. Only transports that
// delimit messages themselves have one. The bytes must not be reused on the next
// call, the message points into them.
type readFrameFunc = func() ([]byte, error)

// NewFuncCodec creates a codec which uses the given functions to read and write. If conn
// implements ConnRemoteAddr, log messages will use it to include the remote address of
// the connection. The decode function must reject invalid JSON, reading a message
// relies on it.
func NewFuncCodec(conn deadlineCloser, encodeMsg encodeMsgFunc, encodeBatch encodeBatchFunc, decode decodeFunc) ServerCodec {
	return newFuncCodec(conn, encodeMsg, encodeBatch, decode, nil)
}

// newFuncCodec is NewFuncCodec with the frame reader the built in transports use.
// A transport with a frame reader never calls decode, so it may be nil.
func newFuncCodec(conn deadlineCloser, encodeMsg encodeMsgFunc, encodeBatch encodeBatchFunc, decode decodeFunc, readFrame readFrameFunc) *jsonCodec {
	codec := &jsonCodec{
		closeCh:     make(chan interface{}),
		encodeMsg:   encodeMsg,
		encodeBatch: encodeBatch,
		decode:      decode,
		readFrame:   readFrame,
		conn:        conn,
	}
	if ra, ok := conn.(ConnRemoteAddr); ok {
		codec.remote = ra.RemoteAddr()
	}
	return codec
}

// NewCodec creates a codec on the given connection. If conn implements ConnRemoteAddr, log
// messages will use it to include the remote address of the connection.
func NewCodec(conn Conn) ServerCodec {
	dec := json.NewDecoder(conn)
	dec.UseNumber()
	var buf []byte
	encodeMsg := func(ctx context.Context, msg *jsonrpcMessage, isError bool) error {
		buf = appendMessage(buf[:0], msg)
		buf = append(buf, '\n')
		_, err := conn.Write(buf)
		return err
	}
	encodeBatch := func(ctx context.Context, msgs []*jsonrpcMessage, isError bool) error {
		buf = appendBatch(buf[:0], msgs)
		buf = append(buf, '\n')
		_, err := conn.Write(buf)
		return err
	}
	return NewFuncCodec(conn, encodeMsg, encodeBatch, dec.Decode)
}

// appendMessage appends the JSON-RPC encoding of msg to buf.
func appendMessage(buf []byte, msg *jsonrpcMessage) []byte {
	buf = append(buf, `{"jsonrpc":"2.0"`...)
	if msg.ID != nil {
		buf = append(buf, `,"id":`...)
		buf = append(buf, msg.ID...)
	}
	if msg.Method != "" {
		buf = append(buf, `,"method":`...)
		buf = jsonw.AppendQuotedString(buf, msg.Method)
	}
	if msg.Params != nil {
		buf = append(buf, `,"params":`...)
		buf = append(buf, msg.Params...)
	}
	if msg.Error != nil {
		buf = append(buf, `,"error":`...)
		buf = append(buf, msg.Error...)
	}
	if msg.Result != nil {
		buf = append(buf, `,"result":`...)
		buf = append(buf, msg.Result...)
	}
	buf = append(buf, '}')
	return buf
}

// appendBatch appends the JSON-RPC encoding of a message batch to buf.
func appendBatch(buf []byte, msgs []*jsonrpcMessage) []byte {
	buf = append(buf, '[')
	for i, msg := range msgs {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = appendMessage(buf, msg)
	}
	buf = append(buf, ']')
	return buf
}

func (c *jsonCodec) peerInfo() PeerInfo {
	// This returns "ipc" because all other built-in transports have a separate codec type.
	return PeerInfo{Transport: "ipc", RemoteAddr: c.remote}
}

func (c *jsonCodec) remoteAddr() string {
	return c.remote
}

func (c *jsonCodec) readBatch() (messages []*jsonrpcMessage, batch bool, err error) {
	rawmsg, err := c.readMessage()
	if err != nil {
		return nil, false, err
	}
	messages, batch = parseMessage(rawmsg)
	for i, msg := range messages {
		if msg == nil {
			// Message is JSON 'null'. Replace with zero value so it
			// will be treated like any other invalid message.
			messages[i] = new(jsonrpcMessage)
		}
	}
	return messages, batch, nil
}

// readMessage returns the bytes of the next message, checked to be valid JSON.
func (c *jsonCodec) readMessage() (json.RawMessage, error) {
	// A stream has no framing, so the decoder finds the message end and checks it.
	if c.readFrame == nil {
		// Decode the next JSON object in the input stream.
		// This verifies basic syntax, etc.
		var rawmsg json.RawMessage
		if err := c.decode(&rawmsg); err != nil {
			return nil, err
		}
		return rawmsg, nil
	}
	// The transport delimits the message, so one read and one check will do.
	// Decoding into a json.RawMessage would scan twice and copy.
	frame, err := c.readFrame()
	if err != nil {
		return nil, err
	}
	if !json.Valid(frame) {
		// Decode the broken message to report where it went wrong. Unmarshal
		// checks syntax the same way Valid does, so it fails here too. The
		// fallback only guards against the two ever disagreeing.
		var rawmsg json.RawMessage
		err := json.Unmarshal(frame, &rawmsg)
		if err == nil {
			err = errors.New("invalid JSON request")
		}
		return nil, err
	}
	return frame, nil
}

func (c *jsonCodec) writeJSON(ctx context.Context, msg *jsonrpcMessage, isError bool) error {
	c.encMu.Lock()
	defer c.encMu.Unlock()

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(defaultWriteTimeout)
	}
	c.conn.SetWriteDeadline(deadline)
	return c.encodeMsg(ctx, msg, isError)
}

func (c *jsonCodec) writeJSONBatch(ctx context.Context, msgs []*jsonrpcMessage, isError bool) error {
	c.encMu.Lock()
	defer c.encMu.Unlock()

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(defaultWriteTimeout)
	}
	c.conn.SetWriteDeadline(deadline)
	return c.encodeBatch(ctx, msgs, isError)
}

func (c *jsonCodec) close() {
	c.closer.Do(func() {
		close(c.closeCh)
		c.conn.Close()
	})
}

// closed returns a channel which will be closed when Close is called
func (c *jsonCodec) closed() <-chan interface{} {
	return c.closeCh
}

// parseMessage parses raw bytes as a (batch of) JSON-RPC message(s). There are no error
// checks in this function because the raw message has already been syntax-checked when it
// is called. Any non-JSON-RPC messages in the input return the zero value of
// jsonrpcMessage.
func parseMessage(raw json.RawMessage) ([]*jsonrpcMessage, bool) {
	if !isBatch(raw) {
		// readBatch rejects a nil message, which is what null must become.
		if isJSONNull(raw) {
			return []*jsonrpcMessage{nil}, false
		}
		msgs := []*jsonrpcMessage{{}}
		fillMessage(raw, msgs[0])
		return msgs, false
	}
	var msgs []*jsonrpcMessage
	forEachJSONElement(raw, func(elem []byte) {
		// readBatch rejects a nil message, which is what null must become.
		if isJSONNull(elem) {
			msgs = append(msgs, nil)
			return
		}
		msg := new(jsonrpcMessage)
		fillMessage(elem, msg)
		msgs = append(msgs, msg)
	})
	return msgs, true
}

// fillMessage picks a message apart into msg. Input that does not hold an object
// leaves msg zero, and the handler rejects it later.
func fillMessage(input []byte, msg *jsonrpcMessage) {
	// The raw fields point into input rather than being copied out of it, which
	// matters because params is nearly all of a large request.
	redo := false
	forEachJSONField(input, func(key, value []byte) {
		switch string(key) {
		case "jsonrpc":
			// The string fields go through encoding/json to unescape them.
			json.Unmarshal(value, &msg.Version)
		case "id":
			msg.ID = value
		case "method":
			json.Unmarshal(value, &msg.Method)
		case "params":
			msg.Params = value
		case "error":
			msg.Error = value
		case "result":
			msg.Result = value
		default:
			// encoding/json matched field names case insensitively and
			// unescaped them, so an unknown key may still name a field.
			// Redo the message with it to keep that behavior.
			redo = true
		}
	})
	if redo {
		*msg = jsonrpcMessage{}
		json.Unmarshal(input, msg)
	}
}

// isBatch returns true when the first non-whitespace characters is '['
func isBatch(raw json.RawMessage) bool {
	for _, c := range raw {
		// skip insignificant whitespace (http://www.ietf.org/rfc/rfc4627.txt)
		if c == 0x20 || c == 0x09 || c == 0x0a || c == 0x0d {
			continue
		}
		return c == '['
	}
	return false
}

// parsePositionalArguments tries to parse the given args to an array of values with the
// given types. It returns the parsed values or an error when the args could not be
// parsed. Missing optional arguments are returned as reflect.Zero values.
func parsePositionalArguments(rawArgs json.RawMessage, types []reflect.Type) ([]reflect.Value, error) {
	var args []reflect.Value
	switch {
	case len(bytes.TrimSpace(rawArgs)) == 0 || isJSONNull(rawArgs):
		// "params" is optional and may be empty. Also allow "params":null even though it's
		// not in the spec because our own client used to send it.
	case isBatch(rawArgs):
		// Read argument array.
		var err error
		if args, err = parseArgumentArray(rawArgs, types); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("non-array args")
	}
	// Set any missing args to nil.
	for i := len(args); i < len(types); i++ {
		if types[i].Kind() != reflect.Pointer {
			return nil, fmt.Errorf("missing value for required argument %d", i)
		}
		args = append(args, reflect.Zero(types[i]))
	}
	return args, nil
}

// parseArgumentArray decodes an already syntax-checked argument array.
func parseArgumentArray(rawArgs json.RawMessage, types []reflect.Type) ([]reflect.Value, error) {
	// Cutting the array into elements first means each argument is decoded once.
	// A json.Decoder would walk every argument twice, once to find where it ends.
	args := make([]reflect.Value, 0, len(types))
	var scanErr error
	forEachJSONElement(rawArgs, func(elem []byte) {
		if scanErr != nil {
			return
		}
		i := len(args)
		if i >= len(types) {
			scanErr = fmt.Errorf("too many arguments, want at most %d", len(types))
			return
		}
		argval := reflect.New(types[i])
		if err := decodeArgument(elem, argval.Interface()); err != nil {
			scanErr = fmt.Errorf("invalid argument %d: %v", i, err)
			return
		}
		if argval.IsNil() && types[i].Kind() != reflect.Pointer {
			scanErr = fmt.Errorf("missing value for required argument %d", i)
			return
		}
		args = append(args, argval.Elem())
	})
	return args, scanErr
}

// decodeArgument decodes one already syntax-checked argument value.
func decodeArgument(elem []byte, arg any) error {
	// A type that unmarshals itself is called directly, which skips the
	// validation pass json.Unmarshal runs first.
	if u, ok := arg.(json.Unmarshaler); ok && !isJSONNull(elem) {
		return u.UnmarshalJSON(elem)
	}
	return json.Unmarshal(elem, arg)
}

// parseSubscriptionName extracts the subscription name from an encoded argument array.
func parseSubscriptionName(rawArgs json.RawMessage) (string, error) {
	dec := json.NewDecoder(bytes.NewReader(rawArgs))
	if tok, _ := dec.Token(); tok != json.Delim('[') {
		return "", errors.New("non-array args")
	}
	v, _ := dec.Token()
	method, ok := v.(string)
	if !ok {
		return "", errors.New("expected subscription name as first argument")
	}
	return method, nil
}
