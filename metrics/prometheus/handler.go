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
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/metrics"
)

// Handler returns http handler which dump metrics in prometheus format
func Handler(reg metrics.Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := newCollector()
		defer c.reset()

		reg.Each(func(name string, i interface{}) {
			switch m := i.(type) {
			case metrics.Counter:
				ms := m.Snapshot()
				c.addCounter(name, ms)
			case metrics.Gauge:
				ms := m.Snapshot()
				c.addGuage(name, ms)
			case metrics.GaugeFloat64:
				ms := m.Snapshot()
				c.addGuageFloat64(name, ms)
			case metrics.Histogram:
				ms := m.Snapshot()
				c.addHistogram(name, ms)
			case metrics.Meter:
				ms := m.Snapshot()
				c.addMeter(name, ms)
			case metrics.Timer:
				ms := m.Snapshot()
				c.addTimer(name, ms)
			case metrics.ResettingTimer:
				ms := m.Snapshot()
				c.addResettingTimer(name, ms)
			}
		})

		res := c.result()
		defer giveBuf(res)

		w.Header().Add("Content-Type", "text/plain")
		w.Header().Add("Content-Length", fmt.Sprint(res.Len()))
		w.Write(res.Bytes())
	})
}
