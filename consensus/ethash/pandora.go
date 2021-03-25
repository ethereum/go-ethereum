package ethash

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	common2 "github.com/silesiacoin/bls/common"
	"github.com/silesiacoin/bls/herumi"
	"time"
)

const (
	// Time expressed in seconds
	slotTimeDuration = 6
	signatureSize    = 96
)

// Use decorator pattern to get there as fast as possible
// This is a prototype, it can be designed way better
type Pandora struct {
	sealer *remoteSealer
}

type BlsSignatureBytes [signatureSize]byte

type PandoraExtraData struct {
	Slot          uint64
	Epoch         uint64
	ProposerIndex uint64
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
	ValidatorsList [32]common2.PublicKey `json:"validatorList"`
	// Unix timestamp of consensus start. This will be used to extract time slot
	EpochTimeStart time.Time

	EpochTimeStartUnix uint64 `json:"epochTimeStart"`

	// Slot time duration
	SlotTimeDuration time.Duration `json:"slotTimeDuration"`
}

// This is done only to have vanguard spec done in minimal codebase to exchange informations with pandora.
// In this approach you could have multiple execution engines connected via urls []string
// In this approach you are also compatible with any current toolsets for mining because you use already defined api
func StartRemotePandora(executionEngine *Ethash, urls []string, noverify bool) (sealer *remoteSealer) {
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

	return
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
			if s.submitWork(result.nonce, result.mixDigest, result.hash) {
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

	return
}

// NewMnimalConsensusInfo should be used to represent validator set for epoch
func NewMinimalConsensusInfo(epoch uint64) (consensusInfo interface{}) {
	consensusInfo = &MinimalEpochConsensusInfo{
		Epoch:            epoch,
		SlotTimeDuration: slotTimeDuration,
	}
	return
}

func (pandoraMode *MinimalEpochConsensusInfo) AssignValidators(validatorsList [32]common2.PublicKey) {
	pandoraMode.ValidatorsList = validatorsList
	return
}

// This function should be used to extract epoch start from genesis
func (pandoraMode *MinimalEpochConsensusInfo) AssignEpochStartFromGenesis(genesisTime time.Time) {
	epochNumber := pandoraMode.Epoch
	genesisTimeUnix := uint64(genesisTime.Unix())
	slotDuration := pandoraMode.SlotTimeDuration * time.Second
	timePassed := epochNumber*uint64(slotDuration.Seconds()) + genesisTimeUnix
	pandoraMode.EpochTimeStart = time.Unix(int64(timePassed), 0)
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
	derivedEpoch := int(relativeTime / (pandoraEpochLength * slotTimeDuration))

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

func (ethash *Ethash) verifyPandoraHeader(header *types.Header) (err error) {
	headerTime := header.Time
	minimalConsensus, err := ethash.getMinimalConsensus(header)

	if nil != err {
		return
	}

	// Check if time slot is within desired boundaries. To consider if needed.
	// We could maybe have an assumption that cache should be invalidated before use.
	epochTimeStart := minimalConsensus.EpochTimeStart
	epochDuration := pandoraEpochLength * time.Duration(slotTimeDuration) * time.Second
	epochTimeEnd := epochTimeStart.Add(epochDuration)

	if headerTime < uint64(epochTimeStart.Unix()) || headerTime >= uint64(epochTimeEnd.Unix()) {
		err = fmt.Errorf(
			"header time not within expected boundary. Got: %d, and should be from: %d to: %d",
			headerTime,
			epochTimeStart.Unix(),
			epochTimeEnd.Unix(),
		)

		return
	}

	extractedProposerIndex := (headerTime - uint64(epochTimeStart.Unix())) / slotTimeDuration

	// Check to not overflow the index
	if extractedProposerIndex > uint64(len(minimalConsensus.ValidatorsList)) {
		err = fmt.Errorf("extracted validator index overflows validator length")

		return
	}

	publicKey := minimalConsensus.ValidatorsList[extractedProposerIndex]
	blsSginatureBytes := BlsSignatureBytes{}
	copy(blsSginatureBytes[:], header.Extra[len(header.Extra)-signatureSize:])
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

	// Verify extraData field (does they match derived one)
	// to consider if we really want to do this here
	extraData := &PandoraExtraData{}
	err = rlp.DecodeBytes(header.Extra, extraData)

	if nil != err {
		return
	}

	expectedExtra, err := NewPandoraExtraData(header, minimalConsensus)

	if nil != err {
		return
	}

	expectedRlp, err := rlp.EncodeToBytes(expectedExtra)

	if nil != err {
		return
	}

	// Add signature to expected extraData
	copy(expectedRlp[:], signature.Marshal())

	if !bytes.Equal(expectedRlp, header.Extra) {
		err = fmt.Errorf("invalid extraData field, expected: %v, got %v", expectedExtra, extraData)
	}

	return
}

func NewPandoraExtraData(header *types.Header, minimalConsensus *MinimalEpochConsensusInfo) (
	extraData *PandoraExtraData,
	err error,
) {
	derivedEpoch := minimalConsensus.Epoch
	epochTimeStart := minimalConsensus.EpochTimeStart
	headerTime := header.Time

	extractedProposerIndex := (headerTime - uint64(epochTimeStart.Unix())) / slotTimeDuration

	// Check to not overflow the index
	if extractedProposerIndex > uint64(len(minimalConsensus.ValidatorsList)) {
		err = fmt.Errorf("extracted validator index overflows validator length")

		return
	}

	extraData = &PandoraExtraData{
		Slot:          uint64(len(minimalConsensus.ValidatorsList))*derivedEpoch + extractedProposerIndex,
		Epoch:         derivedEpoch,
		ProposerIndex: extractedProposerIndex,
	}

	return
}

func (pandoraExtraDataSealed *PandoraExtraDataSealed) FromExtraDataAndSignature(
	pandoraExtraData PandoraExtraData,
	signature herumi.Signature,
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
