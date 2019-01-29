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
	"encoding/binary"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/syndtr/goleveldb/leveldb"
)

//AccountMetrics abstracts away the metrics DB and
//the reporter to persist metrics
type AccountingMetrics struct {
	reporter *reporter
}

//Close will be called when the node is being shutdown
//for a graceful cleanup
func (am *AccountingMetrics) Close() {
	close(am.reporter.quit)
	// wait for reporter loop to finish saving metrics
	// before reporter database is closed
	select {
	case <-time.After(10 * time.Second):
		log.Error("accounting metrics reporter timeout")
	case <-am.reporter.done:
	}
	am.reporter.db.Close()
}

//reporter is an internal structure used to write p2p accounting related
//metrics to a LevelDB. It will periodically write the accrued metrics to the DB.
type reporter struct {
	reg      metrics.Registry //the registry for these metrics (independent of other metrics)
	interval time.Duration    //duration at which the reporter will persist metrics
	db       *leveldb.DB      //the actual DB
	quit     chan struct{}    //quit the reporter loop
	done     chan struct{}    //signal that reporter loop is done
}

//NewMetricsDB creates a new LevelDB instance used to persist metrics defined
//inside p2p/protocols/accounting.go
func NewAccountingMetrics(r metrics.Registry, d time.Duration, path string) *AccountingMetrics {
	var val = make([]byte, 8)
	var err error

	//Create the LevelDB
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		log.Error(err.Error())
		return nil
	}

	//Check for all defined metrics that there is a value in the DB
	//If there is, assign it to the metric. This means that the node
	//has been running before and that metrics have been persisted.
	metricsMap := map[string]metrics.Counter{
		"account.balance.credit": mBalanceCredit,
		"account.balance.debit":  mBalanceDebit,
		"account.bytes.credit":   mBytesCredit,
		"account.bytes.debit":    mBytesDebit,
		"account.msg.credit":     mMsgCredit,
		"account.msg.debit":      mMsgDebit,
		"account.peerdrops":      mPeerDrops,
		"account.selfdrops":      mSelfDrops,
	}
	//iterate the map and get the values
	for key, metric := range metricsMap {
		val, err = db.Get([]byte(key), nil)
		//until the first time a value is being written,
		//this will return an error.
		//it could be beneficial though to log errors later,
		//but that would require a different logic
		if err == nil {
			metric.Inc(int64(binary.BigEndian.Uint64(val)))
		}
	}

	//create the reporter
	rep := &reporter{
		reg:      r,
		interval: d,
		db:       db,
		quit:     make(chan struct{}),
		done:     make(chan struct{}),
	}

	//run the go routine
	go rep.run()

	m := &AccountingMetrics{
		reporter: rep,
	}

	return m
}

//run is the goroutine which periodically sends the metrics to the configured LevelDB
func (r *reporter) run() {
	// signal that the reporter loop is done
	defer close(r.done)

	intervalTicker := time.NewTicker(r.interval)

	for {
		select {
		case <-intervalTicker.C:
			//at each tick send the metrics
			if err := r.save(); err != nil {
				log.Error("unable to send metrics to LevelDB", "err", err)
				//If there is an error in writing, exit the routine; we assume here that the error is
				//severe and don't attempt to write again.
				//Also, this should prevent leaking when the node is stopped
				return
			}
		case <-r.quit:
			//graceful shutdown
			if err := r.save(); err != nil {
				log.Error("unable to send metrics to LevelDB", "err", err)
			}
			return
		}
	}
}

//send the metrics to the DB
func (r *reporter) save() error {
	//create a LevelDB Batch
	batch := leveldb.Batch{}
	//for each metric in the registry (which is independent)...
	r.reg.Each(func(name string, i interface{}) {
		metric, ok := i.(metrics.Counter)
		if ok {
			//assuming every metric here to be a Counter (separate registry)
			//...create a snapshot...
			ms := metric.Snapshot()
			byteVal := make([]byte, 8)
			binary.BigEndian.PutUint64(byteVal, uint64(ms.Count()))
			//...and save the value to the DB
			batch.Put([]byte(name), byteVal)
		}
	})
	return r.db.Write(&batch, nil)
}
