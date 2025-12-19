// Copyright 2025 The go-ethereum Authors
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

package catalyst

import "github.com/ethereum/go-ethereum/metrics"

var (
	// Number of blobs requested via getBlobsV2
	getBlobsRequestedCounter = metrics.NewRegisteredCounter("engine/getblobs/requested", nil)

	// Number of blobs requested via getBlobsV2 that are present in the blobpool
	getBlobsAvailableCounter = metrics.NewRegisteredCounter("engine/getblobs/available", nil)

	// Number of times getBlobsV2 responded with “hit”
	getBlobsV2RequestHit = metrics.NewRegisteredCounter("engine/getblobs/hit", nil)

	// Number of times getBlobsV2 responded with “miss”
	getBlobsV2RequestMiss = metrics.NewRegisteredCounter("engine/getblobs/miss", nil)
)
