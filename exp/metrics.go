// Copyright 2015 The go-expanse Authors
// This file is part of the go-expanse library.
//
// The go-expanse library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-expanse library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-expanse library. If not, see <http://www.gnu.org/licenses/>.

package exp

import (
	"github.com/expanse-project/go-expanse/metrics"
)

var (
	propTxnInPacketsMeter    = metrics.NewMeter("exp/prop/txns/in/packets")
	propTxnInTrafficMeter    = metrics.NewMeter("exp/prop/txns/in/traffic")
	propTxnOutPacketsMeter   = metrics.NewMeter("exp/prop/txns/out/packets")
	propTxnOutTrafficMeter   = metrics.NewMeter("exp/prop/txns/out/traffic")
	propHashInPacketsMeter   = metrics.NewMeter("exp/prop/hashes/in/packets")
	propHashInTrafficMeter   = metrics.NewMeter("exp/prop/hashes/in/traffic")
	propHashOutPacketsMeter  = metrics.NewMeter("exp/prop/hashes/out/packets")
	propHashOutTrafficMeter  = metrics.NewMeter("exp/prop/hashes/out/traffic")
	propBlockInPacketsMeter  = metrics.NewMeter("exp/prop/blocks/in/packets")
	propBlockInTrafficMeter  = metrics.NewMeter("exp/prop/blocks/in/traffic")
	propBlockOutPacketsMeter = metrics.NewMeter("exp/prop/blocks/out/packets")
	propBlockOutTrafficMeter = metrics.NewMeter("exp/prop/blocks/out/traffic")
	reqHashInPacketsMeter    = metrics.NewMeter("exp/req/hashes/in/packets")
	reqHashInTrafficMeter    = metrics.NewMeter("exp/req/hashes/in/traffic")
	reqHashOutPacketsMeter   = metrics.NewMeter("exp/req/hashes/out/packets")
	reqHashOutTrafficMeter   = metrics.NewMeter("exp/req/hashes/out/traffic")
	reqBlockInPacketsMeter   = metrics.NewMeter("exp/req/blocks/in/packets")
	reqBlockInTrafficMeter   = metrics.NewMeter("exp/req/blocks/in/traffic")
	reqBlockOutPacketsMeter  = metrics.NewMeter("exp/req/blocks/out/packets")
	reqBlockOutTrafficMeter  = metrics.NewMeter("exp/req/blocks/out/traffic")
)
