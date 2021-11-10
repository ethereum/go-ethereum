package engine_v2

import (
	"fmt"
	"sync"
	"time"

	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/common/countdown"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/consensus/clique"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
)

type XDPoS_v2 struct {
	config *params.XDPoSConfig // Consensus engine configuration parameters
	db     ethdb.Database      // Database to store and retrieve snapshot checkpoints

	signer common.Address  // Ethereum address of the signing key
	signFn clique.SignerFn // Signer function to authorize hashes with
	lock   sync.RWMutex    // Protects the signer fields

	BroadcastCh   chan interface{}
	BFTQueue      chan interface{}
	timeoutWorker *countdown.CountdownTimer // Timer to generate broadcast timeout msg if threashold reached

	timeoutPool        *utils.Pool
	currentRound       utils.Round
	highestVotedRound  utils.Round
	highestQuorumCert  *utils.QuorumCert
	// LockQC in XDPoS Consensus 2.0, used in voting rule
	lockQuorumCert     *utils.QuorumCert
	highestTimeoutCert *utils.TimeoutCert
	highestCommitBlock *utils.BlockInfo
}

func New(config *params.XDPoSConfig, db ethdb.Database) *XDPoS_v2 {
	// Setup Timer
	duration := time.Duration(config.V2.TimeoutWorkerDuration) * time.Millisecond
	timer := countdown.NewCountDown(duration)
	timeoutPool := utils.NewPool(config.V2.CertThreshold)
	engine := &XDPoS_v2{
		config:        config,
		db:            db,
		timeoutWorker: timer,
		BroadcastCh:   make(chan interface{}),
		BFTQueue:      make(chan interface{}),
		timeoutPool:   timeoutPool,
	}
	// Add callback to the timer
	timer.OnTimeoutFn = engine.onCountdownTimeout

	return engine
}

/*
	Testing tools
*/
// Test only. Never to be used for mainnet implementation
func NewFaker(db ethdb.Database, config *params.XDPoSConfig) *XDPoS_v2 {
	var fakeEngine *XDPoS_v2
	// Set any missing consensus parameters to their defaults
	conf := config
	// Setup Timer
	duration := time.Duration(config.V2.TimeoutWorkerDuration) * time.Millisecond
	timer := countdown.NewCountDown(duration)
	timeoutPool := utils.NewPool(2)

	// Allocate the snapshot caches and create the engine
	fakeEngine = &XDPoS_v2{
		config:        conf,
		db:            db,
		timeoutWorker: timer,
		BroadcastCh:   make(chan interface{}),
		BFTQueue:      make(chan interface{}),
		timeoutPool:   timeoutPool,
	}
	// Add callback to the timer
	timer.OnTimeoutFn = fakeEngine.onCountdownTimeout
	return fakeEngine
}

// Test only.
func (x *XDPoS_v2) SetNewRoundFaker(newRound utils.Round) {
	// Reset a bunch of things
	x.timeoutWorker.Reset()
	x.currentRound = newRound
}

// Authorize injects a private key into the consensus engine to mint new blocks with.
func (x *XDPoS_v2) Authorize(signer common.Address, signFn clique.SignerFn) {
	x.lock.Lock()
	defer x.lock.Unlock()

	x.signer = signer
	x.signFn = signFn
}

func (x *XDPoS_v2) Author(header *types.Header) (common.Address, error) {
	return common.Address{}, nil
}

func (x *XDPoS_v2) VerifyHeader(chain consensus.ChainReader, header *types.Header, fullVerify bool) error {
	return nil
}

// Push mesages(i.e vote, sync info & timeout) into BFTQueue. This funciton shall be called by BFT protocal manager
func (x *XDPoS_v2) Enqueue() error {
	return nil
}

// Main function for the v2 consensus.
func (x *XDPoS_v2) Dispatcher() error {
	// 1. Pull message from the BFTQueue and call the relevant handler by message type, such as vote, timeout or syncInfo
	// 2. Only 1 message processing at the time
	return nil
}

/*
	SyncInfo workflow
*/
// Verify syncInfo and trigger trigger process QC or TC if successful
func (x *XDPoS_v2) VerifySyncInfoMessage(syncInfo utils.SyncInfo) error {
	/*
		1. Verify items including:
				- verifyQC
				- verifyTC
		2. Broadcast(Not part of consensus)
	*/
	return nil
}

func (x *XDPoS_v2) SyncInfoHandler(header *types.Header) error {
	/*
		1. processQC
		2. processTC
	*/
	return nil
}

/*
	Vote workflow
*/
func (x *XDPoS_v2) VerifyVoteMessage(vote utils.Vote) error {
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

func (x *XDPoS_v2) VoteHandler() {
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
/*
	  1. Check signature:
				- Use ecRecover to get the public key
				- Use the above public key to find out the xdc address
				- Use the above xdc address to check against the master node(For the running epoch)
		2. Broadcast(Not part of consensus)
*/
func (x *XDPoS_v2) VerifyTimeoutMessage(timeoutMsg utils.Timeout) (bool, error) {
	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(utils.TimeoutSigHash(&timeoutMsg.Round).Bytes(), timeoutMsg.Signature)
	if err != nil {
		return false, fmt.Errorf("Error while verifying time out message: %v", err)
	}
	var signerAddress common.Address
	copy(signerAddress[:], crypto.Keccak256(pubkey[1:])[12:])
	masternodes := x.getCurrentRoundMasterNodes()
	for _, mn := range masternodes {
		if mn == signerAddress {
			return true, nil
		}
	}

	return false, fmt.Errorf("Masternodes does not contain signer address. Master node list %v, Signer address: %v", masternodes, signerAddress)
}

/*
	1. checkRoundNumber()
	2. Collect timeout (TODO)
	3. Genrate TC (TODO)
	4. processTC()
	5. generateSyncInfo()
*/
func (x *XDPoS_v2) TimeoutHandler(timeout *utils.Timeout) {
	// Collect timeout, generate TC
	timeoutCert := x.timeoutPool.Add(timeout)
	// If TC is generated
	if timeoutCert != nil {
		//TODO: processTC(),generateSyncInfo()
	}
}

/*
	Process Block workflow
*/
func (x *XDPoS_v2) ProcessBlockHandler() {
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
func (x *XDPoS_v2) generateBlockInfo() error {
	return nil
}

// To be used by different message verification. Verify local DB block info against the received block information(i.e hash, blockNum, round)
func (x *XDPoS_v2) VerifyBlockInfo(blockInfo utils.BlockInfo) error {
	return nil
}

func (x *XDPoS_v2) verifyQC(header *types.Header) error {
	/*
		1. Verify signer signatures: (List of signatures)
					- Use ecRecover to get the public key
					- Use the above public key to find out the xdc address
					- Use the above xdc address to check against the master node list(For the received QC epoch)
		2. Verify blockInfo
	*/
	return nil
}

func (x *XDPoS_v2) verifyTC(header *types.Header) error {
	/*
		1. Verify signer signature: (List of signatures)
					- Use ecRecover to get the public key
					- Use the above public key to find out the xdc address
					- Use the above xdc address to check against the master node list(For the received TC epoch)
	*/
	return nil
}

// Update local QC variables including highestQC & lockQC, as well as update commit blockInfo before call
/*
	1. Update HighestQC and LockQC
	2. Update commit block info (TODO)
	3. Check QC round >= node's currentRound. If yes, call setNewRound
*/
func (x *XDPoS_v2) processQC(quorumCert *utils.QuorumCert) error {
	if x.highestQuorumCert == nil || quorumCert.ProposedBlockInfo.Round > x.highestQuorumCert.ProposedBlockInfo.Round {
		x.highestQuorumCert = quorumCert
		//TODO: do I need a clone?
	}
	//TODO: x.blockchain.getBlock(quorumCert.ProposedBlockInfo.Hash) then get the QC inside that block header
	//TODO: update lockQC
	//TODO: find parent and grandparent and grandgrandparent block, check round number, if so, commit grandgrandparent
	if quorumCert.ProposedBlockInfo.Round >= x.currentRound {
		x.setNewRound(quorumCert.ProposedBlockInfo.Round + 1)
	}
	return nil
}

/*
	1. Update highestTC
	2. Check TC round >= node's currentRound. If yes, call setNewRound
*/
func (x *XDPoS_v2) processTC(timeoutCert *utils.TimeoutCert) error {
	if x.highestTimeoutCert == nil || timeoutCert.Round > x.highestTimeoutCert.Round {
		x.highestTimeoutCert = timeoutCert
	}
	if timeoutCert.Round >= x.currentRound {
		x.setNewRound(timeoutCert.Round + 1)
	}
	return nil
}

/*
	1. Set currentRound = QC round + 1 (or TC round +1)
	2. Reset timer
	3. Reset vote and timeout Pools
*/
func (x *XDPoS_v2) setNewRound(round utils.Round) error {
	x.currentRound = round
	//TODO: tell miner now it's a new round and start mine if it's leader
	//TODO: reset timer
	//TODO: vote pools
	x.timeoutPool.Clear()
	return nil
}

// Verify round number against node's local round number(Should be equal)
func (x *XDPoS_v2) checkRoundNumber(header *types.Header) error {
	return nil
}

// Hot stuff rule to decide whether this node is eligible to vote for the received block
func (x *XDPoS_v2) verifyVotingRule(header *types.Header) error {
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
func (x *XDPoS_v2) sendVote(header *types.Header) error {
	// First step: Generate the signature by using node's private key(The signature is the blockInfo signature)
	// Second step: Construct the vote struct with the above signature & blockinfo struct
	// Third step: Send the vote to broadcast channel
	return nil
}

// Generate and send timeout into BFT channel.
/*
	1. timeout.round = currentRound
	2. Sign the signature
	3. send to broadcast channel
*/
func (x *XDPoS_v2) sendTimeout() error {
	// Don't hold the signer fields for the entire sealing procedure
	x.lock.RLock()
	signer, signFn := x.signer, x.signFn
	x.lock.RUnlock()

	signedHash, err := signFn(accounts.Account{Address: signer}, utils.TimeoutSigHash(&x.currentRound).Bytes())
	if err != nil {
		return fmt.Errorf("Error while signing for timeout message")
	}
	timeoutMsg := &utils.Timeout{
		Round:     x.currentRound,
		Signature: signedHash,
	}
	x.broadcastToBftChannel(timeoutMsg)
	return nil
}

// Generate and send syncInfo into Broadcast channel. The SyncInfo includes local highest QC & TC
func (x *XDPoS_v2) sendSyncInfo() error {
	return nil
}

/*
	Function that will be called by timer when countdown reaches its threshold.
	In the engine v2, we would need to broadcast timeout messages to other peers
*/
func (x *XDPoS_v2) onCountdownTimeout(time time.Time) error {
	err := x.sendTimeout()
	if err != nil {
		log.Error("Error while sending out timeout message at time: ", time)
		return err
	}
	return nil
}

func (x *XDPoS_v2) broadcastToBftChannel(msg interface{}) {
	x.BroadcastCh <- msg
}

func (x *XDPoS_v2) getCurrentRoundMasterNodes() []common.Address {
	return []common.Address{}
}
