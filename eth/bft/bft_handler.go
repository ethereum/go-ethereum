package bft

import (
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/log"
)

const maxBlockDist = 7 // Maximum allowed backward distance from the chain head, 7 is just a magic number indicate very close block

// Define Boradcast Group functions
type broadcastVoteFn func(*types.Vote)
type broadcastTimeoutFn func(*types.Timeout)
type broadcastSyncInfoFn func(*types.SyncInfo)

// chainHeightFn is a callback type to retrieve the current chain height.
type chainHeightFn func() uint64

type Bfter struct {
	epoch uint64

	blockChainReader consensus.ChainReader
	broadcastCh      chan interface{}
	quit             chan struct{}
	consensus        ConsensusFns
	broadcast        BroadcastFns
	chainHeight      chainHeightFn // Retrieves the current chain's height
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

func New(broadcasts BroadcastFns, blockChainReader *core.BlockChain, chainHeight chainHeightFn) *Bfter {
	return &Bfter{
		broadcast:        broadcasts,
		blockChainReader: blockChainReader,
		chainHeight:      chainHeight,

		quit:        make(chan struct{}),
		broadcastCh: make(chan interface{}),
	}
}

// Create this function to avoid massive test change
func (b *Bfter) InitEpochNumber() {
	b.epoch = b.blockChainReader.Config().XDPoS.Epoch
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

func (b *Bfter) Vote(peer string, vote *types.Vote) error {
	log.Trace("Receive Vote", "hash", vote.Hash().Hex(), "voted block hash", vote.ProposedBlockInfo.Hash.Hex(), "number", vote.ProposedBlockInfo.Number, "round", vote.ProposedBlockInfo.Round)

	voteBlockNum := vote.ProposedBlockInfo.Number.Int64()
	if dist := voteBlockNum - int64(b.chainHeight()); dist < -maxBlockDist || dist > maxBlockDist {
		log.Debug("Discarded propagated vote, too far away", "peer", peer, "number", voteBlockNum, "hash", vote.ProposedBlockInfo.Hash, "distance", dist)
		return nil
	}

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
func (b *Bfter) Timeout(peer string, timeout *types.Timeout) error {
	log.Debug("Receive Timeout", "timeout", timeout)

	gapNum := timeout.GapNumber

	// dist times 3, ex: timeout message's gap number is based on block and find out it's epoch switch number, then mod 900 then minus 450
	if dist := int64(gapNum) - int64(b.chainHeight()); dist < -int64(b.epoch)*3 || dist > int64(b.epoch)*3 {
		log.Debug("Discarded propagated timeout, too far away", "peer", peer, "gapNumber", gapNum, "hash", timeout.Hash, "distance", dist)
		return nil
	}

	verified, err := b.consensus.verifyTimeout(b.blockChainReader, timeout)
	if err != nil {
		log.Error("Verify BFT Timeout", "timeoutRound", timeout.Round, "timeoutGapNum", gapNum, "error", err)
		return err
	}

	if verified {
		b.broadcastCh <- timeout
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
func (b *Bfter) SyncInfo(peer string, syncInfo *types.SyncInfo) error {
	log.Debug("Receive SyncInfo", "syncInfo", syncInfo)

	qcBlockNum := syncInfo.HighestQuorumCert.ProposedBlockInfo.Number.Int64()
	if dist := qcBlockNum - int64(b.chainHeight()); dist < -maxBlockDist || dist > maxBlockDist {
		log.Debug("Discarded propagated syncInfo, too far away", "peer", peer, "blockNum", qcBlockNum, "hash", syncInfo.Hash, "distance", dist)
		return nil
	}

	verified, err := b.consensus.verifySyncInfo(b.blockChainReader, syncInfo)
	if err != nil {
		log.Error("Verify BFT SyncInfo", "error", err)
		return err
	}

	// Process only if verified and qualified
	if verified {
		b.broadcastCh <- syncInfo
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
	log.Info("BFT Loop Start")
	for {
		select {
		case <-b.quit:
			log.Warn("BFT Loop Close")
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
