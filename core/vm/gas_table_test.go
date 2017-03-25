package vm

import (
	"math"
	"testing"
)

func TestMemoryGasCost(t *testing.T) {
	size := uint64(math.MaxUint64 - 64)
	_, err := memoryGasCost(&Memory{}, size)
	if err != nil {
		t.Error("didn't expect error:", err)
	}

	_, err = memoryGasCost(&Memory{}, size+32)
	if err != nil {
		t.Error("didn't expect error:", err)
	}

	_, err = memoryGasCost(&Memory{}, size+33)
	if err == nil {
		t.Error("expected error")
	}
}
