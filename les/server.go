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

const bufLimitRatio = 6000 // fixed bufLimit/MRR ratio

type LesServer struct {
	lesCommons

	fcManager    *flowcontrol.ClientManager // nil if our node is client only
	costTracker  *costTracker
	defParams    flowcontrol.ServerParams
	lesTopics    []discv5.Topic
	privateKey   *ecdsa.PrivateKey
	quitSync     chan struct{}
	onlyAnnounce bool

	thcNormal, thcBlockProcessing int // serving thread count for normal operation and block processing mode

	maxPeers           int
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
			chtIndexer:       light.NewChtIndexer(eth.ChainDb(), nil, params.CHTFrequency, params.HelperTrieProcessConfirmations),
			bloomTrieIndexer: light.NewBloomTrieIndexer(eth.ChainDb(), nil, params.BloomBitsBlocks, params.BloomTrieFrequency),
			protocolManager:  pm,
		},
		costTracker:  newCostTracker(eth.ChainDb(), config),
		quitSync:     quitSync,
		lesTopics:    lesTopics,
		onlyAnnounce: config.OnlyAnnounce,
	}

	logger := log.New()
	pm.server = srv
	srv.thcNormal = config.LightServ * 4 / 100
	if srv.thcNormal < 4 {
		srv.thcNormal = 4
	}
	srv.thcBlockProcessing = config.LightServ/100 + 1
	srv.fcManager = flowcontrol.NewClientManager(nil, &mclock.System{})

	chtSectionCount, _, _ := srv.chtIndexer.Sections()
	if chtSectionCount != 0 {
		chtLastSection := chtSectionCount - 1
		chtSectionHead := srv.chtIndexer.SectionHead(chtLastSection)
		chtRoot := light.GetChtRoot(pm.chainDb, chtLastSection, chtSectionHead)
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
	return srv, nil
}

func (s *LesServer) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "les",
			Version:   "1.0",
			Service:   NewPrivateLightServerAPI(s),
			Public:    false,
		},
	}
}

// startEventLoop starts an event handler loop that updates the recharge curve of
// the client manager and adjusts the client pool's size according to the total
// capacity updates coming from the client manager
func (s *LesServer) startEventLoop() {
	s.protocolManager.wg.Add(1)

	var processing bool
	blockProcFeed := make(chan bool, 100)
	s.protocolManager.blockchain.(*core.BlockChain).SubscribeBlockProcessingEvent(blockProcFeed)
	totalRechargeCh := make(chan uint64, 100)
	totalRecharge := s.costTracker.subscribeTotalRecharge(totalRechargeCh)
	totalCapacityCh := make(chan uint64, 100)
	updateRecharge := func() {
		if processing {
			s.protocolManager.servingQueue.setThreads(s.thcBlockProcessing)
			s.fcManager.SetRechargeCurve(flowcontrol.PieceWiseLinear{{0, 0}, {totalRecharge, totalRecharge}})
		} else {
			s.protocolManager.servingQueue.setThreads(s.thcNormal)
			s.fcManager.SetRechargeCurve(flowcontrol.PieceWiseLinear{{0, 0}, {totalRecharge / 10, totalRecharge}, {totalRecharge, totalRecharge}})
		}
	}
	updateRecharge()
	totalCapacity := s.fcManager.SubscribeTotalCapacity(totalCapacityCh)
	s.priorityClientPool.setLimits(s.maxPeers, totalCapacity)

	go func() {
		for {
			select {
			case processing = <-blockProcFeed:
				updateRecharge()
			case totalRecharge = <-totalRechargeCh:
				updateRecharge()
			case totalCapacity = <-totalCapacityCh:
				s.priorityClientPool.setLimits(s.maxPeers, totalCapacity)
			case <-s.protocolManager.quitSync:
				s.protocolManager.wg.Done()
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
	s.maxPeers = s.config.LightPeers
	totalRecharge := s.costTracker.totalRecharge()
	if s.maxPeers > 0 {
		s.freeClientCap = minCapacity //totalRecharge / uint64(s.maxPeers)
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
	freePeers := int(totalRecharge / s.freeClientCap)
	if freePeers < s.maxPeers {
		log.Warn("Light peer count limited", "specified", s.maxPeers, "allowed", freePeers)
	}

	s.freeClientPool = newFreeClientPool(s.chainDb, s.freeClientCap, 10000, mclock.System{}, func(id string) { go s.protocolManager.removePeer(id) })
	s.priorityClientPool = newPriorityClientPool(s.freeClientCap, s.protocolManager.peers, s.freeClientPool)

	s.protocolManager.peers.notify(s.priorityClientPool)
	s.startEventLoop()
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
	go func() {
		<-s.protocolManager.noMorePeers
	}()
	s.freeClientPool.stop()
	s.costTracker.stop()
	s.protocolManager.Stop()
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
							p := p
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
