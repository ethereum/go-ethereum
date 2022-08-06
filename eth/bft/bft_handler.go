package bft

import (
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/log"
)

//Define Boradcast Group functions
type broadcastVoteFn func(*types.Vote)
type broadcastTimeoutFn func(*types.Timeout)
type broadcastSyncInfoFn func(*types.SyncInfo)

type Bfter struct {
	blockChainReader consensus.ChainReader
	broadcastCh      chan interface{}
	quit             chan struct{}
	consensus        ConsensusFns
	broadcast        BroadcastFns
}

type ConsensusFns struct {
	verifyVote  func(consensus.ChainReader, *types.Vote) (bool, error)
	voteHandler func(consensus.ChainReader, *types.Vote) error

	verifyTimeout  func(consensus.ChainReader, *types.Timeout) (bool, error)
	timeoutHandler func(consensus.ChainReader, *types.Timeout) error

	verifySyncInfo  func(consensus.ChainReader, *types.SyncInfo) (bool, error)
	syncInfoHandler func(consensus.ChainReader, *types.SyncInfo) error
}

type BroadcastFns struct {
	Vote     broadcastVoteFn
	Timeout  broadcastTimeoutFn
	SyncInfo broadcastSyncInfoFn
}

func New(broadcasts BroadcastFns, blockChainReader *core.BlockChain) *Bfter {

	return &Bfter{
		quit:             make(chan struct{}),
		broadcastCh:      make(chan interface{}),
		broadcast:        broadcasts,
		blockChainReader: blockChainReader,
	}
}

func (b *Bfter) SetConsensusFuns(engine consensus.Engine) {
	e := engine.(*XDPoS.XDPoS)
	b.broadcastCh = e.EngineV2.BroadcastCh
	b.consensus = ConsensusFns{
		verifySyncInfo: e.EngineV2.VerifySyncInfoMessage,
		verifyVote:     e.EngineV2.VerifyVoteMessage,
		verifyTimeout:  e.EngineV2.VerifyTimeoutMessage,

		voteHandler:     e.EngineV2.VoteHandler,
		timeoutHandler:  e.EngineV2.TimeoutHandler,
		syncInfoHandler: e.EngineV2.SyncInfoHandler,
	}
}

func (b *Bfter) Vote(vote *types.Vote) error {
	log.Trace("Receive Vote", "hash", vote.Hash().Hex(), "voted block hash", vote.ProposedBlockInfo.Hash.Hex(), "number", vote.ProposedBlockInfo.Number, "round", vote.ProposedBlockInfo.Round)

	verified, err := b.consensus.verifyVote(b.blockChainReader, vote)

	if err != nil {
		log.Error("Verify BFT Vote", "error", err)
		return err
	}

	b.broadcastCh <- vote

	if verified {
		err = b.consensus.voteHandler(b.blockChainReader, vote)
		if err != nil {
			if _, ok := err.(*utils.ErrIncomingMessageRoundTooFarFromCurrentRound); ok {
				log.Debug("vote round not equal", "error", err, "vote", vote.Hash())
				return err
			}
			log.Error("handle BFT Vote", "error", err)
			return err
		}
	}

	return nil
}
func (b *Bfter) Timeout(timeout *types.Timeout) error {
	log.Debug("Receive Timeout", "timeout", timeout)

	verified, err := b.consensus.verifyTimeout(b.blockChainReader, timeout)
	if err != nil {
		log.Error("Verify BFT Timeout", "timeoutRound", timeout.Round, "timeoutGapNum", timeout.GapNumber, "error", err)
		return err
	}

	b.broadcastCh <- timeout
	if verified {
		err = b.consensus.timeoutHandler(b.blockChainReader, timeout)
		if err != nil {
			if _, ok := err.(*utils.ErrIncomingMessageRoundNotEqualCurrentRound); ok {
				log.Debug("timeout round not equal", "error", err)
				return err
			}
			log.Error("handle BFT Timeout", "error", err)
			return err
		}
	}

	return nil
}
func (b *Bfter) SyncInfo(syncInfo *types.SyncInfo) error {
	log.Debug("Receive SyncInfo", "syncInfo", syncInfo)

	verified, err := b.consensus.verifySyncInfo(b.blockChainReader, syncInfo)
	if err != nil {
		log.Error("Verify BFT SyncInfo", "error", err)
		return err
	}

	b.broadcastCh <- syncInfo
	// Process only if verified and qualified
	if verified {
		err = b.consensus.syncInfoHandler(b.blockChainReader, syncInfo)
		if err != nil {
			log.Error("handle BFT SyncInfo", "error", err)
			return err
		}
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
			case *types.Vote:
				go b.broadcast.Vote(v)
			case *types.Timeout:
				go b.broadcast.Timeout(v)
			case *types.SyncInfo:
				go b.broadcast.SyncInfo(v)
			default:
				log.Error("Unknown message type received", "value", v)
			}
		}
	}
}
