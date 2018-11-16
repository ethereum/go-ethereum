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

//reporter is an internal structure used to write p2p accounting related
//metrics to a LevelDB. It will periodically write the accrued metrics to the DB.
type reporter struct {
	reg      metrics.Registry //the registry for these metrics (independent of other metrics)
	interval time.Duration    //duration at which the reporter will persist metrics
	db       *leveldb.DB      //the actual DB
}

//NewMetricsDB creates a new LevelDB instance used to persist metrics defined
//inside p2p/protocols/accounting.go
func NewMetricsDB(r metrics.Registry, d time.Duration, path string) *leveldb.DB {
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
	val, err = db.Get([]byte("account.balance.credit"), nil)
	if err == nil {
		mBalanceCredit.Inc(int64(binary.BigEndian.Uint64(val)))
	}
	val, err = db.Get([]byte("account.balance.debit"), nil)
	if err == nil {
		mBalanceDebit.Inc(int64(binary.BigEndian.Uint64(val)))
	}
	val, err = db.Get([]byte("account.bytes.credit"), nil)
	if err == nil {
		mBytesCredit.Inc(int64(binary.BigEndian.Uint64(val)))
	}
	val, err = db.Get([]byte("account.bytes.debit"), nil)
	if err == nil {
		mBytesDebit.Inc(int64(binary.BigEndian.Uint64(val)))
	}
	val, err = db.Get([]byte("account.msg.credit"), nil)
	if err == nil {
		mMsgCredit.Inc(int64(binary.BigEndian.Uint64(val)))
	}
	val, err = db.Get([]byte("account.msg.debit"), nil)
	if err == nil {
		mMsgDebit.Inc(int64(binary.BigEndian.Uint64(val)))
	}
	val, err = db.Get([]byte("account.peerdrops"), nil)
	if err == nil {
		mPeerDrops.Inc(int64(binary.BigEndian.Uint64(val)))
	}
	val, err = db.Get([]byte("account.selfdrops"), nil)
	if err == nil {
		mSelfDrops.Inc(int64(binary.BigEndian.Uint64(val)))
	}

	//create the reporter
	reg := &reporter{
		reg:      r,
		interval: d,
		db:       db,
	}

	//run the go routine
	go reg.run()

	return db

}

//run is the go routine which periodically sends the metrics to the configued LevelDB
func (r *reporter) run() {
	intervalTicker := time.NewTicker(r.interval)

	for _ = range intervalTicker.C {
		//at each tick send the metrics
		if err := r.send(); err != nil {
			log.Error("unable to send metrics to LevelDB. err=%v", "err", err)
			//If there is an error in writing, exit the routine; we assume here that the error is
			//severe and don't attempt to write again.
			//Also, this should prevent leaking when the node is stopped
			return
		}
	}
}

//send the metrics to the DB
func (r *reporter) send() error {
	var err error
	//for each metric in the registry (which is independent)...
	r.reg.Each(func(name string, i interface{}) {
		switch metric := i.(type) {
		//assuming every metric here to be a Counter (separate registry)
		case metrics.Counter:
			//...create a snapshot...
			ms := metric.Snapshot()
			byteVal := make([]byte, 8)
			binary.BigEndian.PutUint64(byteVal, uint64(ms.Count()))
			//...and save the value to the DB
			err = r.db.Put([]byte(name), byteVal, nil)
		default:
		}
	})
	return err
}
