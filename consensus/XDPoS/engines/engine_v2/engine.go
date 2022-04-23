package engine_v2

import (
	"encoding/json"
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
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
	lru "github.com/hashicorp/golang-lru"
)

type XDPoS_v2 struct {
	config       *params.XDPoSConfig // Consensus engine configuration parameters
	db           ethdb.Database      // Database to store and retrieve snapshot checkpoints
	isInitilised bool                // status of v2 variables

	snapshots       *lru.ARCCache // Snapshots for gap block
	signatures      *lru.ARCCache // Signatures of recent blocks to speed up mining
	epochSwitches   *lru.ARCCache // infos of epoch: master nodes, epoch switch block info, parent of that info
	verifiedHeaders *lru.ARCCache

	signer   common.Address  // Ethereum address of the signing key
	signFn   clique.SignerFn // Signer function to authorize hashes with
	lock     sync.RWMutex    // Protects the signer fields
	signLock sync.RWMutex    // Protects the signer fields

	BroadcastCh  chan interface{}
	waitPeriodCh chan int

	timeoutWorker *countdown.CountdownTimer // Timer to generate broadcast timeout msg if threashold reached
	timeoutCount  int                       // number of timeout being sent

	timeoutPool           *utils.Pool
	votePool              *utils.Pool
	currentRound          utils.Round
	highestSelfMinedRound utils.Round
	highestVotedRound     utils.Round
	highestQuorumCert     *utils.QuorumCert
	// lockQuorumCert in XDPoS Consensus 2.0, used in voting rule
	lockQuorumCert     *utils.QuorumCert
	highestTimeoutCert *utils.TimeoutCert
	highestCommitBlock *utils.BlockInfo

	HookReward  func(chain consensus.ChainReader, state *state.StateDB, parentState *state.StateDB, header *types.Header) (map[string]interface{}, error)
	HookPenalty func(chain consensus.ChainReader, number *big.Int, parentHash common.Hash, candidates []common.Address) ([]common.Address, error)

	forensics *Forensics
}

func New(config *params.XDPoSConfig, db ethdb.Database, waitPeriodCh chan int) *XDPoS_v2 {
	// Setup timeoutTimer
	duration := time.Duration(config.V2.TimeoutPeriod) * time.Second
	timeoutTimer := countdown.NewCountDown(duration)

	snapshots, _ := lru.NewARC(utils.InmemorySnapshots)
	signatures, _ := lru.NewARC(utils.InmemorySnapshots)
	epochSwitches, _ := lru.NewARC(int(utils.InmemoryEpochs))
	verifiedHeaders, _ := lru.NewARC(utils.InmemorySnapshots)

	timeoutPool := utils.NewPool(config.V2.CertThreshold)
	votePool := utils.NewPool(config.V2.CertThreshold)
	engine := &XDPoS_v2{
		config:       config,
		db:           db,
		isInitilised: false,

		signatures: signatures,

		verifiedHeaders: verifiedHeaders,
		snapshots:       snapshots,
		epochSwitches:   epochSwitches,
		timeoutWorker:   timeoutTimer,
		BroadcastCh:     make(chan interface{}),
		waitPeriodCh:    waitPeriodCh,

		timeoutPool: timeoutPool,
		votePool:    votePool,

		highestSelfMinedRound: utils.Round(0),

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
			GapNumber:  0,
		},
		highestVotedRound:  utils.Round(0),
		highestCommitBlock: nil,
		forensics:          NewForensics(),
	}
	// Add callback to the timer
	timeoutTimer.OnTimeoutFn = engine.OnCountdownTimeout

	engine.periodicJob()

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

func (x *XDPoS_v2) Initial(chain consensus.ChainReader, header *types.Header) error {
	log.Info("[Initial] initial v2 related parameters")

	if x.highestQuorumCert.ProposedBlockInfo.Hash != (common.Hash{}) { // already initialized
		log.Error("[Initial] Already initialized", "x.highestQuorumCert.ProposedBlockInfo.Hash", x.highestQuorumCert.ProposedBlockInfo.Hash)
		return nil
	}

	var quorumCert *utils.QuorumCert
	var err error

	if header.Number.Int64() == x.config.V2.SwitchBlock.Int64() {
		log.Info("[Initial] highest QC for consensus v2 first block")
		blockInfo := &utils.BlockInfo{
			Hash:   header.Hash(),
			Round:  utils.Round(0),
			Number: header.Number,
		}
		quorumCert = &utils.QuorumCert{
			ProposedBlockInfo: blockInfo,
			Signatures:        nil,
			GapNumber:         header.Number.Uint64() - x.config.Gap,
		}

		// can not call processQC because round is equal to default
		x.currentRound = 1
		x.highestQuorumCert = quorumCert

	} else {
		log.Info("[Initial] highest QC from current header")
		quorumCert, _, _, err = x.getExtraFields(header)
		if err != nil {
			return err
		}
		err = x.processQC(chain, quorumCert)
		if err != nil {
			return err
		}
	}

	// Initial first v2 snapshot
	if header.Number.Uint64() < x.config.V2.SwitchBlock.Uint64()+x.config.Gap {

		checkpointBlockNumber := header.Number.Uint64() - header.Number.Uint64()%x.config.Epoch
		checkpointHeader := chain.GetHeaderByNumber(checkpointBlockNumber)

		lastGapNum := checkpointBlockNumber - x.config.Gap
		lastGapHeader := chain.GetHeaderByNumber(lastGapNum)

		log.Info("[Initial] init first snapshot")
		_, _, masternodes, err := x.getExtraFields(checkpointHeader)
		if err != nil {
			log.Error("[Initial] Error while get masternodes", "error", err)
			return err
		}
		snap := newSnapshot(lastGapNum, lastGapHeader.Hash(), masternodes)
		x.snapshots.Add(snap.Hash, snap)
		err = storeSnapshot(snap, x.db)
		if err != nil {
			log.Error("[Initial] Error while store snapshot", "error", err)
			return err
		}
	}

	// Initial timeout
	log.Info("[Initial] miner wait period", "period", x.config.V2.WaitPeriod)
	// avoid deadlock
	go func() {
		x.waitPeriodCh <- x.config.V2.WaitPeriod
	}()

	// Kick-off the countdown timer
	x.timeoutWorker.Reset(chain)

	log.Info("[Initial] finish initialisation")

	return nil
}

// Check if it's my turn to mine a block. Note: The second return value `preIndex` is useless in V2 engine
func (x *XDPoS_v2) YourTurn(chain consensus.ChainReader, parent *types.Header, signer common.Address) (bool, error) {
	x.lock.RLock()
	defer x.lock.RUnlock()

	if !x.isInitilised {
		err := x.Initial(chain, parent)
		if err != nil {
			log.Error("[YourTurn] Error while initialising last v2 variables", "ParentBlockHash", parent.Hash(), "Error", err)
			return false, err
		}
		x.isInitilised = true
	}

	waitedTime := time.Now().Unix() - parent.Time.Int64()
	if waitedTime < int64(x.config.V2.MinePeriod) {
		log.Trace("[YourTurn] wait after mine period", "minePeriod", x.config.V2.MinePeriod, "waitedTime", waitedTime)
		return false, nil
	}

	round := x.currentRound
	isMyTurn, err := x.yourturn(chain, round, parent, signer)
	if err != nil {
		log.Warn("[Yourturn] Error while checking if i am qualified to mine", "round", round, "error", err)
	}

	return isMyTurn, err
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (x *XDPoS_v2) Prepare(chain consensus.ChainReader, header *types.Header) error {

	x.lock.RLock()
	currentRound := x.currentRound
	highestQC := x.highestQuorumCert
	x.lock.RUnlock()

	if header.ParentHash != highestQC.ProposedBlockInfo.Hash {
		log.Warn("[Prepare] parent hash and QC hash does not match", "blockNum", header.Number, "parentHash", header.ParentHash, "QCHash", highestQC.ProposedBlockInfo.Hash, "QCNumber", highestQC.ProposedBlockInfo.Number)
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

	x.signLock.RLock()
	signer := x.signer
	x.signLock.RUnlock()

	isMyTurn, err := x.yourturn(chain, currentRound, parent, signer)
	if err != nil {
		log.Error("[Prepare] Error while checking if it's still my turn to mine", "currentRound", currentRound, "ParentHash", parent.Hash().Hex(), "ParentNumber", parent.Number.Uint64(), "error", err)
		return err
	}
	if !isMyTurn {
		return consensus.ErrNotReadyToMine
	}
	// Set the correct difficulty
	header.Difficulty = x.calcDifficulty(chain, parent, signer)
	log.Debug("CalcDifficulty ", "number", header.Number, "difficulty", header.Difficulty)

	isEpochSwitchBlock, _, err := x.IsEpochSwitch(header)
	if err != nil {
		log.Error("[Prepare] Error while trying to determine if header is an epoch switch during Prepare", "header", header, "Error", err)
		return err
	}
	if isEpochSwitchBlock {
		masterNodes, penalties, err := x.calcMasternodes(chain, header.Number, header.ParentHash)
		if err != nil {
			return err
		}
		for _, v := range masterNodes {
			header.Validators = append(header.Validators, v[:]...)
		}
		for _, v := range penalties {
			header.Penalties = append(header.Penalties, v[:]...)
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

	if header.Coinbase != signer {
		log.Error("[Prepare] The mined blocker header coinbase address mismatch with waller address", "headerCoinbase", header.Coinbase.Hex(), "WalletAddress", signer.Hex())
		return consensus.ErrCoinbaseMismatch
	}

	return nil
}

// Finalize implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given, and returns the final block.
func (x *XDPoS_v2) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, parentState *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	// set block reward

	isEpochSwitch, _, err := x.IsEpochSwitch(header)
	if err != nil {
		log.Error("[Finalize] IsEpochSwitch bug!", "err", err)
		return nil, err
	}
	if x.HookReward != nil && isEpochSwitch {
		rewards, err := x.HookReward(chain, state, parentState, header)
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

	// Don't hold the signer fields for the entire sealing procedure
	x.signLock.RLock()
	signer, signFn := x.signer, x.signFn
	x.signLock.RUnlock()

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

	// Mark the highestSelfMinedRound to make sure we only mine once per round
	var decodedExtraField utils.ExtraFields_v2
	err = utils.DecodeBytesExtraFields(header.Extra, &decodedExtraField)
	if err != nil {
		log.Error("[Seal] Error when decode extra field to get the round number from v2 block during sealing", "Hash", header.Hash().Hex(), "Number", header.Number.Uint64(), "Error", err)
		return nil, err
	}
	x.highestSelfMinedRound = decodedExtraField.Round

	return block.WithSeal(header), nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have based on the previous blocks in the chain and the
// current signer.
func (x *XDPoS_v2) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	return x.calcDifficulty(chain, parent, x.signer)
}

func (x *XDPoS_v2) IsAuthorisedAddress(chain consensus.ChainReader, header *types.Header, address common.Address) bool {
	x.lock.RLock()
	defer x.lock.RUnlock()

	_, round, _, err := x.getExtraFields(header)
	if err != nil {
		log.Error("[IsAuthorisedAddress] Fail to decode v2 extra data", "Hash", header.Hash().Hex(), "Extra", header.Extra, "Error", err)
		return false
	}
	blockRound := round

	masterNodes := x.GetMasternodes(chain, header)

	if len(masterNodes) == 0 {
		log.Error("[IsAuthorisedAddress] Fail to find any master nodes from current block round epoch", "Hash", header.Hash().Hex(), "Round", blockRound, "Number", header.Number)
		return false
	}

	for index, masterNodeAddress := range masterNodes {
		if masterNodeAddress == address {
			log.Debug("[IsAuthorisedAddress] Found matching master node address", "index", index, "Address", address, "MasterNodes", masterNodes)
			return true
		}
	}

	log.Warn("Not authorised address", "Address", address.Hex(), "Hash", header.Hash().Hex())
	for index, mn := range masterNodes {
		log.Warn("Master node list item", "mn", mn.Hex(), "index", index)
	}

	return false
}

func (x *XDPoS_v2) GetSnapshot(chain consensus.ChainReader, header *types.Header) (*SnapshotV2, error) {
	number := header.Number.Uint64()
	log.Trace("get snapshot", "number", number)
	snap, err := x.getSnapshot(chain, number, false)
	if err != nil {
		return nil, err
	}
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
	snap := newSnapshot(number, header.Hash(), masterNodes)
	x.lock.RUnlock()

	err := storeSnapshot(snap, x.db)
	if err != nil {
		log.Error("[UpdateMasternodes] Error while store snashot", "hash", header.Hash(), "currentRound", x.currentRound, "error", err)
		return err
	}
	x.snapshots.Add(snap.Hash, snap)

	nm := []string{}
	for _, n := range ms {
		nm = append(nm, n.Address.String())
	}
	log.Info("New set of masternodes has been updated to snapshot", "number", snap.Number, "hash", snap.Hash, "new masternodes", nm)

	return nil
}

func (x *XDPoS_v2) VerifyHeader(chain consensus.ChainReader, header *types.Header, fullVerify bool) error {
	err := x.verifyHeader(chain, header, nil, fullVerify)
	if err != nil {
		log.Warn("[VerifyHeader] Fail to verify header", "fullVerify", fullVerify, "blockNum", header.Number, "blockHash", header.Hash(), "error", err)
	}
	return err
}

// Verify a list of headers
func (x *XDPoS_v2) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, fullVerifies []bool, abort <-chan struct{}, results chan<- error) {
	go func() {
		for i, header := range headers {
			err := x.verifyHeader(chain, header, headers[:i], fullVerifies[i])
			log.Warn("[VerifyHeaders] Fail to verify header", "fullVerify", fullVerifies[i], "blockNum", header.Number, "blockHash", header.Hash(), "error", err)
			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
}

/*
	SyncInfo workflow
*/
// Verify syncInfo and trigger process QC or TC if successful
func (x *XDPoS_v2) VerifySyncInfoMessage(chain consensus.ChainReader, syncInfo *utils.SyncInfo) (bool, error) {
	/*
		1. Check QC and TC against highest QC TC. Skip if none of them need to be updated
		2. Verify items including:
				- verifyQC
				- verifyTC
		3. Broadcast(Not part of consensus)
	*/

	if (x.highestQuorumCert.ProposedBlockInfo.Round >= syncInfo.HighestQuorumCert.ProposedBlockInfo.Round) && (x.highestTimeoutCert.Round >= syncInfo.HighestTimeoutCert.Round) {
		log.Warn("[VerifySyncInfoMessage] Round from incoming syncInfo message is no longer qualified", "Highest QC Round", x.highestQuorumCert.ProposedBlockInfo.Round, "Incoming SyncInfo QC Round", syncInfo.HighestQuorumCert.ProposedBlockInfo.Round, "highestTimeoutCert Round", x.highestTimeoutCert.Round, "Incoming syncInfo TC Round", syncInfo.HighestTimeoutCert.Round)
		return false, nil
	}

	err := x.verifyQC(chain, syncInfo.HighestQuorumCert, nil)
	if err != nil {
		log.Warn("SyncInfo message verification failed due to QC", "error", err)
		return false, err
	}
	err = x.verifyTC(chain, syncInfo.HighestTimeoutCert)
	if err != nil {
		log.Warn("SyncInfo message verification failed due to TC", "error", err)
		return false, err
	}
	return true, nil
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
	return x.processTC(chain, syncInfo.HighestTimeoutCert)
}

/*
	Vote workflow
*/
func (x *XDPoS_v2) VerifyVoteMessage(chain consensus.ChainReader, vote *utils.Vote) (bool, error) {
	/*
		  1. Check vote round with current round for fast fail(disqualifed)
		  2. Get masterNode list from snapshot by using vote.GapNumber
		  3. Check signature:
					- Use ecRecover to get the public key
					- Use the above public key to find out the xdc address
					- Use the above xdc address to check against the master node list from step 1(For the running epoch)
			4. Broadcast(Not part of consensus)
	*/
	if vote.ProposedBlockInfo.Round < x.currentRound {
		log.Warn("[VerifyVoteMessage] Disqualified vote message as the proposed round does not match currentRound", "vote.ProposedBlockInfo.Round", vote.ProposedBlockInfo.Round, "currentRound", x.currentRound)
		return false, nil
	}

	snapshot, err := x.getSnapshot(chain, vote.GapNumber, true)
	if err != nil {
		log.Error("[VerifyVoteMessage] fail to get snapshot for a vote message", "BlockNum", vote.ProposedBlockInfo.Number, "Hash", vote.ProposedBlockInfo.Hash, "Error", err.Error())
	}
	verified, _, err := x.verifyMsgSignature(utils.VoteSigHash(&utils.VoteForSign{
		ProposedBlockInfo: vote.ProposedBlockInfo,
		GapNumber:         vote.GapNumber,
	}), vote.Signature, snapshot.NextEpochMasterNodes)
	if err != nil {
		for i, mn := range snapshot.NextEpochMasterNodes {
			log.Warn("[VerifyVoteMessage] Master node list item", "index", i, "Master node", mn.Hex())
		}
		log.Warn("[VerifyVoteMessage] Error while verifying vote message", "votedBlockNum", vote.ProposedBlockInfo.Number.Uint64(), "votedBlockHash", vote.ProposedBlockInfo.Hash.Hex(), "Error", err.Error())
	}
	return verified, err
}

// Consensus entry point for processing vote message to produce QC
func (x *XDPoS_v2) VoteHandler(chain consensus.ChainReader, voteMsg *utils.Vote) error {
	x.lock.Lock()
	defer x.lock.Unlock()
	return x.voteHandler(chain, voteMsg)
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
	snap, err := x.getSnapshot(chain, timeoutMsg.GapNumber, true)
	if err != nil {
		log.Error("[VerifyTimeoutMessage] Fail to get snapshot when verifying timeout message!", "messageGapNumber", timeoutMsg.GapNumber)
	}
	if snap == nil || len(snap.NextEpochMasterNodes) == 0 {
		log.Error("[VerifyTimeoutMessage] Something wrong with the snapshot from gapNumber", "messageGapNumber", timeoutMsg.GapNumber, "snapshot", snap)
		return false, fmt.Errorf("Empty master node lists from snapshot")
	}

	verified, _, err := x.verifyMsgSignature(utils.TimeoutSigHash(&utils.TimeoutForSign{
		Round:     timeoutMsg.Round,
		GapNumber: timeoutMsg.GapNumber,
	}), timeoutMsg.Signature, snap.NextEpochMasterNodes)
	return verified, err
}

/*
	Entry point for handling timeout message to process below:
	1. checkRoundNumber()
	2. Collect timeout
	3. Once timeout pool reached threshold, it will trigger the call to the function "onTimeoutPoolThresholdReached"
*/
func (x *XDPoS_v2) TimeoutHandler(blockChainReader consensus.ChainReader, timeout *utils.Timeout) error {
	x.lock.Lock()
	defer x.lock.Unlock()
	return x.timeoutHandler(blockChainReader, timeout)
}

/*
	Proposed Block workflow
*/
func (x *XDPoS_v2) ProposedBlockHandler(chain consensus.ChainReader, blockHeader *types.Header) error {
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
	quorumCert, round, _, err := x.getExtraFields(blockHeader)
	if err != nil {
		return err
	}

	err = x.verifyQC(chain, quorumCert, nil)
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
	err = x.processQC(chain, quorumCert)
	if err != nil {
		log.Error("[ProposedBlockHandler] Fail to processQC", "QC proposed blockInfo round number", quorumCert.ProposedBlockInfo.Round, "QC proposed blockInfo hash", quorumCert.ProposedBlockInfo.Hash)
		return err
	}

	err = x.allowedToSend(chain, blockHeader, "vote")
	if err != nil {
		return err
	}

	verified, err := x.verifyVotingRule(chain, blockInfo, quorumCert)
	if err != nil {
		return err
	}
	if verified {
		return x.sendVote(chain, blockInfo)
	} else {
		log.Info("Failed to pass the voting rule verification", "ProposeBlockHash", blockInfo.Hash)
	}

	return nil
}

/*
	QC & TC Utils
*/

// To be used by different message verification. Verify local DB block info against the received block information(i.e hash, blockNum, round)
func (x *XDPoS_v2) VerifyBlockInfo(blockChainReader consensus.ChainReader, blockInfo *utils.BlockInfo, blockHeader *types.Header) error {
	/*
		1. Check if is able to get header by hash from the chain
		2. Check the header from step 1 matches what's in the blockInfo. This includes the block number and the round
	*/
	if blockHeader == nil {
		blockHeader = blockChainReader.GetHeaderByHash(blockInfo.Hash)
		if blockHeader == nil {
			log.Warn("[VerifyBlockInfo] No such header in the chain", "BlockInfoHash", blockInfo.Hash.Hex(), "BlockInfoNum", blockInfo.Number, "BlockInfoRound", blockInfo.Round, "currentHeaderNum", blockChainReader.CurrentHeader().Number)
			return fmt.Errorf("[VerifyBlockInfo] header doesn't exist for the received blockInfo at hash: %v", blockInfo.Hash.Hex())
		}
	} else {
		// If blockHeader present, then its value shall consistent with what's provided in the blockInfo
		if blockHeader.Hash() != blockInfo.Hash {
			log.Warn("[VerifyBlockInfo] BlockHeader and blockInfo mismatch", "BlockInfoHash", blockInfo.Hash.Hex(), "BlockHeaderHash", blockHeader.Hash())
			return fmt.Errorf("[VerifyBlockInfo] Provided blockheader does not match what's in the blockInfo")
		}
	}

	if blockHeader.Number.Cmp(blockInfo.Number) != 0 {
		log.Warn("[VerifyBlockInfo] Block Number mismatch", "BlockInfoHash", blockInfo.Hash.Hex(), "BlockInfoNum", blockInfo.Number, "BlockInfoRound", blockInfo.Round, "blockHeaderNum", blockHeader.Number)
		return fmt.Errorf("[VerifyBlockInfo] chain header number does not match for the received blockInfo at hash: %v", blockInfo.Hash.Hex())
	}

	// Switch block is a v1 block, there is no valid extra to decode, nor its round
	if blockInfo.Number.Cmp(x.config.V2.SwitchBlock) == 0 {
		if blockInfo.Round != 0 {
			log.Error("[VerifyBlockInfo] Switch block round is not 0", "BlockInfoHash", blockInfo.Hash.Hex(), "BlockInfoNum", blockInfo.Number, "BlockInfoRound", blockInfo.Round, "blockHeaderNum", blockHeader.Number)
			return fmt.Errorf("[VerifyBlockInfo] switch block round have to be 0")
		}
		return nil
	}
	// Check round

	_, round, _, err := x.getExtraFields(blockHeader)
	if err != nil {
		log.Error("[VerifyBlockInfo] Fail to decode extra field", "BlockInfoHash", blockInfo.Hash.Hex(), "BlockInfoNum", blockInfo.Number, "BlockInfoRound", blockInfo.Round, "blockHeaderNum", blockHeader.Number)
		return err
	}
	if round != blockInfo.Round {
		log.Warn("[VerifyBlockInfo] Block extra round mismatch with blockInfo", "BlockInfoHash", blockInfo.Hash.Hex(), "BlockInfoNum", blockInfo.Number, "BlockInfoRound", blockInfo.Round, "blockHeaderNum", blockHeader.Number, "blockRound", round)
		return fmt.Errorf("[VerifyBlockInfo] chain block's round does not match from blockInfo at hash: %v and block round: %v, blockInfo Round: %v", blockInfo.Hash.Hex(), round, blockInfo.Round)
	}

	return nil
}

func (x *XDPoS_v2) verifyQC(blockChainReader consensus.ChainReader, quorumCert *utils.QuorumCert, parentHeader *types.Header) error {
	/*
		1. Check if num of QC signatures is >= x.config.v2.CertThreshold
		2. Get epoch master node list by hash
		3. Verify signer signatures: (List of signatures)
					- Use ecRecover to get the public key
					- Use the above public key to find out the xdc address
					- Use the above xdc address to check against the master node list from step 1(For the received QC epoch)
		4. Verify gapNumber = epochSwitchNumber - epochSwitchNumber%Epoch - Gap
		5. Verify blockInfo
	*/
	epochInfo, err := x.getEpochSwitchInfo(blockChainReader, parentHeader, quorumCert.ProposedBlockInfo.Hash)
	if err != nil {
		log.Error("[verifyQC] Error when getting epoch switch Info to verify QC", "Error", err)
		return fmt.Errorf("Fail to verify QC due to failure in getting epoch switch info")
	}

	signatures, duplicates := UniqueSignatures(quorumCert.Signatures)
	if len(duplicates) != 0 {
		for _, d := range duplicates {
			log.Warn("[verifyQC] duplicated signature in QC", "duplicate", common.Bytes2Hex(d))
		}
	}
	if quorumCert == nil {
		log.Warn("[verifyQC] QC is Nil")
		return utils.ErrInvalidQC
	} else if (quorumCert.ProposedBlockInfo.Number.Uint64() > x.config.V2.SwitchBlock.Uint64()) && (signatures == nil || (len(signatures) < x.config.V2.CertThreshold)) {
		//First V2 Block QC, QC Signatures is initial nil
		log.Warn("[verifyHeader] Invalid QC Signature is nil or empty", "QC", quorumCert, "QCNumber", quorumCert.ProposedBlockInfo.Number, "Signatures len", len(signatures))
		return utils.ErrInvalidQC
	}

	var wg sync.WaitGroup
	wg.Add(len(signatures))
	var haveError error

	for _, signature := range signatures {
		go func(sig utils.Signature) {
			defer wg.Done()
			verified, _, err := x.verifyMsgSignature(utils.VoteSigHash(&utils.VoteForSign{
				ProposedBlockInfo: quorumCert.ProposedBlockInfo,
				GapNumber:         quorumCert.GapNumber,
			}), sig, epochInfo.Masternodes)
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
	epochSwitchNumber := epochInfo.EpochSwitchBlockInfo.Number.Uint64()
	gapNumber := epochSwitchNumber - epochSwitchNumber%x.config.Epoch - x.config.Gap
	if gapNumber != quorumCert.GapNumber {
		log.Error("[verifyQC] gap number mismatch", "BlockInfoHash", quorumCert.ProposedBlockInfo.Hash, "Gap", quorumCert.GapNumber, "GapShouldBe", gapNumber)
		return fmt.Errorf("gap number mismatch %v", quorumCert)
	}

	return x.VerifyBlockInfo(blockChainReader, quorumCert.ProposedBlockInfo, parentHeader)
}

// Update local QC variables including highestQC & lockQuorumCert, as well as commit the blocks that satisfy the algorithm requirements
func (x *XDPoS_v2) processQC(blockChainReader consensus.ChainReader, incomingQuorumCert *utils.QuorumCert) error {
	log.Trace("[ProcessQC][Before]", "HighQC", x.highestQuorumCert)
	// 1. Update HighestQC
	if incomingQuorumCert.ProposedBlockInfo.Round > x.highestQuorumCert.ProposedBlockInfo.Round {
		x.highestQuorumCert = incomingQuorumCert
	}
	// 2. Get QC from header and update lockQuorumCert(lockQuorumCert is the parent of highestQC)
	proposedBlockHeader := blockChainReader.GetHeaderByHash(incomingQuorumCert.ProposedBlockInfo.Hash)
	if proposedBlockHeader == nil {
		log.Error("[processQC] Block not found using the QC", "quorumCert.ProposedBlockInfo.Hash", incomingQuorumCert.ProposedBlockInfo.Hash, "incomingQuorumCert.ProposedBlockInfo.Number", incomingQuorumCert.ProposedBlockInfo.Number)
		return fmt.Errorf("Block not found, number: %v, hash: %v", incomingQuorumCert.ProposedBlockInfo.Number, incomingQuorumCert.ProposedBlockInfo.Hash)
	}
	if proposedBlockHeader.Number.Cmp(x.config.V2.SwitchBlock) > 0 {
		// Extra field contain parent information
		proposedBlockQuorumCert, round, _, err := x.getExtraFields(proposedBlockHeader)
		if err != nil {
			return err
		}
		if x.lockQuorumCert == nil || proposedBlockQuorumCert.ProposedBlockInfo.Round > x.lockQuorumCert.ProposedBlockInfo.Round {
			x.lockQuorumCert = proposedBlockQuorumCert
		}

		proposedBlockRound := &round
		// 3. Update commit block info
		_, err = x.commitBlocks(blockChainReader, proposedBlockHeader, proposedBlockRound, incomingQuorumCert)
		if err != nil {
			log.Error("[processQC] Error while to commitBlocks", "proposedBlockRound", proposedBlockRound)
			return err
		}
	}
	// 4. Set new round
	if incomingQuorumCert.ProposedBlockInfo.Round >= x.currentRound {
		err := x.setNewRound(blockChainReader, incomingQuorumCert.ProposedBlockInfo.Round+1)
		if err != nil {
			log.Error("[processQC] Fail to setNewRound", "new round to set", incomingQuorumCert.ProposedBlockInfo.Round+1)
			return err
		}
	}
	log.Trace("[ProcessQC][After]", "HighQC", x.highestQuorumCert)
	return nil
}

/*
	1. Set currentRound = QC round + 1 (or TC round +1)
	2. Reset timer
	3. Reset vote and timeout Pools
*/
func (x *XDPoS_v2) setNewRound(blockChainReader consensus.ChainReader, round utils.Round) error {
	x.currentRound = round
	x.timeoutCount = 0
	//TODO: tell miner now it's a new round and start mine if it's leader
	x.timeoutWorker.Reset(blockChainReader)
	//TODO: vote pools
	x.timeoutPool.Clear()
	return nil
}

func (x *XDPoS_v2) broadcastToBftChannel(msg interface{}) {
	go func() {
		x.BroadcastCh <- msg
	}()
}

func (x *XDPoS_v2) getSyncInfo() *utils.SyncInfo {
	return &utils.SyncInfo{
		HighestQuorumCert:  x.highestQuorumCert,
		HighestTimeoutCert: x.highestTimeoutCert,
	}
}

//Find parent and grandparent, check round number, if so, commit grandparent(grandGrandParent of currentBlock)
func (x *XDPoS_v2) commitBlocks(blockChainReader consensus.ChainReader, proposedBlockHeader *types.Header, proposedBlockRound *utils.Round, incomingQc *utils.QuorumCert) (bool, error) {
	// XDPoS v1.0 switch to v2.0, skip commit
	if big.NewInt(0).Sub(proposedBlockHeader.Number, big.NewInt(2)).Cmp(x.config.V2.SwitchBlock) <= 0 {
		return false, nil
	}
	// Find the last two parent block and check their rounds are the continuous
	parentBlock := blockChainReader.GetHeaderByHash(proposedBlockHeader.ParentHash)

	_, round, _, err := x.getExtraFields(parentBlock)
	if err != nil {
		log.Error("Fail to execute first DecodeBytesExtraFields for commiting block", "ProposedBlockHash", proposedBlockHeader.Hash())
		return false, err
	}
	if *proposedBlockRound-1 != round {
		log.Debug("[commitBlocks] Rounds not continuous(parent) found when committing block", "proposedBlockRound", proposedBlockRound, "decodedExtraField.Round", round, "proposedBlockHeaderHash", proposedBlockHeader.Hash())
		return false, nil
	}

	// If parent round is continuous, we check grandparent
	grandParentBlock := blockChainReader.GetHeaderByHash(parentBlock.ParentHash)
	_, round, _, err = x.getExtraFields(grandParentBlock)
	if err != nil {
		log.Error("Fail to execute second DecodeBytesExtraFields for commiting block", "parentBlockHash", parentBlock.Hash())
		return false, err
	}
	if *proposedBlockRound-2 != round {
		log.Debug("[commitBlocks] Rounds not continuous(grand parent) found when committing block", "proposedBlockRound", proposedBlockRound, "decodedExtraField.Round", round, "proposedBlockHeaderHash", proposedBlockHeader.Hash())
		return false, nil
	}
	// Commit the grandParent block
	if x.highestCommitBlock == nil || (x.highestCommitBlock.Round < round && x.highestCommitBlock.Number.Cmp(grandParentBlock.Number) == -1) {
		x.highestCommitBlock = &utils.BlockInfo{
			Number: grandParentBlock.Number,
			Hash:   grandParentBlock.Hash(),
			Round:  round,
		}
		log.Debug("Successfully committed block", "Committed block Hash", x.highestCommitBlock.Hash, "Committed round", x.highestCommitBlock.Round)
		// Perform forensics related operation
		var headerQcToBeCommitted []types.Header
		headerQcToBeCommitted = append(headerQcToBeCommitted, *parentBlock, *proposedBlockHeader)
		go x.forensics.SetCommittedQCs(headerQcToBeCommitted, *incomingQc)
		return true, nil
	}
	// Everything else, fail to commit
	return false, nil
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
	if header.Number.Cmp(x.config.V2.SwitchBlock) == 0 {
		log.Info("[IsEpochSwitch] examing last v1 block")
		return true, header.Number.Uint64() / x.config.Epoch, nil
	}

	quorumCert, round, _, err := x.getExtraFields(header)
	if err != nil {
		log.Error("[IsEpochSwitch] decode header error", "err", err, "header", header, "extra", common.Bytes2Hex(header.Extra))
		return false, 0, err
	}
	parentRound := quorumCert.ProposedBlockInfo.Round
	epochStartRound := round - round%utils.Round(x.config.Epoch)
	epochNum := x.config.V2.SwitchBlock.Uint64()/x.config.Epoch + uint64(round)/x.config.Epoch
	// if parent is last v1 block and this is first v2 block, this is treated as epoch switch
	if quorumCert.ProposedBlockInfo.Number.Cmp(x.config.V2.SwitchBlock) == 0 {
		log.Info("[IsEpochSwitch] true, parent equals V2.SwitchBlock", "round", round, "number", header.Number.Uint64(), "hash", header.Hash())
		return true, epochNum, nil
	}
	log.Info("[IsEpochSwitch]", "parent round", parentRound, "round", round, "number", header.Number.Uint64(), "hash", header.Hash())
	return parentRound < epochStartRound, epochNum, nil
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
	epochNum := x.config.V2.SwitchBlock.Uint64()/x.config.Epoch + uint64(epochSwitchInfo.EpochSwitchBlockInfo.Round)/x.config.Epoch
	return currentCheckpointNumber, epochNum, nil
}

func (x *XDPoS_v2) calcMasternodes(chain consensus.ChainReader, blockNum *big.Int, parentHash common.Hash) ([]common.Address, []common.Address, error) {
	snap, err := x.getSnapshot(chain, blockNum.Uint64(), false)
	if err != nil {
		log.Error("[calcMasternodes] Adaptor v2 getSnapshot has error", "err", err)
		return nil, nil, err
	}
	candidates := snap.NextEpochMasterNodes

	if blockNum.Uint64() == x.config.V2.SwitchBlock.Uint64()+1 {
		log.Info("[calcMasternodes] examing first v2 block")
		return candidates, []common.Address{}, nil
	}

	if x.HookPenalty == nil {
		log.Info("[calcMasternodes] no hook penalty defined")
		return candidates, []common.Address{}, nil
	}

	penalties, err := x.HookPenalty(chain, blockNum, parentHash, candidates)
	if err != nil {
		log.Error("[calcMasternodes] Adaptor v2 HookPenalty has error", "err", err)
		return nil, nil, err
	}
	masternodes := common.RemoveItemFromArray(candidates, penalties)
	return masternodes, penalties, nil

}

// Given hash, get master node from the epoch switch block of the epoch
func (x *XDPoS_v2) GetMasternodesByHash(chain consensus.ChainReader, hash common.Hash) []common.Address {
	epochSwitchInfo, err := x.getEpochSwitchInfo(chain, nil, hash)
	if err != nil {
		log.Error("[GetMasternodes] Adaptor v2 getEpochSwitchInfo has error, potentially bug", "err", err)
		return []common.Address{}
	}
	return epochSwitchInfo.Masternodes
}

// Given hash, get master node from the epoch switch block of the previous `limit` epoch
func (x *XDPoS_v2) GetPreviousPenaltyByHash(chain consensus.ChainReader, hash common.Hash, limit int) []common.Address {
	epochSwitchInfo, err := x.getPreviousEpochSwitchInfoByHash(chain, hash, limit)
	if err != nil {
		log.Error("[GetPreviousPenaltyByHash] Adaptor v2 getPreviousEpochSwitchInfoByHash has error, potentially bug", "err", err)
		return []common.Address{}
	}
	header := chain.GetHeaderByHash(epochSwitchInfo.EpochSwitchBlockInfo.Hash)
	return common.ExtractAddressFromBytes(header.Penalties)
}

func (x *XDPoS_v2) FindParentBlockToAssign(chain consensus.ChainReader) *types.Block {
	parent := chain.GetBlock(x.highestQuorumCert.ProposedBlockInfo.Hash, x.highestQuorumCert.ProposedBlockInfo.Number.Uint64())
	if parent == nil {
		log.Error("[FindParentBlockToAssign] Can not find parent block from highestQC proposedBlockInfo", "x.highestQuorumCert.ProposedBlockInfo.Hash", x.highestQuorumCert.ProposedBlockInfo.Hash, "x.highestQuorumCert.ProposedBlockInfo.Number", x.highestQuorumCert.ProposedBlockInfo.Number.Uint64())
	}
	return parent
}

func (x *XDPoS_v2) allowedToSend(chain consensus.ChainReader, blockHeader *types.Header, sendType string) error {
	allowedToSend := false
	// Don't hold the signFn for the whole signing operation
	x.signLock.RLock()
	signer := x.signer
	x.signLock.RUnlock()
	// Check if the node can send this sendType
	masterNodes := x.GetMasternodes(chain, blockHeader)
	for i, mn := range masterNodes {
		if signer == mn {
			log.Debug("[allowedToSend] Yes, I'm allowed to send", "sendType", sendType, "MyAddress", signer.Hex(), "Index in master node list", i)
			allowedToSend = true
			break
		}
	}
	if !allowedToSend {
		for _, mn := range masterNodes {
			log.Debug("[allowedToSend] Master node list", "masterNodeAddress", mn.Hash())
		}
		log.Warn("[allowedToSend] Not in the Masternode list, not suppose to send", "sendType", sendType, "MyAddress", signer.Hex())
		return fmt.Errorf("Not in the master node list, not suppose to %v", sendType)
	}
	return nil
}

// Periodlly execution(Attached to engine initialisation during "new"). Used for pool cleaning etc
func (x *XDPoS_v2) periodicJob() {
	go func() {
		for {
			<-time.After(utils.PeriodicJobPeriod * time.Second)
			x.hygieneVotePool()
			x.hygieneTimeoutPool()
		}
	}()
}
