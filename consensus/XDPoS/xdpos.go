// Copyright (c) 2018 XDCchain
// Copyright 2024 The go-ethereum Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

// Package XDPoS implements the XDC Network's delegated proof-of-stake consensus engine.
// This is a port from XDPoSChain (geth 1.8.x) to modern go-ethereum (1.17.x).
//
// XDPoS is a DPoS (Delegated Proof of Stake) consensus mechanism where masternodes
// take turns producing blocks in a round-robin fashion within epochs.
//
// Key concepts:
// - Epoch: A fixed number of blocks after which the masternode list may change
// - Masternode: Authorized nodes that can produce blocks
// - Gap: Blocks before epoch end to prepare next epoch's masternode list
// - Validator: Double validation - each block needs creator + validator signatures
//
// This implementation includes:
// - Hook system for rewards, penalties, and validator management
// - Block signer caching for transaction signing
// - Double validation support
// - Penalty system for misbehaving masternodes
package XDPoS

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"golang.org/x/crypto/sha3"
)

const (
	inmemorySnapshots      = 128  // Number of recent vote snapshots to keep in memory
	inmemorySignatures     = 4096 // Number of recent block signatures to keep in memory
	blockSignersCacheLimit = 9000 // Cache limit for block signers
	M2ByteLength           = 4    // Length of M2 validator index bytes

	extraVanity = 32                     // Fixed number of extra-data prefix bytes reserved for signer vanity
	extraSeal   = crypto.SignatureLength // Fixed number of extra-data suffix bytes reserved for signer seal (65 bytes)
)

// Masternode represents a masternode with its address and stake
type Masternode struct {
	Address common.Address
	Stake   *big.Int
}

// XDPoS proof-of-stake-voting protocol constants.
var (
	defaultEpochLength = uint64(900) // Default number of blocks per epoch
	defaultPeriod      = uint64(2)   // Default block time in seconds
	defaultGap         = uint64(450) // Default gap blocks before epoch switch

	nonceAuthVote = hexutil.MustDecode("0xffffffffffffffff") // Magic nonce number to vote on adding a new signer
	nonceDropVote = hexutil.MustDecode("0x0000000000000000") // Magic nonce number to vote on removing a signer.

	uncleHash = types.CalcUncleHash(nil) // Always Keccak256(RLP([])) as uncles are meaningless outside of PoW.

	diffInTurn = big.NewInt(2) // Block difficulty for in-turn signatures
	diffNoTurn = big.NewInt(1) // Block difficulty for out-of-turn signatures
)

// Various error messages to mark blocks invalid.
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
	errMissingSignature = errors.New("extra-data 65 byte suffix signature missing")

	// errExtraSigners is returned if non-checkpoint block contain signer data in
	// their extra-data fields.
	errExtraSigners = errors.New("non-checkpoint block contains extra signer list")

	// errInvalidCheckpointSigners is returned if a checkpoint block contains an
	// invalid list of signers (i.e. non divisible by 20 bytes, or not the correct ones).
	errInvalidCheckpointSigners = errors.New("invalid signer list on checkpoint block")

	// errInvalidCheckpointPenalties is returned if penalties list is invalid
	errInvalidCheckpointPenalties = errors.New("invalid penalty list on checkpoint block")

	// errInvalidMixDigest is returned if a block's mix digest is non-zero.
	errInvalidMixDigest = errors.New("non-zero mix digest")

	// errInvalidUncleHash is returned if a block contains an non-empty uncle list.
	errInvalidUncleHash = errors.New("non empty uncle hash")

	// errInvalidDifficulty is returned if the difficulty of a block is not either
	// of 1 or 2, or if the value does not match the turn of the signer.
	errInvalidDifficulty = errors.New("invalid difficulty")

	// ErrInvalidTimestamp is returned if the timestamp of a block is lower than
	// the previous block's timestamp + the minimum block period.
	ErrInvalidTimestamp = errors.New("invalid timestamp")

	// errInvalidVotingChain is returned if an authorization list is attempted to
	// be modified via out-of-range or non-contiguous headers.
	errInvalidVotingChain = errors.New("invalid voting chain")

	// errUnauthorized is returned if a header is signed by a non-authorized entity.
	errUnauthorized = errors.New("unauthorized")

	// errFailedDoubleValidation is returned if double validation fails
	errFailedDoubleValidation = errors.New("wrong pair of creator-validator in double validation")

	// errWaitTransactions is returned if an empty block is attempted to be sealed
	// on an instant chain (0 second period).
	errWaitTransactions = errors.New("waiting for transactions")

	// ErrInvalidCheckpointValidators is returned if validators list is invalid
	ErrInvalidCheckpointValidators = errors.New("invalid validators list on checkpoint block")
)

// SignerFn is a signer callback function to request a hash to be signed by a
// backing account.
type SignerFn func(accounts.Account, string, []byte) ([]byte, error)

// sigHash returns the hash which is used as input for the proof-of-stake-voting
// signing. It is the hash of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
func sigHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()
	encodeSigHeader(hasher, header)
	hasher.Sum(hash[:0])
	return hash
}

// SealHash returns the hash of a block prior to it being sealed (exported).
func SealHash(header *types.Header) (hash common.Hash) {
	return sigHash(header)
}

// encodeSigHeader encodes the header for signing, excluding the signature
func encodeSigHeader(w io.Writer, header *types.Header) {
	enc := []interface{}{
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
		header.Extra[:len(header.Extra)-extraSeal], // Exclude signature
		header.MixDigest,
		header.Nonce,
	}
	// Add post-London fields if present
	if header.BaseFee != nil {
		enc = append(enc, header.BaseFee)
	}
	if header.WithdrawalsHash != nil {
		enc = append(enc, header.WithdrawalsHash)
	}
	if header.BlobGasUsed != nil {
		enc = append(enc, header.BlobGasUsed)
	}
	if header.ExcessBlobGas != nil {
		enc = append(enc, header.ExcessBlobGas)
	}
	if header.ParentBeaconRoot != nil {
		enc = append(enc, header.ParentBeaconRoot)
	}
	if err := rlp.Encode(w, enc); err != nil {
		panic("can't encode: " + err.Error())
	}
}

// ecrecover extracts the Ethereum account address from a signed header.
func ecrecover(header *types.Header, sigcache *lru.Cache[common.Hash, common.Address]) (common.Address, error) {
	// If the signature's already cached, return that
	hash := header.Hash()
	if address, known := sigcache.Get(hash); known {
		return address, nil
	}
	// Retrieve the signature from the header extra-data
	if len(header.Extra) < extraSeal {
		return common.Address{}, errMissingSignature
	}
	signature := header.Extra[len(header.Extra)-extraSeal:]

	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(sigHash(header).Bytes(), signature)
	if err != nil {
		return common.Address{}, err
	}
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])

	sigcache.Add(hash, signer)
	return signer, nil
}

// XDPoS is the delegated-proof-of-stake consensus engine.
type XDPoS struct {
	config *params.XDPoSConfig // Consensus engine configuration parameters
	db     ethdb.Database      // Database to store and retrieve snapshot checkpoints

	recents             *lru.Cache[common.Hash, *Snapshot]         // Snapshots for recent block to speed up reorgs
	signatures          *lru.Cache[common.Hash, common.Address]    // Signatures of recent blocks to speed up mining
	validatorSignatures *lru.Cache[common.Hash, common.Address]    // Validator signatures cache
	verifiedHeaders     *lru.Cache[common.Hash, bool]              // Verified headers cache
	proposals           map[common.Address]bool                    // Current list of proposals we are pushing

	signer common.Address // Ethereum address of the signing key
	signFn SignerFn       // Signer function to authorize hashes with
	lock   sync.RWMutex   // Protects the signer fields

	// Block signers cache for tracking signing transactions
	BlockSigners *lru.Cache[common.Hash, []*types.Transaction]

	// Hook functions for XDC-specific logic
	// These hooks integrate with the XDC smart contracts for rewards, penalties, etc.
	HookReward            func(chain consensus.ChainHeaderReader, state *state.StateDB, header *types.Header) (map[string]interface{}, error)
	HookPenalty           func(chain consensus.ChainHeaderReader, blockNumberEpoc uint64) ([]common.Address, error)
	HookPenaltyTIPSigning func(chain consensus.ChainHeaderReader, header *types.Header, candidate []common.Address) ([]common.Address, error)
	HookValidator         func(header *types.Header, signers []common.Address) ([]byte, error)
	HookVerifyMNs         func(header *types.Header, signers []common.Address) error

	// Testing flags
	fakeDiff bool // Skip difficulty verifications (testing)
}

// New creates a XDPoS delegated-proof-of-stake consensus engine.
func New(config *params.XDPoSConfig, db ethdb.Database) *XDPoS {
	// Set any missing consensus parameters to their defaults
	conf := *config
	if conf.Epoch == 0 {
		conf.Epoch = defaultEpochLength
	}
	if conf.Period == 0 {
		conf.Period = defaultPeriod
	}
	if conf.Gap == 0 {
		conf.Gap = defaultGap
	}

	// Allocate the snapshot caches and create the engine
	recents := lru.NewCache[common.Hash, *Snapshot](inmemorySnapshots)
	signatures := lru.NewCache[common.Hash, common.Address](inmemorySignatures)
	validatorSignatures := lru.NewCache[common.Hash, common.Address](inmemorySignatures)
	verifiedHeaders := lru.NewCache[common.Hash, bool](inmemorySnapshots)
	blockSigners := lru.NewCache[common.Hash, []*types.Transaction](blockSignersCacheLimit)

	return &XDPoS{
		config:              &conf,
		db:                  db,
		recents:             recents,
		signatures:          signatures,
		validatorSignatures: validatorSignatures,
		verifiedHeaders:     verifiedHeaders,
		BlockSigners:        blockSigners,
		proposals:           make(map[common.Address]bool),
	}
}

// Author implements consensus.Engine, returning the Ethereum address recovered
// from the signature in the header's extra-data section.
func (c *XDPoS) Author(header *types.Header) (common.Address, error) {
	return ecrecover(header, c.signatures)
}

// VerifyHeader checks whether a header conforms to the consensus rules.
func (c *XDPoS) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header) error {
	return c.verifyHeaderWithCache(chain, header, nil)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers. The
// method returns a quit channel to abort the operations and a results channel to
// retrieve the async verifications (the order is that of the input slice).
func (c *XDPoS) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for i, header := range headers {
			err := c.verifyHeaderWithCache(chain, header, headers[:i])

			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

func (c *XDPoS) verifyHeaderWithCache(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
	if _, check := c.verifiedHeaders.Get(header.Hash()); check {
		return nil
	}
	err := c.verifyHeader(chain, header, parents)
	if err == nil {
		c.verifiedHeaders.Add(header.Hash(), true)
	}
	return err
}

// verifyHeader checks whether a header conforms to the consensus rules.
func (c *XDPoS) verifyHeader(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
	if header.Number == nil {
		return errUnknownBlock
	}
	number := header.Number.Uint64()

	// Don't waste time checking blocks from the future
	if header.Time > uint64(time.Now().Unix()) {
		return consensus.ErrFutureBlock
	}

	// Checkpoint blocks need to enforce zero beneficiary
	checkpoint := (number % c.config.Epoch) == 0
	if checkpoint && header.Coinbase != (common.Address{}) {
		return errInvalidCheckpointBeneficiary
	}

	// Nonces must be 0x00..0 or 0xff..f, zeroes enforced on checkpoints
	if !bytes.Equal(header.Nonce[:], nonceAuthVote) && !bytes.Equal(header.Nonce[:], nonceDropVote) {
		return errInvalidVote
	}
	if checkpoint && !bytes.Equal(header.Nonce[:], nonceDropVote) {
		return errInvalidCheckpointVote
	}

	// Check that the extra-data contains both the vanity and signature
	if len(header.Extra) < extraVanity {
		return errMissingVanity
	}
	if len(header.Extra) < extraVanity+extraSeal {
		return errMissingSignature
	}

	// Ensure that the extra-data contains a signer list on checkpoint, but none otherwise
	signersBytes := len(header.Extra) - extraVanity - extraSeal
	if !checkpoint && signersBytes != 0 {
		return errExtraSigners
	}
	if checkpoint && signersBytes%common.AddressLength != 0 {
		return errInvalidCheckpointSigners
	}

	// Ensure that the mix digest is zero as we don't have fork protection currently
	if header.MixDigest != (common.Hash{}) {
		return errInvalidMixDigest
	}

	// Ensure that the block doesn't contain any uncles which are meaningless in XDPoS
	if header.UncleHash != uncleHash {
		return errInvalidUncleHash
	}

	// All basic checks passed, verify cascading fields
	return c.verifyCascadingFields(chain, header, parents)
}

// verifyCascadingFields verifies all the header fields that are not standalone,
// rather depend on a batch of previous headers.
func (c *XDPoS) verifyCascadingFields(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
	// The genesis block is the always valid dead-end
	number := header.Number.Uint64()
	if number == 0 {
		return nil
	}

	// Ensure that the block's timestamp isn't too close to its parent
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

	// If the block is a checkpoint block, verify the signer list
	if number%c.config.Epoch == 0 {
		signers := snap.GetSigners()
		penPenalties := []common.Address{}

		// Get penalties if hook is configured
		if c.HookPenalty != nil || c.HookPenaltyTIPSigning != nil {
			var penErr error
			if c.HookPenaltyTIPSigning != nil {
				penPenalties, penErr = c.HookPenaltyTIPSigning(chain, header, signers)
			} else if c.HookPenalty != nil {
				penPenalties, penErr = c.HookPenalty(chain, number)
			}
			if penErr != nil {
				return penErr
			}
			for _, address := range penPenalties {
				log.Debug("Penalty Info", "address", address, "number", number)
			}
		}

		// Remove penalties from signers
		signers = removeItemFromArray(signers, penPenalties)

		// Remove penalties from previous epochs (up to LimitPenaltyEpoch)
		for i := 1; i <= LimitPenaltyEpoch; i++ {
			if number > uint64(i)*c.config.Epoch {
				signers = c.removePenaltiesFromBlock(chain, signers, number-uint64(i)*c.config.Epoch)
			}
		}

		// Verify checkpoint signers match
		extraSuffix := len(header.Extra) - extraSeal
		masternodesFromCheckpointHeader := extractAddressFromBytes(header.Extra[extraVanity:extraSuffix])
		if !compareSignersLists(masternodesFromCheckpointHeader, signers) {
			log.Error("Masternodes lists are different in checkpoint header and snapshot",
				"number", number,
				"masternodes_from_checkpoint_header", masternodesFromCheckpointHeader,
				"masternodes_in_snapshot", signers,
				"penList", penPenalties)
			return errInvalidCheckpointSigners
		}

		// Verify MNs if hook is configured
		if c.HookVerifyMNs != nil {
			if err := c.HookVerifyMNs(header, signers); err != nil {
				return err
			}
		}
	}

	// All basic checks passed, verify the seal and return
	return c.verifySeal(chain, header, parents)
}

// verifySeal checks whether the signature contained in the header satisfies the
// consensus protocol requirements.
func (c *XDPoS) verifySeal(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
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
	creator, err := ecrecover(header, c.signatures)
	if err != nil {
		return err
	}

	var parent *types.Header
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeader(header.ParentHash, number-1)
	}

	difficulty := c.calcDifficulty(chain, parent, creator)
	log.Debug("verify seal block", "number", header.Number, "hash", header.Hash(), "block difficulty", header.Difficulty, "calc difficulty", difficulty, "creator", creator)

	// Ensure that the block's difficulty is meaningful (may not be correct at this point)
	if number > 0 && !c.fakeDiff {
		if header.Difficulty.Int64() != difficulty.Int64() {
			return errInvalidDifficulty
		}
	}

	// Get masternodes for this header
	masternodes := c.GetMasternodes(chain, header)

	// Check if creator is authorized
	if _, ok := snap.Signers[creator]; !ok {
		valid := false
		for _, m := range masternodes {
			if m == creator {
				valid = true
				break
			}
		}
		if !valid {
			log.Debug("Unauthorized creator found", "block number", number, "creator", creator.String())
			return errUnauthorized
		}
	}

	// Check recent signers to prevent spamming
	if len(masternodes) > 1 {
		for seen, recent := range snap.Recents {
			if recent == creator {
				if limit := uint64(2); seen > number-limit {
					if number%c.config.Epoch != 0 {
						return errUnauthorized
					}
				}
			}
		}
	}

	return nil
}

// VerifyUncles implements consensus.Engine, always returning an error for any
// uncles as this consensus mechanism doesn't permit uncles.
func (c *XDPoS) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errors.New("uncles not allowed")
	}
	return nil
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (c *XDPoS) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	header.Coinbase = common.Address{}
	header.Nonce = types.BlockNonce{}

	number := header.Number.Uint64()

	snap, err := c.snapshot(chain, number-1, header.ParentHash, nil)
	if err != nil {
		return err
	}

	if number%c.config.Epoch != 0 {
		c.lock.RLock()
		addresses := make([]common.Address, 0, len(c.proposals))
		for address, authorize := range c.proposals {
			if snap.validVote(address, authorize) {
				addresses = append(addresses, address)
			}
		}
		if len(addresses) > 0 {
			header.Coinbase = addresses[rand.Intn(len(addresses))]
			if c.proposals[header.Coinbase] {
				copy(header.Nonce[:], nonceAuthVote)
			} else {
				copy(header.Nonce[:], nonceDropVote)
			}
		}
		c.lock.RUnlock()
	}

	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}

	header.Difficulty = c.calcDifficulty(chain, parent, c.signer)
	log.Debug("CalcDifficulty", "number", header.Number, "difficulty", header.Difficulty)

	if len(header.Extra) < extraVanity {
		header.Extra = append(header.Extra, bytes.Repeat([]byte{0x00}, extraVanity-len(header.Extra))...)
	}
	header.Extra = header.Extra[:extraVanity]

	masternodes := snap.GetSigners()
	if number >= c.config.Epoch && number%c.config.Epoch == 0 {
		if c.HookPenalty != nil || c.HookPenaltyTIPSigning != nil {
			var penMasternodes []common.Address
			var penErr error

			if c.HookPenaltyTIPSigning != nil {
				penMasternodes, penErr = c.HookPenaltyTIPSigning(chain, header, masternodes)
			} else if c.HookPenalty != nil {
				penMasternodes, penErr = c.HookPenalty(chain, number)
			}
			if penErr != nil {
				return penErr
			}
			if len(penMasternodes) > 0 {
				masternodes = removeItemFromArray(masternodes, penMasternodes)
				for _, address := range penMasternodes {
					log.Debug("Penalty status", "address", address, "number", number)
				}
			}
		}

		for i := 1; i <= LimitPenaltyEpoch; i++ {
			if number > uint64(i)*c.config.Epoch {
				masternodes = c.removePenaltiesFromBlock(chain, masternodes, number-uint64(i)*c.config.Epoch)
			}
		}

		for _, masternode := range masternodes {
			header.Extra = append(header.Extra, masternode[:]...)
		}

		if c.HookValidator != nil {
			_, err := c.HookValidator(header, masternodes)
			if err != nil {
				return err
			}
		}
	}
	header.Extra = append(header.Extra, make([]byte, extraSeal)...)

	header.MixDigest = common.Hash{}

	header.Time = parent.Time + c.config.Period
	if header.Time < uint64(time.Now().Unix()) {
		header.Time = uint64(time.Now().Unix())
	}

	return nil
}

// Finalize implements consensus.Engine.
func (c *XDPoS) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state vm.StateDB, body *types.Body) {
	header.UncleHash = types.CalcUncleHash(nil)
}

// FinalizeAndAssemble implements consensus.Engine.
func (c *XDPoS) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, statedb *state.StateDB, body *types.Body, receipts []*types.Receipt) (*types.Block, error) {
	number := header.Number.Uint64()
	rCheckpoint := c.config.RewardCheckpoint
	if rCheckpoint == 0 {
		rCheckpoint = c.config.Epoch
	}

	if c.HookReward != nil && number%rCheckpoint == 0 {
		rewards, err := c.HookReward(chain, statedb, header)
		if err != nil {
			return nil, err
		}
		if len(StoreRewardFolder) > 0 {
			data, err := json.Marshal(rewards)
			if err == nil {
				err = os.WriteFile(filepath.Join(StoreRewardFolder, header.Number.String()+"."+header.Hash().Hex()), data, 0644)
			}
			if err != nil {
				log.Error("Error when save reward info", "number", header.Number, "hash", header.Hash().Hex(), "err", err)
			}
		}
	}

	header.Root = statedb.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	header.UncleHash = types.CalcUncleHash(nil)

	return types.NewBlock(header, body, receipts, trie.NewStackTrie(nil)), nil
}

// Seal implements consensus.Engine.
func (c *XDPoS) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	header := block.Header()

	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}

	if c.config.Period == 0 && len(block.Transactions()) == 0 && number%c.config.Epoch != 0 {
		return errWaitTransactions
	}

	c.lock.RLock()
	signer, signFn := c.signer, c.signFn
	c.lock.RUnlock()

	snap, err := c.snapshot(chain, number-1, header.ParentHash, nil)
	if err != nil {
		return err
	}
	masternodes := c.GetMasternodes(chain, header)

	if _, authorized := snap.Signers[signer]; !authorized {
		valid := false
		for _, m := range masternodes {
			if m == signer {
				valid = true
				break
			}
		}
		if !valid {
			return errUnauthorized
		}
	}

	if len(masternodes) > 1 {
		for seen, recent := range snap.Recents {
			if recent == signer {
				if limit := uint64(2); number < limit || seen > number-limit {
					if number%c.config.Epoch != 0 {
						log.Info("Signed recently, must wait for others", "len(masternodes)", len(masternodes), "number", number, "limit", limit, "seen", seen, "recent", recent.String())
						<-stop
						return nil
					}
				}
			}
		}
	}

	select {
	case <-stop:
		return nil
	default:
	}

	sighash, err := signFn(accounts.Account{Address: signer}, accounts.MimetypeClique, sigHash(header).Bytes())
	if err != nil {
		return err
	}
	copy(header.Extra[len(header.Extra)-extraSeal:], sighash)

	delay := time.Unix(int64(header.Time), 0).Sub(time.Now())
	if header.Difficulty.Cmp(diffNoTurn) == 0 {
		wiggle := time.Duration(len(masternodes)/2+1) * wiggleTime
		delay += time.Duration(rand.Int63n(int64(wiggle)))
		log.Trace("Out-of-turn signing requested", "wiggle", common.PrettyDuration(wiggle))
	}

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

// CalcDifficulty is the difficulty adjustment algorithm.
func (c *XDPoS) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	return c.calcDifficulty(chain, parent, c.signer)
}

func (c *XDPoS) calcDifficulty(chain consensus.ChainHeaderReader, parent *types.Header, signer common.Address) *big.Int {
	numMNs, preIndex, curIndex, _, err := c.YourTurn(chain, parent, signer)
	if err != nil {
		return big.NewInt(int64(numMNs + curIndex - preIndex))
	}
	return big.NewInt(int64(numMNs - Hop(numMNs, preIndex, curIndex)))
}

// SealHash returns the hash of a block prior to it being sealed.
func (c *XDPoS) SealHash(header *types.Header) common.Hash {
	return SealHash(header)
}

// Close implements consensus.Engine.
func (c *XDPoS) Close() error {
	return nil
}

// APIs implements consensus.Engine.
func (c *XDPoS) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return []rpc.API{{
		Namespace: "xdpos",
		Service:   &API{chain: chain, xdpos: c},
	}}
}

// Authorize injects a private key into the consensus engine.
func (c *XDPoS) Authorize(signer common.Address, signFn SignerFn) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.signer = signer
	c.signFn = signFn
}

// GetPeriod returns the configured block period
func (c *XDPoS) GetPeriod() uint64 { return c.config.Period }

// GetDb returns the database
func (c *XDPoS) GetDb() ethdb.Database { return c.db }

// GetSnapshot returns the snapshot at a given header
func (c *XDPoS) GetSnapshot(chain consensus.ChainHeaderReader, header *types.Header) (*Snapshot, error) {
	number := header.Number.Uint64()
	log.Trace("take snapshot", "number", number, "hash", header.Hash())
	return c.snapshot(chain, number, header.Hash(), nil)
}

// StoreSnapshot stores the snapshot to the database
func (c *XDPoS) StoreSnapshot(snap *Snapshot) error {
	return snap.store(c.db)
}

// YourTurn checks if it's the signer's turn to mine
func (c *XDPoS) YourTurn(chain consensus.ChainHeaderReader, parent *types.Header, signer common.Address) (int, int, int, bool, error) {
	masternodes := c.GetMasternodes(chain, parent)

	snap, err := c.GetSnapshot(chain, parent)
	if err != nil {
		log.Warn("Failed when trying to commit new work", "err", err)
		return 0, -1, -1, false, err
	}
	if len(masternodes) == 0 {
		return 0, -1, -1, false, errors.New("Masternodes not found")
	}

	pre := common.Address{}
	preIndex := -1
	if parent.Number.Uint64() != 0 {
		pre, err = whoIsCreator(snap, parent)
		if err != nil {
			return 0, 0, 0, false, err
		}
		preIndex = position(masternodes, pre)
	}

	curIndex := position(masternodes, signer)
	if signer == c.signer {
		log.Debug("Masternodes cycle info", "number of masternodes", len(masternodes), "previous", pre, "position", preIndex, "current", signer, "position", curIndex)
	}

	if (preIndex+1)%len(masternodes) == curIndex {
		return len(masternodes), preIndex, curIndex, true, nil
	}
	return len(masternodes), preIndex, curIndex, false, nil
}

// GetMasternodes returns the list of masternodes for a given header
func (c *XDPoS) GetMasternodes(chain consensus.ChainHeaderReader, header *types.Header) []common.Address {
	n := header.Number.Uint64()
	e := c.config.Epoch
	switch {
	case n%e == 0:
		return c.GetMasternodesFromCheckpointHeader(header, n, e)
	case n%e != 0:
		h := chain.GetHeaderByNumber(n - (n % e))
		return c.GetMasternodesFromCheckpointHeader(h, n, e)
	default:
		return []common.Address{}
	}
}

// GetMasternodesFromCheckpointHeader extracts masternode list from checkpoint header
func (c *XDPoS) GetMasternodesFromCheckpointHeader(preCheckpointHeader *types.Header, n, e uint64) []common.Address {
	if preCheckpointHeader == nil {
		log.Info("Previous checkpoint's header is empty", "block number", n, "epoch", e)
		return []common.Address{}
	}
	masternodes := make([]common.Address, (len(preCheckpointHeader.Extra)-extraVanity-extraSeal)/common.AddressLength)
	for i := 0; i < len(masternodes); i++ {
		copy(masternodes[i][:], preCheckpointHeader.Extra[extraVanity+i*common.AddressLength:])
	}
	return masternodes
}

// UpdateMasternodes updates the masternode list in the snapshot
func (c *XDPoS) UpdateMasternodes(chain consensus.ChainHeaderReader, header *types.Header, ms []Masternode) error {
	number := header.Number.Uint64()
	log.Trace("take snapshot", "number", number, "hash", header.Hash())

	snap, err := c.snapshot(chain, number, header.Hash(), nil)
	if err != nil {
		return err
	}

	newMasternodes := make(map[common.Address]struct{})
	for _, m := range ms {
		newMasternodes[m.Address] = struct{}{}
	}
	snap.Signers = newMasternodes

	nm := []string{}
	for _, n := range ms {
		nm = append(nm, n.Address.String())
	}
	c.recents.Add(snap.Hash, snap)
	log.Info("New set of masternodes has been updated to snapshot", "number", snap.Number, "hash", snap.Hash, "new masternodes", nm)
	return nil
}

// RecoverSigner recovers the signer from a header
func (c *XDPoS) RecoverSigner(header *types.Header) (common.Address, error) {
	return ecrecover(header, c.signatures)
}

// RecoverValidator recovers the validator from a header's Validator field
func (c *XDPoS) RecoverValidator(header *types.Header) (common.Address, error) {
	hash := header.Hash()
	if address, known := c.validatorSignatures.Get(hash); known {
		return address, nil
	}
	// TODO: Implement when header.Validator field exists
	return ecrecover(header, c.signatures)
}

// CacheData caches signing transactions from a block
func (c *XDPoS) CacheData(header *types.Header, txs []*types.Transaction, receipts []*types.Receipt) []*types.Transaction {
	signTxs := []*types.Transaction{}
	for _, tx := range txs {
		if isSigningTransaction(tx) {
			var b uint64
			for _, r := range receipts {
				if r.TxHash == tx.Hash() {
					if len(r.PostState) > 0 {
						b = types.ReceiptStatusSuccessful
					} else {
						b = r.Status
					}
					break
				}
			}
			if b == types.ReceiptStatusFailed {
				continue
			}
			signTxs = append(signTxs, tx)
		}
	}
	log.Debug("Save tx signers to cache", "hash", header.Hash().String(), "number", header.Number, "len(txs)", len(signTxs))
	c.BlockSigners.Add(header.Hash(), signTxs)
	return signTxs
}

// CacheSigner caches signing transactions by hash
func (c *XDPoS) CacheSigner(hash common.Hash, txs []*types.Transaction) []*types.Transaction {
	signTxs := []*types.Transaction{}
	for _, tx := range txs {
		if isSigningTransaction(tx) {
			signTxs = append(signTxs, tx)
		}
	}
	log.Debug("Save tx signers to cache", "hash", hash.String(), "len(txs)", len(signTxs))
	c.BlockSigners.Add(hash, signTxs)
	return signTxs
}

// snapshot retrieves the authorization snapshot at a given point in time.
func (c *XDPoS) snapshot(chain consensus.ChainHeaderReader, number uint64, hash common.Hash, parents []*types.Header) (*Snapshot, error) {
	var (
		headers []*types.Header
		snap    *Snapshot
	)
	for snap == nil {
		if s, ok := c.recents.Get(hash); ok {
			snap = s
			break
		}
		if (number+c.config.Gap)%c.config.Epoch == 0 {
			if s, err := loadSnapshot(c.config, c.signatures, c.db, hash); err == nil {
				log.Trace("Loaded voting snapshot from disk", "number", number, "hash", hash)
				snap = s
				break
			}
		}
		if number == 0 {
			genesis := chain.GetHeaderByNumber(0)
			if err := c.VerifyHeader(chain, genesis); err != nil {
				return nil, err
			}
			signers := make([]common.Address, (len(genesis.Extra)-extraVanity-extraSeal)/common.AddressLength)
			for i := 0; i < len(signers); i++ {
				copy(signers[i][:], genesis.Extra[extraVanity+i*common.AddressLength:])
			}
			snap = newSnapshot(c.config, c.signatures, 0, genesis.Hash(), signers)
			if err := snap.store(c.db); err != nil {
				return nil, err
			}
			log.Trace("Stored genesis voting snapshot to disk")
			break
		}
		var header *types.Header
		if len(parents) > 0 {
			header = parents[len(parents)-1]
			if header.Hash() != hash || header.Number.Uint64() != number {
				return nil, consensus.ErrUnknownAncestor
			}
			parents = parents[:len(parents)-1]
		} else {
			header = chain.GetHeader(hash, number)
			if header == nil {
				return nil, consensus.ErrUnknownAncestor
			}
		}
		headers = append(headers, header)
		number, hash = number-1, header.ParentHash
	}
	for i := 0; i < len(headers)/2; i++ {
		headers[i], headers[len(headers)-1-i] = headers[len(headers)-1-i], headers[i]
	}
	snap, err := snap.apply(headers)
	if err != nil {
		return nil, err
	}
	c.recents.Add(snap.Hash, snap)

	if (snap.Number+c.config.Gap)%c.config.Epoch == 0 {
		if err = snap.store(c.db); err != nil {
			return nil, err
		}
		log.Trace("Stored voting snapshot to disk", "number", snap.Number, "hash", snap.Hash)
	}
	return snap, err
}

func (c *XDPoS) removePenaltiesFromBlock(chain consensus.ChainHeaderReader, masternodes []common.Address, epochNumber uint64) []common.Address {
	if epochNumber <= 0 {
		return masternodes
	}
	header := chain.GetHeaderByNumber(epochNumber)
	if header == nil {
		return masternodes
	}
	return masternodes
}

// Helper functions

func whoIsCreator(snap *Snapshot, header *types.Header) (common.Address, error) {
	if header.Number.Uint64() == 0 {
		return common.Address{}, errors.New("Don't take block 0")
	}
	return ecrecover(header, snap.sigcache)
}

func position(list []common.Address, x common.Address) int {
	for i, item := range list {
		if item == x {
			return i
		}
	}
	return -1
}

// Hop calculates the distance between two positions in the masternode list
func Hop(length, pre, cur int) int {
	switch {
	case pre < cur:
		return cur - (pre + 1)
	case pre > cur:
		return (length - pre) + (cur - 1)
	default:
		return length - 1
	}
}

func compareSignersLists(list1 []common.Address, list2 []common.Address) bool {
	if len(list1) == 0 && len(list2) == 0 {
		return true
	}
	if len(list1) != len(list2) {
		return false
	}
	l1 := make([]common.Address, len(list1))
	l2 := make([]common.Address, len(list2))
	copy(l1, list1)
	copy(l2, list2)
	sort.Slice(l1, func(i, j int) bool { return l1[i].String() <= l1[j].String() })
	sort.Slice(l2, func(i, j int) bool { return l2[i].String() <= l2[j].String() })
	return reflect.DeepEqual(l1, l2)
}

func removeItemFromArray(array []common.Address, toRemove []common.Address) []common.Address {
	result := make([]common.Address, 0)
	for _, item := range array {
		found := false
		for _, r := range toRemove {
			if item == r {
				found = true
				break
			}
		}
		if !found {
			result = append(result, item)
		}
	}
	return result
}

func extractAddressFromBytes(byteAddresses []byte) []common.Address {
	if len(byteAddresses)%common.AddressLength != 0 {
		return []common.Address{}
	}
	addresses := make([]common.Address, len(byteAddresses)/common.AddressLength)
	for i := 0; i < len(addresses); i++ {
		copy(addresses[i][:], byteAddresses[i*common.AddressLength:])
	}
	return addresses
}

// ExtractValidatorsFromBytes extracts validator indices from bytes
func ExtractValidatorsFromBytes(byteValidators []byte) []int64 {
	lenValidator := len(byteValidators) / M2ByteLength
	var validators []int64
	for i := 0; i < lenValidator; i++ {
		trimByte := bytes.Trim(byteValidators[i*M2ByteLength:(i+1)*M2ByteLength], "\x00")
		intNumber, err := strconv.Atoi(string(trimByte))
		if err != nil {
			log.Error("Can not convert string to integer", "error", err)
			return []int64{}
		}
		validators = append(validators, int64(intNumber))
	}
	return validators
}

func isSigningTransaction(tx *types.Transaction) bool {
	return false // Placeholder - needs XDC contract integration
}

var wiggleTime = 500 * time.Millisecond

var _ consensus.Engine = (*XDPoS)(nil)
