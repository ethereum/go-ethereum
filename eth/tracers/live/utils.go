package live

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mitchellh/mapstructure"
)

func extractAddres(addrs []*common.Address) map[common.Address]struct{} {
	result := make(map[common.Address]struct{}, len(addrs))
	for _, addr := range addrs {
		if addr != nil {
			result[*addr] = struct{}{}
		}
	}
	return result
}

func containsAddress(addrs map[common.Address]struct{}, addr *common.Address) bool {
	if addr == nil {
		return false
	}
	_, ok := addrs[*addr]
	return ok
}

func filterParityTrace(trace interface{}, fromAddrs, toAddrs map[common.Address]struct{}, mode TraceFilterMode) bool {
	var pt native.ParityTrace
	if err := mapstructure.Decode(trace, &pt); err != nil {
		log.Error("Failed to convert into ParityTrace", "err", err)
		return false
	}
	var fromAddr, toAddr *common.Address
	switch pt.Type {
	case "call":
		fromAddr = pt.Action.From
		toAddr = pt.Action.To
	case "create":
		fromAddr = pt.Action.From
		if pt.Result != nil {
			toAddr = pt.Result.Address
		}
	case "suicide":
		fromAddr = pt.Action.SelfDestructed
		toAddr = pt.Action.RefundAddress
	default:
		// No matching for other types
		return false
	}

	fromMatch := len(fromAddrs) == 0 || containsAddress(fromAddrs, fromAddr)
	toMatch := len(toAddrs) == 0 || containsAddress(toAddrs, toAddr)

	if mode == TraceFilterModeIntersection {
		return fromMatch && toMatch
	}
	return fromMatch || toMatch
}

func exportLimitedTraces(gen func(blknum uint64) ([]interface{}, error), fromBlock, toBlock, count, after uint64) ([]interface{}, error) {
	var (
		nExported uint64                         // Number of traces exported
		nSkipped  uint64                         // Number of traces skipped
		results   = make([]interface{}, 0, 1024) // 1024 is the initial capacity
	)

	for blknum := fromBlock; blknum <= toBlock && nExported < count; blknum++ {
		traces, err := gen(uint64(blknum))
		if err != nil {
			return nil, err
		}

		nTraces := uint64(len(traces))
		if after > nSkipped {
			skip := min(after-nSkipped, nTraces)
			nSkipped += skip
			if skip == nTraces {
				// Skip if the whole block is skipped
				continue
			}
			traces = traces[skip:]
		}

		// Export at most the remaining traces
		maxExport := min(count-nExported, uint64(len(traces)))
		results = append(results, traces[:maxExport]...)
		nExported += maxExport
	}
	return results, nil
}
