// Copyright 2016 The go-ethereum Authors
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

package les

import (
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
)

var (
	miscInPacketsMeter  = metrics.NewRegisteredMeter("les/misc/in/packets", nil)
	miscInTrafficMeter  = metrics.NewRegisteredMeter("les/misc/in/traffic", nil)
	miscOutPacketsMeter = metrics.NewRegisteredMeter("les/misc/out/packets", nil)
	miscOutTrafficMeter = metrics.NewRegisteredMeter("les/misc/out/traffic", nil)

	connectionTimer = metrics.NewRegisteredTimer("les/connectionTime", nil)

	totalConnectedGauge   = metrics.NewRegisteredGauge("les/server/totalConnected", nil)
	totalCapacityGauge    = metrics.NewRegisteredGauge("les/server/totalCapacity", nil)
	totalRechargeGauge    = metrics.NewRegisteredGauge("les/server/totalRecharge", nil)
	blockProcessingTimer  = metrics.NewRegisteredTimer("les/server/blockProcessingTime", nil)
	requestServedTimer    = metrics.NewRegisteredTimer("les/server/requestServed", nil)
	requestServedMeter    = metrics.NewRegisteredMeter("les/server/totalRequestServed", nil)
	requestEstimatedMeter = metrics.NewRegisteredMeter("les/server/totalRequestEstimated", nil)
	relativeCostHistogram = metrics.NewRegisteredHistogram("les/server/relativeCost", nil, metrics.NewExpDecaySample(1028, 0.015))
	recentServedGauge     = metrics.NewRegisteredGauge("les/server/recentRequestServed", nil)
	recentEstimatedGauge  = metrics.NewRegisteredGauge("les/server/recentRequestEstimated", nil)
	sqServedGauge         = metrics.NewRegisteredGauge("les/server/servingQueue/served", nil)
	sqQueuedGauge         = metrics.NewRegisteredGauge("les/server/servingQueue/queued", nil)
	clientConnectedMeter  = metrics.NewRegisteredMeter("les/server/clientEvent/connected", nil)
	clientRejectedMeter   = metrics.NewRegisteredMeter("les/server/clientEvent/rejected", nil)
	clientKickedMeter     = metrics.NewRegisteredMeter("les/server/clientEvent/kicked", nil)
	// clientDisconnectedMeter = metrics.NewRegisteredMeter("les/server/clientEvent/disconnected", nil)
	clientFreezeMeter = metrics.NewRegisteredMeter("les/server/clientEvent/freeze", nil)
	clientErrorMeter  = metrics.NewRegisteredMeter("les/server/clientEvent/error", nil)
)

// meteredMsgReadWriter is a wrapper around a p2p.MsgReadWriter, capable of
// accumulating the above defined metrics based on the data stream contents.
type meteredMsgReadWriter struct {
	p2p.MsgReadWriter     // Wrapped message stream to meter
	version           int // Protocol version to select correct meters
}

// newMeteredMsgWriter wraps a p2p MsgReadWriter with metering support. If the
// metrics system is disabled, this function returns the original object.
func newMeteredMsgWriter(rw p2p.MsgReadWriter) p2p.MsgReadWriter {
	if !metrics.Enabled {
		return rw
	}
	return &meteredMsgReadWriter{MsgReadWriter: rw}
}

// Init sets the protocol version used by the stream to know which meters to
// increment in case of overlapping message ids between protocol versions.
func (rw *meteredMsgReadWriter) Init(version int) {
	rw.version = version
}

func (rw *meteredMsgReadWriter) ReadMsg() (p2p.Msg, error) {
	// Read the message and short circuit in case of an error
	msg, err := rw.MsgReadWriter.ReadMsg()
	if err != nil {
		return msg, err
	}
	// Account for the data traffic
	packets, traffic := miscInPacketsMeter, miscInTrafficMeter
	packets.Mark(1)
	traffic.Mark(int64(msg.Size))

	return msg, err
}

func (rw *meteredMsgReadWriter) WriteMsg(msg p2p.Msg) error {
	// Account for the data traffic
	packets, traffic := miscOutPacketsMeter, miscOutTrafficMeter
	packets.Mark(1)
	traffic.Mark(int64(msg.Size))

	// Send the packet to the p2p layer
	return rw.MsgReadWriter.WriteMsg(msg)
}
