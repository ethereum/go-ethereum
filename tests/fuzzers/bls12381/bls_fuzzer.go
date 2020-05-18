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

package bls

func Fuzz(data []byte) int {
	a := new(bls12381G1Add)
	a.Run(data)
	b := new(bls12381G1Mul)
	b.Run(data)
	c := new(bls12381G1MultiExp)
	c.Run(data)
	d := new(bls12381G2Add)
	d.Run(data)
	e := new(bls12381G2Mul)
	e.Run(data)
	f := new(bls12381G2MultiExp)
	f.Run(data)
	g := new(bls12381MapG1)
	g.Run(data)
	h := new(bls12381MapG2)
	h.Run(data)
	i := new(bls12381Pairing)
	i.Run(data)
	return 0
}
