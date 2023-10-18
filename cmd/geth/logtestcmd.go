// Copyright 2023 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
	"github.com/urfave/cli/v2"
	"math"
	"math/big"
	"time"
)

// logTest is an entry point which spits out some logs. This is used by testing
// to verify expected outputs
func logTest(ctx *cli.Context) error {

	{ // big.Int
		ba, _ := new(big.Int).SetString("111222333444555678999", 10)    // "111,222,333,444,555,678,999"
		bb, _ := new(big.Int).SetString("-111222333444555678999", 10)   // "-111,222,333,444,555,678,999"
		bc, _ := new(big.Int).SetString("11122233344455567899900", 10)  // "11,122,233,344,455,567,899,900"
		bd, _ := new(big.Int).SetString("-11122233344455567899900", 10) // "-11,122,233,344,455,567,899,900"
		log.Info("big.Int", "111,222,333,444,555,678,999", ba)
		log.Info("-big.Int", "-111,222,333,444,555,678,999", bb)
		log.Info("big.Int", "11,122,233,344,455,567,899,900", bc)
		log.Info("-big.Int", "-11,122,233,344,455,567,899,900", bd)
	}
	{ //uint256
		ua, _ := uint256.FromDecimal("111222333444555678999")
		ub, _ := uint256.FromDecimal("11122233344455567899900")
		log.Info("uint256", "111,222,333,444,555,678,999", ua)
		log.Info("uint256", "11,122,233,344,455,567,899,900", ub)
	}
	{ // int64
		log.Info("int64", "1,000,000", int64(1000000))
		log.Info("int64", "-1,000,000", int64(-1000000))
		log.Info("int64", "9,223,372,036,854,775,807", math.MaxInt64)
		log.Info("int64", "-9,223,372,036,854,775,808", math.MinInt64)
	}
	{ // uint64
		log.Info("uint64", "1,000,000", uint64(1000000))
		log.Info("uint64", "18,446,744,073,709,551,615", uint64(math.MaxUint64))
	}
	{ // Special characters
		log.Info("Special chars in value", "key", "special \r\n\t chars")
		log.Info("Special chars in key", "special \n\t chars", "value")

		log.Info("nospace", "nospace", "nospace")
		log.Info("with space", "with nospace", "with nospace")

		log.Info("Bash escapes in value", "key", "\u001b[1G\u001b[K\u001b[1A")
		log.Info("Bash escapes in key", "\u001b[1G\u001b[K\u001b[1A", "value")

		log.Info("Bash escapes in message  \u001b[1G\u001b[K\u001b[1A end", "key", "value")

		colored := fmt.Sprintf("\u001B[%dmColored\u001B[0m[", 35)
		log.Info(colored, colored, colored)
	}
	{ // Custom Stringer() - type
		log.Info("Custom Stringer value", "2562047h47m16.854s", common.PrettyDuration(time.Duration(9223372036854775807)))
	}
	{ // Lazy eval
		log.Info("Lazy evaluation of value", "key", log.Lazy{Fn: func() interface{} { return "lazy value" }})
	}
	return nil
}
