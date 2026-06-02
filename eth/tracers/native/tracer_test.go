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

package native_test

import (
	"errors"
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// TestTracerStopRace exercises the concurrent Stop / GetResult path that the
// trace RPC handler uses: a timeout watchdog goroutine calls Stop while the
// main goroutine is still running the trace and will eventually call
// GetResult. Under -race, writes to the interruption reason field must not
// race with reads, for every tracer that implements it.
//
// callTracer, flatCallTracer and erc7562Tracer's GetResult short-circuits on
// an empty callstack ("incorrect number of top-level calls") before loading
// the reason. For those tracers the test pushes a single top-level call frame
// via OnEnter so GetResult reaches the reason.Load() path where the race can
// be observed under -race.
func TestTracerStopRace(t *testing.T) {
	type setup struct {
		name       string
		needsFrame bool // whether GetResult requires a top-level call frame
	}
	cases := []setup{
		{"callTracer", true},
		{"flatCallTracer", true},
		{"4byteTracer", false},
		{"prestateTracer", false},
		{"erc7562Tracer", true},
	}
	for _, s := range cases {
		t.Run(s.name, func(t *testing.T) {
			tr, err := tracers.DefaultDirectory.New(s.name, &tracers.Context{}, nil, params.MainnetChainConfig)
			require.NoError(t, err)

			if s.needsFrame && tr.OnEnter != nil {
				// Push a single top-level call frame so GetResult doesn't
				// short-circuit before reading the interruption reason.
				tr.OnEnter(0, byte(vm.CALL), common.Address{}, common.Address{}, nil, 0, big.NewInt(0))
			}

			stopErr := errors.New("execution timeout")
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				tr.Stop(stopErr)
			}()
			go func() {
				defer wg.Done()
				_, _ = tr.GetResult()
			}()
			wg.Wait()
		})
	}
}
