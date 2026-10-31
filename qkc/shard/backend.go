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

package shard

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posw"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/qkc/rawdb"
	qkctypes "github.com/ethereum/go-ethereum/qkc/types"
	"github.com/holiman/uint256"
)

// ShardBackend owns its blockchain and consensus engine. Its database and cluster
// client are borrowed; their caller stops incoming requests before Stop.
type ShardBackend struct {
	chain         *ShardChain
	engine        consensus.Engine
	clusterClient ClusterClient
	stopOnce      sync.Once
	stopErr       error
}

// New constructs a shard backend without selecting a root tip. chainConfig is
// authoritative; it is attached to a copy of genesis without changing the caller's
// genesis. The same borrowed database stores chain data, roots and received deposits.
func New(chainConfig *params.ChainConfig, genesis *core.Genesis, db ethdb.Database, cache *core.BlockChainConfig, client ClusterClient) (*ShardBackend, error) {
	if chainConfig == nil || chainConfig.QuarkChain == nil {
		return nil, errNotQuarkChain
	}
	if genesis == nil {
		return nil, errors.New("missing shard genesis")
	}
	if db == nil {
		return nil, errors.New("missing shard database")
	}
	if client == nil {
		return nil, errors.New("missing cluster client")
	}
	genesisCopy := *genesis
	genesisCopy.Config = chainConfig
	engine, err := posw.New(chainConfig.QuarkChain, genesisCopy.ToBlock().Header())
	if err != nil {
		return nil, err
	}
	chain, err := core.NewBlockChain(db, &genesisCopy, engine, cache)
	if err != nil {
		engine.Close()
		return nil, err
	}
	return &ShardBackend{chain: NewShardChain(db, chain), engine: engine, clusterClient: client}, nil
}

// Init selects the root checkpoint received from master. Root genesis already
// contains the shard genesis header, so initialization does not report it again.
func (s *ShardBackend) Init(root *qkctypes.RootBlock) error {
	return s.chain.Init(root)
}

// Stop releases owned resources once. The caller retains the database and client.
func (s *ShardBackend) Stop() error {
	s.stopOnce.Do(func() {
		s.chain.chain.Stop()
		s.stopErr = s.engine.Close()
	})
	return s.stopErr
}

// CreateBlockToMine builds an unsealed candidate on the shard head, paying
// coinbase, and returns the adjusted difficulty its seal must meet.
// TODO(Lun): Reuse the shard block builder and parent-state PoSW calculation.
func (s *ShardBackend) CreateBlockToMine(ctx context.Context, coinbase common.Address) (*types.Block, *big.Int, error) {
	return nil, nil, errors.New("CreateBlockToMine not implemented")
}

// InsertMinedBlock imports a sealed candidate through AddShardBlock so mined
// and received blocks follow the same cross-shard delivery and reporting path.
// TODO(Lun): Integrate mined-block insertion with the mining lifecycle.
func (s *ShardBackend) InsertMinedBlock(block *types.Block) error {
	return errors.New("InsertMinedBlock not implemented")
}

// SubscribeChainHeadEvent registers a subscription for shard head changes.
// Until implemented, the subscription reports its error through Err().
// TODO(Lun): Delegate to BlockChain and integrate the subscription with mining.
func (s *ShardBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return event.NewSubscription(func(<-chan struct{}) error {
		return errors.New("SubscribeChainHeadEvent not implemented")
	})
}

// AddRootBlock imports a root and reports whether it became the root tip.
func (s *ShardBackend) AddRootBlock(root *qkctypes.RootBlock) (bool, error) {
	if root == nil {
		return false, fmt.Errorf("%w: nil root block", ErrInvalidRootBlock)
	}
	return s.chain.InsertRootBlock(root)
}

// BroadcastXShardTxList delivers an imported block's deposits to every other
// shard in config, sending an empty list when it has no deposits for that shard.
// Empty lists mark the source block as received so destinations can process a
// root that confirms it. Destinations come from config, not from emitted messages.
// Each ClusterClient call must succeed for all hosts serving that shard. A failure
// stops delivery; retrying repeats all deliveries by block hash.
func (s *ShardBackend) BroadcastXShardTxList(ctx context.Context, blockHash common.Hash) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	deposits, err := XShardDeposits(s.chain.chain, blockHash)
	if err != nil {
		return err
	}
	config := s.chain.chain.Config().QuarkChain
	for _, shardID := range config.ShardIDs {
		if shardID == config.ShardID {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		list := deposits[shardID]
		if list == nil {
			list = []*types.DepositTx{}
		}
		if err := s.clusterClient.BroadcastXShardTxList(ctx, shardID, blockHash, list); err != nil {
			return fmt.Errorf("broadcast block %x deposits to shard %d: %w", blockHash, shardID, err)
		}
	}
	return nil
}

// AddShardBlock imports a block, delivers all cross-shard lists, then reports
// its header to master, matching pyquarkchain's propagation order. This method
// waits for every destination to acknowledge its list, including empty lists,
// before reporting the header, because a root may confirm it afterward.
//
// Import and propagation are not atomic: an error may leave the block stored and
// some recipients notified. A known block must still run both propagation steps
// on retry. Unlike Python's commit marker and per-block futures, no delivery
// progress is tracked here; callers retry and recipients accept duplicates.
func (s *ShardBackend) AddShardBlock(ctx context.Context, block *types.Block) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if block == nil {
		return errors.New("nil shard block")
	}
	if err := s.chain.InsertShardBlock(block); err != nil {
		return err
	}
	if err := s.BroadcastXShardTxList(ctx, block.Hash()); err != nil {
		return err
	}
	// TODO: Replace zero with the actual coinbase amount: reward plus ordinary and cross-shard fees.
	if err := s.clusterClient.SendMinorBlockHeaderToMaster(ctx, block.Header(), new(uint256.Int)); err != nil {
		return fmt.Errorf("report block %x to master: %w", block.Hash(), err)
	}
	return nil
}

// CurrentBlock returns the canonical shard head.
func (s *ShardBackend) CurrentBlock() *types.Header {
	return s.chain.chain.CurrentBlock()
}

// RootTip returns the root-tip hash, or zero before Init, for mining compatibility.
func (s *ShardBackend) RootTip() common.Hash {
	return s.chain.RootTip()
}

// GetRootChainStakes reads stakes at the requested shard block. Only shard zero
// serves root-chain stakes; the blockchain enforces that restriction.
func (s *ShardBackend) GetRootChainStakes(address common.Address, shardBlockHash common.Hash) (*uint256.Int, common.Address, error) {
	header := s.chain.chain.GetHeaderByHash(shardBlockHash)
	if header == nil {
		return nil, common.Address{}, fmt.Errorf("shard block header %s not found", shardBlockHash.Hex())
	}
	return s.chain.chain.GetRootChainStakes(header, address)
}

// AddXshardTxList stores a validated remote block's deposits. An empty list is
// still RLP-encoded and written, distinguishing it from a list not yet received.
// The remote shard validates the list; repeated deliveries overwrite the same key.
func (s *ShardBackend) AddXshardTxList(shardBlockHash common.Hash, deposits []*types.DepositTx) error {
	if err := rawdb.WriteXShardDepositsWithError(s.chain.db, shardBlockHash, deposits); err != nil {
		return fmt.Errorf("store block %x cross-shard deposits: %w", shardBlockHash, err)
	}
	return nil
}

// GetUnconfirmedHeaderList returns the earliest unconfirmed canonical headers
// selected by ShardChain, independently of propagation status.
func (s *ShardBackend) GetUnconfirmedHeaderList() ([]*types.Header, error) {
	return s.chain.GetUnconfirmedHeaderList()
}
