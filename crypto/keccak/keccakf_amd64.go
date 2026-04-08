//go:build amd64 && !purego

package keccak

import "golang.org/x/sys/cpu"

func init() { useASM = cpu.X86.HasBMI1 && cpu.X86.HasBMI2 }

// keccakF1600BMI2 permutes state. When buf != nil, it first XORs rate bytes
// of buf into state, saving one full memory pass.
//
//go:noescape
func keccakF1600BMI2(a *[200]byte, buf *byte)

func keccakF1600(a *[200]byte) {
	keccakF1600BMI2(a, nil)
}

func xorAndPermute(state *[200]byte, buf *byte) {
	keccakF1600BMI2(state, buf)
}
