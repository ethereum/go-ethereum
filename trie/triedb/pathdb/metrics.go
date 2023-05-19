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
	cleanHitMeter   = metrics.NewRegisteredMeter("triedb/clean/hit", nil)
	cleanMissMeter  = metrics.NewRegisteredMeter("triedb/clean/miss", nil)
	cleanReadMeter  = metrics.NewRegisteredMeter("triedb/clean/read", nil)
	cleanWriteMeter = metrics.NewRegisteredMeter("triedb/clean/write", nil)

	dirtyHitMeter         = metrics.NewRegisteredMeter("triedb/dirty/hit", nil)
	dirtyMissMeter        = metrics.NewRegisteredMeter("triedb/dirty/miss", nil)
	dirtyReadMeter        = metrics.NewRegisteredMeter("triedb/dirty/read", nil)
	dirtyWriteMeter       = metrics.NewRegisteredMeter("triedb/dirty/write", nil)
	dirtyNodeHitDepthHist = metrics.NewRegisteredHistogram("triedb/dirty/depth", nil, metrics.NewExpDecaySample(1028, 0.015))

	commitTimeTimer  = metrics.NewRegisteredTimer("triedb/commit/time", nil)
	commitNodesMeter = metrics.NewRegisteredMeter("triedb/commit/nodes", nil)
	commitSizeMeter  = metrics.NewRegisteredMeter("triedb/commit/size", nil)

	gcNodesMeter = metrics.NewRegisteredMeter("triedb/gc/nodes", nil)
	gcSizeMeter  = metrics.NewRegisteredMeter("triedb/gc/size", nil)

	diffLayerSizeMeter  = metrics.NewRegisteredMeter("triedb/diff/size", nil)
	diffLayerNodesMeter = metrics.NewRegisteredMeter("triedb/diff/nodes", nil)

	historyBuildTimeMeter = metrics.NewRegisteredTimer("triedb/history/time", nil)
	historyDataSizeMeter  = metrics.NewRegisteredMeter("triedb/history/size/data", nil)
	historyIndexSizeMeter = metrics.NewRegisteredMeter("triedb/history/size/index", nil)
)
