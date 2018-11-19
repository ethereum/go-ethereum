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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

//TestReporter tests that the metrics being collected for p2p accounting
//are being persisted and available after restart of a node.
//It simulates restarting by just recreating the DB as if the node had restarted.
func TestReporter(t *testing.T) {
	//create a test directory
	dir, err := ioutil.TempDir("", "reporter-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	//setup the metrics
	log.Debug("Setting up metrics first time")
	reportInterval := 5 * time.Millisecond
	metrics := SetupAccountingMetrics(reportInterval, filepath.Join(dir, "test.db"))
	log.Debug("Done.")

	//do some metrics
	mBalanceCredit.Inc(12)
	mBytesCredit.Inc(34)
	mMsgDebit.Inc(9)

	//give the reporter time to write the metrics to DB
	time.Sleep(20 * time.Millisecond)

	//set the metrics to nil - this effectively simulates the node having shut down...
	mBalanceCredit = nil
	mBytesCredit = nil
	mMsgDebit = nil
	//close the DB also, or we can't create a new one
	metrics.Close()

	//setup the metrics again
	log.Debug("Setting up metrics second time")
	metrics = SetupAccountingMetrics(reportInterval, filepath.Join(dir, "test.db"))
	defer metrics.Close()
	log.Debug("Done.")

	//now check the metrics, they should have the same value as before "shutdown"
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
