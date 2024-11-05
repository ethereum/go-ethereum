package vm

import (
	"testing"

	"github.com/holiman/uint256"
)

func BenchmarkStack(b *testing.B) {
	stack := newStackForTesting()
	for i := 0; i < b.N; i++ {
		stack.push(uint256.Int{1})
		stack.pushBytes([]byte{12, 34})
		stack.pushU64(1234)
	}
}
