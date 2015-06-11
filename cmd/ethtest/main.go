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
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/tests"
)

func getFiles(path string) ([]string, error) {
	var files []string
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	switch mode := fi.Mode(); {
	case mode.IsDir():
		fi, _ := ioutil.ReadDir(path)
		files = make([]string, len(fi))
		for i, v := range fi {
			// only go 1 depth and leave directory entires blank
			if !v.IsDir() {
				files[i] = path + v.Name()
			}
		}
	case mode.IsRegular():
		files = make([]string, 1)
		files[0] = path
	}

	return files, nil
}

func main() {
	glog.SetToStderr(true)
	var continueOnError bool = false
	// vm.Debug = true

	if len(os.Args) < 2 {
		glog.Exit("Must specify test type")
	}

	testtype := os.Args[1]
	var pattern string
	if len(os.Args) > 2 {
		pattern = os.Args[2]
	}

	files, err := getFiles(pattern)
	if err != nil {
		glog.Fatal(err)
	}

	for _, testfile := range files {
		// Skip blank entries
		if len(testfile) == 0 {
			continue
		}
		// TODO allow io.Reader to be passed so Stdin can be piped
		// RunVmTest(strings.NewReader(os.Args[2]))
		// RunVmTest(os.Stdin)
		var err error
		switch testtype {
		case "vm", "VMTests":
			err = tests.RunVmTest(testfile)
		case "state", "StateTest":
			err = tests.RunStateTest(testfile)
		case "tx", "TransactionTests":
			err = tests.RunTransactionTests(testfile)
		case "bc", "BlockChainTest":
			err = tests.RunBlockTest(testfile)
		default:
			glog.Fatalln("Invalid test type specified")
		}

		if err != nil {
			if continueOnError {
				glog.Errorln(err)
			} else {
				glog.Fatalln(err)
			}
		}
	}
}
