package bfter

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/engines/engine_v2"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
)

// make different votes based on Signatures
func makeVotes(n int) []utils.Vote {
	var votes []utils.Vote
	for i := 0; i < n; i++ {
		votes = append(votes, utils.Vote{Signature: []byte{byte(i)}})
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

	tester := &bfterTester{}
	tester.bfter = New(broadcasts)
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

	tester.bfter.consensus.verifyVote = func(vote *utils.Vote) error {
		atomic.AddUint32(&verifyCounter, 1)
		return nil
	}

	tester.bfter.consensus.voteHandler = func(vote *utils.Vote) error {
		atomic.AddUint32(&handlerCounter, 1)
		return nil
	}

	tester.bfter.broadcast.Vote = func(*utils.Vote) {
		atomic.AddUint32(&broadcastCounter, 1)
	}

	votes := makeVotes(targetVotes)
	for _, vote := range votes {
		tester.bfter.Vote(&vote)
	}

	time.Sleep(100 * time.Millisecond)

	if int(verifyCounter) != targetVotes || int(handlerCounter) != targetVotes || int(broadcastCounter) != targetVotes {
		t.Fatalf("count mismatch: have %v on verify, %v on handler, %v on broadcast, want %v", verifyCounter, handlerCounter, broadcastCounter, targetVotes)
	}
}

// Tests that vote already being retrieved will not be duplicated.
func TestDuplicateVotes(t *testing.T) {
	tester := newTester()
	verifyCounter := uint32(0)
	handlerCounter := uint32(0)
	broadcastCounter := uint32(0)
	targetVotes := 1

	tester.bfter.consensus.verifyVote = func(vote *utils.Vote) error {
		atomic.AddUint32(&verifyCounter, 1)
		return nil
	}

	tester.bfter.consensus.voteHandler = func(vote *utils.Vote) error {
		atomic.AddUint32(&handlerCounter, 1)
		return nil
	}

	tester.bfter.broadcast.Vote = func(*utils.Vote) {
		atomic.AddUint32(&broadcastCounter, 1)
	}

	vote := utils.Vote{}

	// send twice
	tester.bfter.Vote(&vote)
	tester.bfter.Vote(&vote)

	time.Sleep(50 * time.Millisecond)
	if int(verifyCounter) != targetVotes || int(handlerCounter) != targetVotes || int(broadcastCounter) != targetVotes {
		t.Fatalf("count mismatch: have %v on verify, %v on handler,  %v on broadcast, want %v", verifyCounter, handlerCounter, broadcastCounter, targetVotes)
	}
}

// Test that avoid boardcast if there is bad vote
func TestNotBoardcastInvalidVote(t *testing.T) {
	tester := newTester()
	handlerCounter := uint32(0)
	broadcastCounter := uint32(0)
	targetVotes := 0

	tester.bfter.consensus.verifyVote = func(vote *utils.Vote) error {
		return fmt.Errorf("This is invalid vote")
	}

	tester.bfter.consensus.voteHandler = func(vote *utils.Vote) error {
		atomic.AddUint32(&handlerCounter, 1)
		return nil
	}
	tester.bfter.broadcast.Vote = func(*utils.Vote) {
		atomic.AddUint32(&broadcastCounter, 1)
	}

	vote := utils.Vote{}
	tester.bfter.Vote(&vote)

	time.Sleep(50 * time.Millisecond)
	if int(handlerCounter) != targetVotes || int(broadcastCounter) != targetVotes {
		t.Fatalf("count mismatch: have %v on handler, %v on broadcast, want %v", handlerCounter, broadcastCounter, targetVotes)
	}
}

// TODO: SyncInfo and Timeout Test, should be same as Vote.
// Once all test on vote covered, then duplicate to others
