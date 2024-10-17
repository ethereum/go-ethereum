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
	"errors"
	"fmt"
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/engines/engine_v1"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/engines/engine_v2"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/event"

	"github.com/XinFinOrg/XDPoSChain/consensus/clique"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rpc"
	lru "github.com/hashicorp/golang-lru"
)

const (
	ExtraFieldCheck     = true
	SkipExtraFieldCheck = false
)

func (x *XDPoS) SigHash(header *types.Header) (hash common.Hash) {
	switch x.config.BlockConsensusVersion(header.Number, header.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return x.EngineV2.SignHash(header)
	default: // Default "v1"
		return x.EngineV1.SigHash(header)
	}
}

// XDPoS is the delegated-proof-of-stake consensus engine proposed to support the
// Ethereum testnet following the Ropsten attacks.
type XDPoS struct {
	config *params.XDPoSConfig // Consensus engine configuration parameters
	db     ethdb.Database      // Database to store and retrieve snapshot checkpoints

	// Transaction cache, only make sense for adaptor level
	signingTxsCache *lru.Cache

	// Share Channel
	MinePeriodCh chan int // Miner wait Period Channel

	// Trading and lending service
	GetXDCXService    func() utils.TradingService
	GetLendingService func() utils.LendingService

	// The exact consensus engine with different versions
	EngineV1 *engine_v1.XDPoS_v1
	EngineV2 *engine_v2.XDPoS_v2
}

// Subscribe to consensus engines forensics events. Currently only exist for engine v2
func (x *XDPoS) SubscribeForensicsEvent(ch chan<- types.ForensicsEvent) event.Subscription {
	return x.EngineV2.ForensicsProcessor.SubscribeForensicsEvent(ch)
}

// New creates a XDPoS delegated-proof-of-stake consensus engine with the initial
// signers set to the ones provided by the user.
func New(chainConfig *params.ChainConfig, db ethdb.Database) *XDPoS {
	log.Info("[New] initialise consensus engines")
	config := chainConfig.XDPoS
	// Set any missing consensus parameters to their defaults
	if config.Epoch == 0 {
		config.Epoch = utils.EpochLength
	}

	// For testing and testing project, default to mainnet config
	if config.V2 == nil {
		config.V2 = &params.V2{
			SwitchBlock:   params.XDCMainnetChainConfig.XDPoS.V2.SwitchBlock,
			CurrentConfig: params.MainnetV2Configs[0],
			AllConfigs:    params.MainnetV2Configs,
		}
	}

	if config.V2.SwitchBlock.Uint64()%config.Epoch != 0 {
		panic(fmt.Sprintf("v2 switch number is not epoch switch block %d, epoch %d", config.V2.SwitchBlock.Uint64(), config.Epoch))
	}

	log.Info("xdc config loading", "v2 config", config.V2)

	minePeriodCh := make(chan int)

	// Allocate the snapshot caches and create the engine
	signingTxsCache, _ := lru.New(utils.BlockSignersCacheLimit)

	return &XDPoS{
		config: config,
		db:     db,

		MinePeriodCh: minePeriodCh,

		signingTxsCache: signingTxsCache,
		EngineV1:        engine_v1.New(chainConfig, db),
		EngineV2:        engine_v2.New(chainConfig, db, minePeriodCh),
	}
}

// NewFullFaker creates an ethash consensus engine with a full fake scheme that
// accepts all blocks as valid, without checking any consensus rules whatsoever.
func NewFaker(db ethdb.Database, chainConfig *params.ChainConfig) *XDPoS {
	var fakeEngine *XDPoS
	// Set any missing consensus parameters to their defaults
	conf := params.TestXDPoSMockChainConfig.XDPoS
	if chainConfig != nil {
		conf = chainConfig.XDPoS
	}

	minePeriodCh := make(chan int)

	// Allocate the snapshot caches and create the engine
	signingTxsCache, _ := lru.New(utils.BlockSignersCacheLimit)

	fakeEngine = &XDPoS{
		config: conf,
		db:     db,

		MinePeriodCh: minePeriodCh,

		GetXDCXService:    func() utils.TradingService { return nil },
		GetLendingService: func() utils.LendingService { return nil },

		signingTxsCache: signingTxsCache,
		EngineV1:        engine_v1.NewFaker(db, chainConfig),
		EngineV2:        engine_v2.New(chainConfig, db, minePeriodCh),
	}
	return fakeEngine
}

// Reset parameters after checkpoint due to config may change
func (x *XDPoS) UpdateParams(header *types.Header) {
	switch x.config.BlockConsensusVersion(header.Number, header.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		x.EngineV2.UpdateParams(header)
		return
	default: // Default "v1"
		return
	}
}

func (x *XDPoS) Initial(chain consensus.ChainReader, header *types.Header) error {
	switch x.config.BlockConsensusVersion(header.Number, header.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return x.EngineV2.Initial(chain, header)
	default: // Default "v1"
		return nil
	}
}

/*
	Eth Consensus engine interface implementation
*/
// APIs implements consensus.Engine, returning the user facing RPC API to allow
// controlling the signer voting.
func (x *XDPoS) APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{{
		Namespace: "XDPoS",
		Version:   "1.0",
		Service:   &API{chain: chain, XDPoS: x},
		Public:    true,
	}}
}

// Author implements consensus.Engine, returning the Ethereum address recovered
// from the signature in the header's extra-data section.
func (x *XDPoS) Author(header *types.Header) (common.Address, error) {
	switch x.config.BlockConsensusVersion(header.Number, header.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return x.EngineV2.Author(header)
	default: // Default "v1"
		return x.EngineV1.Author(header)
	}
}

// VerifyHeader checks whether a header conforms to the consensus rules.
func (x *XDPoS) VerifyHeader(chain consensus.ChainReader, header *types.Header, fullVerify bool) error {
	switch x.config.BlockConsensusVersion(header.Number, header.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return x.EngineV2.VerifyHeader(chain, header, fullVerify)
	default: // Default "v1"
		return x.EngineV1.VerifyHeader(chain, header, fullVerify)
	}
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers. The
// method returns a quit channel to abort the operations and a results channel to
// retrieve the async verifications (the order is that of the input slice).
func (x *XDPoS) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, fullVerifies []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	// Split the headers list into v1 and v2 buckets
	var v1headers []*types.Header
	var v2headers []*types.Header

	for _, header := range headers {
		switch x.config.BlockConsensusVersion(header.Number, header.Extra, ExtraFieldCheck) {
		case params.ConsensusEngineVersion2:
			v2headers = append(v2headers, header)
		default: // Default "v1"
			v1headers = append(v1headers, header)
		}
	}

	if v1headers != nil {
		x.EngineV1.VerifyHeaders(chain, v1headers, fullVerifies, abort, results)
	}
	if v2headers != nil {
		x.EngineV2.VerifyHeaders(chain, v2headers, fullVerifies, abort, results)
	}

	return abort, results
}

// VerifyUncles implements consensus.Engine, always returning an error for any
// uncles as this consensus mechanism doesn't permit uncles.
func (x *XDPoS) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	switch x.config.BlockConsensusVersion(block.Number(), block.Extra(), ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return x.EngineV2.VerifyUncles(chain, block)
	default: // Default "v1"
		return x.EngineV1.VerifyUncles(chain, block)
	}
}

// VerifySeal implements consensus.Engine, checking whether the signature contained
// in the header satisfies the consensus protocol requirements.
func (x *XDPoS) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	switch x.config.BlockConsensusVersion(header.Number, header.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return nil
	default: // Default "v1"
		return x.EngineV1.VerifySeal(chain, header)
	}
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (x *XDPoS) Prepare(chain consensus.ChainReader, header *types.Header) error {
	switch x.config.BlockConsensusVersion(header.Number, nil, SkipExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return x.EngineV2.Prepare(chain, header)
	default: // Default "v1"
		return x.EngineV1.Prepare(chain, header)
	}
}

// Finalize implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given, and returns the final block.
func (x *XDPoS) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, parentState *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	switch x.config.BlockConsensusVersion(header.Number, header.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return x.EngineV2.Finalize(chain, header, state, parentState, txs, uncles, receipts)
	default: // Default "v1"
		return x.EngineV1.Finalize(chain, header, state, parentState, txs, uncles, receipts)
	}
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
func (x *XDPoS) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	switch x.config.BlockConsensusVersion(block.Number(), block.Extra(), ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return x.EngineV2.Seal(chain, block, stop)
	default: // Default "v1"
		return x.EngineV1.Seal(chain, block, stop)
	}
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have based on the previous blocks in the chain and the
// current signer.
func (x *XDPoS) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	switch x.config.BlockConsensusVersion(parent.Number, parent.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return x.EngineV2.CalcDifficulty(chain, time, parent)
	default: // Default "v1"
		return x.EngineV1.CalcDifficulty(chain, time, parent)
	}
}

func (x *XDPoS) HandleProposedBlock(chain consensus.ChainReader, header *types.Header) error {
	switch x.config.BlockConsensusVersion(header.Number, header.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return x.EngineV2.ProposedBlockHandler(chain, header)
	default: // Default "v1"
		return nil
	}
}

/*
	XDC specific methods
*/

// Authorize injects a private key into the consensus engine to mint new blocks
// with.
func (x *XDPoS) Authorize(signer common.Address, signFn clique.SignerFn) {
	// Authorize each consensus individually
	x.EngineV1.Authorize(signer, signFn)
	x.EngineV2.Authorize(signer, signFn)
}

func (x *XDPoS) GetPeriod() uint64 {
	return x.config.Period
}

func (x *XDPoS) IsAuthorisedAddress(chain consensus.ChainReader, header *types.Header, address common.Address) bool {
	switch x.config.BlockConsensusVersion(header.Number, header.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return x.EngineV2.IsAuthorisedAddress(chain, header, address)
	default: // Default "v1"
		return x.EngineV1.IsAuthorisedAddress(chain, header, address)
	}
}

func (x *XDPoS) GetMasternodes(chain consensus.ChainReader, header *types.Header) []common.Address {
	switch x.config.BlockConsensusVersion(header.Number, header.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return x.EngineV2.GetMasternodes(chain, header)
	default: // Default "v1"
		return x.EngineV1.GetMasternodes(chain, header)
	}
}

func (x *XDPoS) GetMasternodesByNumber(chain consensus.ChainReader, blockNumber uint64) []common.Address {
	blockHeader := chain.GetHeaderByNumber(blockNumber)
	if blockHeader == nil {
		log.Error("[GetMasternodesByNumber] Unable to find block", "Num", blockNumber)
		return []common.Address{}
	}
	switch x.config.BlockConsensusVersion(big.NewInt(int64(blockNumber)), blockHeader.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return x.EngineV2.GetMasternodes(chain, blockHeader)
	default: // Default "v1"
		return x.EngineV1.GetMasternodes(chain, blockHeader)
	}
}

func (x *XDPoS) YourTurn(chain consensus.ChainReader, parent *types.Header, signer common.Address) (bool, error) {
	switch x.config.BlockConsensusVersion(big.NewInt(parent.Number.Int64()+1), nil, SkipExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return x.EngineV2.YourTurn(chain, parent, signer)
	default: // Default "v1"
		return x.EngineV1.YourTurn(chain, parent, signer)
	}
}

func (x *XDPoS) GetValidator(creator common.Address, chain consensus.ChainReader, header *types.Header) (common.Address, error) {
	switch x.config.BlockConsensusVersion(header.Number, header.Extra, ExtraFieldCheck) {
	default: // Default "v1", v2 does not need this function
		return x.EngineV1.GetValidator(creator, chain, header)
	}
}

func (x *XDPoS) UpdateMasternodes(chain consensus.ChainReader, header *types.Header, ms []utils.Masternode) error {
	// fmt.Println("UpdateMasternodes")
	switch x.config.BlockConsensusVersion(header.Number, header.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return x.EngineV2.UpdateMasternodes(chain, header, ms)
	default: // Default "v1"
		return x.EngineV1.UpdateMasternodes(chain, header, ms)
	}
}

func (x *XDPoS) RecoverSigner(header *types.Header) (common.Address, error) {
	switch x.config.BlockConsensusVersion(header.Number, header.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return common.Address{}, nil
	default: // Default "v1"
		return x.EngineV1.RecoverSigner(header)
	}
}

func (x *XDPoS) RecoverValidator(header *types.Header) (common.Address, error) {
	switch x.config.BlockConsensusVersion(header.Number, header.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return common.Address{}, nil
	default: // Default "v1"
		return x.EngineV1.RecoverValidator(header)
	}
}

// Get master nodes over extra data of previous checkpoint block.
func (x *XDPoS) GetMasternodesFromCheckpointHeader(checkpointHeader *types.Header) []common.Address {
	switch x.config.BlockConsensusVersion(checkpointHeader.Number, checkpointHeader.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return x.EngineV2.GetMasternodesFromEpochSwitchHeader(checkpointHeader)
	default: // Default "v1"
		return x.EngineV1.GetMasternodesFromCheckpointHeader(checkpointHeader)
	}
}

// Check is epoch switch (checkpoint) block
func (x *XDPoS) IsEpochSwitch(header *types.Header) (bool, uint64, error) {
	switch x.config.BlockConsensusVersion(header.Number, header.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return x.EngineV2.IsEpochSwitch(header)
	default: // Default "v1"
		return x.EngineV1.IsEpochSwitch(header)
	}
}

func (x *XDPoS) GetCurrentEpochSwitchBlock(chain consensus.ChainReader, blockNumber *big.Int) (uint64, uint64, error) {
	header := chain.GetHeaderByNumber(blockNumber.Uint64())
	switch x.config.BlockConsensusVersion(blockNumber, header.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return x.EngineV2.GetCurrentEpochSwitchBlock(chain, blockNumber)
	default: // Default "v1"
		return x.EngineV1.GetCurrentEpochSwitchBlock(blockNumber)
	}
}

func (x *XDPoS) CalculateMissingRounds(chain consensus.ChainReader, header *types.Header) (*utils.PublicApiMissedRoundsMetadata, error) {
	switch x.config.BlockConsensusVersion(header.Number, header.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return x.EngineV2.CalculateMissingRounds(chain, header)
	default: // Default "v1"
		return nil, errors.New("Not supported in the v1 consensus")
	}
}

// Same DB across all consensus engines
func (x *XDPoS) GetDb() ethdb.Database {
	return x.db
}

func (x *XDPoS) GetSnapshot(chain consensus.ChainReader, header *types.Header) (*utils.PublicApiSnapshot, error) {
	switch x.config.BlockConsensusVersion(header.Number, header.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		sp, err := x.EngineV2.GetSnapshot(chain, header)
		if err != nil {
			return nil, err
		}
		return &utils.PublicApiSnapshot{
			Number:  sp.Number,
			Hash:    sp.Hash,
			Signers: sp.GetMappedCandidates(),
		}, err
	default: // Default "v1"
		sp, err := x.EngineV1.GetSnapshot(chain, header)
		if err != nil {
			return nil, err
		}
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

func (x *XDPoS) GetAuthorisedSignersFromSnapshot(chain consensus.ChainReader, header *types.Header) ([]common.Address, error) {
	switch x.config.BlockConsensusVersion(header.Number, header.Extra, ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		return x.EngineV2.GetSignersFromSnapshot(chain, header)
	default: // Default "v1"
		return x.EngineV1.GetAuthorisedSignersFromSnapshot(chain, header)
	}
}

func (x *XDPoS) FindParentBlockToAssign(chain consensus.ChainReader, currentBlock *types.Block) *types.Block {
	switch x.config.BlockConsensusVersion(currentBlock.Number(), currentBlock.Extra(), ExtraFieldCheck) {
	case params.ConsensusEngineVersion2:
		block := x.EngineV2.FindParentBlockToAssign(chain)
		if block == nil {
			return currentBlock
		}
		return block
	default: // Default "v1"
		return currentBlock
	}
}

/**
Caching
*/

// Cache signing transaction data into BlockSingers cache object
func (x *XDPoS) CacheNoneTIPSigningTxs(header *types.Header, txs []*types.Transaction, receipts []*types.Receipt) []*types.Transaction {
	signTxs := []*types.Transaction{}
	for _, tx := range txs {
		if tx.IsSigningTransaction() {
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
	x.signingTxsCache.Add(header.Hash(), signTxs)

	return signTxs
}

// Cache
func (x *XDPoS) CacheSigningTxs(hash common.Hash, txs []*types.Transaction) []*types.Transaction {
	signTxs := []*types.Transaction{}
	for _, tx := range txs {
		if tx.IsSigningTransaction() {
			signTxs = append(signTxs, tx)
		}
	}
	log.Debug("Save tx signers to cache", "hash", hash.String(), "len(txs)", len(signTxs))
	x.signingTxsCache.Add(hash, signTxs)
	return signTxs
}

func (x *XDPoS) GetCachedSigningTxs(hash common.Hash) (interface{}, bool) {
	return x.signingTxsCache.Get(hash)
}

func (x *XDPoS) GetEpochSwitchInfoBetween(chain consensus.ChainReader, begin, end *types.Header) ([]*types.EpochSwitchInfo, error) {
	beginBlockVersion := x.config.BlockConsensusVersion(begin.Number, begin.Extra, ExtraFieldCheck)
	endBlockVersion := x.config.BlockConsensusVersion(end.Number, end.Extra, ExtraFieldCheck)
	if beginBlockVersion == params.ConsensusEngineVersion2 && endBlockVersion == params.ConsensusEngineVersion2 {
		return x.EngineV2.GetEpochSwitchInfoBetween(chain, begin, end)
	}
	// Default "v1"
	return nil, errors.New("not supported in the v1 consensus")
}
