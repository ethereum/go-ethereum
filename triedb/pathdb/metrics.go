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
	cleanHitMeter   = metrics.NewRegisteredMeter("pathdb/clean/hit", nil)
	cleanMissMeter  = metrics.NewRegisteredMeter("pathdb/clean/miss", nil)
	cleanReadMeter  = metrics.NewRegisteredMeter("pathdb/clean/read", nil)
	cleanWriteMeter = metrics.NewRegisteredMeter("pathdb/clean/write", nil)

	dirtyHitMeter         = metrics.NewRegisteredMeter("pathdb/dirty/hit", nil)
	dirtyMissMeter        = metrics.NewRegisteredMeter("pathdb/dirty/miss", nil)
	dirtyReadMeter        = metrics.NewRegisteredMeter("pathdb/dirty/read", nil)
	dirtyWriteMeter       = metrics.NewRegisteredMeter("pathdb/dirty/write", nil)
	dirtyNodeHitDepthHist = metrics.NewRegisteredHistogram("pathdb/dirty/depth", nil, metrics.NewExpDecaySample(1028, 0.015))

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
	snapGeneratedAccountMeter     = metrics.NewRegisteredMeter("pathdb/generation/account/generated", nil)
	snapRecoveredAccountMeter     = metrics.NewRegisteredMeter("pathdb/generation/account/recovered", nil)
	snapWipedAccountMeter         = metrics.NewRegisteredMeter("pathdb/generation/account/wiped", nil)
	snapMissallAccountMeter       = metrics.NewRegisteredMeter("pathdb/generation/account/missall", nil)
	snapGeneratedStorageMeter     = metrics.NewRegisteredMeter("pathdb/generation/storage/generated", nil)
	snapRecoveredStorageMeter     = metrics.NewRegisteredMeter("pathdb/generation/storage/recovered", nil)
	snapWipedStorageMeter         = metrics.NewRegisteredMeter("pathdb/generation/storage/wiped", nil)
	snapMissallStorageMeter       = metrics.NewRegisteredMeter("pathdb/generation/storage/missall", nil)
	snapDanglingStorageMeter      = metrics.NewRegisteredMeter("pathdb/generation/storage/dangling", nil)
	snapSuccessfulRangeProofMeter = metrics.NewRegisteredMeter("pathdb/generation/proof/success", nil)
	snapFailedRangeProofMeter     = metrics.NewRegisteredMeter("pathdb/generation/proof/failure", nil)

	snapAccountProveCounter    = metrics.NewRegisteredCounter("pathdb/generation/duration/account/prove", nil)
	snapAccountTrieReadCounter = metrics.NewRegisteredCounter("pathdb/generation/duration/account/trieread", nil)
	snapAccountSnapReadCounter = metrics.NewRegisteredCounter("pathdb/generation/duration/account/snapread", nil)
	snapAccountWriteCounter    = metrics.NewRegisteredCounter("pathdb/generation/duration/account/write", nil)
	snapStorageProveCounter    = metrics.NewRegisteredCounter("pathdb/generation/duration/storage/prove", nil)
	snapStorageTrieReadCounter = metrics.NewRegisteredCounter("pathdb/generation/duration/storage/trieread", nil)
	snapStorageSnapReadCounter = metrics.NewRegisteredCounter("pathdb/generation/duration/storage/snapread", nil)
	snapStorageWriteCounter    = metrics.NewRegisteredCounter("pathdb/generation/duration/storage/write", nil)
	snapStorageCleanCounter    = metrics.NewRegisteredCounter("state/snapshot/generation/duration/storage/clean", nil)
)
