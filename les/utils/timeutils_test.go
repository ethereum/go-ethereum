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

package utils

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
)

func TestUpdateTimer(t *testing.T) {
	timer := NewUpdateTimer(mclock.System{}, -1)
	if timer != nil {
		t.Fatalf("Create update timer with negative threshold")
	}
	sim := &mclock.Simulated{}
	timer = NewUpdateTimer(sim, time.Second)
	if updated := timer.Update(func(diff time.Duration) bool { return true }); updated {
		t.Fatalf("Update the clock without reaching the threshold")
	}
	sim.Run(time.Second)
	if updated := timer.Update(func(diff time.Duration) bool { return true }); !updated {
		t.Fatalf("Doesn't update the clock when reaching the threshold")
	}
	if updated := timer.UpdateAt(sim.Now()+mclock.AbsTime(time.Second), func(diff time.Duration) bool { return true }); !updated {
		t.Fatalf("Doesn't update the clock when reaching the threshold")
	}
	timer = NewUpdateTimer(sim, 0)
	if updated := timer.Update(func(diff time.Duration) bool { return true }); !updated {
		t.Fatalf("Doesn't update the clock without threshold limitaion")
	}
}

type (
	cfPoint struct {
		cost, weight float64
	}
	cfTest struct {
		cutRatio, expLimit float64
		period             []cfPoint
	}
)

func TestCostFilter(t *testing.T) {
	tests := []cfTest{
		cfTest{0.5, 50, []cfPoint{{100, 1}}},
		cfTest{0.1, 800, []cfPoint{{100, 1}, {900, 1}}},
		cfTest{0.1, 900, []cfPoint{{0, 0.1}, {150, 0.1}, {950, 1}}},
	}

	for _, test := range tests {
		cf := NewCostFilter(test.cutRatio, 0.01)
		var (
			index                int
			fc, limit, sum, fsum float64
		)
		for i := 0; i < 1000000; i++ {
			c := test.period[index]
			fc, limit = cf.Filter(c.cost, c.weight)
			sum += c.cost
			fsum += fc
			index++
			if index == len(test.period) {
				index = 0
			}
		}
		expfsum := sum * (1 - test.cutRatio)
		if fsum < expfsum*0.99 || fsum > expfsum*1.01 {
			t.Fatalf("Filtered sum is incorrect (got %f, expected %f)", fsum, expfsum)
		}
		if limit < test.expLimit*0.99 || limit > test.expLimit*1.01 {
			t.Fatalf("Limit is incorrect (got %f, expected %f)", limit, test.expLimit)
		}
	}
}
