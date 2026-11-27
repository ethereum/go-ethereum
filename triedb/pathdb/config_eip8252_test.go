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

package pathdb

import (
	"testing"

	"github.com/ethereum/go-ethereum/params"
)

// TestDefaultStateHistoryCoversReorgRetentionWindow guards EIP-8252: the default
// state history must retain at least REORG_RETENTION_WINDOW recent blocks so the
// state of any block forking within that window can be reconstructed without
// resyncing. A zero default (full chain) also satisfies the requirement.
func TestDefaultStateHistoryCoversReorgRetentionWindow(t *testing.T) {
	if got := params.ReorgRetentionWindow; got != 8192*32 {
		t.Fatalf("ReorgRetentionWindow = %d, want 8192 epochs * 32 slots = %d", got, 8192*32)
	}
	if h := Defaults.StateHistory; h != 0 && h < params.ReorgRetentionWindow {
		t.Fatalf("default StateHistory = %d is below EIP-8252 REORG_RETENTION_WINDOW = %d", h, params.ReorgRetentionWindow)
	}
}
