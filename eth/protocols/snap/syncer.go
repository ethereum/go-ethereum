// Copyright 2026 The go-ethereum Authors
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

package snap

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
)

// Progress is the set of snap-syncer progress that eth/downloader surfaces in
// ethereum.SyncProgress. The two syncer versions report it via different types
// (syncProgress / syncProgressV2). The adapters normalize to this.
type Progress struct {
	AccountSynced  uint64
	AccountBytes   common.StorageSize
	BytecodeSynced uint64
	BytecodeBytes  common.StorageSize
	StorageSynced  uint64
	StorageBytes   common.StorageSize

	// Healing-phase status. Reported by snap/1 only.
	TrienodeHealSynced uint64
	TrienodeHealBytes  common.StorageSize
	BytecodeHealSynced uint64
	BytecodeHealBytes  common.StorageSize
	HealingTrienodes   uint64
	HealingBytecode    uint64
}

// Syncer is the uniform view over the snap/1 (*syncer) and snap/2 (*syncerV2)
// state syncers, consumed by eth/downloader. Peers are passed as SyncPeerV2,
// which is a superset of SyncPeer, so a single peer value works for both
// underlying syncers.
type Syncer interface {
	Sync(pivot *types.Header, cancel chan struct{}) error
	Progress() Progress
	Register(peer SyncPeerV2) error
	Unregister(id string) error
	OnAccounts(peer SyncPeerV2, id uint64, hashes []common.Hash, accounts [][]byte, proof [][]byte) error
	OnStorage(peer SyncPeerV2, id uint64, hashes [][]common.Hash, slots [][][]byte, proof [][]byte) error
	OnByteCodes(peer SyncPeerV2, id uint64, bytecodes [][]byte) error
	OnTrieNodes(peer SyncPeerV2, id uint64, trienodes [][]byte) error
	OnAccessLists(peer SyncPeerV2, id uint64, lists rlp.RawList[rlp.RawValue]) error

	// FrozenPivot returns the pivot header the syncer is bound to, or nil if
	// the pivot may still be chosen and moved freely.
	FrozenPivot() *types.Header

	// Version is the snap protocol version this syncer implements.
	Version() uint
}

// NewV1Syncer returns a Syncer backed by the snap/1 state syncer.
func NewV1Syncer(db ethdb.Database, scheme string) Syncer {
	return syncerV1Adapter{newSyncer(db, scheme)}
}

// NewV2Syncer returns a Syncer backed by the snap/2 state syncer.
func NewV2Syncer(db ethdb.Database, scheme string) Syncer {
	return syncerV2Adapter{newSyncerV2(db, scheme)}
}

// syncerV1Adapter adapts the snap/1 *syncer to Syncer.
type syncerV1Adapter struct{ *syncer }

func (s syncerV1Adapter) Sync(pivot *types.Header, cancel chan struct{}) error {
	return s.syncer.Sync(pivot.Root, cancel)
}

func (s syncerV1Adapter) Progress() Progress {
	progress, pending := s.syncer.Progress()
	return Progress{
		AccountSynced:      progress.AccountSynced,
		AccountBytes:       progress.AccountBytes,
		BytecodeSynced:     progress.BytecodeSynced,
		BytecodeBytes:      progress.BytecodeBytes,
		StorageSynced:      progress.StorageSynced,
		StorageBytes:       progress.StorageBytes,
		TrienodeHealSynced: progress.TrienodeHealSynced,
		TrienodeHealBytes:  progress.TrienodeHealBytes,
		BytecodeHealSynced: progress.BytecodeHealSynced,
		BytecodeHealBytes:  progress.BytecodeHealBytes,
		HealingTrienodes:   pending.TrienodeHeal,
		HealingBytecode:    pending.BytecodeHeal,
	}
}

// The snap/1 syncer's methods take SyncPeer. SyncPeerV2 is a superset, so the
// incoming peer satisfies them directly. Explicit forwarders are needed because
// the parameter types differ.
func (s syncerV1Adapter) Register(peer SyncPeerV2) error { return s.syncer.Register(peer) }
func (s syncerV1Adapter) OnAccounts(peer SyncPeerV2, id uint64, hashes []common.Hash, accounts [][]byte, proof [][]byte) error {
	return s.syncer.OnAccounts(peer, id, hashes, accounts, proof)
}
func (s syncerV1Adapter) OnStorage(peer SyncPeerV2, id uint64, hashes [][]common.Hash, slots [][][]byte, proof [][]byte) error {
	return s.syncer.OnStorage(peer, id, hashes, slots, proof)
}
func (s syncerV1Adapter) OnByteCodes(peer SyncPeerV2, id uint64, bytecodes [][]byte) error {
	return s.syncer.OnByteCodes(peer, id, bytecodes)
}
func (s syncerV1Adapter) OnTrieNodes(peer SyncPeerV2, id uint64, trienodes [][]byte) error {
	return s.syncer.OnTrieNodes(peer, id, trienodes)
}

// OnAccessLists is a no-op for snap/1, which never requests BALs.
func (syncerV1Adapter) OnAccessLists(SyncPeerV2, uint64, rlp.RawList[rlp.RawValue]) error {
	return nil
}

// Version is SNAP1
func (syncerV1Adapter) Version() uint { return SNAP1 }

// FrozenPivot is always nil for snap/1: the sync target must keep tracking
// the chain head, ensuring the state is available in the network, so the
// pivot is never frozen.
func (syncerV1Adapter) FrozenPivot() *types.Header { return nil }

// syncerV2Adapter adapts the snap/2 *syncerV2 to Syncer. Its peer-facing methods
// already take SyncPeerV2 and its Sync already takes a header, so only Progress
// (different return type) and OnTrieNodes (absent) need wrapping.
type syncerV2Adapter struct{ *syncerV2 }

func (s syncerV2Adapter) Progress() Progress {
	progress := s.syncerV2.Progress()
	return Progress{
		AccountSynced:  progress.AccountSynced,
		AccountBytes:   progress.AccountBytes,
		BytecodeSynced: progress.BytecodeSynced,
		BytecodeBytes:  progress.BytecodeBytes,
		StorageSynced:  progress.StorageSynced,
		StorageBytes:   progress.StorageBytes,
	}
}

// OnTrieNodes is a no-op for snap/2, which heals via BALs rather than trie nodes.
// Stale responses from snap/1 peers are silently ignored.
func (syncerV2Adapter) OnTrieNodes(SyncPeerV2, uint64, [][]byte) error { return nil }

// Version is SNAP2; snap/2 needs SNAP2 peers to serve the BAL requests it issues.
func (syncerV2Adapter) Version() uint { return SNAP2 }
