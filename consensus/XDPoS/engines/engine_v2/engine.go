package engine_v2

import (
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/common/countdown"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
)

type XDPoS_v2 struct {
	config        *params.XDPoSConfig // Consensus engine configuration parameters
	db            ethdb.Database      // Database to store and retrieve snapshot checkpoints
	BroadcastCh   chan interface{}
	BFTQueue      chan interface{}
	timeoutWorker *countdown.CountdownTimer // Timer to generate broadcast timeout msg if threashold reached
}

func New(config *params.XDPoSConfig, db ethdb.Database) *XDPoS_v2 {
	// Setup Timer
	duration := time.Duration(config.ConsensusV2Config.TimeoutWorkerDuration) * time.Millisecond
	timer := countdown.NewCountDown(duration)

	engine := &XDPoS_v2{
		config:        config,
		db:            db,
		timeoutWorker: timer,
	}
	// Add callback to the timer
	timer.OnTimeoutFn = engine.onCountdownTimeout

	return engine
}

func NewFaker(db ethdb.Database, config *params.XDPoSConfig) *XDPoS_v2 {
	var fakeEngine *XDPoS_v2
	// Set any missing consensus parameters to their defaults
	conf := config
	// Setup Timer
	duration := time.Duration(config.ConsensusV2Config.TimeoutWorkerDuration) * time.Millisecond
	timer := countdown.NewCountDown(duration)

	// Allocate the snapshot caches and create the engine
	fakeEngine = &XDPoS_v2{
		config:        conf,
		db:            db,
		timeoutWorker: timer,
	}
	return fakeEngine
}

func (consensus *XDPoS_v2) Author(header *types.Header) (common.Address, error) {
	return common.Address{}, nil
}

func (consensus *XDPoS_v2) VerifyHeader(chain consensus.ChainReader, header *types.Header, fullVerify bool) error {
	return nil
}

// Push mesages(i.e vote, sync info & timeout) into BFTQueue. This funciton shall be called by BFT protocal manager
func (consensus *XDPoS_v2) Enqueue() error {
	return nil
}

// Main function for the v2 consensus.
func (consensus *XDPoS_v2) Dispatcher() error {
	// 1. Pull message from the BFTQueue and call the relevant handler by message type, such as vote, timeout or syncInfo
	// 2. Only 1 message processing at the time
	return nil
}

/*
	SyncInfo workflow
*/
// Verify syncInfo and trigger trigger process QC or TC if successful
func (consensus *XDPoS_v2) VerifySyncInfoMessage(header *types.Header) error {
	/*
		1. Verify items including:
				- verifyQC
				- verifyTC
		2. Broadcast(Not part of consensus)
	*/
	return nil
}

func (consensus *XDPoS_v2) SyncInfoHandler(header *types.Header) error {
	/*
		1. processQC
		2. processTC
	*/
	return nil
}

/*
	Vote workflow
*/
func (consensus *XDPoS_v2) VerifyVoteMessage() error {
	/*
		  1. Check signature:
					- Use ecRecover to get the public key
					- Use the above public key to find out the xdc address
					- Use the above xdc address to check against the master node list(For the running epoch)
			2. Verify blockInfo
			3. Broadcast(Not part of consensus)
	*/
	return nil
}

func (consensus *XDPoS_v2) VoteHandler() {
	/*
		1. checkRoundNumber
		3. Collect vote (TODO)
		4. Genrate QC (TODO)
		5. processQC
	*/
}

/*
	Timeout workflow
*/
// Verify timeout message type from peers in bft.go
func (consensus *XDPoS_v2) VerifyTimeoutMessage() error {
	/*
		  1. Check signature:
					- Use ecRecover to get the public key
					- Use the above public key to find out the xdc address
					- Use the above xdc address to check against the master node(For the running epoch)
			2. Broadcast(Not part of consensus)
	*/
	return nil
}

func (consensus *XDPoS_v2) TimeoutHandler() {
	/*
		1. checkRoundNumber()
		2. Collect timeout (TODO)
		3. Genrate TC (TODO)
		4. processTC()
		5. generateSyncInfo()
	*/
}

/*
	Process Block workflow
*/
func (consensus *XDPoS_v2) ProcessBlockHandler() {
	/*
		1. processQC()
		2. verifyVotingRule()
		3. sendVote()

	*/
}

/*
	QC & TC Utils
*/

// Genrate blockInfo which contains Hash, round and blockNumber and send to queue
func (consensus *XDPoS_v2) generateBlockInfo() error {
	return nil
}

// To be used by different message verification. Verify local DB block info against the received block information(i.e hash, blockNum, round)
func (consensus *XDPoS_v2) verifyBlockInfo(header *types.Header) error {
	return nil
}

func (consensus *XDPoS_v2) verifyQC(header *types.Header) error {
	/*
		1. Verify signer signatures: (List of signatures)
					- Use ecRecover to get the public key
					- Use the above public key to find out the xdc address
					- Use the above xdc address to check against the master node list(For the received QC epoch)
		2. Verify blockInfo
	*/
	return nil
}

func (consensus *XDPoS_v2) verifyTC(header *types.Header) error {
	/*
		1. Verify signer signature: (List of signatures)
					- Use ecRecover to get the public key
					- Use the above public key to find out the xdc address
					- Use the above xdc address to check against the master node list(For the received TC epoch)
	*/
	return nil
}

// Update local QC variables including highestQC & lockQC, as well as update commit blockInfo before call
func (consensus *XDPoS_v2) processQC(header *types.Header) error {
	/*
		1. Update HighestQC and LockQC
		2. Update commit block info (TODO)
		3. Check QC round >= node's currentRound. If yes, call setNewRound
	*/
	return nil
}

func (consensus *XDPoS_v2) processTC(header *types.Header) error {
	/*
		1. Update highestTC
		2. Check TC round >= node's currentRound. If yes, call setNewRound
	*/
	return nil
}

func (consensus *XDPoS_v2) setNewRound() error {
	/*
		1. Set currentRound = QC round + 1 (or TC round +1)
		2. Reset timer
		3. Reset vote and timeout Pools
	*/
	return nil
}

// Verify round number against node's local round number(Should be equal)
func (consensus *XDPoS_v2) checkRoundNumber(header *types.Header) error {
	return nil
}

// Hot stuff rule to decide whether this node is eligible to vote for the received block
func (consensus *XDPoS_v2) verifyVotingRule(header *types.Header) error {
	/*
		Make sure this node has not voted for this round. We can have a variable highestVotedRound, and check currentRound > highestVotedRound.
		HotStuff Voting rule:
		header's round == local current round, AND (one of the following two:)
		header's block extends LockQC's ProposedBlockInfo (we need a isExtending(block_a, block_b) function), OR
		header's QC's ProposedBlockInfo.Round > LockQC's ProposedBlockInfo.Round
	*/
	return nil
}

// Once Hot stuff voting rule has verified, this node can then send vote
func (consensus *XDPoS_v2) sendVote(header *types.Header) error {
	// First step: Generate the signature by using node's private key(The signature is the blockInfo signature)
	// Second step: Construct the vote struct with the above signature & blockinfo struct
	// Third step: Send the vote to broadcast channel
	return nil
}

// Generate and send timeout into BFT channel.
func (consensus *XDPoS_v2) sendTimeout() error {
	/*
		1. timeout.round = currentRound
		2. Sign the signature
		3. send to broadcast channel
	*/
	return nil
}

// Generate and send syncInfo into Broadcast channel. The SyncInfo includes local highest QC & TC
func (consensus *XDPoS_v2) sendSyncInfo() error {
	return nil
}

/*
	Function that will be called by timer when countdown reaches its threshold.
	In the engine v2, we would need to broadcast timeout messages to other peers
*/
func (consensus *XDPoS_v2) onCountdownTimeout(time time.Time) error {
	err := consensus.sendTimeout()
	if err != nil {
		log.Error("Error while sending out timeout message at time: ", time)
	}
	return nil
}
