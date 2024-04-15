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

	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
)

var (
	testServer1 = testServer("testServer1")
	testServer2 = testServer("testServer2")
	testServer3 = testServer("testServer3")
	testServer4 = testServer("testServer4")

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

type testServer string

func (t testServer) Name() string {
	return string(t)
}

func TestValidatedHead(t *testing.T) {
	chain := &TestCommitteeChain{}
	ht := &TestHeadTracker{}
	headSync := NewHeadSync(ht, chain)
	ts := NewTestScheduler(t, headSync)

	ht.ExpValidated(t, 0, nil)

	ts.AddServer(testServer1, 1)
	ts.ServerEvent(EvNewSignedHead, testServer1, testSHead1)
	ts.Run(1)
	// announced head should be queued because of uninitialized chain
	ht.ExpValidated(t, 1, nil)

	chain.SetNextSyncPeriod(0) // initialize chain
	ts.Run(2)
	// expect previously queued head to be validated
	ht.ExpValidated(t, 2, []types.SignedHeader{testSHead1})

	chain.SetNextSyncPeriod(1)
	ts.ServerEvent(EvNewSignedHead, testServer1, testSHead2)
	ts.AddServer(testServer2, 1)
	ts.ServerEvent(EvNewSignedHead, testServer2, testSHead2)
	ts.Run(3)
	// expect both head announcements to be validated instantly
	ht.ExpValidated(t, 3, []types.SignedHeader{testSHead2, testSHead2})

	ts.ServerEvent(EvNewSignedHead, testServer1, testSHead3)
	ts.AddServer(testServer3, 1)
	ts.ServerEvent(EvNewSignedHead, testServer3, testSHead4)
	ts.Run(4)
	// future period announced heads should be queued
	ht.ExpValidated(t, 4, nil)

	chain.SetNextSyncPeriod(2)
	ts.Run(5)
	// testSHead3 can be validated now but not testSHead4
	ht.ExpValidated(t, 5, []types.SignedHeader{testSHead3})

	// server 3 disconnected without proving period 3, its announced head should be dropped
	ts.RemoveServer(testServer3)
	ts.Run(6)
	ht.ExpValidated(t, 6, nil)

	chain.SetNextSyncPeriod(3)
	ts.Run(7)
	// testSHead4 could be validated now but it's not queued by any registered server
	ht.ExpValidated(t, 7, nil)

	ts.ServerEvent(EvNewSignedHead, testServer2, testSHead4)
	ts.Run(8)
	// now testSHead4 should be validated
	ht.ExpValidated(t, 8, []types.SignedHeader{testSHead4})
}

func TestPrefetchHead(t *testing.T) {
	chain := &TestCommitteeChain{}
	ht := &TestHeadTracker{}
	headSync := NewHeadSync(ht, chain)
	ts := NewTestScheduler(t, headSync)

	ht.ExpPrefetch(t, 0, testHead0) // no servers registered

	ts.AddServer(testServer1, 1)
	ts.ServerEvent(EvNewHead, testServer1, testHead1)
	ts.Run(1)
	ht.ExpPrefetch(t, 1, testHead1) // s1: h1

	ts.AddServer(testServer2, 1)
	ts.ServerEvent(EvNewHead, testServer2, testHead2)
	ts.Run(2)
	ht.ExpPrefetch(t, 2, testHead2) // s1: h1, s2: h2

	ts.ServerEvent(EvNewHead, testServer1, testHead2)
	ts.Run(3)
	ht.ExpPrefetch(t, 3, testHead2) // s1: h2, s2: h2

	ts.AddServer(testServer3, 1)
	ts.ServerEvent(EvNewHead, testServer3, testHead3)
	ts.Run(4)
	ht.ExpPrefetch(t, 4, testHead2) // s1: h2, s2: h2, s3: h3

	ts.AddServer(testServer4, 1)
	ts.ServerEvent(EvNewHead, testServer4, testHead4)
	ts.Run(5)
	ht.ExpPrefetch(t, 5, testHead2) // s1: h2, s2: h2, s3: h3, s4: h4

	ts.ServerEvent(EvNewHead, testServer2, testHead3)
	ts.Run(6)
	ht.ExpPrefetch(t, 6, testHead3) // s1: h2, s2: h3, s3: h3, s4: h4

	ts.RemoveServer(testServer3)
	ts.Run(7)
	ht.ExpPrefetch(t, 7, testHead4) // s1: h2, s2: h3, s4: h4

	ts.RemoveServer(testServer1)
	ts.Run(8)
	ht.ExpPrefetch(t, 8, testHead4) // s2: h3, s4: h4

	ts.RemoveServer(testServer4)
	ts.Run(9)
	ht.ExpPrefetch(t, 9, testHead3) // s2: h3

	ts.RemoveServer(testServer2)
	ts.Run(10)
	ht.ExpPrefetch(t, 10, testHead0) // no servers registered
}
