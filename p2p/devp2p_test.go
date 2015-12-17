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

package p2p

import (
	"errors"
	"net"
	"reflect"
	"sync"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

func TestProtocolHandshake(t *testing.T) {
	var (
		prv0, prv1 = newkey(), newkey()
		fd0, fd1   = net.Pipe()
		hs0        = &protoHandshake{Version: 3, ID: discover.PubkeyID(&prv0.PublicKey), Caps: []Cap{{"a", 0}, {"b", 2}}}
		hs1        = &protoHandshake{Version: 3, ID: discover.PubkeyID(&prv1.PublicKey), Caps: []Cap{{"c", 1}, {"d", 3}}}
		wg         sync.WaitGroup
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		conn := newDevConn(fd0, prv0, nil)

		phs, err := conn.doProtoHandshake(hs0)
		if err != nil {
			t.Errorf("dial side proto handshake error: %v", err)
			return
		}
		if !reflect.DeepEqual(phs, hs1) {
			t.Errorf("dial side proto handshake mismatch:\ngot: %s\nwant: %s\n", spew.Sdump(phs), spew.Sdump(hs1))
			return
		}
		conn.close(DiscQuitting)
	}()
	go func() {
		defer wg.Done()
		conn := newDevConn(fd1, prv1, &prv0.PublicKey)

		phs, err := conn.doProtoHandshake(hs1)
		if err != nil {
			t.Errorf("listen side proto handshake error: %v", err)
			return
		}
		if !reflect.DeepEqual(phs, hs0) {
			t.Errorf("listen side proto handshake mismatch:\ngot: %s\nwant: %s\n", spew.Sdump(phs), spew.Sdump(hs0))
			return
		}

		if err := ExpectMsg(conn.protocols[0], discMsg, []DiscReason{DiscQuitting}); err != nil {
			t.Errorf("error receiving disconnect: %v", err)
		}
	}()
	wg.Wait()
}

func TestProtocolHandshakeErrors(t *testing.T) {
	our := &protoHandshake{Version: 3, Caps: []Cap{{"foo", 2}, {"bar", 3}}, Name: "quux"}
	id := randomID()
	tests := []struct {
		code uint64
		msg  interface{}
		err  error
	}{
		{
			code: discMsg,
			msg:  []DiscReason{DiscQuitting},
			err:  DiscQuitting,
		},
		{
			code: 0x989898,
			msg:  []byte{1},
			err:  errors.New("expected handshake, got 989898"),
		},
		{
			code: handshakeMsg,
			msg:  make([]byte, baseProtocolMaxMsgSize+2),
			err:  errors.New("message too big"),
		},
		{
			code: handshakeMsg,
			msg:  []byte{1, 2, 3},
			err:  newPeerError(errInvalidMsg, "(code 0) (size 4) rlp: expected input list for p2p.protoHandshake"),
		},
		{
			code: handshakeMsg,
			msg:  &protoHandshake{Version: 9944, ID: id},
			err:  DiscIncompatibleVersion,
		},
		{
			code: handshakeMsg,
			msg:  &protoHandshake{Version: 3},
			err:  DiscInvalidIdentity,
		},
	}

	for i, test := range tests {
		p1, p2 := MsgPipe()
		go Send(p1, test.code, test.msg)
		_, err := readProtocolHandshake(p2, our)
		if !reflect.DeepEqual(err, test.err) {
			t.Errorf("test %d: error mismatch: got %q, want %q", i, err, test.err)
		}
		p1.Close()
	}
}
