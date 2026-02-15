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
	base string
	reg  metrics.Registry

	txReceived       *metrics.Meter
	txSent           *metrics.Meter
	pooledTxSent     *metrics.Meter
	pooledTxReceived *metrics.Meter
	annReceived      *metrics.Meter
	annSent          *metrics.Meter
}

// newPeerMeters registers and returns peer-level meters.
func newPeerMeters(base string, r metrics.Registry) *peerMeters {
	if r == nil {
		r = metrics.DefaultRegistry
	}
	return &peerMeters{
		base: base,
		reg:  r,

		txReceived:       metrics.NewRegisteredMeter(base+"/txReceived", r),
		txSent:           metrics.NewRegisteredMeter(base+"/txSent", r),
		pooledTxSent:     metrics.NewRegisteredMeter(base+"/pooledTxSent", r),
		pooledTxReceived: metrics.NewRegisteredMeter(base+"/pooledTxReceived", r),
		annReceived:      metrics.NewRegisteredMeter(base+"/annReceived", r),
		annSent:          metrics.NewRegisteredMeter(base+"/annSent", r),
	}
}

func (m *peerMeters) Close() {
	m.reg.Unregister(m.base + "/txReceived")
	m.reg.Unregister(m.base + "/txSent")
	m.reg.Unregister(m.base + "/pooledTxSent")
	m.reg.Unregister(m.base + "/pooledTxReceived")
	m.reg.Unregister(m.base + "/annReceived")
	m.reg.Unregister(m.base + "/annSent")
}
