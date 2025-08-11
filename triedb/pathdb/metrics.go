// Copyright 2023 The go-ethereum Authors
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

	dirtyNodeHitMeter     = metrics.NewRegisteredMeter("pathdb/dirty/node/hit", nil)
	dirtyNodeMissMeter    = metrics.NewRegisteredMeter("pathdb/dirty/node/miss", nil)
	dirtyNodeReadMeter    = metrics.NewRegisteredMeter("pathdb/dirty/node/read", nil)
	dirtyNodeWriteMeter   = metrics.NewRegisteredMeter("pathdb/dirty/node/write", nil)
	dirtyNodeHitDepthHist = metrics.NewRegisteredHistogram("pathdb/dirty/node/depth", nil, metrics.NewExpDecaySample(1028, 0.015))

	stateAccountInexMeter     = metrics.NewRegisteredMeter("pathdb/state/account/inex/total", nil)
	stateStorageInexMeter     = metrics.NewRegisteredMeter("pathdb/state/storage/inex/total", nil)
	stateAccountInexDiskMeter = metrics.NewRegisteredMeter("pathdb/state/account/inex/disk", nil)
	stateStorageInexDiskMeter = metrics.NewRegisteredMeter("pathdb/state/storage/inex/disk", nil)

	stateAccountExistMeter     = metrics.NewRegisteredMeter("pathdb/state/account/exist/total", nil)
	stateStorageExistMeter     = metrics.NewRegisteredMeter("pathdb/state/storage/exist/total", nil)
	stateAccountExistDiskMeter = metrics.NewRegisteredMeter("pathdb/state/account/exist/disk", nil)
	stateStorageExistDiskMeter = metrics.NewRegisteredMeter("pathdb/state/storage/exist/disk", nil)

	dirtyStateHitMeter     = metrics.NewRegisteredMeter("pathdb/dirty/state/hit", nil)
	dirtyStateMissMeter    = metrics.NewRegisteredMeter("pathdb/dirty/state/miss", nil)
	dirtyStateReadMeter    = metrics.NewRegisteredMeter("pathdb/dirty/state/read", nil)
	dirtyStateWriteMeter   = metrics.NewRegisteredMeter("pathdb/dirty/state/write", nil)
	dirtyStateHitDepthHist = metrics.NewRegisteredHistogram("pathdb/dirty/state/depth", nil, metrics.NewExpDecaySample(1028, 0.015))

	nodeCleanFalseMeter = metrics.NewRegisteredMeter("pathdb/clean/false", nil)
	nodeDirtyFalseMeter = metrics.NewRegisteredMeter("pathdb/dirty/false", nil)
	nodeDiskFalseMeter  = metrics.NewRegisteredMeter("pathdb/disk/false", nil)
	nodeDiffFalseMeter  = metrics.NewRegisteredMeter("pathdb/diff/false", nil)

	commitTimeTimer     = metrics.NewRegisteredResettingTimer("pathdb/commit/time", nil)
	commitNodesMeter    = metrics.NewRegisteredMeter("pathdb/commit/nodes", nil)
	commitAccountsMeter = metrics.NewRegisteredMeter("pathdb/commit/accounts", nil)
	commitStoragesMeter = metrics.NewRegisteredMeter("pathdb/commit/slots", nil)
	commitBytesMeter    = metrics.NewRegisteredMeter("pathdb/commit/bytes", nil)

	gcTrieNodeMeter      = metrics.NewRegisteredMeter("pathdb/gc/node/count", nil)
	gcTrieNodeBytesMeter = metrics.NewRegisteredMeter("pathdb/gc/node/bytes", nil)
	gcAccountMeter       = metrics.NewRegisteredMeter("pathdb/gc/account/count", nil)
	gcAccountBytesMeter  = metrics.NewRegisteredMeter("pathdb/gc/account/bytes", nil)
	gcStorageMeter       = metrics.NewRegisteredMeter("pathdb/gc/storage/count", nil)
	gcStorageBytesMeter  = metrics.NewRegisteredMeter("pathdb/gc/storage/bytes", nil)

	historyBuildTimeMeter  = metrics.NewRegisteredResettingTimer("pathdb/history/time", nil)
	historyDataBytesMeter  = metrics.NewRegisteredMeter("pathdb/history/bytes/data", nil)
	historyIndexBytesMeter = metrics.NewRegisteredMeter("pathdb/history/bytes/index", nil)

	indexHistoryTimer   = metrics.NewRegisteredResettingTimer("pathdb/history/index/time", nil)
	unindexHistoryTimer = metrics.NewRegisteredResettingTimer("pathdb/history/unindex/time", nil)

	lookupAddLayerTimer    = metrics.NewRegisteredResettingTimer("pathdb/lookup/add/time", nil)
	lookupRemoveLayerTimer = metrics.NewRegisteredResettingTimer("pathdb/lookup/remove/time", nil)

	historicalAccountReadTimer = metrics.NewRegisteredResettingTimer("pathdb/history/account/reads", nil)
	historicalStorageReadTimer = metrics.NewRegisteredResettingTimer("pathdb/history/storage/reads", nil)
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
