/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * @authors:
 * 	Jeffrey Wilcke <i@jev.io>
 */

package main

import (
	"os"

	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/tests"
)

func main() {
	// helper.Logger.SetLogLevel(5)
	// vm.Debug = true

	if len(os.Args) < 2 {
		glog.Exit("Must specify test type")
	}

	test := os.Args[1]

	// var code int
	switch test {
	case "vm", "VMTests":
		if len(os.Args) > 2 {
			if err := tests.RunVmTest(os.Args[2]); err != nil {
				glog.Errorln(err)
			}
		} else {
			glog.Exit("Must supply file argument")
		}
	case "state", "StateTest":
		if len(os.Args) > 2 {
			if err := tests.RunStateTest(os.Args[2]); err != nil {
				glog.Errorln(err)
			}
			// code = RunVmTest(strings.NewReader(os.Args[2]))
		} else {
			glog.Exit("Must supply file argument")
			// code = RunVmTest(os.Stdin)
		}
	case "tx", "TransactionTests":
		if len(os.Args) > 2 {
			if err := tests.RunTransactionTests(os.Args[2]); err != nil {
				glog.Errorln(err)
			}
		} else {
			glog.Exit("Must supply file argument")
		}
	case "bc", "BlockChainTest":
		if len(os.Args) > 2 {
			if err := tests.RunBlockTest(os.Args[2]); err != nil {
				glog.Errorln(err)
			}
		} else {
			glog.Exit("Must supply file argument")
		}
	default:
		glog.Exit("Invalid test type specified")
	}

	// os.Exit(code)
}
