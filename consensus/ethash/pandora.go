package ethash

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"math/big"
	"time"
)

// Use decorator pattern to get there as fast as possible
// This is a prototype, it can be designed way better
type Pandora struct {
	sealer *remoteSealer
}

// This should be cached or retrieved in a handshake with vanguard
type MinimalEpochConsensusInfo struct {
	// Epoch number
	EpochNumber *big.Int
	// Change it to BLS signature keys
	ValidatorList []string
	// Unix timestamp of consensus start. This will be used to extract time slot
	GenesisStart time.Time
	// Slot time duration
	SlotTimeDuration time.Duration
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

// This is inserted here temporarily to show what should be implemented in vanguard to make this symbiosis work.
func vanguardFunction(block *types.Block) (err error) {
	// HEADER CREATION HERE ON REMOTE SEALER, empty comments have TODO()
	header := block.Header()
	// - get parentHash from previous slot and check with given parentHash
	parentHash := header.ParentHash
	// TODO: Fetch from beacon block hash of the parent, if it does not match. Make it dynamic check
	expectedHash := common.HexToHash("0x0000000000000000000000000000001")

	if parentHash.String() != expectedHash.String() {
		err = fmt.Errorf("parentHash delivered: %s does not match the latest block hash: %s",
			parentHash.String(),
			expectedHash.String(),
		)

		return
	}

	// - generate timestamp
	header.Time = uint64(time.Now().UnixNano())
	// - generate coinbase // -> IMHO coinbase should be generated on PANDORA side as it is in mining package
	zeroAddress := common.Address{}

	if zeroAddress == header.Coinbase {
		// As a fallback we could use BLS pubkey converted to common.Address
		err = fmt.Errorf("coinbase for sigining cannot be empty, refusing to sign")

		return
	}

	// - generate blockNumber // -> this should be forwarded first and received from PANDORA in this approach
	if header.Number.Cmp(big.NewInt(0)) == -1 {
		err = fmt.Errorf("negative block number")

		return
	}

	// - insert randao into difficulty, not to the nonce -> To debate about it
	// could be epochs randao reveal

	// Get from extraData field:
	// Very dummy check for now, extend it
	minExtradataLen := 1

	if len(header.Extra) < minExtradataLen {
		err = fmt.Errorf(
			"invalid extradataField len, got: %d, expected at least: %d",
			len(header.Extra),
			minExtradataLen,
		)

		return
	}

	return
}
