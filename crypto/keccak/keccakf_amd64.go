//go:build amd64 && !purego

package keccak

import (
	"unsafe"

	"golang.org/x/sys/cpu"
)

func init() { useASM = cpu.X86.HasBMI2 }

//go:noescape
func keccakF1600BMI2(a *[200]byte)

func keccakF1600(a *[200]byte) {
	keccakF1600BMI2(a)
}

func xorAndPermute(state *[200]byte, buf *byte) {
	xorIn(state, unsafe.Slice(buf, rate))
	keccakF1600(state)
}
