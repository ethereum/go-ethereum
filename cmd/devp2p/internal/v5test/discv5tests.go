// Copyright 2020 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package v5test

import (
	"bytes"

	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/p2p/discover/v5wire"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
)

// Suite is the discv5 test suite.
type Suite struct {
	Dest             *enode.Node
	Listen1, Listen2 string // listening addresses
}

func (s *Suite) listen() *testenv {
	return newTestEnv(s.Dest, s.Listen1, s.Listen2)
}

func (s *Suite) AllTests() []utesting.Test {
	return []utesting.Test{
		{Name: "Ping", Fn: s.TestPing},
		{Name: "TalkRequest", Fn: s.TestTalkRequest},
		{Name: "FindnodeZeroDistance", Fn: s.TestFindnodeZeroDistance},
	}
}

// This test sends PING and expects a PONG response.
func (s *Suite) TestPing(t *utesting.T) {
	te := s.listen()
	defer te.close()

	id := te.nextReqID()
	resp := te.reqresp(te.l1, &v5wire.Ping{ReqID: id})
	switch resp := resp.(type) {
	case *v5wire.Pong:
		if !bytes.Equal(resp.ReqID, id) {
			t.Fatalf("wrong request ID %x in PONG, want %x", resp.ReqID, id)
		}
		if !resp.ToIP.Equal(laddr(te.l1).IP) {
			t.Fatalf("wrong destination IP %v in PONG, want %v", resp.ToIP, laddr(te.l1).IP)
		}
		if int(resp.ToPort) != laddr(te.l1).Port {
			t.Fatalf("wrong destination port %v in PONG, want %v", resp.ToPort, laddr(te.l1).Port)
		}
	default:
		t.Fatal("expected PONG, got", resp.Name())
	}
}

// This test sends TALKREQ and expects an empty TALKRESP response.
func (s *Suite) TestTalkRequest(t *utesting.T) {
	te := s.listen()
	defer te.close()

	// Non-empty request ID.
	id := te.nextReqID()
	resp := te.reqresp(te.l1, &v5wire.TalkRequest{ReqID: id, Protocol: "test-protocol"})
	switch resp := resp.(type) {
	case *v5wire.TalkResponse:
		if !bytes.Equal(resp.ReqID, id) {
			t.Fatalf("wrong request ID %x in TALKRESP, want %x", resp.ReqID, id)
		}
		if len(resp.Message) > 0 {
			t.Fatalf("non-empty message %x in TALKRESP", resp.Message)
		}
	default:
		t.Fatal("expected TALKRESP, got", resp.Name())
	}

	// Empty request ID.
	resp = te.reqresp(te.l1, &v5wire.TalkRequest{Protocol: "test-protocol"})
	switch resp := resp.(type) {
	case *v5wire.TalkResponse:
		if len(resp.ReqID) > 0 {
			t.Fatalf("wrong request ID %x in TALKRESP, want empty byte array", resp.ReqID)
		}
		if len(resp.Message) > 0 {
			t.Fatalf("non-empty message %x in TALKRESP", resp.Message)
		}
	default:
		t.Fatal("expected TALKRESP, got", resp.Name())
	}
}

// This test checks that the remote node returns itself for FINDNODE with distance zero.
func (s *Suite) TestFindnodeZeroDistance(t *utesting.T) {
	te := s.listen()
	defer te.close()

	id := te.nextReqID()
	resp := te.reqresp(te.l1, &v5wire.Findnode{ReqID: id, Distances: []uint{0}})
	switch resp := resp.(type) {
	case *v5wire.Nodes:
		if !bytes.Equal(resp.ReqID, id) {
			t.Fatalf("wrong request ID %x in NODES, want %x", resp.ReqID, id)
		}
		if len(resp.Nodes) != 1 {
			t.Error("invalid number of entries in NODES response")
		}
		nodes, err := checkRecords(resp.Nodes)
		if err != nil {
			t.Errorf("invalid node in NODES response: %v", err)
		}
		if nodes[0].ID() != te.remote.ID() {
			t.Errorf("ID of response node is %v, want %v", nodes[0].ID(), te.remote.ID())
		}
	default:
		t.Fatal("expected NODES, got", resp.Name())
	}
}

func checkRecords(records []*enr.Record) ([]*enode.Node, error) {
	nodes := make([]*enode.Node, len(records))
	for i := range records {
		n, err := enode.New(enode.ValidSchemes, records[i])
		if err != nil {
			return nil, err
		}
		nodes[i] = n
	}
	return nodes, nil
}
