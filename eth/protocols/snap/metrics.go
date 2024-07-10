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

package snap

import (
	metrics "github.com/ethereum/go-ethereum/metrics"
)

var (
	ingressRegistrationErrorName = "eth/protocols/snap/ingress/registration/error"
	egressRegistrationErrorName  = "eth/protocols/snap/egress/registration/error"

	IngressRegistrationErrorMeter = metrics.NewRegisteredMeter(ingressRegistrationErrorName, nil)
	EgressRegistrationErrorMeter  = metrics.NewRegisteredMeter(egressRegistrationErrorName, nil)

	// accountInnerDeleteGauge is the metric to track how many dangling trie nodes
	// covered by extension node in account trie are deleted during the sync.
	accountInnerDeleteGauge = metrics.NewRegisteredGauge("eth/protocols/snap/sync/delete/account/inner", nil)

	// storageInnerDeleteGauge is the metric to track how many dangling trie nodes
	// covered by extension node in storage trie are deleted during the sync.
	storageInnerDeleteGauge = metrics.NewRegisteredGauge("eth/protocols/snap/sync/delete/storage/inner", nil)

	// accountOuterDeleteGauge is the metric to track how many dangling trie nodes
	// above the committed nodes in account trie are deleted during the sync.
	accountOuterDeleteGauge = metrics.NewRegisteredGauge("eth/protocols/snap/sync/delete/account/outer", nil)

	// storageOuterDeleteGauge is the metric to track how many dangling trie nodes
	// above the committed nodes in storage trie are deleted during the sync.
	storageOuterDeleteGauge = metrics.NewRegisteredGauge("eth/protocols/snap/sync/delete/storage/outer", nil)

	// lookupGauge is the metric to track how many trie node lookups are
	// performed to determine if node needs to be deleted.
	accountInnerLookupGauge = metrics.NewRegisteredGauge("eth/protocols/snap/sync/account/lookup/inner", nil)
	accountOuterLookupGauge = metrics.NewRegisteredGauge("eth/protocols/snap/sync/account/lookup/outer", nil)
	storageInnerLookupGauge = metrics.NewRegisteredGauge("eth/protocols/snap/sync/storage/lookup/inner", nil)
	storageOuterLookupGauge = metrics.NewRegisteredGauge("eth/protocols/snap/sync/storage/lookup/outer", nil)

	// smallStorageGauge is the metric to track how many storages are small enough
	// to retrieved in one or two request.
	smallStorageGauge = metrics.NewRegisteredGauge("eth/protocols/snap/sync/storage/small", nil)

	// largeStorageGauge is the metric to track how many storages are large enough
	// to retrieved concurrently.
	largeStorageGauge = metrics.NewRegisteredGauge("eth/protocols/snap/sync/storage/large", nil)

	// skipStorageHealingGauge is the metric to track how many storages are retrieved
	// in multiple requests but healing is not necessary.
	skipStorageHealingGauge = metrics.NewRegisteredGauge("eth/protocols/snap/sync/storage/noheal", nil)

	// largeStorageDiscardGauge is the metric to track how many chunked storages are
	// discarded during the snap sync.
	largeStorageDiscardGauge = metrics.NewRegisteredGauge("eth/protocols/snap/sync/storage/chunk/discard", nil)
	largeStorageResumedGauge = metrics.NewRegisteredGauge("eth/protocols/snap/sync/storage/chunk/resume", nil)
)
