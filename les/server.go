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
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/les/csvlogger"
	"github.com/ethereum/go-ethereum/les/flowcontrol"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	logFileName          = ""    // csv log file name (disabled if empty)
	logClientPoolMetrics = true  // log client pool metrics
	logClientPoolEvents  = false // detailed client pool event logging

	logRequestServing  = true // log request serving metrics and events
	logBlockProcEvents = true // log block processing events
	logHandlerEvents   = true // log protocol handler events
)

type LesServer struct {
	lesCommons

	handler    *serverHandler
	lesTopics  []discv5.Topic
	privateKey *ecdsa.PrivateKey

	// Flow control and capacity management
	fcManager          *flowcontrol.ClientManager
	costTracker        *costTracker
	defParams          flowcontrol.ServerParams
	servingQueue       *servingQueue
	freeClientPool     *freeClientPool
	priorityClientPool *priorityClientPool

	minCapacity   uint64
	freeClientCap uint64

	threadsIdle int // Request serving threads count when system is idle.
	threadsBusy int // Request serving threads count when system is busy(block insertion).

	logger *csvlogger.Logger // System metrics and events logger
}

func NewLesServer(e *eth.Ethereum, config *eth.Config) (*LesServer, error) {
	// Collect les protocol version information supported by local node.
	lesTopics := make([]discv5.Topic, len(AdvertiseProtocolVersions))
	for i, pv := range AdvertiseProtocolVersions {
		lesTopics[i] = lesTopic(e.BlockChain().Genesis().Hash(), pv)
	}
	// Calculate the number of threads used to service the light client
	// requests based on the user-specified value.
	threads := config.LightServ * 4 / 100
	if threads < 4 {
		threads = 4
	}
	csvLogger := csvlogger.NewLogger(logFileName, time.Second*10, "event, peerId")
	reqLogger := csvLogger
	if !logRequestServing {
		reqLogger = nil
	}
	srv := &LesServer{
		lesCommons: lesCommons{
			genesis:          e.BlockChain().Genesis().Hash(),
			config:           config,
			chainConfig:      e.BlockChain().Config(),
			iConfig:          light.DefaultServerIndexerConfig,
			chainDb:          e.ChainDb(),
			peers:            newPeerSet(false),
			chainReader:      e.BlockChain(),
			chtIndexer:       light.NewChtIndexer(e.ChainDb(), nil, params.CHTFrequency, params.HelperTrieProcessConfirmations),
			bloomTrieIndexer: light.NewBloomTrieIndexer(e.ChainDb(), nil, params.BloomBitsBlocks, params.BloomTrieFrequency),
			closeCh:          make(chan struct{}),
		},
		lesTopics:    lesTopics,
		fcManager:    flowcontrol.NewClientManager(nil, &mclock.System{}),
		servingQueue: newServingQueue(int64(time.Millisecond*10), float64(config.LightServ)/100, reqLogger),
		threadsBusy:  config.LightServ/100 + 1,
		threadsIdle:  threads,
		logger:       csvLogger,
	}
	handlerLog := csvLogger
	if !logHandlerEvents {
		handlerLog = nil
	}
	srv.handler = newServerHandler(srv, e.BlockChain(), e.ChainDb(), e.TxPool(), handlerLog, e.Synced)
	srv.costTracker, srv.minCapacity = newCostTracker(e.ChainDb(), config, reqLogger)
	srv.registrar = newCheckpointRegistrar(srv.chainConfig.CheckpointContract, srv.localCheckpoint)

	// Initialize server capacity management fields.
	srv.freeClientCap = srv.minCapacity
	srv.defParams = flowcontrol.ServerParams{
		BufLimit:    srv.freeClientCap * bufLimitRatio,
		MinRecharge: srv.freeClientCap,
	}
	// LES flow control tries to more or less guarantee the possibility for the
	// clients to send a certain amount of requests at any time and get a quick
	// response. Most of the clients want this guarantee but don't actually need
	// to send requests most of the time. Our goal is to serve as many clients as
	// possible while the actually used server capacity does not exceed the limits
	totalRecharge := srv.costTracker.totalRecharge()
	maxCapacity := srv.freeClientCap * uint64(srv.config.LightPeers)
	if totalRecharge > maxCapacity {
		maxCapacity = totalRecharge
	}
	srv.fcManager.SetCapacityLimits(srv.freeClientCap, maxCapacity, srv.freeClientCap*2)

	metricsLogger, eventLogger := csvLogger, csvLogger
	if !logClientPoolMetrics {
		metricsLogger = nil
	}
	if !logClientPoolEvents {
		eventLogger = nil
	}
	srv.freeClientPool = newFreeClientPool(srv.chainDb, srv.freeClientCap, 10000, mclock.System{}, func(id string) { go srv.peers.unregister(id) }, eventLogger, metricsLogger)
	srv.priorityClientPool = newPriorityClientPool(srv.freeClientCap, srv.peers, srv.freeClientPool, eventLogger, metricsLogger)

	checkpoint := srv.latestLocalCheckpoint()
	if !checkpoint.Empty() {
		log.Info("Loaded latest checkpoint", "section", checkpoint.SectionIndex, "head", checkpoint.SectionHead,
			"chtroot", checkpoint.CHTRoot, "bloomroot", checkpoint.BloomRoot)
	}
	srv.chtIndexer.Start(e.BlockChain())
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
		{
			Namespace: "les",
			Version:   "1.0",
			Service:   NewPrivateLightAPI(&s.lesCommons),
			Public:    false,
		},
	}
}

func (s *LesServer) Protocols() []p2p.Protocol {
	return s.makeProtocols(ServerProtocolVersions, s.handler.runPeer, func(id enode.ID) interface{} {
		if p := s.peers.clientPeer(fmt.Sprintf("%x", id.Bytes())); p != nil {
			return p.Info()
		}
		return nil
	})
}

// Start starts the LES server
func (s *LesServer) Start(srvr *p2p.Server) {
	s.privateKey = srvr.PrivateKey

	s.logger.Start()
	s.handler.start()

	s.wg.Add(1)
	go s.capacityManagement()

	if srvr.DiscV5 != nil {
		for _, topic := range s.lesTopics {
			topic := topic
			go func() {
				logger := log.New("topic", topic)
				logger.Info("Starting topic registration")
				defer logger.Info("Terminated topic registration")

				srvr.DiscV5.RegisterTopic(topic, s.closeCh)
			}()
		}
	}
}

// Stop stops the LES service
func (s *LesServer) Stop() {
	close(s.closeCh)

	// Disconnect existing sessions.
	// This also closes the gate for any new registrations on the peer set.
	// sessions which are already established but not added to pm.peers yet
	// will exit when they try to register.
	s.peers.close()

	s.fcManager.Stop()
	s.freeClientPool.stop()
	s.costTracker.stop()
	s.handler.stop()
	s.servingQueue.stop()
	s.logger.Stop()

	// Note, bloom trie indexer is closed by parent bloombits indexer.
	s.chtIndexer.Close()
	s.wg.Wait()
	log.Info("Les server stopped")
}

func (s *LesServer) SetBloomBitsIndexer(bloomIndexer *core.ChainIndexer) {
	bloomIndexer.AddChildIndexer(s.bloomTrieIndexer)
}

// SetClient sets the rpc client and starts running checkpoint contract if it is not yet watched.
func (s *LesServer) SetContractBackend(backend bind.ContractBackend) {
	if s.registrar != nil {
		s.registrar.start(backend)
	}
}

// capacityManagement starts an event handler loop that updates the recharge curve of
// the client manager and adjusts the client pool's size according to the total
// capacity updates coming from the client manager
func (s *LesServer) capacityManagement() {
	defer s.wg.Done()

	processCh := make(chan bool, 100)
	sub := s.handler.blockchain.SubscribeBlockProcessingEvent(processCh)
	defer sub.Unsubscribe()

	totalRechargeCh := make(chan uint64, 100)
	totalRecharge := s.costTracker.subscribeTotalRecharge(totalRechargeCh)

	totalCapacityCh := make(chan uint64, 100)
	totalCapacity := s.fcManager.SubscribeTotalCapacity(totalCapacityCh)
	s.priorityClientPool.setLimits(s.config.LightPeers, totalCapacity)

	var (
		busy      bool
		freePeers uint64
	)
	updateRecharge := func() {
		if busy {
			if logBlockProcEvents {
				s.logger.Event("block processing started")
			}
			s.servingQueue.setThreads(s.threadsBusy)
			s.fcManager.SetRechargeCurve(flowcontrol.PieceWiseLinear{{0, 0}, {totalRecharge, totalRecharge}})
		} else {
			if logBlockProcEvents {
				s.logger.Event("block processing finished")
			}
			s.servingQueue.setThreads(s.threadsIdle)
			s.fcManager.SetRechargeCurve(flowcontrol.PieceWiseLinear{{0, 0}, {totalRecharge / 10, totalRecharge}, {totalRecharge, totalRecharge}})
		}
	}
	updateRecharge()

	// Record the change of total capacity if required.
	var totalCapChannel *csvlogger.Channel
	if logRequestServing {
		totalCapChannel = s.logger.NewChannel("totalCapacity", 0.01)
	}
	for {
		select {
		case busy = <-processCh:
			updateRecharge()
		case totalRecharge = <-totalRechargeCh:
			updateRecharge()
		case totalCapacity = <-totalCapacityCh:
			totalCapChannel.Update(float64(totalCapacity))

			newFreePeers := totalCapacity / s.freeClientCap
			if newFreePeers < freePeers && newFreePeers < uint64(s.config.LightPeers) {
				log.Warn("Reduced free peer connections", "from", freePeers, "to", newFreePeers)
			}
			freePeers = newFreePeers
			s.priorityClientPool.setLimits(s.config.LightPeers, totalCapacity)
		case <-s.closeCh:
			return
		}
	}
}
