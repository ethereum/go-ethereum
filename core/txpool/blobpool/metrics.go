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

package blobpool

import "github.com/ethereum/go-ethereum/metrics"

var (
	// datacapGauge tracks the user's configured capacity for the blob pool. It
	// is mostly a way to expose/debug issues.
	datacapGauge = metrics.NewRegisteredGauge("blobpool/datacap", nil)

	// The below metrics track the per-datastore metrics for the primary blob
	// store and the temporary limbo store.
	datausedGauge = metrics.NewRegisteredGauge("blobpool/dataused", nil)
	datarealGauge = metrics.NewRegisteredGauge("blobpool/datareal", nil)
	slotusedGauge = metrics.NewRegisteredGauge("blobpool/slotused", nil)

	limboDatausedGauge = metrics.NewRegisteredGauge("blobpool/limbo/dataused", nil)
	limboDatarealGauge = metrics.NewRegisteredGauge("blobpool/limbo/datareal", nil)
	limboSlotusedGauge = metrics.NewRegisteredGauge("blobpool/limbo/slotused", nil)

	// The below metrics track the per-shelf metrics for the primary blob store
	// and the temporary limbo store.
	shelfDatausedGaugeName = "blobpool/shelf_%d/dataused"
	shelfDatagapsGaugeName = "blobpool/shelf_%d/datagaps"
	shelfSlotusedGaugeName = "blobpool/shelf_%d/slotused"
	shelfSlotgapsGaugeName = "blobpool/shelf_%d/slotgaps"

	limboShelfDatausedGaugeName = "blobpool/limbo/shelf_%d/dataused"
	limboShelfDatagapsGaugeName = "blobpool/limbo/shelf_%d/datagaps"
	limboShelfSlotusedGaugeName = "blobpool/limbo/shelf_%d/slotused"
	limboShelfSlotgapsGaugeName = "blobpool/limbo/shelf_%d/slotgaps"

	// The oversized metrics aggregate the shelf stats above the max blob count
	// limits to track transactions that are just huge, but don't contain blobs.
	//
	// There are no oversized data in the limbo, it only contains blobs and some
	// constant metadata.
	oversizedDatausedGauge = metrics.NewRegisteredGauge("blobpool/oversized/dataused", nil)
	oversizedDatagapsGauge = metrics.NewRegisteredGauge("blobpool/oversized/datagaps", nil)
	oversizedSlotusedGauge = metrics.NewRegisteredGauge("blobpool/oversized/slotused", nil)
	oversizedSlotgapsGauge = metrics.NewRegisteredGauge("blobpool/oversized/slotgaps", nil)

	// basefeeGauge and blobfeeGauge track the current network 1559 base fee and
	// 4844 blob fee respectively.
	basefeeGauge = metrics.NewRegisteredGauge("blobpool/basefee", nil)
	blobfeeGauge = metrics.NewRegisteredGauge("blobpool/blobfee", nil)

	// pooltipGauge is the configurable miner tip to permit a transaction into
	// the pool.
	pooltipGauge = metrics.NewRegisteredGauge("blobpool/pooltip", nil)

	// addwait/time, resetwait/time and getwait/time track the rough health of
	// the pool and whether it's capable of keeping up with the load from the
	// network.
	addwaitHist   = metrics.NewRegisteredHistogram("blobpool/addwait", nil, metrics.NewExpDecaySample(1028, 0.015))
	addtimeHist   = metrics.NewRegisteredHistogram("blobpool/addtime", nil, metrics.NewExpDecaySample(1028, 0.015))
	getwaitHist   = metrics.NewRegisteredHistogram("blobpool/getwait", nil, metrics.NewExpDecaySample(1028, 0.015))
	gettimeHist   = metrics.NewRegisteredHistogram("blobpool/gettime", nil, metrics.NewExpDecaySample(1028, 0.015))
	pendwaitHist  = metrics.NewRegisteredHistogram("blobpool/pendwait", nil, metrics.NewExpDecaySample(1028, 0.015))
	pendtimeHist  = metrics.NewRegisteredHistogram("blobpool/pendtime", nil, metrics.NewExpDecaySample(1028, 0.015))
	resetwaitHist = metrics.NewRegisteredHistogram("blobpool/resetwait", nil, metrics.NewExpDecaySample(1028, 0.015))
	resettimeHist = metrics.NewRegisteredHistogram("blobpool/resettime", nil, metrics.NewExpDecaySample(1028, 0.015))

	// The below metrics track various cases where transactions are dropped out
	// of the pool. Most are exceptional, some are chain progression and some
	// threshold cappings.
	dropInvalidMeter     = metrics.NewRegisteredMeter("blobpool/drop/invalid", nil)     // Invalid transaction, consensus change or bugfix, neutral-ish
	dropDanglingMeter    = metrics.NewRegisteredMeter("blobpool/drop/dangling", nil)    // First nonce gapped, bad
	dropFilledMeter      = metrics.NewRegisteredMeter("blobpool/drop/filled", nil)      // State full-overlap, chain progress, ok
	dropOverlappedMeter  = metrics.NewRegisteredMeter("blobpool/drop/overlapped", nil)  // State partial-overlap, chain progress, ok
	dropRepeatedMeter    = metrics.NewRegisteredMeter("blobpool/drop/repeated", nil)    // Repeated nonce, bad
	dropGappedMeter      = metrics.NewRegisteredMeter("blobpool/drop/gapped", nil)      // Non-first nonce gapped, bad
	dropOverdraftedMeter = metrics.NewRegisteredMeter("blobpool/drop/overdrafted", nil) // Balance exceeded, bad
	dropOvercappedMeter  = metrics.NewRegisteredMeter("blobpool/drop/overcapped", nil)  // Per-account cap exceeded, bad
	dropOverflownMeter   = metrics.NewRegisteredMeter("blobpool/drop/overflown", nil)   // Global disk cap exceeded, neutral-ish
	dropUnderpricedMeter = metrics.NewRegisteredMeter("blobpool/drop/underpriced", nil) // Gas tip changed, neutral
	dropReplacedMeter    = metrics.NewRegisteredMeter("blobpool/drop/replaced", nil)    // Transaction replaced, neutral

	// The below metrics track various outcomes of transactions being added to
	// the pool.
	addInvalidMeter      = metrics.NewRegisteredMeter("blobpool/add/invalid", nil)      // Invalid transaction, reject, neutral
	addUnderpricedMeter  = metrics.NewRegisteredMeter("blobpool/add/underpriced", nil)  // Gas tip too low, neutral
	addStaleMeter        = metrics.NewRegisteredMeter("blobpool/add/stale", nil)        // Nonce already filled, reject, bad-ish
	addGappedMeter       = metrics.NewRegisteredMeter("blobpool/add/gapped", nil)       // Nonce gapped, reject, bad-ish
	addOverdraftedMeter  = metrics.NewRegisteredMeter("blobpool/add/overdrafted", nil)  // Balance exceeded, reject, neutral
	addOvercappedMeter   = metrics.NewRegisteredMeter("blobpool/add/overcapped", nil)   // Per-account cap exceeded, reject, neutral
	addNoreplaceMeter    = metrics.NewRegisteredMeter("blobpool/add/noreplace", nil)    // Replacement fees or tips too low, neutral
	addNonExclusiveMeter = metrics.NewRegisteredMeter("blobpool/add/nonexclusive", nil) // Plain transaction from same account exists, reject, neutral
	addValidMeter        = metrics.NewRegisteredMeter("blobpool/add/valid", nil)        // Valid transaction, add, neutral
)
