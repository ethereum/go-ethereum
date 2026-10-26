// Copyright 2026 The go-ethereum Authors
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

package eip7807

import (
	"bytes"
	"math/rand"
	"strings"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/electra"
	"github.com/attestantio/go-eth2-client/spec/gloas"
	"github.com/ethereum/go-ethereum/common"
)

// erigonRequests is the single withdrawal request of the erigontech/erigon
// PR 23854 fixture: source 0x11..11, a zero pubkey, amount 8.
func erigonRequests() [][]byte {
	rec := append(append(bytes.Repeat([]byte{0x11}, 20), make([]byte, 48)...), 8, 0, 0, 0, 0, 0, 0, 0)
	return [][]byte{append([]byte{0x01}, rec...)}
}

// randomRequests builds one request per type up to maxType, with the given
// record counts, from a seeded source.
func randomRequests(seed int64, counts []int) [][]byte {
	rng := rand.New(rand.NewSource(seed))
	var requests [][]byte
	for typ, count := range counts {
		req := []byte{byte(typ)}
		size := 0
		for _, n := range requestFields[typ] {
			size += n
		}
		for i := 0; i < count; i++ {
			rec := make([]byte, size)
			rng.Read(rec)
			req = append(req, rec...)
		}
		requests = append(requests, req)
	}
	return requests
}

// oracleRoots computes both container roots with go-eth2-client, an
// independent SSZ implementation already in the module.
func oracleRoots(t *testing.T, requests [][]byte) (electraRoot, gloasRoot common.Hash) {
	t.Helper()
	lists, err := splitRequests(requests)
	if err != nil {
		t.Fatal(err)
	}
	var e electra.ExecutionRequests
	var g gloas.ExecutionRequests
	for _, rec := range lists[0] {
		d := new(electra.DepositRequest)
		if err := d.UnmarshalSSZ(rec); err != nil {
			t.Fatal(err)
		}
		e.Deposits = append(e.Deposits, d)
		g.Deposits = append(g.Deposits, d)
	}
	for _, rec := range lists[1] {
		w := new(electra.WithdrawalRequest)
		if err := w.UnmarshalSSZ(rec); err != nil {
			t.Fatal(err)
		}
		e.Withdrawals = append(e.Withdrawals, w)
		g.Withdrawals = append(g.Withdrawals, w)
	}
	for _, rec := range lists[2] {
		c := new(electra.ConsolidationRequest)
		if err := c.UnmarshalSSZ(rec); err != nil {
			t.Fatal(err)
		}
		e.Consolidations = append(e.Consolidations, c)
		g.Consolidations = append(g.Consolidations, c)
	}
	for _, rec := range lists[3] {
		d := new(gloas.BuilderDepositRequest)
		if err := d.UnmarshalSSZ(rec); err != nil {
			t.Fatal(err)
		}
		g.BuilderDeposits = append(g.BuilderDeposits, d)
	}
	for _, rec := range lists[4] {
		x := new(gloas.BuilderExitRequest)
		if err := x.UnmarshalSSZ(rec); err != nil {
			t.Fatal(err)
		}
		g.BuilderExits = append(g.BuilderExits, x)
	}
	er, err := e.HashTreeRoot()
	if err != nil {
		t.Fatal(err)
	}
	gr, err := g.HashTreeRoot()
	if err != nil {
		t.Fatal(err)
	}
	return er, gr
}

// TestRequestsRootErigon pins the Electra root of the Erigon fixture's
// request, which Erigon checked against remerkleable, and the Gloas root of
// the same request.
func TestRequestsRootErigon(t *testing.T) {
	root, err := RequestsRoot(erigonRequests())
	if err != nil {
		t.Fatal(err)
	}
	if want := common.HexToHash("0xb1c4c413201f39ef0e4528edbd42e595a5258578a22e8634a9dee79684fdf245"); root != want {
		t.Errorf("Electra root %x, want %x", root, want)
	}
	lists, _ := splitRequests(erigonRequests())
	if got, want := gloasRequestsRoot(lists), common.HexToHash("0x9a5200aeb0991e2619450f00e87550eb56faf87637d778645d85fc505e4a9205"); got != want {
		t.Errorf("Gloas root %x, want %x", got, want)
	}
	if e, g := oracleRoots(t, erigonRequests()); e != root || g != gloasRequestsRoot(lists) {
		t.Errorf("oracle Electra %x Gloas %x", e, g)
	}
}

// TestRequestsRootOracle compares both containers with go-eth2-client over
// request sets of every shape: empty, one type, every type, and the Electra
// limits.
func TestRequestsRootOracle(t *testing.T) {
	for i, counts := range [][]int{
		{},
		{1},
		{0, 1},
		{0, 0, 1},
		{3, 2, 1},
		{8192, 16, 2},
		{1, 1, 1, 1, 1},
		{0, 0, 0, 2, 3},
		{5, 16, 2, 7, 1},
	} {
		requests := randomRequests(int64(i), counts)
		lists, err := splitRequests(requests)
		if err != nil {
			t.Fatal(err)
		}
		wantE, wantG := oracleRoots(t, requests)
		if len(counts) <= 3 {
			if got, err := electraRequestsRoot(lists); err != nil || got != wantE {
				t.Errorf("case %d: Electra root %x, %v, oracle %x", i, got, err, wantE)
			}
		}
		if got := gloasRequestsRoot(lists); got != wantG {
			t.Errorf("case %d: Gloas root %x, oracle %x", i, got, wantG)
		}
	}
}

// TestRequestsRootRejects covers the shapes RequestsRoot refuses.
func TestRequestsRootRejects(t *testing.T) {
	withdrawal := erigonRequests()[0]
	deposit := randomRequests(1, []int{1})[0]
	for _, c := range []struct {
		name     string
		requests [][]byte
		err      string
	}{
		{"unknown type", [][]byte{append([]byte{0x05}, 1, 2, 3)}, "unknown request type 5"},
		{"out of order", [][]byte{withdrawal, deposit}, "out of order"},
		{"duplicate type", [][]byte{withdrawal, withdrawal}, "out of order"},
		{"partial record", [][]byte{withdrawal[:len(withdrawal)-1]}, "not a multiple"},
		{"builder request", randomRequests(2, []int{0, 0, 0, 1}), "no field in the Electra"},
		{"over the withdrawal limit", randomRequests(3, []int{0, 17}), "exceed the limit 16"},
	} {
		if _, err := RequestsRoot(c.requests); err == nil || !strings.Contains(err.Error(), c.err) {
			t.Errorf("%s: error %v, want %q", c.name, err, c.err)
		}
	}
	// A request carrying only its type byte is skipped, as CalcRequestsHash skips it.
	with, err := RequestsRoot([][]byte{{0x00}, withdrawal, {0x02}})
	if err != nil {
		t.Fatal(err)
	}
	if without, _ := RequestsRoot([][]byte{withdrawal}); with != without {
		t.Errorf("empty requests change the root: %x vs %x", with, without)
	}
}
