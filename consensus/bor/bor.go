package bor

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"

	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	ethereum "github.com/maticnetwork/bor"
	"github.com/maticnetwork/bor/accounts"
	"github.com/maticnetwork/bor/accounts/abi"
	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/common/hexutil"
	"github.com/maticnetwork/bor/consensus"
	"github.com/maticnetwork/bor/consensus/misc"
	"github.com/maticnetwork/bor/core"
	"github.com/maticnetwork/bor/core/state"
	"github.com/maticnetwork/bor/core/types"
	"github.com/maticnetwork/bor/core/vm"
	"github.com/maticnetwork/bor/crypto"
	"github.com/maticnetwork/bor/ethdb"
	"github.com/maticnetwork/bor/event"
	"github.com/maticnetwork/bor/internal/ethapi"
	"github.com/maticnetwork/bor/log"
	"github.com/maticnetwork/bor/params"
	"github.com/maticnetwork/bor/rlp"
	"github.com/maticnetwork/bor/rpc"

	lru "github.com/hashicorp/golang-lru"
	"golang.org/x/crypto/sha3"
)

const validatorsetABI = `[{"constant":true,"inputs":[{"name":"span","type":"uint256"}],"name":"getSpan","outputs":[{"name":"number","type":"uint256"},{"name":"startBlock","type":"uint256"},{"name":"endBlock","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"number","type":"uint256"}],"name":"getBorValidators","outputs":[{"name":"","type":"address[]"},{"name":"","type":"uint256[]"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"span","type":"uint256"},{"name":"signer","type":"address"}],"name":"isProducer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"newSpan","type":"uint256"},{"name":"startBlock","type":"uint256"},{"name":"endBlock","type":"uint256"},{"name":"validatorBytes","type":"bytes"},{"name":"producerBytes","type":"bytes"}],"name":"commitSpan","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"span","type":"uint256"},{"name":"signer","type":"address"}],"name":"isValidator","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[],"name":"proposeSpan","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"currentSpanNumber","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getNextSpan","outputs":[{"name":"number","type":"uint256"},{"name":"startBlock","type":"uint256"},{"name":"endBlock","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getInitialValidators","outputs":[{"name":"","type":"address[]"},{"name":"","type":"uint256[]"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"spanProposalPending","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getCurrentSpan","outputs":[{"name":"number","type":"uint256"},{"name":"startBlock","type":"uint256"},{"name":"endBlock","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"number","type":"uint256"}],"name":"getSpanByBlock","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getValidators","outputs":[{"name":"","type":"address[]"},{"name":"","type":"uint256[]"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"vote","type":"bytes"},{"name":"sigs","type":"bytes"},{"name":"txBytes","type":"bytes"},{"name":"proof","type":"bytes"}],"name":"validateValidatorSet","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
const stateReceiverABI = `[{"constant":true,"inputs":[{"name":"","type":"uint256"}],"name":"states","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"recordBytes","type":"bytes"}],"name":"commitState","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"getPendingStates","outputs":[{"name":"","type":"uint256[]"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"SYSTEM_ADDRESS","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"validatorSet","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"vote","type":"bytes"},{"name":"sigs","type":"bytes"},{"name":"txBytes","type":"bytes"},{"name":"proof","type":"bytes"}],"name":"validateValidatorSet","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"isValidatorSetContract","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"stateId","type":"uint256"}],"name":"proposeState","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"signer","type":"address"}],"name":"isProducer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"signer","type":"address"}],"name":"isValidator","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"}]`

const (
	checkpointInterval = 1024 // Number of blocks after which to save the vote snapshot to the database
	inmemorySnapshots  = 128  // Number of recent vote snapshots to keep in memory
	inmemorySignatures = 4096 // Number of recent block signatures to keep in memory

	wiggleTime = 5000 * time.Millisecond // Random delay (per signer) to allow concurrent signers
)

// Bor protocol constants.
var (
	defaultSprintLength = uint64(64) // Default number of blocks after which to checkpoint and reset the pending votes

	extraVanity = 32 // Fixed number of extra-data prefix bytes reserved for signer vanity
	extraSeal   = 65 // Fixed number of extra-data suffix bytes reserved for signer seal

	uncleHash = types.CalcUncleHash(nil) // Always Keccak256(RLP([])) as uncles are meaningless outside of PoW.

	diffInTurn = big.NewInt(2) // Block difficulty for in-turn signatures
	diffNoTurn = big.NewInt(1) // Block difficulty for out-of-turn signatures

	validatorHeaderBytesLength = common.AddressLength + 20 // address + power
	systemAddress              = common.HexToAddress("0xffffFFFfFFffffffffffffffFfFFFfffFFFfFFfE")
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	// errUnknownBlock is returned when the list of signers is requested for a block
	// that is not part of the local blockchain.
	errUnknownBlock = errors.New("unknown block")

	// errInvalidCheckpointBeneficiary is returned if a checkpoint/epoch transition
	// block has a beneficiary set to non-zeroes.
	errInvalidCheckpointBeneficiary = errors.New("beneficiary in checkpoint block non-zero")

	// errInvalidVote is returned if a nonce value is something else that the two
	// allowed constants of 0x00..0 or 0xff..f.
	errInvalidVote = errors.New("vote nonce not 0x00..0 or 0xff..f")

	// errInvalidCheckpointVote is returned if a checkpoint/epoch transition block
	// has a vote nonce set to non-zeroes.
	errInvalidCheckpointVote = errors.New("vote nonce in checkpoint block non-zero")

	// errMissingVanity is returned if a block's extra-data section is shorter than
	// 32 bytes, which is required to store the signer vanity.
	errMissingVanity = errors.New("extra-data 32 byte vanity prefix missing")

	// errMissingSignature is returned if a block's extra-data section doesn't seem
	// to contain a 65 byte secp256k1 signature.
	errMissingSignature = errors.New("extra-data 65 byte signature suffix missing")

	// errExtraValidators is returned if non-sprint-end block contain validator data in
	// their extra-data fields.
	errExtraValidators = errors.New("non-sprint-end block contains extra validator list")

	// errInvalidSpanValidators is returned if a block contains an
	// invalid list of validators (i.e. non divisible by 40 bytes).
	errInvalidSpanValidators = errors.New("invalid validator list on sprint end block")

	// errMismatchingSprintValidators is returned if a sprint block contains a
	// list of validators different than the one the local node calculated.
	errMismatchingSprintValidators = errors.New("mismatching validator list on sprint block")

	// errInvalidMixDigest is returned if a block's mix digest is non-zero.
	errInvalidMixDigest = errors.New("non-zero mix digest")

	// errInvalidUncleHash is returned if a block contains an non-empty uncle list.
	errInvalidUncleHash = errors.New("non empty uncle hash")

	// errInvalidDifficulty is returned if the difficulty of a block neither 1 or 2.
	errInvalidDifficulty = errors.New("invalid difficulty")

	// errWrongDifficulty is returned if the difficulty of a block doesn't match the
	// turn of the signer.
	errWrongDifficulty = errors.New("wrong difficulty")

	// ErrInvalidTimestamp is returned if the timestamp of a block is lower than
	// the previous block's timestamp + the minimum block period.
	ErrInvalidTimestamp = errors.New("invalid timestamp")

	// errOutOfRangeChain is returned if an authorization list is attempted to
	// be modified via out-of-range or non-contiguous headers.
	errOutOfRangeChain = errors.New("out of range or non-contiguous chain")

	// errUnauthorizedSigner is returned if a header is signed by a non-authorized entity.
	errUnauthorizedSigner = errors.New("unauthorized signer")

	// errRecentlySigned is returned if a header is signed by an authorized entity
	// that already signed a header recently, thus is temporarily not allowed to.
	errRecentlySigned = errors.New("recently signed")
)

// SignerFn is a signer callback function to request a header to be signed by a
// backing account.
type SignerFn func(accounts.Account, string, []byte) ([]byte, error)

// ecrecover extracts the Ethereum account address from a signed header.
func ecrecover(header *types.Header, sigcache *lru.ARCCache) (common.Address, error) {
	// If the signature's already cached, return that
	hash := header.Hash()
	if address, known := sigcache.Get(hash); known {
		return address.(common.Address), nil
	}
	// Retrieve the signature from the header extra-data
	if len(header.Extra) < extraSeal {
		return common.Address{}, errMissingSignature
	}
	signature := header.Extra[len(header.Extra)-extraSeal:]

	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(SealHash(header).Bytes(), signature)
	if err != nil {
		return common.Address{}, err
	}
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])

	sigcache.Add(hash, signer)
	return signer, nil
}

// SealHash returns the hash of a block prior to it being sealed.
func SealHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()
	encodeSigHeader(hasher, header)
	hasher.Sum(hash[:0])
	return hash
}

func encodeSigHeader(w io.Writer, header *types.Header) {
	err := rlp.Encode(w, []interface{}{
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
		header.Extra[:len(header.Extra)-65], // Yes, this will panic if extra is too short
		header.MixDigest,
		header.Nonce,
	})
	if err != nil {
		panic("can't encode: " + err.Error())
	}
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have based on the previous blocks in the chain and the
// current signer.
func CalcDifficulty(snap *Snapshot, signer common.Address, sprint uint64) *big.Int {
	return big.NewInt(0).SetUint64(snap.inturn(snap.Number+1, signer, sprint))
}

// CalcProducerDelay is the block delay algorithm based on block time and period / producerDelay values in genesis
func CalcProducerDelay(number uint64, period uint64, sprint uint64, producerDelay uint64) uint64 {
	// When the block is the first block of the sprint, it is expected to be delayed by `producerDelay`.
	// That is to allow time for block propagation in the last sprint
	if number%sprint == 0 {
		return producerDelay
	}
	return period
}

// BorRLP returns the rlp bytes which needs to be signed for the bor
// sealing. The RLP to sign consists of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// Note, the method requires the extra data to be at least 65 bytes, otherwise it
// panics. This is done to avoid accidentally using both forms (signature present
// or not), which could be abused to produce different hashes for the same header.
func BorRLP(header *types.Header) []byte {
	b := new(bytes.Buffer)
	encodeSigHeader(b, header)
	return b.Bytes()
}

// Bor is the matic-bor consensus engine
type Bor struct {
	chainConfig *params.ChainConfig // Chain config
	config      *params.BorConfig   // Consensus engine configuration parameters for bor consensus
	db          ethdb.Database      // Database to store and retrieve snapshot checkpoints

	recents    *lru.ARCCache // Snapshots for recent block to speed up reorgs
	signatures *lru.ARCCache // Signatures of recent blocks to speed up mining

	signer common.Address // Ethereum address of the signing key
	signFn SignerFn       // Signer function to authorize hashes with
	lock   sync.RWMutex   // Protects the signer fields

	ethAPI           *ethapi.PublicBlockChainAPI
	validatorSetABI  abi.ABI
	stateReceiverABI abi.ABI
	HeimdallClient   IHeimdallClient

	stateDataFeed event.Feed
	scope         event.SubscriptionScope
	// The fields below are for testing only
	fakeDiff bool // Skip difficulty verifications
}

// New creates a Matic Bor consensus engine.
func New(
	chainConfig *params.ChainConfig,
	db ethdb.Database,
	ethAPI *ethapi.PublicBlockChainAPI,
	heimdallURL string,
) *Bor {
	// get bor config
	borConfig := chainConfig.Bor

	// Set any missing consensus parameters to their defaults
	if borConfig != nil && borConfig.Sprint == 0 {
		borConfig.Sprint = defaultSprintLength
	}

	// Allocate the snapshot caches and create the engine
	recents, _ := lru.NewARC(inmemorySnapshots)
	signatures, _ := lru.NewARC(inmemorySignatures)
	vABI, _ := abi.JSON(strings.NewReader(validatorsetABI))
	sABI, _ := abi.JSON(strings.NewReader(stateReceiverABI))
	heimdallClient, _ := NewHeimdallClient(heimdallURL)

	c := &Bor{
		chainConfig:      chainConfig,
		config:           borConfig,
		db:               db,
		ethAPI:           ethAPI,
		recents:          recents,
		signatures:       signatures,
		validatorSetABI:  vABI,
		stateReceiverABI: sABI,
		HeimdallClient:   heimdallClient,
	}

	return c
}

// Author implements consensus.Engine, returning the Ethereum address recovered
// from the signature in the header's extra-data section.
func (c *Bor) Author(header *types.Header) (common.Address, error) {
	return ecrecover(header, c.signatures)
}

// VerifyHeader checks whether a header conforms to the consensus rules.
func (c *Bor) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	return c.verifyHeader(chain, header, nil)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers. The
// method returns a quit channel to abort the operations and a results channel to
// retrieve the async verifications (the order is that of the input slice).
func (c *Bor) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for i, header := range headers {
			err := c.verifyHeader(chain, header, headers[:i])

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
func (c *Bor) verifyHeader(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	if header.Number == nil {
		return errUnknownBlock
	}
	number := header.Number.Uint64()

	var parent *types.Header
	if len(parents) > 0 { // if parents is nil, len(parents) is zero
		parent = parents[len(parents)-1]
	} else if number > 0 {
		parent = chain.GetHeader(header.ParentHash, number-1)
	}

	if parent != nil && header.Time < parent.Time+CalcProducerDelay(number, c.config.Period, c.config.Sprint, c.config.ProducerDelay) {
		return consensus.ErrBlockTooSoon
	}

	// Don't waste time checking blocks from the future
	if header.Time > uint64(time.Now().Unix()) {
		return consensus.ErrFutureBlock
	}

	if err := validateHeaderExtraField(header.Extra); err != nil {
		return err
	}

	// check extr adata
	isSprintEnd := (number+1)%c.config.Sprint == 0

	// Ensure that the extra-data contains a signer list on checkpoint, but none otherwise
	signersBytes := len(header.Extra) - extraVanity - extraSeal
	if !isSprintEnd && signersBytes != 0 {
		return errExtraValidators
	}
	if isSprintEnd && signersBytes%validatorHeaderBytesLength != 0 {
		return errInvalidSpanValidators
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
	if number > 0 {
		if header.Difficulty == nil {
			return errInvalidDifficulty
		}
	}
	// If all checks passed, validate any special fields for hard forks
	if err := misc.VerifyForkHashes(chain.Config(), header, false); err != nil {
		return err
	}
	// All basic checks passed, verify cascading fields
	return c.verifyCascadingFields(chain, header, parents)
}

// validateHeaderExtraField validates that the extra-data contains both the vanity and signature.
// header.Extra = header.Vanity + header.ProducerBytes (optional) + header.Seal
func validateHeaderExtraField(extraBytes []byte) error {
	if len(extraBytes) < extraVanity {
		return errMissingVanity
	}
	if len(extraBytes) < extraVanity+extraSeal {
		return errMissingSignature
	}
	return nil
}

// verifyCascadingFields verifies all the header fields that are not standalone,
// rather depend on a batch of previous headers. The caller may optionally pass
// in a batch of parents (ascending order) to avoid looking those up from the
// database. This is useful for concurrently verifying a batch of new headers.
func (c *Bor) verifyCascadingFields(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	// The genesis block is the always valid dead-end
	number := header.Number.Uint64()
	if number == 0 {
		return nil
	}

	// Ensure that the block's timestamp isn't too close to it's parent
	var parent *types.Header
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeader(header.ParentHash, number-1)
	}

	if parent == nil || parent.Number.Uint64() != number-1 || parent.Hash() != header.ParentHash {
		return consensus.ErrUnknownAncestor
	}

	if parent.Time+c.config.Period > header.Time {
		return ErrInvalidTimestamp
	}

	// Retrieve the snapshot needed to verify this header and cache it
	snap, err := c.snapshot(chain, number-1, header.ParentHash, parents)
	if err != nil {
		return err
	}

	isSprintEnd := (number+1)%c.config.Sprint == 0
	// verify the validator list in the last sprint block
	if isSprintEnd {
		validatorsBytes := make([]byte, len(snap.ValidatorSet.Validators)*validatorHeaderBytesLength)

		currentValidators := snap.ValidatorSet.Copy().Validators
		// sort validator by address
		sort.Sort(ValidatorsByAddress(currentValidators))
		for i, validator := range currentValidators {
			copy(validatorsBytes[i*validatorHeaderBytesLength:], validator.HeaderBytes())
		}
		// len(header.Extra) >= extraVanity+extraSeal has already been validated in validateHeaderExtraField, so this won't result in a panic
		if !bytes.Equal(header.Extra[extraVanity:len(header.Extra)-extraSeal], validatorsBytes) {
			return errMismatchingSprintValidators
		}
	}

	// All basic checks passed, verify the seal and return
	return c.verifySeal(chain, header, parents)
}

// snapshot retrieves the authorization snapshot at a given point in time.
func (c *Bor) snapshot(chain consensus.ChainReader, number uint64, hash common.Hash, parents []*types.Header) (*Snapshot, error) {
	// Search for a snapshot in memory or on disk for checkpoints
	var (
		headers []*types.Header
		snap    *Snapshot
	)

	for snap == nil {
		// If an in-memory snapshot was found, use that
		if s, ok := c.recents.Get(hash); ok {
			snap = s.(*Snapshot)
			break
		}

		// If an on-disk checkpoint snapshot can be found, use that
		if number%checkpointInterval == 0 {
			if s, err := loadSnapshot(c.config, c.signatures, c.db, hash, c.ethAPI); err == nil {
				log.Trace("Loaded snapshot from disk", "number", number, "hash", hash)
				snap = s
				break
			}
		}

		// If we're at the genesis, snapshot the initial state. Alternatively if we're
		// at a checkpoint block without a parent (light client CHT), or we have piled
		// up more headers than allowed to be reorged (chain reinit from a freezer),
		// consider the checkpoint trusted and snapshot it.
		// TODO fix this
		if number == 0 {
			checkpoint := chain.GetHeaderByNumber(number)
			if checkpoint != nil {
				// get checkpoint data
				hash := checkpoint.Hash()

				// get validators and current span
				validators, err := c.GetCurrentValidators(number, number+1)
				if err != nil {
					return nil, err
				}

				// new snap shot
				snap = newSnapshot(c.config, c.signatures, number, hash, validators, c.ethAPI)
				if err := snap.store(c.db); err != nil {
					return nil, err
				}
				log.Info("Stored checkpoint snapshot to disk", "number", number, "hash", hash)
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

	// check if snapshot is nil
	if snap == nil {
		return nil, fmt.Errorf("Unknown error while retrieving snapshot at block number %v", number)
	}

	// Previous snapshot found, apply any pending headers on top of it
	for i := 0; i < len(headers)/2; i++ {
		headers[i], headers[len(headers)-1-i] = headers[len(headers)-1-i], headers[i]
	}

	snap, err := snap.apply(headers)
	if err != nil {
		return nil, err
	}
	c.recents.Add(snap.Hash, snap)

	// If we've generated a new checkpoint snapshot, save to disk
	if snap.Number%checkpointInterval == 0 && len(headers) > 0 {
		if err = snap.store(c.db); err != nil {
			return nil, err
		}
		log.Trace("Stored snapshot to disk", "number", snap.Number, "hash", snap.Hash)
	}
	return snap, err
}

// VerifyUncles implements consensus.Engine, always returning an error for any
// uncles as this consensus mechanism doesn't permit uncles.
func (c *Bor) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errors.New("uncles not allowed")
	}
	return nil
}

// VerifySeal implements consensus.Engine, checking whether the signature contained
// in the header satisfies the consensus protocol requirements.
func (c *Bor) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	return c.verifySeal(chain, header, nil)
}

// verifySeal checks whether the signature contained in the header satisfies the
// consensus protocol requirements. The method accepts an optional list of parent
// headers that aren't yet part of the local blockchain to generate the snapshots
// from.
func (c *Bor) verifySeal(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	// Verifying the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}
	// Retrieve the snapshot needed to verify this header and cache it
	snap, err := c.snapshot(chain, number-1, header.ParentHash, parents)
	if err != nil {
		return err
	}

	// Resolve the authorization key and check against signers
	signer, err := ecrecover(header, c.signatures)
	if err != nil {
		return err
	}
	if !snap.ValidatorSet.HasAddress(signer.Bytes()) {
		return errUnauthorizedSigner
	}

	if _, err = snap.GetSignerSuccessionNumber(signer); err != nil {
		return err
	}

	// Ensure that the difficulty corresponds to the turn-ness of the signer
	if !c.fakeDiff {
		difficulty := snap.inturn(header.Number.Uint64(), signer, c.config.Sprint)
		if header.Difficulty.Uint64() != difficulty {
			return errWrongDifficulty
		}
	}

	return nil
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (c *Bor) Prepare(chain consensus.ChainReader, header *types.Header) error {
	// If the block isn't a checkpoint, cast a random vote (good enough for now)
	header.Coinbase = common.Address{}
	header.Nonce = types.BlockNonce{}

	number := header.Number.Uint64()
	// Assemble the validator snapshot to check which votes make sense
	snap, err := c.snapshot(chain, number-1, header.ParentHash, nil)
	if err != nil {
		return err
	}

	// Set the correct difficulty
	header.Difficulty = CalcDifficulty(snap, c.signer, c.config.Sprint)

	// Ensure the extra data has all it's components
	if len(header.Extra) < extraVanity {
		header.Extra = append(header.Extra, bytes.Repeat([]byte{0x00}, extraVanity-len(header.Extra))...)
	}
	header.Extra = header.Extra[:extraVanity]

	// get validator set if number
	if (number+1)%c.config.Sprint == 0 {
		newValidators, err := c.GetCurrentValidators(snap.Number, number+1)
		if err != nil {
			return errors.New("unknown validators")
		}

		// sort validator by address
		sort.Sort(ValidatorsByAddress(newValidators))
		for _, validator := range newValidators {
			header.Extra = append(header.Extra, validator.HeaderBytes()...)
		}
	}

	// add extra seal space
	header.Extra = append(header.Extra, make([]byte, extraSeal)...)

	// Mix digest is reserved for now, set to empty
	header.MixDigest = common.Hash{}

	// Ensure the timestamp has the correct delay
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}

	header.Time = parent.Time + CalcProducerDelay(number, c.config.Period, c.config.Sprint, c.config.ProducerDelay)
	if header.Time < uint64(time.Now().Unix()) {
		header.Time = uint64(time.Now().Unix())
	}
	return nil
}

// Finalize implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given.
func (c *Bor) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header) {
	// commit span
	headerNumber := header.Number.Uint64()

	if headerNumber%c.config.Sprint == 0 {
		cx := chainContext{Chain: chain, Bor: c}
		// check and commit span
		if err := c.checkAndCommitSpan(state, header, cx); err != nil {
			log.Error("Error while committing span", "error", err)
			return
		}
		// commit statees
		if err := c.CommitStates(state, header, cx); err != nil {
			log.Error("Error while committing states", "error", err)
			return
		}
	}

	// No block rewards in PoA, so the state remains as is and uncles are dropped
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	header.UncleHash = types.CalcUncleHash(nil)
}

// FinalizeAndAssemble implements consensus.Engine, ensuring no uncles are set,
// nor block rewards given, and returns the final block.
func (c *Bor) FinalizeAndAssemble(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	// commit span
	if header.Number.Uint64()%c.config.Sprint == 0 {
		cx := chainContext{Chain: chain, Bor: c}

		// check and commit span
		err := c.checkAndCommitSpan(state, header, cx)
		if err != nil {
			log.Error("Error while committing span", "error", err)
			return nil, err
		}

		// commit statees
		if err := c.CommitStates(state, header, cx); err != nil {
			log.Error("Error while committing states", "error", err)
			// return nil, err
		}
	}

	// No block rewards in PoA, so the state remains as is and uncles are dropped
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	header.UncleHash = types.CalcUncleHash(nil)

	// Assemble and return the final block for sealing
	return types.NewBlock(header, txs, nil, receipts), nil
}

// Authorize injects a private key into the consensus engine to mint new blocks
// with.
func (c *Bor) Authorize(signer common.Address, signFn SignerFn) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.signer = signer
	c.signFn = signFn
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
func (c *Bor) Seal(chain consensus.ChainReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	header := block.Header()

	// Sealing the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}
	// For 0-period chains, refuse to seal empty blocks (no reward but would spin sealing)
	if c.config.Period == 0 && len(block.Transactions()) == 0 {
		log.Info("Sealing paused, waiting for transactions")
		return nil
	}
	// Don't hold the signer fields for the entire sealing procedure
	c.lock.RLock()
	signer, signFn := c.signer, c.signFn
	c.lock.RUnlock()

	snap, err := c.snapshot(chain, number-1, header.ParentHash, nil)
	if err != nil {
		return err
	}

	// Bail out if we're unauthorized to sign a block
	if !snap.ValidatorSet.HasAddress(signer.Bytes()) {
		return errUnauthorizedSigner
	}

	successionNumber, err := snap.GetSignerSuccessionNumber(signer)
	if err != nil {
		return err
	}

	// Sweet, the protocol permits us to sign the block, wait for our time
	delay := time.Unix(int64(header.Time), 0).Sub(time.Now()) // nolint: gosimple
	wiggle := time.Duration(2*c.config.Period) * time.Second * time.Duration(successionNumber)
	delay += wiggle

	log.Info("Out-of-turn signing requested", "wiggle", common.PrettyDuration(wiggle))
	log.Info("Sealing block with", "number", number, "delay", delay, "headerDifficulty", header.Difficulty, "signer", signer.Hex(), "proposer", snap.ValidatorSet.GetProposer().Address.Hex())

	// Sign all the things!
	sighash, err := signFn(accounts.Account{Address: signer}, accounts.MimetypeBor, BorRLP(header))
	if err != nil {
		return err
	}
	copy(header.Extra[len(header.Extra)-extraSeal:], sighash)

	// Wait until sealing is terminated or delay timeout.
	log.Trace("Waiting for slot to sign and propagate", "delay", common.PrettyDuration(delay))
	go func() {
		select {
		case <-stop:
			return
		case <-time.After(delay):
		}

		select {
		case results <- block.WithSeal(header):
		default:
			log.Warn("Sealing result is not read by miner", "sealhash", SealHash(header))
		}
	}()

	return nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have based on the previous blocks in the chain and the
// current signer.
func (c *Bor) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	snap, err := c.snapshot(chain, parent.Number.Uint64(), parent.Hash(), nil)
	if err != nil {
		return nil
	}
	return CalcDifficulty(snap, c.signer, c.config.Sprint)
}

// SealHash returns the hash of a block prior to it being sealed.
func (c *Bor) SealHash(header *types.Header) common.Hash {
	return SealHash(header)
}

// APIs implements consensus.Engine, returning the user facing RPC API to allow
// controlling the signer voting.
func (c *Bor) APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{{
		Namespace: "bor",
		Version:   "1.0",
		Service:   &API{chain: chain, bor: c},
		Public:    false,
	}}
}

// Close implements consensus.Engine. It's a noop for bor as there are no background threads.
func (c *Bor) Close() error {
	return nil
}

// Checks if "force" proposeSpan has been set
func (c *Bor) isSpanPending(snapshotNumber uint64) (bool, error) {
	blockNr := rpc.BlockNumber(snapshotNumber)
	method := "spanProposalPending"

	// get packed data
	data, err := c.validatorSetABI.Pack(method)
	if err != nil {
		log.Error("Unable to pack tx for spanProposalPending", "error", err)
		return false, err
	}

	msgData := (hexutil.Bytes)(data)
	toAddress := common.HexToAddress(c.config.ValidatorContract)
	gas := (hexutil.Uint64)(uint64(math.MaxUint64 / 2))
	result, err := c.ethAPI.Call(context.Background(), ethapi.CallArgs{
		Gas:  &gas,
		To:   &toAddress,
		Data: &msgData,
	}, blockNr)
	if err != nil {
		return false, err
	}

	var ret0 = new(bool)
	if err := c.validatorSetABI.Unpack(ret0, method, result); err != nil {
		return false, err
	}

	return *ret0, nil
}

// GetCurrentSpan get current span from contract
func (c *Bor) GetCurrentSpan(snapshotNumber uint64) (*Span, error) {
	// block
	blockNr := rpc.BlockNumber(snapshotNumber)

	// method
	method := "getCurrentSpan"

	data, err := c.validatorSetABI.Pack(method)
	if err != nil {
		log.Error("Unable to pack tx for getCurrentSpan", "error", err)
		return nil, err
	}

	msgData := (hexutil.Bytes)(data)
	toAddress := common.HexToAddress(c.config.ValidatorContract)
	gas := (hexutil.Uint64)(uint64(math.MaxUint64 / 2))
	result, err := c.ethAPI.Call(context.Background(), ethapi.CallArgs{
		Gas:  &gas,
		To:   &toAddress,
		Data: &msgData,
	}, blockNr)
	if err != nil {
		return nil, err
	}

	// span result
	ret := new(struct {
		Number     *big.Int
		StartBlock *big.Int
		EndBlock   *big.Int
	})
	if err := c.validatorSetABI.Unpack(ret, method, result); err != nil {
		return nil, err
	}

	// create new span
	span := Span{
		ID:         ret.Number.Uint64(),
		StartBlock: ret.StartBlock.Uint64(),
		EndBlock:   ret.EndBlock.Uint64(),
	}

	return &span, nil
}

// GetCurrentValidators get current validators
func (c *Bor) GetCurrentValidators(snapshotNumber uint64, blockNumber uint64) ([]*Validator, error) {
	// block
	blockNr := rpc.BlockNumber(snapshotNumber)

	// method
	method := "getBorValidators"

	data, err := c.validatorSetABI.Pack(method, big.NewInt(0).SetUint64(blockNumber))
	if err != nil {
		log.Error("Unable to pack tx for getValidator", "error", err)
		return nil, err
	}

	// call
	msgData := (hexutil.Bytes)(data)
	toAddress := common.HexToAddress(c.config.ValidatorContract)
	gas := (hexutil.Uint64)(uint64(math.MaxUint64 / 2))
	result, err := c.ethAPI.Call(context.Background(), ethapi.CallArgs{
		Gas:  &gas,
		To:   &toAddress,
		Data: &msgData,
	}, blockNr)
	if err != nil {
		panic(err)
		// return nil, err
	}

	var (
		ret0 = new([]common.Address)
		ret1 = new([]*big.Int)
	)
	out := &[]interface{}{
		ret0,
		ret1,
	}

	if err := c.validatorSetABI.Unpack(out, method, result); err != nil {
		return nil, err
	}

	valz := make([]*Validator, len(*ret0))
	for i, a := range *ret0 {
		valz[i] = &Validator{
			Address:     a,
			VotingPower: (*ret1)[i].Int64(),
		}
	}

	return valz, nil
}

func (c *Bor) checkAndCommitSpan(
	state *state.StateDB,
	header *types.Header,
	chain core.ChainContext,
) error {
	headerNumber := header.Number.Uint64()
	pending := false
	var span *Span = nil
	errors := make(chan error)
	go func() {
		var err error
		pending, err = c.isSpanPending(headerNumber - 1)
		errors <- err
	}()

	go func() {
		var err error
		span, err = c.GetCurrentSpan(headerNumber - 1)
		errors <- err
	}()

	var err error
	for i := 0; i < 2; i++ {
		err = <-errors
		if err != nil {
			close(errors)
			return err
		}
	}
	close(errors)

	// commit span if there is new span pending or span is ending or end block is not set
	if pending || c.needToCommitSpan(span, headerNumber) {
		err := c.fetchAndCommitSpan(span.ID+1, state, header, chain)
		return err
	}

	return nil
}

func (c *Bor) needToCommitSpan(span *Span, headerNumber uint64) bool {
	// if span is nil
	if span == nil {
		return false
	}

	// check span is not set initially
	if span.EndBlock == 0 {
		return true
	}

	// if current block is first block of last sprint in current span
	if span.EndBlock > c.config.Sprint && span.EndBlock-c.config.Sprint+1 == headerNumber {
		return true
	}

	return false
}

func (c *Bor) fetchAndCommitSpan(
	newSpanID uint64,
	state *state.StateDB,
	header *types.Header,
	chain core.ChainContext,
) error {
	response, err := c.HeimdallClient.FetchWithRetry("bor", "span", strconv.FormatUint(newSpanID, 10))

	if err != nil {
		return err
	}

	var heimdallSpan HeimdallSpan
	if err := json.Unmarshal(response.Result, &heimdallSpan); err != nil {
		return err
	}

	// check if chain id matches with heimdall span
	if heimdallSpan.ChainID != c.chainConfig.ChainID.String() {
		return fmt.Errorf(
			"Chain id proposed span, %s, and bor chain id, %s, doesn't match",
			heimdallSpan.ChainID,
			c.chainConfig.ChainID,
		)
	}

	// get validators bytes
	var validators []MinimalVal
	for _, val := range heimdallSpan.ValidatorSet.Validators {
		validators = append(validators, val.MinimalVal())
	}
	validatorBytes, err := rlp.EncodeToBytes(validators)
	if err != nil {
		return err
	}

	// get producers bytes
	var producers []MinimalVal
	for _, val := range heimdallSpan.SelectedProducers {
		producers = append(producers, val.MinimalVal())
	}
	producerBytes, err := rlp.EncodeToBytes(producers)
	if err != nil {
		return err
	}

	// method
	method := "commitSpan"
	log.Info("✅ Committing new span",
		"id", heimdallSpan.ID,
		"startBlock", heimdallSpan.StartBlock,
		"endBlock", heimdallSpan.EndBlock,
		"validatorBytes", hex.EncodeToString(validatorBytes),
		"producerBytes", hex.EncodeToString(producerBytes),
	)

	// get packed data
	data, err := c.validatorSetABI.Pack(method,
		big.NewInt(0).SetUint64(heimdallSpan.ID),
		big.NewInt(0).SetUint64(heimdallSpan.StartBlock),
		big.NewInt(0).SetUint64(heimdallSpan.EndBlock),
		validatorBytes,
		producerBytes,
	)
	if err != nil {
		log.Error("Unable to pack tx for commitSpan", "error", err)
		return err
	}

	// get system message
	msg := getSystemMessage(common.HexToAddress(c.config.ValidatorContract), data)

	// apply message
	return applyMessage(msg, state, header, c.chainConfig, chain)
}

// GetPendingStateProposals get pending state proposals
func (c *Bor) GetPendingStateProposals(snapshotNumber uint64) ([]*big.Int, error) {
	// block
	blockNr := rpc.BlockNumber(snapshotNumber)

	// method
	method := "getPendingStates"

	data, err := c.stateReceiverABI.Pack(method)
	if err != nil {
		log.Error("Unable to pack tx for getPendingStates", "error", err)
		return nil, err
	}

	msgData := (hexutil.Bytes)(data)
	toAddress := common.HexToAddress(c.config.StateReceiverContract)
	gas := (hexutil.Uint64)(uint64(math.MaxUint64 / 2))
	result, err := c.ethAPI.Call(context.Background(), ethapi.CallArgs{
		Gas:  &gas,
		To:   &toAddress,
		Data: &msgData,
	}, blockNr)
	if err != nil {
		return nil, err
	}

	var ret = new([]*big.Int)
	if err := c.stateReceiverABI.Unpack(ret, method, result); err != nil {
		return nil, err
	}

	return *ret, nil
}

// CommitStates commit states
func (c *Bor) CommitStates(
	state *state.StateDB,
	header *types.Header,
	chain core.ChainContext,
) error {
	// get pending state proposals
	stateIds, err := c.GetPendingStateProposals(header.Number.Uint64() - 1)
	if err != nil {
		return err
	}

	// state ids
	if len(stateIds) > 0 {
		log.Debug("Found new proposed states", "numberOfStates", len(stateIds))
	}

	method := "commitState"

	// itereate through state ids
	for _, stateID := range stateIds {
		// fetch from heimdall
		response, err := c.HeimdallClient.FetchWithRetry("clerk", "event-record", strconv.FormatUint(stateID.Uint64(), 10))
		if err != nil {
			return err
		}

		// get event record
		var eventRecord EventRecord
		if err := json.Unmarshal(response.Result, &eventRecord); err != nil {
			return err
		}

		// check if chain id matches with event record
		if eventRecord.ChainID != c.chainConfig.ChainID.String() {
			return fmt.Errorf(
				"Chain id proposed state in span, %s, and bor chain id, %s, doesn't match",
				eventRecord.ChainID,
				c.chainConfig.ChainID,
			)
		}

		log.Info("→ committing new state",
			"id", eventRecord.ID,
			"contract", eventRecord.Contract,
			"data", hex.EncodeToString(eventRecord.Data),
			"txHash", eventRecord.TxHash,
			"chainID", eventRecord.ChainID,
		)
		stateData := types.StateData{
			Did:      eventRecord.ID,
			Contract: eventRecord.Contract,
			Data:     hex.EncodeToString(eventRecord.Data),
			TxHash:   eventRecord.TxHash,
		}

		go func() {
			c.stateDataFeed.Send(core.NewStateChangeEvent{StateData: &stateData})
		}()

		recordBytes, err := rlp.EncodeToBytes(eventRecord)
		if err != nil {
			return err
		}

		// get packed data for commit state
		data, err := c.stateReceiverABI.Pack(method, recordBytes)
		if err != nil {
			log.Error("Unable to pack tx for commitState", "error", err)
			return err
		}

		// get system message
		msg := getSystemMessage(common.HexToAddress(c.config.StateReceiverContract), data)

		// apply message
		if err := applyMessage(msg, state, header, c.chainConfig, chain); err != nil {
			return err
		}
	}

	return nil
}

// SubscribeStateEvent registers a subscription of ChainSideEvent.
func (c *Bor) SubscribeStateEvent(ch chan<- core.NewStateChangeEvent) event.Subscription {
	return c.scope.Track(c.stateDataFeed.Subscribe(ch))
}

func (c *Bor) SetHeimdallClient(h IHeimdallClient) {
	c.HeimdallClient = h
}

func (c *Bor) IsValidatorAction(chain consensus.ChainReader, from common.Address, tx *types.Transaction) bool {
	header := chain.CurrentHeader()
	validators, err := c.GetCurrentValidators(header.Number.Uint64(), header.Number.Uint64()+1)
	if err != nil {
		log.Error("Failed fetching snapshot", err)
		return false
	}

	isValidator := false
	for _, validator := range validators {
		if bytes.Compare(validator.Address.Bytes(), from.Bytes()) == 0 {
			isValidator = true
			break
		}
	}

	return isValidator && (isProposeSpanAction(tx, chain.Config().Bor.ValidatorContract) ||
		isProposeStateAction(tx, chain.Config().Bor.StateReceiverContract))
}

func isProposeSpanAction(tx *types.Transaction, validatorContract string) bool {
	// keccak256('proposeSpan()').slice(0, 4)
	proposeSpanSig, _ := hex.DecodeString("4b0e4d17")
	if tx.Data() == nil || len(tx.Data()) < 4 {
		return false
	}

	return bytes.Compare(proposeSpanSig, tx.Data()[:4]) == 0 &&
		tx.To().String() == validatorContract
}

func isProposeStateAction(tx *types.Transaction, stateReceiverContract string) bool {
	// keccak256('proposeState(uint256)').slice(0, 4)
	proposeStateSig, _ := hex.DecodeString("ede01f17")
	if tx.Data() == nil || len(tx.Data()) < 4 {
		return false
	}

	return bytes.Compare(proposeStateSig, tx.Data()[:4]) == 0 &&
		tx.To().String() == stateReceiverContract
}

//
// Private methods
//

//
// Chain context
//

// chain context
type chainContext struct {
	Chain consensus.ChainReader
	Bor   consensus.Engine
}

func (c chainContext) Engine() consensus.Engine {
	return c.Bor
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

// get system message
func getSystemMessage(toAddress common.Address, data []byte) callmsg {
	return callmsg{
		ethereum.CallMsg{
			From:     systemAddress,
			Gas:      math.MaxUint64 / 2,
			GasPrice: big.NewInt(0),
			Value:    big.NewInt(0),
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
) error {
	// Create a new context to be used in the EVM environment
	context := core.NewEVMContext(msg, header, chainContext, &header.Coinbase)
	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms.
	vmenv := vm.NewEVM(context, state, chainConfig, vm.Config{})
	// Apply the transaction to the current state (included in the env)
	_, _, err := vmenv.Call(
		vm.AccountRef(msg.From()),
		*msg.To(),
		msg.Data(),
		msg.Gas(),
		msg.Value(),
	)
	// Update the state with pending changes
	if err != nil {
		state.Finalise(true)
	}

	return nil
}

func validatorContains(a []*Validator, x *Validator) (*Validator, bool) {
	for _, n := range a {
		if bytes.Compare(n.Address.Bytes(), x.Address.Bytes()) == 0 {
			return n, true
		}
	}
	return nil, false
}

func getUpdatedValidatorSet(oldValidatorSet *ValidatorSet, newVals []*Validator) *ValidatorSet {
	v := oldValidatorSet
	oldVals := v.Validators

	var changes []*Validator
	for _, ov := range oldVals {
		if f, ok := validatorContains(newVals, ov); ok {
			ov.VotingPower = f.VotingPower
		} else {
			ov.VotingPower = 0
		}

		changes = append(changes, ov)
	}

	for _, nv := range newVals {
		if _, ok := validatorContains(changes, nv); !ok {
			changes = append(changes, nv)
		}
	}

	v.UpdateWithChangeSet(changes)
	return v
}
