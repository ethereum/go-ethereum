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
