// Copyright 2018 The go-ethereum Authors
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
	"testing"
)

func TestIntPoolPoolGet(t *testing.T) {
	poolOfIntPools.pools = make([]*intPool, 0, poolDefaultCap)

	nip := poolOfIntPools.get()
	if nip == nil {
		t.Fatalf("Invalid pool allocation")
	}
}

func TestIntPoolPoolPut(t *testing.T) {
	poolOfIntPools.pools = make([]*intPool, 0, poolDefaultCap)

	nip := poolOfIntPools.get()
	if len(poolOfIntPools.pools) != 0 {
		t.Fatalf("Pool got added to list when none should have been")
	}

	poolOfIntPools.put(nip)
	if len(poolOfIntPools.pools) == 0 {
		t.Fatalf("Pool did not get added to list when one should have been")
	}
}

func TestIntPoolPoolReUse(t *testing.T) {
	poolOfIntPools.pools = make([]*intPool, 0, poolDefaultCap)
	nip := poolOfIntPools.get()
	poolOfIntPools.put(nip)
	poolOfIntPools.get()

	if len(poolOfIntPools.pools) != 0 {
		t.Fatalf("Invalid number of pools. Got %d, expected %d", len(poolOfIntPools.pools), 0)
	}
}
