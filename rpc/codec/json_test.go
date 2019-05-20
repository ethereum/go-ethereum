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

package codec

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"
)

type jsonTestConn struct {
	buffer *bytes.Buffer
}

func newJsonTestConn(data []byte) *jsonTestConn {
	return &jsonTestConn{
		buffer: bytes.NewBuffer(data),
	}
}

func (self *jsonTestConn) Read(p []byte) (n int, err error) {
	return self.buffer.Read(p)
}

func (self *jsonTestConn) Write(p []byte) (n int, err error) {
	return self.buffer.Write(p)
}

func (self *jsonTestConn) Close() error {
	// not implemented
	return nil
}

func (self *jsonTestConn) LocalAddr() net.Addr {
	// not implemented
	return nil
}

func (self *jsonTestConn) RemoteAddr() net.Addr {
	// not implemented
	return nil
}

func (self *jsonTestConn) SetDeadline(t time.Time) error {
	return nil
}

func (self *jsonTestConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (self *jsonTestConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func TestJsonDecoderWithValidRequest(t *testing.T) {
	reqdata := []byte(`{"jsonrpc":"2.0","method":"modules","params":[],"id":64}`)
	decoder := newJsonTestConn(reqdata)

	jsonDecoder := NewJsonCoder(decoder)
	requests, batch, err := jsonDecoder.ReadRequest()

	if err != nil {
		t.Errorf("Read valid request failed - %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("Expected to get a single request but got %d", len(requests))
	}

	if batch {
		t.Errorf("Got batch indication while expecting single request")
	}

	if requests[0].Id != float64(64) {
		t.Errorf("Expected req.Id == 64 but got %v", requests[0].Id)
	}

	if requests[0].Method != "modules" {
		t.Errorf("Expected req.Method == 'modules' got '%s'", requests[0].Method)
	}
}

func TestJsonDecoderWithValidBatchRequest(t *testing.T) {
	reqdata := []byte(`[{"jsonrpc":"2.0","method":"modules","params":[],"id":64},
		{"jsonrpc":"2.0","method":"modules","params":[],"id":64}]`)
	decoder := newJsonTestConn(reqdata)

	jsonDecoder := NewJsonCoder(decoder)
	requests, batch, err := jsonDecoder.ReadRequest()

	if err != nil {
		t.Errorf("Read valid batch request failed - %v", err)
	}

	if len(requests) != 2 {
		t.Errorf("Expected to get two requests but got %d", len(requests))
	}

	if !batch {
		t.Errorf("Got no batch indication while expecting batch request")
	}

	for i := 0; i < len(requests); i++ {
		if requests[i].Id != float64(64) {
			t.Errorf("Expected req.Id == 64 but got %v", requests[i].Id)
		}

		if requests[i].Method != "modules" {
			t.Errorf("Expected req.Method == 'modules' got '%s'", requests[i].Method)
		}
	}
}

func TestJsonDecoderWithInvalidIncompleteMessage(t *testing.T) {
	reqdata := []byte(`{"jsonrpc":"2.0","method":"modules","pa`)
	decoder := newJsonTestConn(reqdata)

	jsonDecoder := NewJsonCoder(decoder)
	requests, batch, err := jsonDecoder.ReadRequest()

	if err != io.ErrUnexpectedEOF {
		t.Errorf("Expected to read an incomplete request err but got %v", err)
	}

	// remaining message
	decoder.Write([]byte(`rams":[],"id:64"}`))
	requests, batch, err = jsonDecoder.ReadRequest()

	if err == nil {
		t.Errorf("Expected an error but got nil")
	}

	if len(requests) != 0 {
		t.Errorf("Expected to get no requests but got %d", len(requests))
	}

	if batch {
		t.Errorf("Got batch indication while expecting non batch")
	}
}
