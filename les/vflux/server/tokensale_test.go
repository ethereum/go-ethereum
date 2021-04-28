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

package server

import (
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/les/utils"
	"github.com/ethereum/go-ethereum/les/vflux"
)

func randomTokenLimit() int64 { return rand.Int63n(1000000) + 1 }

func randomLogBasePrice() utils.Fixed64 {
	return utils.Fixed64(rand.Int63n(int64(utils.Uint64ToFixed64(20))+1)) - utils.Uint64ToFixed64(10)
}

func TestBondingCurve(t *testing.T) {
	tokenLimit, logBasePrice := randomTokenLimit(), randomLogBasePrice()
	bc := newBondingCurve(linHyperCurve, tokenLimit, logBasePrice)
	var tokenAmount int64
	currencyAmount := big.NewInt(0)

	exchange := func(minAmount, maxAmount int64, maxCost *big.Int, exact bool) bool {
		if dt, dc := bc.exchange(minAmount, maxAmount, maxCost); dc != nil {
			// success; do checks before returning
			if (dt < minAmount || dt > maxAmount) && (tokenAmount+dt != 0) { // valid if all tokens are sold
				t.Fatalf("Exchanged token amount %v outside requested range (%v to %v)", dt, minAmount, maxAmount)
			}
			tokenAmount += dt
			if tokenAmount < 0 {
				t.Fatalf("Total token amount %v is negative", tokenAmount)
			}
			if exact {
				if dc.Cmp(maxCost) != 0 {
					t.Fatalf("Exchanged currency amount %v does not match expected %v", dc, maxCost)
				}
			} else {
				if dc.Cmp(maxCost) > 0 {
					t.Fatalf("Exchanged currency amount %v bigger than specified maximum %v", dc, maxCost)
				}
			}
			currencyAmount.Add(currencyAmount, dc)
			if currencyAmount.Sign() < 0 {
				t.Fatalf("Total currency amount %v is negative", currencyAmount)
			}
			return true
		}
		return false
	}

	randomExchange := func() {
		for {
			// repeat until an exchange is successful
			minAmount := rand.Int63n(2000001) - 1000000
			maxAmount := rand.Int63n(2000001) - 1000000
			if maxAmount < minAmount {
				minAmount, maxAmount = maxAmount, minAmount
			}
			mc := rand.Float64() * 1000000 * math.Pow(2, rand.Float64()*20-10)
			maxCost, _ := big.NewFloat(mc).Int(nil)
			if exchange(minAmount, maxAmount, maxCost, false) {
				return
			}
		}
	}

	queryAndExchange := func() {
		var amount int64
		var maxCost *big.Int
		for {
			// repeat until exchange is possible
			amount = rand.Int63n(2000001) - 1000000
			if maxCost = bc.price(amount); maxCost != nil {
				break
			}
		}
		if !exchange(amount, amount, maxCost, true) {
			t.Fatalf("Exchange based on query failed (amount: %v  cost: %v)", amount, maxCost)
		}
	}

	checkUnitPrice := func() {
		if tokenLimit < 100 || tokenLimit-tokenAmount <= tokenLimit/100 {
			// skip corner cases with high expected deviance
			return
		}
		expPrice := logBasePrice.Pow2() * vflux.LinHyper((float64(tokenAmount)*2+1)/float64(tokenLimit)) // / float64(tokenLimit)
		maxDiff := expPrice / 100
		if maxDiff < 2 {
			maxDiff = 2
		}
		price := bc.price(1)
		if price == nil {
			t.Fatalf("Unit price calculation failed")
		}
		fprice, _ := new(big.Float).SetInt(price).Float64()
		if fprice < expPrice-maxDiff || fprice > expPrice+maxDiff {
			t.Fatalf("Unit price %v does not match expected %v", fprice, expPrice)
		}
	}

	adjust := func(targetTokenLimit int64, targetLogBasePrice utils.Fixed64) {
		doExchanges := rand.Intn(2) == 1
		for {
			earned, success := bc.adjust(tokenAmount, targetTokenLimit, targetLogBasePrice)
			if earned.Sign() < 0 {
				t.Fatalf("Earned amount %v is negative", earned)
			}
			currencyAmount.Sub(currencyAmount, earned)
			if currencyAmount.Sign() < 0 {
				t.Fatalf("Total currency amount %v is negative", currencyAmount)
			}
			if success {
				return
			}
			// target curve parameters not reached yet; simulate token spending and
			// optionally do further exchange operations until target is reached
			if doExchanges {
				randomExchange()
			}
			tokenAmount -= (tokenAmount + 99) / 100
		}
	}

	for i := 0; i < 10000; i++ {
		randomExchange()
		queryAndExchange()
		checkUnitPrice()
		tokenLimit, logBasePrice = randomTokenLimit(), randomLogBasePrice()
		adjust(tokenLimit, logBasePrice)
	}
	// sell remaining tokens
	if !exchange(-tokenAmount, -tokenAmount, big.NewInt(0), false) {
		t.Fatalf("Selling remaining token amount %v failed", tokenAmount)
	}
	if currencyAmount.Sign() != 0 {
		t.Fatalf("Remaining currency amount %v (expected zero)", currencyAmount)
	}
}
