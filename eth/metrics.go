package eth

import "github.com/rcrowley/go-metrics"

var (
	propTxnInPacketsMeter    = metrics.GetOrRegisterMeter("eth/prop/txns/in/packets", metrics.DefaultRegistry)
	propTxnInTrafficMeter    = metrics.GetOrRegisterMeter("eth/prop/txns/in/traffic", metrics.DefaultRegistry)
	propTxnOutPacketsMeter   = metrics.GetOrRegisterMeter("eth/prop/txns/out/packets", metrics.DefaultRegistry)
	propTxnOutTrafficMeter   = metrics.GetOrRegisterMeter("eth/prop/txns/out/traffic", metrics.DefaultRegistry)
	propHashInPacketsMeter   = metrics.GetOrRegisterMeter("eth/prop/hashes/in/packets", metrics.DefaultRegistry)
	propHashInTrafficMeter   = metrics.GetOrRegisterMeter("eth/prop/hashes/in/traffic", metrics.DefaultRegistry)
	propHashOutPacketsMeter  = metrics.GetOrRegisterMeter("eth/prop/hashes/out/packets", metrics.DefaultRegistry)
	propHashOutTrafficMeter  = metrics.GetOrRegisterMeter("eth/prop/hashes/out/traffic", metrics.DefaultRegistry)
	propBlockInPacketsMeter  = metrics.GetOrRegisterMeter("eth/prop/blocks/in/packets", metrics.DefaultRegistry)
	propBlockInTrafficMeter  = metrics.GetOrRegisterMeter("eth/prop/blocks/in/traffic", metrics.DefaultRegistry)
	propBlockOutPacketsMeter = metrics.GetOrRegisterMeter("eth/prop/blocks/out/packets", metrics.DefaultRegistry)
	propBlockOutTrafficMeter = metrics.GetOrRegisterMeter("eth/prop/blocks/out/traffic", metrics.DefaultRegistry)
	reqHashInPacketsMeter    = metrics.GetOrRegisterMeter("eth/req/hashes/in/packets", metrics.DefaultRegistry)
	reqHashInTrafficMeter    = metrics.GetOrRegisterMeter("eth/req/hashes/in/traffic", metrics.DefaultRegistry)
	reqHashOutPacketsMeter   = metrics.GetOrRegisterMeter("eth/req/hashes/out/packets", metrics.DefaultRegistry)
	reqHashOutTrafficMeter   = metrics.GetOrRegisterMeter("eth/req/hashes/out/traffic", metrics.DefaultRegistry)
	reqBlockInPacketsMeter   = metrics.GetOrRegisterMeter("eth/req/blocks/in/packets", metrics.DefaultRegistry)
	reqBlockInTrafficMeter   = metrics.GetOrRegisterMeter("eth/req/blocks/in/traffic", metrics.DefaultRegistry)
	reqBlockOutPacketsMeter  = metrics.GetOrRegisterMeter("eth/req/blocks/out/packets", metrics.DefaultRegistry)
	reqBlockOutTrafficMeter  = metrics.GetOrRegisterMeter("eth/req/blocks/out/traffic", metrics.DefaultRegistry)
)
