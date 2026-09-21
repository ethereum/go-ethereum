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

//go:build windows

package metrics

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32           = windows.NewLazySystemDLL("kernel32.dll")
	procGetSystemTimes = kernel32.NewProc("GetSystemTimes")
)

// readCPUTimes returns the aggregated busy time of all CPUs, in seconds since
// boot. Windows has no iowait accounting, so globalWait is 0.
func readCPUTimes() (globalTime, globalWait float64, err error) {
	var idle, kernel, user windows.Filetime
	r1, _, callErr := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idle)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)))
	if r1 == 0 {
		return 0, 0, callErr
	}
	// The FILETIME values count 100ns intervals; the kernel time includes the
	// idle time, which needs to be subtracted to obtain the busy time.
	ticks := func(ft windows.Filetime) uint64 {
		return uint64(ft.HighDateTime)<<32 | uint64(ft.LowDateTime)
	}
	busy := ticks(user) + ticks(kernel) - ticks(idle)
	return float64(busy) / 1e7, 0, nil
}
