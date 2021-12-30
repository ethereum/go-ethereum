package engine_v2

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"sync"
	"time"

	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/common/countdown"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/consensus/clique"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
	lru "github.com/hashicorp/golang-lru"
)

type XDPoS_v2 struct {
	config *params.XDPoSConfig // Consensus engine configuration parameters
	db     ethdb.Database      // Database to store and retrieve snapshot checkpoints

	recents    *lru.ARCCache // Snapshots for recent block to speed up reorgs
	signatures *lru.ARCCache // Signatures of recent blocks to speed up mining

	signer   common.Address  // Ethereum address of the signing key
	signFn   clique.SignerFn // Signer function to authorize hashes with
	lock     sync.RWMutex    // Protects the signer fields
	signLock sync.RWMutex    // Protects the signer fields

	BroadcastCh   chan interface{}
	timeoutWorker *countdown.CountdownTimer // Timer to generate broadcast timeout msg if threashold reached

	timeoutPool       *utils.Pool
	votePool          *utils.Pool
	currentRound      utils.Round
	highestVotedRound utils.Round
	highestQuorumCert *utils.QuorumCert
	// lockQuorumCert in XDPoS Consensus 2.0, used in voting rule
	lockQuorumCert     *utils.QuorumCert
	highestTimeoutCert *utils.TimeoutCert
	highestCommitBlock *utils.BlockInfo

	HookReward func(chain consensus.ChainReader, state *state.StateDB, parentState *state.StateDB, header *types.Header) (error, map[string]interface{})
}

func New(config *params.XDPoSConfig, db ethdb.Database) *XDPoS_v2 {
	// Setup Timer
	duration := time.Duration(config.V2.TimeoutWorkerDuration) * time.Second
	timer := countdown.NewCountDown(duration)
	timeoutPool := utils.NewPool(config.V2.CertThreshold)

	recents, _ := lru.NewARC(utils.InmemorySnapshots)
	signatures, _ := lru.NewARC(utils.InmemorySnapshots)

	votePool := utils.NewPool(config.V2.CertThreshold)
	engine := &XDPoS_v2{
		config:     config,
		db:         db,
		signatures: signatures,

		recents:       recents,
		timeoutWorker: timer,
		BroadcastCh:   make(chan interface{}),
		timeoutPool:   timeoutPool,
		votePool:      votePool,

		highestTimeoutCert: &utils.TimeoutCert{
			Round:      utils.Round(0),
			Signatures: []utils.Signature{},
		},
		highestQuorumCert: &utils.QuorumCert{
			ProposedBlockInfo: &utils.BlockInfo{
				Hash:   common.Hash{},
				Round:  utils.Round(0),
				Number: big.NewInt(0),
			},
			Signatures: []utils.Signature{},
		},
		highestVotedRound:  utils.Round(0),
		highestCommitBlock: nil,
	}
	// Add callback to the timer
	timer.OnTimeoutFn = engine.onCountdownTimeout

	return engine
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (x *XDPoS_v2) Prepare(chain consensus.ChainReader, header *types.Header) error {
	// Verify mined block parent matches highest QC
	x.lock.Lock()
	// Check header if it is the first consensus v2 block, if so, assign initial values to current round and highestQC
	if header.Number.Cmp(big.NewInt(0).Add(x.config.XDPoSV2Block, big.NewInt(1))) == 0 {
		log.Info("[Prepare] Initilising highest QC for consensus v2 first block", "Block Num", header.Number.String(), "BlockHash", header.Hash())
		// Generate new parent blockInfo and put it into QC
		parentBlockInfo := &utils.BlockInfo{
			Hash:   header.ParentHash,
			Round:  utils.Round(0),
			Number: big.NewInt(0).Sub(header.Number, big.NewInt(1)),
		}
		quorumCert := &utils.QuorumCert{
			ProposedBlockInfo: parentBlockInfo,
			Signatures:        nil,
		}
		x.currentRound = 1
		x.highestQuorumCert = quorumCert
	}

	currentRound := x.currentRound
	highestQC := x.highestQuorumCert
	x.lock.Unlock()

	if (highestQC == nil) || (header.ParentHash != highestQC.ProposedBlockInfo.Hash) {
		return consensus.ErrNotReadyToPropose
	}

	extra := utils.ExtraFields_v2{
		Round:      currentRound,
		QuorumCert: highestQC,
	}

	header.Nonce = types.BlockNonce{}

	number := header.Number.Uint64()
	parent := chain.GetHeader(header.ParentHash, number-1)
	log.Info("Preparing new block!", "Number", number, "Parent Hash", parent.Hash())
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	// Set the correct difficulty
	header.Difficulty = x.calcDifficulty(chain, parent, x.signer)
	log.Debug("CalcDifficulty ", "number", header.Number, "difficulty", header.Difficulty)

	// TODO: previous round should sit on previous Epoch and x.currentRound should >= Epoch number
	if number%x.config.Epoch == 0 {
		snap, err := x.snapshot(chain, number-1, header.ParentHash, nil)
		if err != nil {
			return err
		}
		masternodes := snap.GetMasterNodes()
		//TODO: remove penalty nodes and add comeback nodes
		for _, v := range masternodes {
			header.Validators = append(header.Validators, v[:]...)
		}
	}

	extraBytes, err := extra.EncodeToBytes()
	if err != nil {
		return err
	}

	header.Extra = extraBytes

	// Mix digest is reserved for now, set to empty
	header.MixDigest = common.Hash{}

	// Ensure the timestamp has the correct delay

	//　TODO: if timestamp > current time, how to deal with future timestamp
	header.Time = new(big.Int).Add(parent.Time, new(big.Int).SetUint64(x.config.Period))
	if header.Time.Int64() < time.Now().Unix() {
		header.Time = big.NewInt(time.Now().Unix())
	}

	return nil
}

// Finalize implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given, and returns the final block.
func (x *XDPoS_v2) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, parentState *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	// set block reward
	number := header.Number.Uint64()
	rCheckpoint := chain.Config().XDPoS.RewardCheckpoint

	// _ = c.CacheData(header, txs, receipts)

	if x.HookReward != nil && number%rCheckpoint == 0 {
		err, rewards := x.HookReward(chain, state, parentState, header)
		if err != nil {
			return nil, err
		}
		if len(common.StoreRewardFolder) > 0 {
			data, err := json.Marshal(rewards)
			if err == nil {
				err = ioutil.WriteFile(filepath.Join(common.StoreRewardFolder, header.Number.String()+"."+header.Hash().Hex()), data, 0644)
			}
			if err != nil {
				log.Error("Error when save reward info ", "number", header.Number, "hash", header.Hash().Hex(), "err", err)
			}
		}
	}

	// the state remains as is and uncles are dropped
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	header.UncleHash = types.CalcUncleHash(nil)

	// Assemble and return the final block for sealing
	return types.NewBlock(header, txs, nil, receipts), nil
}

// Authorize injects a private key into the consensus engine to mint new blocks with.
func (x *XDPoS_v2) Authorize(signer common.Address, signFn clique.SignerFn) {
	x.signLock.Lock()
	defer x.signLock.Unlock()

	x.signer = signer
	x.signFn = signFn
}

func (x *XDPoS_v2) Author(header *types.Header) (common.Address, error) {
	return utils.EcrecoverV2(header, x.signatures)
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
func (x *XDPoS_v2) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	header := block.Header()

	// Sealing the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return nil, utils.ErrUnknownBlock
	}
	// For 0-period chains, refuse to seal empty blocks (no reward but would spin sealing)
	// checkpoint blocks have no tx
	if x.config.Period == 0 && len(block.Transactions()) == 0 && number%x.config.Epoch != 0 {
		return nil, utils.ErrWaitTransactions
	}
	// Don't hold the signer fields for the entire sealing procedure
	x.signLock.RLock()
	signer, signFn := x.signer, x.signFn
	x.signLock.RUnlock()

	// Bail out if we're unauthorized to sign a block
	snap, err := x.snapshot(chain, number-1, header.ParentHash, nil)
	if err != nil {
		return nil, err
	}
	masternodes := x.GetMasternodes(chain, header)
	if _, authorized := snap.MasterNodes[signer]; !authorized {
		valid := false
		for _, m := range masternodes {
			if m == signer {
				valid = true
				break
			}
		}
		if !valid {
			return nil, utils.ErrUnauthorized
		}
	}

	select {
	case <-stop:
		return nil, nil
	default:
	}

	// Sign all the things!
	signature, err := signFn(accounts.Account{Address: signer}, utils.SigHashV2(header).Bytes())
	if err != nil {
		return nil, err
	}
	header.Validator = signature

	return block.WithSeal(header), nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have based on the previous blocks in the chain and the
// current signer.
func (x *XDPoS_v2) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	return x.calcDifficulty(chain, parent, x.signer)
}

// TODO: what should be new difficulty
func (x *XDPoS_v2) calcDifficulty(chain consensus.ChainReader, parent *types.Header, signer common.Address) *big.Int {
	// TODO: The difference of round number between parent round and current round
	return big.NewInt(1)
}

// Copy from v1
func (x *XDPoS_v2) YourTurn(chain consensus.ChainReader, parent *types.Header, signer common.Address) (int, int, int, bool, error) {
	snap, err := x.GetSnapshot(chain, parent)
	if err != nil {
		log.Error("[YourTurn] Failed while getting snapshot", "parentHash", parent.Hash(), "err", err)
		return 0, -1, -1, false, err
	}
	masternodes := x.GetMasternodes(chain, parent)
	if len(masternodes) == 0 {
		return 0, -1, -1, false, errors.New("Masternodes not found")
	}
	pre := common.Address{}
	// masternode[0] has chance to create block 1
	preIndex := -1
	if parent.Number.Uint64() != 0 {
		pre, err = whoIsCreator(snap, parent)
		if err != nil {
			return 0, 0, 0, false, err
		}
		preIndex = utils.Position(masternodes, pre)
	}
	curIndex := utils.Position(masternodes, signer)
	if signer == x.signer {
		log.Debug("Masternodes cycle info", "number of masternodes", len(masternodes), "previous", pre, "position", preIndex, "current", signer, "position", curIndex)
	}
	for i, s := range masternodes {
		log.Debug("Masternode:", "index", i, "address", s.String())
	}
	if (preIndex+1)%len(masternodes) == curIndex {
		return len(masternodes), preIndex, curIndex, true, nil
	}
	return len(masternodes), preIndex, curIndex, false, nil
}

// Copy from v1
func whoIsCreator(snap *SnapshotV2, header *types.Header) (common.Address, error) {
	if header.Number.Uint64() == 0 {
		return common.Address{}, errors.New("Don't take block 0")
	}
	m, err := utils.EcrecoverV2(header, snap.sigcache)
	if err != nil {
		return common.Address{}, err
	}
	return m, nil
}

// Copy from v1
func (x *XDPoS_v2) GetMasternodes(chain consensus.ChainReader, header *types.Header) []common.Address {
	n := header.Number.Uint64()
	e := x.config.Epoch
	switch {
	case n%e == 0:
		return utils.GetMasternodesFromCheckpointHeader(header)
	case n%e != 0:
		h := chain.GetHeaderByNumber(n - (n % e))
		return utils.GetMasternodesFromCheckpointHeader(h)
	default:
		return []common.Address{}
	}
}

// Copy from v1
func (x *XDPoS_v2) GetSnapshot(chain consensus.ChainReader, header *types.Header) (*SnapshotV2, error) {
	number := header.Number.Uint64()
	log.Trace("get snapshot", "number", number, "hash", header.Hash())
	snap, err := x.snapshot(chain, number, header.Hash(), nil)
	if err != nil {
		return nil, err
	}
	return snap, nil
}

// snapshot retrieves the authorization snapshot at a given point in time.
func (x *XDPoS_v2) snapshot(chain consensus.ChainReader, number uint64, hash common.Hash, parents []*types.Header) (*SnapshotV2, error) {
	// Search for a SnapshotV2 in memory or on disk for checkpoints
	var (
		headers []*types.Header
		snap    *SnapshotV2
	)
	for snap == nil {
		// If an in-memory SnapshotV2 was found, use that
		if s, ok := x.recents.Get(hash); ok {
			snap = s.(*SnapshotV2)
			break
		}
		// If an on-disk checkpoint snapshot can be found, use that
		// checkpoint snapshot = checkpoint - gap
		if (number+x.config.Gap)%x.config.Epoch == 0 {
			if s, err := loadSnapshot(x.signatures, x.db, hash); err == nil {
				log.Trace("Loaded snapshot form disk", "number", number, "hash", hash)
				snap = s
				break
			}
		}
		// If we're at 0 block, make a snapshot
		// TODO: We may need to store snapshot at the v1 -> v2 switch block
		if number == 0 {
			genesis := chain.GetHeaderByNumber(0)
			if err := x.VerifyHeader(chain, genesis, true); err != nil {
				return nil, err
			}
			signers := make([]common.Address, (len(genesis.Extra)-utils.ExtraVanity-utils.ExtraSeal)/common.AddressLength)
			for i := 0; i < len(signers); i++ {
				copy(signers[i][:], genesis.Extra[utils.ExtraVanity+i*common.AddressLength:])
			}
			snap = newSnapshot(x.signatures, 0, genesis.Hash(), x.currentRound, x.highestQuorumCert, signers)
			if err := storeSnapshot(snap, x.db); err != nil {
				return nil, err
			}
			log.Trace("Stored genesis voting snapshot to disk")
			break
		}
		// No snapshot for this header, gather the header and move backward
		var header *types.Header
		if len(parents) > 0 {
			// If we have explicit parents, pick from there (enforced)
			header = parents[len(parents)-1]
			if header.Hash() != hash || header.Number.Uint64() != number {
				return nil, consensus.ErrUnknownAncestor
			}
			parents = parents[:len(parents)-1]
		} else {
			// No explicit parents (or no more left), reach out to the database
			header = chain.GetHeader(hash, number)
			if header == nil {
				log.Error("[Seal] Failed due to no header found", "hash", hash, "number", number)
				return nil, consensus.ErrUnknownAncestor
			}
		}
		headers = append(headers, header)
		number, hash = number-1, header.ParentHash
	}
	// Previous snapshot found, apply any pending headers on top of it
	for i := 0; i < len(headers)/2; i++ {
		headers[i], headers[len(headers)-1-i] = headers[len(headers)-1-i], headers[i]
	}
	snap, err := snap.apply(headers)
	if err != nil {
		return nil, err
	}
	x.recents.Add(snap.Hash, snap)

	// If we've generated a new checkpoint snapshot, save to disk
	// TODO how to save correct snapshot
	if uint64(snap.Round)%x.config.Epoch == x.config.Gap {
		if err = storeSnapshot(snap, x.db); err != nil {
			return nil, err
		}
		log.Trace("Stored snapshot to disk", "round number", snap.Round, "hash", snap.Hash)
	}
	return snap, err
}

func (x *XDPoS_v2) VerifyHeader(chain consensus.ChainReader, header *types.Header, fullVerify bool) error {
	return nil
}

// Utils for test to get current Pool size
func (x *XDPoS_v2) GetVotePoolSize(vote *utils.Vote) int {
	return x.votePool.Size(vote)
}

// Utils for test to get Timeout Pool Size
func (x *XDPoS_v2) GetTimeoutPoolSize(timeout *utils.Timeout) int {
	return x.timeoutPool.Size(timeout)
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
	return x.verifyMsgSignature(utils.VoteSigHash(vote.ProposedBlockInfo), vote.Signature)
}

// Consensus entry point for processing vote message to produce QC
func (x *XDPoS_v2) VoteHandler(chain consensus.ChainReader, voteMsg *utils.Vote) error {
	x.lock.Lock()
	defer x.lock.Unlock()
	return x.voteHandler(chain, voteMsg)
}

func (x *XDPoS_v2) voteHandler(chain consensus.ChainReader, voteMsg *utils.Vote) error {

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
			return err
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
	return x.timeoutHandler(timeout)
}

func (x *XDPoS_v2) timeoutHandler(timeout *utils.Timeout) error {
	// 1. checkRoundNumber
	if timeout.Round != x.currentRound {
		return &utils.ErrIncomingMessageRoundNotEqualCurrentRound{
			IncomingRound: timeout.Round,
			CurrentRound:  x.currentRound}
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
func (x *XDPoS_v2) ProposedBlockHandler(blockChainReader consensus.ChainReader, blockHeader *types.Header) error {
	x.lock.Lock()
	defer x.lock.Unlock()

	/*
		1. Verify QC
		2. Generate blockInfo
		3. processQC(): process the QC inside the proposed block
		4. verifyVotingRule(): the proposed block's info is extracted into BlockInfo and verified for voting
		5. sendVote()
	*/
	// Get QC and Round from Extra
	var decodedExtraField utils.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(blockHeader.Extra, &decodedExtraField)
	if err != nil {
		return err
	}
	quorumCert := decodedExtraField.QuorumCert
	round := decodedExtraField.Round

	err = x.verifyQC(quorumCert)
	if err != nil {
		log.Error("[ProposedBlockHandler] Fail to verify QC", "Extra round", round, "QC proposed BlockInfo Hash", quorumCert.ProposedBlockInfo.Hash)
		return err
	}

	// Generate blockInfo
	blockInfo := &utils.BlockInfo{
		Hash:   blockHeader.Hash(),
		Round:  round,
		Number: blockHeader.Number,
	}
	err = x.processQC(blockChainReader, quorumCert)
	if err != nil {
		log.Error("[ProposedBlockHandler] Fail to processQC", "QC proposed blockInfo round number", quorumCert.ProposedBlockInfo.Round, "QC proposed blockInfo hash", quorumCert.ProposedBlockInfo.Hash)
		return err
	}
	verified, err := x.verifyVotingRule(blockChainReader, blockInfo, quorumCert)
	if err != nil {
		return err
	}
	if verified {
		return x.sendVote(blockChainReader, blockInfo)
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
	log.Trace("[ProcessQC][Before]", "HighQC", x.highestQuorumCert)
	// 1. Update HighestQC
	if quorumCert.ProposedBlockInfo.Round > x.highestQuorumCert.ProposedBlockInfo.Round {
		x.highestQuorumCert = quorumCert
	}
	// 2. Get QC from header and update lockQuorumCert(lockQuorumCert is the parent of highestQC)
	proposedBlockHeader := blockChainReader.GetHeaderByHash(quorumCert.ProposedBlockInfo.Hash)
	if proposedBlockHeader.Number.Cmp(x.config.XDPoSV2Block) > 0 {
		// Extra field contain parent information
		var decodedExtraField utils.ExtraFields_v2
		err := utils.DecodeBytesExtraFields(proposedBlockHeader.Extra, &decodedExtraField)
		if err != nil {
			return err
		}
		if x.lockQuorumCert == nil || decodedExtraField.QuorumCert.ProposedBlockInfo.Round > x.lockQuorumCert.ProposedBlockInfo.Round {
			x.lockQuorumCert = decodedExtraField.QuorumCert
		}

		proposedBlockRound := &decodedExtraField.Round
		// 3. Update commit block info
		_, err = x.commitBlocks(blockChainReader, proposedBlockHeader, proposedBlockRound)
		if err != nil {
			log.Error("[processQC] Fail to commitBlocks", "proposedBlockRound", proposedBlockRound)
			return err
		}
	}
	// 4. Set new round
	if quorumCert.ProposedBlockInfo.Round >= x.currentRound {
		err := x.setNewRound(quorumCert.ProposedBlockInfo.Round + 1)
		if err != nil {
			log.Error("[processQC] Fail to setNewRound", "new round to set", quorumCert.ProposedBlockInfo.Round+1)
			return err
		}
	}
	log.Trace("[ProcessQC][After]", "HighQC", x.highestQuorumCert)
	return nil
}

/*
	1. Update highestTC
	2. Check TC round >= node's currentRound. If yes, call setNewRound
*/
func (x *XDPoS_v2) processTC(timeoutCert *utils.TimeoutCert) error {
	if timeoutCert.Round > x.highestTimeoutCert.Round {
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
	// XDPoS v1.0 switch to v2.0, the proposed block can always pass voting rule
	if x.lockQuorumCert == nil {
		return true, nil
	}
	isExtended, err := x.isExtendingFromAncestor(blockChainReader, blockInfo, x.lockQuorumCert.ProposedBlockInfo)
	if err != nil {
		return false, err
	}
	if isExtended || (quorumCert.ProposedBlockInfo.Round > x.lockQuorumCert.ProposedBlockInfo.Round) {
		return true, nil
	}

	return false, nil
}

// Once Hot stuff voting rule has verified, this node can then send vote
func (x *XDPoS_v2) sendVote(chainReader consensus.ChainReader, blockInfo *utils.BlockInfo) error {
	// First step: Update the highest Voted round
	// Second step: Generate the signature by using node's private key(The signature is the blockInfo signature)
	// Third step: Construct the vote struct with the above signature & blockinfo struct
	// Forth step: Send the vote to broadcast channel

	signedHash, err := x.signSignature(utils.VoteSigHash(blockInfo))
	if err != nil {
		log.Error("signSignature when sending out Vote", "BlockInfoHash", blockInfo.Hash, "Error", err)
		return err
	}

	x.highestVotedRound = x.currentRound
	voteMsg := &utils.Vote{
		ProposedBlockInfo: blockInfo,
		Signature:         signedHash,
	}

	err = x.voteHandler(chainReader, voteMsg)
	if err != nil {
		log.Error("sendVote error", "BlockInfoHash", blockInfo.Hash, "Error", err)
		return err
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
		log.Error("signSignature when sending out TC", "Error", err)
		return err
	}
	timeoutMsg := &utils.Timeout{
		Round:     x.currentRound,
		Signature: signedHash,
	}

	err = x.timeoutHandler(timeoutMsg)
	if err != nil {
		log.Error("TimeoutHandler error", "TimeoutRound", timeoutMsg.Round, "Error", err)
		return err
	}
	x.broadcastToBftChannel(timeoutMsg)
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

func (x *XDPoS_v2) getSyncInfo() *utils.SyncInfo {
	return &utils.SyncInfo{
		HighestQuorumCert:  x.highestQuorumCert,
		HighestTimeoutCert: x.highestTimeoutCert,
	}
}

//Find parent and grandparent, check round number, if so, commit grandparent(grandGrandParent of currentBlock)
func (x *XDPoS_v2) commitBlocks(blockChainReader consensus.ChainReader, proposedBlockHeader *types.Header, proposedBlockRound *utils.Round) (bool, error) {
	// XDPoS v1.0 switch to v2.0, skip commit
	if big.NewInt(0).Sub(proposedBlockHeader.Number, big.NewInt(2)).Cmp(x.config.XDPoSV2Block) <= 0 {
		return false, nil
	}
	// Find the last two parent block and check their rounds are the continuous
	parentBlock := blockChainReader.GetHeaderByHash(proposedBlockHeader.ParentHash)

	var decodedExtraField utils.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(parentBlock.Extra, &decodedExtraField)
	if err != nil {
		log.Error("Fail to execute first DecodeBytesExtraFields for commiting block", "ProposedBlockHash", proposedBlockHeader.Hash())
		return false, err
	}
	if *proposedBlockRound-1 != decodedExtraField.Round {
		log.Debug("[commitBlocks] Rounds not continuous(parent) found when committing block", "proposedBlockRound", proposedBlockRound, "decodedExtraField.Round", decodedExtraField.Round, "proposedBlockHeaderHash", proposedBlockHeader.Hash())
		return false, nil
	}

	// If parent round is continuous, we check grandparent
	grandParentBlock := blockChainReader.GetHeaderByHash(parentBlock.ParentHash)
	err = utils.DecodeBytesExtraFields(grandParentBlock.Extra, &decodedExtraField)
	if err != nil {
		log.Error("Fail to execute second DecodeBytesExtraFields for commiting block", "parentBlockHash", parentBlock.Hash())
		return false, err
	}
	if *proposedBlockRound-2 != decodedExtraField.Round {
		log.Debug("[commitBlocks] Rounds not continuous(grand parent) found when committing block", "proposedBlockRound", proposedBlockRound, "decodedExtraField.Round", decodedExtraField.Round, "proposedBlockHeaderHash", proposedBlockHeader.Hash())
		return false, nil
	}
	// Commit the grandParent block
	if x.highestCommitBlock == nil || (x.highestCommitBlock.Round < decodedExtraField.Round && x.highestCommitBlock.Number.Cmp(grandParentBlock.Number) == -1) {
		x.highestCommitBlock = &utils.BlockInfo{
			Number: grandParentBlock.Number,
			Hash:   grandParentBlock.Hash(),
			Round:  decodedExtraField.Round,
		}
		log.Debug("👴 Successfully committed block", "Committed block Hash", x.highestCommitBlock.Hash, "Committed round", x.highestCommitBlock.Round)
		return true, nil
	}
	// Everything else, fail to commit
	return false, nil
}

func (x *XDPoS_v2) isExtendingFromAncestor(blockChainReader consensus.ChainReader, currentBlock *utils.BlockInfo, ancestorBlock *utils.BlockInfo) (bool, error) {
	blockNumDiff := int(big.NewInt(0).Sub(currentBlock.Number, ancestorBlock.Number).Int64())

	nextBlockHash := currentBlock.Hash
	for i := 0; i < blockNumDiff; i++ {
		parentBlock := blockChainReader.GetHeaderByHash(nextBlockHash)
		if parentBlock == nil {
			return false, fmt.Errorf("Could not find its parent block when checking whether currentBlock %v with hash %v is extending from the ancestorBlock %v", currentBlock.Number, currentBlock.Hash, ancestorBlock.Number)
		} else {
			nextBlockHash = parentBlock.ParentHash
		}
		log.Debug("[isExtendingFromAncestor] Found parent block", "CurrentBlockHash", currentBlock.Hash, "ParentHash", nextBlockHash)
	}

	if nextBlockHash == ancestorBlock.Hash {
		return true, nil
	}
	return false, nil
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
func (x *XDPoS_v2) GetCurrentRound() utils.Round {
	return x.currentRound
}

// Utils for test to check currentRound value
func (x *XDPoS_v2) GetProperties() (utils.Round, *utils.QuorumCert, *utils.QuorumCert, utils.Round, *utils.BlockInfo) {
	x.lock.Lock()
	defer x.lock.Unlock()
	return x.currentRound, x.lockQuorumCert, x.highestQuorumCert, x.highestVotedRound, x.highestCommitBlock
}
