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

package trie

import "github.com/ethereum/go-ethereum/metrics"

var (
	triedbCleanHitMeter   = metrics.NewRegisteredMeter("trie/triedb/clean/hit", nil)
	triedbCleanMissMeter  = metrics.NewRegisteredMeter("trie/triedb/clean/miss", nil)
	triedbCleanReadMeter  = metrics.NewRegisteredMeter("trie/triedb/clean/read", nil)
	triedbCleanWriteMeter = metrics.NewRegisteredMeter("trie/triedb/clean/write", nil)

	triedbFallbackHitMeter  = metrics.NewRegisteredMeter("trie/triedb/clean/fallback/hit", nil)
	triedbFallbackReadMeter = metrics.NewRegisteredMeter("trie/triedb/clean/fallback/read", nil)

	triedbDirtyHitMeter   = metrics.NewRegisteredMeter("trie/triedb/dirty/hit", nil)
	triedbDirtyMissMeter  = metrics.NewRegisteredMeter("trie/triedb/dirty/miss", nil)
	triedbDirtyReadMeter  = metrics.NewRegisteredMeter("trie/triedb/dirty/read", nil)
	triedbDirtyWriteMeter = metrics.NewRegisteredMeter("trie/triedb/dirty/write", nil)

	triedbDirtyNodeHitDepthHist = metrics.NewRegisteredHistogram("trie/triedb/dirty/depth", nil, metrics.NewExpDecaySample(1028, 0.015))

	triedbCommitTimeTimer  = metrics.NewRegisteredTimer("trie/triedb/commit/time", nil)
	triedbCommitNodesMeter = metrics.NewRegisteredMeter("trie/triedb/commit/nodes", nil)
	triedbCommitSizeMeter  = metrics.NewRegisteredMeter("trie/triedb/commit/size", nil)

	triedbDiffLayerSizeMeter  = metrics.NewRegisteredMeter("trie/triedb/diff/size", nil)
	triedbDiffLayerNodesMeter = metrics.NewRegisteredMeter("trie/triedb/diff/nodes", nil)

	triedbReverseDiffTimeTimer = metrics.NewRegisteredTimer("trie/triedb/reversediff/time", nil)
	triedbReverseDiffSizeMeter = metrics.NewRegisteredMeter("trie/triedb/reversediff/size", nil)
)
