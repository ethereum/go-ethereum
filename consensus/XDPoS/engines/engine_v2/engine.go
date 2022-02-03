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

	snapshots     *lru.ARCCache // Snapshots for gap block
	signatures    *lru.ARCCache // Signatures of recent blocks to speed up mining
	epochSwitches *lru.ARCCache // infos of epoch: master nodes, epoch switch block info, parent of that info

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

	snapshots, _ := lru.NewARC(utils.InmemorySnapshots)
	signatures, _ := lru.NewARC(utils.InmemorySnapshots)
	epochSwitches, _ := lru.NewARC(int(utils.InmemoryEpochs))

	votePool := utils.NewPool(config.V2.CertThreshold)
	engine := &XDPoS_v2{
		config:     config,
		db:         db,
		signatures: signatures,

		snapshots:     snapshots,
		epochSwitches: epochSwitches,
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
	timer.OnTimeoutFn = engine.OnCountdownTimeout

	return engine
}

/* V2 Block
SignerFn is a signer callback function to request a hash to be signed by a
backing account.
type SignerFn func(accounts.Account, []byte) ([]byte, error)

sigHash returns the hash which is used as input for the delegated-proof-of-stake
signing. It is the hash of the entire header apart from the 65 byte signature
contained at the end of the extra data.
*/
func (x *XDPoS_v2) SignHash(header *types.Header) (hash common.Hash) {
	return sigHash(header)
}

func (x *XDPoS_v2) Initial(chain consensus.ChainReader, header *types.Header, masternodes []common.Address) error {
	log.Info("[Initial] initial v2 related parameters")

	if x.highestQuorumCert.ProposedBlockInfo.Round != 0 { //already initialized
		log.Warn("[Initial] Already initialized")
		return nil
	}

	x.lock.Lock()
	defer x.lock.Unlock()
	// Check header if it is the first consensus v2 block, if so, assign initial values to current round and highestQC

	log.Info("[Initial] highest QC for consensus v2 first block", "Block Num", header.Number.String(), "BlockHash", header.Hash())
	// Generate new parent blockInfo and put it into QC
	blockInfo := &utils.BlockInfo{
		Hash:   header.Hash(),
		Round:  utils.Round(0),
		Number: header.Number,
	}
	quorumCert := &utils.QuorumCert{
		ProposedBlockInfo: blockInfo,
		Signatures:        nil,
	}
	x.currentRound = 1
	x.highestQuorumCert = quorumCert

	// Initial snapshot
	lastGapNum := header.Number.Uint64() - header.Number.Uint64()%x.config.Epoch - x.config.Gap
	lastGapHeader := chain.GetHeaderByNumber(lastGapNum)

	snap := newSnapshot(lastGapNum, lastGapHeader.Hash(), x.currentRound, x.highestQuorumCert, masternodes)
	x.snapshots.Add(snap.Hash, snap)
	storeSnapshot(snap, x.db)
	return nil
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (x *XDPoS_v2) Prepare(chain consensus.ChainReader, header *types.Header) error {

	x.lock.RLock()
	currentRound := x.currentRound
	highestQC := x.highestQuorumCert
	x.lock.RUnlock()

	if header.ParentHash != highestQC.ProposedBlockInfo.Hash {
		return consensus.ErrNotReadyToPropose
	}

	extra := utils.ExtraFields_v2{
		Round:      currentRound,
		QuorumCert: highestQC,
	}

	extraBytes, err := extra.EncodeToBytes()
	if err != nil {
		return err
	}
	header.Extra = extraBytes

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

	isEpochSwitchBlock, _, err := x.IsEpochSwitch(header)
	if err != nil {
		log.Error("[Prepare] Error while trying to determine if header is an epoch switch during Prepare", "header", header, "Error", err)
		return err
	}
	if isEpochSwitchBlock {
		snap, err := x.getSnapshot(chain, number)
		if err != nil {
			return err
		}
		masternodes := snap.NextEpochMasterNodes
		//TODO: remove penalty nodes and add comeback nodes, or change this logic into yourturn function
		for _, v := range masternodes {
			header.Validators = append(header.Validators, v[:]...)
		}
	}

	// Mix digest is reserved for now, set to empty
	header.MixDigest = common.Hash{}

	// Ensure the timestamp has the correct delay
	// TODO: Proper deal with time
	// TODO: if timestamp > current time, how to deal with future timestamp
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
	return ecrecover(header, x.signatures)
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
	isEpochSwitch, _, err := x.IsEpochSwitch(header)
	if err != nil {
		log.Error("[Seal] Error while checking whether header is a epoch switch during sealing", "Header", header)
	}
	if x.config.Period == 0 && len(block.Transactions()) == 0 && !isEpochSwitch {
		return nil, utils.ErrWaitTransactions
	}
	// Don't hold the signer fields for the entire sealing procedure
	x.signLock.RLock()
	signer, signFn := x.signer, x.signFn
	x.signLock.RUnlock()

	// Bail out if we're unauthorized to sign a block
	masternodes := x.GetMasternodes(chain, header)
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

	select {
	case <-stop:
		return nil, nil
	default:
	}

	// Sign all the things!
	signature, err := signFn(accounts.Account{Address: signer}, sigHash(header).Bytes())
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

// Check if it's my turm to mine a block. Note: The second return value `preIndex` is useless in V2 engine
func (x *XDPoS_v2) YourTurn(chain consensus.ChainReader, parent *types.Header, signer common.Address) (bool, error) {
	x.lock.RLock()
	defer x.lock.RUnlock()

	round := x.currentRound
	isEpochSwitch, _, err := x.IsEpochSwitchAtRound(round, parent)
	if err != nil {
		log.Error("[YourTurn] check epoch switch at round failed", "Error", err)
		return false, err
	}
	var masterNodes []common.Address
	if isEpochSwitch {
		if x.config.XDPoSV2Block.Cmp(parent.Number) == 0 {
			snap, err := x.getSnapshot(chain, x.config.XDPoSV2Block.Uint64())
			if err != nil {
				log.Error("[YourTurn] Cannot find snapshot at gap num of last V1", "err", err, "number", x.config.XDPoSV2Block.Uint64())
				return false, err
			}
			// the initial snapshot of v1->v2 switch containes penalites node
			masterNodes = snap.NextEpochMasterNodes
		} else {
			snap, err := x.getSnapshot(chain, parent.Number.Uint64()+1)
			if err != nil {
				log.Error("[YourTurn] Cannot find snapshot at gap block", "err", err, "number", x.config.XDPoSV2Block.Uint64())
				return false, err
			}
			masterNodes = snap.NextEpochMasterNodes
			// TODO: calculate master nodes with penalty and comback
		}
	} else {
		// this block and parent belong to the same epoch
		masterNodes = x.GetMasternodes(chain, parent)
	}

	if len(masterNodes) == 0 {
		log.Error("[YourTurn] Fail to find any master nodes from current block round epoch", "Hash", parent.Hash(), "CurrentRound", round, "Number", parent.Number)
		return false, errors.New("Masternodes not found")
	}
	leaderIndex := uint64(round) % x.config.Epoch % uint64(len(masterNodes))

	curIndex := utils.Position(masterNodes, signer)
	if signer == x.signer {
		log.Debug("[YourTurn] masterNodes cycle info", "number of masternodes", len(masterNodes), "current", signer, "position", curIndex, "parentBlock", parent)
	}
	for i, s := range masterNodes {
		log.Debug("[YourTurn] Masternode:", "index", i, "address", s.String(), "parentBlockNum", parent.Number)
	}

	if masterNodes[leaderIndex] == signer {
		return true, nil
	}
	log.Warn("[YourTurn] Not authorised signer", "signer", signer, "MN", masterNodes, "Hash", parent.Hash(), "masterNodes[leaderIndex]", masterNodes[leaderIndex], "signer", signer)
	return false, nil
}

func (x *XDPoS_v2) IsAuthorisedAddress(chain consensus.ChainReader, header *types.Header, address common.Address) bool {
	x.lock.RLock()
	defer x.lock.RUnlock()

	var extraField utils.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(header.Extra, &extraField)
	if err != nil {
		log.Error("[IsAuthorisedAddress] Fail to decode v2 extra data", "Hash", header.Hash(), "Extra", header.Extra, "Error", err)
		return false
	}
	blockRound := extraField.Round

	masterNodes := x.GetMasternodes(chain, header)

	if len(masterNodes) == 0 {
		log.Error("[IsAuthorisedAddress] Fail to find any master nodes from current block round epoch", "Hash", header.Hash(), "Round", blockRound, "Number", header.Number)
		return false
	}
	// leaderIndex := uint64(blockRound) % x.config.Epoch % uint64(len(masterNodes))
	for index, masterNodeAddress := range masterNodes {
		if masterNodeAddress == address {
			log.Debug("[IsAuthorisedAddress] Found matching master node address", "index", index, "Address", address, "MasterNodes", masterNodes)
			return true
		}
	}

	log.Warn("Not authorised address", "Address", address, "MN", masterNodes, "Hash", header.Hash())
	return false
}

// Copy from v1
func (x *XDPoS_v2) GetSnapshot(chain consensus.ChainReader, header *types.Header) (*SnapshotV2, error) {
	number := header.Number.Uint64()
	log.Trace("get snapshot", "number", number)
	snap, err := x.getSnapshot(chain, number)
	if err != nil {
		return nil, err
	}
	return snap, nil
}

// snapshot retrieves the authorization snapshot at a given point in time.
func (x *XDPoS_v2) getSnapshot(chain consensus.ChainReader, number uint64) (*SnapshotV2, error) {
	// checkpoint snapshot = checkpoint - gap
	gapBlockNum := number - number%x.config.Epoch - x.config.Gap
	gapBlockHash := chain.GetHeaderByNumber(gapBlockNum).Hash()
	log.Debug("get snapshot from gap block", "number", gapBlockNum, "hash", gapBlockHash.Hex())

	// If an in-memory SnapshotV2 was found, use that
	if s, ok := x.snapshots.Get(gapBlockHash); ok {
		snap := s.(*SnapshotV2)
		log.Trace("Loaded snapshot from memory", "number", gapBlockNum, "hash", gapBlockHash)
		return snap, nil
	}
	// If an on-disk checkpoint snapshot can be found, use that
	snap, err := loadSnapshot(x.signatures, x.db, gapBlockHash)
	if err != nil {
		log.Error("Cannot find snapshot from last gap block", "err", err, "number", gapBlockNum, "hash", gapBlockHash)
		return nil, err
	}

	log.Trace("Loaded snapshot from disk", "number", gapBlockNum, "hash", gapBlockHash)
	x.snapshots.Add(snap.Hash, snap)
	return snap, nil
}

func (x *XDPoS_v2) UpdateMasternodes(chain consensus.ChainReader, header *types.Header, ms []utils.Masternode) error {
	number := header.Number.Uint64()
	log.Trace("take snapshot", "number", number, "hash", header.Hash())

	masterNodes := []common.Address{}
	for _, m := range ms {
		masterNodes = append(masterNodes, m.Address)
	}

	x.lock.RLock()
	snap := newSnapshot(number, header.Hash(), x.currentRound, x.highestQuorumCert, masterNodes)
	x.lock.RUnlock()

	storeSnapshot(snap, x.db)
	x.snapshots.Add(snap.Hash, snap)

	nm := []string{}
	for _, n := range ms {
		nm = append(nm, n.Address.String())
	}
	log.Info("New set of masternodes has been updated to snapshot", "number", snap.Number, "hash", snap.Hash, "new masternodes", nm)

	return nil
}

func (x *XDPoS_v2) VerifyHeader(chain consensus.ChainReader, header *types.Header, fullVerify bool) error {
	return nil
}

// TODO: Yet to be implemented XIN-135
func (x *XDPoS_v2) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, fullVerifies []bool, abort <-chan struct{}, results chan<- error) {
	go func() {
		for range headers {
			select {
			case <-abort:
				return
			case results <- nil:
			}
		}
	}()
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
func (x *XDPoS_v2) VerifySyncInfoMessage(chain consensus.ChainReader, syncInfo *utils.SyncInfo) error {
	/*
		1. Verify items including:
				- verifyQC
				- verifyTC
		2. Broadcast(Not part of consensus)
	*/
	err := x.verifyQC(chain, syncInfo.HighestQuorumCert)
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
	x.lock.Lock()
	defer x.lock.Unlock()
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
func (x *XDPoS_v2) VerifyVoteMessage(chain consensus.ChainReader, vote *utils.Vote) (bool, error) {
	/*
		  1. Get masterNode list from snapshot
		  2. Check signature:
					- Use ecRecover to get the public key
					- Use the above public key to find out the xdc address
					- Use the above xdc address to check against the master node list from step 1(For the running epoch)
			4. Broadcast(Not part of consensus)
	*/
	snapshot, err := x.getSnapshot(chain, vote.ProposedBlockInfo.Number.Uint64())
	if err != nil {
		log.Error("[VerifyVoteMessage] fail to get snapshot for a vote message", "BlockNum", vote.ProposedBlockInfo.Number, "Hash", vote.ProposedBlockInfo.Hash, "Error", err.Error())
	}
	verified, err := x.verifyMsgSignature(utils.VoteSigHash(vote.ProposedBlockInfo), vote.Signature, snapshot.NextEpochMasterNodes)
	if err != nil {
		log.Error("[VerifyVoteMessage] Error while verifying vote message", "Error", err.Error())
	}
	return verified, err
}

// Consensus entry point for processing vote message to produce QC
func (x *XDPoS_v2) VoteHandler(chain consensus.ChainReader, voteMsg *utils.Vote) error {
	x.lock.Lock()
	defer x.lock.Unlock()
	return x.voteHandler(chain, voteMsg)
}

func (x *XDPoS_v2) voteHandler(chain consensus.ChainReader, voteMsg *utils.Vote) error {

	// 1. checkRoundNumber
	if (voteMsg.ProposedBlockInfo.Round != x.currentRound) && (voteMsg.ProposedBlockInfo.Round != x.currentRound+1) {
		return &utils.ErrIncomingMessageRoundTooFarFromCurrentRound{
			Type:          "vote",
			IncomingRound: voteMsg.ProposedBlockInfo.Round,
			CurrentRound:  x.currentRound,
		}
	}

	// Collect vote
	thresholdReached, numberOfVotesInPool, pooledVotes := x.votePool.Add(voteMsg)
	if thresholdReached {
		log.Info(fmt.Sprintf("Vote pool threashold reached: %v, number of items in the pool: %v", thresholdReached, numberOfVotesInPool))

		// Check if the block already exist, otherwise we try luck with the next vote
		proposedBlockHeader := chain.GetHeaderByHash(voteMsg.ProposedBlockInfo.Hash)
		if proposedBlockHeader == nil {
			log.Warn("[voteHandler] The proposed block from vote message does not exist yet, wait for the next vote to try again", "Hash", voteMsg.ProposedBlockInfo.Hash, "Round", voteMsg.ProposedBlockInfo.Round)
			return nil
		}

		err := x.onVotePoolThresholdReached(chain, pooledVotes, voteMsg, proposedBlockHeader)
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
func (x *XDPoS_v2) onVotePoolThresholdReached(chain consensus.ChainReader, pooledVotes map[common.Hash]utils.PoolObj, currentVoteMsg utils.PoolObj, proposedBlockHeader *types.Header) error {

	masternodes := x.GetMasternodes(chain, proposedBlockHeader)

	// Filter out non-Master nodes signatures
	var wg sync.WaitGroup
	wg.Add(len(pooledVotes))
	signatureSlice := make([]utils.Signature, len(pooledVotes))
	counter := 0
	for h, vote := range pooledVotes {
		go func(hash common.Hash, v *utils.Vote, i int) {
			defer wg.Done()
			verified, err := x.verifyMsgSignature(utils.VoteSigHash(v.ProposedBlockInfo), v.Signature, masternodes)
			if !verified || err != nil {
				log.Warn("[onVotePoolThresholdReached] Skip not verified vote signatures when building QC", "Error", err.Error(), "verified", verified)
			} else {
				signatureSlice[i] = v.Signature
			}
		}(h, vote.(*utils.Vote), counter)
		counter++
	}
	wg.Wait()

	// The signature list may contain empty entey. we only care the ones with values
	var validSignatureSlice []utils.Signature
	for _, v := range signatureSlice {
		if len(v) != 0 {
			validSignatureSlice = append(validSignatureSlice, v)
		}
	}

	// Skip and wait for the next vote to process again if valid votes is less than what we required
	if len(validSignatureSlice) < x.config.V2.CertThreshold {
		log.Warn("[onVotePoolThresholdReached] Not enough valid signatures to generate QC", "VotesSignaturesAfterFilter", validSignatureSlice, "NumberOfValidVotes", len(validSignatureSlice), "NumberOfVotes", len(pooledVotes))
		return nil
	}
	// Genrate QC
	quorumCert := &utils.QuorumCert{
		ProposedBlockInfo: currentVoteMsg.(*utils.Vote).ProposedBlockInfo,
		Signatures:        validSignatureSlice,
	}
	err := x.processQC(chain, quorumCert)
	if err != nil {
		log.Error("Error while processing QC in the Vote handler after reaching pool threshold, ", err)
		return err
	}
	log.Info("🗳 Successfully processed the vote and produced QC!")
	// clean up vote at the same poolKey. and pookKey is proposed block hash
	x.votePool.ClearPoolKeyByObj(currentVoteMsg)
	return nil
}

/*
	Timeout workflow
*/
// Verify timeout message type from peers in bft.go
/*
		1. Get master node list by timeout msg round
	  2. Check signature:
				- Use ecRecover to get the public key
				- Use the above public key to find out the xdc address
				- Use the above xdc address to check against the master node list from step 1(For the running epoch)
		3. Broadcast(Not part of consensus)
*/
func (x *XDPoS_v2) VerifyTimeoutMessage(chain consensus.ChainReader, timeoutMsg *utils.Timeout) (bool, error) {

	masternodes := x.GetMasternodesAtRound(chain, timeoutMsg.Round, chain.CurrentHeader())
	return x.verifyMsgSignature(utils.TimeoutSigHash(&timeoutMsg.Round), timeoutMsg.Signature, masternodes)
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
			Type:          "timeout",
			IncomingRound: timeout.Round,
			CurrentRound:  x.currentRound,
		}
	}
	// Collect timeout, generate TC
	isThresholdReached, numberOfTimeoutsInPool, pooledTimeouts := x.timeoutPool.Add(timeout)
	// Threshold reached
	if isThresholdReached {
		log.Info(fmt.Sprintf("Timeout pool threashold reached: %v, number of items in the pool: %v", isThresholdReached, numberOfTimeoutsInPool))
		err := x.onTimeoutPoolThresholdReached(pooledTimeouts, timeout)
		if err != nil {
			return err
		}
		// clean up timeout message at the same poolKey. and pookKey is proposed block hash
		x.timeoutPool.ClearPoolKeyByObj(timeout)
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

	err = x.verifyQC(blockChainReader, quorumCert)
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
	/*
		1. Check if is able to get header by hash from the chain
		2. Check the header from step 1 matches what's in the blockInfo. This includes the block number and the round
	*/
	return nil
}

func (x *XDPoS_v2) verifyQC(blockChainReader consensus.ChainReader, quorumCert *utils.QuorumCert) error {
	/*
		1. Check if num of QC signatures is >= x.config.v2.CertThreshold
		2. Get epoch master node list by hash
		3. Verify signer signatures: (List of signatures)
					- Use ecRecover to get the public key
					- Use the above public key to find out the xdc address
					- Use the above xdc address to check against the master node list from step 1(For the received QC epoch)
		4. Verify blockInfo
	*/
	epochInfo, err := x.getEpochSwitchInfo(blockChainReader, nil, quorumCert.ProposedBlockInfo.Hash)
	if err != nil {
		log.Error("[verifyQC] Error when getting epoch switch Info to verify QC", "Error", err)
		return fmt.Errorf("Fail to verify QC due to failure in getting epoch switch info")
	}

	var wg sync.WaitGroup
	wg.Add(len(quorumCert.Signatures))
	var haveError error

	for _, signature := range quorumCert.Signatures {
		go func(sig utils.Signature) {
			defer wg.Done()
			verified, err := x.verifyMsgSignature(utils.VoteSigHash(quorumCert.ProposedBlockInfo), sig, epochInfo.Masternodes)
			if err != nil {
				log.Error("[verifyQC] Error while verfying QC message signatures", "Error", err)
				haveError = fmt.Errorf("Error while verfying QC message signatures")
				return
			}
			if !verified {
				log.Warn("[verifyQC] Signature not verified doing QC verification", "QC", quorumCert)
				haveError = fmt.Errorf("Fail to verify QC due to signature mis-match")
				return
			}
		}(signature)
	}
	wg.Wait()
	if haveError != nil {
		return haveError
	}

	return x.VerifyBlockInfo(quorumCert.ProposedBlockInfo)
}

// TODO: Unhold, wait till proposal finalise
func (x *XDPoS_v2) verifyTC(timeoutCert *utils.TimeoutCert) error {
	/*
		1. Get epoch master node list by round/number with chain's current header
		2. Verify signer signature: (List of signatures)
					- Use ecRecover to get the public key
					- Use the above public key to find out the xdc address
					- Use the above xdc address to check against the master node list from step 1(For the received TC epoch)
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

	if quorumCert.ProposedBlockInfo.Round > x.lockQuorumCert.ProposedBlockInfo.Round {
		return true, nil
	}

	isExtended, err := x.isExtendingFromAncestor(blockChainReader, blockInfo, x.lockQuorumCert.ProposedBlockInfo)
	if err != nil {
		return false, err
	}
	if isExtended {
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

func (x *XDPoS_v2) verifyMsgSignature(signedHashToBeVerified common.Hash, signature utils.Signature, masternodes []common.Address) (bool, error) {
	if len(masternodes) == 0 {
		return false, fmt.Errorf("Empty masternode list detected when verifying message signatures")
	}
	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(signedHashToBeVerified.Bytes(), signature)
	if err != nil {
		return false, fmt.Errorf("Error while verifying message: %v", err)
	}
	var signerAddress common.Address
	copy(signerAddress[:], crypto.Keccak256(pubkey[1:])[12:])
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
func (x *XDPoS_v2) OnCountdownTimeout(time time.Time) error {
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

func (x *XDPoS_v2) GetMasternodesAtRound(chain consensus.ChainReader, round utils.Round, currentHeader *types.Header) []common.Address {
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
	x.lock.RLock()
	defer x.lock.RUnlock()
	return x.currentRound
}

// Utils for test to check currentRound value
func (x *XDPoS_v2) GetProperties() (utils.Round, *utils.QuorumCert, *utils.QuorumCert, utils.Round, *utils.BlockInfo) {
	x.lock.RLock()
	defer x.lock.RUnlock()
	return x.currentRound, x.lockQuorumCert, x.highestQuorumCert, x.highestVotedRound, x.highestCommitBlock
}

// Get master nodes over extra data of epoch switch block.
func (x *XDPoS_v2) GetMasternodesFromEpochSwitchHeader(epochSwitchHeader *types.Header) []common.Address {
	if epochSwitchHeader == nil {
		log.Error("[GetMasternodesFromEpochSwitchHeader] use nil epoch switch block to get master nodes")
		return []common.Address{}
	}
	masternodes := make([]common.Address, len(epochSwitchHeader.Validators)/common.AddressLength)
	for i := 0; i < len(masternodes); i++ {
		copy(masternodes[i][:], epochSwitchHeader.Validators[i*common.AddressLength:])
	}

	return masternodes
}

func (x *XDPoS_v2) IsEpochSwitch(header *types.Header) (bool, uint64, error) {
	// Return true directly if we are examing the last v1 block. This could happen if the calling function is examing parent block
	if header.Number.Cmp(x.config.XDPoSV2Block) == 0 {
		log.Info("[IsEpochSwitch] examing last v1 block 👯‍♂️")
		return true, header.Number.Uint64() / x.config.Epoch, nil
	}

	var decodedExtraField utils.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(header.Extra, &decodedExtraField)
	if err != nil {
		log.Error("[IsEpochSwitch] decode header error", "err", err, "header", header, "extra", common.Bytes2Hex(header.Extra))
		return false, 0, err
	}
	parentRound := decodedExtraField.QuorumCert.ProposedBlockInfo.Round
	round := decodedExtraField.Round
	epochStartRound := round - round%utils.Round(x.config.Epoch)
	epochNum := x.config.XDPoSV2Block.Uint64()/x.config.Epoch + uint64(round)/x.config.Epoch
	// if parent is last v1 block and this is first v2 block, this is treated as epoch switch
	if decodedExtraField.QuorumCert.ProposedBlockInfo.Number.Cmp(x.config.XDPoSV2Block) == 0 {
		log.Info("[IsEpochSwitch] true, parent equals XDPoSV2Block", "round", round, "number", header.Number.Uint64(), "hash", header.Hash())
		return true, epochNum, nil
	}
	log.Info("[IsEpochSwitch]", "parent round", parentRound, "round", round, "number", header.Number.Uint64(), "hash", header.Hash())
	return parentRound < epochStartRound, epochNum, nil
}

// IsEpochSwitchAtRound() is used by miner to check whether it mines a block in the same epoch with parent
func (x *XDPoS_v2) IsEpochSwitchAtRound(round utils.Round, parentHeader *types.Header) (bool, uint64, error) {
	epochNum := x.config.XDPoSV2Block.Uint64()/x.config.Epoch + uint64(round)/x.config.Epoch
	// if parent is last v1 block and this is first v2 block, this is treated as epoch switch
	if parentHeader.Number.Cmp(x.config.XDPoSV2Block) == 0 {
		return true, epochNum, nil
	}
	var decodedExtraField utils.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(parentHeader.Extra, &decodedExtraField)
	if err != nil {
		log.Error("[IsEpochSwitch] decode header error", "err", err, "header", parentHeader, "extra", common.Bytes2Hex(parentHeader.Extra))
		return false, 0, err
	}
	parentRound := decodedExtraField.Round
	epochStartRound := round - round%utils.Round(x.config.Epoch)
	return parentRound < epochStartRound, epochNum, nil
}

// Given header and its hash, get epoch switch info from the epoch switch block of that epoch,
// header is allow to be nil.
func (x *XDPoS_v2) getEpochSwitchInfo(chain consensus.ChainReader, header *types.Header, hash common.Hash) (*utils.EpochSwitchInfo, error) {
	e, ok := x.epochSwitches.Get(hash)
	if ok {
		log.Debug("[getEpochSwitchInfo] cache hit", "hash", hash.Hex())
		epochSwitchInfo := e.(*utils.EpochSwitchInfo)
		return epochSwitchInfo, nil
	}
	h := header
	if h == nil {
		log.Debug("[getEpochSwitchInfo] header missing, get header", "hash", hash.Hex())
		h = chain.GetHeaderByHash(hash)
	}
	isEpochSwitch, _, err := x.IsEpochSwitch(h)
	if err != nil {
		return nil, err
	}
	if isEpochSwitch {
		log.Debug("[getEpochSwitchInfo] header is epoch switch", "hash", hash.Hex(), "number", h.Number.Uint64())
		var epochSwitchInfo *utils.EpochSwitchInfo
		// Special case, in case of last v1 block, we manually build the epoch switch info
		if h.Number.Cmp(x.config.XDPoSV2Block) == 0 {
			masternodes := decodeMasternodesFromHeaderExtra(h)
			epochSwitchInfo = &utils.EpochSwitchInfo{
				Masternodes: masternodes,
				EpochSwitchBlockInfo: &utils.BlockInfo{
					Hash:   hash,
					Number: h.Number,
					Round:  utils.Round(0),
				},
				EpochSwitchParentBlockInfo: nil,
			}
		} else { // v2 normal flow
			masternodes := x.GetMasternodesFromEpochSwitchHeader(h)
			// create the epoch switch info and cache it
			var decodedExtraField utils.ExtraFields_v2
			err = utils.DecodeBytesExtraFields(h.Extra, &decodedExtraField)
			if err != nil {
				return nil, err
			}
			epochSwitchInfo = &utils.EpochSwitchInfo{
				Masternodes: masternodes,
				EpochSwitchBlockInfo: &utils.BlockInfo{
					Hash:   hash,
					Number: h.Number,
					Round:  decodedExtraField.Round,
				},
				EpochSwitchParentBlockInfo: decodedExtraField.QuorumCert.ProposedBlockInfo,
			}
		}

		x.epochSwitches.Add(hash, epochSwitchInfo)
		return epochSwitchInfo, nil
	}
	epochSwitchInfo, err := x.getEpochSwitchInfo(chain, nil, h.ParentHash)
	if err != nil {
		log.Error("[getEpochSwitchInfo] recursive error", "err", err, "hash", hash.Hex(), "number", h.Number.Uint64())
		return nil, err
	}
	log.Debug("[getEpochSwitchInfo] get epoch switch info recursively", "hash", hash.Hex(), "number", h.Number.Uint64())
	x.epochSwitches.Add(hash, epochSwitchInfo)
	return epochSwitchInfo, nil
}

// Given header, get master node from the epoch switch block of that epoch
func (x *XDPoS_v2) GetMasternodes(chain consensus.ChainReader, header *types.Header) []common.Address {
	epochSwitchInfo, err := x.getEpochSwitchInfo(chain, header, header.Hash())
	if err != nil {
		log.Error("[GetMasternodes] Adaptor v2 getEpochSwitchInfo has error, potentially bug", "err", err)
		return []common.Address{}
	}
	return epochSwitchInfo.Masternodes
}

func (x *XDPoS_v2) GetCurrentEpochSwitchBlock(chain consensus.ChainReader, blockNum *big.Int) (uint64, uint64, error) {
	header := chain.GetHeaderByNumber(blockNum.Uint64())
	epochSwitchInfo, err := x.getEpochSwitchInfo(chain, header, header.Hash())
	if err != nil {
		log.Error("[GetCurrentEpochSwitchBlock] Fail to get epoch switch info", "Num", header.Number, "Hash", header.Hash())
		return 0, 0, err
	}

	currentCheckpointNumber := epochSwitchInfo.EpochSwitchBlockInfo.Number.Uint64()
	epochNum := x.config.XDPoSV2Block.Uint64()/x.config.Epoch + uint64(epochSwitchInfo.EpochSwitchBlockInfo.Round)/x.config.Epoch
	return currentCheckpointNumber, epochNum, nil
}
