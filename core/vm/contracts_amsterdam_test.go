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

package vm

import (
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

// TestActivePrecompilesAmsterdamWithoutOsaka checks that a chain which
// activates Amsterdam without an explicit Osaka timestamp (a valid config,
// both timestamps are optional) still exposes the Osaka precompile set.
// Amsterdam is a superset of Osaka, so the P256VERIFY precompile at 0x0100
// (EIP-7951, shipped in Osaka) must be active both in the execution set and
// in the warm-address list.
func TestActivePrecompilesAmsterdamWithoutOsaka(t *testing.T) {
	config := *params.MergedTestChainConfig
	config.OsakaTime = nil
	config.AmsterdamTime = new(uint64)
	if err := config.CheckConfigForkOrder(); err != nil {
		t.Fatalf("config with Amsterdam but no Osaka rejected: %v", err)
	}
	rules := config.Rules(big.NewInt(0), true, 0)
	if !rules.IsAmsterdam || rules.IsOsaka {
		t.Fatalf("unexpected rules: IsAmsterdam=%v, IsOsaka=%v", rules.IsAmsterdam, rules.IsOsaka)
	}
	p256 := common.BytesToAddress([]byte{0x01, 0x00})
	if _, ok := ActivePrecompiledContracts(rules)[p256]; !ok {
		t.Errorf("P256VERIFY precompile %v missing from execution set at Amsterdam", p256)
	}
	if !slices.Contains(ActivePrecompiles(rules), p256) {
		t.Errorf("P256VERIFY precompile %v missing from warm-address list at Amsterdam", p256)
	}
}
