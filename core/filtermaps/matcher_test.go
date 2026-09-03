// Copyright 2024 The go-ethereum Authors
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

package filtermaps

import (
	"context"
	crand "crypto/rand"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestMatcher(t *testing.T) {
	ts := newTestSetup(t)
	defer ts.close()

	ts.chain.addBlocks(100, 10, 10, 4, true)
	ts.setHistory(0, false)
	ts.fm.WaitIdle()

	for i := 0; i < 2000; i++ {
		bhash := ts.chain.canonical[rand.Intn(len(ts.chain.canonical))]
		receipts := ts.chain.receipts[bhash]
		if len(receipts) == 0 {
			continue
		}
		receipt := receipts[rand.Intn(len(receipts))]
		if len(receipt.Logs) == 0 {
			continue
		}
		log := receipt.Logs[rand.Intn(len(receipt.Logs))]
		var ok bool
		addresses := make([]common.Address, rand.Intn(3))
		for i := range addresses {
			crand.Read(addresses[i][:])
		}
		if len(addresses) > 0 {
			addresses[rand.Intn(len(addresses))] = log.Address
			ok = true
		}
		topics := make([][]common.Hash, rand.Intn(len(log.Topics)+1))
		for j := range topics {
			topics[j] = make([]common.Hash, rand.Intn(3))
			for i := range topics[j] {
				crand.Read(topics[j][i][:])
			}
			if len(topics[j]) > 0 {
				topics[j][rand.Intn(len(topics[j]))] = log.Topics[j]
				ok = true
			}
		}
		if !ok {
			continue // cannot search for match-all pattern
		}
		mb := ts.fm.NewMatcherBackend()
		logs, err := GetPotentialMatches(context.Background(), mb, 0, 1000, addresses, topics)
		mb.Close()
		if err != nil {
			t.Fatalf("Log search error: %v", err)
		}
		var found bool
		for _, l := range logs {
			if l == log {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Log search did not return expected log (addresses: %v, topics: %v, expected log: %v)", addresses, topics, *log)
		}
	}
}
