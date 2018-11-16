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

type reporter struct {
	reg      metrics.Registry
	interval time.Duration
	db       *leveldb.DB
}

func NewMetricsDB(r metrics.Registry, d time.Duration, path string) *leveldb.DB {
	var val = make([]byte, 8)
	var err error

	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		log.Error(err.Error())
		return nil
	}

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

	reg := &reporter{
		reg:      r,
		interval: d,
		db:       db,
	}

	go reg.run()

	return db

}

func (r *reporter) run() {
	intervalTicker := time.NewTicker(r.interval)

	for _ = range intervalTicker.C {
		if err := r.send(); err != nil {
			log.Error("unable to send metrics to LevelDB. err=%v", "err", err)
			return
		}
	}
}

func (r *reporter) send() error {
	var err error
	r.reg.Each(func(name string, i interface{}) {
		switch metric := i.(type) {
		case metrics.Counter:
			ms := metric.Snapshot()
			byteVal := make([]byte, 8)
			binary.BigEndian.PutUint64(byteVal, uint64(ms.Count()))
			err = r.db.Put([]byte(name), byteVal, nil)
		}
	})

	return err
}
