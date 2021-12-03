package engine_v2

import (
	"fmt"
	"math/big"
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

	signer   common.Address  // Ethereum address of the signing key
	signFn   clique.SignerFn // Signer function to authorize hashes with
	signLock sync.RWMutex    // Protects the signer fields

	BroadcastCh   chan interface{}
	timeoutWorker *countdown.CountdownTimer // Timer to generate broadcast timeout msg if threashold reached

	lock              sync.RWMutex // Protects the currentRound fields etc
	timeoutPool       *utils.Pool
	votePool          *utils.Pool
	currentRound      utils.Round
	highestVotedRound utils.Round
	highestQuorumCert *utils.QuorumCert
	// lockQuorumCert in XDPoS Consensus 2.0, used in voting rule
	lockQuorumCert     *utils.QuorumCert
	highestTimeoutCert *utils.TimeoutCert
	highestCommitBlock *utils.BlockInfo
}

func New(config *params.XDPoSConfig, db ethdb.Database) *XDPoS_v2 {
	// Setup Timer
	duration := time.Duration(config.V2.TimeoutWorkerDuration) * time.Millisecond
	timer := countdown.NewCountDown(duration)
	timeoutPool := utils.NewPool(config.V2.CertThreshold)
	votePool := utils.NewPool(config.V2.CertThreshold)
	engine := &XDPoS_v2{
		config:             config,
		db:                 db,
		timeoutWorker:      timer,
		BroadcastCh:        make(chan interface{}),
		timeoutPool:        timeoutPool,
		votePool:           votePool,
		highestTimeoutCert: &utils.TimeoutCert{},
		highestQuorumCert:  &utils.QuorumCert{},
		highestVotedRound:  utils.Round(0),
	}
	// Add callback to the timer
	timer.OnTimeoutFn = engine.onCountdownTimeout

	return engine
}

/*
	Testing tools
*/
func (x *XDPoS_v2) SetNewRoundFaker(newRound utils.Round, resetTimer bool) {
	x.lock.Lock()
	defer x.lock.Unlock()
	// Reset a bunch of things
	if resetTimer {
		x.timeoutWorker.Reset()
	}
	x.currentRound = newRound
}

// Utils for test to check currentRound value
func (x *XDPoS_v2) GetProperties() (utils.Round, *utils.QuorumCert, *utils.QuorumCert) {
	x.lock.Lock()
	defer x.lock.Unlock()
	return x.currentRound, x.lockQuorumCert, x.highestQuorumCert
}

// Authorize injects a private key into the consensus engine to mint new blocks with.
func (x *XDPoS_v2) Authorize(signer common.Address, signFn clique.SignerFn) {
	x.signLock.Lock()
	defer x.signLock.Unlock()

	x.signer = signer
	x.signFn = signFn
}

func (x *XDPoS_v2) Author(header *types.Header) (common.Address, error) {
	return common.Address{}, nil
}

func (x *XDPoS_v2) VerifyHeader(chain consensus.ChainReader, header *types.Header, fullVerify bool) error {
	return nil
}

/*
	SyncInfo workflow
*/
// Verify syncInfo and trigger process QC or TC if successful
func (x *XDPoS_v2) VerifySyncInfoMessage(syncInfo *utils.SyncInfo) error {
	/*
		1. Verify items including:
				- verifyQC
				- verifyTC
		2. Broadcast(Not part of consensus)
	*/
	err := x.verifyQC(syncInfo.HighestQuorumCert)
	if err != nil {
		log.Warn("SyncInfo message verification failed due to QC", err)
		return err
	}
	err = x.verifyTC(syncInfo.HighestTimeoutCert)
	if err != nil {
		log.Warn("SyncInfo message verification failed due to TC", err)
		return err
	}
	return nil
}

func (x *XDPoS_v2) SyncInfoHandler(chain consensus.ChainReader, syncInfo *utils.SyncInfo) error {
	x.signLock.Lock()
	defer x.signLock.Unlock()
	/*
		1. processQC
		2. processTC
	*/
	err := x.processQC(chain, syncInfo.HighestQuorumCert)
	if err != nil {
		return err
	}
	return x.processTC(syncInfo.HighestTimeoutCert)
}

/*
	Vote workflow
*/
func (x *XDPoS_v2) VerifyVoteMessage(vote *utils.Vote) (bool, error) {
	/*
		  1. Check signature:
					- Use ecRecover to get the public key
					- Use the above public key to find out the xdc address
					- Use the above xdc address to check against the master node list(For the running epoch)
			2. Verify blockInfo
			3. Broadcast(Not part of consensus)
	*/
	return x.verifyMsgSignature(utils.VoteSigHash(&vote.ProposedBlockInfo), vote.Signature)
}

// Consensus entry point for processing vote message to produce QC
func (x *XDPoS_v2) VoteHandler(chain consensus.ChainReader, voteMsg *utils.Vote) error {
	x.lock.Lock()
	defer x.lock.Unlock()

	// 1. checkRoundNumber
	if voteMsg.ProposedBlockInfo.Round != x.currentRound {
		return fmt.Errorf("Vote message round number: %v does not match currentRound: %v", voteMsg.ProposedBlockInfo.Round, x.currentRound)
	}

	// Collect vote
	thresholdReached, numberOfVotesInPool, pooledVotes := x.votePool.Add(voteMsg)
	if thresholdReached {
		log.Debug("Vote pool threashold reached: %v, number of items in the pool: %v", thresholdReached, numberOfVotesInPool)
		err := x.onVotePoolThresholdReached(chain, pooledVotes, voteMsg)
		if err != nil {
			return nil
		}
	}

	return nil
}

/*
	Function that will be called by votePool when it reached threshold.
	In the engine v2, we will need to generate and process QC
*/
func (x *XDPoS_v2) onVotePoolThresholdReached(chain consensus.ChainReader, pooledVotes map[common.Hash]utils.PoolObj, currentVoteMsg utils.PoolObj) error {
	signatures := []utils.Signature{}
	for _, v := range pooledVotes {
		signatures = append(signatures, v.(*utils.Vote).Signature)
	}
	// Genrate QC
	quorumCert := &utils.QuorumCert{
		ProposedBlockInfo: currentVoteMsg.(*utils.Vote).ProposedBlockInfo,
		Signatures:        signatures,
	}
	err := x.processQC(chain, quorumCert)
	if err != nil {
		log.Error("Error while processing QC in the Vote handler after reaching pool threshold, ", err)
		return err
	}
	log.Info("🗳 Successfully processed the vote and produced QC!")
	return nil
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
func (x *XDPoS_v2) VerifyTimeoutMessage(timeoutMsg *utils.Timeout) (bool, error) {
	return x.verifyMsgSignature(utils.TimeoutSigHash(&timeoutMsg.Round), timeoutMsg.Signature)
}

/*
	Entry point for handling timeout message to process below:
	1. checkRoundNumber()
	2. Collect timeout
	3. Once timeout pool reached threshold, it will trigger the call to the function "onTimeoutPoolThresholdReached"
*/
func (x *XDPoS_v2) TimeoutHandler(timeout *utils.Timeout) error {
	x.lock.Lock()
	defer x.lock.Unlock()

	// 1. checkRoundNumber
	if timeout.Round != x.currentRound {
		return fmt.Errorf("Timeout message round number: %v does not match currentRound: %v", timeout.Round, x.currentRound)
	}
	// Collect timeout, generate TC
	isThresholdReached, numberOfTimeoutsInPool, pooledTimeouts := x.timeoutPool.Add(timeout)
	// Threshold reached
	if isThresholdReached {
		log.Debug("Timeout pool threashold reached: %v, number of items in the pool: %v", isThresholdReached, numberOfTimeoutsInPool)
		err := x.onTimeoutPoolThresholdReached(pooledTimeouts, timeout)
		if err != nil {
			return err
		}
	}
	return nil
}

/*
	Function that will be called by timeoutPool when it reached threshold.
	In the engine v2, we will need to:
		1. Genrate TC
		2. processTC()
		3. generateSyncInfo()
*/
func (x *XDPoS_v2) onTimeoutPoolThresholdReached(pooledTimeouts map[common.Hash]utils.PoolObj, currentTimeoutMsg utils.PoolObj) error {
	signatures := []utils.Signature{}
	for _, v := range pooledTimeouts {
		signatures = append(signatures, v.(*utils.Timeout).Signature)
	}
	// Genrate TC
	timeoutCert := &utils.TimeoutCert{
		Round:      currentTimeoutMsg.(*utils.Timeout).Round,
		Signatures: signatures,
	}
	// Process TC
	err := x.processTC(timeoutCert)
	if err != nil {
		log.Error("Error while processing TC in the Timeout handler after reaching pool threshold, ", err.Error())
		return err
	}
	// Generate and broadcast syncInfo
	syncInfo := x.getSyncInfo()
	x.broadcastToBftChannel(syncInfo)

	log.Info("⏰ Successfully processed the timeout message and produced TC & SyncInfo!")
	return nil
}

/*
	Proposed Block workflow
*/
func (x *XDPoS_v2) ProposedBlockHandler(blockChainReader consensus.ChainReader, blockInfo *utils.BlockInfo, quorumCert *utils.QuorumCert) error {
	x.lock.Lock()
	defer x.lock.Unlock()

	/*
		1. processQC(): process the QC inside the proposed block
		2. verifyVotingRule(): the proposed block's info is extracted into BlockInfo and verified for voting
		3. sendVote()
	*/
	err := x.processQC(blockChainReader, quorumCert)
	if err != nil {
		return err
	}
	verified, err := x.verifyVotingRule(blockChainReader, blockInfo, quorumCert)
	if err != nil {
		return err
	}
	if verified {
		return x.sendVote(blockInfo)
	} else {
		log.Info("Failed to pass the voting rule verification", "ProposeBlockHash", blockInfo.Hash)
	}

	return nil
}

/*
	QC & TC Utils
*/

// To be used by different message verification. Verify local DB block info against the received block information(i.e hash, blockNum, round)
func (x *XDPoS_v2) VerifyBlockInfo(blockInfo *utils.BlockInfo) error {
	return nil
}

func (x *XDPoS_v2) verifyQC(quorumCert *utils.QuorumCert) error {
	/*
		1. Verify signer signatures: (List of signatures)
					- Use ecRecover to get the public key
					- Use the above public key to find out the xdc address
					- Use the above xdc address to check against the master node list(For the received QC epoch)
		2. Verify blockInfo
	*/
	return nil
}

func (x *XDPoS_v2) verifyTC(timeoutCert *utils.TimeoutCert) error {
	/*
		1. Verify signer signature: (List of signatures)
					- Use ecRecover to get the public key
					- Use the above public key to find out the xdc address
					- Use the above xdc address to check against the master node list(For the received TC epoch)
	*/
	return nil
}

// Update local QC variables including highestQC & lockQuorumCert, as well as commit the blocks that satisfy the algorithm requirements
func (x *XDPoS_v2) processQC(blockChainReader consensus.ChainReader, quorumCert *utils.QuorumCert) error {
	// 1. Update HighestQC
	if x.highestQuorumCert == nil || (quorumCert.ProposedBlockInfo.Round > x.highestQuorumCert.ProposedBlockInfo.Round) {
		x.highestQuorumCert = quorumCert
	}
	// 2. Get QC from header and update lockQuorumCert(lockQuorumCert is the parent of highestQC)
	proposedBlockHeader := blockChainReader.GetHeaderByHash(quorumCert.ProposedBlockInfo.Hash)
	// Extra field contain parent information
	var decodedExtraField utils.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(proposedBlockHeader.Extra, &decodedExtraField)
	if err != nil {
		return err
	}
	x.lockQuorumCert = &decodedExtraField.QuorumCert

	proposedBlockRound := &decodedExtraField.Round
	// 3. Update commit block info
	_, err = x.commitBlocks(blockChainReader, proposedBlockHeader, proposedBlockRound)
	if err != nil {
		return err
	}
	// 4. Set new round
	if quorumCert.ProposedBlockInfo.Round >= x.currentRound {
		err := x.setNewRound(quorumCert.ProposedBlockInfo.Round + 1)
		if err != nil {
			return err
		}
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
		err := x.setNewRound(timeoutCert.Round + 1)
		if err != nil {
			return err
		}
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
	x.timeoutWorker.Reset()
	//TODO: vote pools
	x.timeoutPool.Clear()
	return nil
}

// Hot stuff rule to decide whether this node is eligible to vote for the received block
func (x *XDPoS_v2) verifyVotingRule(blockChainReader consensus.ChainReader, blockInfo *utils.BlockInfo, quorumCert *utils.QuorumCert) (bool, error) {
	// Make sure this node has not voted for this round.
	if x.currentRound <= x.highestVotedRound {
		return false, nil
	}
	/*
		HotStuff Voting rule:
		header's round == local current round, AND (one of the following two:)
		header's block extends lockQuorumCert's ProposedBlockInfo (we need a isExtending(block_a, block_b) function), OR
		header's QC's ProposedBlockInfo.Round > lockQuorumCert's ProposedBlockInfo.Round
	*/
	if blockInfo.Round != x.currentRound {
		return false, nil
	}
	isExtended, err := x.isExtendingFromAncestor(blockChainReader, blockInfo, &x.lockQuorumCert.ProposedBlockInfo)
	if err != nil {
		return false, err
	}
	if isExtended || (quorumCert.ProposedBlockInfo.Round > x.lockQuorumCert.ProposedBlockInfo.Round) {
		return true, nil
	}

	return false, nil
}

// Once Hot stuff voting rule has verified, this node can then send vote
func (x *XDPoS_v2) sendVote(blockInfo *utils.BlockInfo) error {
	// First step: Update the highest Voted round
	// Second step: Generate the signature by using node's private key(The signature is the blockInfo signature)
	// Third step: Construct the vote struct with the above signature & blockinfo struct
	// Forth step: Send the vote to broadcast channel

	signedHash, err := x.signSignature(utils.VoteSigHash(blockInfo))
	if err != nil {
		return err
	}

	x.highestVotedRound = x.currentRound
	voteMsg := &utils.Vote{
		ProposedBlockInfo: *blockInfo,
		Signature:         signedHash,
	}
	x.broadcastToBftChannel(voteMsg)
	return nil
}

// Generate and send timeout into BFT channel.
/*
	1. timeout.round = currentRound
	2. Sign the signature
	3. send to broadcast channel
*/
func (x *XDPoS_v2) sendTimeout() error {
	signedHash, err := x.signSignature(utils.TimeoutSigHash(&x.currentRound))
	if err != nil {
		return err
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

func (x *XDPoS_v2) signSignature(signingHash common.Hash) (utils.Signature, error) {
	// Don't hold the signFn for the whole signing operation
	x.signLock.RLock()
	signer, signFn := x.signer, x.signFn
	x.signLock.RUnlock()

	signedHash, err := signFn(accounts.Account{Address: signer}, signingHash.Bytes())
	if err != nil {
		return nil, fmt.Errorf("Error while signing hash")
	}
	return signedHash, nil
}

func (x *XDPoS_v2) verifyMsgSignature(signedHashToBeVerified common.Hash, signature utils.Signature) (bool, error) {
	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(signedHashToBeVerified.Bytes(), signature)
	if err != nil {
		return false, fmt.Errorf("Error while verifying message: %v", err)
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
	Function that will be called by timer when countdown reaches its threshold.
	In the engine v2, we would need to broadcast timeout messages to other peers
*/
func (x *XDPoS_v2) onCountdownTimeout(time time.Time) error {
	x.lock.Lock()
	defer x.lock.Unlock()

	err := x.sendTimeout()
	if err != nil {
		log.Error("Error while sending out timeout message at time: ", time)
		return err
	}
	return nil
}

func (x *XDPoS_v2) broadcastToBftChannel(msg interface{}) {
	go func() {
		x.BroadcastCh <- msg
	}()
}

func (x *XDPoS_v2) getCurrentRoundMasterNodes() []common.Address {
	return []common.Address{}
}

func (x *XDPoS_v2) getSyncInfo() utils.SyncInfo {
	return utils.SyncInfo{
		HighestQuorumCert:  x.highestQuorumCert,
		HighestTimeoutCert: x.highestTimeoutCert,
	}
}

//TODO: find parent and grandparent and grandgrandparent block, check round number, if so, commit grandgrandparent
func (x *XDPoS_v2) commitBlocks(blockChainReader consensus.ChainReader, proposedBlockHeader *types.Header, proposedBlockRound *utils.Round) (bool, error) {
	// Find the last two parent block and check their rounds are the continous
	parentBlock := blockChainReader.GetHeaderByHash(proposedBlockHeader.ParentHash)

	var decodedExtraField utils.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(parentBlock.Extra, &decodedExtraField)
	if err != nil {
		return false, err
	}
	if *proposedBlockRound-1 != decodedExtraField.Round {
		return false, nil
	}

	// If parent round is continous, we check grandparent
	grandParentBlock := blockChainReader.GetHeaderByHash(parentBlock.ParentHash)
	err = utils.DecodeBytesExtraFields(grandParentBlock.Extra, &decodedExtraField)
	if err != nil {
		return false, err
	}
	if *proposedBlockRound-2 != decodedExtraField.Round {
		return false, nil
	}
	// TODO: Commit the grandParent block

	return true, nil
}

func (x *XDPoS_v2) isExtendingFromAncestor(blockChainReader consensus.ChainReader, currentBlock *utils.BlockInfo, ancestorBlock *utils.BlockInfo) (bool, error) {
	blockNumDiff := int(big.NewInt(0).Sub(currentBlock.Number, ancestorBlock.Number).Int64())

	nextBlockHash := currentBlock.Hash
	for i := 0; i < blockNumDiff; i++ {
		parentBlock := blockChainReader.GetHeaderByHash(nextBlockHash)
		if parentBlock == nil {
			return false, fmt.Errorf("Could not find its parent block when checking whether currentBlock %v is extending from the ancestorBlock %v", currentBlock.Number, ancestorBlock.Number)
		} else {
			nextBlockHash = parentBlock.ParentHash
		}
	}

	if nextBlockHash == ancestorBlock.Hash {
		return true, nil
	}
	return false, nil
}
