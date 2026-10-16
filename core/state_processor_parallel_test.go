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

package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/params"
)

func TestSequentialExecutionReason(t *testing.T) {
	var (
		amsterdam = balChainConfig()
		header    = &types.Header{Number: big.NewInt(1), Time: 1}
		withList  = types.NewBlockWithHeader(header).WithAccessListUnsafe(&bal.BlockAccessList{})
		noList    = types.NewBlockWithHeader(header)
	)
	tests := []struct {
		name                     string
		block                    *types.Block
		config                   *params.ChainConfig
		witness, trace, disabled bool
		want                     string
	}{
		{"parallel", withList, amsterdam, false, false, false, ""},
		{"disabled", withList, amsterdam, true, true, true, "disabled"},
		{"tracer", withList, amsterdam, true, true, false, "tracer"},
		{"witness", withList, amsterdam, true, false, false, "witness"},
		{"no access list", noList, params.MergedTestChainConfig, false, false, false, "no-access-list"},
		{"pre-amsterdam", withList, params.MergedTestChainConfig, false, false, false, "pre-amsterdam"},
	}
	for _, tt := range tests {
		got := sequentialExecutionReason(tt.block, tt.config, tt.witness, tt.trace, tt.disabled)
		if got != tt.want {
			t.Errorf("%s: reason %q, want %q", tt.name, got, tt.want)
		}
		if parallel := supportsParallelExecution(tt.block, tt.config, tt.witness, tt.trace, tt.disabled); parallel != (tt.want == "") {
			t.Errorf("%s: supportsParallelExecution %v, want %v", tt.name, parallel, tt.want == "")
		}
	}
}
