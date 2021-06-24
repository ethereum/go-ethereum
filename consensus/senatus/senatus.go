package senatus

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	lru "github.com/hashicorp/golang-lru"
	"golang.org/x/crypto/sha3"
)

const (
	inMemorySnapshots  = 128  // Number of recent snapshots to keep in memory
	inMemorySignatures = 4096 // Number of recent block signatures to keep in memory

	checkpointInterval = 1024        // Number of blocks after which to save the snapshot to the database
	defaultEpochLength = uint64(200) // Default number of blocks of checkpoint to update validatorSet from contract

	extraVanity = 32 // Fixed number of extra-data prefix bytes reserved for signer vanity

	extraSeal = 65 // Fixed number of extra-data suffix bytes reserved for signer seal

	validatorBytesLength = common.AddressLength
	wiggleTime           = 500 * time.Millisecond // Random delay (per validator) to allow concurrent validators
	fixedWiggleTime      = 200 * time.Millisecond // Fixed delay time for out-of-turn validator
	maxValidatorNum      = 101                    // Max validators allowed to seal.
)

var (
	uncleHash = types.CalcUncleHash(nil) // Always Keccak256(RLP([])) as uncles are meaningless outside of PoW.

	diffInTurn = big.NewInt(2) // Block difficulty for in-turn signatures
	diffNoTurn = big.NewInt(1) // Block difficulty for out-of-turn signatures

	BlockReward = big.NewInt(1e+18) // Block reward

	validatorContract     = "0x0000000000000000000000000000000000001000"
	slashContract         = "0x0000000000000000000000000000000000001001"
	validatorContractAddr = common.HexToAddress(validatorContract)
	slashContractAddr     = common.HexToAddress(slashContract)
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	// errUnknownBlock is returned when the list of validators is requested for a block
	// that is not part of the local blockchain.
	errUnknownBlock = errors.New("unknown block")

	// errMissingVanity is returned if a block's extra-data section is shorter than
	// 32 bytes, which is required to store the validator vanity.
	errMissingVanity = errors.New("extra-data 32 byte vanity prefix missing")

	// errMissingSignature is returned if a block's extra-data section doesn't seem
	// to contain a 65 byte secp256k1 signature.
	errMissingSignature = errors.New("extra-data 65 byte signature suffix missing")

	// errExtraValidators is returned if non-checkpoint block contain validator data in
	// their extra-data fields.
	errExtraValidators = errors.New("non-checkpoint block contains extra validator list")

	// errInvalidMixDigest is returned if a block's mix digest is non-zero.
	errInvalidMixDigest = errors.New("non-zero mix digest")

	// errInvalidUncleHash is returned if a block contains an non-empty uncle list.
	errInvalidUncleHash = errors.New("non empty uncle hash")

	// errInvalidDifficulty is returned if the difficulty of a block neither 1 or 2.
	errInvalidDifficulty = errors.New("invalid difficulty")

	// errWrongDifficulty is returned if the difficulty of a block doesn't match the
	// turn of the validator.
	errWrongDifficulty = errors.New("wrong difficulty")

	// errOutOfRangeChain is returned if an authorization list is attempted to
	// be modified via out-of-range or non-contiguous headers.
	errOutOfRangeChain = errors.New("out of range or non-contiguous chain")

	// errBlockHashInconsistent is returned if an authorization list is attempted to
	// insert an inconsistent block.
	errBlockHashInconsistent = errors.New("the block hash is inconsistent")

	// ErrInvalidTimestamp is returned if the timestamp of a block is lower than
	// the previous block's timestamp + the minimum block period.
	ErrInvalidTimestamp = errors.New("invalid timestamp")

	// errMismatchingEpochValidators is returned if a sprint block contains a
	// list of validators different than the one the local node calculated.
	errMismatchingEpochValidators = errors.New("mismatching validator list on epoch block")
	// errUnauthorizedValidator is returned if a header is signed by a non-authorized entity.
	errUnauthorizedValidator = errors.New("unauthorized validator")

	// errRecentlySigned is returned if a header is signed by an authorized entity
	// that already signed a header recently, thus is temporarily not allowed to.
	errRecentlySigned = errors.New("recently signed")

	// errInvalidValidatorLen is returned if validators length is zero or bigger than maxValidatorNum.
	errInvalidValidatorsLength = errors.New("invalid validators length")

	// errInvalidGasPrice is return if tx gas price is less than minimalGasPrice
	errInvalidGasPrice = errors.New("invalid gas price")

	// errUnauthorizedTransaction is return if tx is invalid system contract tx
	errUnauthorizedTransaction = errors.New("UnAuthorized transaction")
)

// SignerFn is a signer callback function to request a header to be signed by a
// backing account.
type SignerFn func(accounts.Account, string, []byte) ([]byte, error)
type SignerTxFn func(accounts.Account, *types.Transaction, *big.Int) (*types.Transaction, error)

// ecrecover extracts the Ethereum account address from a signed header.
func ecrecover(header *types.Header, sigCache *lru.ARCCache, chainID *big.Int) (common.Address, error) {
	// If the signature's already cached, return that
	hash := header.Hash()
	if address, known := sigCache.Get(hash); known {
		return address.(common.Address), nil
	}
	// Retrieve the signature from the header extra-data
	if len(header.Extra) < extraSeal {
		return common.Address{}, errMissingSignature
	}
	signature := header.Extra[len(header.Extra)-extraSeal:]

	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(SealHash(header, chainID).Bytes(), signature)
	if err != nil {
		return common.Address{}, err
	}

	var validator common.Address
	copy(validator[:], crypto.Keccak256(pubkey[1:])[12:])

	sigCache.Add(hash, validator)
	return validator, nil
}

// Senatus is the proof-of-staking consensus engine of CSC(CoinEx Smart Chain).
type Senatus struct {
	chainConfig *params.ChainConfig
	config      *params.SenatusConfig // Consensus engine configuration parameters
	db          ethdb.Database        // Database to store and retrieve snapshot checkpoints

	recents    *lru.ARCCache // Snapshots for recent block to speed up reorgs
	signatures *lru.ARCCache // Signatures of recent blocks to speed up mining

	validator common.Address // Ethereum address of the signing key
	signFn    SignerFn       // Validator function to authorize hashes with
	signTxFn  SignerTxFn
	lock      sync.RWMutex // Protects the signer fields

	signer types.Signer

	ethAPI       *ethapi.PublicBlockChainAPI
	validatorABI abi.ABI
	slashABI     abi.ABI

	// The fields below are for testing only
	fakeDiff bool // Skip difficulty verifications
}

// New creates a Senatus consensus engine
func New(chainConfig *params.ChainConfig, db ethdb.Database, ethAPI *ethapi.PublicBlockChainAPI) *Senatus {
	senatusConfig := *chainConfig.Senatus
	if senatusConfig.Epoch == 0 {
		senatusConfig.Epoch = defaultEpochLength
	}

	// Allocate the snapshot caches and create the engine
	recents, _ := lru.NewARC(inMemorySnapshots)
	signatures, _ := lru.NewARC(inMemorySignatures)

	valABI, err := abi.JSON(strings.NewReader(validatorABI))
	if err != nil {
		panic(err)
	}

	slaABI, err := abi.JSON(strings.NewReader(slashABI))
	if err != nil {
		panic(err)
	}

	c := &Senatus{
		chainConfig:  chainConfig,
		config:       &senatusConfig,
		db:           db,
		recents:      recents,
		signatures:   signatures,
		ethAPI:       ethAPI,
		validatorABI: valABI,
		slashABI:     slaABI,
		signer:       types.NewEIP155Signer(chainConfig.ChainID),
	}
	return c
}

// Author implements consensus.Engine, returning the Ethereum address recovered
// from the signature in the header's extra-data section.
func (s *Senatus) Author(header *types.Header) (common.Address, error) {
	return ecrecover(header, s.signatures, s.chainConfig.ChainID)
}

// VerifyHeader checks whether a header conforms to the consensus rules.
func (s *Senatus) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, seal bool) error {
	return s.verifyHeader(chain, header, nil)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers. The
// method returns a quit channel to abort the operations and a results channel to
// retrieve the async verifications (the order is that of the input slice).
func (s *Senatus) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for i, header := range headers {
			err := s.verifyHeader(chain, header, headers[:i])

			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

// verifyHeader checks whether a header conforms to the consensus rules.The
// caller may optionally pass in a batch of parents (ascending order) to avoid
// looking those up from the database. This is useful for concurrently verifying
// a batch of new headers.
func (s *Senatus) verifyHeader(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
	if header.Number == nil {
		return errUnknownBlock
	}
	number := header.Number.Uint64()

	// Don't waste time checking blocks from the future
	if header.Time > uint64(time.Now().Unix()) {
		return consensus.ErrFutureBlock
	}
	// Check that the extra-data contains the vanity, validators and signature.
	if len(header.Extra) < extraVanity {
		return errMissingVanity
	}
	if len(header.Extra) < extraVanity+extraSeal {
		return errMissingSignature
	}

	// check extra data
	isEpoch := number%s.config.Epoch == 0

	// Ensure that the extra-data contains a signer list on checkpoint, but none otherwise
	validatorsBytes := len(header.Extra) - extraVanity - extraSeal
	if !isEpoch && validatorsBytes != 0 {
		return errExtraValidators
	}

	if isEpoch && validatorsBytes%validatorBytesLength != 0 {
		return errExtraValidators
	}

	// Ensure that the mix digest is zero as we don't have fork protection currently
	if header.MixDigest != (common.Hash{}) {
		return errInvalidMixDigest
	}
	// Ensure that the block doesn't contain any uncles which are meaningless in PoA
	if header.UncleHash != uncleHash {
		return errInvalidUncleHash
	}
	// Ensure that the block's difficulty is meaningful (may not be correct at this point)
	if number > 0 && header.Difficulty == nil {
		return errInvalidDifficulty
	}

	// If all checks passed, validate any special fields for hard forks
	if err := misc.VerifyForkHashes(chain.Config(), header, false); err != nil {
		return err
	}
	// All basic checks passed, verify cascading fields
	return s.verifyCascadingFields(chain, header, parents)
}

// verifyCascadingFields verifies all the header fields that are not standalone,
// rather depend on a batch of previous headers. The caller may optionally pass
// in a batch of parents (ascending order) to avoid looking those up from the
// database. This is useful for concurrently verifying a batch of new headers.
func (s *Senatus) verifyCascadingFields(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
	// The genesis block is the always valid dead-end
	number := header.Number.Uint64()
	if number == 0 {
		return nil
	}

	var parent *types.Header
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeader(header.ParentHash, number-1)
	}

	if parent == nil || parent.Number.Uint64() != number-1 || parent.Hash() != header.ParentHash {
		return consensus.ErrUnknownAncestor
	}

	if parent.Time+s.config.Period > header.Time {
		return ErrInvalidTimestamp
	}

	// Verify that the gas limit is <= MaxGasTarget and >= MinGasTarget
	if header.GasLimit > params.MaxGasTarget || header.GasLimit < params.MinGasTarget {
		return fmt.Errorf("invalid gasLimit: have %v, max: %v, min: %v", header.GasLimit, params.MaxGasTarget, params.MinGasTarget)
	}

	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}

	// Verify that the gas limit remains within allowed bounds
	diff := int64(parent.GasLimit) - int64(header.GasLimit)
	if diff < 0 {
		diff *= -1
	}
	limit := parent.GasLimit / params.GasLimitBoundDivisor

	if uint64(diff) >= limit {
		return fmt.Errorf("invalid gas limit: have %d, want %d += %d", header.GasLimit, parent.GasLimit, limit)
	}

	// All basic checks passed, verify the seal and return
	return s.verifySeal(chain, header, parents)
}

// snapshot retrieves the authorization snapshot at a given point in time.
func (s *Senatus) snapshot(chain consensus.ChainHeaderReader, number uint64, hash common.Hash, parents []*types.Header) (*Snapshot, error) {
	// Search for a snapshot in memory or on disk for checkpoints
	var (
		headers []*types.Header
		snap    *Snapshot
	)
	for snap == nil {
		// If an in-memory snapshot was found, use that
		if s, ok := s.recents.Get(hash); ok {
			snap = s.(*Snapshot)
			break
		}
		// If an on-disk checkpoint snapshot can be found, use that
		if number%checkpointInterval == 0 {
			if s, err := loadSnapshot(s.config, s.signatures, s.db, hash); err == nil {
				log.Trace("loaded snapshot from disk", "number", number, "hash", hash)
				snap = s
				break
			}
		}
		// If we're at the genesis, snapshot the initial state. Alternatively if we're
		// at a checkpoint block without a parent (light client CHT), or we have piled
		// up more headers than allowed to be reorged (chain reinit from a freezer),
		// consider the checkpoint trusted and snapshot it.
		if number == 0 || (number%s.config.Epoch == 0 && (len(headers) > params.FullImmutabilityThreshold || chain.GetHeaderByNumber(number-1) == nil)) {
			checkpoint := chain.GetHeaderByNumber(number)
			if checkpoint != nil {
				hash := checkpoint.Hash()
				// get validators from header
				validators := make([]common.Address, (len(checkpoint.Extra)-extraVanity-extraSeal)/common.AddressLength)
				for i := 0; i < len(validators); i++ {
					copy(validators[i][:], checkpoint.Extra[extraVanity+i*common.AddressLength:])
				}
				snap = newSnapshot(s.config, s.signatures, number, hash, validators)
				if err := snap.store(s.db); err != nil {
					return nil, err
				}
				log.Info("stored checkpoint snapshot to disk", "number", number, "hash", hash)
				break
			}
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
	snap, err := snap.apply(headers, chain, parents, s.chainConfig.ChainID)
	if err != nil {
		return nil, err
	}
	s.recents.Add(snap.Hash, snap)

	// If we've generated a new checkpoint snapshot, save to disk
	if snap.Number%checkpointInterval == 0 && len(headers) > 0 {
		if err = snap.store(s.db); err != nil {
			return nil, err
		}
	}
	return snap, err
}

// VerifyUncles implements consensus.Engine, always returning an error for any
// uncles as this consensus mechanism doesn't permit uncles.
func (s *Senatus) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errors.New("uncles not allowed")
	}
	return nil
}

// VerifySeal implements consensus.Engine, checking whether the signature contained
// in the header satisfies the consensus protocol requirements.
func (s *Senatus) VerifySeal(chain consensus.ChainHeaderReader, header *types.Header) error {
	return s.verifySeal(chain, header, nil)
}

// verifySeal checks whether the signature contained in the header satisfies the
// consensus protocol requirements. The method accepts an optional list of parent
// headers that aren't yet part of the local blockchain to generate the snapshots
// from.
func (s *Senatus) verifySeal(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
	// Verifying the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}
	// Retrieve the snapshot needed to verify this header and cache it
	snap, err := s.snapshot(chain, number-1, header.ParentHash, parents)
	if err != nil {
		return err
	}

	// Resolve the authorization key and check against signers
	signer, err := ecrecover(header, s.signatures, s.chainConfig.ChainID)
	if err != nil {
		return err
	}
	if _, ok := snap.Validators[signer]; !ok {
		return errUnauthorizedValidator
	}
	for seen, recent := range snap.Recents {
		if recent == signer {
			// Signer is among recents, only fail if the current block doesn't shift it out
			if limit := uint64(len(snap.Validators)/2 + 1); seen > number-limit {
				return errRecentlySigned
			}
		}
	}
	// Ensure that the difficulty corresponds to the turn-ness of the signer
	if !s.fakeDiff {
		inturn := snap.inturn(header.Number.Uint64(), signer)
		if inturn && header.Difficulty.Cmp(diffInTurn) != 0 {
			return errWrongDifficulty
		}
		if !inturn && header.Difficulty.Cmp(diffNoTurn) != 0 {
			return errWrongDifficulty
		}
	}
	return nil
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (s *Senatus) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	// If the block isn't a checkpoint, cast a random vote (good enough for now)
	header.Coinbase = s.validator
	header.Nonce = types.BlockNonce{}

	number := header.Number.Uint64()
	snap, err := s.snapshot(chain, number-1, header.ParentHash, nil)
	if err != nil {
		return err
	}

	// Set the correct difficulty
	header.Difficulty = calcDifficulty(snap, s.validator)

	// Ensure the extra data has all its components
	if len(header.Extra) < extraVanity {
		header.Extra = append(header.Extra, bytes.Repeat([]byte{0x00}, extraVanity-len(header.Extra))...)
	}
	header.Extra = header.Extra[:extraVanity]

	if number%s.config.Epoch == 0 {
		validators, err := s.getTopValidator(chain, header)
		if err != nil {
			return err
		}

		for _, validator := range validators {
			header.Extra = append(header.Extra, validator.Bytes()...)
		}
	}
	header.Extra = append(header.Extra, make([]byte, extraSeal)...)

	// Mix digest is reserved for now, set to empty
	header.MixDigest = common.Hash{}

	// Ensure the timestamp has the correct delay
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	header.Time = parent.Time + s.config.Period
	if header.Time < uint64(time.Now().Unix()) {
		header.Time = uint64(time.Now().Unix())
	}
	return nil
}

// Finalize implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given.
func (s *Senatus) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs *[]*types.Transaction, uncles []*types.Header, allLogs *[]*types.Log, receipts *[]*types.Receipt, systemTxs *[]*types.Transaction, usedGas *uint64) error {
	chainContext := chainContext{Chain: chain, senatus: s}
	// Initialize all system contracts at block 1
	if header.Number.Cmp(common.Big1) == 0 {
		if err := s.initializeSystemContracts(chain, state, header, chainContext,
			txs, allLogs, receipts, systemTxs, usedGas, false); err != nil {
			log.Error("initialize contract failed")
			return err
		}
		log.Trace("initialize system contract success")
	}

	if err := s.verifyTxsGasPrice(*txs, header); err != nil {
		log.Error("tx price under the minimal price of gas")
		return err
	}

	if header.Difficulty.Cmp(diffInTurn) != 0 {
		err := s.slash(chain, state, header, chainContext,
			txs, allLogs, receipts, systemTxs, usedGas, false)
		if err != nil {
			return err
		}
	}

	err := s.collectBlockReward(chain, state, header, chainContext)
	if err != nil {
		return err
	}
	// If the block is a epoch end block, verify the validator list
	// The verification can only be done when the state is ready, it can't be done in VerifyHeader.
	if header.Number.Uint64()%s.config.Epoch == 0 {
		err = s.distributeReward(chain, state, header, chainContext,
			txs, allLogs, receipts, systemTxs, usedGas, false)
		if err != nil {
			log.Error("distribute reward failed", "block hash", header.Hash(), "err", err)
			return err
		}

		newValidators, err := s.epochProcess(chain,
			state, header, chainContext,
			txs, allLogs, receipts, systemTxs, usedGas, false)
		if err != nil {
			return err
		}
		validatorsBytes := make([]byte, len(newValidators)*validatorBytesLength)
		for i, validator := range newValidators {
			copy(validatorsBytes[i*validatorBytesLength:], validator.Bytes())
		}

		extraSuffix := len(header.Extra) - extraSeal
		if !bytes.Equal(header.Extra[extraVanity:extraSuffix], validatorsBytes) {
			return errMismatchingEpochValidators
		}
	}
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	header.UncleHash = types.CalcUncleHash(nil)
	return nil
}

// FinalizeAndAssemble implements consensus.Engine, ensuring no uncles are set,
// nor block rewards given, and returns the final block.
func (s *Senatus) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, []*types.Receipt, error) {
	chainContext := chainContext{Chain: chain, senatus: s}
	// Initialize all system contracts at block 1
	if header.Number.Cmp(common.Big1) == 0 {
		if err := s.initializeSystemContracts(chain, state, header, chainContext,
			&txs, nil, &receipts, nil, &header.GasUsed, true); err != nil {
			log.Error("initialize contract failed when assemble block")
			return nil, nil, err
		}
		log.Trace("initialize system contract success")
	}

	if err := s.verifyTxsGasPrice(txs, header); err != nil {
		log.Error("tx price under the minimal price of gas when assemble block")
		return nil, nil, err
		//panic(err)
	}

	if header.Difficulty.Cmp(diffInTurn) != 0 {
		err := s.slash(chain, state, header, chainContext,
			&txs, nil, &receipts, nil, &header.GasUsed, true)
		if err != nil {
			log.Error("slash validator failed when assemble block", "block hash", header.Hash())
			return nil, nil, err
		}
	}

	err := s.collectBlockReward(chain, state, header, chainContext)
	if err != nil {
		return nil, nil, err
	}

	// If the block is a epoch end block, verify the validator list
	// The verification can only be done when the state is ready, it can't be done in VerifyHeader.
	if header.Number.Uint64()%s.config.Epoch == 0 {
		err = s.distributeReward(chain, state, header, chainContext,
			&txs, nil, &receipts, nil, &header.GasUsed, true)
		if err != nil {
			log.Error("distribute reward failed when assemble block", "block hash", header.Hash(), "err", err)
			return nil, nil, err
		}
		_, err = s.epochProcess(chain, state, header, chainContext, &txs, nil, &receipts, nil, &header.GasUsed, true)
		if err != nil {
			log.Error("epoch process failed", "block hash", header.Hash(), "err", err)
			return nil, nil, err
		}
	}

	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	header.UncleHash = types.CalcUncleHash(nil)
	// Assemble and return the final block for sealing
	return types.NewBlock(header, txs, nil, receipts, new(trie.Trie)), receipts, nil
}

// Authorize injects a private key into the consensus engine to mint new blocks
// with.
func (s *Senatus) Authorize(validator common.Address, signFn SignerFn, signTxFn SignerTxFn) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.validator = validator
	s.signFn = signFn
	s.signTxFn = signTxFn
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
func (s *Senatus) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	header := block.Header()

	// Sealing the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}
	// For 0-period chains, refuse to seal empty blocks (no reward but would spin sealing)
	if s.config.Period == 0 && len(block.Transactions()) == 0 {
		log.Info("sealing paused, waiting for transactions")
		return nil
	}
	// Don't hold the signer fields for the entire sealing procedure
	s.lock.RLock()
	validator, signFn := s.validator, s.signFn
	s.lock.RUnlock()

	// Bail out if we're unauthorized to sign a block
	snap, err := s.snapshot(chain, number-1, header.ParentHash, nil)
	if err != nil {
		return err
	}
	if _, authorized := snap.Validators[validator]; !authorized {
		return errUnauthorizedValidator
	}
	// If we're amongst the recent signers, wait for the next block
	for seen, recent := range snap.Recents {
		if recent == validator {
			// Salidator is among recents, only wait if the current block doesn't shift it out
			if limit := uint64(len(snap.Validators)/2 + 1); number < limit || seen > number-limit {
				log.Info("signed recently, must wait for others")
				return nil
			}
		}
	}
	// Sweet, the protocol permits us to sign the block, wait for our time
	delay := time.Unix(int64(header.Time), 0).Sub(time.Now()) // nolint: gosimple
	if header.Difficulty.Cmp(diffNoTurn) == 0 {
		// It's not our turn explicitly to sign, delay it a bit
		wiggle := time.Duration(len(snap.Validators)/2+1) * wiggleTime
		delay += time.Duration(rand.Int63n(int64(wiggle))) + fixedWiggleTime

		log.Trace("out-of-turn signing requested", "wiggle", common.PrettyDuration(wiggle))
	}
	// Sign all the things!
	sighash, err := signFn(accounts.Account{Address: validator}, accounts.MimetypeSenatus, SenatusRLP(header, s.chainConfig.ChainID))
	if err != nil {
		return err
	}
	copy(header.Extra[len(header.Extra)-extraSeal:], sighash)
	// Wait until sealing is terminated or delay timeout.
	log.Trace("waiting for slot to sign and propagate", "delay", common.PrettyDuration(delay))
	go func() {
		select {
		case <-stop:
			return
		case <-time.After(delay):
		}

		select {
		case results <- block.WithSeal(header):
		default:
			log.Warn("sealing result is not read by miner", "sealhash", SealHash(header, s.chainConfig.ChainID))
		}
	}()

	return nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have:
// * DIFF_NOTURN(2) if BLOCK_NUMBER % SIGNER_COUNT != SIGNER_INDEX
// * DIFF_INTURN(1) if BLOCK_NUMBER % SIGNER_COUNT == SIGNER_INDEX
func (s *Senatus) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	snap, err := s.snapshot(chain, parent.Number.Uint64(), parent.Hash(), nil)
	if err != nil {
		return nil
	}
	return calcDifficulty(snap, s.validator)
}

func calcDifficulty(snap *Snapshot, validator common.Address) *big.Int {
	if snap.inturn(snap.Number+1, validator) {
		return new(big.Int).Set(diffInTurn)
	}
	return new(big.Int).Set(diffNoTurn)
}

// SealHash returns the hash of a block prior to it being sealed.
func (s *Senatus) SealHash(header *types.Header) common.Hash {
	return SealHash(header, s.chainConfig.ChainID)
}

// Close implements consensus.Engine. It's a noop for senatus as there are no background threads.
func (s *Senatus) Close() error {
	return nil
}

// APIs implements consensus.Engine, returning the user facing RPC API to allow
// controlling the signer voting.
func (s *Senatus) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return []rpc.API{{
		Namespace: "senatus",
		Version:   "1.0",
		Service:   &API{chain: chain, senatus: s},
		Public:    false,
	}}
}

func CalcBlockReward(chainConfig *params.ChainConfig, blockNumber *big.Int) *big.Int {
	return BlockReward
}

// SealHash returns the hash of a block prior to it being sealed.
func SealHash(header *types.Header, chainID *big.Int) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()
	encodeSigHeader(hasher, header, chainID)
	hasher.Sum(hash[:0])
	return hash
}

// SenatusRLP returns the rlp bytes which needs to be signed for the proof-of-authority
// sealing. The RLP to sign consists of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
// Note, the method requires the extra data to be at least 65 bytes, otherwise it
// panics. This is done to avoid accidentally using both forms (signature present
// or not), which could be abused to produce different hashes for the same header.
func SenatusRLP(header *types.Header, chainID *big.Int) []byte {
	b := new(bytes.Buffer)
	encodeSigHeader(b, header, chainID)
	return b.Bytes()
}

func encodeSigHeader(w io.Writer, header *types.Header, chainID *big.Int) {
	err := rlp.Encode(w, []interface{}{
		chainID,
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra[:len(header.Extra)-crypto.SignatureLength], // Yes, this will panic if extra is too short
		header.MixDigest,
		header.Nonce,
	})
	if err != nil {
		panic("can't encode: " + err.Error())
	}
}

func (s *Senatus) initializeSystemContracts(chain consensus.ChainHeaderReader, state *state.StateDB, header *types.Header, chainContext core.ChainContext,
	txs *[]*types.Transaction, alllogs *[]*types.Log, receipts *[]*types.Receipt, receivedTxs *[]*types.Transaction, usedGas *uint64, mining bool) error {
	snap, err := s.snapshot(chain, 0, header.ParentHash, nil)
	if err != nil {
		return err
	}
	genesisValidators := snap.validators()
	if len(genesisValidators) == 0 || len(genesisValidators) > maxValidatorNum {
		return errInvalidValidatorsLength
	}
	method := "initialize"
	validatorData, err := s.validatorABI.Pack(method, genesisValidators)
	if err != nil {
		return err
	}

	slashData, err := s.slashABI.Pack(method)
	if err != nil {
		return err
	}

	contracts := []struct {
		addr common.Address
		data []byte
	}{
		{validatorContractAddr, validatorData},
		{slashContractAddr, slashData},
	}
	for _, c := range contracts {
		msg := s.wrapSystemContractMessage(header.Coinbase, c.addr, c.data, big.NewInt(0))
		// apply message
		log.Trace("init contract", "block hash", header.Hash(), "contract", c)
		err = s.applyTransaction(msg, state, header, chainContext, txs, alllogs, receipts, receivedTxs, usedGas, mining)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Senatus) getTopValidator(chain consensus.ChainHeaderReader, header *types.Header) ([]common.Address, error) {
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return []common.Address{}, consensus.ErrUnknownAncestor
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	method := "getValidatorCandidate"

	data, err := s.validatorABI.Pack(method)
	if err != nil {
		log.Error("can't pack data for getValidatorCandidate", "error", err)
		return []common.Address{}, err
	}

	blockNr := rpc.BlockNumberOrHash{
		BlockNumber:      nil,
		BlockHash:        &header.ParentHash,
		RequireCanonical: false,
	}

	msgData := (hexutil.Bytes)(data)
	gas := (hexutil.Uint64)(uint64(math.MaxUint64 / 2))
	result, err := s.ethAPI.Call(ctx, ethapi.CallArgs{
		Gas:  &gas,
		To:   &validatorContractAddr,
		Data: &msgData,
	}, blockNr, nil)
	if err != nil {
		return nil, err
	}
	var (
		candidates     = new([]common.Address)
		stakingAmounts = new([]*big.Int)
		candidateSize  = new(*big.Int)
	)

	out := &[]interface{}{
		candidates,
		stakingAmounts,
		candidateSize,
	}
	err = s.validatorABI.UnpackIntoInterface(out, method, result)
	if err != nil {
		log.Error("getTopValidator", "error", err)
		return nil, err
	}
	candidatieSizeInt := int((*candidateSize).Int64())
	topValidatorNum := maxValidatorNum
	if candidatieSizeInt < topValidatorNum {
		topValidatorNum = candidatieSizeInt
	}
	validators := make([]common.Address, 0, topValidatorNum)
	for i := 0; i < topValidatorNum; i++ {
		validators = append(validators, (*candidates)[i])
	}
	sort.Sort(validatorsAscending(validators))
	shuffleIndex := int(header.Number.Uint64() % uint64(topValidatorNum))
	validators[shuffleIndex], validators[topValidatorNum-1] = validators[topValidatorNum-1], validators[shuffleIndex]
	return validators, nil
}

func (s *Senatus) slash(chain consensus.ChainHeaderReader, state *state.StateDB, header *types.Header, chainContext core.ChainContext,
	txs *[]*types.Transaction, allLogs *[]*types.Log, receipts *[]*types.Receipt, receivedTxs *[]*types.Transaction, usedGas *uint64, mining bool) error {
	number := header.Number.Uint64()
	snap, err := s.snapshot(chain, number-1, header.ParentHash, nil)
	if err != nil {
		return err
	}

	validators := snap.validators()
	outTurnValidator := validators[number%uint64(len(validators))]
	// check sigend recently or not
	signedRecently := false
	for _, recent := range snap.Recents {
		if recent == outTurnValidator {
			signedRecently = true
			break
		}
	}

	if signedRecently {
		return nil
	}

	method := "slash"
	data, err := s.slashABI.Pack(method, outTurnValidator)
	if err != nil {
		log.Error("unable to pack tx for slash", "error", err, "spoil validator address", outTurnValidator)
		return err
	}
	msg := s.wrapSystemContractMessage(header.Coinbase, slashContractAddr, data, big.NewInt(0))
	return s.applyTransaction(msg, state, header, chainContext, txs, allLogs, receipts, receivedTxs, usedGas, mining)
}

func (s *Senatus) distributeReward(chain consensus.ChainHeaderReader,
	state *state.StateDB, header *types.Header, chainContext core.ChainContext,
	txs *[]*types.Transaction, allLogs *[]*types.Log, receipts *[]*types.Receipt, receivedTxs *[]*types.Transaction, usedGas *uint64, mining bool) error {
	reward := state.GetBalance(consensus.FeeRecoder)
	if reward.Cmp(common.Big0) <= 0 {
		return nil
	}

	state.SetBalance(consensus.FeeRecoder, big.NewInt(0))
	state.AddBalance(header.Coinbase, reward)
	method := "distributeBlockReward"
	data, err := s.validatorABI.Pack(method)
	if err != nil {
		log.Error("unable to pack tx for distributing reward", "error", err)
		return err
	}
	msg := s.wrapSystemContractMessage(header.Coinbase, validatorContractAddr, data, reward)
	return s.applyTransaction(msg, state, header, chainContext, txs, allLogs, receipts, receivedTxs, usedGas, mining)
}

func (s *Senatus) collectBlockReward(chain consensus.ChainHeaderReader,
	state *state.StateDB, header *types.Header, chainContext core.ChainContext) error {
	blockReward := CalcBlockReward(s.chainConfig, header.Number)
	if blockReward.Cmp(common.Big0) <= 0 {
		return nil
	}
	state.AddBalance(consensus.FeeRecoder, blockReward)
	return nil
}

func (s *Senatus) epochProcess(chain consensus.ChainHeaderReader,
	state *state.StateDB, header *types.Header, chainContext core.ChainContext,
	txs *[]*types.Transaction, allLogs *[]*types.Log, receipts *[]*types.Receipt, receivedTxs *[]*types.Transaction, usedGas *uint64, mining bool) ([]common.Address, error) {

	if err := s.updateActivatedValidators(chain,
		state, header, chainContext,
		txs, allLogs, receipts, receivedTxs, usedGas, mining); err != nil {
		return []common.Address{}, err
	}

	newSortedValidators, err := s.getTopValidator(chain, header)
	if err != nil {
		return []common.Address{}, err
	}

	if err := s.decreaseMissedBlocksCounter(chain,
		state, header, chainContext,
		txs, allLogs, receipts, receivedTxs, usedGas, mining); err != nil {
		return []common.Address{}, err
	}

	return newSortedValidators, nil
}

func (s *Senatus) updateActivatedValidators(chain consensus.ChainHeaderReader,
	state *state.StateDB, header *types.Header, chainContext core.ChainContext,
	txs *[]*types.Transaction, allLogs *[]*types.Log, receipts *[]*types.Receipt, receivedTxs *[]*types.Transaction, usedGas *uint64, mining bool) error {
	method := "updateActivatedValidators"
	data, err := s.validatorABI.Pack(method)
	if err != nil {
		log.Error("unable to pack tx for updateActivatedValidators", "error", err)
		return err
	}
	msg := s.wrapSystemContractMessage(header.Coinbase, validatorContractAddr, data, big.NewInt(0))
	return s.applyTransaction(msg, state, header, chainContext, txs, allLogs, receipts, receivedTxs, usedGas, mining)
}

func (s *Senatus) decreaseMissedBlocksCounter(chain consensus.ChainHeaderReader,
	state *state.StateDB, header *types.Header, chainContext core.ChainContext,
	txs *[]*types.Transaction, allLogs *[]*types.Log, receipts *[]*types.Receipt, receivedTxs *[]*types.Transaction, usedGas *uint64, mining bool) error {
	method := "decreaseMissedBlocksCounter"
	data, err := s.slashABI.Pack(method)
	if err != nil {
		log.Error("unable to pack tx for decreaseMissedBlocksCounter", "error", err)
		return err
	}
	msg := s.wrapSystemContractMessage(header.Coinbase, slashContractAddr, data, big.NewInt(0))
	return s.applyTransaction(msg, state, header, chainContext, txs, allLogs, receipts, receivedTxs, usedGas, mining)
}

func (s *Senatus) verifyTxsGasPrice(txs []*types.Transaction, header *types.Header) error {
	if len(txs) <= 0 {
		return nil
	}

	for _, tx := range txs {
		if ok, _ := s.IsSystemTransaction(tx, header); ok {
			continue
		}

		if tx.GasPrice().Cmp(params.MinimalGasPrice) < 0 {
			return errInvalidGasPrice
		}
	}
	return nil
}

func (s *Senatus) applyTransaction(
	msg callmsg,
	state *state.StateDB,
	header *types.Header,
	chainContext core.ChainContext,
	txs *[]*types.Transaction, allLogs *[]*types.Log, receipts *[]*types.Receipt,
	receivedTxs *[]*types.Transaction, usedGas *uint64, mining bool,
) (err error) {
	nonce := state.GetNonce(msg.From())
	expectedTx := types.NewTransaction(nonce, *msg.To(), msg.Value(), msg.Gas(), msg.GasPrice(), msg.Data())
	expectedHash := s.signer.Hash(expectedTx)
	if msg.From() == s.validator && mining {
		expectedTx, err = s.signTxFn(accounts.Account{Address: msg.From()}, expectedTx, s.chainConfig.ChainID)
		if err != nil {
			return err
		}
	} else {
		if receivedTxs == nil || len(*receivedTxs) == 0 || (*receivedTxs)[0] == nil {
			return errors.New("supposed to get a actual transaction, but get none")
		}
		actualTx := (*receivedTxs)[0]
		if !bytes.Equal(s.signer.Hash(actualTx).Bytes(), expectedHash.Bytes()) {
			return fmt.Errorf("expected tx hash %v, get %v", expectedHash.String(), actualTx.Hash().String())
		}
		expectedTx = actualTx
		// move to next
		*receivedTxs = (*receivedTxs)[1:]
	}
	state.Prepare(expectedTx.Hash(), common.Hash{}, len(*txs))
	gasUsed, err := applyMessage(msg, state, header, s.chainConfig, chainContext)
	if err != nil {
		return err
	}
	*txs = append(*txs, expectedTx)
	var root []byte
	if s.chainConfig.IsByzantium(header.Number) {
		state.Finalise(true)
	} else {
		root = state.IntermediateRoot(s.chainConfig.IsEIP158(header.Number)).Bytes()
	}
	receipt := types.NewReceipt(root, false, *usedGas)
	receipt.TxHash = expectedTx.Hash()
	receipt.GasUsed = gasUsed

	// Set the receipt logs and create a bloom for filtering
	receipt.Logs = state.GetLogs(expectedTx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	receipt.BlockHash = state.BlockHash()
	receipt.BlockNumber = header.Number
	receipt.TransactionIndex = uint(state.TxIndex())
	*receipts = append(*receipts, receipt)
	if !mining {
		*allLogs = append(*allLogs, receipt.Logs...)
	}
	state.SetNonce(msg.From(), nonce+1)
	return nil
}

func (s *Senatus) IsSystemTransaction(tx *types.Transaction, header *types.Header) (bool, error) {
	if tx.To() == nil {
		return false, nil
	}
	sender, err := types.Sender(s.signer, tx)
	if err != nil {
		return false, errUnauthorizedTransaction
	}
	if sender == header.Coinbase && s.IsSystemContract(tx.To()) && tx.GasPrice().Cmp(big.NewInt(0)) == 0 {
		return true, nil
	}
	return false, nil
}

func (s *Senatus) IsSystemContract(to *common.Address) bool {
	if to == nil {
		return false
	}
	return *to == validatorContractAddr || *to == slashContractAddr
}

// chain context
type chainContext struct {
	Chain   consensus.ChainHeaderReader
	senatus consensus.Engine
}

func (c chainContext) Engine() consensus.Engine {
	return c.senatus
}

func (c chainContext) GetHeader(hash common.Hash, number uint64) *types.Header {
	return c.Chain.GetHeader(hash, number)
}

// callmsg implements core.Message to allow passing it as a transaction simulator.
type callmsg struct {
	ethereum.CallMsg
}

func (m callmsg) From() common.Address { return m.CallMsg.From }
func (m callmsg) Nonce() uint64        { return 0 }
func (m callmsg) CheckNonce() bool     { return false }
func (m callmsg) To() *common.Address  { return m.CallMsg.To }
func (m callmsg) GasPrice() *big.Int   { return m.CallMsg.GasPrice }
func (m callmsg) Gas() uint64          { return m.CallMsg.Gas }
func (m callmsg) Value() *big.Int      { return m.CallMsg.Value }
func (m callmsg) Data() []byte         { return m.CallMsg.Data }

func (s *Senatus) wrapSystemContractMessage(from, toAddress common.Address, data []byte, value *big.Int) callmsg {
	return callmsg{
		ethereum.CallMsg{
			From:     from,
			Gas:      math.MaxUint64 / 2,
			GasPrice: big.NewInt(0),
			Value:    value,
			To:       &toAddress,
			Data:     data,
		},
	}
}

// apply message
func applyMessage(
	msg callmsg,
	state *state.StateDB,
	header *types.Header,
	chainConfig *params.ChainConfig,
	chainContext core.ChainContext,
) (uint64, error) {
	// Create a new context to be used in the EVM environment
	context := core.NewEVMBlockContext(header, chainContext, &header.Coinbase)
	//creates a new transaction context for a single transaction.
	txContext := core.NewEVMTxContext(msg)
	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms.
	vmenv := vm.NewEVM(context, txContext, state, chainConfig, vm.Config{})
	// Apply the transaction to the current state (included in the env)
	ret, returnGas, err := vmenv.Call(
		vm.AccountRef(msg.From()),
		*msg.To(),
		msg.Data(),
		msg.Gas(),
		msg.Value(),
	)
	if err != nil {
		log.Error("apply message failed", "msg", string(ret), "err", err)
	}
	return msg.Gas() - returnGas, err
}

type validatorStaking struct {
	address       common.Address
	stakingAmount uint64
}

type validatorStakings []validatorStaking

func (vals validatorStakings) Len() int {
	return len(vals)
}

func (vals validatorStakings) Less(i, j int) bool {
	return vals[i].stakingAmount > vals[j].stakingAmount
}

func (vals validatorStakings) Swap(i, j int) {
	vals[i], vals[j] = vals[j], vals[i]
}

// ValidatorContratAddress returns the validator contract address
func ValidatorContratAddress() common.Address {
	return validatorContractAddr
}

// SlashContractAddress returns the slash contract address
func SlashContractAddress() common.Address {
	return slashContractAddr
}

// ValidatorContractABI returns the validator contract abi
func ValidatorContractABI() string {
	return validatorABI
}

//  SlashContractABI return the slash contract abi
func SlashContractABI() string {
	return slashABI
}
