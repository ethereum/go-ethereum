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

// Package les implements the Light Ethereum Subprotocol.
package les

import (
	"crypto/ecdsa"
	"encoding/binary"
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/les/flowcontrol"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	bufLimitRatio = 6000 // fixed bufLimit/MRR ratio
	makeCostStats = true // make request cost statistics during operation
)

var (
	reqAvgTime = requestCostTable{
		GetBlockHeadersMsg:     {150000, 30000},
		GetBlockBodiesMsg:      {0, 700000},
		GetReceiptsMsg:         {0, 1000000},
		GetCodeMsg:             {0, 450000},
		GetProofsV1Msg:         {0, 600000},
		GetProofsV2Msg:         {0, 600000},
		GetHeaderProofsMsg:     {0, 1000000},
		GetHelperTrieProofsMsg: {0, 1000000},
		SendTxMsg:              {0, 450000},
		SendTxV2Msg:            {0, 450000},
		GetTxStatusMsg:         {0, 250000},
	}
	reqMaxInSize = requestCostTable{
		GetBlockHeadersMsg:     {40, 0},
		GetBlockBodiesMsg:      {0, 40},
		GetReceiptsMsg:         {0, 40},
		GetCodeMsg:             {0, 80},
		GetProofsV1Msg:         {0, 80},
		GetProofsV2Msg:         {0, 80},
		GetHeaderProofsMsg:     {0, 20},
		GetHelperTrieProofsMsg: {0, 20},
		SendTxMsg:              {0, 66000},
		SendTxV2Msg:            {0, 66000},
		GetTxStatusMsg:         {0, 50},
	}
	reqMaxOutSize = requestCostTable{
		GetBlockHeadersMsg:     {0, 556},
		GetBlockBodiesMsg:      {0, 100000},
		GetReceiptsMsg:         {0, 200000},
		GetCodeMsg:             {0, 50000},
		GetProofsV1Msg:         {0, 4000},
		GetProofsV2Msg:         {0, 4000},
		GetHeaderProofsMsg:     {0, 4000},
		GetHelperTrieProofsMsg: {0, 4000},
		SendTxMsg:              {0, 0},
		SendTxV2Msg:            {0, 100},
		GetTxStatusMsg:         {0, 100},
	}
	minBufLimit = 100000000
	minCapacity = uint64((minBufLimit-1)/bufLimitRatio + 1)
)

type LesServer struct {
	lesCommons

	fcManager    *flowcontrol.ClientManager // nil if our node is client only
	fcCostStats  *requestCostStats
	defParams    flowcontrol.ServerParams
	lesTopics    []discv5.Topic
	privateKey   *ecdsa.PrivateKey
	quitSync     chan struct{}
	onlyAnnounce bool

	totalCapacity                 uint64
	rcNormal, rcBlockProcessing   flowcontrol.PieceWiseLinear // buffer recharge curve for normal operation and block processing mode
	thcNormal, thcBlockProcessing int                         // serving thread count for normal operation and block processing mode

	inSizeCostFactor, outSizeCostFactor float64

	globalCostFactor float64
	gcfUpdateCh      chan gcfUpdate
	gcfLock          sync.RWMutex

	freeClientCap      uint64
	freeClientPool     *freeClientPool
	priorityClientPool *priorityClientPool
}

func NewLesServer(eth *eth.Ethereum, config *eth.Config) (*LesServer, error) {
	quitSync := make(chan struct{})
	pm, err := NewProtocolManager(
		eth.BlockChain().Config(),
		light.DefaultServerIndexerConfig,
		false,
		config.NetworkId,
		eth.EventMux(),
		eth.Engine(),
		newPeerSet(),
		eth.BlockChain(),
		eth.TxPool(),
		eth.ChainDb(),
		nil,
		nil,
		nil,
		quitSync,
		new(sync.WaitGroup),
		config.ULC)
	if err != nil {
		return nil, err
	}

	lesTopics := make([]discv5.Topic, len(AdvertiseProtocolVersions))
	for i, pv := range AdvertiseProtocolVersions {
		lesTopics[i] = lesTopic(eth.BlockChain().Genesis().Hash(), pv)
	}

	srv := &LesServer{
		lesCommons: lesCommons{
			config:           config,
			chainDb:          eth.ChainDb(),
			iConfig:          light.DefaultServerIndexerConfig,
			chtIndexer:       light.NewChtIndexer(eth.ChainDb(), nil, params.CHTFrequencyServer, params.HelperTrieProcessConfirmations),
			bloomTrieIndexer: light.NewBloomTrieIndexer(eth.ChainDb(), nil, params.BloomBitsBlocks, params.BloomTrieFrequency),
			protocolManager:  pm,
		},
		quitSync:     quitSync,
		lesTopics:    lesTopics,
		onlyAnnounce: config.OnlyAnnounce,
	}

	logger := log.New()
	pm.server = srv

	capNormal := uint64(config.LightServ) * flowcontrol.FixedPointMultiplier / 100
	srv.rcNormal = flowcontrol.PieceWiseLinear{{0, 0} /*{capNormal / 10, capNormal}, */, {capNormal, capNormal}}
	// limit the serving thread count to at least 4 times the targeted average
	// capacity, allowing more paralellization in short-term load spikes but
	// still limiting the total thread count at a reasonable level
	srv.thcNormal = int(capNormal * 4 / flowcontrol.FixedPointMultiplier)
	if srv.thcNormal < 4 {
		srv.thcNormal = 4
	}
	// while processing blocks use half of the normal target capacity
	capBlockProcessing := capNormal / 2
	srv.rcBlockProcessing = flowcontrol.PieceWiseLinear{{0, 0} /*{capBlockProcessing / 10, capBlockProcessing}, */, {capBlockProcessing, capBlockProcessing}}
	// limit the serving thread count just above the targeted average capacity,
	// ensuring that block processing is minimally hindered
	srv.thcBlockProcessing = int(capBlockProcessing/flowcontrol.FixedPointMultiplier) + 1

	pm.servingQueue.setThreads(srv.thcNormal)
	srv.fcManager = flowcontrol.NewClientManager(srv.rcNormal, &mclock.System{})

	srv.totalCapacity = capNormal
	if config.LightBandwidthIn > 0 {
		srv.inSizeCostFactor = float64(srv.totalCapacity) / float64(config.LightBandwidthIn)
	}
	if config.LightBandwidthOut > 0 {
		srv.outSizeCostFactor = float64(srv.totalCapacity) / float64(config.LightBandwidthOut)
	}
	if makeCostStats {
		srv.fcCostStats = newCostStats(srv.makeCostList().decode())
	}

	chtV1SectionCount, _, _ := srv.chtIndexer.Sections() // indexer still uses LES/1 4k section size for backwards server compatibility
	chtV2SectionCount := chtV1SectionCount / (params.CHTFrequencyClient / params.CHTFrequencyServer)
	if chtV2SectionCount != 0 {
		// convert to LES/2 section
		chtLastSection := chtV2SectionCount - 1
		// convert last LES/2 section index back to LES/1 index for chtIndexer.SectionHead
		chtLastSectionV1 := (chtLastSection+1)*(params.CHTFrequencyClient/params.CHTFrequencyServer) - 1
		chtSectionHead := srv.chtIndexer.SectionHead(chtLastSectionV1)
		chtRoot := light.GetChtRoot(pm.chainDb, chtLastSectionV1, chtSectionHead)
		logger.Info("Loaded CHT", "section", chtLastSection, "head", chtSectionHead, "root", chtRoot)
	}
	bloomTrieSectionCount, _, _ := srv.bloomTrieIndexer.Sections()
	if bloomTrieSectionCount != 0 {
		bloomTrieLastSection := bloomTrieSectionCount - 1
		bloomTrieSectionHead := srv.bloomTrieIndexer.SectionHead(bloomTrieLastSection)
		bloomTrieRoot := light.GetBloomTrieRoot(pm.chainDb, bloomTrieLastSection, bloomTrieSectionHead)
		logger.Info("Loaded bloom trie", "section", bloomTrieLastSection, "head", bloomTrieSectionHead, "root", bloomTrieRoot)
	}

	srv.chtIndexer.Start(eth.BlockChain())
	srv.blockProcLoop(pm)
	srv.gcfLoop()
	return srv, nil
}

func (s *LesServer) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "les",
			Version:   "1.0",
			Service:   NewPrivateLesServerAPI(s),
			Public:    false,
		},
	}
}

func (s *LesServer) blockProcLoop(pm *ProtocolManager) {
	pm.wg.Add(1)
	procFeedback := make(chan bool, 10)
	pm.blockchain.(*core.BlockChain).SetProcFeedback(procFeedback)
	go func() {
		for {
			select {
			case processing := <-procFeedback:
				if processing {
					pm.servingQueue.setThreads(s.thcBlockProcessing)
					s.fcManager.SetRechargeCurve(s.rcBlockProcessing)
				} else {
					pm.servingQueue.setThreads(s.thcNormal)
					s.fcManager.SetRechargeCurve(s.rcNormal)
				}
			case <-pm.quitSync:
				pm.wg.Done()
				return
			}
		}
	}()
}

func (s *LesServer) Protocols() []p2p.Protocol {
	return s.makeProtocols(ServerProtocolVersions)
}

// Start starts the LES server
func (s *LesServer) Start(srvr *p2p.Server) {
	maxPeers := s.config.LightPeers
	if maxPeers > 0 {
		s.freeClientCap = s.totalCapacity / uint64(maxPeers)
		if s.freeClientCap < minCapacity {
			s.freeClientCap = minCapacity
		}
		if s.freeClientCap > 0 {
			s.defParams = flowcontrol.ServerParams{
				BufLimit:    s.freeClientCap * bufLimitRatio,
				MinRecharge: s.freeClientCap,
			}
		}
	}
	freePeers := int(s.totalCapacity / s.freeClientCap)
	if freePeers < maxPeers {
		log.Warn("Light peer count limited", "specified", maxPeers, "allowed", freePeers)
	}

	s.freeClientPool = newFreeClientPool(s.chainDb, s.freeClientCap, 10000, mclock.System{}, s.protocolManager.removePeer)
	s.priorityClientPool = newPriorityClientPool(s.freeClientCap, s.protocolManager.peers, s.freeClientPool)
	s.priorityClientPool.setLimits(maxPeers, s.totalCapacity)
	s.protocolManager.peers.notify(s.priorityClientPool)

	s.protocolManager.Start(s.config.LightPeers)
	if srvr.DiscV5 != nil {
		for _, topic := range s.lesTopics {
			topic := topic
			go func() {
				logger := log.New("topic", topic)
				logger.Info("Starting topic registration")
				defer logger.Info("Terminated topic registration")

				srvr.DiscV5.RegisterTopic(topic, s.quitSync)
			}()
		}
	}
	s.privateKey = srvr.PrivateKey
	s.protocolManager.blockLoop()
}

func (s *LesServer) SetBloomBitsIndexer(bloomIndexer *core.ChainIndexer) {
	bloomIndexer.AddChildIndexer(s.bloomTrieIndexer)
}

// Stop stops the LES service
func (s *LesServer) Stop() {
	s.chtIndexer.Close()
	// bloom trie indexer is closed by parent bloombits indexer
	if s.fcCostStats != nil {
		s.fcCostStats.printStats()
	}
	go func() {
		<-s.protocolManager.noMorePeers
	}()
	s.freeClientPool.stop()
	s.protocolManager.Stop()
}

func (s *LesServer) makeCostList() RequestCostList {
	maxCost := func(avgTime, inSize, outSize uint64) uint64 {
		globalCostFactor := s.getGlobalCostFactor()

		cost := avgTime * 2
		inSizeCost := uint64(float64(inSize) * s.inSizeCostFactor * globalCostFactor * 1.2)
		if inSizeCost > cost {
			cost = inSizeCost
		}
		outSizeCost := uint64(float64(outSize) * s.outSizeCostFactor * globalCostFactor * 1.2)
		if outSizeCost > cost {
			cost = outSizeCost
		}
		return cost
	}
	var list RequestCostList
	for code, data := range reqAvgTime {
		list = append(list, requestCostListItem{
			MsgCode:  code,
			BaseCost: maxCost(data.baseCost, reqMaxInSize[code].baseCost, reqMaxOutSize[code].baseCost),
			ReqCost:  maxCost(data.reqCost, reqMaxInSize[code].reqCost, reqMaxOutSize[code].reqCost),
		})
	}
	return list
}

const (
	gcfMinWeight      = time.Second * 10
	gcfMaxWeight      = time.Minute
	gcfUsageThreshold = 0.5
	gcfUsageTC        = time.Second
	gcfDbKey          = "_globalCostFactor"
)

type gcfUpdate struct {
	avgTime, servingTime float64
}

func (s *LesServer) gcfLoop() {
	s.protocolManager.wg.Add(1)
	var gcf, gcfUsage, gcfSum, gcfWeight float64
	lastUpdate := mclock.Now()
	expUpdate := lastUpdate

	gcf = 1
	data, _ := s.protocolManager.chainDb.Get([]byte(gcfDbKey))
	if len(data) == 16 {
		gcfSum = math.Float64frombits(binary.BigEndian.Uint64(data[0:8]))
		gcfWeight = math.Float64frombits(binary.BigEndian.Uint64(data[8:16]))
		if gcfWeight >= float64(gcfMinWeight) {
			gcf = gcfSum / gcfWeight
		}
	}
	s.globalCostFactor = gcf
	s.gcfUpdateCh = make(chan gcfUpdate, 100)

	go func() {
		for {
			select {
			case r := <-s.gcfUpdateCh:
				now := mclock.Now()
				max := r.servingTime * gcf
				if r.avgTime > max {
					max = r.avgTime
				}
				dt := float64(now - expUpdate)
				expUpdate = now
				gcfUsage = gcfUsage*math.Exp(-dt/float64(gcfUsageTC)) + max*1000000/float64(gcfUsageTC)

				if gcfUsage >= gcfUsageThreshold*float64(s.totalCapacity)*gcf {
					gcfSum += r.avgTime
					gcfWeight += r.servingTime
					if time.Duration(now-lastUpdate) > time.Second && gcfWeight >= float64(gcfMinWeight) {
						gcf = gcfSum / gcfWeight
						if gcfWeight >= float64(gcfMaxWeight) {
							gcfSum = gcf * float64(gcfMaxWeight)
							gcfWeight = float64(gcfMaxWeight)
						}
						lastUpdate = now
						s.gcfLock.Lock()
						s.globalCostFactor = gcf
						s.gcfLock.Unlock()
						log.Debug("globalCostFactor updated", "gcf", gcf, "weight", time.Duration(gcfWeight))
					}
				}
			case <-s.protocolManager.quitSync:
				var data [16]byte
				binary.BigEndian.PutUint64(data[0:8], math.Float64bits(gcfSum))
				binary.BigEndian.PutUint64(data[8:16], math.Float64bits(gcfWeight))
				s.protocolManager.chainDb.Put([]byte(gcfDbKey), data[:])
				log.Debug("globalCostFactor saved", "sum", time.Duration(gcfSum), "weight", time.Duration(gcfWeight))
				s.protocolManager.wg.Done()
				return
			}
		}
	}()
}

func (s *LesServer) getGlobalCostFactor() float64 {
	s.gcfLock.RLock()
	defer s.gcfLock.RUnlock()

	return s.globalCostFactor
}

func (s *LesServer) updateGlobalCostFactor(avgTime, servingTime uint64) {
	s.gcfUpdateCh <- gcfUpdate{float64(avgTime), float64(servingTime)}
}

type (
	requestCosts struct {
		baseCost, reqCost uint64
	}
	requestCostTable map[uint64]*requestCosts

	RequestCostList     []requestCostListItem
	requestCostListItem struct {
		MsgCode, BaseCost, ReqCost uint64
	}
)

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

func (pm *ProtocolManager) blockLoop() {
	pm.wg.Add(1)
	headCh := make(chan core.ChainHeadEvent, 10)
	headSub := pm.blockchain.SubscribeChainHeadEvent(headCh)
	go func() {
		var lastHead *types.Header
		lastBroadcastTd := common.Big0
		for {
			select {
			case ev := <-headCh:
				peers := pm.peers.AllPeers()
				if len(peers) > 0 {
					header := ev.Block.Header()
					hash := header.Hash()
					number := header.Number.Uint64()
					td := rawdb.ReadTd(pm.chainDb, hash, number)
					if td != nil && td.Cmp(lastBroadcastTd) > 0 {
						var reorg uint64
						if lastHead != nil {
							reorg = lastHead.Number.Uint64() - rawdb.FindCommonAncestor(pm.chainDb, header, lastHead).Number.Uint64()
						}
						lastHead = header
						lastBroadcastTd = td

						log.Debug("Announcing block to peers", "number", number, "hash", hash, "td", td, "reorg", reorg)

						announce := announceData{Hash: hash, Number: number, Td: td, ReorgDepth: reorg}
						var (
							signed         bool
							signedAnnounce announceData
						)

						for _, p := range peers {
							switch p.announceType {

							case announceTypeSimple:
								p.queueSend(func() { p.SendAnnounce(announce) })
							case announceTypeSigned:
								if !signed {
									signedAnnounce = announce
									signedAnnounce.sign(pm.server.privateKey)
									signed = true
								}

								p.queueSend(func() { p.SendAnnounce(signedAnnounce) })
							}
						}
					}
				}
			case <-pm.quitSync:
				headSub.Unsubscribe()
				pm.wg.Done()
				return
			}
		}
	}()
}
