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
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/registrar"
	"github.com/ethereum/go-ethereum/contracts/registrar/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/les/flowcontrol"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/rpc"
)

// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
const SubscribeChainHeadEvent = 10

type LesServer struct {
	config          *eth.Config
	backend         *eth.EthAPIBackend
	chaindb         ethdb.Database
	protocolManager *ProtocolManager
	fcManager       *flowcontrol.ClientManager // nil if our node is client only
	fcCostStats     *requestCostStats
	defParams       *flowcontrol.ServerParams
	lesTopics       []discv5.Topic
	privateKey      *ecdsa.PrivateKey
	quitSync        chan struct{}

	// Checkpoint relative fields
	registrar        *registrar.Registrar     // Handler for checkpoint contract
	stableCheckpoint *light.TrustedCheckpoint // The nearest stable checkpoint

	// Indexers
	chtIndexer       *core.ChainIndexer // Indexers for creating cht root for each block section
	bloomTrieIndexer *core.ChainIndexer // Indexers for creating bloom trie root for each block section
}

func NewLesServer(e *eth.Ethereum, config *eth.Config) (*LesServer, error) {
	quitSync := make(chan struct{})
	pm, err := NewProtocolManager(e.BlockChain().Config(), false, ServerProtocolVersions, config.NetworkId, e.EventMux(), e.Engine(), newPeerSet(), e.BlockChain(), e.TxPool(), e.ChainDb(), nil, nil, nil, quitSync, new(sync.WaitGroup))
	if err != nil {
		return nil, err
	}

	lesTopics := make([]discv5.Topic, len(AdvertiseProtocolVersions))
	for i, pv := range AdvertiseProtocolVersions {
		lesTopics[i] = lesTopic(e.BlockChain().Genesis().Hash(), pv)
	}

	srv := &LesServer{
		config:           config,
		backend:          e.APIBackend,
		chaindb:          e.ChainDb(),
		protocolManager:  pm,
		quitSync:         quitSync,
		lesTopics:        lesTopics,
		chtIndexer:       light.NewChtIndexer(e.ChainDb(), false),
		bloomTrieIndexer: light.NewBloomTrieIndexer(e.ChainDb(), false),
	}
	logger := log.New()

	chtV1SectionCount, _, _ := srv.chtIndexer.Sections() // indexer still uses LES/1 4k section size for backwards server compatibility
	chtV2SectionCount := chtV1SectionCount / (light.CHTFrequencyClient / light.CHTFrequencyServer)
	if chtV2SectionCount != 0 {
		// convert to LES/2 section
		chtLastSection := chtV2SectionCount - 1
		// convert last LES/2 section index back to LES/1 index for chtIndexer.SectionHead
		chtLastSectionV1 := (chtLastSection+1)*(light.CHTFrequencyClient/light.CHTFrequencyServer) - 1
		chtSectionHead := srv.chtIndexer.SectionHead(chtLastSectionV1)
		chtRoot := light.GetChtV2Root(srv.chaindb, chtLastSection, chtSectionHead)
		logger.Info("Loaded CHT", "section", chtLastSection, "head", chtSectionHead, "root", chtRoot)
	}
	bloomTrieSectionCount, _, _ := srv.bloomTrieIndexer.Sections()
	if bloomTrieSectionCount != 0 {
		bloomTrieLastSection := bloomTrieSectionCount - 1
		bloomTrieSectionHead := srv.bloomTrieIndexer.SectionHead(bloomTrieLastSection)
		bloomTrieRoot := light.GetBloomTrieRoot(srv.chaindb, bloomTrieLastSection, bloomTrieSectionHead)
		logger.Info("Loaded bloom trie", "section", bloomTrieLastSection, "head", bloomTrieSectionHead, "root", bloomTrieRoot)
	}

	srv.chtIndexer.Start(e.BlockChain())
	pm.server = srv

	srv.defParams = &flowcontrol.ServerParams{
		BufLimit:    300000000,
		MinRecharge: 50000,
	}
	srv.fcManager = flowcontrol.NewClientManager(uint64(config.LightServ), 10, 1000000000)
	srv.fcCostStats = newCostStats(e.ChainDb())
	if addr, ok := registrar.RegistrarAddr[e.BlockChain().Genesis().Hash()]; ok {
		registrar, err := registrar.NewRegistrar(addr, eth.NewContractBackend(e.APIBackend, e.APIBackend, false))
		if err != nil {
			return nil, err
		}
		srv.registrar = registrar
	}
	return srv, nil
}

func (s *LesServer) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
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
	if s.registrar != nil {
		s.stableCheckpoint = s.recoverCheckpoint()
		go s.checkpointLoop()
	}
}

func (s *LesServer) SetBloomBitsIndexer(bloomIndexer *core.ChainIndexer) {
	bloomIndexer.AddChildIndexer(s.bloomTrieIndexer)
}

// Stop stops the LES service
func (s *LesServer) Stop() {
	s.chtIndexer.Close()
	// bloom trie indexer is closed by parent bloombits indexer
	s.fcCostStats.store()
	s.fcManager.Stop()
	go func() {
		<-s.protocolManager.noMorePeers
	}()
	s.protocolManager.Stop()
}

// APIs implements LesServer, returns all API service provided by les server.
func (s *LesServer) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "les",
			Version:   "1.0",
			Service:   NewPublicLesServerAPI(s),
			Public:    true,
		},
	}
}

// latestCheckpoint finds the common stored section index and returns a set of
// post-processed trie roots (CHT and BloomTrie) associated with
// the appropriate section index and head hash as a checkpoint package.
//
// Note for cht, the section size in LES1 is 4K, so indexer still uses LES/1
// 4k section size for backwards server compatibility. For bloomTrie, the size
// of the section used for indexer is 32K.
func (s *LesServer) latestCheckpoint() (uint64, common.Hash, common.Hash, common.Hash) {
	chtCount, _, _ := s.chtIndexer.Sections()
	bloomTrieCount, _, _ := s.bloomTrieIndexer.Sections()
	count := chtCount / (light.CHTFrequencyClient / light.CHTFrequencyServer)
	// Cap the section index if the two sections are not consistent.
	if count > bloomTrieCount {
		count = bloomTrieCount
	}
	if count == 0 {
		// No checkpoint information can be provided.
		return 0, common.Hash{}, common.Hash{}, common.Hash{}
	}
	sectionHead, chtRoot, bloomTrieRoot := s.getCheckpoint(count - 1)
	return count - 1, sectionHead, chtRoot, bloomTrieRoot
}

// getCheckpoint returns a set of post-processed trie roots (CHT and BloomTrie)
// associated with the appropriate head hash by specific section index.
func (s *LesServer) getCheckpoint(index uint64) (common.Hash, common.Hash, common.Hash) {
	// convert last LES/2 section index back to LES/1 index for chtIndexer.SectionHead
	latest := (index+1)*(light.CHTFrequencyClient/light.CHTFrequencyServer) - 1

	sectionHead := s.chtIndexer.SectionHead(latest)
	chtRoot := light.GetChtRoot(s.protocolManager.chainDb, latest, sectionHead)
	bloomTrieRoot := light.GetBloomTrieRoot(s.protocolManager.chainDb, index, sectionHead)
	return sectionHead, chtRoot, bloomTrieRoot
}

// checkpointLoop starts a standalone goroutine to watch new checkpoint events and updates local's stable checkpoint.
func (s *LesServer) checkpointLoop() (err error) {
	var (
		eventCh      = make(chan *contract.ContractNewCheckpointEvent)
		headCh       = make(chan core.ChainHeadEvent, SubscribeChainHeadEvent)
		announcement = make(map[uint64]common.Hash)
	)
	eventSub, err := s.registrar.WatchNewCheckpointEvent(eventCh)
	if err != nil {
		return err
	}
	headSub := s.backend.SubscribeChainHeadEvent(headCh)
	if headSub == nil {
		eventSub.Unsubscribe()
		return errors.New("subscribe head event failed")
	}
	defer func() {
		eventSub.Unsubscribe()
		headSub.Unsubscribe()
	}()

	for {
		select {
		case event := <-eventCh:
			if event == nil {
				// This should never happen.
				log.Info("Ignore empty checkpoint event")
				continue
			}
			// Note several duplicate events may be received because of chain reorg and the modification of the latest checkpoint.
			if s.stableCheckpoint == nil || event.Index.Uint64() >= s.stableCheckpoint.SectionIdx {
				log.Info("Receive new checkpoint event", "section", event.Index, "hash", common.Hash(event.CheckpointHash).Hex(),
					"grantor", event.Grantor.Hex())
				announcement[event.Index.Uint64()] = common.Hash(event.CheckpointHash)
			}
		case head := <-headCh:
			number := head.Block.NumberU64()
			if number < light.CheckpointConfirmations+light.CheckpointFrequency {
				continue
			}
			idx := (number-light.CheckpointConfirmations)/light.CheckpointFrequency - 1
			if s.stableCheckpoint == nil || idx > s.stableCheckpoint.SectionIdx {
				hash, ok := announcement[idx]
				if !ok {
					continue
				}
				sectionHead := s.bloomTrieIndexer.SectionHead(idx)
				checkpoint := &light.TrustedCheckpoint{
					SectionIdx:    idx,
					SectionHead:   sectionHead,
					ChtRoot:       light.GetChtV2Root(s.chaindb, idx, sectionHead),
					BloomTrieRoot: light.GetBloomTrieRoot(s.chaindb, idx, sectionHead),
				}
				if checkpoint.HashEqual(common.Hash(hash)) {
					light.WriteTrustedCheckpoint(s.chaindb, checkpoint)
					s.stableCheckpoint = checkpoint
					log.Info("Update stable checkpoint", "section", checkpoint.SectionIdx)
					delete(announcement, idx)
				}
			}
		case <-s.quitSync:
			// Les server is closed.
			return
		}
	}
}

// recoveryCheckpoint filters checkpoint announcement events and recovers stable checkpoint.
func (s *LesServer) recoverCheckpoint() *light.TrustedCheckpoint {
	var (
		sectionCnt, _, _ = s.bloomTrieIndexer.Sections()
		stable           = light.ReadTrustedCheckpoint(s.chaindb)
		unstableIdx      = sectionCnt - 1
		headHash         = rawdb.ReadHeadHeaderHash(s.chaindb)
		headNumber       = rawdb.ReadHeaderNumber(s.chaindb, headHash)
	)
	if headNumber == nil {
		return nil
	}
	for stable == nil || stable.SectionIdx < unstableIdx {
		if (unstableIdx+1)*light.CheckpointFrequency+light.CheckpointConfirmations <= *headNumber {
			iter, err := s.registrar.FilterNewCheckpointEvent(*headNumber, unstableIdx, light.CheckpointFrequency, light.CheckpointProcessConfirmations)
			if err == nil {
				for iter.Next() {
					sectionHead := s.bloomTrieIndexer.SectionHead(unstableIdx)
					checkpoint := &light.TrustedCheckpoint{
						SectionIdx:    unstableIdx,
						SectionHead:   sectionHead,
						ChtRoot:       light.GetChtV2Root(s.chaindb, unstableIdx, sectionHead),
						BloomTrieRoot: light.GetBloomTrieRoot(s.chaindb, unstableIdx, sectionHead),
					}
					if checkpoint.HashEqual(common.Hash(iter.Event.CheckpointHash)) {
						light.WriteTrustedCheckpoint(s.chaindb, checkpoint)
						iter.Close()
						log.Info("Recover checkpoint", "index", checkpoint.SectionIdx)
						return checkpoint
					}
				}
				iter.Close()
			}
		}
		if unstableIdx == 0 {
			break
		}
		unstableIdx -= 1
	}
	if stable == nil {
		log.Info("No stable checkpoint")
	} else {
		log.Info("Recover checkpoint", "index", stable.SectionIdx)
	}
	return stable
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
								select {
								case p.announceChn <- announce:
								default:
									pm.removePeer(p.id)
								}

							case announceTypeSigned:
								if !signed {
									signedAnnounce = announce
									signedAnnounce.sign(pm.server.privateKey)
									signed = true
								}

								select {
								case p.announceChn <- signedAnnounce:
								default:
									pm.removePeer(p.id)
								}
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
