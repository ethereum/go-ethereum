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

package ecp

import (
	"bufio"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"strconv"
	"strings"
)

const (
	firstDelim    = '\r'
	lastDelim     = '\n'
	delim         = "\r\n"
	simpleStrMark = '+'
	errMark       = '-'
	intMark       = ':'
	bulkStrMark   = '$'
	arrayMark     = '*'

	methodSep    = "."
	successReply = "+OK\r\n"
)

type ValueType int

const (
	simpleStrValue ValueType = 1 << iota
	integerValue
	binaryValue
	arrayValue
	errorValue
	nullValue
)

// ServerCodec implements reading and writing RPC messages for the server side of a RPC session.
type ServerCodec interface {
	// Read the next request
	Read() (*request, error)
	// WriteResponse writes an array of values as a response to the client
	WriteResponse([]interface{}) error
	// WriteError writes an error to the client
	WriteError(error) error
	// Close the underlying data stream(s)
	Close() error
}

// ECPCodec implements the REdis Server Protocol (RESP).
// See for more information http://redis.io/topics/protocol.
type ECPCodec struct {
	rc io.ReadCloser
	r  *bufio.Reader
	wc io.WriteCloser
	w  *bufio.Writer
}

// Create a new Ethereum Client Protocol Codec. It reads incoming requests from the supplied
// reader and writes responses to the supplied writer.
func NewECPCodec(r io.ReadCloser, w io.WriteCloser) ServerCodec {
	return &ECPCodec{r, bufio.NewReader(r), w, bufio.NewWriter(w)}
}

type message struct {
	Kind ValueType
	Val  interface{}
}

// Close underlying reader and writer
func (c *ECPCodec) Close() error {
	if err := c.w.Flush(); err != nil {
		return err
	}

	c.rc.Close()
	c.wc.Close()

	return nil
}

// Read reads a new RPC request.
func (c *ECPCodec) Read() (*request, error) {
	msg, err := c.read()
	if err != nil {
		return nil, err
	}

	if msg.Kind == arrayValue {
		if values, ok := msg.Val.([]*message); ok {
			if len(values) > 0 {
				req := new(request)
				method := strings.Split(fmt.Sprintf("%s", values[0].Val), methodSep)
				if len(method) != 2 {
					return nil, &invalidRequestError{}
				}

				req.service, req.method = method[0], method[1]
				req.args = make([]interface{}, len(values)-1)
				for i := 1; i < len(values); i++ {
					req.args[i-1] = values[i].Val
				}

				return req, nil
			}
		}
	}

	return nil, &invalidRequestError{}
}

func (c *ECPCodec) read() (*message, error) {
	mark, err := c.r.ReadByte()
	if err != nil {
		return nil, err
	}

	switch mark {
	case simpleStrMark:
		return c.readSimpleStr()
	case bulkStrMark:
		return c.readBulkStr()
	case intMark:
		return c.readInteger()
	case arrayMark:
		return c.readArray()
	case errMark:
		return c.readError()
	}

	return nil, &invalidLeadInError{mark}
}

func (c *ECPCodec) readSimpleStr() (*message, error) {
	str, err := c.r.ReadBytes(firstDelim)
	if err != nil {
		return nil, err
	}

	if nl, err := c.r.ReadByte(); err != nil {
		return nil, err
	} else if nl != lastDelim {
		return nil, &unexpectedByteError{lastDelim, nl}
	}

	return &message{simpleStrValue, string(str[:len(str)-1])}, nil
}

func (c *ECPCodec) readError() (*message, error) {
	e, err := c.r.ReadBytes(firstDelim)
	if err != nil {
		return nil, err
	}

	if nl, err := c.r.ReadByte(); err != nil {
		return nil, err
	} else if nl != lastDelim {
		return nil, &unexpectedByteError{lastDelim, nl}
	}

	return &message{errorValue, string(e[:len(e)-1])}, nil
}

func (c *ECPCodec) readInteger() (*message, error) {
	i, err := c.r.ReadBytes(firstDelim)
	if err != nil {
		return nil, err
	}

	if nl, err := c.r.ReadByte(); err != nil {
		return nil, err
	} else if nl != lastDelim {
		return nil, &unexpectedByteError{lastDelim, nl}
	}

	val, err := strconv.ParseInt(string(i[:len(i)-1]), 10, 64)
	if err != nil {
		return nil, err
	}
	return &message{integerValue, val}, nil
}

func (c *ECPCodec) readBulkStr() (*message, error) {
	n, err := c.r.ReadBytes(firstDelim)
	if err != nil {
		return nil, err
	}

	nl, err := c.r.ReadByte()
	if err != nil {
		return nil, err
	} else if nl != lastDelim {
		return nil, &unexpectedByteError{lastDelim, nl}
	}

	cnt, err := strconv.ParseInt(string(n[:len(n)-1]), 10, 64)
	if err != nil {
		return nil, err
	}

	if cnt == -1 {
		return &message{nullValue, nil}, nil
	}
	if cnt < 0 {
		return nil, &invalidRequestError{}
	}

	val := make([]byte, cnt)
	tot := int64(0)
	for tot < cnt {
		n, err := c.r.Read(val[tot:])
		if err != nil {
			return nil, err
		}
		tot += int64(n)
	}

	if cr, err := c.r.ReadByte(); err != nil {
		return nil, err
	} else if cr != firstDelim {
		return nil, &unexpectedByteError{lastDelim, cr}
	}

	if nl, err := c.r.ReadByte(); err != nil {
		return nil, err
	} else if nl != lastDelim {
		return nil, &unexpectedByteError{lastDelim, nl}
	}

	return &message{binaryValue, val}, nil
}

func (c *ECPCodec) readArray() (*message, error) {
	n, err := c.r.ReadBytes(firstDelim)
	if err != nil {
		return nil, err
	}

	if nl, err := c.r.ReadByte(); err != nil {
		return nil, err
	} else if nl != lastDelim {
		return nil, &unexpectedByteError{lastDelim, nl}
	}

	nElem, err := strconv.ParseInt(string(n[:len(n)-1]), 10, 64)
	if err != nil {
		return nil, err
	}

	arr := make([]*message, nElem)

	for i := int64(0); i < nElem; i++ {
		if arr[i], err = c.read(); err != nil {
			return nil, err
		}
	}

	return &message{arrayValue, arr}, nil
}

// WriteResponse writes a values to the client. When values is nil the default
// success response is written.
func (c *ECPCodec) WriteResponse(values []interface{}) error {
	defer c.w.Flush()

	if values == nil {
		_, e := c.w.WriteString(successReply)
		return e
	}

	return c.writeArray(values)
}

// WriteError writes the supplied error to the client
func (c *ECPCodec) WriteError(err error) error {
	defer c.w.Flush()
	_, e := c.w.WriteString(fmt.Sprintf("%c%s%s", errMark, err, delim))
	return e
}

func (c *ECPCodec) writeSimpleString(str string) error {
	_, err := c.w.WriteString(fmt.Sprintf("%c%s%s", simpleStrMark, str, delim))
	return err
}

func (c *ECPCodec) writeInteger(i int64) error {
	_, err := c.w.WriteString(fmt.Sprintf("%c%d%s", intMark, i, delim))
	return err
}

func (c *ECPCodec) writeBinary(bin []byte) error {
	if bin == nil {
		_, err := c.w.WriteString(fmt.Sprintf("%c-1%s", bulkStrMark, delim))
		return err
	}

	if _, err := c.w.WriteString(fmt.Sprintf("%c%d%s", bulkStrMark, len(bin), delim)); err != nil {
		return err
	}

	if _, err := c.w.Write(bin); err != nil {
		return err
	}

	_, err := c.w.WriteString(delim)
	return err
}

func (c *ECPCodec) writeStruct(obj interface{}) error {
	val := reflect.ValueOf(obj)
	fields := make([]interface{}, val.NumField())
	cnt := 0
	for i := 0; i < val.NumField(); i++ {
		if val.Field(i).CanInterface() { // skip unexported fields
			fields[cnt] = val.Field(i).Interface()
			cnt++
		}
	}
	return c.writeArray(fields[:cnt])
}

func (c *ECPCodec) writeArray(arr []interface{}) (err error) {
	if arr == nil {
		_, err = c.w.WriteString(fmt.Sprintf("%c-1%s", arrayMark, delim))
		return
	}

	// array length prefix
	if _, err = c.w.WriteString(fmt.Sprintf("%c%d%s", arrayMark, len(arr), delim)); err != nil {
		return
	}

	for i := 0; i < len(arr) && err == nil; i++ {
		switch val := arr[i].(type) {
		case string:
			err = c.writeSimpleString(val)
		case *string:
			err = c.writeSimpleString(*val)
		case []byte:
			err = c.writeBinary(val)
		case big.Int:
			err = c.writeBinary(val.Bytes())
		case *big.Int:
			err = c.writeBinary(val.Bytes())
		case int64:
			err = c.writeInteger(val)
		case nil: // use nil bulk string to represent nil values
			err = c.writeBinary(nil)
		case error:
			err = c.WriteError(val)
		default:
			typ := reflect.TypeOf(val)
			if typ.Kind() == reflect.Struct {
				err = c.writeStruct(val)
			} else if a, ok := val.([]interface{}); ok {
				err = c.writeArray(a)
			} else {
				err = &unsupportedTypeError{typ.Name()}
			}
		}
	}

	return
}
