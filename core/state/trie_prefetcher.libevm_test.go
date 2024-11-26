// Copyright 2024 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ava-labs/libevm/common"
)

type synchronisingWorkerPool struct {
	t                             *testing.T
	executed, unblock             chan struct{}
	done                          bool
	preconditionsToStopPrefetcher int
}

var _ WorkerPool = (*synchronisingWorkerPool)(nil)

func (p *synchronisingWorkerPool) Execute(fn func()) {
	fn()
	select {
	case <-p.executed:
	default:
		close(p.executed)
	}

	<-p.unblock
	assert.False(p.t, p.done, "Done() called before Execute() returns")
	p.preconditionsToStopPrefetcher++
}

func (p *synchronisingWorkerPool) Done() {
	p.done = true
	p.preconditionsToStopPrefetcher++
}

func TestStopPrefetcherWaitsOnWorkers(t *testing.T) {
	pool := &synchronisingWorkerPool{
		t:        t,
		executed: make(chan struct{}),
		unblock:  make(chan struct{}),
	}
	opt := WithWorkerPools(func() WorkerPool { return pool })

	db := filledStateDB()
	db.prefetcher = newTriePrefetcher(db.db, db.originalRoot, "", opt)
	db.prefetcher.prefetch(common.Hash{}, common.Hash{}, common.Address{}, [][]byte{{}})

	go func() {
		<-pool.executed
		// Sleep otherwise there is a small chance that we close pool.unblock
		// between db.StopPrefetcher() returning and the assertion.
		time.Sleep(time.Second)
		close(pool.unblock)
	}()

	<-pool.executed
	db.StopPrefetcher()
	// If this errors then either Execute() hadn't returned or Done() wasn't
	// called.
	assert.Equalf(t, 2, pool.preconditionsToStopPrefetcher, "%T.StopPrefetcher() returned early", db)
}
