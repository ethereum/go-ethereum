// Copyright 2025 The go-ethereum Authors
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

package common

import "time"

// CalculateETA calculates the estimated remaining time based on the
// number of finished task, remaining task, and the time cost for finished task.
func CalculateETA(done, left uint64, elapsed time.Duration) time.Duration {
	if done == 0 || elapsed.Milliseconds() == 0 {
		return 0
	}

	speed := float64(done) / float64(elapsed.Milliseconds())
	return time.Duration(float64(left)/speed) * time.Millisecond
}
