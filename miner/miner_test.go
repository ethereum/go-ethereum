// Copyright 2020 The go-ethereum Authors
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

// Package miner implements Ethereum block creation and mining.
package miner

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/downloader"
)

func TestMiner(t *testing.T) {
	t.Parallel()

	minerBor := NewBorDefaultMiner(t)
	defer func() {
		minerBor.Cleanup(false)
		minerBor.Ctrl.Finish()
	}()

	miner := minerBor.Miner
	mux := minerBor.Mux

	miner.Start(common.HexToAddress("0x12345"))
	waitForMiningState(t, miner, true)

	// Start the downloader
	mux.Post(downloader.StartEvent{})
	waitForMiningState(t, miner, false)

	// Stop the downloader and wait for the update loop to run
	mux.Post(downloader.DoneEvent{})
	waitForMiningState(t, miner, true)

	// Subsequent downloader events after a successful DoneEvent should not cause the
	// miner to start or stop. This prevents a security vulnerability
	// that would allow entities to present fake high blocks that would
	// stop mining operations by causing a downloader sync
	// until it was discovered they were invalid, whereon mining would resume.
	mux.Post(downloader.StartEvent{})
	waitForMiningState(t, miner, true)

	mux.Post(downloader.FailedEvent{})
	waitForMiningState(t, miner, true)
}

// TestMinerDownloaderFirstFails tests that mining is only
// permitted to run indefinitely once the downloader sees a DoneEvent (success).
// An initial FailedEvent should allow mining to stop on a subsequent
// downloader StartEvent.
func TestMinerDownloaderFirstFails(t *testing.T) {
	t.Parallel()

	minerBor := NewBorDefaultMiner(t)
	defer func() {
		minerBor.Cleanup(false)
		minerBor.Ctrl.Finish()
	}()

	miner := minerBor.Miner
	mux := minerBor.Mux

	miner.Start(common.HexToAddress("0x12345"))
	waitForMiningState(t, miner, true)

	// Start the downloader
	mux.Post(downloader.StartEvent{})
	waitForMiningState(t, miner, false)

	// Stop the downloader and wait for the update loop to run
	mux.Post(downloader.FailedEvent{})
	waitForMiningState(t, miner, true)

	// Since the downloader hasn't yet emitted a successful DoneEvent,
	// we expect the miner to stop on next StartEvent.
	mux.Post(downloader.StartEvent{})
	waitForMiningState(t, miner, false)

	// Downloader finally succeeds.
	mux.Post(downloader.DoneEvent{})
	waitForMiningState(t, miner, true)

	// Downloader starts again.
	// Since it has achieved a DoneEvent once, we expect miner
	// state to be unchanged.
	mux.Post(downloader.StartEvent{})
	waitForMiningState(t, miner, true)

	mux.Post(downloader.FailedEvent{})
	waitForMiningState(t, miner, true)
}

func TestMinerStartStopAfterDownloaderEvents(t *testing.T) {
	t.Parallel()

	minerBor := NewBorDefaultMiner(t)
	defer func() {
		minerBor.Cleanup(false)
		minerBor.Ctrl.Finish()
	}()

	miner := minerBor.Miner
	mux := minerBor.Mux

	miner.Start(common.HexToAddress("0x12345"))
	waitForMiningState(t, miner, true)

	// Start the downloader
	mux.Post(downloader.StartEvent{})
	waitForMiningState(t, miner, false)

	// Downloader finally succeeds.
	mux.Post(downloader.DoneEvent{})
	waitForMiningState(t, miner, true)

	miner.Stop()
	waitForMiningState(t, miner, false)

	miner.Start(common.HexToAddress("0x678910"))
	waitForMiningState(t, miner, true)

	miner.Stop()
	waitForMiningState(t, miner, false)
}

func TestStartWhileDownload(t *testing.T) {
	t.Parallel()

	minerBor := NewBorDefaultMiner(t)
	defer func() {
		minerBor.Cleanup(false)
		minerBor.Ctrl.Finish()
	}()

	miner := minerBor.Miner
	mux := minerBor.Mux

	waitForMiningState(t, miner, false)
	miner.Start(common.HexToAddress("0x12345"))
	waitForMiningState(t, miner, true)

	// Stop the downloader and wait for the update loop to run
	mux.Post(downloader.StartEvent{})
	waitForMiningState(t, miner, false)

	// Starting the miner after the downloader should not work
	miner.Start(common.HexToAddress("0x12345"))
	waitForMiningState(t, miner, false)
}

func TestStartStopMiner(t *testing.T) {
	t.Parallel()

	minerBor := NewBorDefaultMiner(t)
	defer func() {
		minerBor.Cleanup(false)
		minerBor.Ctrl.Finish()
	}()

	miner := minerBor.Miner

	waitForMiningState(t, miner, false)
	miner.Start(common.HexToAddress("0x12345"))

	waitForMiningState(t, miner, true)

	miner.Stop()

	waitForMiningState(t, miner, false)
}

func TestCloseMiner(t *testing.T) {
	t.Parallel()

	minerBor := NewBorDefaultMiner(t)
	defer func() {
		minerBor.Cleanup(true)
		minerBor.Ctrl.Finish()
	}()

	miner := minerBor.Miner

	waitForMiningState(t, miner, false)

	miner.Start(common.HexToAddress("0x12345"))

	waitForMiningState(t, miner, true)

	// Terminate the miner and wait for the update loop to run
	miner.Close()

	waitForMiningState(t, miner, false)
}

// TestMinerSetEtherbase checks that etherbase becomes set even if mining isn't
// possible at the moment
func TestMinerSetEtherbase(t *testing.T) {
	t.Parallel()

	minerBor := NewBorDefaultMiner(t)
	defer func() {
		minerBor.Cleanup(false)
		minerBor.Ctrl.Finish()
	}()

	miner := minerBor.Miner
	mux := minerBor.Mux

	// Start with a 'bad' mining address
	miner.Start(common.HexToAddress("0xdead"))
	waitForMiningState(t, miner, true)

	// Start the downloader
	mux.Post(downloader.StartEvent{})
	waitForMiningState(t, miner, false)

	// Now user tries to configure proper mining address
	miner.Start(common.HexToAddress("0x1337"))

	// Stop the downloader and wait for the update loop to run
	mux.Post(downloader.DoneEvent{})

	waitForMiningState(t, miner, true)

	// The miner should now be using the good address
	if got, exp := miner.coinbase, common.HexToAddress("0x1337"); got != exp {
		t.Fatalf("Wrong coinbase, got %x expected %x", got, exp)
	}
}

// waitForMiningState waits until either
// * the desired mining state was reached
// * a timeout was reached which fails the test
func waitForMiningState(t *testing.T, m *Miner, mining bool) {
	t.Helper()

	var state bool
	for i := 0; i < 100; i++ {
		time.Sleep(10 * time.Millisecond)
		if state = m.Mining(); state == mining {
			return
		}
	}

	t.Fatalf("Mining() == %t, want %t", state, mining)
}
