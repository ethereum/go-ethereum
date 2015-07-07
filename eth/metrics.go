// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package eth

import (
	"github.com/ethereum/go-ethereum/metrics"
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
)
