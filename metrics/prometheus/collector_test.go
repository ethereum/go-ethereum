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

package prometheus

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
)

func TestMain(m *testing.M) {
	metrics.Enabled = true
	os.Exit(m.Run())
}

func TestCollector(t *testing.T) {
	c := newCollector()

	counter := metrics.NewCounter()
	counter.Inc(12345)
	c.addCounter("test/counter", counter)

	counterfloat64 := metrics.NewCounterFloat64()
	counterfloat64.Inc(54321.98)
	c.addCounterFloat64("test/counter_float64", counterfloat64)

	gauge := metrics.NewGauge()
	gauge.Update(23456)
	c.addGauge("test/gauge", gauge)

	gaugeFloat64 := metrics.NewGaugeFloat64()
	gaugeFloat64.Update(34567.89)
	c.addGaugeFloat64("test/gauge_float64", gaugeFloat64)

	gaugeInfo := metrics.NewGaugeInfo()
	gaugeInfo.Update(metrics.GaugeInfoValue{
		"version":           "1.10.18-unstable",
		"arch":              "amd64",
		"os":                "linux",
		"commit":            "7caa2d8163ae3132c1c2d6978c76610caee2d949",
		"protocol_versions": "64 65 66",
	})
	c.addGaugeInfo("geth/info", gaugeInfo)

	histogram := metrics.NewHistogram(&metrics.NilSample{})
	c.addHistogram("test/histogram", histogram)

	meter := metrics.NewMeter()
	defer meter.Stop()
	meter.Mark(9999999)
	c.addMeter("test/meter", meter)

	timer := metrics.NewTimer()
	defer timer.Stop()
	timer.Update(20 * time.Millisecond)
	timer.Update(21 * time.Millisecond)
	timer.Update(22 * time.Millisecond)
	timer.Update(120 * time.Millisecond)
	timer.Update(23 * time.Millisecond)
	timer.Update(24 * time.Millisecond)
	c.addTimer("test/timer", timer)

	resettingTimer := metrics.NewResettingTimer()
	resettingTimer.Update(10 * time.Millisecond)
	resettingTimer.Update(11 * time.Millisecond)
	resettingTimer.Update(12 * time.Millisecond)
	resettingTimer.Update(120 * time.Millisecond)
	resettingTimer.Update(13 * time.Millisecond)
	resettingTimer.Update(14 * time.Millisecond)
	c.addResettingTimer("test/resetting_timer", resettingTimer.Snapshot())

	emptyResettingTimer := metrics.NewResettingTimer().Snapshot()
	c.addResettingTimer("test/empty_resetting_timer", emptyResettingTimer)

	var want string
	if wantB, err := os.ReadFile("./testdata/prometheus.want"); err != nil {
		t.Fatal(err)
	} else {
		want = string(wantB)
	}
	have := c.buff.String()
	if have != want {
		t.Logf("have\n%v", have)
		t.Logf("have vs want:\n %v", findFirstDiffPos(have, want))
		t.Fatal("unexpected collector output")
	}
}

func findFirstDiffPos(a, b string) string {
	x, y := []byte(a), []byte(b)
	var res []byte
	for i, ch := range x {
		if i > len(y) {
			res = append(res, ch)
			res = append(res, fmt.Sprintf("<-- diff: %#x vs EOF", ch)...)
			break
		}
		if ch != y[i] {
			res = append(res, fmt.Sprintf("<-- diff: %#x (%c) vs %#x (%c)", ch, ch, y[i], y[i])...)
			break
		}
		res = append(res, ch)
	}
	if len(res) > 100 {
		res = res[len(res)-100:]
	}
	return string(res)
}
