// Copyright 2019 The go-ethereum Authors
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
	"bytes"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/metrics"
)

var (
	countersKey        = []byte("counters")
	gaugesKey          = []byte("gauges")
	metersKey          = []byte("meters")
	histogramsKey      = []byte("histograms")
	timersKey          = []byte("timers")
	resettingTimersKey = []byte("resettingTimers")

	countersHeader       = []byte("# HELP counters is set of geth counters\n# TYPE counters gauge\n")
	gaugesHeader         = []byte("# HELP gauges is set of geth gauges\n# TYPE gauges gauge\n")
	meterHeader          = []byte("# HELP meters is set of geth meters\n# TYPE meters counter\n")
	histogramHeader      = []byte("# HELP histograms is set of geth histograms\n# TYPE histograms summary\n")
	timerHeader          = []byte("# HELP timers is set of geth timers\n# TYPE timers summary\n")
	resettingTimerHeader = []byte("# HELP resetting_timers is set of geth resetting timers\n# TYPE resetting_timers summary\n")

	counterSuffix           = []byte("_count")
	nameTagTemplate         = "{name=\"%s\"} %v\n"
	nameQuantileTagTemplate = "{name=\"%s\",quantile=\"%s\"} %v\n"
)

var bufPool sync.Pool

func getBuf() *bytes.Buffer {
	buf := bufPool.Get()
	if buf == nil {
		return &bytes.Buffer{}
	}
	return buf.(*bytes.Buffer)
}

func giveBuf(buf *bytes.Buffer) {
	buf.Reset()
	bufPool.Put(buf)
}

type collector struct {
	counters        *bytes.Buffer
	gauges          *bytes.Buffer
	histograms      *bytes.Buffer
	meters          *bytes.Buffer
	timers          *bytes.Buffer
	resettingTimers *bytes.Buffer
}

func newCollector() *collector {
	return &collector{
		counters:        getBuf(),
		gauges:          getBuf(),
		histograms:      getBuf(),
		meters:          getBuf(),
		timers:          getBuf(),
		resettingTimers: getBuf(),
	}
}

func (c *collector) reset() {
	giveBuf(c.counters)
	giveBuf(c.gauges)
	giveBuf(c.histograms)
	giveBuf(c.meters)
	giveBuf(c.timers)
	giveBuf(c.resettingTimers)
}

func (c *collector) result() *bytes.Buffer {
	buf := getBuf()
	if c.counters.Len() > 0 {
		buf.Write(countersHeader)
		buf.Write(c.counters.Bytes())
	}

	if c.gauges.Len() > 0 {
		buf.Write(gaugesHeader)
		buf.Write(c.gauges.Bytes())
	}

	if c.meters.Len() > 0 {
		buf.Write(meterHeader)
		buf.Write(c.meters.Bytes())
	}

	if c.histograms.Len() > 0 {
		buf.Write(histogramHeader)
		buf.Write(c.histograms.Bytes())
	}

	if c.timers.Len() > 0 {
		buf.Write(timerHeader)
		buf.Write(c.timers.Bytes())
	}

	if c.resettingTimers.Len() > 0 {
		buf.Write(resettingTimerHeader)
		buf.Write(c.resettingTimers.Bytes())
	}

	return buf
}

func (c *collector) addCounter(name string, m metrics.Counter) {
	writeGuageCounter(c.counters, countersKey, name, m.Count())
}

func (c *collector) addGuage(name string, m metrics.Gauge) {
	writeGuageCounter(c.gauges, gaugesKey, name, m.Value())
}

func (c *collector) addGuageFloat64(name string, m metrics.GaugeFloat64) {
	writeGuageCounter(c.gauges, gaugesKey, name, m.Value())
}

func (c *collector) addHistogram(name string, m metrics.Histogram) {
	ps := m.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
	writeSummaryCounter(c.histograms, histogramsKey, name, m.Count())
	writeSummaryPercentile(c.histograms, histogramsKey, name, "0.5", ps[0])
	writeSummaryPercentile(c.histograms, histogramsKey, name, "0.75", ps[1])
	writeSummaryPercentile(c.histograms, histogramsKey, name, "0.95", ps[2])
	writeSummaryPercentile(c.histograms, histogramsKey, name, "0.99", ps[3])
	writeSummaryPercentile(c.histograms, histogramsKey, name, "0.999", ps[4])
	writeSummaryPercentile(c.histograms, histogramsKey, name, "0.9999", ps[5])
}

func (c *collector) addMeter(name string, m metrics.Meter) {
	writeGuageCounter(c.meters, metersKey, name, m.Count())
}

func (c *collector) addTimer(name string, m metrics.Timer) {
	ps := m.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
	writeSummaryCounter(c.timers, timersKey, name, m.Count())
	writeSummaryPercentile(c.timers, timersKey, name, "0.50", ps[0])
	writeSummaryPercentile(c.timers, timersKey, name, "0.75", ps[1])
	writeSummaryPercentile(c.timers, timersKey, name, "0.95", ps[2])
	writeSummaryPercentile(c.timers, timersKey, name, "0.99", ps[3])
	writeSummaryPercentile(c.timers, timersKey, name, "0.999", ps[4])
	writeSummaryPercentile(c.timers, timersKey, name, "0.9999", ps[5])
}

func (c *collector) addResettingTimer(name string, m metrics.ResettingTimer) {
	if len(m.Values()) <= 0 {
		return
	}
	ps := m.Percentiles([]float64{50, 95, 99})
	val := m.Values()
	writeSummaryCounter(c.resettingTimers, resettingTimersKey, name, len(val))
	writeSummaryPercentile(c.resettingTimers, resettingTimersKey, name, "0.50", ps[0])
	writeSummaryPercentile(c.resettingTimers, resettingTimersKey, name, "0.95", ps[1])
	writeSummaryPercentile(c.resettingTimers, resettingTimersKey, name, "0.99", ps[2])
}

func writeGuageCounter(buf *bytes.Buffer, key []byte, name string, value interface{}) {
	buf.Write(key)
	buf.WriteString(fmt.Sprintf(nameTagTemplate, name, value))
}

func writeSummaryCounter(buf *bytes.Buffer, key []byte, name string, value interface{}) {
	buf.Write(append(key, counterSuffix...))
	buf.WriteString(fmt.Sprintf(nameTagTemplate, name, value))
}

func writeSummaryPercentile(buf *bytes.Buffer, key []byte, name, p string, value interface{}) {
	buf.Write(key)
	buf.WriteString(fmt.Sprintf(nameQuantileTagTemplate, name, p, value))
}
