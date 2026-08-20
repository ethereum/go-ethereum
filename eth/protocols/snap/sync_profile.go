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

package snap

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// Profile buckets, one per response type.
const (
	profAccount = iota
	profBytecode
	profStorage
	profKinds
)

var profKindNames = [profKinds]string{
	"account",
	"bytecode",
	"storage",
}

// profStat aggregates wall-clock samples of one instrumentation point.
type profStat struct {
	count atomic.Int64
	sum   atomic.Int64 // nanoseconds
	max   atomic.Int64 // nanoseconds
}

func (p *profStat) observe(d time.Duration) {
	p.count.Add(1)
	p.sum.Add(int64(d))

	// Best effort
	if old := p.max.Load(); int64(d) > old {
		p.max.CompareAndSwap(old, int64(d))
	}
}

// String renders the aggregate as count/total/mean/max.
func (p *profStat) String() string {
	cnt, sum, max := p.count.Load(), p.sum.Load(), p.max.Load()
	if cnt == 0 {
		return "-"
	}
	return fmt.Sprintf("n=%d total=%v avg=%v max=%v", cnt, common.PrettyDuration(sum), common.PrettyDuration(sum/cnt), common.PrettyDuration(max))
}

// syncProfile aggregates wall-clock statistics about each operation in
// the snap sync, quantifying how much the loop's serialization costs
// overall sync throughput.
type syncProfile struct {
	idle     profStat // runloop blocked in select, waiting for events
	schedule profStat // runloop top-of-loop cleanup/assignment work

	deliver [profKinds]profStat // peer threads blocked handing a verified response to the runloop
	process [profKinds]profStat // runloop handling one response
	commit  [profKinds]profStat // batch writes within the response handlers

	// commitTally accumulates the batch-write time of the current runloop
	// operation, letting the enclosing process observation net the commits
	// out.
	commitTally time.Duration
}

// observeCommit records one batch write, tallying it up so the enclosing
// process/forward observation can net it out.
func (s *syncer) observeCommit(kind int, start time.Time) {
	elapsed := time.Since(start)
	s.profile.commit[kind].observe(elapsed)
	s.profile.commitTally += elapsed
}

// reportProfile dumps the accumulated statistics.
func (s *syncer) reportProfile() {
	log.Info("State sync profile", "idle", s.profile.idle.String(), "schedule", s.profile.schedule.String())
	for i, name := range profKindNames {
		log.Info("State sync profile: "+name, "wait", s.profile.deliver[i].String(), "process", s.profile.process[i].String(), "commit", s.profile.commit[i].String())
	}
}
