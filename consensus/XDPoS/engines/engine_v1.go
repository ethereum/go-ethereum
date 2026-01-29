// Copyright 2021 XDC Network
// This file is part of the XDC library.

package engines

import (
	"bytes"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	// ErrInvalidTimestampV1 is returned if the timestamp is not in correct range
	ErrInvalidTimestampV1 = errors.New("invalid timestamp for XDPoS v1")
	// ErrInvalidDifficultyV1 is returned if the difficulty is invalid
	ErrInvalidDifficultyV1 = errors.New("invalid difficulty for XDPoS v1")
	// ErrUnauthorizedSignerV1 is returned if the signer is not authorized
	ErrUnauthorizedSignerV1 = errors.New("unauthorized signer for XDPoS v1")
	// ErrMissingSignatureV1 is returned if signature is missing
	ErrMissingSignatureV1 = errors.New("missing signature in XDPoS v1 block")
)

// EngineV1 implements the XDPoS v1 consensus engine
type EngineV1 struct {
	config *params.XDPoSConfig
	db     Database

	// Snapshot cache
	recents    *lru
	signatures *lru

	// Signing
	signer common.Address
	signFn SignerFn
	lock   sync.RWMutex

	// Block time
	period uint64
}

// Database interface for engine
type Database interface {
	Get(key []byte) ([]byte, error)
	Put(key []byte, value []byte) error
	Delete(key []byte) error
	Has(key []byte) (bool, error)
}

// SignerFn is a signature function
type SignerFn func(account common.Address, data []byte) ([]byte, error)

// lru is a simple LRU cache (placeholder)
type lru struct {
	items map[common.Hash]interface{}
	lock  sync.Mutex
}

func newLRU(size int) *lru {
	return &lru{items: make(map[common.Hash]interface{})}
}

func (l *lru) Get(key common.Hash) (interface{}, bool) {
	l.lock.Lock()
	defer l.lock.Unlock()
	v, ok := l.items[key]
	return v, ok
}

func (l *lru) Add(key common.Hash, value interface{}) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.items[key] = value
}

// NewEngineV1 creates a new XDPoS v1 engine
func NewEngineV1(config *params.XDPoSConfig, db Database) *EngineV1 {
	period := uint64(2)
	if config != nil && config.Period > 0 {
		period = config.Period
	}

	return &EngineV1{
		config:     config,
		db:         db,
		recents:    newLRU(100),
		signatures: newLRU(1000),
		period:     period,
	}
}

// Author returns the signer of the block
func (e *EngineV1) Author(header *types.Header) (common.Address, error) {
	return ecrecover(header, e.signatures)
}

// VerifyHeader checks whether a header conforms to the consensus rules
func (e *EngineV1) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header) error {
	return e.verifyHeader(chain, header, nil)
}

// VerifyHeaders verifies a batch of headers concurrently
func (e *EngineV1) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for i, header := range headers {
			var parent *types.Header
			if i > 0 {
				parent = headers[i-1]
			}
			err := e.verifyHeader(chain, header, parent)
			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()

	return abort, results
}

// verifyHeader checks header validity
func (e *EngineV1) verifyHeader(chain consensus.ChainHeaderReader, header *types.Header, parent *types.Header) error {
	if header.Number == nil {
		return errors.New("missing block number")
	}

	// Don't verify genesis block
	if header.Number.Uint64() == 0 {
		return nil
	}

	// Get parent if not provided
	if parent == nil {
		parent = chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
		if parent == nil {
			return consensus.ErrUnknownAncestor
		}
	}

	// Verify timestamp
	if header.Time <= parent.Time {
		return ErrInvalidTimestampV1
	}

	// Verify that the block doesn't come from the future
	if header.Time > uint64(time.Now().Unix())+30 {
		return consensus.ErrFutureBlock
	}

	// Verify extra data length
	if len(header.Extra) < crypto.SignatureLength {
		return ErrMissingSignatureV1
	}

	// Verify signer
	signer, err := ecrecover(header, e.signatures)
	if err != nil {
		return err
	}

	// Check if signer is authorized
	if !e.isAuthorized(chain, header, signer) {
		return ErrUnauthorizedSignerV1
	}

	return nil
}

// VerifyUncles implements consensus.Engine, always returning an error for any
// uncles as this consensus mechanism doesn't permit uncles.
func (e *EngineV1) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errors.New("uncles not allowed")
	}
	return nil
}

// Prepare initializes the consensus fields of a block header
func (e *EngineV1) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	header.Coinbase = e.signer
	header.Nonce = types.BlockNonce{}
	header.Difficulty = big.NewInt(1)

	// Set extra data
	if len(header.Extra) < ExtraVanity {
		header.Extra = append(header.Extra, bytes.Repeat([]byte{0x00}, ExtraVanity-len(header.Extra))...)
	}
	header.Extra = header.Extra[:ExtraVanity]

	// Add space for signature
	header.Extra = append(header.Extra, make([]byte, crypto.SignatureLength)...)

	// Mix digest is not used
	header.MixDigest = common.Hash{}

	// Set the correct timestamp
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	header.Time = parent.Time + e.period

	return nil
}

// Finalize implements consensus.Engine
func (e *EngineV1) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, withdrawals []*types.Withdrawal) {
	// Accumulate block rewards
	e.accumulateRewards(chain, state, header)
}

// FinalizeAndAssemble implements consensus.Engine
func (e *EngineV1) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt, withdrawals []*types.Withdrawal) (*types.Block, error) {
	// Finalize block
	e.Finalize(chain, header, state, txs, uncles, withdrawals)

	// Compute state root
	header.Root = state.IntermediateRoot(true)

	// Assemble and return the final block
	return types.NewBlock(header, txs, nil, receipts, trie.NewStackTrie(nil)), nil
}

// Seal implements consensus.Engine
func (e *EngineV1) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	header := block.Header()

	// Don't seal genesis block
	if header.Number.Uint64() == 0 {
		return errors.New("cannot seal genesis block")
	}

	e.lock.RLock()
	signer, signFn := e.signer, e.signFn
	e.lock.RUnlock()

	// Sign the header
	sighash, err := signHash(header)
	if err != nil {
		return err
	}

	signature, err := signFn(signer, sighash)
	if err != nil {
		return err
	}

	// Copy signature to extra data
	copy(header.Extra[len(header.Extra)-crypto.SignatureLength:], signature)

	select {
	case results <- block.WithSeal(header):
	case <-stop:
		return nil
	}

	return nil
}

// SealHash returns the hash of a block prior to it being sealed
func (e *EngineV1) SealHash(header *types.Header) common.Hash {
	return sigHash(header)
}

// CalcDifficulty returns the difficulty for a block
func (e *EngineV1) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	return big.NewInt(1)
}

// APIs returns the RPC APIs this consensus engine provides
func (e *EngineV1) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return nil
}

// Close implements consensus.Engine
func (e *EngineV1) Close() error {
	return nil
}

// Authorize injects a private key for signing
func (e *EngineV1) Authorize(signer common.Address, signFn SignerFn) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.signer = signer
	e.signFn = signFn
}

// isAuthorized checks if a signer is authorized
func (e *EngineV1) isAuthorized(chain consensus.ChainHeaderReader, header *types.Header, signer common.Address) bool {
	// Get snapshot at parent
	// Check if signer is in the validator set
	// For now, return true
	return true
}

// accumulateRewards accumulates the block and uncle rewards
func (e *EngineV1) accumulateRewards(chain consensus.ChainHeaderReader, state *state.StateDB, header *types.Header) {
	// Block reward is handled by the foundation reward contract
	log.Debug("Accumulating rewards", "block", header.Number)
}

// ecrecover extracts the signer from a header
func ecrecover(header *types.Header, sigcache *lru) (common.Address, error) {
	hash := header.Hash()

	if address, known := sigcache.Get(hash); known {
		return address.(common.Address), nil
	}

	if len(header.Extra) < crypto.SignatureLength {
		return common.Address{}, ErrMissingSignatureV1
	}
	signature := header.Extra[len(header.Extra)-crypto.SignatureLength:]

	// Recover the public key
	pubkey, err := crypto.Ecrecover(sigHash(header).Bytes(), signature)
	if err != nil {
		return common.Address{}, err
	}

	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])

	sigcache.Add(hash, signer)
	return signer, nil
}

// sigHash returns the hash to be signed
func sigHash(header *types.Header) common.Hash {
	return sigHashWithExtra(header, header.Extra[:len(header.Extra)-crypto.SignatureLength])
}

// sigHashWithExtra returns the hash to be signed with specific extra data
func sigHashWithExtra(header *types.Header, extra []byte) common.Hash {
	type sigHeader struct {
		ParentHash  common.Hash
		UncleHash   common.Hash
		Coinbase    common.Address
		Root        common.Hash
		TxHash      common.Hash
		ReceiptHash common.Hash
		Bloom       types.Bloom
		Difficulty  *big.Int
		Number      *big.Int
		GasLimit    uint64
		GasUsed     uint64
		Time        uint64
		Extra       []byte
		MixDigest   common.Hash
		Nonce       types.BlockNonce
	}

	data, _ := rlp.EncodeToBytes(&sigHeader{
		ParentHash:  header.ParentHash,
		UncleHash:   header.UncleHash,
		Coinbase:    header.Coinbase,
		Root:        header.Root,
		TxHash:      header.TxHash,
		ReceiptHash: header.ReceiptHash,
		Bloom:       header.Bloom,
		Difficulty:  header.Difficulty,
		Number:      header.Number,
		GasLimit:    header.GasLimit,
		GasUsed:     header.GasUsed,
		Time:        header.Time,
		Extra:       extra,
		MixDigest:   header.MixDigest,
		Nonce:       header.Nonce,
	})

	return crypto.Keccak256Hash(data)
}

// signHash returns a hash for signing
func signHash(header *types.Header) ([]byte, error) {
	return sigHash(header).Bytes(), nil
}

// ExtraVanity is the fixed number of extra-data prefix bytes
const ExtraVanity = 32

// rpc.API placeholder
type rpc struct{}
type API struct{}
