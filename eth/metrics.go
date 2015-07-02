// Copyright 2015 The go-ethereum Authors
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

import (
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
)

var (
	propTxnInPacketsMeter    = metrics.NewMeter("eth/prop/txns/in/packets")
	propTxnInTrafficMeter    = metrics.NewMeter("eth/prop/txns/in/traffic")
	propTxnOutPacketsMeter   = metrics.NewMeter("eth/prop/txns/out/packets")
	propTxnOutTrafficMeter   = metrics.NewMeter("eth/prop/txns/out/traffic")
	propHashInPacketsMeter   = metrics.NewMeter("eth/prop/hashes/in/packets")
	propHashInTrafficMeter   = metrics.NewMeter("eth/prop/hashes/in/traffic")
	propHashOutPacketsMeter  = metrics.NewMeter("eth/prop/hashes/out/packets")
	propHashOutTrafficMeter  = metrics.NewMeter("eth/prop/hashes/out/traffic")
	propBlockInPacketsMeter  = metrics.NewMeter("eth/prop/blocks/in/packets")
	propBlockInTrafficMeter  = metrics.NewMeter("eth/prop/blocks/in/traffic")
	propBlockOutPacketsMeter = metrics.NewMeter("eth/prop/blocks/out/packets")
	propBlockOutTrafficMeter = metrics.NewMeter("eth/prop/blocks/out/traffic")
	reqHashInPacketsMeter    = metrics.NewMeter("eth/req/hashes/in/packets")
	reqHashInTrafficMeter    = metrics.NewMeter("eth/req/hashes/in/traffic")
	reqHashOutPacketsMeter   = metrics.NewMeter("eth/req/hashes/out/packets")
	reqHashOutTrafficMeter   = metrics.NewMeter("eth/req/hashes/out/traffic")
	reqBlockInPacketsMeter   = metrics.NewMeter("eth/req/blocks/in/packets")
	reqBlockInTrafficMeter   = metrics.NewMeter("eth/req/blocks/in/traffic")
	reqBlockOutPacketsMeter  = metrics.NewMeter("eth/req/blocks/out/packets")
	reqBlockOutTrafficMeter  = metrics.NewMeter("eth/req/blocks/out/traffic")
	reqHeaderInPacketsMeter  = metrics.NewMeter("eth/req/header/in/packets")
	reqHeaderInTrafficMeter  = metrics.NewMeter("eth/req/header/in/traffic")
	reqHeaderOutPacketsMeter = metrics.NewMeter("eth/req/header/out/packets")
	reqHeaderOutTrafficMeter = metrics.NewMeter("eth/req/header/out/traffic")
	reqStateInPacketsMeter   = metrics.NewMeter("eth/req/state/in/packets")
	reqStateInTrafficMeter   = metrics.NewMeter("eth/req/state/in/traffic")
	reqStateOutPacketsMeter  = metrics.NewMeter("eth/req/state/out/packets")
	reqStateOutTrafficMeter  = metrics.NewMeter("eth/req/state/out/traffic")
	miscInPacketsMeter       = metrics.NewMeter("eth/misc/in/packets")
	miscInTrafficMeter       = metrics.NewMeter("eth/misc/in/traffic")
	miscOutPacketsMeter      = metrics.NewMeter("eth/misc/out/packets")
	miscOutTrafficMeter      = metrics.NewMeter("eth/misc/out/traffic")
)

// meteredMsgReadWriter is a wrapper around a p2p.MsgReadWriter, capable of
// accumulating the above defined metrics based on the data stream contents.
type meteredMsgReadWriter struct {
	p2p.MsgReadWriter
}

// newMeteredMsgWriter wraps a p2p MsgReadWriter with metering support. If the
// metrics system is disabled, this fucntion returns the original object.
func newMeteredMsgWriter(rw p2p.MsgReadWriter) p2p.MsgReadWriter {
	if !metrics.Enabled {
		return rw
	}
	return &meteredMsgReadWriter{rw}
}

func (rw *meteredMsgReadWriter) ReadMsg() (p2p.Msg, error) {
	// Read the message and short circuit in case of an error
	msg, err := rw.MsgReadWriter.ReadMsg()
	if err != nil {
		return msg, err
	}
	// Account for the data traffic
	packets, traffic := miscInPacketsMeter, miscInTrafficMeter
	switch msg.Code {
	case BlockHashesMsg:
		packets, traffic = reqHashInPacketsMeter, reqHashInTrafficMeter
	case BlocksMsg:
		packets, traffic = reqBlockInPacketsMeter, reqBlockInTrafficMeter
	case BlockHeadersMsg:
		packets, traffic = reqHeaderInPacketsMeter, reqHeaderInTrafficMeter
	case NodeDataMsg:
		packets, traffic = reqStateInPacketsMeter, reqStateInTrafficMeter
	case NewBlockHashesMsg:
		packets, traffic = propHashInPacketsMeter, propHashInTrafficMeter
	case NewBlockMsg:
		packets, traffic = propBlockInPacketsMeter, propBlockInTrafficMeter
	case TxMsg:
		packets, traffic = propTxnInPacketsMeter, propTxnInTrafficMeter
	}
	packets.Mark(1)
	traffic.Mark(int64(msg.Size))

	return msg, err
}

func (rw *meteredMsgReadWriter) WriteMsg(msg p2p.Msg) error {
	// Account for the data traffic
	packets, traffic := miscOutPacketsMeter, miscOutTrafficMeter
	switch msg.Code {
	case BlockHashesMsg:
		packets, traffic = reqHashOutPacketsMeter, reqHashOutTrafficMeter
	case BlocksMsg:
		packets, traffic = reqBlockOutPacketsMeter, reqBlockOutTrafficMeter
	case BlockHeadersMsg:
		packets, traffic = reqHeaderOutPacketsMeter, reqHeaderOutTrafficMeter
	case NodeDataMsg:
		packets, traffic = reqStateOutPacketsMeter, reqStateOutTrafficMeter
	case NewBlockHashesMsg:
		packets, traffic = propHashOutPacketsMeter, propHashOutTrafficMeter
	case NewBlockMsg:
		packets, traffic = propBlockOutPacketsMeter, propBlockOutTrafficMeter
	case TxMsg:
		packets, traffic = propTxnOutPacketsMeter, propTxnOutTrafficMeter
	}
	packets.Mark(1)
	traffic.Mark(int64(msg.Size))

	// Send the packet to the p2p layer
	return rw.MsgReadWriter.WriteMsg(msg)
}
