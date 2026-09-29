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
	"testing"

	"github.com/ethereum/go-ethereum/params"
)

func TestGasPoolAvailable(t *testing.T) {
	// Before Amsterdam the plain remainder is all that counts.
	gp := NewGasPool(60_000_000)
	if err := gp.CheckGasLegacy(40_000_000); err != nil {
		t.Fatal(err)
	}
	if err := gp.ChargeGasLegacy(10_000_000, 30_000_000); err != nil {
		t.Fatal(err)
	}
	if have, want := gp.Available(false), uint64(30_000_000); have != want {
		t.Fatalf("legacy: have %d, want %d", have, want)
	}
	// After Amsterdam a transaction reserves min(limit, MaxTxGas) execution
	// and its full limit state gas. With 28M execution left, more than a cap,
	// any limit fits the execution dimension, so only the 55M of state left
	// bounds the limit.
	gp = NewGasPool(60_000_000)
	if err := gp.ChargeGasAmsterdam(32_000_000, 5_000_000, 32_000_000); err != nil {
		t.Fatal(err)
	}
	if have, want := gp.Available(true), uint64(55_000_000); have != want {
		t.Fatalf("amsterdam, execution unbound: have %d, want %d", have, want)
	}
	// With less than a cap of execution gas left, both dimensions bind.
	gp = NewGasPool(60_000_000)
	if err := gp.ChargeGasAmsterdam(50_000_000, 20_000_000, 50_000_000); err != nil {
		t.Fatal(err)
	}
	if have, want := gp.Available(true), uint64(10_000_000); have != want {
		t.Fatalf("amsterdam, execution bound: have %d, want %d", have, want)
	}
	gp = NewGasPool(60_000_000)
	if err := gp.ChargeGasAmsterdam(50_000_000, 55_000_000, 55_000_000); err != nil {
		t.Fatal(err)
	}
	if have, want := gp.Available(true), uint64(5_000_000); have != want {
		t.Fatalf("amsterdam, state bound: have %d, want %d", have, want)
	}
	// Available agrees with the reservation check on either side of the cap.
	gp = NewGasPool(60_000_000)
	if err := gp.ChargeGasAmsterdam(32_000_000, 0, 32_000_000); err != nil {
		t.Fatal(err)
	}
	for _, gas := range []uint64{params.MaxTxGas - 1, params.MaxTxGas, 30_000_000, 60_000_000, 60_000_001} {
		fits := gp.CheckGasAmsterdam(min(gas, params.MaxTxGas), gas) == nil
		if have := gas <= gp.Available(true); have != fits {
			t.Fatalf("gas %d: available says %v, check says %v", gas, have, fits)
		}
	}
}
