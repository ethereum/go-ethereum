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
	"sync"

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
	bufLimitRatio = 6000  // fixed bufLimit/MRR ratio
	makeCostStats = false // make request cost statistics during operation
)

type LesServer struct {
	lesCommons

	fcManager    *flowcontrol.ClientManager // nil if our node is client only
	fcCostList   RequestCostList
	fcCostTable  requestCostTable
	fcCostStats  *requestCostStats
	defParams    flowcontrol.ServerParams
	lesTopics    []discv5.Topic
	privateKey   *ecdsa.PrivateKey
	quitSync     chan struct{}
	onlyAnnounce bool

	totalCapacity, minCapacity, minBufLimit, bufLimitRatio uint64
	bwcNormal, bwcBlockProcessing                          flowcontrol.PieceWiseLinear // capacity curve for normal operation and block processing mode
	thcNormal, thcBlockProcessing                          int                         // serving thread count for normal operation and block processing mode
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

	bwNormal := uint64(config.LightServ) * flowcontrol.FixedPointMultiplier / 100
	srv.bwcNormal = flowcontrol.PieceWiseLinear{{0, 0} /*{bwNormal / 10, bwNormal}, */, {bwNormal, bwNormal}}
	// limit the serving thread count to at least 4 times the targeted average
	// capacity, allowing more paralellization in short-term load spikes but
	// still limiting the total thread count at a reasonable level
	srv.thcNormal = int(bwNormal * 4 / flowcontrol.FixedPointMultiplier)
	if srv.thcNormal < 4 {
		srv.thcNormal = 4
	}
	// while processing blocks use half of the normal target capacity
	bwBlockProcessing := bwNormal / 2
	srv.bwcBlockProcessing = flowcontrol.PieceWiseLinear{{0, 0} /*{bwBlockProcessing / 10, bwBlockProcessing}, */, {bwBlockProcessing, bwBlockProcessing}}
	// limit the serving thread count just above the targeted average capacity,
	// ensuring that block processing is minimally hindered
	srv.thcBlockProcessing = int(bwBlockProcessing/flowcontrol.FixedPointMultiplier) + 1

	pm.servingQueue.setThreads(srv.thcNormal)
	srv.fcManager = flowcontrol.NewClientManager(srv.bwcNormal, &mclock.System{})

	srv.totalCapacity = bwNormal
	if config.LightBandwidthIn > 0 {
		pm.inSizeCostFactor = float64(srv.totalCapacity) / float64(config.LightBandwidthIn)
	}
	if config.LightBandwidthOut > 0 {
		pm.outSizeCostFactor = float64(srv.totalCapacity) / float64(config.LightBandwidthOut)
	}
	srv.fcCostList, srv.minBufLimit = pm.benchmarkCosts(srv.thcNormal, pm.inSizeCostFactor, pm.outSizeCostFactor)
	srv.fcCostTable = srv.fcCostList.decode()
	if makeCostStats {
		srv.fcCostStats = newCostStats(srv.fcCostTable)
	}

	srv.minCapacity = (srv.minBufLimit-1)/bufLimitRatio + 1

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
					s.fcManager.SetRechargeCurve(s.bwcBlockProcessing)
				} else {
					pm.servingQueue.setThreads(s.thcNormal)
					s.fcManager.SetRechargeCurve(s.bwcNormal)
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
