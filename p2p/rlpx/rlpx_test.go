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
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
	"io"
	"net"
	"testing"
)

type testRLPXMsg struct {
	code uint64
	size uint32
	payload io.Reader
	err error
}

func TestConn_Handshake(t *testing.T) {
	// make 2 peers
	conn1, conn2 := net.Pipe()
	// make key pairs
	key1 := newkey()
	key2 := newkey()

	peer1 := NewConn(conn1, &key2.PublicKey) // dialer
	peer2 := NewConn(conn2, nil) // listener

	doHandshake(t, peer1, peer2, key1, key2)
}

func TestConn_ReadWriteMsg(t *testing.T) {
	// make 2 peers
	conn1, conn2 := net.Pipe()
	// make key pairs
	key1 := newkey()
	key2 := newkey()

	peer1 := NewConn(conn1, &key2.PublicKey) // dialer
	peer2 := NewConn(conn2, nil) // listener

	doHandshake(t, peer1, peer2, key1, key2)

	msgChan := make(chan testRLPXMsg, 1)

	go func(msgChan chan testRLPXMsg) {
		var msg testRLPXMsg
		msg.code, msg.size, msg.payload, msg.err = peer1.ReadMsg()

		msgChan <- msg
	}(msgChan)

	size, payload, err := rlp.EncodeToReader([]byte("success"))
	if err != nil {
		t.Fatalf("could not rlp encode payload: %v", err)
	}

	if _, err := peer2.WriteMsg(0, uint32(size), payload); err != nil {
		t.Fatal(err)
	}

	msg := <- msgChan

	buf := make([]byte, 8)
	if _, err := msg.payload.Read(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "success", string(buf[1:8]))
}

func doHandshake(t *testing.T, peer1, peer2 *Conn, key1, key2 *ecdsa.PrivateKey) {
	keyChan := make(chan *ecdsa.PublicKey)

	go func(keyChan chan *ecdsa.PublicKey) {
		pubKey, err := peer2.Handshake(key2)
		if err != nil {
			t.Fatalf("peer2 could not do handshake: %v", err)
		}
		keyChan <- pubKey
	}(keyChan)

	pubKey2, err := peer1.Handshake(key1)
	if err != nil {
		t.Fatalf("peer1 could not do handshake: %v", err)
	}

	pubKey1 := <- keyChan

	// confirm successful handshake
	if !assert.Equal(t, pubKey1, &key1.PublicKey) || !assert.Equal(t, pubKey2, &key2.PublicKey) {
		t.Fatal("unsuccessful handshake")
	}
}

func newkey() *ecdsa.PrivateKey {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic("couldn't generate key: " + err.Error())
	}
	return key
}
