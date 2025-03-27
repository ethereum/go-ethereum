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

package eth

import "github.com/ethereum/go-ethereum/metrics"

type peerMeters struct {
	txReceived *metrics.Meter
	txSent     *metrics.Meter

	txReplyInMeter          *metrics.Meter
	txReplyKnownMeter       *metrics.Meter
	txReplyUnderpricedMeter *metrics.Meter
	txReplyOtherRejectMeter *metrics.Meter
}

// newPeerMeters registers and returns peer-level meters.
func newPeerMeters(base string, r metrics.Registry) *peerMeters {
	return &peerMeters{
		txReceived: metrics.NewRegisteredMeter(base+"/txReceived", r),
		txSent:     metrics.NewRegisteredMeter(base+"/txSent", r),

		txReplyInMeter:          metrics.NewRegisteredMeter(base+"/eth/fetcher/transaction/replies/in", nil),
		txReplyKnownMeter:       metrics.NewRegisteredMeter(base+"/eth/fetcher/transaction/replies/known", nil),
		txReplyUnderpricedMeter: metrics.NewRegisteredMeter(base+"/eth/fetcher/transaction/replies/underpriced", nil),
		txReplyOtherRejectMeter: metrics.NewRegisteredMeter(base+"/eth/fetcher/transaction/replies/otherreject", nil),
	}
}
