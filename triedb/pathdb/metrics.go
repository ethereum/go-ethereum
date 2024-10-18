// Copyright 2022 The go-ethereum Authors
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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package pathdb

import "github.com/ethereum/go-ethereum/metrics"

var (
	cleanNodeHitMeter   = metrics.NewRegisteredMeter("pathdb/clean/node/hit", nil)
	cleanNodeMissMeter  = metrics.NewRegisteredMeter("pathdb/clean/node/miss", nil)
	cleanNodeReadMeter  = metrics.NewRegisteredMeter("pathdb/clean/node/read", nil)
	cleanNodeWriteMeter = metrics.NewRegisteredMeter("pathdb/clean/node/write", nil)

	cleanStateHitMeter   = metrics.NewRegisteredMeter("pathdb/clean/state/hit", nil)
	cleanStateMissMeter  = metrics.NewRegisteredMeter("pathdb/clean/state/miss", nil)
	cleanStateReadMeter  = metrics.NewRegisteredMeter("pathdb/clean/state/read", nil)
	cleanStateWriteMeter = metrics.NewRegisteredMeter("pathdb/clean/state/write", nil)

	stateAccountMissMeter = metrics.NewRegisteredMeter("pathdb/state/account/miss/total", nil)
	stateAccountHitMeter  = metrics.NewRegisteredMeter("pathdb/state/account/hit/total", nil)
	stateStorageMissMeter = metrics.NewRegisteredMeter("pathdb/state/storage/miss/total", nil)
	stateStorageHitMeter  = metrics.NewRegisteredMeter("pathdb/state/storage/hit/total", nil)

	stateAccountDiskMissMeter = metrics.NewRegisteredMeter("pathdb/state/account/miss/disk", nil)
	stateAccountDiskHitMeter  = metrics.NewRegisteredMeter("pathdb/state/account/hit/disk", nil)
	stateStorageDiskMissMeter = metrics.NewRegisteredMeter("pathdb/state/storage/miss/disk", nil)
	stateStorageDiskHitMeter  = metrics.NewRegisteredMeter("pathdb/state/storage/hit/disk", nil)

	dirtyNodeHitMeter     = metrics.NewRegisteredMeter("pathdb/dirty/hit/node", nil)
	dirtyNodeMissMeter    = metrics.NewRegisteredMeter("pathdb/dirty/miss/node", nil)
	dirtyNodeReadMeter    = metrics.NewRegisteredMeter("pathdb/dirty/read/node", nil)
	dirtyNodeWriteMeter   = metrics.NewRegisteredMeter("pathdb/dirty/write/node", nil)
	dirtyNodeHitDepthHist = metrics.NewRegisteredHistogram("pathdb/dirty/depth/node", nil, metrics.NewExpDecaySample(1028, 0.015))

	dirtyStateHitMeter     = metrics.NewRegisteredMeter("pathdb/dirty/hit/state", nil)
	dirtyStateMissMeter    = metrics.NewRegisteredMeter("pathdb/dirty/miss/state", nil)
	dirtyStateReadMeter    = metrics.NewRegisteredMeter("pathdb/dirty/read/state", nil)
	dirtyStateWriteMeter   = metrics.NewRegisteredMeter("pathdb/dirty/write/state", nil)
	dirtyStateHitDepthHist = metrics.NewRegisteredHistogram("pathdb/dirty/depth/state", nil, metrics.NewExpDecaySample(1028, 0.015))

	cleanFalseMeter = metrics.NewRegisteredMeter("pathdb/clean/false", nil)
	dirtyFalseMeter = metrics.NewRegisteredMeter("pathdb/dirty/false", nil)
	diskFalseMeter  = metrics.NewRegisteredMeter("pathdb/disk/false", nil)
	diffFalseMeter  = metrics.NewRegisteredMeter("pathdb/diff/false", nil)

	commitTimeTimer     = metrics.NewRegisteredTimer("pathdb/commit/time", nil)
	commitNodesMeter    = metrics.NewRegisteredMeter("pathdb/commit/nodes", nil)
	commitAccountsMeter = metrics.NewRegisteredMeter("pathdb/commit/accounts", nil)
	commitStoragesMeter = metrics.NewRegisteredMeter("pathdb/commit/slots", nil)
	commitBytesMeter    = metrics.NewRegisteredMeter("pathdb/commit/bytes", nil)

	gcTrieNodeMeter      = metrics.NewRegisteredMeter("pathdb/gc/trienode/count", nil)
	gcTrieNodeBytesMeter = metrics.NewRegisteredMeter("pathdb/gc/trienode/bytes", nil)
	gcAccountMeter       = metrics.NewRegisteredMeter("pathdb/gc/account/count", nil)
	gcAccountBytesMeter  = metrics.NewRegisteredMeter("pathdb/gc/account/bytes", nil)
	gcStorageMeter       = metrics.NewRegisteredMeter("pathdb/gc/storage/count", nil)
	gcStorageBytesMeter  = metrics.NewRegisteredMeter("pathdb/gc/storage/bytes", nil)

	historyBuildTimeMeter  = metrics.NewRegisteredTimer("pathdb/history/time", nil)
	historyDataBytesMeter  = metrics.NewRegisteredMeter("pathdb/history/bytes/data", nil)
	historyIndexBytesMeter = metrics.NewRegisteredMeter("pathdb/history/bytes/index", nil)
)

// Metrics in generation
var (
	generatedAccountMeter     = metrics.NewRegisteredMeter("pathdb/generation/account/generated", nil)
	recoveredAccountMeter     = metrics.NewRegisteredMeter("pathdb/generation/account/recovered", nil)
	wipedAccountMeter         = metrics.NewRegisteredMeter("pathdb/generation/account/wiped", nil)
	missallAccountMeter       = metrics.NewRegisteredMeter("pathdb/generation/account/missall", nil)
	generatedStorageMeter     = metrics.NewRegisteredMeter("pathdb/generation/storage/generated", nil)
	recoveredStorageMeter     = metrics.NewRegisteredMeter("pathdb/generation/storage/recovered", nil)
	wipedStorageMeter         = metrics.NewRegisteredMeter("pathdb/generation/storage/wiped", nil)
	missallStorageMeter       = metrics.NewRegisteredMeter("pathdb/generation/storage/missall", nil)
	danglingStorageMeter      = metrics.NewRegisteredMeter("pathdb/generation/storage/dangling", nil)
	successfulRangeProofMeter = metrics.NewRegisteredMeter("pathdb/generation/proof/success", nil)
	failedRangeProofMeter     = metrics.NewRegisteredMeter("pathdb/generation/proof/failure", nil)

	accountProveCounter    = metrics.NewRegisteredCounter("pathdb/generation/duration/account/prove", nil)
	accountTrieReadCounter = metrics.NewRegisteredCounter("pathdb/generation/duration/account/trieread", nil)
	accountSnapReadCounter = metrics.NewRegisteredCounter("pathdb/generation/duration/account/snapread", nil)
	accountWriteCounter    = metrics.NewRegisteredCounter("pathdb/generation/duration/account/write", nil)
	storageProveCounter    = metrics.NewRegisteredCounter("pathdb/generation/duration/storage/prove", nil)
	storageTrieReadCounter = metrics.NewRegisteredCounter("pathdb/generation/duration/storage/trieread", nil)
	storageSnapReadCounter = metrics.NewRegisteredCounter("pathdb/generation/duration/storage/snapread", nil)
	storageWriteCounter    = metrics.NewRegisteredCounter("pathdb/generation/duration/storage/write", nil)
	storageCleanCounter    = metrics.NewRegisteredCounter("state/snapshot/generation/duration/storage/clean", nil)
)
