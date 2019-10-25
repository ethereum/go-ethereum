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
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	httpRequestGauge = metrics.NewRegisteredGauge("les/client/req/http/count", nil)
	httpRequestTimer = metrics.NewRegisteredTimer("les/cleint/req/http/duration", nil)
	p2pRequestGauge  = metrics.NewRegisteredGauge("les/client/req/p2p/count", nil)
	p2pRequestTimer  = metrics.NewRegisteredTimer("les/cleint/req/p2p/duration", nil)
)

// LesOdr implements light.OdrBackend
type LesOdr struct {
	db                                         ethdb.Database
	indexerConfig                              *light.IndexerConfig
	chtIndexer, bloomTrieIndexer, bloomIndexer *core.ChainIndexer
	p2pRtr                                     *p2pRetriever
	httpRtr                                    *httpRetriever
	stop                                       chan struct{}
}

func NewLesOdr(db ethdb.Database, config *light.IndexerConfig, p2prtr *p2pRetriever, httprtr *httpRetriever) *LesOdr {
	return &LesOdr{
		db:            db,
		indexerConfig: config,
		p2pRtr:        p2prtr,
		httpRtr:       httprtr,
		stop:          make(chan struct{}),
	}
}

// Stop cancels all pending retrievals
func (odr *LesOdr) Stop() {
	close(odr.stop)
}

// Database returns the backing database
func (odr *LesOdr) Database() ethdb.Database {
	return odr.db
}

// SetIndexers adds the necessary chain indexers to the ODR backend
func (odr *LesOdr) SetIndexers(chtIndexer, bloomTrieIndexer, bloomIndexer *core.ChainIndexer) {
	odr.chtIndexer = chtIndexer
	odr.bloomTrieIndexer = bloomTrieIndexer
	odr.bloomIndexer = bloomIndexer
}

// ChtIndexer returns the CHT chain indexer
func (odr *LesOdr) ChtIndexer() *core.ChainIndexer {
	return odr.chtIndexer
}

// BloomTrieIndexer returns the bloom trie chain indexer
func (odr *LesOdr) BloomTrieIndexer() *core.ChainIndexer {
	return odr.bloomTrieIndexer
}

// BloomIndexer returns the bloombits chain indexer
func (odr *LesOdr) BloomIndexer() *core.ChainIndexer {
	return odr.bloomIndexer
}

// IndexerConfig returns the indexer config.
func (odr *LesOdr) IndexerConfig() *light.IndexerConfig {
	return odr.indexerConfig
}

const (
	MsgBlockBodies = iota
	MsgCode
	MsgReceipts
	MsgProofsV2
	MsgHelperTrieProofs
	MsgTxStatus
)

// Msg encodes a LES message that delivers reply data for a request
type Msg struct {
	MsgType int
	ReqID   uint64
	Obj     interface{}
}

// Retrieve tries to fetch an object from the LES network.
// If the network retrieval was successful, it stores the object in local db.
func (odr *LesOdr) Retrieve(ctx context.Context, req light.OdrRequest) error {
	defer func(start time.Time) {
		log.Debug("Retrieved data", "elapsed", common.PrettyDuration(time.Since(start)))
	}(time.Now())

	var (
		count   int
		wg      sync.WaitGroup
		errorCh = make(chan error, 2)

		ctx1, cancelFn1 = context.WithCancel(ctx)
		ctx2, cancelFn2 = context.WithCancel(ctx)
	)
	// retrieve invokes given retrival action, update metrics no matter successful
	// or not, return error via buffered channel.
	retrieve := func(method string, action func() error, successCallback func(), gauge metrics.Gauge, timer metrics.Timer) {
		defer wg.Done()

		defer func(start time.Time) {
			gauge.Update(gauge.Value() + 1)
			timer.UpdateSince(start)
			log.Debug("Retrieved data", "method", method, "elasped", common.PrettyDuration(time.Since(start)))
		}(time.Now())

		err := action()
		if err == nil {
			successCallback()
		}
		errorCh <- err
	}
	// If p2p retriever is available, spin it up.
	if odr.p2pRtr != nil {
		wg.Add(1)
		count += 1

		reqID := genReqID()
		rq := &distReq{
			getCost: func(dp distPeer) uint64 { return LesRequest(req).GetCost(dp.(*serverPeer)) },
			canSend: func(dp distPeer) bool {
				p := dp.(*serverPeer)
				if !p.onlyAnnounce {
					return LesRequest(req).CanSend(p)
				}
				return false
			},
			request: func(dp distPeer) func() {
				p := dp.(*serverPeer)
				cost := LesRequest(req).GetCost(p)
				p.fcServer.QueuedRequest(reqID, cost)
				return func() { LesRequest(req).Request(reqID, p) }
			},
		}
		go retrieve("p2p", func() error {
			return odr.p2pRtr.retrieve(ctx1, reqID, rq, func(p distPeer, msg *Msg) error { return LesRequest(req).Validate(odr.db, msg) }, odr.stop)
		}, func() {
			cancelFn2() // Explicitly stop http retriever
		}, p2pRequestGauge, p2pRequestTimer)
	}
	// If http retriever is available, spin it up.
	if odr.httpRtr != nil {
		wg.Add(1)
		count += 1
		go retrieve("http", func() error {
			return odr.httpRtr.retrieve(ctx2, LesRequest(req))
		}, func() {
			cancelFn1() //  Explicitly stop p2p retriever
		}, httpRequestGauge, httpRequestTimer)
	}
	if count == 0 {
		return errors.New("no available retriever")
	}
	// Waiting the response. If any returned error is nil, regard data
	// retreval successfully.
	wg.Wait()

	var mix string
	for i := 0; i < count; i++ {
		if err := <-errorCh; err != nil {
			mix = mix + ":" + err.Error()
		} else {
			req.StoreResult(odr.db) // retrieved from network, store in db
			return nil
		}
	}
	return errors.New(mix)
}
