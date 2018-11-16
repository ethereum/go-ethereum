// Copyright 2018 The go-ethereum Authors
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

package protocols

import (
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/rcrowley/go-metrics"
)

type reporter struct {
	reg        metrics.Registry
	interval   time.Duration
	stateStore *state.DBStore
}

func NewMetricsStateStore(r metrics.Registry, d time.Duration, path string) {
	stateStore, err := state.NewDBStore(path)
	if err != nil {
		return
	}

	rep := &reporter{
		reg:        r,
		interval:   d,
		stateStore: stateStore,
	}

	rep.run()
}

func (r *reporter) run() {
	intervalTicker := time.NewTicker(r.interval)

	for _ = range intervalTicker.C {
		if err := r.send(); err != nil {
			log.Error("unable to send metrics to InfluxDB. err=%v", err)
		}
	}
}

func (r *reporter) send() error {

	var err error

	r.reg.Each(func(name string, i interface{}) {
		switch metric := i.(type) {
		case metrics.Counter:
			ms := metric.Snapshot()
			err = r.stateStore.Put(name, ms.Count())
		}
	})

	return err
}
