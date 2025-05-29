package live

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/node"
)

func BenchmarkNoopTracerHooks(b *testing.B) {
	noopHooks, err := newNoopTracer(nil)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		noopHooks.OnTxEnd(&types.Receipt{}, nil)
	}
}

func BenchmarkRecoverTracerHooks(b *testing.B) {
	noopHooks, err := newNoopTracer(nil)
	if err != nil {
		b.Fatal(err)
	}
	stack, err := node.New(&node.Config{})
	if err != nil {
		b.Fatal(err)
	}
	defer stack.Close()

	recoverHooks, err := tracers.NewRecoverTracer(stack, noopHooks, false)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recoverHooks.OnTxEnd(&types.Receipt{}, nil)
	}
}
