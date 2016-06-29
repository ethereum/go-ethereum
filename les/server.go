// Copyright 2015 The go-ethereum Authors
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

// Package les implements the Light Ethereum Subprotocol.
package les

import (
	"sync"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/les/flowcontrol"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
)

type LesServer struct {
	protocolManager *ProtocolManager
	fcManager       *flowcontrol.ClientManager // nil if our node is client only
	fcCostStats     *requestCostStats
	defParams       *flowcontrol.ServerParams
}

func NewLesServer(eth *eth.FullNodeService, config *eth.Config) (*LesServer, error) {
	pm, err := NewProtocolManager(config.ChainConfig, false, config.NetworkId, eth.EventMux(), eth.Pow(), eth.BlockChain(), eth.TxPool(), eth.ChainDb(), nil, nil)
	if err != nil {
		return nil, err
	}
	pm.broadcastBlockLoop()

	srv := &LesServer{protocolManager: pm}
	pm.server = srv

	srv.defParams = &flowcontrol.ServerParams{
		BufLimit:    300000000,
		MinRecharge: 50000,
	}
	srv.fcManager = flowcontrol.NewClientManager(uint64(config.LightServ), 10, 1000000000)
	srv.fcCostStats = newCostStats(eth.ChainDb())
	return srv, nil
}

func (s *LesServer) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

func (s *LesServer) Start() {
	s.protocolManager.Start()
}

func (s *LesServer) Stop() {
	s.fcCostStats.store()
	s.fcManager.Stop()
	go func() {
		<-s.protocolManager.noMorePeers
	}()
	s.protocolManager.Stop()
}

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

func (table requestCostTable) encode() RequestCostList {
	list := make(RequestCostList, len(table))
	for idx, code := range reqList {
		list[idx].MsgCode = code
		list[idx].BaseCost = table[code].baseCost
		list[idx].ReqCost = table[code].reqCost
	}
	return list
}

type requestCostStats struct {
	lock     sync.RWMutex
	db       ethdb.Database
	avg      requestCostTable
	baseCost uint64
}

var rcStatsKey = []byte("requestCostStats")

func newCostStats(db ethdb.Database) *requestCostStats {
	table := make(requestCostTable)
	for _, code := range reqList {
		table[code] = &requestCosts{0, 100000}
	}

	/*	if db != nil {
		var cl RequestCostList
		data, err := db.Get(rcStatsKey)
		if err == nil {
			err = rlp.DecodeBytes(data, &cl)
		}
		if err == nil {
			t := cl.decode()
			for code, entry := range t {
				table[code] = entry
			}
		}
	}*/

	return &requestCostStats{
		db:       db,
		avg:      table,
		baseCost: 100000,
	}
}

func (s *requestCostStats) store() {
	s.lock.Lock()
	defer s.lock.Unlock()

	list := s.avg.encode()
	if data, err := rlp.EncodeToBytes(list); err == nil {
		s.db.Put(rcStatsKey, data)
	}
}

func (s *requestCostStats) getCurrentList() RequestCostList {
	s.lock.Lock()
	defer s.lock.Unlock()

	list := make(RequestCostList, len(s.avg))
	for idx, code := range reqList {
		list[idx].MsgCode = code
		list[idx].BaseCost = s.baseCost
		list[idx].ReqCost = s.avg[code].reqCost * 2
	}
	return list
}

func (s *requestCostStats) update(msgCode, reqCnt, cost uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	c, ok := s.avg[msgCode]
	if !ok || reqCnt == 0 {
		return
	}
	cost = cost / reqCnt
	if cost > c.reqCost {
		c.reqCost += (cost - c.reqCost) / 10
	} else {
		c.reqCost -= (c.reqCost - cost) / 100
	}
}
