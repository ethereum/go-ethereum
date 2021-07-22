package DDosAttack

import (
	"fmt"
	"testing"
)

func TestGetBlockNumber(t *testing.T) {
	fmt.Print(GetBlockNumber())
}

func TestTraceBlock(t *testing.T) {
	number := int64(317)
	fmt.Print(TraceBlock(number))
}
