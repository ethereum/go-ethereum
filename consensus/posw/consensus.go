// Copyright 2026 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package posw

import (
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var errOlderBlockTime = errors.New("timestamp must be greater than parent")

// PoSW implements shard Ethash consensus with a parent-state staking discount.
// MixDigest carries the base difficulty; geth's Difficulty and Nonce stay zero.
// Header checks need no state; VerifySeal applies the discount at body import.
type PoSW struct {
	config   *params.ChainConfig          // Validated when the engine is constructed.
	gasLimit uint64                       // Fixed shard gas limit copied from genesis, matching pyquarkchain.
	cacheMu  sync.Mutex                   // Serializes cache creation, not hashimoto verification.
	caches   *lru.Cache[uint64, []uint32] // Immutable verification data, keyed by Ethash epoch.
}

var _ consensus.Engine = (*PoSW)(nil)
var _ consensus.SealVerifier = (*PoSW)(nil)

// New validates the shard configuration and saves the genesis gas limit.
// The caller supplies genesis so engine creation does not require a running chain.
func New(config *params.ChainConfig, genesis *types.Header) (*PoSW, error) {
	if config == nil || config.QuarkChain == nil {
		return nil, errors.New("posw: missing shard configuration")
	}
	if err := config.CheckQuarkChain(); err != nil {
		return nil, err
	}
	if genesis == nil {
		return nil, errors.New("posw: missing shard genesis header")
	}
	if genesis.GasLimit > params.MaxGasLimit {
		return nil, fmt.Errorf("invalid genesis gas limit: have %d, max %d", genesis.GasLimit, params.MaxGasLimit)
	}
	return &PoSW{config: config, gasLimit: genesis.GasLimit, caches: lru.NewCache[uint64, []uint32](2)}, nil
}

// Author returns the coinbase receiving this shard block's subsidy and fees.
func (p *PoSW) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

// VerifyHeader checks a non-genesis header without reading state or verifying its seal.
func (p *PoSW) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header) error {
	if header.Number == nil || !header.Number.IsUint64() || header.Number.Sign() == 0 {
		return consensus.ErrInvalidNumber
	}
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	return p.verifyHeader(chain, header, parent, uint64(time.Now().Unix()))
}

// VerifyHeaders checks a contiguous batch of non-genesis headers, as in Ethash.
// Parent state may not exist yet; state-dependent seals are checked in ValidateBody.
func (p *PoSW) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))
	if len(headers) == 0 {
		return abort, results
	}
	now := uint64(time.Now().Unix())

	go func() {
		for i, header := range headers {
			var parent *types.Header
			var err error
			if header.Number == nil || !header.Number.IsUint64() || header.Number.Sign() == 0 {
				err = consensus.ErrInvalidNumber
			} else if i == 0 {
				parent = chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
			} else if headers[i-1].Hash() == header.ParentHash {
				parent = headers[i-1]
			}

			if err == nil && parent == nil {
				err = consensus.ErrUnknownAncestor
			}
			if err == nil {
				err = p.verifyHeader(chain, header, parent, now)
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

// verifyHeader checks local fields, then configuration and parent-dependent rules.
// Root references and shard fork choice belong to ShardChain; body commitments,
// state-dependent seals and execution results are checked during full import.
func (p *PoSW) verifyHeader(chain consensus.ChainHeaderReader, header, parent *types.Header, now uint64) error {
	if err := header.SanityCheckQKC(); err != nil {
		return err
	}
	extra, _ := header.QKCExtra() // Format and version were checked above.
	if extra.ChainID.Cmp(p.config.ChainID) != 0 {
		return fmt.Errorf("invalid chain ID: have %d, want %d", extra.ChainID, p.config.ChainID)
	}
	// Match pyquarkchain: every shard block uses its genesis gas limit.
	if header.GasLimit != p.gasLimit {
		return fmt.Errorf("invalid gas limit: have %d, want %d", header.GasLimit, p.gasLimit)
	}
	if want := header.GasLimit / params.XShardGasLimitDivisor; extra.XShardGasLimit != want {
		return fmt.Errorf("invalid cross-shard gas limit: have %d, want %d", extra.XShardGasLimit, want)
	}
	// A parent from the same batch may have failed its own header checks.
	if parent.Number == nil || new(big.Int).Sub(header.Number, parent.Number).Cmp(common.Big1) != 0 {
		return consensus.ErrInvalidNumber
	}
	// Check time before calculating difficulty to avoid unsigned subtraction wrapping.
	if header.Time <= parent.Time {
		return errOlderBlockTime
	}
	if header.Time > now+15 {
		return consensus.ErrFutureBlock
	}
	if p.CalcDifficulty(chain, header.Time, parent).Cmp(header.QKCDifficulty()) != 0 {
		return errInvalidDifficulty
	}
	// TODO: BaseFee support follows ChainConfig's London setting for now;
	// revisit this check if QKC needs a different BaseFee policy.
	config := chain.Config()
	if config.IsLondon(header.Number) {
		// Check BaseFee separately: Ethereum's gas-limit adjustment, including
		// the London activation increase, does not apply to QKC's fixed limit.
		if header.BaseFee == nil {
			return errors.New("header is missing baseFee")
		}
		if config.IsLondon(parent.Number) && parent.BaseFee == nil {
			return errors.New("parent header is missing baseFee")
		}
		// Batch verification can supply an invalid parent. CalcBaseFee divides
		// by its gas target once London is active, so that target must be positive.
		if config.IsLondon(parent.Number) && parent.GasLimit/config.ElasticityMultiplier() == 0 {
			return fmt.Errorf("invalid parent gas limit: %d", parent.GasLimit)
		}
		if want := eip1559.CalcBaseFee(config, parent); header.BaseFee.Cmp(want) != 0 {
			return fmt.Errorf("invalid baseFee: have %s, want %s", header.BaseFee, want)
		}
	} else {
		if header.BaseFee != nil {
			return errors.New("base fee before London")
		}
	}
	// Preserve fork-dependent header encoding. QKC requires empty withdrawals
	// and zero blob/beacon fields because it has no corresponding data sources.
	if config.IsShanghai(header.Number, header.Time) != (header.WithdrawalsHash != nil) {
		return errors.New("invalid withdrawals hash presence")
	}
	if header.WithdrawalsHash != nil && *header.WithdrawalsHash != types.EmptyWithdrawalsHash {
		return errors.New("qkc withdrawals must be empty")
	}
	cancun := config.IsCancun(header.Number, header.Time)
	if cancun != (header.ParentBeaconRoot != nil) {
		return errors.New("invalid parent beacon root presence")
	}
	if header.ParentBeaconRoot != nil && *header.ParentBeaconRoot != (common.Hash{}) {
		return errors.New("qkc parent beacon root must be zero")
	}
	if cancun != (header.BlobGasUsed != nil) {
		return errors.New("invalid blob gas used presence")
	}
	// TODO: Also disable blob transactions in the transaction pool to match this consensus restriction.
	if header.BlobGasUsed != nil && *header.BlobGasUsed != 0 {
		return errors.New("qkc blob gas used must be zero")
	}
	if cancun != (header.ExcessBlobGas != nil) {
		return errors.New("invalid excess blob gas presence")
	}
	if header.ExcessBlobGas != nil && *header.ExcessBlobGas != 0 {
		return errors.New("qkc excess blob gas must be zero")
	}
	if config.IsPrague(header.Number, header.Time) != (header.RequestsHash != nil) {
		return errors.New("invalid requests hash presence")
	}
	amsterdam := config.IsAmsterdam(header.Number, header.Time)
	if amsterdam != (header.BlockAccessListHash != nil) {
		return errors.New("invalid block access list hash presence")
	}
	if amsterdam != (header.SlotNumber != nil) {
		return errors.New("invalid slot number presence")
	}
	if header.SlotNumber != nil && *header.SlotNumber != 0 {
		return errors.New("qkc slot number must be zero")
	}
	// RequestsHash and BlockAccessListHash commit to execution results;
	// their contents are checked during full block import.
	return nil
}

// VerifyUncles reports unexpected uncles. Header and body commitment checks
// still require an empty uncle list, and Finalize pays no uncle rewards.
func (p *PoSW) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) != 0 {
		log.Warn("Shard block contains uncles", "hash", block.Hash(), "uncles", len(block.Uncles()))
	}
	return nil
}

// CalcDifficulty returns the base PoW difficulty, before the staking discount.
// Header validation and Prepare must first check time > parent.Time.
func (p *PoSW) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	return CalcDifficulty(p.config.QuarkChain.PoSW, time, parent.Time, parent.QKCDifficulty())
}

// Prepare writes the base difficulty before execution, since the EVM also sees
// MixDigest as PREVRANDAO. Staking changes the seal target, not these header fields.
// It fixes the candidate's gas limit to the shard genesis value.
func (p *PoSW) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	extra, err := header.QKCExtra()
	if err != nil {
		return err
	}
	if extra.Version != 0 || extra.ChainID.Cmp(p.config.ChainID) != 0 {
		return fmt.Errorf("invalid qkc version or shard: version %d, shard %d", extra.Version, extra.ChainID)
	}
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	if header.Time <= parent.Time {
		return errOlderBlockTime
	}
	difficulty := p.CalcDifficulty(chain, header.Time, parent)
	if difficulty.Sign() <= 0 || difficulty.BitLen() > 256 {
		return errInvalidDifficulty
	}
	header.GasLimit = p.gasLimit
	header.MixDigest = common.BigToHash(difficulty)
	header.Difficulty, header.Nonce = new(big.Int), types.BlockNonce{}
	// Clear any existing seal before preparing new mining work.
	extra.Nonce, extra.MixHash = types.BlockNonce{}, common.Hash{}
	header.Extra = extra.Encode()
	return nil
}

// Finalize credits the subsidy after separately rounding epoch decay and reward
// tax, as in pyquarkchain. Transaction fees are accounted for elsewhere.
func (p *PoSW) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state vm.StateDB, body *types.Body) {
	reward := p.config.QuarkChain.PoSW.BlockRewardAt(header.Number)
	reward = p.config.QuarkChain.AfterTax(reward)
	if reward.Sign() != 0 {
		state.AddBalance(header.Coinbase, uint256.MustFromBig(reward), tracing.BalanceIncreaseRewardMineBlock)
	}
}

// SealHash excludes QKCExtra's nonce and mix hash, while retaining base difficulty.
// The interface cannot return errors: malformed Extra yields zero here, but
// header validation rejects it and VerifySeal propagates the QKCSealHash error.
func (p *PoSW) SealHash(header *types.Header) common.Hash {
	hash, _ := header.QKCSealHash()
	return hash
}

// Seal is reserved for the shard mining implementation.
func (p *PoSW) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	// TODO(Lun): Implement shard mining and sealing.
	return errors.New("posw: sealing not implemented")
}

// Close releases cached verification data. In-flight verification retains its
// own slice references, so clearing the cache does not invalidate those reads.
func (p *PoSW) Close() error {
	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()
	p.caches.Purge()
	return nil
}
