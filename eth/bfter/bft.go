package bfter

import (
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/log"
	lru "github.com/hashicorp/golang-lru"
)

const (
	messageLimit = 1024
)

//Define Boradcast Group functions
type broadcastVoteFn func(*utils.Vote)
type broadcastTimeoutFn func(*utils.Timeout)
type broadcastSyncInfoFn func(*utils.SyncInfo)

type Bfter struct {
	blockCahinReader consensus.ChainReader
	broadcastCh      chan interface{}
	quit             chan struct{}
	consensus        ConsensusFns
	broadcast        BroadcastFns

	// Message Cache
	knownVotes     *lru.ARCCache
	knownSyncInfos *lru.ARCCache
	knownTimeouts  *lru.ARCCache
}

type ConsensusFns struct {
	verifyVote  func(*utils.Vote) error
	voteHandler func(consensus.ChainReader, *utils.Vote) error

	verifyTimeout  func(*utils.Timeout) error
	timeoutHandler func(*utils.Timeout) error

	verifySyncInfo  func(*utils.SyncInfo) error
	syncInfoHandler func(*utils.SyncInfo) error
}

type BroadcastFns struct {
	Vote     broadcastVoteFn
	Timeout  broadcastTimeoutFn
	SyncInfo broadcastSyncInfoFn
}

func New(broadcasts BroadcastFns, blockCahinReader *core.BlockChain) *Bfter {
	knownVotes, _ := lru.NewARC(messageLimit)
	knownSyncInfos, _ := lru.NewARC(messageLimit)
	knownTimeouts, _ := lru.NewARC(messageLimit)
	return &Bfter{
		quit:             make(chan struct{}),
		broadcastCh:      make(chan interface{}),
		broadcast:        broadcasts,
		knownVotes:       knownVotes,
		knownSyncInfos:   knownSyncInfos,
		knownTimeouts:    knownTimeouts,
		blockCahinReader: blockCahinReader,
	}
}

func (b *Bfter) SetConsensusFuns(engine consensus.Engine) {
	e := engine.(*XDPoS.XDPoS)
	b.broadcastCh = e.EngineV2.BroadcastCh
	b.consensus = ConsensusFns{
		verifySyncInfo: e.VerifySyncInfo,
		verifyVote:     e.VerifyVote,
		verifyTimeout:  e.VerifyTimeout,

		voteHandler:     e.EngineV2.VoteHandler,
		timeoutHandler:  e.EngineV2.TimeoutHandler,
		syncInfoHandler: e.EngineV2.SyncInfoHandler,
	}
}

// TODO: rename
func (b *Bfter) Vote(vote *utils.Vote) error {
	log.Trace("Receive Vote", "vote", vote)
	if b.knownVotes.Contains(vote.Hash()) {
		log.Trace("Discarded vote, known vote", "Signature", vote.Signature, "hash", vote.Hash())
		return nil
	}

	err := b.consensus.verifyVote(vote)
	if err != nil {
		log.Error("Verify BFT Vote", "error", err)
		return err
	}
	b.knownVotes.Add(vote.Hash(), true)
	b.broadcastCh <- vote

	err = b.consensus.voteHandler(b.blockCahinReader, vote)
	if err != nil {
		log.Error("handle BFT Vote", "error", err)
		return err
	}
	return nil
}
func (b *Bfter) Timeout(timeout *utils.Timeout) error {
	log.Trace("Receive Timeout", "timeout", timeout)
	if b.knownVotes.Contains(timeout.Hash()) {
		log.Trace("Discarded Timeout, known Timeout", "Signature", timeout.Signature, "hash", timeout.Hash(), "round", timeout.Round)
		return nil
	}
	err := b.consensus.verifyTimeout(timeout)
	if err != nil {
		log.Error("Verify BFT Timeout", "error", err)
		return err
	}
	b.knownTimeouts.Add(timeout.Hash(), true)
	b.broadcastCh <- timeout

	err = b.consensus.timeoutHandler(timeout)
	if err != nil {
		log.Error("handle BFT Timeout", "error", err)
		return err
	}
	return nil
}
func (b *Bfter) SyncInfo(syncInfo *utils.SyncInfo) error {
	log.Trace("Receive SyncInfo", "syncInfo", syncInfo)
	if b.knownVotes.Contains(syncInfo.Hash()) {
		log.Trace("Discarded SyncInfo, known SyncInfo", "hash", syncInfo.Hash())
		return nil
	}
	err := b.consensus.verifySyncInfo(syncInfo)
	if err != nil {
		log.Error("Verify BFT SyncInfo", "error", err)
		return err
	}

	b.knownSyncInfos.Add(syncInfo.Hash(), true)
	b.broadcastCh <- syncInfo

	err = b.consensus.syncInfoHandler(syncInfo)
	if err != nil {
		log.Error("handle BFT SyncInfo", "error", err)
		return err
	}
	return nil
}

// Start Bft receiver
func (b *Bfter) Start() {
	go b.loop()
}
func (b *Bfter) Stop() {
	close(b.quit)
}
func (b *Bfter) loop() {
	for {
		select {
		case <-b.quit:
			return
		case obj := <-b.broadcastCh:
			switch v := obj.(type) {
			case *utils.Vote:
				go b.broadcast.Vote(v)
			case *utils.Timeout:
				go b.broadcast.Timeout(v)
			case *utils.SyncInfo:
				go b.broadcast.SyncInfo(v)
			default:
				log.Error("Unknown message type received", "value", v)
			}
		}
	}
}
