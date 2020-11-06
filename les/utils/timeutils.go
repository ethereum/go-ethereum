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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
)

type UpdateTimer struct {
	clock     mclock.Clock
	lock      sync.Mutex
	last      mclock.AbsTime
	threshold time.Duration
}

func NewUpdateTimer(clock mclock.Clock, threshold time.Duration) *UpdateTimer {
	// We don't accept the update threshold less than 0.
	if threshold < 0 {
		return nil
	}
	// Don't panic for lazy users
	if clock == nil {
		clock = mclock.System{}
	}
	return &UpdateTimer{
		clock:     clock,
		last:      clock.Now(),
		threshold: threshold,
	}
}

func (t *UpdateTimer) Update(callback func(diff time.Duration) bool) bool {
	return t.UpdateAt(t.clock.Now(), callback)
}

func (t *UpdateTimer) UpdateAt(at mclock.AbsTime, callback func(diff time.Duration) bool) bool {
	t.lock.Lock()
	defer t.lock.Unlock()

	diff := time.Duration(at - t.last)
	if diff < 0 {
		diff = 0
	}
	if diff < t.threshold {
		return false
	}
	if callback(diff) {
		t.last = at
		return true
	}
	return false
}

// CostFilter is used for time based request cost metering because time measurements
// can sometimes produce very high outlier values. Each request has a prior cost estimate
// normalized to the 0 < priorCost <= 1 range (it can be constant 1 if not used).
// The filter uses a slowly changing upper limit that is scaled with the prior cost:
// filteredCost = MIN(cost, limit*priorCost)
// The limit is automatically adjusted so that AVG(filteredCost) == AVG(cost)*(1-cutRatio).
type CostFilter struct {
	cutRatio, updateRate, limit float64
}

// NewCostFilter creates a new CostFilter
func NewCostFilter(cutRatio, updateRate float64) *CostFilter {
	return &CostFilter{
		cutRatio:   cutRatio,
		updateRate: updateRate,
	}
}

// Filter returns the filtered cost value and the current cost limit
// Note: 0 < priorCost <= 1; filteredCost <= costLimit * priorCost
func (cf *CostFilter) Filter(cost, priorCost float64) (filteredCost, costLimit float64) {
	var cut float64
	limit := cf.limit * priorCost
	if cost > limit {
		filteredCost = limit
		cut = cost - limit
	} else {
		filteredCost = cost
	}
	costLimit = cf.limit
	cf.limit += (cut - cost*cf.cutRatio) * cf.updateRate
	return
}
