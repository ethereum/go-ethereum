package system_contract

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/big"
	"time"

	"golang.org/x/crypto/sha3"

	"github.com/scroll-tech/go-ethereum/accounts"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/consensus"
	"github.com/scroll-tech/go-ethereum/consensus/misc"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rlp"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/scroll-tech/go-ethereum/trie"
)

var (
	extraSeal = crypto.SignatureLength   // Fixed number of extra-data suffix bytes reserved for signer seal
	uncleHash = types.CalcUncleHash(nil) // Always Keccak256(RLP([])) as uncles are meaningless outside of PoW.
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	// errUnknownBlock is returned when the list of signers is requested for a block
	// that is not part of the local blockchain.
	errUnknownBlock = errors.New("unknown block")

	// errInvalidCoinbase is returned if a coinbase value is non-zero
	errInvalidCoinbase = errors.New("coinbase not empty")

	// errInvalidNonce is returned if a nonce value is non-zero
	errInvalidNonce = errors.New("nonce not empty nor zero")

	// errMissingSignature is returned if a block's extra-data section doesn't seem
	// to contain a 65 byte secp256k1 signature.
	errMissingSignature = errors.New("extra-data 65 byte signature missing")

	// errInvalidMixDigest is returned if a block's mix digest is non-zero.
	errInvalidMixDigest = errors.New("non-zero mix digest")

	// errInvalidUncleHash is returned if a block contains an non-empty uncle list.
	errInvalidUncleHash = errors.New("non empty uncle hash")

	// errInvalidDifficulty is returned if a difficulty value is non-zero
	errInvalidDifficulty = errors.New("non-one difficulty")

	// errInvalidTimestamp is returned if the timestamp of a block is lower than
	// the previous block's timestamp.
	errInvalidTimestamp = errors.New("invalid timestamp")

	// errInvalidExtra is returned if the extra data is not empty
	errInvalidExtra = errors.New("invalid extra")
)

// ErrUnauthorizedSigner is returned if a header is signed by a non-authorized entity.
var ErrUnauthorizedSigner = errors.New("unauthorized signer")

// SignerFn hashes and signs the data to be signed by a backing account.
type SignerFn func(signer accounts.Account, mimeType string, message []byte) ([]byte, error)

// Author implements consensus.Engine, returning the Ethereum address recovered
// from the signature in the header's block-signature section.
func (s *SystemContract) Author(header *types.Header) (common.Address, error) {
	return ecrecover(header)
}

// VerifyHeader checks whether a header conforms to the consensus rules of a
// given engine.
func (s *SystemContract) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, seal bool) error {
	return s.verifyHeader(chain, header, nil)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications (the order is that of
// the input slice).
func (s *SystemContract) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool, parent *types.Header) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for i, header := range headers {
			parents := headers[:i]
			if len(parents) == 0 && parent != nil {
				parents = []*types.Header{parent}
			}

			err := s.verifyHeader(chain, header, parents)
			if err != nil {
				log.Error("Error verifying headers", "err", err)
			}
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
func (s *SystemContract) verifyHeader(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
	if header.Number == nil {
		return errUnknownBlock
	}

	// Don't waste time checking blocks from the future.
	// We add 100ms leeway since the scroll_worker internal timers might trigger early.
	now := time.Now()
	if header.Time > uint64(now.Unix()) && time.Unix(int64(header.Time), 0).Sub(now) > 100*time.Millisecond {
		return consensus.ErrFutureBlock
	}
	// Ensure that the coinbase is zero
	if header.Coinbase != (common.Address{}) {
		return errInvalidCoinbase
	}
	// Ensure that the nonce is zero
	if header.Nonce != (types.BlockNonce{}) {
		return errInvalidNonce
	}
	// Check that the BlockSignature contains signature if block is not requested
	if header.Number.Cmp(big.NewInt(0)) != 0 && len(header.BlockSignature) != extraSeal {
		return errMissingSignature
	}
	// Ensure that the mix digest is zero
	if header.MixDigest != (common.Hash{}) {
		return errInvalidMixDigest
	}
	// Ensure that the block doesn't contain any uncles which are meaningless in PoA
	if header.UncleHash != uncleHash {
		return errInvalidUncleHash
	}
	// Ensure that the difficulty is one
	if header.Difficulty.Cmp(common.Big1) != 0 {
		return errInvalidDifficulty
	}
	// Verify that the gas limit is <= 2^63-1
	cap := uint64(0x7fffffffffffffff)
	if header.GasLimit > cap {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, cap)
	}
	if len(header.Extra) > 0 {
		return errInvalidExtra
	}
	//// All basic checks passed, verify cascading fields
	return s.verifyCascadingFields(chain, header, parents)
}

// verifyCascadingFields verifies all the header fields that are not standalone,
// rather depend on a batch of previous headers. The caller may optionally pass
// in a batch of parents (ascending order) to avoid looking those up from the
// database. This is useful for concurrently verifying a batch of new headers.
func (s *SystemContract) verifyCascadingFields(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
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
	if header.Time < parent.Time {
		return errInvalidTimestamp
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}
	if !chain.Config().IsCurie(header.Number) {
		// Verify BaseFee not present before EIP-1559 fork.
		if header.BaseFee != nil {
			return fmt.Errorf("invalid baseFee before fork: have %d, want <nil>", header.BaseFee)
		}
		if err := misc.VerifyGaslimit(parent.GasLimit, header.GasLimit); err != nil {
			return err
		}
	} else if err := misc.VerifyEip1559Header(chain.Config(), parent, header); err != nil {
		// Verify the header's EIP-1559 attributes.
		return err
	}

	signer, err := ecrecover(header)
	if err != nil {
		return err
	}

	s.lock.RLock()
	defer s.lock.RUnlock()

	if signer != s.signerAddressL1 {
		log.Error("Unauthorized signer", "Got", signer, "Expected:", s.signerAddressL1)
		return ErrUnauthorizedSigner
	}

	return nil
}

// VerifyUncles implements consensus.Engine, always returning an error for any
// uncles as this consensus mechanism doesn't permit uncles.
func (s *SystemContract) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errors.New("uncles not allowed")
	}
	return nil
}

// CalcBlocksPerSecond returns the number of blocks per second
// Uses the BlocksPerSecond configuration parameter directly
// Default is 1 block per second if not specified
func CalcBlocksPerSecond(blocksPerSecond uint64) uint64 {
	if blocksPerSecond == 0 {
		return 1 // Default to 1 block per second
	}
	return blocksPerSecond
}

// CalcPeriodMs calculates the period in milliseconds between blocks
// based on the blocks per second configuration
func CalcPeriodMs(blocksPerSecond uint64) uint64 {
	bps := CalcBlocksPerSecond(blocksPerSecond)
	return 1000 / bps
}

func (s *SystemContract) CalcTimestamp(parent *types.Header) uint64 {
	var timestamp uint64
	if s.config.Period == 1 {
		// Get the base timestamp (in seconds)
		timestamp = parent.Time

		blocksPerSecond := CalcBlocksPerSecond(s.config.BlocksPerSecond)

		// Calculate the block index within the current period for the next block
		blockIndex := parent.Number.Uint64() % blocksPerSecond

		// If this block is the last one in the current second, increment the timestamp
		// We compare with blocksPerSecond-1 because blockIndex is 0-based
		if blockIndex == blocksPerSecond-1 {
			timestamp++
		}
	} else {
		timestamp = parent.Time + s.config.Period
	}

	// If RelaxedPeriod is enabled, always set the header timestamp to now (ie the time we start building it) as
	// we don't know when it will be sealed
	if s.config.RelaxedPeriod || timestamp < uint64(time.Now().Unix()) {
		timestamp = uint64(time.Now().Unix())
	}

	return timestamp
}

// Prepare initializes the consensus fields of a block header according to the
// rules of a particular engine. Update only timestamp and prepare ExtraData for Signature
func (s *SystemContract) Prepare(chain consensus.ChainHeaderReader, header *types.Header, timeOverride *uint64) error {
	// Make sure unused fields are empty
	header.Coinbase = common.Address{}
	header.Nonce = types.BlockNonce{}
	header.MixDigest = common.Hash{}

	// Prepare EuclidV2-related fields
	header.BlockSignature = make([]byte, extraSeal)
	header.IsEuclidV2 = true
	header.Extra = nil

	// Ensure the timestamp has the correct delay
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	if timeOverride != nil {
		header.Time = *timeOverride
	} else {
		header.Time = s.CalcTimestamp(parent)
	}

	// Difficulty must be 1
	header.Difficulty = big.NewInt(1)

	return nil
}

// Finalize implements consensus.Engine. There is no post-transaction
// No rules here
func (s *SystemContract) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header) {
	// No block rewards in PoA, so the state remains as is
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	header.UncleHash = types.CalcUncleHash(nil)
}

// FinalizeAndAssemble implements consensus.Engine, ensuring no uncles are set,
// nor block rewards given, and returns the final block.
func (s *SystemContract) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	// Finalize block
	s.Finalize(chain, header, state, txs, uncles)

	// Assemble and return the final block for sealing.
	return types.NewBlock(header, txs, nil, receipts, trie.NewStackTrie(nil)), nil
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
func (s *SystemContract) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	header := block.Header()
	// Sealing the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}

	// For 0-period chains, refuse to seal empty blocks (no reward but would spin sealing)
	if s.config.Period == 0 && len(block.Transactions()) == 0 {
		return errors.New("sealing paused while waiting for transactions")
	}

	// Don't hold the signer fields for the entire sealing procedure
	s.lock.RLock()
	signer, signFn := s.signer, s.signFn
	signerAddressL1 := s.signerAddressL1
	s.lock.RUnlock()

	// Bail out if we are unauthorized to sign a block
	if signer != signerAddressL1 {
		return ErrUnauthorizedSigner
	}

	// Sweet, the protocol permits us to sign the block, wait for our time
	delay := time.Unix(int64(header.Time), 0).Sub(time.Now()) // nolint: gosimple

	// Sign all the things!
	sighash, err := signFn(accounts.Account{Address: signer}, accounts.MimetypeClique, SystemContractRLP(header))
	if err != nil {
		return err
	}
	copy(header.BlockSignature[0:], sighash)
	// Wait until sealing is terminated or delay timeout.
	log.Trace("Waiting for slot to sign and propagate", "delay", common.PrettyDuration(delay))
	go func() {
		defer close(results)

		select {
		case <-stop:
			return
		case <-time.After(delay):
		}

		select {
		case results <- block.WithSeal(header):
		case <-time.After(time.Second):
			log.Warn("Sealing result is not read by miner", "sealhash", SealHash(header))
		}
	}()

	return nil
}

// SealHash returns the hash of a block prior to it being sealed.
func (s *SystemContract) SealHash(header *types.Header) (hash common.Hash) {
	return SealHash(header)
}

// SealHash returns the hash of a block prior to it being sealed.
func SealHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()
	encodeSigHeader(hasher, header)
	hasher.(crypto.KeccakState).Read(hash[:])
	return hash
}

// ecrecover extracts the Ethereum account address from a signed header.
func ecrecover(header *types.Header) (common.Address, error) {
	signature := header.BlockSignature[0:]

	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(SealHash(header).Bytes(), signature)
	if err != nil {
		return common.Address{}, err
	}
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])

	return signer, nil
}

// SystemContractRLP returns the rlp bytes which needs to be signed for the system contract
// sealing. The RLP to sign consists of the entire header apart from the ExtraData
func SystemContractRLP(header *types.Header) []byte {
	b := new(bytes.Buffer)
	encodeSigHeader(b, header)
	return b.Bytes()
}

// CalcDifficulty implements consensus.Engine. There is no difficulty rules here
func (s *SystemContract) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	return nil
}

// APIs implements consensus.Engine, returning the user facing RPC API to allow
// controlling the signer voting.
func (s *SystemContract) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return []rpc.API{{
		// note: cannot use underscore in namespace,
		// but overlap with another module's name space works fine.
		Namespace: "scroll",
		Version:   "1.0",
		Service:   &API{system_contract: s},
		Public:    false,
	}}
}

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
		header.MixDigest,
		header.Nonce,
	}
	if header.BaseFee != nil {
		enc = append(enc, header.BaseFee)
	}
	if header.WithdrawalsHash != nil {
		panic("unexpected withdrawal hash value")
	}
	if err := rlp.Encode(w, enc); err != nil {
		panic("can't encode: " + err.Error())
	}
}
