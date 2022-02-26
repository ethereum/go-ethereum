package bft

import (
	"fmt"

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
	blockChainReader consensus.ChainReader
	broadcastCh      chan interface{}
	quit             chan struct{}
	consensus        ConsensusFns
	broadcast        BroadcastFns

	// Message Cache
	knownVotes     *lru.Cache
	knownSyncInfos *lru.Cache
	knownTimeouts  *lru.Cache
}

type ConsensusFns struct {
	verifyVote  func(chain consensus.ChainReader, vote *utils.Vote) (bool, error)
	voteHandler func(consensus.ChainReader, *utils.Vote) error

	verifyTimeout  func(*utils.Timeout) error
	timeoutHandler func(*utils.Timeout) error

	verifySyncInfo  func(*utils.SyncInfo) error
	syncInfoHandler func(consensus.ChainReader, *utils.SyncInfo) error
}

type BroadcastFns struct {
	Vote     broadcastVoteFn
	Timeout  broadcastTimeoutFn
	SyncInfo broadcastSyncInfoFn
}

func New(broadcasts BroadcastFns, blockChainReader *core.BlockChain) *Bfter {
	knownVotes, _ := lru.New(messageLimit)
	knownSyncInfos, _ := lru.New(messageLimit)
	knownTimeouts, _ := lru.New(messageLimit)
	return &Bfter{
		quit:             make(chan struct{}),
		broadcastCh:      make(chan interface{}),
		broadcast:        broadcasts,
		knownVotes:       knownVotes,
		knownSyncInfos:   knownSyncInfos,
		knownTimeouts:    knownTimeouts,
		blockChainReader: blockChainReader,
	}
}

func (b *Bfter) SetConsensusFuns(engine consensus.Engine) {
	e := engine.(*XDPoS.XDPoS)
	b.broadcastCh = e.EngineV2.BroadcastCh
	b.consensus = ConsensusFns{
		verifySyncInfo: e.VerifySyncInfo,
		verifyVote:     e.EngineV2.VerifyVoteMessage,
		verifyTimeout:  e.VerifyTimeout,

		voteHandler:     e.EngineV2.VoteHandler,
		timeoutHandler:  e.EngineV2.TimeoutHandler,
		syncInfoHandler: e.EngineV2.SyncInfoHandler,
	}
}

// TODO: rename
func (b *Bfter) Vote(vote *utils.Vote) error {
	log.Trace("Receive Vote", "hash", vote.Hash(), "voted block hash", vote.ProposedBlockInfo.Hash.Hex(), "number", vote.ProposedBlockInfo.Number, "round", vote.ProposedBlockInfo.Round, "signature", vote.Signature)
	if exist, _ := b.knownVotes.ContainsOrAdd(vote.Hash(), true); exist {
		log.Info("Discarded vote, known vote", "vote hash", vote.Hash(), "voted block hash", vote.ProposedBlockInfo.Hash.Hex(), "number", vote.ProposedBlockInfo.Number, "round", vote.ProposedBlockInfo.Round)
		return nil
	}

	verified, err := b.consensus.verifyVote(b.blockChainReader, vote)

	if err != nil || !verified {
		log.Error("Verify BFT Vote", "error", err, "verified", verified)
		if !verified {
			return fmt.Errorf("Fail to verify vote")
		}
		return err
	}

	b.broadcastCh <- vote

	err = b.consensus.voteHandler(b.blockChainReader, vote)
	if err != nil {
		if _, ok := err.(*utils.ErrIncomingMessageRoundTooFarFromCurrentRound); ok {
			log.Warn("vote round not equal", "error", err, "vote", vote.Hash())
			return err
		}
		log.Error("handle BFT Vote", "error", err)
		return err
	}
	return nil
}
func (b *Bfter) Timeout(timeout *utils.Timeout) error {
	log.Trace("Receive Timeout", "timeout", timeout)
	if exist, _ := b.knownTimeouts.ContainsOrAdd(timeout.Hash(), true); exist {
		log.Trace("Discarded Timeout, known Timeout", "Signature", timeout.Signature, "hash", timeout.Hash(), "round", timeout.Round)
		return nil
	}
	err := b.consensus.verifyTimeout(timeout)
	if err != nil {
		log.Error("Verify BFT Timeout", "error", err)
		return err
	}
	b.broadcastCh <- timeout

	err = b.consensus.timeoutHandler(timeout)
	if err != nil {
		if _, ok := err.(*utils.ErrIncomingMessageRoundNotEqualCurrentRound); ok {
			log.Warn("timeout round not equal", "error", err)
			return err
		}
		log.Error("handle BFT Timeout", "error", err)
		return err
	}
	return nil
}
func (b *Bfter) SyncInfo(syncInfo *utils.SyncInfo) error {
	log.Trace("Receive SyncInfo", "syncInfo", syncInfo)
	if exist, _ := b.knownSyncInfos.ContainsOrAdd(syncInfo.Hash(), true); exist {
		log.Trace("Discarded SyncInfo, known SyncInfo", "hash", syncInfo.Hash())
		return nil
	}
	err := b.consensus.verifySyncInfo(syncInfo)
	if err != nil {
		log.Error("Verify BFT SyncInfo", "error", err)
		return err
	}

	b.broadcastCh <- syncInfo

	err = b.consensus.syncInfoHandler(b.blockChainReader, syncInfo)
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
