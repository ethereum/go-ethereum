// Copyright 2023 The go-ethereum Authors
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

package internal

import (
	"time"

	"github.com/ethereum/go-ethereum/metrics"
)

// ExampleMetrics returns an ordered registry populated with a sample of metrics.
func ExampleMetrics() metrics.Registry {
	var registry = metrics.NewOrderedRegistry()

	metrics.NewRegisteredCounterFloat64("test/counter", registry).Inc(12345)
	metrics.NewRegisteredCounterFloat64("test/counter_float64", registry).Inc(54321.98)
	metrics.NewRegisteredGauge("test/gauge", registry).Update(23456)
	metrics.NewRegisteredGaugeFloat64("test/gauge_float64", registry).Update(34567.89)
	metrics.NewRegisteredGaugeInfo("test/gauge_info", registry).Update(
		metrics.GaugeInfoValue{
			"version":           "1.10.18-unstable",
			"arch":              "amd64",
			"os":                "linux",
			"commit":            "7caa2d8163ae3132c1c2d6978c76610caee2d949",
			"protocol_versions": "64 65 66",
		})
	metrics.NewRegisteredHistogram("test/histogram", registry, metrics.NewSampleSnapshot(3, []int64{1, 2, 3}))
	registry.Register("test/meter", metrics.NewInactiveMeter())
	{
		timer := metrics.NewRegisteredResettingTimer("test/resetting_timer", registry)
		timer.Update(10 * time.Millisecond)
		timer.Update(11 * time.Millisecond)
		timer.Update(12 * time.Millisecond)
		timer.Update(120 * time.Millisecond)
		timer.Update(13 * time.Millisecond)
		timer.Update(14 * time.Millisecond)
	}
	{
		timer := metrics.NewRegisteredTimer("test/timer", registry)
		timer.Update(20 * time.Millisecond)
		timer.Update(21 * time.Millisecond)
		timer.Update(22 * time.Millisecond)
		timer.Update(120 * time.Millisecond)
		timer.Update(23 * time.Millisecond)
		timer.Update(24 * time.Millisecond)
		timer.Stop()
	}
	registry.Register("test/empty_resetting_timer", metrics.NewResettingTimer().Snapshot())
	return registry
}
