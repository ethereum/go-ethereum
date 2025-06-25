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
	testServer1 = testServer("testServer1")
	testServer2 = testServer("testServer2")
	testServer3 = testServer("testServer3")
	testServer4 = testServer("testServer4")
	testServer5 = testServer("testServer5")

	testHead0 = types.HeadInfo{}
	testHead1 = types.HeadInfo{Slot: 123, BlockRoot: common.Hash{1}}
	testHead2 = types.HeadInfo{Slot: 124, BlockRoot: common.Hash{2}}
	testHead3 = types.HeadInfo{Slot: 124, BlockRoot: common.Hash{3}}
	testHead4 = types.HeadInfo{Slot: 125, BlockRoot: common.Hash{4}}

	testOptUpdate1 = types.OptimisticUpdate{SignatureSlot: 0x0124, Attested: types.HeaderWithExecProof{Header: types.Header{Slot: 0x0123, StateRoot: common.Hash{1}}}}
	testOptUpdate2 = types.OptimisticUpdate{SignatureSlot: 0x2010, Attested: types.HeaderWithExecProof{Header: types.Header{Slot: 0x200e, StateRoot: common.Hash{2}}}}
	// testOptUpdate3 is at the end of period 1 but signed in period 2
	testOptUpdate3 = types.OptimisticUpdate{SignatureSlot: 0x4000, Attested: types.HeaderWithExecProof{Header: types.Header{Slot: 0x3fff, StateRoot: common.Hash{3}}}}
	testOptUpdate4 = types.OptimisticUpdate{SignatureSlot: 0x6444, Attested: types.HeaderWithExecProof{Header: types.Header{Slot: 0x6443, StateRoot: common.Hash{4}}}}
)

func finality(opt types.OptimisticUpdate) types.FinalityUpdate {
	return types.FinalityUpdate{
		SignatureSlot: opt.SignatureSlot,
		Attested:      opt.Attested,
		Finalized:     types.HeaderWithExecProof{Header: types.Header{Slot: (opt.Attested.Header.Slot - 64) & uint64(0xffffffffffffffe0)}},
	}
}

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
	ts.ServerEvent(EvNewOptimisticUpdate, testServer1, testOptUpdate1)
	ts.Run(1, testServer1, ReqFinality{})
	// announced head should be queued because of uninitialized chain
	ht.ExpValidated(t, 1, nil)

	chain.SetNextSyncPeriod(0) // initialize chain
	ts.Run(2)
	// expect previously queued head to be validated
	ht.ExpValidated(t, 2, []types.OptimisticUpdate{testOptUpdate1})

	chain.SetNextSyncPeriod(1)
	ts.ServerEvent(EvNewFinalityUpdate, testServer1, finality(testOptUpdate2))
	ts.ServerEvent(EvNewOptimisticUpdate, testServer1, testOptUpdate2)
	ts.AddServer(testServer2, 1)
	ts.ServerEvent(EvNewOptimisticUpdate, testServer2, testOptUpdate2)
	ts.Run(3)
	// expect both head announcements to be validated instantly
	ht.ExpValidated(t, 3, []types.OptimisticUpdate{testOptUpdate2, testOptUpdate2})

	ts.ServerEvent(EvNewOptimisticUpdate, testServer1, testOptUpdate3)
	ts.AddServer(testServer3, 1)
	ts.ServerEvent(EvNewOptimisticUpdate, testServer3, testOptUpdate4)
	// finality should be requested from both servers
	ts.Run(4, testServer1, ReqFinality{}, testServer3, ReqFinality{})
	// future period announced heads should be queued
	ht.ExpValidated(t, 4, nil)

	chain.SetNextSyncPeriod(2)
	ts.Run(5)
	// testOptUpdate3 can be validated now but not testOptUpdate4
	ht.ExpValidated(t, 5, []types.OptimisticUpdate{testOptUpdate3})

	ts.AddServer(testServer4, 1)
	ts.ServerEvent(EvNewOptimisticUpdate, testServer4, testOptUpdate3)
	// new server joined with recent optimistic update but still no finality; should be requested
	ts.Run(6, testServer4, ReqFinality{})
	ht.ExpValidated(t, 6, []types.OptimisticUpdate{testOptUpdate3})

	ts.AddServer(testServer5, 1)
	ts.RequestEvent(request.EvResponse, ts.Request(6, 1), finality(testOptUpdate3))
	ts.ServerEvent(EvNewOptimisticUpdate, testServer5, testOptUpdate3)
	// finality update request answered; new server should not be requested
	ts.Run(7)
	ht.ExpValidated(t, 7, []types.OptimisticUpdate{testOptUpdate3})

	// server 3 disconnected without proving period 3, its announced head should be dropped
	ts.RemoveServer(testServer3)
	ts.Run(8)
	ht.ExpValidated(t, 8, nil)

	chain.SetNextSyncPeriod(3)
	ts.Run(9)
	// testOptUpdate4 could be validated now but it's not queued by any registered server
	ht.ExpValidated(t, 9, nil)

	ts.ServerEvent(EvNewFinalityUpdate, testServer2, finality(testOptUpdate4))
	ts.ServerEvent(EvNewOptimisticUpdate, testServer2, testOptUpdate4)
	ts.Run(10)
	// now testOptUpdate4 should be validated
	ht.ExpValidated(t, 10, []types.OptimisticUpdate{testOptUpdate4})
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
