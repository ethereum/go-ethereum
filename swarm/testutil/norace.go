// Copyright 2019 The go-ethereum Authors
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

// +build !race

package testutil

// RaceEnabled is true when -race flag is provided to the go tool. This const
// might be used in tests to skip some cases as the race detector may increase
// memory usage 5-10x and execution time by 2-20x. That might causes problems
// on Travis. Please, use this flag sparingly and keep your unit tests
// as light on resources as possible.
const RaceEnabled = false
