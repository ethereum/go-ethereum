// Copyright 2018 The go-ethereum Authors
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

package protocols

import (
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/rcrowley/go-metrics"
)

func TestReporter(t *testing.T) {
	dir := os.TempDir()
	defer os.RemoveAll(dir)

	stateStore, err := state.NewDBStore(dir + "/test.db")
	if err != nil {
		return
	}

	rep := &reporter{
		reg:        metrics.NewRegistry(),
		interval:   time.Millisecond,
		stateStore: stateStore,
	}
	go rep.run()
	time.Sleep(1 * time.Second)
	mBalanceCredit.Inc(12)
	mBytesCredit.Inc(34)
	mMsgDebit.Inc(9)

	rep = nil
	stateStore.Close()
	stateStore, err = state.NewDBStore(dir + "/test.db")
	if err != nil {
		return
	}
	rep = &reporter{
		reg:        metrics.NewRegistry(),
		interval:   time.Millisecond,
		stateStore: stateStore,
	}
	go rep.run()
	time.Sleep(1 * time.Second)
	mBalanceCredit.Inc(11)
	mBytesCredit.Inc(22)
	mMsgDebit.Inc(7)

	if mBalanceCredit.Count() != 23 {
		t.Fatalf("Expected counter to be %d, but is %d", 23, mBalanceCredit.Count())
	}
	if mBytesCredit.Count() != 56 {
		t.Fatalf("Expected counter to be %d, but is %d", 23, mBytesCredit.Count())
	}
	if mMsgDebit.Count() != 16 {
		t.Fatalf("Expected counter to be %d, but is %d", 23, mMsgDebit.Count())
	}
}
