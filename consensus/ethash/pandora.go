package ethash

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/silesiacoin/bls/herumi"
	"time"
	vbls "vuvuzela.io/crypto/bls"
)

const (
	// Time expressed in seconds
	slotTimeDuration = 6
)

// Use decorator pattern to get there as fast as possible
// This is a prototype, it can be designed way better
type Pandora struct {
	sealer *remoteSealer
}

// This should be cached or retrieved in a handshake with vanguard
type MinimalEpochConsensusInfo struct {
	// Epoch number
	epoch uint64
	// Validators list 32 public bls keys. slot(n) in epoch is represented by index(n) in MinimalConsensusInfo
	validatorsList [32]*vbls.PublicKey
	// Unix timestamp of consensus start. This will be used to extract time slot
	epochTimeStart time.Time
	// Slot time duration
	slotTimeDuration time.Duration
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
			// Verify submitted PoW solution based on maintained mining blocks.
			// TODO: change verification of submitted block
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
		epoch:            epoch,
		slotTimeDuration: slotTimeDuration,
	}
	return
}

func (pandoraMode *MinimalEpochConsensusInfo) AssignValidators(validatorsList [32]*vbls.PublicKey) {
	pandoraMode.validatorsList = validatorsList
	return
}

// This function should be used to extract epoch start from genesis
func (pandoraMode *MinimalEpochConsensusInfo) AssignEpochStartFromGenesis(genesisTime uint64) {
	epochNumber := pandoraMode.epoch
	// validator should be unique per epoch
	slotsPerEpoch := uint64(len(pandoraMode.validatorsList))
	slotDuration := pandoraMode.slotTimeDuration * time.Second
	timePassed := epochNumber*slotsPerEpoch*uint64(slotDuration.Nanoseconds()) + genesisTime
	pandoraMode.epochTimeStart = time.Time{}.Add(time.Duration(timePassed))
}

// EpochSet will retrieve minimalConsensusInfo for epoch derived from block height and its future
func (ethash *Ethash) epochSet(block uint64) (current *MinimalEpochConsensusInfo, ok bool) {
	epoch := block / pandoraEpochLength
	currentSet, ok := ethash.mci.cache.Get(epoch)
	current = currentSet.(*MinimalEpochConsensusInfo)

	return
}

func (ethash *Ethash) verifyPandoraHeader(header *types.Header) (err error) {
	mciCache := ethash.mci

	if nil == mciCache {
		err = fmt.Errorf("mci lru cache cannot be empty to run pandora mode")

		return
	}

	// Retrieve genesis info for derivation
	cache := mciCache.cache
	genesisInfo, ok := cache.Get(0)

	if !ok {
		err = fmt.Errorf("cannot get minimal consensus info for genesis")

		return
	}

	minimalGenesisConsensusInfo := genesisInfo.(*MinimalEpochConsensusInfo)
	genesisStart := minimalGenesisConsensusInfo.epochTimeStart

	// Extract epoch
	headerTime := header.Time
	relativeTime := headerTime - uint64(genesisStart.Unix())
	derivedEpoch := relativeTime / pandoraEpochLength

	// Get minimal consensus info for counted epoch
	minimalConsensusCache, ok := cache.Get(derivedEpoch)

	if !ok {
		err = fmt.Errorf("missing minimal consensus info for epoch %d", derivedEpoch)

		return
	}

	minimalConsensus := minimalConsensusCache.(*MinimalEpochConsensusInfo)

	// Check if time slot is within desired boundaries. To consider if needed.
	// We could maybe have an assumption that cache should be invalidated before use.
	epochTimeStart := minimalConsensus.epochTimeStart
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

	extractedValidatorIndex := (headerTime-uint64(epochTimeStart.Unix()))/slotTimeDuration - 1
	publicKey := minimalConsensus.validatorsList[extractedValidatorIndex]
	mixDigest := header.MixDigest
	// Check if signature of header is valid
	messages := make([][]byte, 0)
	sealHash := ethash.SealHash(header)
	messages = append(messages, sealHash.Bytes())
	signature := [32]byte{}
	copy(signature[:], mixDigest.Bytes())
	pubKeySet := make([]*vbls.PublicKey, 0)
	pubKeySet = append(pubKeySet, publicKey)
	signatureValid := herumi.VerifyCompressed(pubKeySet, messages, &signature)

	if !signatureValid {
		err = fmt.Errorf(
			"invalid mixDigest: %s in header hash: %s",
			header.MixDigest.String(),
			header.Hash().String(),
		)
	}

	return
}
