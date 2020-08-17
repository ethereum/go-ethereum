// Copyright 2020 The go-ethereum Authors
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

package rlpx

import (
	"bytes"
	"crypto/rand"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/sha3"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRLPX_ReadWrite(t *testing.T) {
	srv := startTestServer()
	defer srv.Close()

	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	rlpx := NewRLPX(conn)
	// make rw frame
	var (
		aesSecret      = make([]byte, 16)
		macSecret      = make([]byte, 16)
		egressMACinit  = make([]byte, 32)
		ingressMACinit = make([]byte, 32)
	)
	for _, s := range [][]byte{aesSecret, macSecret, egressMACinit, ingressMACinit} {
		rand.Read(s)
	}
	buf := new(bytes.Buffer)
	rlpx.RW = NewRLPXFrameRW(buf, aesSecret, macSecret, sha3.NewLegacyKeccak256(), sha3.NewLegacyKeccak256())

	// encode payload
	size, reader, err := rlp.EncodeToReader([]byte("success"))
	if err != nil {
		t.Fatal(err)
	}
	// create and write message
	rawMsg := RawRLPXMessage{
		Code:       1,
		Size:       uint32(size),
		Payload:    reader,
	}
	err = rlpx.Write(rawMsg)
	if err != nil {
		t.Fatal(err)
	}
	// read message
	msg, err := rlpx.Read()
	if err != nil {
		t.Fatal(err)
	}
	// decode payload
	decodedMsg := make([]byte, 7)
	if err := rlp.Decode(msg.Payload, &decodedMsg); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "success", string(decodedMsg))
}

func TestRLPX_Close(t *testing.T) {
	srv := startTestServer()
	defer srv.Close()

	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	rlpx := NewRLPX(conn)
	rlpx.Close()

	if _, err := rlpx.Conn.Write([]byte("failure")); err == nil {
		t.Fatal("connection was not successfully closed")
	}
}

type testHandler struct {}
func (t *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

func startTestServer() *httptest.Server {
	handler := &testHandler{}
	return httptest.NewServer(handler)
}
