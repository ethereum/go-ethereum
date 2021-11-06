package bfter

import (
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/log"
	lru "github.com/hashicorp/golang-lru"
)

const (
	messageLimit = 1024
)

//Define Verify Group functions
type VerifySyncInfoFn func(utils.SyncInfo) error
type VerifyVoteFn func(utils.Vote) error
type VerifyTimeoutFn func(utils.Timeout) error

//Define Boradcast Group functions
type broadcastVoteFn func(utils.Vote)
type broadcastTimeoutFn func(utils.Timeout)
type broadcastSyncInfoFn func(utils.SyncInfo)

type Bfter struct {
	broadcastCh chan interface{}
	quit        chan struct{}
	consensus   ConsensusFns
	broadcast   BroadcastFns

	// Message Cache
	knownVotes     *lru.ARCCache
	knownSyncInfos *lru.ARCCache
	knownTimeouts  *lru.ARCCache
}

type ConsensusFns struct {
	verifySyncInfo VerifySyncInfoFn
	verifyVote     VerifyVoteFn
	verifyTimeout  VerifyTimeoutFn
}

type BroadcastFns struct {
	Vote     broadcastVoteFn
	Timeout  broadcastTimeoutFn
	SyncInfo broadcastSyncInfoFn
}

func New(broadcasts BroadcastFns) *Bfter {
	knownVotes, _ := lru.NewARC(messageLimit)
	knownSyncInfos, _ := lru.NewARC(messageLimit)
	knownTimeouts, _ := lru.NewARC(messageLimit)
	return &Bfter{
		quit:           make(chan struct{}),
		broadcastCh:    make(chan interface{}),
		broadcast:      broadcasts,
		knownVotes:     knownVotes,
		knownSyncInfos: knownSyncInfos,
		knownTimeouts:  knownTimeouts,
	}
}

func (b *Bfter) SetConsensusFuns(engine consensus.Engine) {
	e := engine.(*XDPoS.XDPoS)
	b.broadcastCh = e.EngineV2.BroadcastCh
	b.consensus = ConsensusFns{
		verifySyncInfo: e.VerifySyncInfo,
		verifyVote:     e.VerifyVote,
		verifyTimeout:  e.VerifyTimeout,
	}
}

// TODO: rename
func (b *Bfter) Vote(vote utils.Vote) {
	log.Trace("Receive Vote", "vote", vote)

	if b.knownVotes.Contains(vote.Hash()) {
		log.Trace("Discarded vote, known vote", "Signature", vote.Signature, "hash", vote.Hash())
		return
	}

	err := b.consensus.verifyVote(vote)
	if err != nil {
		log.Error("Verify BFT Vote", "error", err)
		return
	}

	b.knownVotes.Add(vote.Hash(), true)
	b.broadcastCh <- vote
}

func (b *Bfter) Timeout(timeout utils.Timeout) {
	log.Trace("Receive Timeout", "timeout", timeout)

	if b.knownVotes.Contains(timeout.Hash()) {
		log.Trace("Discarded Timeout, known Timeout", "Signature", timeout.Signature, "hash", timeout.Hash(), "round", timeout.Round)
		return
	}

	err := b.consensus.verifyTimeout(timeout)
	if err != nil {
		log.Error("Verify BFT Timeout", "error", err)
		return
	}

	b.knownTimeouts.Add(timeout.Hash(), true)
	b.broadcastCh <- timeout
}

func (b *Bfter) SyncInfo(syncInfo utils.SyncInfo) {
	log.Trace("Receive SyncInfo", "syncInfo", syncInfo)

	if b.knownVotes.Contains(syncInfo.Hash()) {
		log.Trace("Discarded SyncInfo, known SyncInfo", "hash", syncInfo.Hash())
		return
	}

	err := b.consensus.verifySyncInfo(syncInfo)
	if err != nil {
		log.Error("Verify BFT SyncInfo", "error", err)
		return
	}

	b.knownSyncInfos.Add(syncInfo.Hash(), true)
	b.broadcastCh <- syncInfo
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
			case utils.Vote:
				go b.broadcast.Vote(v)
			case utils.Timeout:
				go b.broadcast.Timeout(v)
			case utils.SyncInfo:
				go b.broadcast.SyncInfo(v)
			default:
				log.Error("Unknown message type received, value: %v", v)
			}
		}
	}
}
