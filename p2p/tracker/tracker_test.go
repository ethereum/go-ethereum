// Copyright 2026 The go-ethereum Authors
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

package tracker

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
)

// This checks that metrics gauges for pending requests are be decremented when a
// Tracker is stopped.
func TestMetricsOnStop(t *testing.T) {
	metrics.Enable()

	cap := p2p.Cap{Name: "test", Version: 1}
	tr := New(cap, "peer1", time.Minute)

	// Track some requests with different ReqCodes.
	var id uint64
	for i := 0; i < 3; i++ {
		tr.Track(Request{ID: id, ReqCode: 0x01, RespCode: 0x02, Size: 1})
		id++
	}
	for i := 0; i < 5; i++ {
		tr.Track(Request{ID: id, ReqCode: 0x03, RespCode: 0x04, Size: 1})
		id++
	}

	gauge1 := tr.trackedGauge(0x01)
	gauge2 := tr.trackedGauge(0x03)

	if gauge1.Snapshot().Value() != 3 {
		t.Fatalf("gauge1 value mismatch: got %d, want 3", gauge1.Snapshot().Value())
	}
	if gauge2.Snapshot().Value() != 5 {
		t.Fatalf("gauge2 value mismatch: got %d, want 5", gauge2.Snapshot().Value())
	}

	tr.Stop()

	if gauge1.Snapshot().Value() != 0 {
		t.Fatalf("gauge1 value after stop: got %d, want 0", gauge1.Snapshot().Value())
	}
	if gauge2.Snapshot().Value() != 0 {
		t.Fatalf("gauge2 value after stop: got %d, want 0", gauge2.Snapshot().Value())
	}
}
