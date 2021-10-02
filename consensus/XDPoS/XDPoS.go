// Copyright (c) 2021 XDPoSChain
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

// Package XDPoS is the adaptor for different consensus engine.
package XDPoS

import (
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/engines/engine_v1"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/consensus/clique"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rpc"
	lru "github.com/hashicorp/golang-lru"
)

func SigHash(header *types.Header) (hash common.Hash) {
	switch params.BlockConsensusVersion(header.Number) {
	// TODO: Add switch case for 2.0 later
	default: // Default "1.0"
		return engine_v1.SigHash(header)
	}
}

// XDPoS is the delegated-proof-of-stake consensus engine proposed to support the
// Ethereum testnet following the Ropsten attacks.
type XDPoS struct {
	config *params.XDPoSConfig // Consensus engine configuration parameters
	db     ethdb.Database      // Database to store and retrieve snapshot checkpoints

	BlockSigners               *lru.Cache
	HookReward                 func(chain consensus.ChainReader, state *state.StateDB, parentState *state.StateDB, header *types.Header) (error, map[string]interface{})
	HookPenalty                func(chain consensus.ChainReader, blockNumberEpoc uint64) ([]common.Address, error)
	HookPenaltyTIPSigning      func(chain consensus.ChainReader, header *types.Header, candidate []common.Address) ([]common.Address, error)
	HookValidator              func(header *types.Header, signers []common.Address) ([]byte, error)
	HookVerifyMNs              func(header *types.Header, signers []common.Address) error
	GetXDCXService             func() utils.TradingService
	GetLendingService          func() utils.LendingService
	HookGetSignersFromContract func(blockHash common.Hash) ([]common.Address, error)

	EngineV1 engine_v1.XDPoS_v1
}

// New creates a XDPoS delegated-proof-of-stake consensus engine with the initial
// signers set to the ones provided by the user.
func New(config *params.XDPoSConfig, db ethdb.Database) *XDPoS {
	// Set any missing consensus parameters to their defaults
	conf := *config
	if conf.Epoch == 0 {
		conf.Epoch = utils.EpochLength
	}

	// Allocate the snapshot caches and create the engine
	BlockSigners, _ := lru.New(utils.BlockSignersCacheLimit)

	return &XDPoS{
		config: &conf,

		BlockSigners: BlockSigners,
		EngineV1:     *engine_v1.New(&conf, db),
	}
}

// NewFullFaker creates an ethash consensus engine with a full fake scheme that
// accepts all blocks as valid, without checking any consensus rules whatsoever.

/*
	Eth Consensus engine interface implementation
*/
// APIs implements consensus.Engine, returning the user facing RPC API to allow
// controlling the signer voting.
func (c *XDPoS) APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{{
		Namespace: "XDPoS",
		Version:   "1.0",
		Service:   &API{chain: chain, XDPoS: c},
		Public:    true,
	}}
}

// Author implements consensus.Engine, returning the Ethereum address recovered
// from the signature in the header's extra-data section.
func (c *XDPoS) Author(header *types.Header) (common.Address, error) {
	switch params.BlockConsensusVersion(header.Number) {
	default: // Default "1.0"
		return c.EngineV1.Author(header)
	}
}

// VerifyHeader checks whether a header conforms to the consensus rules.
func (c *XDPoS) VerifyHeader(chain consensus.ChainReader, header *types.Header, fullVerify bool) error {
	switch params.BlockConsensusVersion(header.Number) {
	default: // Default "1.0"
		return c.EngineV1.VerifyHeader(chain, header, fullVerify)
	}
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers. The
// method returns a quit channel to abort the operations and a results channel to
// retrieve the async verifications (the order is that of the input slice).
func (c *XDPoS) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, fullVerifies []bool) (chan<- struct{}, <-chan error) {
	// TODO: (Hashlab) This funciton is a special case
	return c.EngineV1.VerifyHeaders(chain, headers, fullVerifies)
}

// VerifyUncles implements consensus.Engine, always returning an error for any
// uncles as this consensus mechanism doesn't permit uncles.
func (c *XDPoS) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	switch params.BlockConsensusVersion(block.Number()) {
	default: // Default "1.0"
		return c.EngineV1.VerifyUncles(chain, block)
	}
}

// VerifySeal implements consensus.Engine, checking whether the signature contained
// in the header satisfies the consensus protocol requirements.
func (c *XDPoS) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	switch params.BlockConsensusVersion(header.Number) {
	default: // Default "1.0"
		return c.EngineV1.VerifySeal(chain, header)
	}
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (c *XDPoS) Prepare(chain consensus.ChainReader, header *types.Header) error {
	switch params.BlockConsensusVersion(header.Number) {
	default: // Default "1.0"
		return c.EngineV1.Prepare(chain, header)
	}
}

// Finalize implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given, and returns the final block.
func (c *XDPoS) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, parentState *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	switch params.BlockConsensusVersion(header.Number) {
	default: // Default "1.0"
		return c.EngineV1.Finalize(chain, header, state, parentState, txs, uncles, receipts)
	}
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
func (c *XDPoS) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	switch params.BlockConsensusVersion(block.Number()) {
	default: // Default "1.0"
		return c.EngineV1.Seal(chain, block, stop)
	}
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have based on the previous blocks in the chain and the
// current signer.
func (c *XDPoS) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	switch params.BlockConsensusVersion(parent.Number) {
	default: // Default "1.0"
		return c.EngineV1.CalcDifficulty(chain, time, parent)
	}
}

/*
	XDC specific methods
*/

// Authorize injects a private key into the consensus engine to mint new blocks
// with.
func (c *XDPoS) Authorize(signer common.Address, signFn clique.SignerFn) {
	// Authorize each consensus individually
	c.EngineV1.Authorize(signer, signFn)
}

func (c *XDPoS) GetPeriod() uint64 {
	return c.config.Period
}

func (c *XDPoS) IsAuthorisedAddress(header *types.Header, chain consensus.ChainReader, address common.Address) bool {
	switch params.BlockConsensusVersion(header.Number) {
	default: // Default "1.0"
		return c.EngineV1.IsAuthorisedAddress(header, chain, address)
	}
}

func (c *XDPoS) GetMasternodes(chain consensus.ChainReader, header *types.Header) []common.Address {
	switch params.BlockConsensusVersion(header.Number) {
	default: // Default "1.0"
		return c.EngineV1.GetMasternodes(chain, header)
	}
}

func (c *XDPoS) YourTurn(chain consensus.ChainReader, parent *types.Header, signer common.Address) (int, int, int, bool, error) {
	switch params.BlockConsensusVersion(parent.Number) {
	default: // Default "1.0"
		return c.EngineV1.YourTurn(chain, parent, signer)
	}
}

func (c *XDPoS) GetValidator(creator common.Address, chain consensus.ChainReader, header *types.Header) (common.Address, error) {
	switch params.BlockConsensusVersion(header.Number) {
	default: // Default "1.0"
		return c.EngineV1.GetValidator(creator, chain, header)
	}
}

func (c *XDPoS) UpdateMasternodes(chain consensus.ChainReader, header *types.Header, ms []utils.Masternode) error {
	switch params.BlockConsensusVersion(header.Number) {
	default: // Default "1.0"
		return c.EngineV1.UpdateMasternodes(chain, header, ms)
	}
}

func (c *XDPoS) RecoverSigner(header *types.Header) (common.Address, error) {
	switch params.BlockConsensusVersion(header.Number) {
	default: // Default "1.0"
		return c.EngineV1.RecoverSigner(header)
	}
}

func (c *XDPoS) RecoverValidator(header *types.Header) (common.Address, error) {
	switch params.BlockConsensusVersion(header.Number) {
	default: // Default "1.0"
		return c.EngineV1.RecoverValidator(header)
	}
}

// Get master nodes over extra data of previous checkpoint block.
func (c *XDPoS) GetMasternodesFromCheckpointHeader(preCheckpointHeader *types.Header, n, e uint64) []common.Address {
	switch params.BlockConsensusVersion(preCheckpointHeader.Number) {
	default: // Default "1.0"
		return c.EngineV1.GetMasternodesFromCheckpointHeader(preCheckpointHeader, n, e)
	}
}

func (c *XDPoS) CacheData(header *types.Header, txs []*types.Transaction, receipts []*types.Receipt) []*types.Transaction {
	switch params.BlockConsensusVersion(header.Number) {
	default: // Default "1.0"
		return c.EngineV1.CacheData(header, txs, receipts)
	}
}

// Same DB across all consensus engines
func (c *XDPoS) GetDb() ethdb.Database {
	return c.db
}

func (c *XDPoS) GetSnapshot(chain consensus.ChainReader, header *types.Header) (*utils.PublicApiSnapshot, error) {
	switch params.BlockConsensusVersion(header.Number) {
	default: // Default "1.0"
		sp, err := c.EngineV1.GetSnapshot(chain, header)
		// Convert to a standard PublicApiSnapshot type, otherwise it's a breaking change to API
		return &utils.PublicApiSnapshot{
			Number:  sp.Number,
			Hash:    sp.Hash,
			Signers: sp.Signers,
			Recents: sp.Recents,
			Votes:   sp.Votes,
			Tally:   sp.Tally,
		}, err
	}
}

func (c *XDPoS) GetAuthorisedSignersFromSnapshot(chain consensus.ChainReader, header *types.Header) ([]common.Address, error) {
	switch params.BlockConsensusVersion(header.Number) {
	default: // Default "1.0"
		return c.EngineV1.GetAuthorisedSignersFromSnapshot(chain, header)
	}
}

// TODO: (Hashlab) Can be further refactored
func (c *XDPoS) CheckMNTurn(chain consensus.ChainReader, parent *types.Header, signer common.Address) bool {
	switch params.BlockConsensusVersion(parent.Number) {
	default: // Default "1.0"
		return c.EngineV1.CheckMNTurn(chain, parent, signer)
	}
}

// TODO: (Hashlab) Need further work on refactor this method
func (c *XDPoS) CacheSigner(hash common.Hash, txs []*types.Transaction) []*types.Transaction {
	return c.EngineV1.CacheSigner(hash, txs)
}

// TODO: (Hashlab)Get signer coinbase
func (c *XDPoS) Signer() common.Address {
	return c.EngineV1.Signer()
}
