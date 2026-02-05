// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

//go:build tamago && riscv64

package zisk

import (
	"unsafe"
)

const (
	// ZisK I/O addresses
	INPUT_ADDR     = 0x90000000
	QEMU_EXIT_ADDR = 0x100000
	QEMU_EXIT_CODE = 0x5555
	OUTPUT_ADDR    = 0xa001_0000
	UART_ADDR      = 0xa000_0200
	ARCH_ID_ZISK   = 0xFFFEEEE // TEMPORARY  // TODO register one
	MAX_INPUT      = 0x2000
	MAX_OUTPUT     = 0x1_0000
)

var outputCount uint32 = 0

//go:linkname ramStart runtime.ramStart
var ramStart uint64 = 0xa0020000 // Match ZisK's RAM location

//go:linkname ramSize runtime.ramSize
var ramSize uint64 = 0x1FFE0000 // Match ZisK's RAM size (~512MB)

// ramStackOffset is always defined here as there's no linkramstackoffset build tag
//
//go:linkname ramStackOffset runtime.ramStackOffset
var ramStackOffset uint64 = 0x100000 // 1MB stack (matching ZisK)

// Bloc sets the heap start address to bypass initBloc()
//
//go:linkname Bloc runtime.Bloc
var Bloc uintptr = 0xa0100000 // Start heap after stack (ramStart + ramStackOffset)

// printk implementation for zkVM
//
//go:linkname printk runtime.printk
func printk(c byte) {
	// TODO: This is a stub. Just write to the output address
	// Write directly to OUTPUT_ADDR
	// Format: [count:u32][data:bytes]
	// First update the count at OUTPUT_ADDR
	outputCount++
	*(*uint32)(unsafe.Pointer(uintptr(OUTPUT_ADDR))) = outputCount

	// Write the byte at OUTPUT_ADDR + 4 + (outputCount-1)
	*(*byte)(unsafe.Pointer(uintptr(OUTPUT_ADDR + 4 + outputCount - 1))) = c
}

// hwinit1 is now defined in hwinit1.s
// we use it to set A0/A1 registers to the input and output address

// Use this as a stub timer. It is all single threaded, and there is no concept of time.
// This may return the cycle count in the future.
var timer int64 = 0

//go:linkname nanotime1 runtime.nanotime1
func nanotime1() int64 {
	// Return deterministic time for zkVM
	// Could be based on instruction count or fixed increments
	timer++
	return timer * 1000
}

//go:linkname initRNG runtime.initRNG
func initRNG() {
	// Deterministic RNG initialization
	// TODO: There is no "proper" rng so nothing to init.
}

//go:linkname getRandomData runtime.getRandomData
func getRandomData(b []byte) {
	// Deterministic "random" data
	// In a real zkVM, this might come from the input
	for i := range b {
		b[i] = byte(i & 0xFF)
	}
}

// Init initializes the zkVM board
func Init() {
	timer = 0
}

// Shutdown is defined in shutdown.s and uses ecall to exit
func Shutdown()
