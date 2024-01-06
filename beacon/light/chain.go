// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package light

import (
	"context"
	"fmt"
	"math/big"
	"time"

	eth2spec "github.com/attestantio/go-eth2-client/spec"
	"github.com/ethereum/go-ethereum/beacon/beaclient"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	ctypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

// LightClient tracks the head of the chain using the light client protocol,
// which assumes the majority of beacon chain sync committee is honest.
type LightClient struct {
	beacon *beaclient.Client
	store  *store

	chainHeadFeed event.Feed
	quitCh        chan struct{}
}

// Bootstrap retrieves a light client bootstrap and authenticates it against the
// provided trusted root.
func Bootstrap(server string, headers []string, root common.Hash) (*LightClient, error) {
	api, err := beaclient.NewClient(context.Background(), server, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to beacon server: %w", err)
	}
	bs, err := api.Bootstrap(root)
	if err != nil {
		return nil, fmt.Errorf("failed to get bootstrap data: %w", err)
	}
	if bs.Header.Hash() != root {
		return nil, fmt.Errorf("bootstrap root did not match requested: want %s, got %s", root, bs.Header.Hash())
	}
	if err := bs.Valid(); err != nil {
		return nil, fmt.Errorf("failed to validate bootstrap data: %w", err)
	}
	current, err := bs.Committee.Deserialize()
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize committee")
	}
	return &LightClient{
		beacon: api,
		store: &store{
			config:     params.SepoliaChainConfig,
			current:    current,
			optimistic: &bs.Header.Header,
			finalized:  &bs.Header.Header,
		},
		quitCh: make(chan struct{}),
	}, nil
}

// ChainHeadEvent returns an authenticated execution payload associated with the
// latest accepted head of the beacon chain.
type ChainHeadEvent struct {
	Data *engine.ExecutableData
}

// SubscribeChainHeadEvent allows callers to subscribe a provided channel to new
// head updates.
func (c *LightClient) SubscribeChainHeadEvent(ch chan<- ChainHeadEvent) event.Subscription {
	return c.chainHeadFeed.Subscribe(ch)
}

// Finalized returns the latest finalized head known to the light client.
func (c *LightClient) Finalized() *types.Header {
	return c.store.finalized
}

// Start executes the main active loop of the light client which drives the
// underlying light client store.
func (c *LightClient) Start() error {
	var (
		ticker       = time.NewTicker(params.SlotLength * time.Second)
		lastFinality = time.Time{}
	)
	for {
		select {
		case <-c.quitCh:
			return nil
		case <-ticker.C:
			log.Trace("Blsync status", "period", c.store.finalizedPeriod(), "active", c.store.currActive, "prevActive", c.store.prevActive, "currCommittee", c.store.current != nil, "nextCommittee", c.store.next != nil)
			if c.store.next == nil {
				log.Debug("Fetching committee update", "period", c.store.finalizedPeriod()+1)
				updates, err := c.beacon.GetRangeUpdate(c.store.finalizedPeriod(), 1)
				if err != nil {
					log.Error("Failed to fetch next committee", "err", err)
				} else {
					for _, update := range updates {
						log.Trace("New beacon range update", "slot", update.AttestedHeader.Slot, "root", update.AttestedHeader.Hash(), "sigslot", update.SignatureSlot, "period", update.AttestedHeader.SyncPeriod())
						if err := c.store.Insert(update); err != nil {
							log.Error("Failed to insert committee update", "err", err)
							break
						}
					}
				}
			}

			var (
				update *types.LightClientUpdate
				err    error
			)
			if time.Since(lastFinality) > time.Minute*5 {
				lastFinality = time.Now()
				update, err = c.beacon.GetFinalityUpdate()
			} else {
				update, err = c.beacon.GetOptimisticUpdate()
			}
			if err != nil {
				log.Error("Failed to retrieve update", "err", err)
				continue
			}
			log.Trace("New beacon update", "slot", update.AttestedHeader.Slot, "root", update.AttestedHeader.Hash(), "sigslot", update.SignatureSlot, "period", update.AttestedHeader.SyncPeriod(), "hasFinalized", update.FinalizedHeader != nil, "hasNext", update.NextSyncCommittee != nil)
			if err := c.store.Insert(update); err != nil {
				log.Error("Failed to insert update", "err", err)
				continue
			}
			head := update.AttestedHeader
			log.Info("Beacon head updated", "slot", head.Slot, "root", head.Hash(), "finalized", c.Finalized().Hash(), "signers", update.SyncAggregate.SignerCount())

			// Fetch full execution payload from beacon provider and send to head feed.
			data, err := c.fetchExecutableData(head.Hash())
			if err != nil {
				log.Error("Failed to insert update", "err", err)
				continue
			}
			c.chainHeadFeed.Send(ChainHeadEvent{Data: data})
		}
	}
}

// Stop halts the light client.
func (c *LightClient) Stop() error {
	close(c.quitCh)
	return nil
}

// fetchExecutableData retrieves the full beacon block associated with the beacon
// block root and returns the inner execution payload.
func (c *LightClient) fetchExecutableData(head common.Hash) (*engine.ExecutableData, error) {
	block, err := c.beacon.GetBlock(head)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution payload: %w", err)
	}
	// Compute the root of the block and verify it matches the root the sync
	// committee signed.
	root, err := block.Root()
	if err != nil {
		return nil, fmt.Errorf("failed to compute root for beacon block: %w", err)
	}
	if common.Hash(root) != head {
		return nil, fmt.Errorf("unable to verify block body against sync committee update")
	}
	return versionedBlockToExecutableData(block), nil
}

// versionedBlockToExecutableData parses versioned blocks and returns a generic
// execution payload object.
func versionedBlockToExecutableData(block *eth2spec.VersionedSignedBeaconBlock) *engine.ExecutableData {
	var ep *engine.ExecutableData
	switch block.Version {
	case eth2spec.DataVersionPhase0:
		panic("phase0 block has no execution payload to send")
	case eth2spec.DataVersionAltair:
		panic("altair block has no execution payload to send")
	case eth2spec.DataVersionBellatrix:
		p := block.Bellatrix.Message.Body.ExecutionPayload
		ep = &engine.ExecutableData{
			ParentHash:    common.Hash(p.ParentHash),
			FeeRecipient:  common.Address(p.FeeRecipient),
			StateRoot:     p.StateRoot,
			ReceiptsRoot:  p.ReceiptsRoot,
			LogsBloom:     p.LogsBloom[:],
			Random:        p.PrevRandao,
			Number:        p.BlockNumber,
			GasLimit:      p.GasLimit,
			GasUsed:       p.GasUsed,
			Timestamp:     p.Timestamp,
			ExtraData:     p.ExtraData,
			BaseFeePerGas: new(big.Int).SetBytes(reverse(p.BaseFeePerGas[:])),
			BlockHash:     common.Hash(p.BlockHash),
			Transactions:  [][]byte{},
			Withdrawals:   nil,
			BlobGasUsed:   nil,
			ExcessBlobGas: nil,
		}
		for _, tx := range p.Transactions {
			ep.Transactions = append(ep.Transactions, tx)
		}
	case eth2spec.DataVersionCapella:
		p := block.Capella.Message.Body.ExecutionPayload
		ep = &engine.ExecutableData{
			ParentHash:    common.Hash(p.ParentHash),
			FeeRecipient:  common.Address(p.FeeRecipient),
			StateRoot:     p.StateRoot,
			ReceiptsRoot:  p.ReceiptsRoot,
			LogsBloom:     p.LogsBloom[:],
			Random:        p.PrevRandao,
			Number:        p.BlockNumber,
			GasLimit:      p.GasLimit,
			GasUsed:       p.GasUsed,
			Timestamp:     p.Timestamp,
			ExtraData:     p.ExtraData,
			BaseFeePerGas: new(big.Int).SetBytes(reverse(p.BaseFeePerGas[:])),
			BlockHash:     common.Hash(p.BlockHash),
			Transactions:  [][]byte{},
			Withdrawals:   nil,
			BlobGasUsed:   nil,
			ExcessBlobGas: nil,
		}
		for _, tx := range p.Transactions {
			ep.Transactions = append(ep.Transactions, tx)
		}
		for _, wx := range p.Withdrawals {
			ep.Withdrawals = append(ep.Withdrawals, &ctypes.Withdrawal{
				Index:     uint64(wx.Index),
				Validator: uint64(wx.ValidatorIndex),
				Address:   common.Address(wx.Address),
				Amount:    uint64(wx.Amount),
			})
		}
	case eth2spec.DataVersionDeneb:
		p := block.Deneb.Message.Body.ExecutionPayload
		ep = &engine.ExecutableData{
			ParentHash:    common.Hash(p.ParentHash),
			FeeRecipient:  common.Address(p.FeeRecipient),
			StateRoot:     common.Hash(p.StateRoot),
			ReceiptsRoot:  common.Hash(p.ReceiptsRoot),
			LogsBloom:     p.LogsBloom[:],
			Random:        p.PrevRandao,
			Number:        p.BlockNumber,
			GasLimit:      p.GasLimit,
			GasUsed:       p.GasUsed,
			Timestamp:     p.Timestamp,
			ExtraData:     p.ExtraData,
			BaseFeePerGas: p.BaseFeePerGas.ToBig(),
			BlockHash:     common.Hash(p.BlockHash),
			Transactions:  [][]byte{},
			Withdrawals:   nil,
			BlobGasUsed:   &p.BlobGasUsed,
			ExcessBlobGas: &p.ExcessBlobGas,
		}
		for _, tx := range p.Transactions {
			ep.Transactions = append(ep.Transactions, tx)
		}
		for _, wx := range p.Withdrawals {
			ep.Withdrawals = append(ep.Withdrawals, &ctypes.Withdrawal{
				Index:     uint64(wx.Index),
				Validator: uint64(wx.ValidatorIndex),
				Address:   common.Address(wx.Address),
				Amount:    uint64(wx.Amount),
			})
		}
	default:
		panic("unknown beacon block version")
	}
	return ep
}

func reverse(b []byte) []byte {
	for i := 0; i < len(b)/2; i++ {
		j := len(b) - i - 1
		b[i], b[j] = b[j], b[i]
	}
	return b
}
