package bft

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/engines/engine_v2"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/stretchr/testify/assert"
)

// make different votes based on Signatures
func makeVotes(n int) []types.Vote {
	var votes []types.Vote
	for i := 0; i < n; i++ {
		votes = append(votes, types.Vote{
			ProposedBlockInfo: &types.BlockInfo{},
			Signature:         []byte{byte(i)},
			GapNumber:         0,
		})
	}
	return votes
}

// bfterTester is a test simulator for mocking out bfter worker.
type bfterTester struct {
	bfter *Bfter
}

// newTester creates a new bft fetcher test mocker.
func newTester() *bfterTester {
	testConsensus := &XDPoS.XDPoS{EngineV2: &engine_v2.XDPoS_v2{}}
	broadcasts := BroadcastFns{}
	blockChain := &core.BlockChain{}

	tester := &bfterTester{}
	tester.bfter = New(broadcasts, blockChain)
	tester.bfter.SetConsensusFuns(testConsensus)
	tester.bfter.broadcastCh = make(chan interface{})
	tester.bfter.Start()

	return tester
}

// Tests that a bfter accepts vote and process verfiy and broadcast
func TestSequentialVotes(t *testing.T) {
	tester := newTester()
	verifyCounter := uint32(0)
	handlerCounter := uint32(0)
	broadcastCounter := uint32(0)
	targetVotes := 10

	tester.bfter.consensus.verifyVote = func(chain consensus.ChainReader, vote *types.Vote) (bool, error) {
		atomic.AddUint32(&verifyCounter, 1)
		return true, nil
	}

	tester.bfter.consensus.voteHandler = func(chain consensus.ChainReader, vote *types.Vote) error {
		atomic.AddUint32(&handlerCounter, 1)
		return nil
	}

	tester.bfter.broadcast.Vote = func(*types.Vote) {
		atomic.AddUint32(&broadcastCounter, 1)
	}

	votes := makeVotes(targetVotes)
	for _, vote := range votes {
		err := tester.bfter.Vote(&vote)
		if err != nil {
			t.Fatal(err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	if int(verifyCounter) != targetVotes || int(handlerCounter) != targetVotes || int(broadcastCounter) != targetVotes {
		t.Fatalf("count mismatch: have %v on verify, %v on handler, %v on broadcast, want %v", verifyCounter, handlerCounter, broadcastCounter, targetVotes)
	}
}

// Test that avoid boardcast if there is bad vote
func TestNotBoardcastInvalidVote(t *testing.T) {
	tester := newTester()
	handlerCounter := uint32(0)
	broadcastCounter := uint32(0)
	targetVotes := 0

	tester.bfter.consensus.verifyVote = func(chain consensus.ChainReader, vote *types.Vote) (bool, error) {
		return false, fmt.Errorf("This is invalid vote")
	}

	tester.bfter.consensus.voteHandler = func(chain consensus.ChainReader, vote *types.Vote) error {
		atomic.AddUint32(&handlerCounter, 1)
		return nil
	}
	tester.bfter.broadcast.Vote = func(*types.Vote) {
		atomic.AddUint32(&broadcastCounter, 1)
	}

	vote := types.Vote{ProposedBlockInfo: &types.BlockInfo{}}
	tester.bfter.Vote(&vote)

	time.Sleep(50 * time.Millisecond)
	if int(handlerCounter) != targetVotes || int(broadcastCounter) != targetVotes {
		t.Fatalf("count mismatch: have %v on handler, %v on broadcast, want %v", handlerCounter, broadcastCounter, targetVotes)
	}
}

func TestBoardcastButNotProcessDisqualifiedVotes(t *testing.T) {
	tester := newTester()
	handlerCounter := uint32(0)
	broadcastCounter := uint32(0)
	targetVotes := 0

	tester.bfter.consensus.verifyVote = func(chain consensus.ChainReader, vote *types.Vote) (bool, error) {
		return false, nil // return false but with nil in error means the message is valid but disqualified
	}

	tester.bfter.consensus.voteHandler = func(chain consensus.ChainReader, vote *types.Vote) error {
		atomic.AddUint32(&handlerCounter, 1)
		return nil
	}
	tester.bfter.broadcast.Vote = func(*types.Vote) {
		atomic.AddUint32(&broadcastCounter, 1)
	}

	vote := types.Vote{ProposedBlockInfo: &types.BlockInfo{}}
	tester.bfter.Vote(&vote)

	time.Sleep(50 * time.Millisecond)
	if int(handlerCounter) != targetVotes || int(broadcastCounter) != 1 {
		t.Fatalf("count mismatch: have %v on handler, %v on broadcast, want %v", handlerCounter, broadcastCounter, targetVotes)
	}
}

func TestBoardcastButNotProcessDisqualifiedTimeout(t *testing.T) {
	tester := newTester()
	handlerCounter := uint32(0)
	broadcastCounter := uint32(0)
	targetTimeout := 0

	tester.bfter.consensus.verifyTimeout = func(chain consensus.ChainReader, timeout *types.Timeout) (bool, error) {
		return false, nil // return false but with nil in error means the message is valid but disqualified
	}

	tester.bfter.consensus.timeoutHandler = func(chain consensus.ChainReader, timeout *types.Timeout) error {
		atomic.AddUint32(&handlerCounter, 1)
		return nil
	}
	tester.bfter.broadcast.Timeout = func(*types.Timeout) {
		atomic.AddUint32(&broadcastCounter, 1)
	}

	timeout := types.Timeout{}
	tester.bfter.Timeout(&timeout)

	time.Sleep(50 * time.Millisecond)
	if int(handlerCounter) != targetTimeout || int(broadcastCounter) != 1 {
		t.Fatalf("count mismatch: have %v on handler, %v on broadcast, want %v", handlerCounter, broadcastCounter, targetTimeout)
	}
}

func TestBoardcastButNotProcessDisqualifiedSyncInfo(t *testing.T) {
	tester := newTester()
	handlerCounter := uint32(0)
	broadcastCounter := uint32(0)
	targetSyncInfo := 0

	tester.bfter.consensus.verifySyncInfo = func(chain consensus.ChainReader, syncInfo *types.SyncInfo) (bool, error) {
		return false, nil // return false but with nil in error means the message is valid but disqualified
	}

	tester.bfter.consensus.syncInfoHandler = func(chain consensus.ChainReader, syncInfo *types.SyncInfo) error {
		atomic.AddUint32(&handlerCounter, 1)
		return nil
	}
	tester.bfter.broadcast.SyncInfo = func(*types.SyncInfo) {
		atomic.AddUint32(&broadcastCounter, 1)
	}

	syncInfo := types.SyncInfo{}
	tester.bfter.SyncInfo(&syncInfo)

	time.Sleep(50 * time.Millisecond)
	if int(handlerCounter) != targetSyncInfo || int(broadcastCounter) != 1 {
		t.Fatalf("count mismatch: have %v on handler, %v on broadcast, want %v", handlerCounter, broadcastCounter, targetSyncInfo)
	}
}

// TODO: SyncInfo and Timeout Test, should be same as Vote.
// Once all test on vote covered, then duplicate to others

func TestTimeoutHandler(t *testing.T) {
	tester := newTester()
	verifyCounter := uint32(0)
	handlerCounter := uint32(0)
	broadcastCounter := uint32(0)
	targetVotes := 1

	tester.bfter.consensus.verifyTimeout = func(consensus.ChainReader, *types.Timeout) (bool, error) {
		atomic.AddUint32(&verifyCounter, 1)
		return true, nil
	}

	tester.bfter.consensus.timeoutHandler = func(chain consensus.ChainReader, timeout *types.Timeout) error {
		atomic.AddUint32(&handlerCounter, 1)
		return nil
	}

	tester.bfter.broadcast.Timeout = func(*types.Timeout) {
		atomic.AddUint32(&broadcastCounter, 1)
	}

	timeoutMsg := &types.Timeout{}

	err := tester.bfter.Timeout(timeoutMsg)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)

	if int(verifyCounter) != targetVotes || int(handlerCounter) != targetVotes || int(broadcastCounter) != targetVotes {
		t.Fatalf("count mismatch: have %v on verify, %v on handler, %v on broadcast, want %v", verifyCounter, handlerCounter, broadcastCounter, targetVotes)
	}
}

func TestTimeoutHandlerRoundNotEqual(t *testing.T) {
	tester := newTester()

	tester.bfter.consensus.verifyTimeout = func(consensus.ChainReader, *types.Timeout) (bool, error) {
		return true, nil
	}

	tester.bfter.consensus.timeoutHandler = func(chain consensus.ChainReader, timeout *types.Timeout) error {
		return &utils.ErrIncomingMessageRoundNotEqualCurrentRound{
			Type:          "timeout",
			IncomingRound: types.Round(1),
			CurrentRound:  types.Round(2),
		}
	}

	tester.bfter.broadcast.Timeout = func(*types.Timeout) {}

	timeoutMsg := &types.Timeout{}

	err := tester.bfter.Timeout(timeoutMsg)
	assert.Equal(t, "timeout message round number: 1 does not match currentRound: 2", err.Error())
}
