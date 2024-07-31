package vm

import (
	"testing"

	"github.com/holiman/uint256"
)

func BenchmarkStackSwap(b *testing.B) {
	stack := newstack()
	stack.push(new(uint256.Int))
	stack.push(new(uint256.Int))

	b.Run("swap(1)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			stack.swap(1)
		}
	})

	b.Run("swap1()", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			stack.swap1()
		}
	})
}
