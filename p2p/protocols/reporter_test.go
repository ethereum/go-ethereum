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
	reportInterval := 2 * time.Millisecond
	metrics := SetupAccountingMetrics(reportInterval, filepath.Join(dir, "test.db"))
	log.Debug("Done.")

	//change metrics
	mBalanceCredit.Inc(12)
	mBytesCredit.Inc(34)
	mMsgDebit.Inc(9)

	//store expected metrics
	expectedBalanceCredit := mBalanceCredit.Count()
	expectedBytesCredit := mBytesCredit.Count()
	expectedMsgDebit := mMsgDebit.Count()

	//give the reporter time to write the metrics to DB
	time.Sleep(20 * time.Millisecond)

	//close the DB also, or we can't create a new one
	metrics.Close()

	//clear the metrics - this effectively simulates the node having shut down...
	mBalanceCredit.Clear()
	mBytesCredit.Clear()
	mMsgDebit.Clear()

	//setup the metrics again
	log.Debug("Setting up metrics second time")
	metrics = SetupAccountingMetrics(reportInterval, filepath.Join(dir, "test.db"))
	defer metrics.Close()
	log.Debug("Done.")

	//now check the metrics, they should have the same value as before "shutdown"
	if mBalanceCredit.Count() != expectedBalanceCredit {
		t.Fatalf("Expected counter to be %d, but is %d", expectedBalanceCredit, mBalanceCredit.Count())
	}
	if mBytesCredit.Count() != expectedBytesCredit {
		t.Fatalf("Expected counter to be %d, but is %d", expectedBytesCredit, mBytesCredit.Count())
	}
	if mMsgDebit.Count() != expectedMsgDebit {
		t.Fatalf("Expected counter to be %d, but is %d", expectedMsgDebit, mMsgDebit.Count())
	}
}
