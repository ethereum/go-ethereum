// Copyright 2020 The go-ethereum Authors
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

//go:build !ios && !js && !wasip1 && !tinygo
// +build !ios,!js,!wasip1,!tinygo

package metrics

import "github.com/ethereum/go-ethereum/log"

// ReadCPUStats retrieves the current CPU stats.
func ReadCPUStats(stats *CPUStats) {
	globalTime, globalWait, err := readCPUTimes()
	if err != nil {
		log.Error("Could not read cpu stats", "err", err)
		return
	}
	stats.GlobalTime = globalTime
	stats.GlobalWait = globalWait
	stats.LocalTime = getProcessCPUTime()
}
