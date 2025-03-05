// Copyright 2023 The go-ethereum Authors
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

package ethconfig

import "fmt"

// HistoryMode represents the blockchain history mode for pruning.
type HistoryMode uint32

const (
	AllHistory    HistoryMode = iota // Keep all history
	PrunedHistory                    // Prune history beyond StateHistory and TransactionHistory
)

func (mode HistoryMode) IsValid() bool {
	return mode == AllHistory || mode == PrunedHistory
}

// String implements the stringer interface.
func (mode HistoryMode) String() string {
	switch mode {
	case AllHistory:
		return "full"
	case PrunedHistory:
		return "pruned"
	default:
		return "unknown"
	}
}

func (mode HistoryMode) MarshalText() ([]byte, error) {
	switch mode {
	case AllHistory:
		return []byte("full"), nil
	case PrunedHistory:
		return []byte("pruned"), nil
	default:
		return nil, fmt.Errorf("unknown history mode %d", mode)
	}
}

func (mode *HistoryMode) UnmarshalText(text []byte) error {
	switch string(text) {
	case "full":
		*mode = AllHistory
	case "pruned":
		*mode = PrunedHistory
	default:
		return fmt.Errorf(`unknown history mode %q, want "full" or "pruned"`, text)
	}
	return nil
}
