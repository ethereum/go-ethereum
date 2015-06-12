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
 * 	Taylor Gerring <taylor.gerring@gmail.com>
 */

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/codegangsta/cli"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/tests"
)

var (
	continueOnError = false
	testExtension   = ".json"
	defaultTest     = "all"
	defaultDir      = "."
	allTests        = []string{"BlockTests", "StateTests", "TransactionTests", "VMTests"}

	TestFlag = cli.StringFlag{
		Name:  "test",
		Usage: "Test type (string): VMTests, TransactionTests, StateTests, BlockTests",
		Value: defaultTest,
	}
	FileFlag = cli.StringFlag{
		Name:   "file",
		Usage:  "Test file or directory. Directories are searched for .json files 1 level deep",
		Value:  defaultDir,
		EnvVar: "ETHEREUM_TEST_PATH",
	}
	ContinueOnErrorFlag = cli.BoolFlag{
		Name:  "continue",
		Usage: "Continue running tests on error (true) or [default] exit immediately (false)",
	}
)

func runTest(test, file string) error {
	// glog.Infoln("runTest", test, file)
	var err error
	switch test {
	case "bc", "BlockTest", "BlockTests", "BlockChainTest":
		err = tests.RunBlockTest(file)
	case "st", "state", "StateTest", "StateTests":
		err = tests.RunStateTest(file)
	case "tx", "TransactionTest", "TransactionTests":
		err = tests.RunTransactionTests(file)
	case "vm", "VMTest", "VMTests":
		err = tests.RunVmTest(file)
	default:
		err = fmt.Errorf("Invalid test type specified:", test)
	}
	return err
}

func getFiles(path string) ([]string, error) {
	// glog.Infoln("getFiles", path)
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
			if !v.IsDir() && v.Name()[len(v.Name())-len(testExtension):len(v.Name())] == testExtension {
				files[i] = filepath.Join(path, v.Name())
				// glog.Infoln("Found file", files[i])
			}
		}
	case mode.IsRegular():
		files = make([]string, 1)
		files[0] = path
	}

	return files, nil
}

func runSuite(test, file string) {
	var tests []string

	if test == defaultTest {
		tests = allTests
	} else {
		tests = []string{test}
	}

	for _, curTest := range tests {
		// glog.Infoln("runSuite", curTest, file)
		var err error
		var files []string
		if test == defaultTest {
			files, err = getFiles(filepath.Join(file, curTest))

		} else {
			files, err = getFiles(file)
		}
		if err != nil {
			glog.Fatalln(err)
		}

		if len(files) == 0 {
			glog.Warningln("No files matched path")
		}
		for _, testfile := range files {
			// Skip blank entries
			if len(testfile) == 0 {
				continue
			}

			// TODO allow io.Reader to be passed so Stdin can be piped
			// RunVmTest(strings.NewReader(os.Args[2]))
			// RunVmTest(os.Stdin)
			err := runTest(curTest, testfile)
			if err != nil {
				if continueOnError {
					glog.Errorln(err)
				} else {
					glog.Fatalln(err)
				}
			}

		}
	}
}

func setupApp(c *cli.Context) {
	flagTest := c.GlobalString(TestFlag.Name)
	flagFile := c.GlobalString(FileFlag.Name)
	continueOnError = c.GlobalBool(ContinueOnErrorFlag.Name)

	runSuite(flagTest, flagFile)
}

func main() {
	glog.SetToStderr(true)

	app := cli.NewApp()
	app.Name = "ethtest"
	app.Usage = "go-ethereum test interface"
	app.Action = setupApp
	app.Version = "0.2.0"
	app.Author = "go-ethereum team"

	app.Flags = []cli.Flag{
		TestFlag,
		FileFlag,
		ContinueOnErrorFlag,
	}

	if err := app.Run(os.Args); err != nil {
		glog.Fatalln(err)
	}

}
