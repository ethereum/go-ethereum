package live

import "github.com/ethereum/go-ethereum/common"

func extractAddres(addrs []*common.Address) map[common.Address]struct{} {
	result := make(map[common.Address]struct{}, len(addrs))
	for _, addr := range addrs {
		if addr != nil {
			result[*addr] = struct{}{}
		}
	}
	return result
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
