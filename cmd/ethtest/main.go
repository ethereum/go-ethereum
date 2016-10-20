// Copyright 2014 The go-ethereum Authors
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

// ethtest executes Ethereum JSON tests.
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/tests"
	"gopkg.in/urfave/cli.v1"
)

var (
	continueOnError = false
	testExtension   = ".json"
	defaultTest     = "all"
	defaultDir      = "."
	allTests        = []string{"BlockTests", "StateTests", "TransactionTests", "VMTests", "RLPTests"}
	testDirMapping  = map[string]string{"BlockTests": "BlockchainTests"}
	skipTests       = []string{}

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
	ReadStdInFlag = cli.BoolFlag{
		Name:  "stdin",
		Usage: "Accept input from stdin instead of reading from file",
	}
	SkipTestsFlag = cli.StringFlag{
		Name:  "skip",
		Usage: "Tests names to skip",
	}
	TraceFlag = cli.BoolFlag{
		Name:  "trace",
		Usage: "Enable VM tracing",
	}
)

func runTestWithReader(test string, r io.Reader) error {
	glog.Infoln("runTest", test)
	var err error
	switch strings.ToLower(test) {
	case "bk", "block", "blocktest", "blockchaintest", "blocktests", "blockchaintests":
		err = tests.RunBlockTestWithReader(params.MainNetHomesteadBlock, params.MainNetDAOForkBlock, params.MainNetHomesteadGasRepriceBlock, r, skipTests)
	case "st", "state", "statetest", "statetests":
		rs := &params.ChainConfig{HomesteadBlock: params.MainNetHomesteadBlock, DAOForkBlock: params.MainNetDAOForkBlock, DAOForkSupport: true, EIP150Block: params.MainNetHomesteadGasRepriceBlock}
		err = tests.RunStateTestWithReader(rs, r, skipTests)
	case "tx", "transactiontest", "transactiontests":
		err = tests.RunTransactionTestsWithReader(r, skipTests)
	case "vm", "vmtest", "vmtests":
		err = tests.RunVmTestWithReader(r, skipTests)
	case "rlp", "rlptest", "rlptests":
		err = tests.RunRLPTestWithReader(r, skipTests)
	default:
		err = fmt.Errorf("Invalid test type specified: %v", test)
	}

	if err != nil {
		return err
	}

	return nil
}

func getFiles(path string) ([]string, error) {
	glog.Infoln("getFiles", path)
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
				glog.Infoln("Found file", files[i])
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
		glog.Infoln("runSuite", curTest, file)
		var err error
		var files []string
		if test == defaultTest {
			// check if we have an explicit directory mapping for the test
			if _, ok := testDirMapping[curTest]; ok {
				files, err = getFiles(filepath.Join(file, testDirMapping[curTest]))
			} else {
				// otherwise assume test name
				files, err = getFiles(filepath.Join(file, curTest))
			}
		} else {
			files, err = getFiles(file)
		}
		if err != nil {
			glog.Fatalln(err)
		}

		if len(files) == 0 {
			glog.Warningln("No files matched path")
		}
		for _, curFile := range files {
			// Skip blank entries
			if len(curFile) == 0 {
				continue
			}

			r, err := os.Open(curFile)
			if err != nil {
				glog.Fatalln(err)
			}
			defer r.Close()

			err = runTestWithReader(curTest, r)
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

func setupApp(c *cli.Context) error {
	flagTest := c.GlobalString(TestFlag.Name)
	flagFile := c.GlobalString(FileFlag.Name)
	continueOnError = c.GlobalBool(ContinueOnErrorFlag.Name)
	useStdIn := c.GlobalBool(ReadStdInFlag.Name)
	skipTests = strings.Split(c.GlobalString(SkipTestsFlag.Name), " ")

	if !useStdIn {
		runSuite(flagTest, flagFile)
	} else {
		if err := runTestWithReader(flagTest, os.Stdin); err != nil {
			glog.Fatalln(err)
		}
	}
	return nil
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
		ReadStdInFlag,
		SkipTestsFlag,
		TraceFlag,
	}

	if err := app.Run(os.Args); err != nil {
		glog.Fatalln(err)
	}

}
