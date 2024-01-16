// Copyright 2023 The go-ethereum Authors
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

package sync

import (
	"testing"

	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
)

var (
	testServer1 = 1
	testServer2 = 2
	testServer3 = 3
	testServer4 = 4

	testHead0 = types.HeadInfo{}
	testHead1 = types.HeadInfo{Slot: 123, BlockRoot: common.Hash{1}}
	testHead2 = types.HeadInfo{Slot: 124, BlockRoot: common.Hash{2}}
	testHead3 = types.HeadInfo{Slot: 124, BlockRoot: common.Hash{3}}
	testHead4 = types.HeadInfo{Slot: 125, BlockRoot: common.Hash{4}}

	testSHead1 = types.SignedHeader{SignatureSlot: 0x0124, Header: types.Header{Slot: 0x0123, StateRoot: common.Hash{1}}}
	testSHead2 = types.SignedHeader{SignatureSlot: 0x2010, Header: types.Header{Slot: 0x200e, StateRoot: common.Hash{2}}}
	// testSHead3 is at the end of period 1 but signed in period 2
	testSHead3 = types.SignedHeader{SignatureSlot: 0x4000, Header: types.Header{Slot: 0x3fff, StateRoot: common.Hash{3}}}
	testSHead4 = types.SignedHeader{SignatureSlot: 0x6444, Header: types.Header{Slot: 0x6443, StateRoot: common.Hash{4}}}
)

func TestValidatedHead(t *testing.T) {
	chain := &TestCommitteeChain{}
	ht := &TestHeadTracker{}
	headSync := NewHeadSync(ht, chain)

	ht.ExpValidated(t, 1, nil)
	headSync.Process([]request.Event{
		{Server: testServer1, Type: request.EvRegistered},
		{Server: testServer1, Type: EvNewSignedHead, Data: testSHead1},
	})
	ht.ExpValidated(t, 2, nil)
	chain.SetNextSyncPeriod(0)
	headSync.Process(nil)
	ht.ExpValidated(t, 3, []types.SignedHeader{testSHead1})
	chain.SetNextSyncPeriod(1)
	headSync.Process([]request.Event{
		{Server: testServer1, Type: EvNewSignedHead, Data: testSHead2},
		{Server: testServer2, Type: request.EvRegistered},
		{Server: testServer2, Type: EvNewSignedHead, Data: testSHead2},
	})
	ht.ExpValidated(t, 4, []types.SignedHeader{testSHead2, testSHead2})
	headSync.Process([]request.Event{
		{Server: testServer1, Type: EvNewSignedHead, Data: testSHead3},
		{Server: testServer3, Type: request.EvRegistered},
		{Server: testServer3, Type: EvNewSignedHead, Data: testSHead4},
	})
	ht.ExpValidated(t, 5, nil)
	chain.SetNextSyncPeriod(2)
	headSync.Process(nil)
	ht.ExpValidated(t, 6, []types.SignedHeader{testSHead3})
	headSync.Process([]request.Event{
		{Server: testServer3, Type: request.EvUnregistered},
	})
	ht.ExpValidated(t, 7, nil)
	chain.SetNextSyncPeriod(3)
	headSync.Process(nil)
	ht.ExpValidated(t, 8, nil)
	headSync.Process([]request.Event{
		{Server: testServer2, Type: EvNewSignedHead, Data: testSHead4},
	})
	ht.ExpValidated(t, 9, []types.SignedHeader{testSHead4})
}

func TestPrefetchHead(t *testing.T) {
	chain := &TestCommitteeChain{}
	ht := &TestHeadTracker{}
	headSync := NewHeadSync(ht, chain)

	ht.ExpPrefetch(t, 1, testHead0) // no servers registered
	headSync.Process([]request.Event{
		{Server: testServer1, Type: request.EvRegistered},
		{Server: testServer1, Type: EvNewHead, Data: testHead1},
	})
	ht.ExpPrefetch(t, 2, testHead1) // s1: h1
	headSync.Process([]request.Event{
		{Server: testServer2, Type: request.EvRegistered},
		{Server: testServer2, Type: EvNewHead, Data: testHead2},
	})
	ht.ExpPrefetch(t, 3, testHead2) // s1: h1, s2: h2
	headSync.Process([]request.Event{
		{Server: testServer1, Type: EvNewHead, Data: testHead2},
	})
	ht.ExpPrefetch(t, 4, testHead2) // s1: h2, s2: h2
	headSync.Process([]request.Event{
		{Server: testServer3, Type: request.EvRegistered},
		{Server: testServer3, Type: EvNewHead, Data: testHead3},
	})
	ht.ExpPrefetch(t, 5, testHead2) // s1: h2, s2: h2, s3: h3
	headSync.Process([]request.Event{
		{Server: testServer4, Type: request.EvRegistered},
		{Server: testServer4, Type: EvNewHead, Data: testHead4},
	})
	ht.ExpPrefetch(t, 6, testHead2) // s1: h2, s2: h2, s3: h3, s4: h4
	headSync.Process([]request.Event{
		{Server: testServer2, Type: EvNewHead, Data: testHead3},
	})
	ht.ExpPrefetch(t, 7, testHead3) // s1: h2, s2: h3, s3: h3, s4: h4
	headSync.Process([]request.Event{
		{Server: testServer3, Type: request.EvUnregistered},
	})
	ht.ExpPrefetch(t, 8, testHead4) // s1: h2, s2: h3, s4: h4
	headSync.Process([]request.Event{
		{Server: testServer1, Type: request.EvUnregistered},
	})
	ht.ExpPrefetch(t, 9, testHead4) // s2: h3, s4: h4
	headSync.Process([]request.Event{
		{Server: testServer4, Type: request.EvUnregistered},
	})
	ht.ExpPrefetch(t, 10, testHead3) // s2: h3
	headSync.Process([]request.Event{
		{Server: testServer2, Type: request.EvUnregistered},
	})
	ht.ExpPrefetch(t, 11, testHead0) // no servers registered
}
