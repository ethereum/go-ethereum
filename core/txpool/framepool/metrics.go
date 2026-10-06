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

package framepool

import "github.com/ethereum/go-ethereum/metrics"

var (
	pendingGauge  = metrics.NewRegisteredGauge("framepool/pending", nil)
	slotusedGauge = metrics.NewRegisteredGauge("framepool/slotused", nil)
	pooltipGauge  = metrics.NewRegisteredGauge("framepool/pooltip", nil)

	acceptedMeter = metrics.NewRegisteredMeter("framepool/add/accepted", nil)
	rejectedMeter = metrics.NewRegisteredMeter("framepool/add/rejected", nil)
	replacedMeter = metrics.NewRegisteredMeter("framepool/add/replaced", nil)
	evictedMeter  = metrics.NewRegisteredMeter("framepool/evicted", nil)
	resettimeHist = metrics.NewRegisteredHistogram("framepool/reset/time", nil, metrics.NewExpDecaySample(1028, 0.015))
)
