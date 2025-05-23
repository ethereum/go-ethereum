package tracersutils

import "github.com/ethereum/go-ethereum/core/vm"

type TraceBlockMetadata struct {
	ShouldIncludeInTraceResult bool
	// must be set if shouldIncludeInTraceResult
	IdxInEthBlock int
	// must be set if !shouldIncludeInTraceResult
	TraceRunnable func(vm.StateDB)
}
