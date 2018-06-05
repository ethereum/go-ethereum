// Copyright 2016 The go-ethereum Authors
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

package les

import (
	"encoding/binary"
	"math"
	"sync"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
)

type requestCosts struct {
	baseCost, reqCost uint64
}

type requestCostTable map[uint64]*requestCosts

type RequestCostList []struct {
	MsgCode, BaseCost, ReqCost uint64
}

func (list RequestCostList) decode() requestCostTable {
	table := make(requestCostTable)
	for _, e := range list {
		table[e.MsgCode] = &requestCosts{
			baseCost: e.BaseCost,
			reqCost:  e.ReqCost,
		}
	}
	return table
}

type linReg struct {
	sumX, sumY, sumXX, sumXY float64
	cnt                      uint64
}

const linRegMaxCnt = 100000

func (l *linReg) add(x, y float64) {
	if l.cnt >= linRegMaxCnt {
		sub := float64(l.cnt+1-linRegMaxCnt) / linRegMaxCnt
		l.sumX -= l.sumX * sub
		l.sumY -= l.sumY * sub
		l.sumXX -= l.sumXX * sub
		l.sumXY -= l.sumXY * sub
		l.cnt = linRegMaxCnt - 1
	}
	l.cnt++
	l.sumX += x
	l.sumY += y
	l.sumXX += x * x
	l.sumXY += x * y
}

func (l *linReg) calc() (b, m float64) {
	if l.cnt == 0 {
		return 0, 0
	}
	cnt := float64(l.cnt)
	d := cnt*l.sumXX - l.sumX*l.sumX
	if d < 0.001 {
		return l.sumY / cnt, 0
	}
	m = (cnt*l.sumXY - l.sumX*l.sumY) / d
	b = (l.sumY / cnt) - (m * l.sumX / cnt)
	return b, m
}

func (l *linReg) toBytes() []byte {
	var arr [40]byte
	binary.BigEndian.PutUint64(arr[0:8], math.Float64bits(l.sumX))
	binary.BigEndian.PutUint64(arr[8:16], math.Float64bits(l.sumY))
	binary.BigEndian.PutUint64(arr[16:24], math.Float64bits(l.sumXX))
	binary.BigEndian.PutUint64(arr[24:32], math.Float64bits(l.sumXY))
	binary.BigEndian.PutUint64(arr[32:40], l.cnt)
	return arr[:]
}

func linRegFromBytes(data []byte) *linReg {
	if len(data) != 40 {
		return nil
	}
	l := &linReg{}
	l.sumX = math.Float64frombits(binary.BigEndian.Uint64(data[0:8]))
	l.sumY = math.Float64frombits(binary.BigEndian.Uint64(data[8:16]))
	l.sumXX = math.Float64frombits(binary.BigEndian.Uint64(data[16:24]))
	l.sumXY = math.Float64frombits(binary.BigEndian.Uint64(data[24:32]))
	l.cnt = binary.BigEndian.Uint64(data[32:40])
	return l
}

type requestCostStats struct {
	lock  sync.RWMutex
	db    ethdb.Database
	stats map[uint64]*linReg
}

type requestCostStatsRlp []struct {
	MsgCode uint64
	Data    []byte
}

var rcStatsKey = []byte("_requestCostStats")

func newCostStats(db ethdb.Database) *requestCostStats {
	stats := make(map[uint64]*linReg)
	for _, code := range reqList {
		stats[code] = &linReg{cnt: 100}
	}

	if db != nil {
		data, err := db.Get(rcStatsKey)
		var statsRlp requestCostStatsRlp
		if err == nil {
			err = rlp.DecodeBytes(data, &statsRlp)
		}
		if err == nil {
			for _, r := range statsRlp {
				if stats[r.MsgCode] != nil {
					if l := linRegFromBytes(r.Data); l != nil {
						stats[r.MsgCode] = l
					}
				}
			}
		}
	}

	return &requestCostStats{
		db:    db,
		stats: stats,
	}
}

func (s *requestCostStats) store() {
	s.lock.Lock()
	defer s.lock.Unlock()

	statsRlp := make(requestCostStatsRlp, len(reqList))
	for i, code := range reqList {
		statsRlp[i].MsgCode = code
		statsRlp[i].Data = s.stats[code].toBytes()
	}

	if data, err := rlp.EncodeToBytes(statsRlp); err == nil {
		s.db.Put(rcStatsKey, data)
	}
}

func (s *requestCostStats) getCurrentList() RequestCostList {
	s.lock.Lock()
	defer s.lock.Unlock()

	list := make(RequestCostList, len(reqList))
	//fmt.Println("RequestCostList")
	for idx, code := range reqList {
		b, m := s.stats[code].calc()
		//fmt.Println(code, s.stats[code].cnt, b/1000000, m/1000000)
		if m < 0 {
			b += m
			m = 0
		}
		if b < 0 {
			b = 0
		}

		list[idx].MsgCode = code
		list[idx].BaseCost = uint64(b * 2)
		list[idx].ReqCost = uint64(m * 2)
	}
	return list
}

func (s *requestCostStats) update(msgCode, reqCnt, cost uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	c, ok := s.stats[msgCode]
	if !ok || reqCnt == 0 {
		return
	}
	c.add(float64(reqCnt), float64(cost))
}
