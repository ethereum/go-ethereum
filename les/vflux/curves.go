// Copyright 2021 The go-ethereum Authors
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

package vflux

import (
	"math"
)

func LinHyper(x float64) float64 {
	if x <= 1 {
		return x
	}
	x = 2 - x
	if x <= 0 {
		return math.Inf(1)
	}
	return 1 / x
}

func InvLinHyper(x float64) float64 {
	if x <= 1 {
		return x
	}
	return 2 - 1/x
}

func LinIntegral(x, dx float64) float64 {
	return dx * (x + (dx * 0.5))
}

func LinHyperIntegral(x, dx float64) float64 {
	var sum float64
	if x <= 1 {
		if x+dx <= 1 {
			return LinIntegral(x, dx)
		} else {
			dx1 := 1 - x
			sum = LinIntegral(x, dx1)
			dx -= dx1
			x = 1
		}
	}
	xx := 2 - x - dx
	if xx > 0 {
		sum += math.Log1p(dx / xx)
	} else {
		sum = math.Inf(1)
	}
	return sum
}

func InvLinIntegral(x, i float64) float64 {
	sq := x*x + 2*i
	if sq < 0 {
		return math.NaN()
	}
	if c := sq * 1e-10; i < c && i > -c {
		return x * i
	} else {
		return math.Sqrt(sq) - x
	}
}

func InvLinHyperIntegral(x, i float64) float64 {
	if x >= 2 {
		return 0
	}
	var dx float64
	if x < 1 {
		dx = InvLinIntegral(x, i)
		if x+dx <= 1 {
			return dx
		}
		dx = 1 - x
		i -= LinIntegral(x, dx)
		if i <= 0 {
			return dx
		}
		x = 1
	}
	r := math.Expm1(i)
	return dx + (2-x)*r/(r+1)
}
