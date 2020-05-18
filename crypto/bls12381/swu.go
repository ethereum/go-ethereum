// Copyright 2020 The go-ethereum Authors
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

package bls12381

// swuMapG1 is implementation of Simplified Shallue-van de Woestijne-Ulas Method
// follows the implmentation at draft-irtf-cfrg-hash-to-curve-06.
func swuMapG1(u *fe) (*fe, *fe) {
	var params = swuParamsForG1
	var tv [4]*fe
	for i := 0; i < 4; i++ {
		tv[i] = new(fe)
	}
	square(tv[0], u)
	mul(tv[0], tv[0], params.z)
	square(tv[1], tv[0])
	x1 := new(fe)
	add(x1, tv[0], tv[1])
	inverse(x1, x1)
	e1 := x1.isZero()
	one := new(fe).one()
	add(x1, x1, one)
	if e1 {
		x1.set(params.zInv)
	}
	mul(x1, x1, params.minusBOverA)
	gx1 := new(fe)
	square(gx1, x1)
	add(gx1, gx1, params.a)
	mul(gx1, gx1, x1)
	add(gx1, gx1, params.b)
	x2 := new(fe)
	mul(x2, tv[0], x1)
	mul(tv[1], tv[0], tv[1])
	gx2 := new(fe)
	mul(gx2, gx1, tv[1])
	e2 := !isQuadraticNonResidue(gx1)
	x, y2 := new(fe), new(fe)
	if e2 {
		x.set(x1)
		y2.set(gx1)
	} else {
		x.set(x2)
		y2.set(gx2)
	}
	y := new(fe)
	sqrt(y, y2)
	if y.sign() != u.sign() {
		neg(y, y)
	}
	return x, y
}

// swuMapG2 is implementation of Simplified Shallue-van de Woestijne-Ulas Method
// defined at draft-irtf-cfrg-hash-to-curve-06.
func swuMapG2(e *fp2, u *fe2) (*fe2, *fe2) {
	if e == nil {
		e = newFp2()
	}
	params := swuParamsForG2
	var tv [4]*fe2
	for i := 0; i < 4; i++ {
		tv[i] = e.new()
	}
	e.square(tv[0], u)
	e.mul(tv[0], tv[0], params.z)
	e.square(tv[1], tv[0])
	x1 := e.new()
	e.add(x1, tv[0], tv[1])
	e.inverse(x1, x1)
	e1 := x1.isZero()
	e.add(x1, x1, e.one())
	if e1 {
		x1.set(params.zInv)
	}
	e.mul(x1, x1, params.minusBOverA)
	gx1 := e.new()
	e.square(gx1, x1)
	e.add(gx1, gx1, params.a)
	e.mul(gx1, gx1, x1)
	e.add(gx1, gx1, params.b)
	x2 := e.new()
	e.mul(x2, tv[0], x1)
	e.mul(tv[1], tv[0], tv[1])
	gx2 := e.new()
	e.mul(gx2, gx1, tv[1])
	e2 := !e.isQuadraticNonResidue(gx1)
	x, y2 := e.new(), e.new()
	if e2 {
		x.set(x1)
		y2.set(gx1)
	} else {
		x.set(x2)
		y2.set(gx2)
	}
	y := e.new()
	e.sqrt(y, y2)
	if y.sign() != u.sign() {
		e.neg(y, y)
	}
	return x, y
}

var swuParamsForG1 = struct {
	z           *fe
	zInv        *fe
	a           *fe
	b           *fe
	minusBOverA *fe
}{
	a:           &fe{3415322872136444497, 9675504606121301699, 13284745414851768802, 2873609449387478652, 2897906769629812789, 1536947672689614213},
	b:           &fe{18129637713272545760, 11144507692959411567, 10108153527111632324, 9745270364868568433, 14587922135379007624, 469008097655535723},
	z:           &fe{9830232086645309404, 1112389714365644829, 8603885298299447491, 11361495444721768256, 5788602283869803809, 543934104870762216},
	zInv:        &fe{1047701040585522704, 6568704757426767313, 7461573184509654906, 5499015922318795030, 11226104418450030905, 1048548528059189658},
	minusBOverA: &fe{370847444534405118, 4269648997187665026, 1978763176675559811, 2677363437243537255, 11096866317338941469, 683609622716391635},
}

var swuParamsForG2 = struct {
	z           *fe2
	zInv        *fe2
	a           *fe2
	b           *fe2
	minusBOverA *fe2
}{
	a: &fe2{
		fe{0, 0, 0, 0, 0, 0},
		fe{16517514583386313282, 74322656156451461, 16683759486841714365, 815493829203396097, 204518332920448171, 1306242806803223655},
	},
	b: &fe2{
		fe{2515823342057463218, 7982686274772798116, 7934098172177393262, 8484566552980779962, 4455086327883106868, 1323173589274087377},
		fe{2515823342057463218, 7982686274772798116, 7934098172177393262, 8484566552980779962, 4455086327883106868, 1323173589274087377},
	},
	z: &fe2{
		fe{9794203289623549276, 7309342082925068282, 1139538881605221074, 15659550692327388916, 16008355200866287827, 582484205531694093},
		fe{4897101644811774638, 3654671041462534141, 569769440802610537, 17053147383018470266, 17227549637287919721, 291242102765847046},
	},
	zInv: &fe2{
		fe{12452452969679491344, 11374291236854484173, 13099329512014041791, 17416955488833933518, 4817360797345214593, 1382542053011693074},
		fe{16399576568092893731, 5746367929944742296, 886009817557060804, 7754232252852521560, 3003423379798094998, 1182527591141693329},
	},
	minusBOverA: &fe2{
		fe{10393275865055580083, 6888480573845999877, 11497223857339693790, 14306043441748627554, 5078453791572287059, 1040691004897901061},
		fe{3009155151022283512, 13768405011380760314, 14385194789933939525, 11380038592375636572, 333649986898415235, 833107612749638805},
	},
}
