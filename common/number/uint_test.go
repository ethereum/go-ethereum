// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package number

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestSet(t *testing.T) {
	a := Uint(0)
	b := Uint(10)
	a.Set(b)
	if a.num.Cmp(b.num) != 0 {
		t.Error("didn't compare", a, b)
	}

	c := Uint(0).SetBytes(common.Hex2Bytes("0a"))
	if c.num.Cmp(big.NewInt(10)) != 0 {
		t.Error("c set bytes failed.")
	}
}

func TestInitialiser(t *testing.T) {
	check := false
	init := NewInitialiser(func(x *Number) *Number {
		check = true
		return x
	})
	a := init(0).Add(init(1), init(2))
	if a.Cmp(init(3)) != 0 {
		t.Error("expected 3. got", a)
	}
	if !check {
		t.Error("expected limiter to be called")
	}
}

func TestGet(t *testing.T) {
	a := Uint(10)
	if a.Uint64() != 10 {
		t.Error("expected to get 10. got", a.Uint64())
	}

	a = Uint(10)
	if a.Int64() != 10 {
		t.Error("expected to get 10. got", a.Int64())
	}
}

func TestCmp(t *testing.T) {
	a := Uint(10)
	b := Uint(10)
	c := Uint(11)

	if a.Cmp(b) != 0 {
		t.Error("a b == 0 failed", a, b)
	}

	if a.Cmp(c) >= 0 {
		t.Error("a c < 0 failed", a, c)
	}

	if c.Cmp(b) <= 0 {
		t.Error("c b > 0 failed", c, b)
	}
}

func TestMaxArith(t *testing.T) {
	a := Uint(0).Add(MaxUint256, One)
	if a.Cmp(Zero) != 0 {
		t.Error("expected max256 + 1 = 0 got", a)
	}

	a = Uint(0).Sub(Uint(0), One)
	if a.Cmp(MaxUint256) != 0 {
		t.Error("expected 0 - 1 = max256 got", a)
	}

	a = Int(0).Sub(Int(0), One)
	if a.Cmp(MinOne) != 0 {
		t.Error("expected 0 - 1 = -1 got", a)
	}
}

func TestConversion(t *testing.T) {
	a := Int(-1)
	b := a.Uint256()
	if b.Cmp(MaxUint256) != 0 {
		t.Error("expected -1 => unsigned to return max. got", b)
	}
}
