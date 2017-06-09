// Copyright 2017 The go-ethereum Authors
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
package pot

//
// import (
// 	"math/rand"
// 	"reflect"
// 	"testing"
//
// 	"github.com/ethereum/go-ethereum/common"
// )
// //
// // func (Address) Generate(rand *rand.Rand, size int) reflect.Value {
// // 	var id Address
// // 	for i := 0; i < len(id); i++ {
// // 		id[i] = byte(uint8(rand.Intn(255)))
// // 	}
// // 	return reflect.ValueOf(id)
// // }
//
// func TestCommonBitsAddrF(t *testing.T) {
// 	a := Address(common.HexToHash("0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
// 	b := Address(common.HexToHash("0x8123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
// 	c := Address(common.HexToHash("0x4123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
// 	d := Address(common.HexToHash("0x0023456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
// 	e := Address(common.HexToHash("0x01A3456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
// 	ab := CommonBitsAddrF(a, b, func() byte { return byte(0x00) }, 10)
// 	expab := Address(common.HexToHash("0x8000000000000000000000000000000000000000000000000000000000000000"))
//
// 	if ab != expab {
// 		t.Fatalf("%v != %v", ab, expab)
// 	}
// 	ac := CommonBitsAddrF(a, c, func() byte { return byte(0x00) }, 10)
// 	expac := Address(common.HexToHash("0x4000000000000000000000000000000000000000000000000000000000000000"))
//
// 	if ac != expac {
// 		t.Fatalf("%v != %v", ac, expac)
// 	}
// 	ad := CommonBitsAddrF(a, d, func() byte { return byte(0x00) }, 10)
// 	expad := Address(common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"))
//
// 	if ad != expad {
// 		t.Fatalf("%v != %v", ad, expad)
// 	}
// 	ae := CommonBitsAddrF(a, e, func() byte { return byte(0x00) }, 10)
// 	expae := Address(common.HexToHash("0x0180000000000000000000000000000000000000000000000000000000000000"))
//
// 	if ae != expae {
// 		t.Fatalf("%v != %v", ae, expae)
// 	}
// 	acf := CommonBitsAddrF(a, c, func() byte { return byte(0xff) }, 10)
// 	expacf := Address(common.HexToHash("0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"))
//
// 	if acf != expacf {
// 		t.Fatalf("%v != %v", acf, expacf)
// 	}
// 	aeo := CommonBitsAddrF(a, e, func() byte { return byte(0x00) }, 2)
// 	expaeo := Address(common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"))
//
// 	if aeo != expaeo {
// 		t.Fatalf("%v != %v", aeo, expaeo)
// 	}
// 	aep := CommonBitsAddrF(a, e, func() byte { return byte(0xff) }, 2)
// 	expaep := Address(common.HexToHash("0x3fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"))
//
// 	if aep != expaep {
// 		t.Fatalf("%v != %v", aep, expaep)
// 	}
//
// }
//
// func TestRandomAddressAt(t *testing.T) {
// 	var a Address
// 	for i := 0; i < 100; i++ {
// 		a = RandomAddress()
// 		prox := rand.Intn(255)
// 		b := RandomAddressAt(a, prox)
// 		p, _ := proximity(a, b)
// 		if p != prox {
// 			t.Fatalf("incorrect address prox(%v, %v) == %v (expected %v)", a, b, p, prox)
// 		}
// 	}
// }
//
// const (
// 	maxTestPOs   = 1000
// 	testPOkeylen = 9
// )
//
// func TestPOs(t *testing.T) {
// 	for i := 0; i < maxTestPOs; i++ {
// 		length := rand.Intn(256) + 1
// 		v0 := RandomAddress().Bin()[:length]
// 		v1 := RandomAddress().Bin()[:length]
// 		a0 := NewBoolAddress(v0)
// 		a1 := NewBoolAddress(v1)
// 		b0 := NewHashAddress(v0)
// 		b1 := NewHashAddress(v1)
// 		pos := rand.Intn(length) + 1
// 		apo, aeq := a0.PO(a1, pos)
// 		bpo, beq := b0.PO(b1, pos)
// 		if bpo == 256 {
// 			bpo = length
// 		}
// 		a0s := a0.String()
// 		if a0s != v0 {
// 			t.Fatalf("incorrect bool address. expected %v, got %v", v0, a0s)
// 		}
// 		a1s := a1.String()
// 		if a1s != v1 {
// 			t.Fatalf("incorrect bool address. expected %v, got %v", v1, a1s)
// 		}
// 		b0s := b0.String()[:length]
// 		if b0s != v0 {
// 			t.Fatalf("incorrect hash address. expected %v, got %v", v0, b0s)
// 		}
// 		b1s := b1.String()[:length]
// 		if b1s != v1 {
// 			t.Fatalf("incorrect hash address. expected %v, got %v", v1, b1s)
// 		}
// 		if apo != bpo {
// 			t.Fatalf("PO does not match for %v X %v (pos: %v): expected %v, got %v", v0, v1, pos, apo, bpo)
// 		}
// 		if aeq != beq {
// 			t.Fatalf("PO equality does not match for %v X %v (pos: %v): expected %v, got %v", v0, v1, pos, aeq, beq)
// 		}
// 	}
// }
