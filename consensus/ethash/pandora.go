package ethash

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	common2 "github.com/silesiacoin/bls/common"
	"github.com/silesiacoin/bls/herumi"
)

const (
	// Time expressed in seconds
	SlotTimeDuration = 6
	validatorListLen = 32
	signatureSize    = 96
)

// Use decorator pattern to get there as fast as possible
// This is a prototype, it can be designed way better
type Pandora struct {
	sealer *remoteSealer
}

type BlsSignatureBytes [signatureSize]byte

type PandoraExtraData struct {
	Slot  uint64
	Epoch uint64
	Turn  uint64
}

type MinimalEpochConsensusInfoPayload struct {
	// Epoch number
	Epoch uint64 `json:"epoch"`
	// Validators public key list for specific epoch
	ValidatorList    [32]string    `json:"validatorList"`
	EpochTimeStart   uint64        `json:"epochTimeStart"`
	SlotTimeDuration time.Duration `json:"slotTimeDuration"`
}

type PandoraExtraDataSealed struct {
	PandoraExtraData
	BlsSignatureBytes *BlsSignatureBytes
}

// This should be cached or retrieved in a handshake with vanguard
type MinimalEpochConsensusInfo struct {
	// Epoch number
	Epoch uint64 `json:"epoch"`
	// Validators list 32 public bls keys. slot(n) in Epoch is represented by index(n) in MinimalConsensusInfo
	ValidatorsList [validatorListLen]common2.PublicKey `json:"validatorList"`
	// Unix timestamp of consensus start. This will be used to extract time slot
	EpochTimeStart time.Time

	EpochTimeStartUnix uint64 `json:"epochTimeStart"`

	// Slot time duration
	SlotTimeDuration time.Duration `json:"SlotTimeDuration"`
}

func NewPandora(
	config Config,
	notify []string,
	noverify bool,
	minimalConsensusInfo interface{},
	orcSubscribe bool,
) *Ethash {
	config.PowMode = ModePandora
	ethash := New(config, notify, noverify)
	ethash.mci = newlru("epochSet", int(math.Pow(2, 7)), NewMinimalConsensusInfo)

	consensusInfo := minimalConsensusInfo.([]*params.MinimalEpochConsensusInfo)
	genesisConsensusTimeStart := consensusInfo[0]
	mci := ethash.mci
	mciCache := mci.cache

	// Fill cache with minimal consensus info
	for index, currentConsensusInfo := range consensusInfo {
		convertedInfo := NewMinimalConsensusInfo(currentConsensusInfo.Epoch)
		pandoraConsensusInfo := convertedInfo.(*MinimalEpochConsensusInfo)
		pandoraConsensusInfo.AssignEpochStartFromGenesis(time.Unix(
			int64(genesisConsensusTimeStart.EpochTimeStart),
			0,
		))
		pandoraConsensusInfo.AssignValidators(currentConsensusInfo.ValidatorList)
		mciCache.Add(index, pandoraConsensusInfo)
	}

	ethash.remote = StartRemotePandora(ethash, notify, noverify, orcSubscribe)

	return ethash
}

// This is done only to have vanguard spec done in minimal codebase to exchange information with pandora.
// In this approach you could have multiple execution engines connected via urls []string
// In this approach you are also compatible with any current toolsets for mining because you use already defined api
func StartRemotePandora(
	executionEngine *Ethash,
	urls []string,
	noverify bool,
	orcSubscribe bool,
) (sealer *remoteSealer) {
	ctx, cancel := context.WithCancel(context.Background())
	sealer = &remoteSealer{
		ethash:       executionEngine,
		noverify:     noverify,
		notifyURLs:   urls,
		notifyCtx:    ctx,
		cancelNotify: cancel,
		works:        make(map[common.Hash]*types.Block),
		rates:        make(map[common.Hash]hashrate),
		workCh:       make(chan *sealTask),
		fetchWorkCh:  make(chan *sealWork),
		submitWorkCh: make(chan *mineResult),
		fetchRateCh:  make(chan chan uint64),
		submitRateCh: make(chan *hashrate),
		requestExit:  make(chan struct{}),
		exitCh:       make(chan struct{}),
	}

	pandora := Pandora{sealer: sealer}
	go pandora.Loop()
	go pandora.HandleOrchestratorSubscriptions(orcSubscribe, ctx, 0)

	return
}

func (pandora *Pandora) HandleOrchestratorSubscriptions(orcSubscribe bool, ctx context.Context, retry int) {
	sealer := pandora.sealer
	ethashEngine := sealer.ethash
	config := ethashEngine.config
	logger := config.Log

	// Early return, do not subscribe to orchestrator client
	if !orcSubscribe {
		return
	}

	retryTimeout := time.Second * 5
	maxRetries := 2 ^ 32
	retry++

	if retry > maxRetries {
		logger.Crit("Orchestrator is offline for too long. Please check your connection")

		return
	}

	lastKnownEpoch := uint64(0)
	subscription, channel, err, errChan := pandora.SubscribeToMinimalConsensusInformation(lastKnownEpoch, ctx)

	// This will create loop
	if nil != err {
		logger.Error("could not start remote pandora",
			"err", err.Error(),
			"addr", pandora.sealer.notifyURLs[0],
		)
		time.Sleep(retryTimeout)
		pandora.HandleOrchestratorSubscriptions(orcSubscribe, ctx, retry)

		return
	}

	defer func() {
		logger.Trace("Pandora is closing Orchestrator subscriptions")
		subscription.Unsubscribe()
	}()

	ticker := time.NewTimer(time.Second * 5)

	insertFunc := func(
		minimalConsensus *MinimalEpochConsensusInfoPayload,
	) (currentErr error) {
		logger.Info(
			"Received minimalConsensusInfo",
			"epoch", minimalConsensus.Epoch,
			"epochTimeStart", minimalConsensus.EpochTimeStart,
			"validatorListLen", len(minimalConsensus.ValidatorList),
		)
		coreMinimalConsensus := NewMinimalConsensusInfo(minimalConsensus.Epoch).(*MinimalEpochConsensusInfo)
		coreMinimalConsensus.EpochTimeStart = time.Unix(int64(minimalConsensus.EpochTimeStart), 0)
		coreMinimalConsensus.EpochTimeStartUnix = minimalConsensus.EpochTimeStart
		coreMinimalConsensus.ValidatorsList = [32]common2.PublicKey{}
		lastKnownEpoch = minimalConsensus.Epoch

		for index, validator := range minimalConsensus.ValidatorList {
			// Create dummy key for genesis epoch slot 0
			// This fallback is done because orchestrators sending only 0x instead of full public key
			// for slot 0
			if 0 == index && 0 == coreMinimalConsensus.Epoch {
				secretKey, _ := herumi.RandKey()
				pubKey := secretKey.PublicKey()
				validator = hexutil.Encode(pubKey.Marshal())
			}

			publicKeyBytes, currentErr := hexutil.Decode(validator)

			if nil != currentErr {
				errChan <- currentErr

				break
			}

			coreMinimalConsensus.ValidatorsList[index], currentErr = herumi.PublicKeyFromBytes(publicKeyBytes)

			if nil != currentErr {
				errChan <- currentErr

				break
			}
		}

		currentErr = ethashEngine.InsertMinimalConsensusInfo(minimalConsensus.Epoch, coreMinimalConsensus)

		return
	}

	for {
		select {
		case <-ticker.C:
			epochFromCache, exists := ethashEngine.mci.cache.Get(int(lastKnownEpoch))
			logger.Info(
				"awaiting for orchestrator information",
				"epoch", lastKnownEpoch,
				"exists", exists,
				"epochFromCache", epochFromCache,
			)
		case payload := <-channel:
			currentErr := insertFunc(payload)

			if nil != currentErr {
				errChan <- currentErr
				logger.Error(
					"error during payload consumption",
					"err",
					currentErr.Error(),
				)
				return
			}
		case err = <-subscription.Err():
			if nil != err {
				errChan <- err
				logger.Error(
					"error during pandora subscription pipe",
					"err",
					err.Error(),
				)

				return
			}
		case err = <-errChan:
			logger.Error(
				"error during pandora subscription",
				"err",
				err.Error(),
			)
		}
	}
}

func (pandora *Pandora) Loop() {
	s := pandora.sealer

	defer func() {
		s.ethash.config.Log.Trace("Pandora remote sealer is exiting")
		s.cancelNotify()
		s.reqWG.Wait()
		close(s.exitCh)
	}()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case work := <-s.workCh:
			// Update current work with new received block.
			// Note same work can be past twice, happens when changing CPU threads.
			s.results = work.results
			pandora.makeWork(work.block)
			s.notifyWork()

		case work := <-s.fetchWorkCh:
			// Return current mining work to remote miner.
			if s.currentBlock == nil {
				work.errc <- errNoMiningWork
			} else {
				work.res <- s.currentWork
			}

		case result := <-s.submitWorkCh:
			// Verify submitted PoS solution based on maintained mining blocks.
			if pandora.submitWork(result.nonce, result.mixDigest, result.hash, result.blsSeal) {
				result.errc <- nil
			} else {
				result.errc <- errInvalidSealResult
			}

		case result := <-s.submitRateCh:
			// Trace remote sealer's hash rate by submitted value.
			s.rates[result.id] = hashrate{rate: result.rate, ping: time.Now()}
			close(result.done)

		case req := <-s.fetchRateCh:
			// Gather all hash rate submitted by remote sealer.
			var total uint64
			for _, rate := range s.rates {
				// this could overflow
				total += rate.rate
			}
			req <- total

		case <-ticker.C:
			// Clear stale submitted hash rate.
			for id, rate := range s.rates {
				if time.Since(rate.ping) > 10*time.Second {
					delete(s.rates, id)
				}
			}
			// Clear stale pending blocks
			if s.currentBlock != nil {
				for hash, block := range s.works {
					if block.NumberU64()+staleThreshold <= s.currentBlock.NumberU64() {
						delete(s.works, hash)
					}
				}
			}

		case <-s.requestExit:
			return
		}
	}
}

// makeWork creates a work package for external miner.
//
// The work package consists of 3 strings:
//   result[0], 32 bytes hex encoded current block header pos-hash
//   result[1], 32 bytes hex encoded receipt hash for transaction proof
//   result[2], hex encoded rlp block header
//   result[3], hex encoded block number
func (pandora *Pandora) makeWork(block *types.Block) {
	sealer := pandora.sealer
	rlpHeader, _ := rlp.EncodeToBytes(block.Header())

	hash := sealer.ethash.SealHash(block.Header())
	sealer.currentWork[0] = hash.Hex()
	sealer.currentWork[1] = block.Header().ReceiptHash.Hex()
	sealer.currentWork[2] = hexutil.Encode(rlpHeader)
	sealer.currentWork[3] = hexutil.Encode(block.Header().Number.Bytes())

	// Trace the seal work fetched by remote sealer.
	sealer.currentBlock = block
	sealer.works[hash] = block
}

// submitWork verifies the submitted pow solution, returning
// whether the solution was accepted or not (not can be both a bad pow as well as
// any other error, like no pending work or stale mining result).
func (pandora *Pandora) submitWork(nonce types.BlockNonce, mixDigest common.Hash, sealhash common.Hash, blsSignatureBytes *BlsSignatureBytes) bool {
	sealer := pandora.sealer
	if sealer.currentBlock == nil {
		sealer.ethash.config.Log.Error("Pending work without block", "sealhash", sealhash)
		return false
	}
	// Make sure the work submitted is present
	block := sealer.works[sealhash]
	if block == nil {
		sealer.ethash.config.Log.Warn("Work submitted but none pending", "sealhash", sealhash, "curnumber", sealer.currentBlock.NumberU64())
		return false
	}
	// Verify the correctness of submitted result.
	header := block.Header()
	header.Nonce = nonce
	header.MixDigest = mixDigest
	extraDataWithSignature := new(PandoraExtraDataSealed)
	blsSignature, err := herumi.SignatureFromBytes(blsSignatureBytes[:])

	if nil != err {
		return false
	}

	pandoraExtraData := new(PandoraExtraData)
	err = rlp.DecodeBytes(header.Extra, pandoraExtraData)

	if nil != err {
		return false
	}

	extraDataWithSignature.FromExtraDataAndSignature(*pandoraExtraData, blsSignature)
	header.Extra, err = rlp.EncodeToBytes(extraDataWithSignature)

	if nil != err {
		sealer.ethash.config.Log.Warn("Invalid extraData in header", "sealhash", sealhash, "err", err)
		return false
	}

	start := time.Now()
	if !sealer.noverify {
		if err := sealer.ethash.verifySeal(nil, header, true); err != nil {
			sealer.ethash.config.Log.Warn("Invalid proof-of-work submitted", "sealhash", sealhash, "elapsed", common.PrettyDuration(time.Since(start)), "err", err)
			return false
		}
	}
	// Make sure the result channel is assigned.
	if sealer.results == nil {
		sealer.ethash.config.Log.Warn("Ethash result channel is empty, submitted mining result is rejected")
		return false
	}
	sealer.ethash.config.Log.Trace("Verified correct proof-of-work", "sealhash", sealhash, "elapsed", common.PrettyDuration(time.Since(start)))

	// Solutions seems to be valid, return to the miner and notify acceptance.
	solution := block.WithSeal(header)

	// The submitted solution is within the scope of acceptance.
	if solution.NumberU64()+staleThreshold > sealer.currentBlock.NumberU64() {
		select {
		case sealer.results <- solution:
			sealer.ethash.config.Log.Debug("Work submitted is acceptable", "number", solution.NumberU64(), "sealhash", sealhash, "hash", solution.Hash())
			return true
		default:
			sealer.ethash.config.Log.Warn("Sealing result is not read by miner", "mode", "remote", "sealhash", sealhash)
			return false
		}
	}
	// The submitted block is too old to accept, drop it.
	sealer.ethash.config.Log.Warn("Work submitted is too old", "number", solution.NumberU64(), "sealhash", sealhash, "hash", solution.Hash())
	return false
}

// NewMnimalConsensusInfo should be used to represent validator set for epoch
func NewMinimalConsensusInfo(epoch uint64) (consensusInfo interface{}) {
	consensusInfo = &MinimalEpochConsensusInfo{
		Epoch:            epoch,
		SlotTimeDuration: SlotTimeDuration,
	}
	return
}

func (pandoraMode *MinimalEpochConsensusInfo) AssignValidators(validatorsList [validatorListLen]common2.PublicKey) {
	pandoraMode.ValidatorsList = validatorsList
}

// This function should be used to extract epoch start from genesis
func (pandoraMode *MinimalEpochConsensusInfo) AssignEpochStartFromGenesis(genesisTime time.Time) {
	epochNumber := pandoraMode.Epoch
	genesisTimeUnix := uint64(genesisTime.Unix())
	slotDuration := pandoraMode.SlotTimeDuration * time.Second
	timePassed := epochNumber*uint64(slotDuration.Seconds())*uint64(validatorListLen) + genesisTimeUnix
	pandoraMode.EpochTimeStart = time.Unix(int64(timePassed), 0)
	pandoraMode.EpochTimeStartUnix = uint64(pandoraMode.EpochTimeStart.Unix())
}

func (ethash *Ethash) IsPandoraModeEnabled() (isPandora bool) {
	return ModePandora == ethash.config.PowMode
}

// In subscription design we should listen to any minimal consensus information that was passed
// from vanguard to orchestrator. As a param we pass epoch number which will be used to determine from which point
// we should start receiving subscriptions.
// This process should be infinite.
// For the first iteration we use first of notify urls to reach orchestrator
func (pandora *Pandora) SubscribeToMinimalConsensusInformation(epoch uint64, ctx context.Context) (
	subscription *rpc.ClientSubscription,
	channel chan *MinimalEpochConsensusInfoPayload,
	err error,
	errChan chan error,
) {
	orchestratorEndpoints := pandora.sealer.notifyURLs
	notifyURLsLen := len(pandora.sealer.notifyURLs)

	if notifyURLsLen < 1 {
		err = fmt.Errorf("there must be at least one in notifyURLs, got: %d", notifyURLsLen)

		return
	}

	// We use only first
	orchestratorEndpoint := orchestratorEndpoints[0]
	client, err := rpc.Dial(orchestratorEndpoint)

	if nil != err {
		return
	}

	channel = make(chan *MinimalEpochConsensusInfoPayload)
	subscription, err = client.Subscribe(
		ctx,
		"orc",
		channel,
		"minimalConsensusInfo",
		epoch,
	)

	if nil != err {
		return
	}

	return
}

func (ethash *Ethash) getMinimalConsensus(header *types.Header) (
	minimalConsensus *MinimalEpochConsensusInfo,
	err error,
) {
	mciCache := ethash.mci

	if nil == mciCache {
		err = fmt.Errorf("mci lru cache cannot be empty to run pandora mode")

		return
	}

	// Retrieve genesis info for derivation
	cache := mciCache.cache
	genesisInfo, okGenesis := cache.Get(0)

	if !okGenesis {
		err = fmt.Errorf("cannot get minimal consensus info for genesis")

		return
	}

	minimalGenesisConsensusInfo := genesisInfo.(*MinimalEpochConsensusInfo)
	genesisStart := minimalGenesisConsensusInfo.EpochTimeStart

	// Extract epoch
	headerTime := header.Time
	relativeTime := headerTime - uint64(genesisStart.Unix())

	if relativeTime < 0 {
		err = fmt.Errorf(
			"awaiting for vanguard to start. Left: %ds",
			time.Unix(int64(relativeTime)*-1, 0).Second())
		log.Error(err.Error())

		return
	}

	derivedEpoch := int(relativeTime / (pandoraEpochLength * SlotTimeDuration))

	// Get minimal consensus info for counted epoch
	minimalConsensusCache, okDerived := cache.Get(derivedEpoch)

	if !okDerived {
		err = fmt.Errorf(
			"missing minimal consensus info for epoch %d, relative: %d, start: %d",
			derivedEpoch,
			relativeTime,
			genesisStart.Unix(),
		)

		return
	}

	minimalConsensus = minimalConsensusCache.(*MinimalEpochConsensusInfo)

	return
}

func (ethash *Ethash) PreparePandoraHeader(header *types.Header) (err error) {
	minimalConsensus, err := ethash.getMinimalConsensus(header)

	if nil != err {
		return
	}

	extraData, err := NewPandoraExtraData(header, minimalConsensus)

	if nil != err {
		return
	}

	encodedExtraData, err := rlp.EncodeToBytes(extraData)

	if nil != err {
		return
	}

	header.Extra = encodedExtraData

	return
}

func (pandoraMode *MinimalEpochConsensusInfo) extractValidator(timestamp uint64) (
	err error,
	extractedTurn uint64,
	validator common2.PublicKey,
) {
	epochTimeStart := pandoraMode.EpochTimeStart
	epochDuration := pandoraEpochLength * time.Duration(SlotTimeDuration) * time.Second
	epochTimeEnd := epochTimeStart.Add(epochDuration)

	if timestamp < uint64(epochTimeStart.Unix()) || timestamp >= uint64(epochTimeEnd.Unix()) {
		err = fmt.Errorf(
			"time not within expected boundary. Got: %d, and should be from: %d to: %d",
			timestamp,
			epochTimeStart.Unix(),
			epochTimeEnd.Unix(),
		)

		return
	}

	extractedTurn = (timestamp - uint64(epochTimeStart.Unix())) / SlotTimeDuration

	// Check to not overflow the index
	if extractedTurn > uint64(len(pandoraMode.ValidatorsList)) {
		err = fmt.Errorf("extracted validator index overflows validator length")

		return
	}

	validator = pandoraMode.ValidatorsList[extractedTurn]

	return
}

func (ethash *Ethash) verifyPandoraHeader(header *types.Header) (err error) {
	headerTime := header.Time
	minimalConsensus, err := ethash.getMinimalConsensus(header)

	if nil != err {
		return
	}

	// Check if time slot is within desired boundaries. To consider if needed.
	// We could maybe have an assumption that cache should be invalidated before use.
	err, _, publicKey := minimalConsensus.extractValidator(headerTime)

	if nil != err {
		return
	}

	pandoraExtraDataSealed := new(PandoraExtraDataSealed)
	err = rlp.DecodeBytes(header.Extra, pandoraExtraDataSealed)

	if nil != err {
		return
	}

	blsSginatureBytes := pandoraExtraDataSealed.BlsSignatureBytes
	signature, err := herumi.SignatureFromBytes(blsSginatureBytes[:])

	if nil != err {
		return
	}

	// Check if signature of header is valid
	sealHash := ethash.SealHash(header)
	signatureValid := signature.Verify(publicKey, sealHash[:])

	// Seal signature verification has higher priority than integrity of the header itself
	// TODO: this should be somehow distributed to slashing.
	// If you sign dumb stuff and you are already a validator you should get slashed.
	// Problematic if key got leaked, validator then could loose all the stake
	// In my opinion if somebody looses the staked key he already lost his usefulness to the network
	if !signatureValid {
		err = fmt.Errorf(
			"invalid signature: %s in header hash: %s with sealHash: %s",
			signature.Marshal(),
			header.Hash().String(),
			sealHash.String(),
		)

		return
	}

	expectedExtra, err := NewPandoraExtraData(header, minimalConsensus)

	if nil != err {
		return
	}

	expectedExtraDataSealed := new(PandoraExtraDataSealed)
	commonSignature, err := herumi.SignatureFromBytes(pandoraExtraDataSealed.BlsSignatureBytes[:])

	if nil != err {
		return
	}

	expectedExtraDataSealed.FromExtraDataAndSignature(*expectedExtra, commonSignature)
	expectedRlp, err := rlp.EncodeToBytes(expectedExtraDataSealed)

	if nil != err {
		return
	}

	if !bytes.Equal(expectedRlp, header.Extra) {
		err = fmt.Errorf("invalid extraData field, expected: %v, got %v", expectedExtra, pandoraExtraDataSealed)
	}

	return
}

// This should be used only by trusted orchestrator
func (ethash *Ethash) InsertMinimalConsensusInfo(
	epoch uint64,
	consensusInfo *MinimalEpochConsensusInfo,
) (err error) {
	config := ethash.config
	powMode := config.PowMode

	if ModePandora != powMode {
		err = fmt.Errorf("ethash is not in pandora mode")

		return
	}

	convertedInfo := NewMinimalConsensusInfo(consensusInfo.Epoch)
	pandoraConsensusInfo := convertedInfo.(*MinimalEpochConsensusInfo)
	pandoraConsensusInfo.EpochTimeStartUnix = consensusInfo.EpochTimeStartUnix
	pandoraConsensusInfo.EpochTimeStart = consensusInfo.EpochTimeStart

	// In this mode we do not invalidate the mciCache!
	// This is hell risky, we should first get epoch from mciCache and check if it is already inserted.
	// If so, we need to resolve this conflict
	pandoraConsensusInfo.AssignValidators(consensusInfo.ValidatorsList)
	mci := ethash.mci
	mciCache := mci.cache
	mciCache.Add(int(epoch), pandoraConsensusInfo)

	return
}

func (ethash *Ethash) IsMinimalConsensusPresentForTime(timestamp uint64) (present bool) {
	header := &types.Header{Time: timestamp}
	_, err := ethash.getMinimalConsensus(header)
	present = nil == err

	return
}

func (ethash *Ethash) IsInGenesisSlot(timestamp uint64) (isGenesisSlot bool) {
	mci := ethash.mci
	mciCache := mci.cache
	genesisInfo, present := mciCache.Get(0)

	if !present {
		return
	}

	minimalConsensusInfo := genesisInfo.(*MinimalEpochConsensusInfo)
	epoch := minimalConsensusInfo.Epoch

	if 0 != epoch {
		return
	}

	err, index, _ := minimalConsensusInfo.extractValidator(timestamp)

	if nil != err {
		return
	}

	return index == 0
}

func NewPandoraExtraData(header *types.Header, minimalConsensus *MinimalEpochConsensusInfo) (
	extraData *PandoraExtraData,
	err error,
) {
	derivedEpoch := minimalConsensus.Epoch
	epochTimeStart := minimalConsensus.EpochTimeStart
	headerTime := header.Time

	extractedTurn := (headerTime - uint64(epochTimeStart.Unix())) / SlotTimeDuration

	// Check to not overflow the index
	if extractedTurn > uint64(len(minimalConsensus.ValidatorsList)) {
		err = fmt.Errorf("extracted validator index overflows validator length")

		return
	}

	extraData = &PandoraExtraData{
		Slot:  uint64(len(minimalConsensus.ValidatorsList))*derivedEpoch + extractedTurn,
		Epoch: derivedEpoch,
		Turn:  extractedTurn,
	}

	return
}

func (pandoraExtraDataSealed *PandoraExtraDataSealed) FromExtraDataAndSignature(
	pandoraExtraData PandoraExtraData,
	signature common2.Signature,
) {
	var blsSignatureBytes BlsSignatureBytes
	signatureBytes := signature.Marshal()

	if len(signatureBytes) != signatureSize {
		panic("Incompatible bls mode detected")
	}

	copy(blsSignatureBytes[:], signatureBytes[:])
	pandoraExtraDataSealed.PandoraExtraData = pandoraExtraData
	pandoraExtraDataSealed.BlsSignatureBytes = &blsSignatureBytes
}

func (pandoraExtraDataSealed *PandoraExtraDataSealed) FromHeader(header *types.Header) {
	err := rlp.DecodeBytes(header.Extra, pandoraExtraDataSealed)

	if nil != err {
		panic(err.Error())
	}
}
