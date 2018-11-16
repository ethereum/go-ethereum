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

	"github.com/ethereum/go-ethereum/log"
)

func TestReporter(t *testing.T) {
	dir := os.TempDir()
	defer os.RemoveAll(dir)

	log.Debug("Setting up metrics first time")
	reportInterval := 100 * time.Millisecond
	db := SetupAccountingMetrics(reportInterval, dir+"/test.db")
	log.Debug("Done.")

	mBalanceCredit.Inc(12)
	mBytesCredit.Inc(34)
	mMsgDebit.Inc(9)

	//give the reporter time to write to DB
	time.Sleep(500 * time.Millisecond)

	mBalanceCredit = nil
	mBytesCredit = nil
	mMsgDebit = nil
	db.Close()

	log.Debug("Setting up metrics second time")
	SetupAccountingMetrics(reportInterval, dir+"/test.db")
	log.Debug("Done.")

	if mBalanceCredit.Count() != 12 {
		t.Fatalf("Expected counter to be %d, but is %d", 12, mBalanceCredit.Count())
	}
	if mBytesCredit.Count() != 34 {
		t.Fatalf("Expected counter to be %d, but is %d", 23, mBytesCredit.Count())
	}
	if mMsgDebit.Count() != 9 {
		t.Fatalf("Expected counter to be %d, but is %d", 9, mMsgDebit.Count())
	}
}
