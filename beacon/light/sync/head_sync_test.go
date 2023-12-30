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

	testSHead1 = types.SignedHeader{Header: types.Header{Slot: 123, StateRoot: common.Hash{1}}}
)

func TestValidatedHead(t *testing.T) {
	tracker := &testTracker{}
	chain := &testCommitteeChain{}
	ht := &testHeadTracker{}
	headSync := NewHeadSync(ht, chain)

	ht.expValidated(t, 1, nil)
	headSync.Process(tracker, nil, []request.ServerEvent{
		{Server: testServer1, Type: request.EvRegistered},
		{Server: testServer1, Type: EvNewSignedHead, Data: testSHead1},
	})
	ht.expValidated(t, 2, nil)
	chain.init = true
	headSync.Process(tracker, nil, nil)
	ht.expValidated(t, 3, []types.SignedHeader{testSHead1})
}

func TestPrefetchHead(t *testing.T) {
	tracker := &testTracker{}
	chain := &testCommitteeChain{}
	ht := &testHeadTracker{}
	headSync := NewHeadSync(ht, chain)

	ht.expPrefetch(t, 1, testHead0) // no servers registered
	headSync.Process(tracker, nil, []request.ServerEvent{
		{Server: testServer1, Type: request.EvRegistered},
		{Server: testServer1, Type: EvNewHead, Data: testHead1},
	})
	ht.expPrefetch(t, 2, testHead1) // s1: h1
	headSync.Process(tracker, nil, []request.ServerEvent{
		{Server: testServer2, Type: request.EvRegistered},
		{Server: testServer2, Type: EvNewHead, Data: testHead2},
	})
	ht.expPrefetch(t, 3, testHead2) // s1: h1, s2: h2
	headSync.Process(tracker, nil, []request.ServerEvent{
		{Server: testServer1, Type: EvNewHead, Data: testHead2},
	})
	ht.expPrefetch(t, 4, testHead2) // s1: h2, s2: h2
	headSync.Process(tracker, nil, []request.ServerEvent{
		{Server: testServer3, Type: request.EvRegistered},
		{Server: testServer3, Type: EvNewHead, Data: testHead3},
	})
	ht.expPrefetch(t, 5, testHead2) // s1: h2, s2: h2, s3: h3
	headSync.Process(tracker, nil, []request.ServerEvent{
		{Server: testServer4, Type: request.EvRegistered},
		{Server: testServer4, Type: EvNewHead, Data: testHead4},
	})
	ht.expPrefetch(t, 6, testHead2) // s1: h2, s2: h2, s3: h3, s4: h4
	headSync.Process(tracker, nil, []request.ServerEvent{
		{Server: testServer2, Type: EvNewHead, Data: testHead3},
	})
	ht.expPrefetch(t, 7, testHead3) // s1: h2, s2: h3, s3: h3, s4: h4
	headSync.Process(tracker, nil, []request.ServerEvent{
		{Server: testServer3, Type: request.EvUnregistered},
	})
	ht.expPrefetch(t, 8, testHead4) // s1: h2, s2: h3, s4: h4
	headSync.Process(tracker, nil, []request.ServerEvent{
		{Server: testServer1, Type: request.EvUnregistered},
	})
	ht.expPrefetch(t, 9, testHead4) // s2: h3, s4: h4
	headSync.Process(tracker, nil, []request.ServerEvent{
		{Server: testServer4, Type: request.EvUnregistered},
	})
	ht.expPrefetch(t, 10, testHead3) // s2: h3
	headSync.Process(tracker, nil, []request.ServerEvent{
		{Server: testServer2, Type: request.EvUnregistered},
	})
	ht.expPrefetch(t, 11, testHead0) // no servers registered
}
